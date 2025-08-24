package repository

import (
	"log/slog"
)

type ProxyData struct {
	ProxyName string
}

type ProxyRepository interface {
	ListProxies() ([]ProxyData, error)
}

type proxyRepository struct {
	logger  *slog.Logger
	proxies []ProxyData
}

func NewProxyRepository(logger *slog.Logger) ProxyRepository {
	return &proxyRepository{
		logger: logger,
		proxies: []ProxyData{
			{"vless_tls"},
			{"shadowsocks"},
		},
	}
}

func (pr *proxyRepository) ListProxies() ([]ProxyData, error) {
	pr.logger.Debug("calling list proxies")
	return pr.proxies, nil
}
