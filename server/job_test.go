package main

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-user-attribute-sync-testing/server/sync"
)

// TestRunSync covers the sync job reading directly from the KV store provider.
func TestRunSync(t *testing.T) {
	newPlugin := func(t *testing.T) (*Plugin, *plugintest.API) {
		t.Helper()

		api := &plugintest.API{}
		t.Cleanup(func() { api.AssertExpectations(t) })
		mockLogs(api)

		client := pluginapi.NewClient(api, &plugintest.Driver{})

		return &Plugin{
			client:          client,
			attributeSource: sync.NewKVStoreProvider(client),
			fieldIDCache: &sync.FieldIDCache{
				FieldNameToID:  map[string]string{},
				OptionNameToID: map[string]string{},
			},
		}, api
	}

	t.Run("no stored document", func(t *testing.T) {
		p, api := newPlugin(t)
		api.On("KVGet", sync.AttributesStoreKey).Return(nil, nil).Once()

		p.runSync()

		// No further calls happen — proven by api.AssertExpectations finding nothing else
		// registered, since neither GetUserByEmail nor UpsertPropertyValues is expected.
	})

	t.Run("a stored document", func(t *testing.T) {
		p, api := newPlugin(t)
		api.On("KVGet", sync.AttributesStoreKey).
			Return(storedValue(t, time.Now(), []byte(`{"version": 2, "users": [{"email":"nobody@example.com"}]}`)), nil).Once()

		notFoundErr := model.NewAppError("GetUserByEmail", "app.user.get_by_email.app_error", nil, "", 404)
		api.On("GetUserByEmail", "nobody@example.com").Return(nil, notFoundErr).Once()

		p.runSync()

		// UpsertPropertyValues is not expected, so a call would fail the mock — proving the
		// per-user failure is logged and skipped rather than written.
	})
}
