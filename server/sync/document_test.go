package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseAttributesDocument_ValidDocument tests that a well-formed v2 document returns its
// user records.
func TestParseAttributesDocument_ValidDocument(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`{"version": 2, "users": [
		{"email": "user1@example.com", "job_title": "Engineer"},
		{"email": "user2@example.com", "job_title": "Sales"}
	]}`))

	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1@example.com", users[0]["email"])
	assert.Equal(t, "Engineer", users[0]["job_title"])
	assert.Equal(t, "user2@example.com", users[1]["email"])
	assert.Equal(t, "Sales", users[1]["job_title"])
}

// TestParseAttributesDocument_BareArrayRejected tests that the previous format — a bare JSON
// array of user records — is no longer accepted.
func TestParseAttributesDocument_BareArrayRejected(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`[{"email": "user1@example.com"}]`))

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "not a valid attributes document")
}

// TestParseAttributesDocument_UnsupportedVersionRejected tests that a document naming a version
// this plugin does not support is rejected, and the error says which version is supported.
func TestParseAttributesDocument_UnsupportedVersionRejected(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`{"version": 1, "users": []}`))

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "unsupported document version 1")
	assert.Contains(t, err.Error(), "only version 2 is supported")
}

// TestParseAttributesDocument_MissingVersionRejected tests that a document with no version key
// is rejected the same way an unsupported one is.
func TestParseAttributesDocument_MissingVersionRejected(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`{"users": []}`))

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "unsupported document version 0")
}

// TestParseAttributesDocument_UsersAbsent tests that a document with no users list is valid and
// parses with no records, since a document can carry only field definitions.
func TestParseAttributesDocument_UsersAbsent(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`{"version": 2}`))

	require.NoError(t, err)
	assert.Empty(t, users)
}

// TestParseAttributesDocument_NonObjectUserRejected tests that a users entry that is not a JSON
// object is rejected.
func TestParseAttributesDocument_NonObjectUserRejected(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`{"version": 2, "users": ["not an object"]}`))

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "not a valid attributes document")
}

// TestParseAttributesDocument_NotJSONRejected tests that text that is not JSON at all is
// rejected.
func TestParseAttributesDocument_NotJSONRejected(t *testing.T) {
	users, err := ParseAttributesDocument([]byte(`not json`))

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "not a valid attributes document")
}
