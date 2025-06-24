package components

import (
	"context"
	"slices"
	"strings"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/tui/abstract"
)

type MemberList struct {
	*mauview.Flex
	app         abstract.App
	Members     []id.UserID
	PowerLevels *event.PowerLevelsEventContent
	elements    map[id.UserID]mauview.Component
}

func NewMemberList(ctx context.Context, app abstract.App, members []id.UserID, powerLevels *event.PowerLevelsEventContent) *MemberList {
	list := &MemberList{
		Flex:        mauview.NewFlex(),
		app:         app,
		Members:     members,
		PowerLevels: powerLevels,
		elements:    make(map[id.UserID]mauview.Component),
	}
	list.AddFixedComponent(mauview.NewTextField().SetText("Member List"), 1)
	list.Render()
	return list
}

func (ml *MemberList) powerLevelsOrDefault() *event.PowerLevelsEventContent {
	if ml.PowerLevels == nil {
		return &event.PowerLevelsEventContent{}
	}
	return ml.PowerLevels
}

func (ml *MemberList) sortedMembers() []id.UserID {
	newMembers := make([]id.UserID, len(ml.Members))
	copy(newMembers, ml.Members)
	pl := ml.powerLevelsOrDefault()
	slices.SortFunc(newMembers, func(a, b id.UserID) int {
		aPL := pl.GetUserLevel(a)
		bPL := pl.GetUserLevel(b)
		if aPL != bPL {
			return bPL - aPL // Higher power level first
		}
		return strings.Compare(a.String(), b.String())
	})
	return newMembers
}

func (ml *MemberList) Render() {
	for _, element := range ml.elements {
		ml.RemoveComponent(element)
	}
	for _, userID := range ml.sortedMembers() {
		e := mauview.NewButton(userID.String())
		ml.AddFixedComponent(e, 1)
		ml.elements[userID] = e
		ml.app.Gmx().Log.Debug().Msgf("Added member %s to member list", userID)
	}
	ml.app.App().Redraw()
}
