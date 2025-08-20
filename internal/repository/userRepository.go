package repository

import (
	"fmt"
	"log/slog"
	"slices"
)

type UserCreateData struct {
	Username      string
	ProxyProtocol string
}

type UserData struct {
	UserCreateData
	UsedTraffic int64
}

type UserRepository interface {
	GetUsers() ([]UserData, error)
	CreateUser(user UserCreateData) error
}

type userRepository struct {
	logger *slog.Logger
	users  []UserData
}

func (repo *userRepository) GetUsers() ([]UserData, error) {
	return repo.users, nil
}

func (repo *userRepository) CreateUser(user UserCreateData) error {
	if slices.ContainsFunc(repo.users, func(u UserData) bool {
		return u.Username == user.Username
	}) {
		return fmt.Errorf("user with name %s already exists", user.Username)
	}
	repo.users = append(repo.users, UserData{
		UserCreateData: user,
		UsedTraffic:    0,
	})
	return nil
}

func NewUserRepository(logger *slog.Logger) UserRepository {
	return &userRepository{
		logger: logger,
		users: []UserData{
			UserData{
				UserCreateData: UserCreateData{Username: "tvorec228", ProxyProtocol: "vless_tls"},
				UsedTraffic:    0,
			},
			UserData{
				UserCreateData: UserCreateData{Username: "andrewTannenbaum", ProxyProtocol: "vless_tls"},
				UsedTraffic:    0,
			},
			UserData{
				UserCreateData: UserCreateData{Username: "testUser1", ProxyProtocol: "vless_tls"},
				UsedTraffic:    0,
			},
		},
	}
}
