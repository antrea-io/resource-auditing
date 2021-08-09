package types

type TagRequestType string

const (
	TagCreate TagRequestType = "create"
	TagDelete TagRequestType = "delete"
)

type TagRequest struct {
	Type   TagRequestType `json:"type,omitempty"`
	Tag    string         `json:"tag,omitempty"`
	Sha    string         `json:"sha,omitempty"`
	Author string         `json:"author,omitempty"`
	Email  string         `json:"email,omitempty"`
}

type RollbackRequest struct {
	Tag string `json:"tag,omitempty"`
	Sha string `json:"sha,omitempty"`
}
