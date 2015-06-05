package events

import (
	"github.com/mostlygeek/reaper/aws"
	"github.com/mostlygeek/reaper/config"
)

type EventReporter interface {
	NewEvent(title string, text string, fields map[string]string, tags []string)
	NewStatistic(name string, value float64, tags []string)
	NewReapableInstanceEvent(i *aws.Instance)
	NewReapableASGEvent(a *aws.AutoScalingGroup)
}

// implements EventReporter but does nothing
type NoEventReporter struct{}

func (n NoEventReporter) NewEvent(title string, text string, fields map[string]string, tags []string) {
}
func (n NoEventReporter) NewStatistic(name string, value float64, tags []string) {}
func (n NoEventReporter) NewReapableInstanceEvent(i *aws.Instance)               {}
func (n NoEventReporter) NewReapableASGEvent(a *aws.AutoScalingGroup)            {}

type InstanceEventData struct {
	Config   *config.Config
	Instance *aws.Instance
}

type ASGEventData struct {
	Config *config.Config
	ASG    *aws.AutoScalingGroup
}
