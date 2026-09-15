package sync

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestSyncClient(t *testing.T) (*pluginapi.Client, *plugintest.API) {
	t.Helper()

	api := &plugintest.API{}
	t.Cleanup(func() { api.AssertExpectations(t) })
	mockSyncLogs(api)

	return pluginapi.NewClient(api, &plugintest.Driver{}), api
}

func mockSyncLogs(api *plugintest.API) {
	const maxLogFields = 8
	for _, method := range []string{"LogDebug", "LogInfo", "LogWarn", "LogError"} {
		for n := 0; n <= maxLogFields; n++ {
			args := make([]interface{}, n+1)
			for i := range args {
				args[i] = mock.Anything
			}
			api.On(method, args...).Maybe()
		}
	}
}

func TestSyncDocument(t *testing.T) {
	groupID := "test-group-id"
	pluginID := "test-plugin-id"

	t.Run("creates a field then writes a value for it in one call", func(t *testing.T) {
		client, api := newTestSyncClient(t)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "job_title" && f.ID == ""
		})).Return(&model.PropertyField{ID: "generated_id_1", Name: "job_title", Type: model.PropertyFieldTypeText}, nil)
		mockEmptyFieldSearch(api, groupID)

		user := &model.User{Id: "user1", Email: "user1@example.com"}
		api.On("GetUserByEmail", "user1@example.com").Return(user, nil)
		api.On("UpsertPropertyValues", mock.MatchedBy(func(values []*model.PropertyValue) bool {
			return len(values) == 1 && values[0].FieldID == "generated_id_1" && values[0].TargetID == "user1"
		})).Return([]*model.PropertyValue{}, nil)

		doc := AttributesDocument{
			Version: SupportedDocumentVersion,
			Fields: FieldSchema{
				User: []FieldDefinition{{
					Name:        "job_title",
					DisplayName: "Job Title",
					Type:        model.PropertyFieldTypeText,
					AccessMode:  model.PropertyAccessModePublic,
				}},
			},
			Users: []map[string]interface{}{
				{"email": "user1@example.com", "job_title": "Engineer"},
			},
		}

		summary, err := SyncDocument(client, groupID, pluginID, doc)
		require.NoError(t, err)
		assert.Equal(t, 1, summary.FieldsCreated)
		assert.Equal(t, 0, summary.FieldsUpdated)
		assert.Equal(t, 0, summary.FieldsDeleted)
		assert.Equal(t, 0, summary.FieldsSkipped)
		assert.Equal(t, 1, summary.UsersSynced)
		assert.Equal(t, 0, summary.UsersSkipped)
		assert.Equal(t, 0, summary.ChannelsSynced)
		assert.Equal(t, 0, summary.ChannelsSkipped)
	})

	t.Run("nil error still reports skipped fields and users", func(t *testing.T) {
		client, api := newTestSyncClient(t)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "job_title"
		})).Return(&model.PropertyField{ID: "generated_id_1", Name: "job_title", Type: model.PropertyFieldTypeText}, nil)
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "broken"
		})).Return(nil, errors.New("create failed"))
		mockEmptyFieldSearch(api, groupID)

		api.On("GetUserByEmail", "user1@example.com").Return(&model.User{Id: "user1", Email: "user1@example.com"}, nil)
		api.On("UpsertPropertyValues", mock.Anything).Return([]*model.PropertyValue{}, nil)

		doc := AttributesDocument{
			Version: SupportedDocumentVersion,
			Fields: FieldSchema{
				User: []FieldDefinition{
					{
						Name:        "job_title",
						DisplayName: "Job Title",
						Type:        model.PropertyFieldTypeText,
						AccessMode:  model.PropertyAccessModePublic,
					},
					{
						Name:        "broken",
						DisplayName: "Broken",
						Type:        model.PropertyFieldTypeText,
						AccessMode:  model.PropertyAccessModePublic,
					},
				},
			},
			Users: []map[string]interface{}{
				{"email": "user1@example.com", "job_title": "Engineer"},
				{"job_title": "No Email"},
			},
		}

		summary, err := SyncDocument(client, groupID, pluginID, doc)
		require.NoError(t, err)
		assert.Equal(t, 1, summary.FieldsCreated)
		assert.Equal(t, 1, summary.FieldsSkipped)
		assert.Equal(t, 1, summary.UsersSynced)
		assert.Equal(t, 1, summary.UsersSkipped)
		assert.Equal(t, 0, summary.ChannelsSynced)
		assert.Equal(t, 0, summary.ChannelsSkipped)
	})
}
