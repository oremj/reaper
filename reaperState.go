package reaper

import (
	// "fmt"
	"time"
)

type StateEnum int

const (
	STATE_START StateEnum = iota
	STATE_NOTIFY1
	STATE_NOTIFY2
	STATE_IGNORE
)

func (s StateEnum) String() string {
	switch s {
	case STATE_NOTIFY1:
		return "notify1"
	case STATE_NOTIFY2:
		return "notify2"
	case STATE_IGNORE:
		return "ignore"
	default:
		return "start"
	}
}

type State struct {
	State StateEnum

	// State must be maintained until this time
	Until time.Time
}

func (s *State) String() string {
	return s.State.String() + s_sep + s.Until.Format(s_tformat)
}
