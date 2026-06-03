package credential

import (
	"context"
	"database/sql/driver"
	"testing"
)

type fakeDriver struct{}

type fakeConn struct{}

func (d *fakeDriver) Open(dsn string) (driver.Conn, error) {
	return &fakeConn{}, nil
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (c *fakeConn) Close() error                              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                 { return nil, nil }

func TestConnector_InjectsCredential(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("db_pass", []byte("s3cr3t"))

	connector, err := NewConnector(ConnectorConfig{
		Store:            store,
		CredentialKey:    "db_pass",
		DSNTemplate:      "postgres://user:${CREDENTIAL}@host/db",
		UnderlyingDriver: &fakeDriver{},
	})
	if err != nil {
		t.Fatalf("new connector: %v", err)
	}

	conn, err := connector.Connect(context.Background())
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
}

func TestConnector_MissingCredential(t *testing.T) {
	store := NewMemStore()

	connector, err := NewConnector(ConnectorConfig{
		Store:            store,
		CredentialKey:    "nonexistent",
		DSNTemplate:      "postgres://user:${CREDENTIAL}@host/db",
		UnderlyingDriver: &fakeDriver{},
	})
	if err != nil {
		t.Fatalf("new connector: %v", err)
	}

	_, err = connector.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestConnector_Driver(t *testing.T) {
	drv := &fakeDriver{}
	connector, err := NewConnector(ConnectorConfig{
		Store:            NewMemStore(),
		CredentialKey:    "key",
		DSNTemplate:      "dsn:${CREDENTIAL}",
		UnderlyingDriver: drv,
	})
	if err != nil {
		t.Fatalf("new connector: %v", err)
	}
	if connector.Driver() != drv {
		t.Fatal("expected same driver reference")
	}
}

func TestConnector_ValidationErrors(t *testing.T) {
	drv := &fakeDriver{}
	store := NewMemStore()

	cases := []struct {
		name string
		cfg  ConnectorConfig
	}{
		{"nil store", ConnectorConfig{CredentialKey: "k", DSNTemplate: "${CREDENTIAL}", UnderlyingDriver: drv}},
		{"empty key", ConnectorConfig{Store: store, DSNTemplate: "${CREDENTIAL}", UnderlyingDriver: drv}},
		{"empty dsn", ConnectorConfig{Store: store, CredentialKey: "k", UnderlyingDriver: drv}},
		{"no placeholder", ConnectorConfig{Store: store, CredentialKey: "k", DSNTemplate: "static-dsn", UnderlyingDriver: drv}},
		{"nil driver", ConnectorConfig{Store: store, CredentialKey: "k", DSNTemplate: "${CREDENTIAL}"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewConnector(tc.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
