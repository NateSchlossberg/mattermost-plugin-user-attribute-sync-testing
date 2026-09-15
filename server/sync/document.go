package sync

import (
	"encoding/json"
	"fmt"
)

// SupportedDocumentVersion is the only value of the document's `version` key that this plugin
// accepts. Bumping it is how a future breaking change to the document format is announced,
// the same way this version broke the previous bare-array format.
const SupportedDocumentVersion = 2

// FieldSchema is the document's `fields` object. Only `user` is named; a `channel` list, if
// present, is ignored by encoding/json the same way any other unnamed key is.
type FieldSchema struct {
	User []FieldDefinition `json:"user"`
}

// AttributesDocument is the uploaded file's shape: a version marker, field definitions, and
// user records. encoding/json ignores keys a struct does not name, so a document carrying
// channels (or a fields.channel list) round-trips unchanged rather than being rejected.
type AttributesDocument struct {
	Version int                      `json:"version"`
	Fields  FieldSchema              `json:"fields"`
	Users   []map[string]interface{} `json:"users"`
}

// ParseAttributesDocument decodes raw uploaded bytes into the document, rejecting anything that
// is not a JSON object, carries an unsupported version, or has a malformed record.
func ParseAttributesDocument(raw []byte) (AttributesDocument, error) {
	var doc AttributesDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return AttributesDocument{}, fmt.Errorf("not a valid attributes document: %w", err)
	}

	if doc.Version != SupportedDocumentVersion {
		return AttributesDocument{}, fmt.Errorf("unsupported document version %d: only version %d is supported", doc.Version, SupportedDocumentVersion)
	}

	return doc, nil
}
