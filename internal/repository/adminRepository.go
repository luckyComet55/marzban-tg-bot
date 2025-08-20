package repository

import (
	"fmt"

	"github.com/luckyComet55/marzban-tg-bot/internal/fsm"
)

type AdminState int

const (
	ADMIN_STATE_DEFAULT                  fsm.State = "DEFAULT"
	ADMIN_STATE_SELECT_ACTION            fsm.State = "SELECT_ACTION"
	ADMIN_STATE_CREATE_USER_INPUT_NAME   fsm.State = "STATE_CREATE_USER_INPUT_NAME"
	ADMIN_STATE_CREATE_USER_SELECT_PROXY fsm.State = "CREATE_USER_SELECT_PROXY"
	ADMIN_STATE_CREATE_USER_SUBMIT_DATA  fsm.State = "CREATE_USER_SUBMIT_DATA"
)

type AdminRepository interface {
	GetAdminState(int64) (fsm.State, bool)
	CheckAdminExists(int64) (bool, error)
	AddAdmin(int64) error
	RemoveAdmin(int64) error
	SetAdminState(int64, fsm.State) error
	TriggerAdminTransition(int64, fsm.Event, ...any) error
	SetAdminMeta(int64, string, any) error
	GetAdminMeta(int64, string) (any, error)
	SetAdminData(int64, string, any) error
	GetAdminData(int64, string) (any, error)
}

type adminRepository struct {
	adminStates map[int64]*fsm.FSM
	fsmOrignial *fsm.FSM
}

func (ar *adminRepository) RemoveAdmin(adminID int64) error {
	delete(ar.adminStates, adminID)
	return nil
}

func (ar *adminRepository) GetAdminState(adminID int64) (fsm.State, bool) {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return "", ok
	}
	return fsm.GetCurrent(), ok
}

func (ar *adminRepository) AddAdmin(adminID int64) error {
	if _, ok := ar.adminStates[adminID]; ok {
		return fmt.Errorf("admin with ID %d already exists", adminID)
	}

	ar.adminStates[adminID] = ar.fsmOrignial.Copy()
	return nil
}

func (ar *adminRepository) CheckAdminExists(adminID int64) (bool, error) {
	_, ok := ar.adminStates[adminID]
	return ok, nil
}

func (ar *adminRepository) SetAdminState(adminID int64, state fsm.State) error {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	fsm.SetState(state)
	return fsm.CallEnter(state)
}

func (ar *adminRepository) SetAdminMeta(adminID int64, key string, value any) error {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	fsm.GetContext().Meta[key] = value
	return nil
}

func (ar *adminRepository) GetAdminMeta(adminID int64, key string) (any, error) {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return nil, fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	value, ok := fsm.GetContext().Meta[key]
	if !ok {
		return nil, fmt.Errorf("no meta with key: %s", key)
	}
	return value, nil

}

func (ar *adminRepository) SetAdminData(adminID int64, key string, value any) error {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	fsm.GetContext().Data[key] = value
	return nil
}

func (ar *adminRepository) GetAdminData(adminID int64, key string) (any, error) {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return nil, fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	value, ok := fsm.GetContext().Data[key]
	if !ok {
		return nil, fmt.Errorf("no data with key: %s", key)
	}
	return value, nil
}

func (ar *adminRepository) TriggerAdminTransition(adminID int64, event fsm.Event, input ...any) error {
	fsm, ok := ar.adminStates[adminID]
	if !ok {
		return fmt.Errorf("admin with ID %d does not exist", adminID)
	}

	return fsm.Trigger(event, input...)
}

func NewAdminRepository(f *fsm.FSM) AdminRepository {
	return &adminRepository{
		adminStates: make(map[int64]*fsm.FSM),
		fsmOrignial: f,
	}
}
