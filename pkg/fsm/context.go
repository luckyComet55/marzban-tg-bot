package fsm

type FSMContext struct {
	State State
	Input any
	Data  map[string]any
	Meta  map[string]any
}

func newFSMContext(initial State) *FSMContext {
	return &FSMContext{
		State: initial,
		Input: nil,
		Data:  make(map[string]any),
		Meta:  make(map[string]any),
	}
}
