package reaper

import (
	//"fmt"
	//"github.com/awslabs/aws-sdk-go/aws"
	//"github.com/awslabs/aws-sdk-go/service/ec2"
	. "github.com/tj/go-debug"
)

var (
	debugAWS = Debug("reaper:aws")
	debugAll = Debug("reaper:aws:AllInstances")
)

type SecurityGroups []*SecurityGroup
type SecurityGroup struct {
	id    string
	name  string
	owner string
	vpcId string

	tags map[string]string

	reaper *State
}

func NewSecurityGroup() *SecurityGroup {
	return nil
}
