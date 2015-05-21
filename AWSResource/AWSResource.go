package AWSResource

import (
	"time"
)

type resourceTypeEnum int

const (
	INSTANCE resourceTypeEnum = iota
	SECURITYGROUP
)

type AWSResource struct {
	Id           string
	Name         string
	ResourceType resourceTypeEnum
	Region       string
	State        string
	VpcID        string
	LaunchTime   time.Time
	Owned        bool

	Tags map[string]string

	ReaperVisible  bool
	ReaperStarted  bool
	ReaperNotified bool
	ReaperIgnored  bool
}

func (a *AWSResource) Tagged(tag string) bool {
	_, ok := a.Tags[tag]
	return ok
}
