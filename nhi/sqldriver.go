package nhi

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"strings"
)

type ConnectorConfig struct {
	Provider    *Provider
	DSNTemplate string
	PasswordKey string
	UsernameKey string
	BaseDriver  driver.Driver
}

type Connector struct {
	cfg ConnectorConfig
}

func NewConnector(cfg ConnectorConfig) (*Connector, error) {
	if cfg.Provider == nil {
		return nil, fmt.Errorf("nhi: provider is required")
	}
	if cfg.DSNTemplate == "" {
		return nil, fmt.Errorf("nhi: DSN template is required")
	}
	if cfg.BaseDriver == nil {
		return nil, fmt.Errorf("nhi: base driver is required")
	}
	if cfg.PasswordKey == "" {
		cfg.PasswordKey = "password"
	}
	if cfg.UsernameKey == "" {
		cfg.UsernameKey = "username"
	}
	return &Connector{cfg: cfg}, nil
}

func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	dsn, err := c.buildDSN()
	if err != nil {
		return nil, fmt.Errorf("nhi: build DSN: %w", err)
	}

	if driverCtx, ok := c.cfg.BaseDriver.(driver.DriverContext); ok {
		connector, err := driverCtx.OpenConnector(dsn)
		if err != nil {
			return nil, fmt.Errorf("nhi: open connector: %w", err)
		}
		return connector.Connect(ctx)
	}

	conn, err := c.cfg.BaseDriver.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("nhi: open connection: %w", err)
	}
	return conn, nil
}

func (c *Connector) Driver() driver.Driver {
	return c.cfg.BaseDriver
}

func (c *Connector) buildDSN() (string, error) {
	creds, err := c.cfg.Provider.GetCredentials()
	if err != nil {
		return "", err
	}

	dsn := c.cfg.DSNTemplate
	for k, v := range creds {
		placeholder := "{{" + k + "}}"
		dsn = strings.ReplaceAll(dsn, placeholder, v)
	}

	if strings.Contains(dsn, "{{") {
		return "", fmt.Errorf("DSN template has unresolved placeholders")
	}
	return dsn, nil
}

type PoolConnector struct {
	*Connector
}

func NewPoolConnector(cfg ConnectorConfig) (*PoolConnector, error) {
	c, err := NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	return &PoolConnector{Connector: c}, nil
}

func OpenSQLDB(cfg ConnectorConfig) (*sql.DB, error) {
	connector, err := NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

var _ net.Addr = (*fakeAddr)(nil)

type fakeAddr struct{}

func (fakeAddr) Network() string { return "nhi" }
func (fakeAddr) String() string  { return "nhi-managed" }
