// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	getAllSpaceChildren = `
		SELECT space_id, child_id, child_event_rowid, "order", suggested, parent_event_rowid, canonical, parent_validated
		FROM space_edge
		-- This check should be redundant thanks to parent_validated and validation before insert for children
		--INNER JOIN room ON space_id = room.room_id AND room.room_type = 'm.space'
		WHERE (space_id = $1 OR $1 = '') AND (child_event_rowid IS NOT NULL OR parent_validated)
		ORDER BY space_id, "order", child_id
	`
	getTopLevelSpaces = `
		SELECT space_id
		FROM (SELECT DISTINCT(space_id) FROM space_edge) outeredge
		LEFT JOIN room_account_data ON
			room_account_data.user_id = $1
			AND room_account_data.room_id = outeredge.space_id
			AND room_account_data.type = 'org.matrix.msc3230.space_order'
		WHERE NOT EXISTS(
			SELECT 1
			FROM space_edge inneredge
			INNER JOIN room ON inneredge.space_id = room.room_id
			WHERE inneredge.child_id = outeredge.space_id
				AND (inneredge.child_event_rowid IS NOT NULL OR inneredge.parent_validated)
		) AND EXISTS(SELECT 1 FROM room WHERE room_id = space_id AND room_type = 'm.space')
		ORDER BY room_account_data.content->>'$.order' NULLS LAST, space_id
	`
	revalidateAllParents = `
		UPDATE space_edge
		SET parent_validated=(SELECT EXISTS(
			SELECT 1
			FROM room
				INNER JOIN current_state cs ON cs.room_id = room.room_id AND cs.event_type = 'm.room.power_levels' AND cs.state_key = ''
				INNER JOIN event pls ON cs.event_rowid = pls.rowid
				INNER JOIN event edgeevt ON space_edge.parent_event_rowid = edgeevt.rowid
			WHERE	room.room_id = space_edge.space_id
				AND room.room_type = 'm.space'
				AND COALESCE(
					(
						SELECT value
						FROM json_each(pls.content, '$.users')
						WHERE key=edgeevt.sender AND type='integer'
					),
					pls.content->>'$.users_default',
					0
				) >= COALESCE(
					pls.content->>'$.events."m.space.child"',
					pls.content->>'$.state_default',
					50
				)
		))
		WHERE parent_event_rowid IS NOT NULL
	`
	revalidateAllParentsPointingAtSpaceQuery = revalidateAllParents + ` AND space_id=$1`
	revalidateAllParentsOfRoomQuery          = revalidateAllParents + ` AND child_id=$1`
	revalidateSpecificParentQuery            = revalidateAllParents + ` AND space_id=$1 AND child_id=$2`
	clearSpaceChildrenQuery                  = `
		UPDATE space_edge SET child_event_rowid=NULL, "order"='', suggested=false
		WHERE space_id=$1
	`
	clearSpaceParentsQuery = `
		UPDATE space_edge SET parent_event_rowid=NULL, canonical=false, parent_validated=false
		WHERE child_id=$1
	`
	removeSpaceChildQuery         = clearSpaceChildrenQuery + ` AND child_id=$2`
	removeSpaceParentQuery        = clearSpaceParentsQuery + ` AND space_id=$2`
	deleteEmptySpaceEdgeRowsQuery = `
		DELETE FROM space_edge WHERE child_event_rowid IS NULL AND parent_event_rowid IS NULL
	`
	addSpaceChildQuery = `
		INSERT INTO space_edge (space_id, child_id, child_event_rowid, "order", suggested)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (space_id, child_id) DO UPDATE
			SET child_event_rowid=EXCLUDED.child_event_rowid,
				"order"=EXCLUDED."order",
				suggested=EXCLUDED.suggested
	`
	addSpaceParentQuery = `
		INSERT INTO space_edge (space_id, child_id, parent_event_rowid, canonical)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (space_id, child_id) DO UPDATE
			SET parent_event_rowid=EXCLUDED.parent_event_rowid,
			    canonical=EXCLUDED.canonical,
				parent_validated=false
	`
)

var massInsertSpaceParentBuilder = dbutil.NewMassInsertBuilder[SpaceParentEntry, [1]any](addSpaceParentQuery, "($%d, $1, $%d, $%d)")
var massInsertSpaceChildBuilder = dbutil.NewMassInsertBuilder[SpaceChildEntry, [1]any](addSpaceChildQuery, "($1, $%d, $%d, $%d, $%d)")

type SpaceEdgeQuery struct {
	*dbutil.QueryHelper[*SpaceEdge]
}

func (seq *SpaceEdgeQuery) AddChild(ctx context.Context, spaceID, childID id.RoomID, childEventRowID EventRowID, order string, suggested bool) error {
	return seq.Exec(ctx, addSpaceChildQuery, spaceID, childID, childEventRowID, order, suggested)
}

func (seq *SpaceEdgeQuery) AddParent(ctx context.Context, spaceID, childID id.RoomID, parentEventRowID EventRowID, canonical bool) error {
	return seq.Exec(ctx, addSpaceParentQuery, spaceID, childID, parentEventRowID, canonical)
}

type SpaceParentEntry struct {
	ParentID   id.RoomID
	EventRowID EventRowID
	Canonical  bool
}

func (spe SpaceParentEntry) GetMassInsertValues() [3]any {
	return [...]any{spe.ParentID, spe.EventRowID, spe.Canonical}
}

type SpaceChildEntry struct {
	ChildID    id.RoomID
	EventRowID EventRowID
	Order      string
	Suggested  bool
}

func (sce SpaceChildEntry) GetMassInsertValues() [4]any {
	return [...]any{sce.ChildID, sce.EventRowID, sce.Order, sce.Suggested}
}

func (seq *SpaceEdgeQuery) SetChildren(ctx context.Context, spaceID id.RoomID, children []SpaceChildEntry, removedChildren []id.RoomID, clear bool) error {
	if clear {
		err := seq.Exec(ctx, clearSpaceChildrenQuery, spaceID)
		if err != nil {
			return err
		}
	} else if len(removedChildren) > 0 {
		for _, child := range removedChildren {
			err := seq.Exec(ctx, removeSpaceChildQuery, spaceID, child)
			if err != nil {
				return err
			}
		}
	}
	if len(removedChildren) > 0 {
		err := seq.Exec(ctx, deleteEmptySpaceEdgeRowsQuery, spaceID)
		if err != nil {
			return err
		}
	}
	if len(children) == 0 {
		return nil
	}
	query, params := massInsertSpaceChildBuilder.Build([1]any{spaceID}, children)
	return seq.Exec(ctx, query, params...)
}

func (seq *SpaceEdgeQuery) SetParents(ctx context.Context, childID id.RoomID, parents []SpaceParentEntry, removedParents []id.RoomID, clear bool) error {
	if clear {
		err := seq.Exec(ctx, clearSpaceParentsQuery, childID)
		if err != nil {
			return err
		}
	} else if len(removedParents) > 0 {
		for _, parent := range removedParents {
			err := seq.Exec(ctx, removeSpaceParentQuery, childID, parent)
			if err != nil {
				return err
			}
		}
	}
	if len(removedParents) > 0 {
		err := seq.Exec(ctx, deleteEmptySpaceEdgeRowsQuery)
		if err != nil {
			return err
		}
	}
	if len(parents) == 0 {
		return nil
	}
	query, params := massInsertSpaceParentBuilder.Build([1]any{childID}, parents)
	return seq.Exec(ctx, query, params...)
}

func (seq *SpaceEdgeQuery) RevalidateAllChildrenOfParentValidity(ctx context.Context, spaceID id.RoomID) error {
	return seq.Exec(ctx, revalidateAllParentsPointingAtSpaceQuery, spaceID)
}

func (seq *SpaceEdgeQuery) RevalidateAllParentsOfRoomValidity(ctx context.Context, childID id.RoomID) error {
	return seq.Exec(ctx, revalidateAllParentsOfRoomQuery, childID)
}

func (seq *SpaceEdgeQuery) RevalidateSpecificParentValidity(ctx context.Context, spaceID, childID id.RoomID) error {
	return seq.Exec(ctx, revalidateSpecificParentQuery, spaceID, childID)
}

func (seq *SpaceEdgeQuery) GetAll(ctx context.Context, spaceID id.RoomID) (map[id.RoomID][]*SpaceEdge, error) {
	edges := make(map[id.RoomID][]*SpaceEdge)
	err := seq.QueryManyIter(ctx, getAllSpaceChildren, spaceID).Iter(func(edge *SpaceEdge) (bool, error) {
		edges[edge.SpaceID] = append(edges[edge.SpaceID], edge)
		edge.SpaceID = ""
		if !edge.ParentValidated {
			edge.ParentEventRowID = 0
			edge.Canonical = false
		}
		return true, nil
	})
	return edges, err
}

var roomIDScanner = dbutil.ConvertRowFn[id.RoomID](dbutil.ScanSingleColumn[id.RoomID])

func (seq *SpaceEdgeQuery) GetTopLevelIDs(ctx context.Context, userID id.UserID) ([]id.RoomID, error) {
	return roomIDScanner.NewRowIter(seq.GetDB().Query(ctx, getTopLevelSpaces, userID)).AsList()
}

type SpaceEdge struct {
	SpaceID id.RoomID `json:"space_id,omitempty"`
	ChildID id.RoomID `json:"child_id"`

	ChildEventRowID EventRowID `json:"child_event_rowid,omitempty"`
	Order           string     `json:"order,omitempty"`
	Suggested       bool       `json:"suggested,omitempty"`

	ParentEventRowID EventRowID `json:"parent_event_rowid,omitempty"`
	Canonical        bool       `json:"canonical,omitempty"`
	ParentValidated  bool       `json:"-"`
}

func (se *SpaceEdge) Scan(row dbutil.Scannable) (*SpaceEdge, error) {
	var childRowID, parentRowID sql.NullInt64
	err := row.Scan(
		&se.SpaceID, &se.ChildID,
		&childRowID, &se.Order, &se.Suggested,
		&parentRowID, &se.Canonical, &se.ParentValidated,
	)
	if err != nil {
		return nil, err
	}
	se.ChildEventRowID = EventRowID(childRowID.Int64)
	se.ParentEventRowID = EventRowID(parentRowID.Int64)
	return se, nil
}
