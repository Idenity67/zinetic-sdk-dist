package credential

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
)

type ConnectorConfig struct {
	Store            *MemStore
	CredentialKey    string
	DSNTemplate      string
	UnderlyingDriver driver.Driver
}

type ZineticConnector struct {
	cfg ConnectorConfig
}

func NewConnector(cfg ConnectorConfig) (*ZineticConnector, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("credential store is required")
	}
	if cfg.CredentialKey == "" {
		return nil, fmt.Errorf("credential key is required")
	}
	if cfg.DSNTemplate == "" {
		return nil, fmt.Errorf("DSN template is required")
	}
	if !strings.Contains(cfg.DSNTemplate, "${CREDENTIAL}") {
		return nil, fmt.Errorf("DSN template must contain ${CREDENTIAL} placeholder")
	}
	if cfg.UnderlyingDriver == nil {
		return nil, fmt.Errorf("underlying driver is required")
	}
	return &ZineticConnector{cfg: cfg}, nil
}

func (c *ZineticConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cred, ok := c.cfg.Store.Retrieve(c.cfg.CredentialKey)
	if !ok {
		return nil, fmt.Errorf("credential %q not found in store", c.cfg.CredentialKey)
	}

	dsn := strings.Replace(c.cfg.DSNTemplate, "${CREDENTIAL}", string(cred), 1)

	zeroize(cred)

	conn, err := c.cfg.UnderlyingDriver.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("open connection with fresh credential: %w", err)
	}
	return conn, nil
}

func (c *ZineticConnector) Driver() driver.Driver {
	return c.cfg.UnderlyingDriver
}
