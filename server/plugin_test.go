package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-user-attribute-sync-testing/server/sync"
)

// TestOnActivate covers the three stored-document outcomes: a KV read error
// fails activation; nothing stored or unparseable bytes skip field sync and
// still activate with an empty cache.
func TestOnActivate(t *testing.T) {
	newPlugin := func(t *testing.T) (*Plugin, *plugintest.API) {
		t.Helper()

		api := &plugintest.API{}
		t.Cleanup(func() { api.AssertExpectations(t) })

		p := &Plugin{
			MattermostPlugin: plugin.MattermostPlugin{
				API:    api,
				Driver: &plugintest.Driver{},
			},
		}
		t.Cleanup(func() {
			if p.backgroundJob != nil {
				require.NoError(t, p.backgroundJob.Close())
			}
		})

		return p, api
	}

	mockPropertyGroup := func(api *plugintest.API) {
		api.On("GetPropertyGroup", model.AccessControlPropertyGroupName).
			Return(&model.PropertyGroup{ID: "group-id"}, nil).Once()
	}

	// LastFinished is now so the scheduled job waits instead of calling runSync
	// immediately. Maybe: Close can win the race against the first lock.
	mockJobScheduler := func(api *plugintest.API) {
		meta, err := json.Marshal(struct{ LastFinished time.Time }{LastFinished: time.Now()})
		require.NoError(t, err)
		api.On("KVGet", "cron_AttributeSync").Return(meta, nil).Maybe()
		api.On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Maybe()
	}

	forbidFieldSync := func(t *testing.T, api *plugintest.API) {
		t.Helper()
		fail := func(mock.Arguments) { t.Fatal("SyncFields must not run") }
		api.On("LogInfo", "Syncing field definitions", mock.Anything, mock.Anything).Run(fail).Maybe()
		api.On("GetPropertyFieldByName", mock.Anything, mock.Anything, mock.Anything).Run(fail).Maybe()
	}

	requireEmptyCache := func(t *testing.T, p *Plugin) {
		t.Helper()
		require.NotNil(t, p.fieldIDCache)
		assert.Empty(t, p.fieldIDCache.FieldNameToID)
		assert.Empty(t, p.fieldIDCache.OptionNameToID)
		assert.Empty(t, p.fieldIDCache.FieldNameToType)
	}

	t.Run("KV read error fails activation", func(t *testing.T) {
		p, api := newPlugin(t)
		mockPropertyGroup(api)
		api.On("KVGet", sync.AttributesStoreKey).
			Return(nil, model.NewAppError("KVGet", "kv.get.app_error", nil, "connection refused", http.StatusInternalServerError)).Once()
		mockLogs(api)

		err := p.OnActivate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read stored attributes")
		assert.Nil(t, p.fieldIDCache)
		assert.Nil(t, p.backgroundJob)
	})

	t.Run("no stored document skips field sync", func(t *testing.T) {
		p, api := newPlugin(t)
		mockPropertyGroup(api)
		api.On("KVGet", sync.AttributesStoreKey).Return(nil, nil)
		api.On("LogInfo", "No attributes document stored, skipping field sync").Once()
		forbidFieldSync(t, api)
		mockJobScheduler(api)
		mockLogs(api)

		require.NoError(t, p.OnActivate())
		requireEmptyCache(t, p)
	})

	t.Run("unparseable document skips field sync and still activates", func(t *testing.T) {
		p, api := newPlugin(t)
		mockPropertyGroup(api)
		api.On("KVGet", sync.AttributesStoreKey).
			Return(storedValue(t, time.Now(), []byte("not json")), nil)
		api.On("LogError", "Failed to parse stored attributes document, skipping field sync", "error", mock.AnythingOfType("string")).Once()
		forbidFieldSync(t, api)
		mockJobScheduler(api)
		mockLogs(api)

		require.NoError(t, p.OnActivate())
		requireEmptyCache(t, p)
	})
}
