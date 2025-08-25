package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	pcl "github.com/luckyComet55/marzban-proto-contract/gen/go/contract"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProxyData struct {
	ProxyName string
}

type ProxyRepository interface {
	ListProxies() ([]ProxyData, error)
}

type proxyRepository struct {
	logger *slog.Logger
	client pcl.MarzbanManagementPanelClient
}

func NewProxyRepository(client pcl.MarzbanManagementPanelClient, logger *slog.Logger) ProxyRepository {
	return &proxyRepository{
		logger: logger,
		client: client,
	}
}

func (pr *proxyRepository) ListProxies() ([]ProxyData, error) {
	proxyStream, err := pr.client.ListProxies(context.Background(), &emptypb.Empty{})
	if err != nil {
		pr.logger.Error(err.Error(), "method", "ListProxies")
		return nil, fmt.Errorf("Unexpected error, try again later")
	}
	proxies := make([]ProxyData, 0)
	for {
		proxy, err := proxyStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			pr.logger.Error(err.Error(), "method", "ListProxies")
			return nil, fmt.Errorf("Unexpected error, try again later")
		}
		proxies = append(proxies, ProxyData{
			ProxyName: proxy.ProxyName,
		})
	}
	return proxies, nil
}
