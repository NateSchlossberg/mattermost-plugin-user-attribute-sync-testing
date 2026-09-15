package sync

import "github.com/mattermost/mattermost/server/public/pluginapi"

// Summary counts outcomes from one sync of an attributes document.
type Summary struct {
	FieldsCreated   int `json:"fieldsCreated"`
	FieldsUpdated   int `json:"fieldsUpdated"`
	FieldsDeleted   int `json:"fieldsDeleted"`
	FieldsSkipped   int `json:"fieldsSkipped"`
	UsersSynced     int `json:"usersSynced"`
	UsersSkipped    int `json:"usersSkipped"`
	ChannelsSynced  int `json:"channelsSynced"`
	ChannelsSkipped int `json:"channelsSkipped"`
}

// SyncDocument syncs fields from the document, then user values, using the
// field-ID cache from the first pass. The cache is not returned: nothing
// outside a single sync reads it.
//
//nolint:revive
func SyncDocument(client *pluginapi.Client, groupID, pluginID string, doc AttributesDocument) (Summary, error) {
	var summary Summary

	cache, err := SyncFields(client, groupID, pluginID, doc.Fields.User, &summary)
	if err != nil {
		return summary, err
	}

	if err := SyncUsers(client, groupID, doc.Users, cache, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}
