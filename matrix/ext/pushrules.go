package gomx_ext

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/zyedidia/glob"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/room"
)

// GetPushRules returns the push notification rules for the global scope.
func GetPushRules(client *gomatrix.Client) (*PushRuleset, error) {
	return GetScopedPushRules(client, "global")
}

// GetScopedPushRules returns the push notification rules for the given scope.
func GetScopedPushRules(client *gomatrix.Client, scope string) (resp *PushRuleset, err error) {
	u, _ := url.Parse(client.BuildURL("pushrules", scope))
	// client.BuildURL returns the URL without a trailing slash, but the pushrules endpoint requires the slash.
	u.Path += "/"
	_, err = client.MakeRequest("GET", u.String(), nil, &resp)
	return
}

type PushRuleset struct {
	Override  PushRuleArray
	Content   PushRuleArray
	Room      PushRuleMap
	Sender    PushRuleMap
	Underride PushRuleArray
}

type rawPushRuleset struct {
	Override  PushRuleArray `json:"override"`
	Content   PushRuleArray `json:"content"`
	Room      PushRuleArray `json:"room"`
	Sender    PushRuleArray `json:"sender"`
	Underride PushRuleArray `json:"underride"`
}

func (rs *PushRuleset) UnmarshalJSON(raw []byte) (err error) {
	data := rawPushRuleset{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return
	}

	rs.Override = data.Override.setType(OverrideRule)
	rs.Content = data.Content.setType(ContentRule)
	rs.Room = data.Room.setTypeAndMap(RoomRule)
	rs.Sender = data.Sender.setTypeAndMap(SenderRule)
	rs.Underride = data.Underride.setType(UnderrideRule)
	return
}

func (rs *PushRuleset) MarshalJSON() ([]byte, error) {
	data := rawPushRuleset{
		Override:  rs.Override,
		Content:   rs.Content,
		Room:      rs.Room.unmap(),
		Sender:    rs.Sender.unmap(),
		Underride: rs.Underride,
	}
	return json.Marshal(&data)
}

var DefaultPushActions = make(PushActionArray, 0)

func (rs *PushRuleset) GetActions(room *rooms.Room, event *gomatrix.Event) (match PushActionArray) {
	if match = rs.Override.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Content.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Room.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Sender.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Underride.GetActions(room, event); match != nil {
		return
	}
	return DefaultPushActions
}

type PushRuleArray []*PushRule

func (rules PushRuleArray) setType(typ PushRuleType) PushRuleArray {
	for _, rule := range rules {
		rule.Type = typ
	}
	return rules
}

func (rules PushRuleArray) GetActions(room *rooms.Room, event *gomatrix.Event) PushActionArray {
	for _, rule := range rules {
		if !rule.Match(room, event) {
			continue
		}
		return rule.Actions
	}
	return nil
}

type PushRuleMap struct {
	Map  map[string]*PushRule
	Type PushRuleType
}

func (rules PushRuleArray) setTypeAndMap(typ PushRuleType) PushRuleMap {
	data := PushRuleMap{
		Map:  make(map[string]*PushRule),
		Type: typ,
	}
	for _, rule := range rules {
		rule.Type = typ
		data.Map[rule.RuleID] = rule
	}
	return data
}

func (ruleMap PushRuleMap) GetActions(room *rooms.Room, event *gomatrix.Event) PushActionArray {
	var rule *PushRule
	var found bool
	switch ruleMap.Type {
	case RoomRule:
		rule, found = ruleMap.Map[event.RoomID]
	case SenderRule:
		rule, found = ruleMap.Map[event.Sender]
	}
	if found && rule.Match(room, event) {
		return rule.Actions
	}
	return nil
}

func (ruleMap PushRuleMap) unmap() PushRuleArray {
	array := make(PushRuleArray, len(ruleMap.Map))
	index := 0
	for _, rule := range ruleMap.Map {
		array[index] = rule
		index++
	}
	return array
}

type PushRuleType string

const (
	OverrideRule  PushRuleType = "override"
	ContentRule   PushRuleType = "content"
	RoomRule      PushRuleType = "room"
	SenderRule    PushRuleType = "sender"
	UnderrideRule PushRuleType = "underride"
)

type PushRule struct {
	// The type of this rule.
	Type PushRuleType `json:"-"`
	// The ID of this rule.
	// For room-specific rules and user-specific rules, this is the room or user ID (respectively)
	// For other types of rules, this doesn't affect anything.
	RuleID string `json:"rule_id"`
	// The actions this rule should trigger when matched.
	Actions PushActionArray `json:"actions"`
	// Whether this is a default rule, or has been set explicitly.
	Default bool `json:"default"`
	// Whether or not this push rule is enabled.
	Enabled bool `json:"enabled"`
	// The conditions to match in order to trigger this rule.
	// Only applicable to generic underride/override rules.
	Conditions []*PushCondition `json:"conditions,omitempty"`
	// Pattern for content-specific push rules
	Pattern string `json:"pattern,omitempty"`
}

func (rule *PushRule) Match(room *rooms.Room, event *gomatrix.Event) bool {
	if !rule.Enabled {
		return false
	}
	switch rule.Type {
	case OverrideRule, UnderrideRule:
		return rule.matchConditions(room, event)
	case ContentRule:
		return rule.matchPattern(room, event)
	case RoomRule:
		return rule.RuleID == event.RoomID
	case SenderRule:
		return rule.RuleID == event.Sender
	default:
		return false
	}
}

func (rule *PushRule) matchConditions(room *rooms.Room, event *gomatrix.Event) bool {
	for _, cond := range rule.Conditions {
		if !cond.Match(room, event) {
			return false
		}
	}
	return true
}

