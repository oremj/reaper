package state

import (
	"strings"
	"time"
)

const (
	STATE_START StateEnum = iota
	STATE_NOTIFY1
	STATE_NOTIFY2
	STATE_IGNORE
	ReaperTag     = "REAPER"
	TagSeparator  = "|"
	TagTimeFormat = "2006-01-02 03:04PM MST"
)

type StateEnum int

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
	return s.State.String() + TagSeparator + s.Until.Format(TagTimeFormat)
}

func ParseState(state string) (defaultState *State) {

	defaultState = &State{STATE_START, time.Time{}}

	if state == "" {
		return
	}

	s := strings.Split(state, TagSeparator)

	if len(s) != 2 {
		return
	}

	var stateEnum StateEnum
	switch s[0] {
	case "start":
		stateEnum = STATE_START
	case "notify1":
		stateEnum = STATE_NOTIFY1
	case "notify2":
		stateEnum = STATE_NOTIFY2
	case "ignore":
		stateEnum = STATE_IGNORE
	default:
		return
	}

	t, err := time.Parse(TagTimeFormat, s[1])
	if err != nil {
		return
	}

	return &State{stateEnum, t}
}
