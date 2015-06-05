package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/mostlygeek/reaper/state"
)

type SecurityGroups []*SecurityGroup
type SecurityGroup struct {
	AWSResource
}

func NewSecurityGroup(region string, sg *ec2.SecurityGroup) *SecurityGroup {
	s := SecurityGroup{
		AWSResource{
			id:          *sg.GroupID,
			name:        *sg.GroupName,
			region:      region,
			description: *sg.Description,
			vpc_id:      *sg.VPCID,
			owner_id:    *sg.OwnerID,
			tags:        make(map[string]string),
		},
	}

	for _, tag := range sg.Tags {
		s.tags[*tag.Key] = *tag.Value
	}

	s.reaper = state.ParseState(s.tags[state.ReaperTag])

	return &s
}
