package sync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// AttributesStoreKey holds what the HTTP handlers in server/http_hooks.go uploaded: the file, and
// the timestamp that tells this provider the file is new.
const AttributesStoreKey = "attributes"

// StoredAttributes is the value under AttributesStoreKey.
//
// Data is the file exactly as it was uploaded, so a download returns what the admin gave us.
type StoredAttributes struct {
	LastUpdated time.Time `json:"lastUpdated"`
	Data        []byte    `json:"data"`
}

// ReadStoredAttributes reads the stored file and its timestamp.
func ReadStoredAttributes(client *pluginapi.Client) (StoredAttributes, error) {
	// Unmarshal here rather than letting KV.Get do it, so an unreachable store and a corrupt value
	// are not reported as the same error.
	var raw []byte
	if err := client.KV.Get(AttributesStoreKey, &raw); err != nil {
		return StoredAttributes{}, fmt.Errorf("failed to read %s from the KV store: %w", AttributesStoreKey, err)
	}

	if len(raw) == 0 {
		return StoredAttributes{}, nil
	}

	var stored StoredAttributes
	if err := json.Unmarshal(raw, &stored); err != nil {
		return StoredAttributes{}, fmt.Errorf("malformed data in %s: %w", AttributesStoreKey, err)
	}

	return stored, nil
}

// KVStoreProvider reads user attribute data an admin uploaded through the System Console, which
// the plugin stores in Mattermost's KV store. The KV store lives in Mattermost's Postgres
// database, which is a more stable storage location than file systems in a cloud environment.
//
// Incremental synchronization relies on a stored timestamp: GetUserAttributes returns an empty
// slice until the stored timestamp moves past the last one it processed.
type KVStoreProvider struct {
	client *pluginapi.Client

	// lastTimestampSynced is the latest file timestamp that was processed. It is in-memory only, so a
	// plugin restart re-syncs the stored file once.
	lastTimestampSynced time.Time
}

func NewKVStoreProvider(client *pluginapi.Client) *KVStoreProvider {
	return &KVStoreProvider{
		client: client,
	}
}

// GetUserAttributes reads the user attribute data an admin uploaded, from the KV store.
// On the first call, it returns everything stored.
// On subsequent calls, it checks whether a newer file has been uploaded since the last read:
//   - If newer: reads and returns the updated user data
//   - If unchanged: returns an empty array to signal no new data
func (f *KVStoreProvider) GetUserAttributes() ([]map[string]interface{}, error) {
	// One read gets the file and the timestamp together, so the file is fetched even when it turns
	// out to be unchanged. Keeping it all in one key keeps key management simple; to support very
	// large files read frequently, this can be broken out into separate keys if necessary.
	stored, err := ReadStoredAttributes(f.client)
	if err != nil {
		return nil, err
	}

	// No file has ever been uploaded, or the last one was deleted
	if len(stored.Data) == 0 {
		return nil, fmt.Errorf("no attributes document in the KV store: %s is unset — upload one from the System Console", AttributesStoreKey)
	}

	// Nothing new since the last read, so there is no work to do
	// Note we are checking if the stored file is before OR equal to the last sync, hence the !After call.
	if stored.LastUpdated.IsZero() || !stored.LastUpdated.After(f.lastTimestampSynced) {
		return []map[string]interface{}{}, nil
	}

	// Update the sync after a successful read but before validation so we dont keep reading an invalid file
	f.lastTimestampSynced = stored.LastUpdated

	return ParseAttributesDocument(stored.Data)
}
