package aws

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/mostlygeek/reaper/state"
)

type Terminable interface {
	Terminate() (bool, error)
}

type Stoppable interface {
	Stop() (bool, error)
	ForceStop() (bool, error)
}

//                ,____
//                |---.\
//        ___     |    `
//       / .-\  ./=)
//      |  | |_/\/|
//      ;  |-;| /_|
//     / \_| |/ \ |
//    /      \/\( |
//    |   /  |` ) |
//    /   \ _/    |
//   /--._/  \    |
//   `/|)    |    /
//     /     |   |
//   .'      |   |
//  /         \  |
// (_.-.__.__./  /
// credit: jgs, http://www.chris.com/ascii/index.php?art=creatures/grim%20reapers

type Reapable interface {
	Terminable
	Stoppable
}

type ResourceState int

const (
	PENDING ResourceState = iota
	RUNNING
	SHUTTINGDOWN
	TERMINATED
	STOPPING
	STOPPED
)

// basic AWS resource, has properties that most/all resources have
type AWSResource struct {
	id          string
	name        string
	region      string
	state       ResourceState
	description string
	vpc_id      string
	owner_id    string

	tags map[string]string

	// reaper state
	reaper *state.State
}

func (a *AWSResource) Tagged(tag string) bool {
	_, ok := a.tags[tag]
	return ok
}

func (a *AWSResource) Id() string           { return a.id }
func (a *AWSResource) Name() string         { return a.name }
func (a *AWSResource) Region() string       { return a.region }
func (a *AWSResource) State() ResourceState { return a.state }
func (a *AWSResource) Reaper() *state.State { return a.reaper }

// filter funcs for ResourceState
func (a *AWSResource) Pending() bool      { return a.state == PENDING }
func (a *AWSResource) Running() bool      { return a.state == RUNNING }
func (a *AWSResource) ShuttingDown() bool { return a.state == SHUTTINGDOWN }
func (a *AWSResource) Terminated() bool   { return a.state == TERMINATED }
func (a *AWSResource) Stopping() bool     { return a.state == STOPPING }
func (a *AWSResource) Stopped() bool      { return a.state == STOPPED }

// Tag returns the tag's value or an empty string if it does not exist
func (a *AWSResource) Tag(t string) string { return a.tags[t] }

func (a *AWSResource) Owned() bool { return a.Tagged("Owner") }

// Owner extracts useful information out of the Owner tag which should
// be parsable by mail.ParseAddress
func (a *AWSResource) Owner() *mail.Address {
	// properly formatted email
	if addr, err := mail.ParseAddress(a.Tag("Owner")); err == nil {
		return addr
	}

	// username -> mozilla.com email
	if addr, err := mail.ParseAddress(fmt.Sprintf("%s@mozilla.com", a.Tag("Owner"))); a.Tagged("Owner") && err == nil {
		return addr
	}

	return nil
}

func (a *AWSResource) UpdateReaperState(s *state.State) {
	a.reaper = s
}
func (a *AWSResource) ReaperVisible() bool {
	return time.Now().After(a.reaper.Until)
}
func (a *AWSResource) ReaperStarted() bool {
	return a.reaper.State == state.STATE_START
}
func (a *AWSResource) ReaperNotified(notifyNum int) bool {
	if notifyNum == 1 {
		return a.reaper.State == state.STATE_NOTIFY1
	} else if notifyNum == 2 {
		return a.reaper.State == state.STATE_NOTIFY2
	} else {
		return false
	}
}

func (a *AWSResource) ReaperIgnored() bool {
	return a.reaper.State == state.STATE_IGNORE
}

func UpdateReaperState(region, id string, newState *state.State) error {
	req := &ec2.CreateTagsInput{
		DryRun:    aws.Boolean(false),
		Resources: []*string{aws.String(id)},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String(state.ReaperTag),
				Value: aws.String(newState.String()),
			},
		},
	}

	api := ec2.New(&aws.Config{Region: region})
	_, err := api.CreateTags(req)
	return err
}
