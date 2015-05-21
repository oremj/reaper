package filter

import (
	. "github.com/mostlygeek/reaper/AWSResource"
	"time"
)

type FilterFunc func(AWSResource) bool

func Owned(a AWSResource) bool    { return a.Owned }
func NotOwned(a AWSResource) bool { return !a.Owned }

func AutoScaled(a AWSResource) bool {
	return a.Tagged("aws:autoscalang:groupName")
}
func NotAutoscaled(a AWSResource) bool { return !AutoScaled(a) }

func Id(id string) FilterFunc {
	return func(a AWSResource) bool {
		return a.Id == id
	}
}

func Not(f FilterFunc) FilterFunc {
	return func(a AWSResource) bool {
		return !f(a)
	}
}

func Tagged(tag string) FilterFunc {
	return func(a AWSResource) bool {
		return a.Tagged(tag)
	}
}

func LaunchTimeEqual(time time.Time) FilterFunc {
	return func(a AWSResource) bool {
		return a.LaunchTime.Equal(time)
	}
}

func LaunchTimeAfter(time time.Time) FilterFunc {
	return func(a AWSResource) bool {
		return a.LaunchTime.After(time)
	}
}

func LaunchTimeBefore(time time.Time) FilterFunc {
	return func(a AWSResource) bool {
		return a.LaunchTime.Before(time)
	}
}

func Running(i AWSResource) bool {
	return i.State == "running"
}

// ReaperReady creates a FilterFunc that checks if the instance is qualified
// additional reaper work
func ReaperReady(runningTime time.Duration) FilterFunc {
	return func(i AWSResource) bool {
		if i.ReaperStarted {
			return i.LaunchTime.Add(runningTime).Before(time.Now())
		} else {
			return i.ReaperVisible
		}
	}
}
