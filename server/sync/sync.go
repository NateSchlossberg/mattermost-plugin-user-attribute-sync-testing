package sync

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
