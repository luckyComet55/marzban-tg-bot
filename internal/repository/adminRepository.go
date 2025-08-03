package repository

import "sync"

type adminState int

const (
	ADMIN_STATE_DEFAULT adminState = iota
	ADMIN_STATE_CREATE_USER_INPUT_NAME
	ADMIN_STATE_CREATE_USER_SELECT_PROXY
	ADMIN_STATE_CREATE_USER_SUBMIT_DATA
)

func (state adminState) String() string {
	switch state {
	case ADMIN_STATE_DEFAULT:
		return "DEFAULT"
	case ADMIN_STATE_CREATE_USER_INPUT_NAME:
		return "STATE_CREATE_USER_INPUT_NAME"
	case ADMIN_STATE_CREATE_USER_SELECT_PROXY:
		return "CREATE_USER_SELECT_PROXY"
	case ADMIN_STATE_CREATE_USER_SUBMIT_DATA:
		return "CREATE_USER_SUBMIT_DATA"
	}

	return ""
}

type AdminRepository interface {
	GetAdminState(int64) (adminState, bool)
	SetAdminState(int64, adminState)
}

type adminRepository struct {
	mu          sync.RWMutex
	adminStates map[int64]adminState
}

func (ar *adminRepository) GetAdminState(adminID int64) (adminState, bool) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	state, ok := ar.adminStates[adminID]
	return state, ok
}

func (ar *adminRepository) SetAdminState(adminID int64, state adminState) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.adminStates[adminID] = state
}

func NewAdminRepository() AdminRepository {
	return &adminRepository{
		adminStates: make(map[int64]adminState),
	}
}
