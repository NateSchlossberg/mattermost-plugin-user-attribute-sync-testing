package sync

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

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

// SyncDocument does not return the field-ID cache: nothing outside a single sync reads it.
//
//nolint:revive
func SyncDocument(client *pluginapi.Client, groupID, pluginID string, doc AttributesDocument) (Summary, error) {
	var summary Summary

	userCache, err := SyncFields(client, groupID, pluginID, model.PropertyFieldObjectTypeUser, doc.Fields.User, &summary)
	if err != nil {
		return summary, err
	}

	// The tier is a property of the server, not of the field, so one check
	// gates the whole channel pass. Skipped channel fields are still named by
	// the document, so the deletion pass below keeps them.
	channelCache := NewFieldIDCache()
	if len(doc.Fields.Channel) > 0 {
		if !channelFieldsLicensed(client) {
			client.Log.Warn("Skipping channel field sync: channel attributes require an Enterprise Advanced license",
				"field_count", len(doc.Fields.Channel))
			summary.FieldsSkipped += len(doc.Fields.Channel)
		} else if channelCache, err = SyncFields(client, groupID, pluginID, model.PropertyFieldObjectTypeChannel, doc.Fields.Channel, &summary); err != nil {
			return summary, err
		}
	}

	// One deletion pass per document, after every field pass: run from inside
	// SyncFields, a user-only keep-set would delete the channel fields.
	DeleteOmittedFields(client, groupID, pluginID, doc.Fields.User, doc.Fields.Channel, &summary)

	if err := SyncUsers(client, groupID, doc.Users, userCache, &summary); err != nil {
		return summary, err
	}

	// With the channel field pass skipped the cache is empty, so every channel
	// value is an unknown field and lands in ChannelsSkipped.
	if err := SyncChannels(client, groupID, doc.Channels, channelCache, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}
