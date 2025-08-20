package fsm

import (
	"errors"
	"fmt"
)

func NewFSM(initial State) *FSM {
	return &FSM{
		initial:      initial,
		current:      initial,
		transitions:  make(map[State][]transition),
		onEnter:      make(map[State][]Callback),
		onExit:       make(map[State][]Callback),
		onTransition: make([]TransitionCallback, 0),
		ctx:          newFSMContext(initial),
	}
}

func (fsm *FSM) Copy() *FSM {
	return &FSM{
		initial:      fsm.initial,
		current:      fsm.initial,
		transitions:  fsm.transitions,
		onEnter:      fsm.onEnter,
		onExit:       fsm.onExit,
		onTransition: fsm.onTransition,
		ctx:          newFSMContext(fsm.initial),
	}
}

func (fsm *FSM) GetContext() *FSMContext {
	return fsm.ctx
}

func (fsm *FSM) GetCurrent() State {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	return fsm.current
}

func (fsm *FSM) SetState(state State) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	fsm.current = state
	fsm.ctx.State = state
}

func (fsm *FSM) TransitionWhen(from State, event Event, to State, guard GuardFunc) *FSM {
	_, ok := fsm.transitions[from]
	if !ok {
		fsm.transitions[from] = make([]transition, 0)
	}
	fsm.transitions[from] = append(fsm.transitions[from], transition{event, to, guard})

	return fsm
}

func (fsm *FSM) Transition(from State, event Event, to State) *FSM {
	return fsm.TransitionWhen(from, event, to, nil)
}

func (fsm *FSM) OnEnter(state State, cb Callback) *FSM {
	_, ok := fsm.onEnter[state]
	if !ok {
		fsm.onEnter[state] = make([]Callback, 0)
	}
	fsm.onEnter[state] = append(fsm.onEnter[state], cb)

	return fsm
}

func (fsm *FSM) OnExit(state State, cb Callback) *FSM {
	_, ok := fsm.onExit[state]
	if !ok {
		fsm.onExit[state] = make([]Callback, 0)
	}
	fsm.onExit[state] = append(fsm.onExit[state], cb)

	return fsm
}

func (fsm *FSM) OnTransition(cb TransitionCallback) *FSM {
	fsm.onTransition = append(fsm.onTransition, cb)
	return fsm
}

func (fsm *FSM) Trigger(event Event, input ...any) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if len(input) > 0 {
		fmt.Printf("input is %v", input)
		fsm.ctx.Input = input[0]
		fmt.Printf("now input is %v", fsm.ctx.Input)
	} else {
		fsm.ctx.Input = nil
	}

	transitions, ok := fsm.transitions[fsm.current]
	if !ok {
		return errors.New("bad transition")
	}

	mustTransit := make([]transition, 0)

	for _, transition := range transitions {
		fmt.Printf("transition to %s, event %s, triggered event %s\n", transition.to, transition.event, event)
		if transition.event == event && (transition.guard == nil || transition.guard(fsm.ctx)) {
			mustTransit = append(mustTransit, transition)
		}
	}

	if len(mustTransit) != 1 {
		return fmt.Errorf("ambigous transitions. must be 1, found %d", len(mustTransit))
	}

	t := mustTransit[0]
	prevState := fsm.current
	nextState := t.to

	fsm.ctx.State = prevState

	if onExitCallbackList, ok := fsm.onExit[prevState]; ok {
		for _, cb := range onExitCallbackList {
			if err := cb(fsm.ctx); err != nil {
				return err
			}
		}
	}

	fsm.ctx.State = nextState

	for _, trCb := range fsm.onTransition {
		if err := trCb(prevState, nextState, event, fsm.ctx); err != nil {
			return err
		}
	}

	if onEnterCallbackList, ok := fsm.onEnter[nextState]; ok {
		for _, cb := range onEnterCallbackList {
			if err := cb(fsm.ctx); err != nil {
				return err
			}
		}
	}

	fsm.current = nextState

	return nil
}

func (fsm *FSM) CallEnter(state State) error {
	fsm.ctx.State = state

	if onEnterCallbackList, ok := fsm.onEnter[state]; ok {
		for _, cb := range onEnterCallbackList {
			if err := cb(fsm.ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
