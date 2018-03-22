package pushrules

import (
	"encoding/json"
	"net/url"

	"maunium.net/go/gomatrix"
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

// EventToPushRules converts a m.push_rules event to a PushRuleset by passing the data through JSON.
func EventToPushRules(event *gomatrix.Event) (*PushRuleset, error) {
	content, _ := event.Content["global"]
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	ruleset := &PushRuleset{}
	err = json.Unmarshal(raw, ruleset)
	if err != nil {
		return nil, err
	}

	return ruleset, nil
}
