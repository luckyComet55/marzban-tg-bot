package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	pcl "github.com/luckyComet55/marzban-proto-contract/gen/go/contract"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type UserShortData struct {
	Username string
}

type UserRepository interface {
	GetUsers() ([]UserShortData, error)
	CreateUser(user UserCreateData) (UserData, error)
}

type userRepository struct {
	logger *slog.Logger
	client pcl.MarzbanManagementPanelClient
}

func (repo *userRepository) GetUsers() ([]UserShortData, error) {
	usersStream, err := repo.client.ListUsers(context.Background(), &emptypb.Empty{})
	if err != nil {
		repo.logger.Error(err.Error())
		return nil, err
	}
	users := make([]UserShortData, 0)
	for {
		user, err := usersStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			repo.logger.Error(err.Error())
			return nil, err
		}
		users = append(users, UserShortData{
			Username: user.Username,
		})
	}
	return users, nil
}

func (repo *userRepository) CreateUser(user UserCreateData) (UserData, error) {
	userData, err := repo.client.CreateUser(context.Background(), &pcl.CreateUserInfo{
		Username:      user.Username,
		ProxyProtocol: user.ProxyProtocol,
	})
	if err != nil {
		repo.logger.Error(err.Error())
		var userError error
		if status.Code(err) == codes.AlreadyExists || status.Code(err) == codes.InvalidArgument {
			userError = err
		} else {
			userError = fmt.Errorf("Some error occured, try again later")
		}
		return UserData{}, userError
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
