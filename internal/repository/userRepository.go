package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"

	pcl "github.com/luckyComet55/marzban-proto-contract/gen/go/contract"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type UserCreateData struct {
	Username      string
	ProxyProtocol string
}

type UserData struct {
	UserCreateData
	UsedTraffic int64
	ConfigUrl   string
}

type UserRepository interface {
	GetUsers() ([]UserData, error)
	CreateUser(user UserCreateData) (UserData, error)
}

type userRepository struct {
	logger *slog.Logger
	client pcl.MarzbanManagementPanelClient
}

func (repo *userRepository) GetUsers() ([]UserData, error) {
	usersStream, err := repo.client.ListUsers(context.Background(), &emptypb.Empty{})
	if err != nil {
		repo.logger.Error(err.Error())
		return nil, err
	}
	users := make([]UserData, 0)
	for {
		user, err := usersStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			repo.logger.Error(err.Error())
			return nil, err
		}
		users = append(users, UserData{
			UserCreateData: UserCreateData{user.Username, user.Status},
			UsedTraffic:    int64(user.UsedTraffic),
			ConfigUrl:      user.ConfigUrls[0],
		})
	}
	return users, nil
}

func (repo *userRepository) CreateUser(user UserCreateData) (UserData, error) {
	users, err := repo.GetUsers()
	if err != nil {
		return UserData{}, err
	}
	if slices.ContainsFunc(users, func(u UserData) bool {
		return u.Username == user.Username
	}) {
		return UserData{}, fmt.Errorf("user with name %s already exists", user.Username)
	}
	userData, err := repo.client.CreateUser(context.Background(), &pcl.CreateUserInfo{
		Username:      user.Username,
		ProxyProtocol: user.ProxyProtocol,
	})
	if err != nil {
		repo.logger.Error(err.Error())
		return UserData{}, err
	}
	userMappedData := UserData{
		UserCreateData: UserCreateData{
			userData.Username,
			userData.Status,
		},
		UsedTraffic: int64(userData.UsedTraffic),
		ConfigUrl:   userData.ConfigUrls[0],
	}
	return userMappedData, nil
}

func NewUserRepository(client pcl.MarzbanManagementPanelClient, logger *slog.Logger) UserRepository {
	return &userRepository{
		client: client,
		logger: logger,
	}
}
