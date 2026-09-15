package sync

import (
	"encoding/json"
	"fmt"
)

// SupportedDocumentVersion is the only value of the document's `version` key that this plugin
// accepts. Bumping it is how a future breaking change to the document format is announced,
// the same way this version broke the previous bare-array format.
const SupportedDocumentVersion = 2

// attributesDocument is the uploaded file's shape: a version marker plus user records.
// encoding/json ignores keys a struct does not name, so a document carrying fields or channels
// round-trips unchanged rather than being rejected.
type attributesDocument struct {
	Version int                      `json:"version"`
	Users   []map[string]interface{} `json:"users"`
}

// ParseAttributesDocument decodes raw uploaded bytes into the document's user records, rejecting
// anything that is not a JSON object, carries an unsupported version, or has a malformed record.
func ParseAttributesDocument(raw []byte) ([]map[string]interface{}, error) {
	var doc attributesDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("not a valid attributes document: %w", err)
	}

	if doc.Version != SupportedDocumentVersion {
		return nil, fmt.Errorf("unsupported document version %d: only version %d is supported", doc.Version, SupportedDocumentVersion)
	}

	return doc.Users, nil
}
