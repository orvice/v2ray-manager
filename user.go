package v2raymanager

type User interface {
	GetEmail() string
	GetUUID() string
	GetAlterID() uint32
	GetLevel() uint32
}
