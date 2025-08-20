package fsm

import "sync"

type (
	State              string
	Event              string
	GuardFunc          func(ctx *FSMContext) bool
	Callback           func(ctx *FSMContext) error
	TransitionCallback func(from, to State, event Event, ctx *FSMContext) error

	transition struct {
		event Event
		to    State
		guard GuardFunc
	}

	FSM struct {
		initial      State
		current      State
		transitions  map[State][]transition
		onExit       map[State][]Callback
		onEnter      map[State][]Callback
		onTransition []TransitionCallback

		ctx *FSMContext
		mu  sync.RWMutex
	}
)
