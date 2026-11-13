package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
)

type testSNMPConnection struct {
	host   string
	values map[string]any
}

func (tsc *testSNMPConnection) Host() string {
	return tsc.host
}

func (tsc *testSNMPConnection) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	sp := &gosnmp.SnmpPacket{}
	for _, oid := range oids {
		v, ok := tsc.values[oid]
		if !ok {
			sp.Variables = append(sp.Variables, gosnmp.SnmpPDU{
				Name: oid,
				Type: gosnmp.NoSuchObject,
			})
			continue
		}
		sp.Variables = append(sp.Variables, gosnmp.SnmpPDU{
			Name:  oid,
			Value: v,
		})
	}
	return sp, nil
}
func (tsc *testSNMPConnection) Walk(oid string, wf gosnmp.WalkFunc) error {
	for void, v := range tsc.values {
		if void == oid || (len(void) > len(oid) && void[:len(oid)+1] == oid+".") {
			if err := wf(gosnmp.SnmpPDU{
				Name:  void,
				Value: v,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
func (*testSNMPConnection) Reconnect() error {
	return nil
}

func (*testSNMPConnection) Close() error {
	return nil
}

func TestNewWrapperCommunity(t *testing.T) {
	tests := []struct {
		name     string
		create   func(*testing.T) config.Secret
		expected string
	}{
		{
			name:     "unset",
			expected: "public",
		},
		{
			name:     "static",
			create:   staticSecret("private"),
			expected: "private",
		},
		{
			name:     "secret reference",
			create:   linkedSecret("@{store:community}", "private"),
			expected: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ClientConfig{Version: 2}
			if tt.create != nil {
				cfg.Community = tt.create(t)
			}

			gs, err := NewWrapper(cfg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, gs.Community)
		})
	}
}

func TestNewWrapperSecName(t *testing.T) {
	tests := []struct {
		name     string
		create   func(*testing.T) config.Secret
		expected string
	}{
		{
			name: "unset",
		},
		{
			name:     "static",
			create:   staticSecret("myuser"),
			expected: "myuser",
		},
		{
			name:     "secret reference",
			create:   linkedSecret("@{store:sec_name}", "myuser"),
			expected: "myuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ClientConfig{Version: 3, SecLevel: "noAuthNoPriv"}
			if tt.create != nil {
				cfg.SecName = tt.create(t)
			}

			gs, err := NewWrapper(cfg)
			require.NoError(t, err)
			usm, ok := gs.SecurityParameters.(*gosnmp.UsmSecurityParameters)
			require.True(t, ok)
			require.Equal(t, tt.expected, usm.UserName)
		})
	}
}

func TestNewWrapperUnlinkedSecret(t *testing.T) {
	tests := []struct {
		name     string
		config   ClientConfig
		expected string
	}{
		{
			name: "community",
			config: ClientConfig{
				Version:   2,
				Community: unlinkedSecret(t, "@{store:community}"),
			},
			expected: "getting community failed",
		},
		{
			name: "sec_name",
			config: ClientConfig{
				Version: 3,
				SecName: unlinkedSecret(t, "@{store:sec_name}"),
			},
			expected: "getting security name failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWrapper(tt.config)
			require.ErrorContains(t, err, tt.expected)
		})
	}
}

func staticSecret(value string) func(*testing.T) config.Secret {
	return func(t *testing.T) config.Secret {
		t.Helper()

		s := config.NewSecret([]byte(value))
		t.Cleanup(s.Destroy)

		return s
	}
}

func linkedSecret(reference, value string) func(*testing.T) config.Secret {
	return func(t *testing.T) config.Secret {
		t.Helper()

		s := unlinkedSecret(t, reference)
		resolvers := map[string]telegraf.ResolveFunc{
			reference: func() ([]byte, bool, error) {
				return []byte(value), false, nil
			},
		}
		require.NoError(t, s.Link(resolvers))

		return s
	}
}

func unlinkedSecret(t *testing.T, reference string) config.Secret {
	t.Helper()

	var s config.Secret
	require.NoError(t, s.UnmarshalText([]byte(reference)))
	t.Cleanup(s.Destroy)

	return s
}

func TestReconnectResetsEngineID(t *testing.T) {
	gs, err := NewWrapper(ClientConfig{Version: 3, SecName: config.NewSecret([]byte("user")), EngineID: "stale", EngineBoots: 3, EngineTime: 42})
	require.NoError(t, err)
	require.NoError(t, gs.SetAgent("udp://127.0.0.1:161"))
	require.NoError(t, gs.Connect())
	defer gs.Close()

	// An open connection keeps the discovered engine ID
	require.NoError(t, gs.Reconnect())
	usm, ok := gs.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	require.True(t, ok)
	require.Equal(t, "stale", usm.AuthoritativeEngineID)

	// A closed connection discovers the engine ID again on reconnect
	require.NoError(t, gs.Close())
	require.NoError(t, gs.Reconnect())
	require.NotNil(t, gs.Conn)
	require.Empty(t, usm.AuthoritativeEngineID)
	require.Zero(t, usm.AuthoritativeEngineBoots)
	require.Zero(t, usm.AuthoritativeEngineTime)
}

var tsc = &testSNMPConnection{
	host: "tsc",
	values: map[string]any{
		".1.0.0.0.1.1.0":         "foo",
		".1.0.0.0.1.1.1":         []byte("bar"),
		".1.0.0.0.1.1.2":         []byte(""),
		".1.0.0.0.1.102":         "bad",
		".1.0.0.0.1.2.0":         1,
		".1.0.0.0.1.2.1":         2,
		".1.0.0.0.1.2.2":         0,
		".1.0.0.0.1.3.0":         "0.123",
		".1.0.0.0.1.3.1":         "0.456",
		".1.0.0.0.1.3.2":         "0.000",
		".1.0.0.0.1.3.3":         "9.999",
		".1.0.0.0.1.5.0":         123456,
		".1.0.0.0.1.6.0":         ".1.0.0.0.1.7",
		".1.0.0.0.2.1.0.0":       0,
		".1.0.0.0.2.1.0.1":       1,
		".1.0.0.0.2.2.0":         2,
		".1.0.0.1.1":             "baz",
		".1.0.0.1.2":             234,
		".1.0.0.1.3":             []byte("byte slice"),
		".1.0.0.2.1.5.0.9.9":     11,
		".1.0.0.2.1.5.1.9.9":     22,
		".1.0.0.3.1.1.10":        "instance",
		".1.0.0.3.1.1.11":        "instance2",
		".1.0.0.3.1.1.12":        "instance3",
		".1.0.0.3.1.2.10":        10,
		".1.0.0.3.1.2.11":        20,
		".1.0.0.3.1.2.12":        20,
		".1.0.0.3.1.3.10":        1,
		".1.0.0.3.1.3.11":        2,
		".1.0.0.3.1.3.12":        3,
		".1.3.6.1.2.1.3.1.1.1.0": "foo",
		".1.3.6.1.2.1.3.1.1.1.1": []byte("bar"),
		".1.3.6.1.2.1.3.1.1.1.2": []byte(""),
		".1.3.6.1.2.1.3.1.1.102": "bad",
		".1.3.6.1.2.1.3.1.1.2.0": 1,
		".1.3.6.1.2.1.3.1.1.2.1": 2,
		".1.3.6.1.2.1.3.1.1.2.2": 0,
		".1.3.6.1.2.1.3.1.1.3.0": "1.3.6.1.2.1.3.1.1.3",
		".1.3.6.1.2.1.3.1.1.5.0": 123456,
	},
}