func (rule *PushRule) matchPattern(room *rooms.Room, event *gomatrix.Event) bool {
	pattern, err := glob.Compile(rule.Pattern)
	if err != nil {
		return false
	}
	text, _ := event.Content["body"].(string)
	return pattern.MatchString(text)
}

type PushActionType string

const (
	ActionNotify     PushActionType = "notify"
	ActionDontNotify PushActionType = "dont_notify"
	ActionCoalesce   PushActionType = "coalesce"
	ActionSetTweak   PushActionType = "set_tweak"
)

type PushActionTweak string

const (
	TweakSound     PushActionTweak = "sound"
	TweakHighlight PushActionTweak = "highlight"
)

type PushActionArray []*PushAction

type PushActionArrayShould struct {
	NotifySpecified bool
	Notify          bool
	Highlight       bool

	PlaySound bool
	SoundName string
}

func (actions PushActionArray) Should() (should PushActionArrayShould) {
	for _, action := range actions {
		switch action.Action {
		case ActionNotify, ActionCoalesce:
			should.Notify = true
			should.NotifySpecified = true
		case ActionDontNotify:
			should.Notify = false
			should.NotifySpecified = true
		case ActionSetTweak:
			switch action.Tweak {
			case TweakHighlight:
				var ok bool
				should.Highlight, ok = action.Value.(bool)
				if !ok {
					// Highlight value not specified, so assume true since the tweak is set.
					should.Highlight = true
				}
			case TweakSound:
				should.SoundName = action.Value.(string)
				should.PlaySound = len(should.SoundName) > 0
			}
		}
	}
	return
}

type PushAction struct {
	Action PushActionType
	Tweak  PushActionTweak
	Value  interface{}
}

func (action *PushAction) UnmarshalJSON(raw []byte) error {
	var data interface{}

	err := json.Unmarshal(raw, &data)
	if err != nil {
		return err
	}

	switch val := data.(type) {
	case string:
		action.Action = PushActionType(val)
	case map[string]interface{}:
		tweak, ok := val["set_tweak"].(string)
		if ok {
			action.Action = ActionSetTweak
			action.Tweak = PushActionTweak(tweak)
			action.Value, _ = val["value"]
		}
	}
	return nil
}

func (action *PushAction) MarshalJSON() (raw []byte, err error) {
	if action.Action == ActionSetTweak {
		data := map[string]interface{}{
			"set_tweak": action.Tweak,
			"value":     action.Value,
		}
		return json.Marshal(&data)
	} else {
		data := string(action.Action)
		return json.Marshal(&data)
	}
}

type PushKind string

const (
	KindEventMatch          PushKind = "event_match"
	KindContainsDisplayName PushKind = "contains_display_name"
	KindRoomMemberCount     PushKind = "room_member_count"
)

type PushCondition struct {
	Kind    PushKind `json:"kind"`
	Key     string   `json:"key,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Is      string   `json:"string,omitempty"`
}

var MemberCountFilterRegex = regexp.MustCompile("^(==|[<>]=?)?([0-9]+)$")

func (cond *PushCondition) Match(room *rooms.Room, event *gomatrix.Event) bool {
	switch cond.Kind {
	case KindEventMatch:
		return cond.matchValue(room, event)
	case KindContainsDisplayName:
		return cond.matchDisplayName(room, event)
	case KindRoomMemberCount:
		return cond.matchMemberCount(room, event)
	default:
		return false
	}
}

func (cond *PushCondition) matchValue(room *rooms.Room, event *gomatrix.Event) bool {
	index := strings.IndexRune(cond.Key, '.')
	key := cond.Key
	subkey := ""
	if index > 0 {
		subkey = key[index+1:]
		key = key[0:index]
	}

	pattern, err := glob.Compile(cond.Pattern)
	if err != nil {
		return false
	}

	switch key {
	case "type":
		return pattern.MatchString(event.Type)
	case "sender":
		return pattern.MatchString(event.Sender)
	case "room_id":
		return pattern.MatchString(event.RoomID)
	case "state_key":
		if event.StateKey == nil {
			return cond.Pattern == ""
		}
		return pattern.MatchString(*event.StateKey)
	case "content":
		val, _ := event.Content[subkey].(string)
		return pattern.MatchString(val)
	default:
		return false
	}
}

func (cond *PushCondition) matchDisplayName(room *rooms.Room, event *gomatrix.Event) bool {
	member := room.GetMember(room.Owner)
	if member == nil {
		return false
	}
	text, _ := event.Content["body"].(string)
	return strings.Contains(text, member.DisplayName)
}

func (cond *PushCondition) matchMemberCount(room *rooms.Room, event *gomatrix.Event) bool {
	groupGroups := MemberCountFilterRegex.FindAllStringSubmatch(cond.Is, -1)
	if len(groupGroups) != 1 {
		return false
	}

	operator := "=="
	wantedMemberCount := 0

	group := groupGroups[0]
	if len(group) == 0 {
		return false
	} else if len(group) == 1 {
		wantedMemberCount, _ = strconv.Atoi(group[0])
	} else {
		operator = group[0]
		wantedMemberCount, _ = strconv.Atoi(group[1])
	}

	memberCount := len(room.GetMembers())

	switch operator {
	case "==":
		return wantedMemberCount == memberCount
	case ">":
		return wantedMemberCount > memberCount
	case ">=":
		return wantedMemberCount >= memberCount
	case "<":
		return wantedMemberCount < memberCount
	case "<=":
		return wantedMemberCount <= memberCount
	default:
		return false
	}
}
