package groundwork

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/logger"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/testutil"
)

const (
	defaultTestAgentID = "ec1676cc-583d-48ee-b035-7fb5ed0fcf88"
	defaultHost        = "telegraf"
	defaultAppType     = "TELEGRAF"
	customAppType      = "SYSLOG"
)

func TestWriteWithDebug(t *testing.T) {
	// Generate test metric with default name to test Write logic
	intMetric := testutil.TestMetric(42, "IntMetric")
	srvTok := "88fcf0de5bf7-530b-ee84-d385-cc6761ce"

	// Simulate Groundwork server that should receive custom metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Decode body to use in assertions below
		var obj transit.ResourcesWithServicesRequest
		if err = json.Unmarshal(body, &obj); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Check if server gets proper data
		if obj.Resources[0].Services[0].Name != "IntMetric" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "IntMetric", obj.Resources[0].Services[0].Name)
			return
		}
		if *obj.Resources[0].Services[0].Metrics[0].Value.IntegerValue != int64(42) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %v, actual: %v", int64(42), *obj.Resources[0].Services[0].Metrics[0].Value.IntegerValue)
			return
		}

		// Send back details
		ans := "Content-type: application/json\n\n" + `{"message":"` + srvTok + `"}`
		if _, err = fmt.Fprintln(w, ans); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}
	}))

	i := Groundwork{
		Server:              server.URL,
		AgentID:             defaultTestAgentID,
		Username:            config.NewSecret([]byte(`tu ser`)),
		Password:            config.NewSecret([]byte(`pu ser`)),
		DefaultAppType:      defaultAppType,
		DefaultHost:         defaultHost,
		DefaultServiceState: string(transit.ServiceOk),
		ResourceTag:         "host",
		Log:                 testutil.Logger{},
	}

	buf := new(bytes.Buffer)
	require.NoError(t, logger.SetupLogging(&logger.Config{Debug: true}))
	logger.RedirectLogging(buf)

	require.NoError(t, i.Init())
	require.NoError(t, i.Write([]telegraf.Metric{intMetric}))

	require.NoError(t, logger.CloseLogging())
	require.Contains(t, buf.String(), defaultTestAgentID)
	require.Contains(t, buf.String(), srvTok)

	server.Close()
}

func TestWriteWithDefaults(t *testing.T) {
	// Generate test metric with default name to test Write logic
	intMetric := testutil.TestMetric(42, "IntMetric")

	// Simulate Groundwork server that should receive custom metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Decode body to use in assertions below
		var obj transit.ResourcesWithServicesRequest
		if err = json.Unmarshal(body, &obj); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Check if server gets proper data
		if obj.Context.AgentID != defaultTestAgentID {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", defaultTestAgentID, obj.Context.AgentID)
			return
		}
		if obj.Context.AppType != customAppType {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", customAppType, obj.Context.AppType)
			return
		}
		if obj.Resources[0].Name != defaultHost {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", defaultHost, obj.Resources[0].Name)
			return
		}
		if obj.Resources[0].Services[0].Status != transit.MonitorStatus("SERVICE_OK") {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", transit.MonitorStatus("SERVICE_OK"), obj.Resources[0].Services[0].Status)
			return
		}
		if obj.Resources[0].Services[0].Name != "IntMetric" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "IntMetric", obj.Resources[0].Services[0].Name)
			return
		}
		if *obj.Resources[0].Services[0].Metrics[0].Value.IntegerValue != int64(42) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %v, actual: %v", int64(42), *obj.Resources[0].Services[0].Metrics[0].Value.IntegerValue)
			return
		}
		if len(obj.Groups) != 0 {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("'obj.Groups' should not be empty")
			return
		}

		if _, err = fmt.Fprintln(w, "OK"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}
	}))

	i := Groundwork{
		Log:                 testutil.Logger{},
		Server:              server.URL,
		AgentID:             defaultTestAgentID,
		DefaultHost:         defaultHost,
		DefaultAppType:      customAppType,
		DefaultServiceState: string(transit.ServiceOk),
		client: clients.GWClient{
			AppName: "telegraf",
			AppType: customAppType,
			GWConnection: &clients.GWConnection{
				HostName: server.URL,
			},
		},
	}

	err := i.Write([]telegraf.Metric{intMetric})
	require.NoError(t, err)

	server.Close()
}

func TestWriteWithFields(t *testing.T) {
	// Generate test metric with fields to test Write logic
	floatMetric := testutil.TestMetric(1.0, "FloatMetric")
	floatMetric.AddField("value_cr", 3.0)
	floatMetric.AddField("value_wn", 2.0)
	floatMetric.AddField("message", "Test Message")
	floatMetric.AddField("status", "SERVICE_WARNING")

	// Simulate Groundwork server that should receive custom metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Decode body to use in assertions below
		var obj transit.ResourcesWithServicesRequest
		if err = json.Unmarshal(body, &obj); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Check if server gets proper data
		if obj.Resources[0].Services[0].LastPluginOutput != "Test Message" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Test Message", obj.Resources[0].Services[0].LastPluginOutput)
			return
		}
		if obj.Resources[0].Services[0].Status != transit.MonitorStatus("SERVICE_WARNING") {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", transit.MonitorStatus("SERVICE_WARNING"), obj.Resources[0].Services[0].Status)
			return
		}
		if dt := float64(1.0) - *obj.Resources[0].Services[0].Metrics[0].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(1.0), *obj.Resources[0].Services[0].Metrics[0].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}
		if dt := float64(3.0) - *obj.Resources[0].Services[0].Metrics[0].Thresholds[0].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(3.0), *obj.Resources[0].Services[0].Metrics[0].Thresholds[0].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}
		if dt := float64(2.0) - *obj.Resources[0].Services[0].Metrics[0].Thresholds[1].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(2.0), *obj.Resources[0].Services[0].Metrics[0].Thresholds[1].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}

		if _, err = fmt.Fprintln(w, "OK"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}
	}))

	i := Groundwork{
		Log:            testutil.Logger{},
		Server:         server.URL,
		AgentID:        defaultTestAgentID,
		DefaultHost:    defaultHost,
		DefaultAppType: defaultAppType,
		GroupTag:       "group",
		ResourceTag:    "host",
		client: clients.GWClient{
			AppName: "telegraf",
			AppType: defaultAppType,
			GWConnection: &clients.GWConnection{
				HostName: server.URL,
			},
		},
	}

	err := i.Write([]telegraf.Metric{floatMetric})
	require.NoError(t, err)

	server.Close()
}

func TestWriteWithTags(t *testing.T) {
	// Generate test metric with tags to test Write logic
	floatMetric := testutil.TestMetric(1.0, "FloatMetric")
	floatMetric.AddField("value_cr", 3.0)
	floatMetric.AddField("value_wn", 2.0)
	floatMetric.AddField("message", "Test Message")
	floatMetric.AddField("status", "SERVICE_WARNING")
	floatMetric.AddTag("value_cr", "9.0")
	floatMetric.AddTag("value_wn", "6.0")
	floatMetric.AddTag("message", "Test Tag")
	floatMetric.AddTag("status", "SERVICE_PENDING")
	floatMetric.AddTag("group-tag", "Group01")
	floatMetric.AddTag("resource-tag", "Host01")
	floatMetric.AddTag("service", "Service01")
	floatMetric.AddTag("facility", "FACILITY")
	floatMetric.AddTag("severity", "SEVERITY")

	// Simulate Groundwork server that should receive custom metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Decode body to use in assertions below
		var obj transit.ResourcesWithServicesRequest
		if err = json.Unmarshal(body, &obj); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		// Check if server gets proper data
		if obj.Context.AgentID != defaultTestAgentID {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", defaultTestAgentID, obj.Context.AgentID)
			return
		}
		if obj.Context.AppType != defaultAppType {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", defaultAppType, obj.Context.AppType)
			return
		}
		if obj.Resources[0].Name != "Host01" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Host01", obj.Resources[0].Name)
			return
		}
		if obj.Resources[0].Services[0].Name != "Service01" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Service01", obj.Resources[0].Services[0].Name)
			return
		}
		if *obj.Resources[0].Services[0].Properties["facility"].StringValue != "FACILITY" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "FACILITY", *obj.Resources[0].Services[0].Properties["facility"].StringValue)
			return
		}
		if *obj.Resources[0].Services[0].Properties["severity"].StringValue != "SEVERITY" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "SEVERITY", *obj.Resources[0].Services[0].Properties["severity"].StringValue)
			return
		}
		if obj.Groups[0].GroupName != "Group01" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Group01", obj.Groups[0].GroupName)
			return
		}
		if obj.Groups[0].Resources[0].Name != "Host01" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Host01", obj.Groups[0].Resources[0].Name)
			return
		}
		if obj.Resources[0].Services[0].LastPluginOutput != "Test Tag" {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", "Test Tag", obj.Resources[0].Services[0].LastPluginOutput)
			return
		}
		if obj.Resources[0].Services[0].Status != transit.MonitorStatus("SERVICE_PENDING") {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Not equal, expected: %q, actual: %q", transit.MonitorStatus("SERVICE_PENDING"), obj.Resources[0].Services[0].Status)
			return
		}
		if dt := float64(1.0) - *obj.Resources[0].Services[0].Metrics[0].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(1.0), *obj.Resources[0].Services[0].Metrics[0].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}
		if dt := float64(9.0) - *obj.Resources[0].Services[0].Metrics[0].Thresholds[0].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(9.0), *obj.Resources[0].Services[0].Metrics[0].Thresholds[0].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}
		if dt := float64(6.0) - *obj.Resources[0].Services[0].Metrics[0].Thresholds[1].Value.DoubleValue; !testutil.WithinDefaultDelta(dt) {
			w.WriteHeader(http.StatusInternalServerError)
			t.Errorf("Max difference between %v and %v allowed is %v, but difference was %v",
				float64(6.0), *obj.Resources[0].Services[0].Metrics[0].Thresholds[1].Value.DoubleValue, testutil.DefaultDelta, dt)
			return
		}

		if _, err = fmt.Fprintln(w, "OK"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}
	}))

	i := Groundwork{
		Log:            testutil.Logger{},
		Server:         server.URL,
		AgentID:        defaultTestAgentID,
		DefaultHost:    defaultHost,
		DefaultAppType: defaultAppType,
		GroupTag:       "group-tag",
		ResourceTag:    "resource-tag",
		client: clients.GWClient{
			AppName: "telegraf",
			AppType: defaultAppType,
			GWConnection: &clients.GWConnection{
				HostName: server.URL,
			},
		},
	}

	err := i.Write([]telegraf.Metric{floatMetric})
	require.NoError(t, err)

	server.Close()
}

func TestDefaultServiceState(t *testing.T) {
	i := Groundwork{
		Log:                 testutil.Logger{},
		DefaultHost:         defaultHost,
		DefaultServiceState: string(transit.ServicePending),
		ResourceTag:         "host",
	}
	now := time.Now()
	tests := []struct {
		name     string
		tags     map[string]string
		fields   map[string]any
		expected transit.MonitorStatus
	}{
		{
			name:     "no values",
			fields:   map[string]any{"message": "text only"},
			expected: transit.ServicePending,
		},
		{
			name:     "no thresholds",
			fields:   map[string]any{"value": 95.0},
			expected: transit.ServicePending,
		},
		{
			name:     "invalid status",
			tags:     map[string]string{"status": "bogus"},
			fields:   map[string]any{"value": 95.0},
			expected: transit.ServicePending,
		},
		{
			name:     "thresholds",
			fields:   map[string]any{"value": 95.0, "value_wn": 80.0, "value_cr": 90.0},
			expected: transit.ServiceUnscheduledCritical,
		},
		{
			name:     "threshold tags",
			tags:     map[string]string{"warning": "80", "critical": "90"},
			fields:   map[string]any{"value": 50.0},
			expected: transit.ServiceOk,
		},
		{
			name:     "status tag",
			tags:     map[string]string{"status": "SERVICE_WARNING"},
			fields:   map[string]any{"value": 95.0},
			expected: transit.ServiceWarning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, service := i.parseMetric(metric.New("test", tt.tags, tt.fields, now))
			require.Equal(t, tt.expected, service.Status)
		})
	}
}

// testGroundworkServer returns a test server that decodes the incoming
// request into obj and otherwise acts as a working GroundWork API.
func testGroundworkServer(t *testing.T, status int, obj *transit.ResourcesWithServicesRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		var request transit.ResourcesWithServicesRequest
		if err = json.Unmarshal(body, &request); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			t.Error(err)
			return
		}

		*obj = request
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		if _, err = fmt.Fprintln(w, "OK"); err != nil {
			t.Error(err)
		}
	}))
}

func testGroundworkClient(server *httptest.Server) Groundwork {
	return Groundwork{
		Log:                 testutil.Logger{},
		Server:              server.URL,
		AgentID:             defaultTestAgentID,
		DefaultHost:         defaultHost,
		DefaultAppType:      defaultAppType,
		DefaultServiceState: string(transit.ServiceOk),
		GroupTag:            "group",
		ResourceTag:         "host",
		client: clients.GWClient{
			AppName: "telegraf",
			AppType: defaultAppType,
			GWConnection: &clients.GWConnection{
				HostName: server.URL,
			},
		},
	}
}

func TestWriteWithRules(t *testing.T) {
	tests := []struct {
		name   string
		rules  []rule
		metric telegraf.Metric
		check  func(t *testing.T, obj transit.ResourcesWithServicesRequest)
	}{
		{
			name: "resource from node_name with regex named group",
			rules: []rule{
				{Target: "resource", Tag: "node_name", Pattern: `^(?P<short>[^.]+)`, Template: `{{ .Groups.short }}`},
			},
			metric: metric.New("test", map[string]string{"node_name": "worker-1.example.com"}, map[string]any{"value": int64(42)}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, "worker-1", obj.Resources[0].Name)
			},
		},
		{
			name: "resource falls back to second rule when first does not match",
			rules: []rule{
				{Target: "resource", Tag: "node_name", Pattern: `^(?P<short>[^.]+)\.example\.com$`, Template: `{{ .Groups.short }}`},
				{Target: "resource", Tag: "host", Pattern: `^(?P<short>[^.]+)`, Template: `{{ .Groups.short }}`},
			},
			metric: metric.New("test", map[string]string{"host": "fallback-host.internal"}, map[string]any{"value": int64(42)}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, "fallback-host", obj.Resources[0].Name)
			},
		},
		{
			name: "resource falls back to default_host when no rule matches",
			rules: []rule{
				{Target: "resource", Tag: "node_name", Pattern: `^nomatch`},
			},
			metric: metric.New("test", nil, map[string]any{"value": int64(42)}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, defaultHost, obj.Resources[0].Name)
			},
		},
		{
			name: "group built from two tags with a template",
			rules: []rule{
				{Target: "group", Template: `{{ .Tag "cluster" }}-{{ .Tag "namespace" }}`},
			},
			metric: metric.New("test", map[string]string{"cluster": "prod", "namespace": "infra"}, map[string]any{"value": int64(42)}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Groups, 1)
				require.Equal(t, "prod-infra", obj.Groups[0].GroupName)
			},
		},
		{
			name: "service built from a field with a regex",
			rules: []rule{
				{Target: "service", Field: "unit", Pattern: `^(unit-)?(?P<name>.+)$`, Template: `{{ .Groups.name }}`},
			},
			metric: metric.New("test", nil, map[string]any{"value": int64(42), "unit": "unit-nginx"}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, "nginx", obj.Resources[0].Services[0].Name)
			},
		},
		{
			name: "status built from the message field",
			rules: []rule{
				{Target: "status", Field: "message", Pattern: `(?i)fatal|critical`, Template: "SERVICE_UNSCHEDULED_CRITICAL"},
			},
			metric: metric.New("test", nil, map[string]any{"value": int64(42), "message": "this is a FATAL error"}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, transit.ServiceUnscheduledCritical, obj.Resources[0].Services[0].Status)
			},
		},
		{
			name: "invalid status from a rule falls back to default",
			rules: []rule{
				{Target: "status", Field: "message", Template: "NOT_A_STATUS"},
			},
			metric: metric.New("test", nil, map[string]any{"value": int64(42), "message": "whatever"}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, transit.ServiceOk, obj.Resources[0].Services[0].Status)
			},
		},
		{
			name: "message built from a tag",
			rules: []rule{
				{Target: "message", Tag: "note"},
			},
			metric: metric.New("test", map[string]string{"note": "hello from tag"}, map[string]any{"value": int64(42)}, time.Now()),
			check: func(t *testing.T, obj transit.ResourcesWithServicesRequest) {
				require.Len(t, obj.Resources, 1)
				require.Equal(t, "hello from tag", obj.Resources[0].Services[0].LastPluginOutput)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obj transit.ResourcesWithServicesRequest
			server := testGroundworkServer(t, http.StatusOK, &obj)
			defer server.Close()

			i := testGroundworkClient(server)
			require.NoError(t, validateRules(tt.rules))
			i.Rules = tt.rules

			require.NoError(t, i.Write([]telegraf.Metric{tt.metric}))
			tt.check(t, obj)
		})
	}
}

func TestWriteWithRulesSkipsMetrics(t *testing.T) {
	var obj transit.ResourcesWithServicesRequest
	server := testGroundworkServer(t, http.StatusOK, &obj)
	defer server.Close()

	i := testGroundworkClient(server)
	i.Rules = []rule{
		{Target: "skip", Tag: "kubernetes_namespace", Pattern: "^kube-"},
	}
	require.NoError(t, validateRules(i.Rules))

	skipped := testutil.TestMetric(1, "skipped")
	skipped.AddTag("kubernetes_namespace", "kube-system")
	kept := testutil.TestMetric(2, "kept")
	kept.AddTag("kubernetes_namespace", "default")

	require.NoError(t, i.Write([]telegraf.Metric{skipped, kept}))

	// Only the kept metric is sent.
	require.Len(t, obj.Resources, 1)
	require.Len(t, obj.Resources[0].Services, 1)
	require.Equal(t, "kept", obj.Resources[0].Services[0].Name)
}

func TestWriteWithAllMetricsSkipped(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	i := testGroundworkClient(server)
	i.Rules = []rule{
		{Target: "skip", Tag: "secret", Pattern: ".+"},
	}
	require.NoError(t, validateRules(i.Rules))
	metric := testutil.TestMetric(1, "test")
	metric.AddTag("secret", "yes")

	// All metrics are skipped, so the server must not be called and the
	// batch is considered written.
	require.NoError(t, i.Write([]telegraf.Metric{metric}))
	require.False(t, called)
}

func TestWriteWithSkippedMetricsAndFailedSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	i := testGroundworkClient(server)
	i.Rules = []rule{
		{Target: "skip", Tag: "secret", Pattern: ".+"},
	}
	require.NoError(t, validateRules(i.Rules))
	skipped := testutil.TestMetric(1, "skipped")
	skipped.AddTag("secret", "yes")
	kept := testutil.TestMetric(2, "kept")

	err := i.Write([]telegraf.Metric{skipped, kept})
	var partial *internal.PartialWriteError
	require.ErrorAs(t, err, &partial)
	// The skipped metric is considered handled, the kept metric stays in
	// the buffer to be retried.
	require.Equal(t, []int{0}, partial.MetricsAccept)
	require.Empty(t, partial.MetricsReject)
}

func TestWriteWithRuleSourceNotSentAsProperty(t *testing.T) {
	var obj transit.ResourcesWithServicesRequest
	server := testGroundworkServer(t, http.StatusOK, &obj)
	defer server.Close()

	i := testGroundworkClient(server)
	i.Rules = []rule{
		{Target: "resource", Tag: "node_name"},
		{Target: "status", Field: "severity", Template: "SERVICE_WARNING"},
	}
	require.NoError(t, validateRules(i.Rules))
	metric := testutil.TestMetric(1.0, "test")
	metric.AddTag("node_name", "worker-1")
	metric.AddTag("facility", "FACILITY")
	metric.AddField("severity", "crit")
	metric.AddField("load", 0.5)

	require.NoError(t, i.Write([]telegraf.Metric{metric}))

	svc := obj.Resources[0].Services[0]
	// The rule source tag must not be sent as property, the numeric field
	// used as rule source must still be sent as metric.
	_, ok := svc.Properties["node_name"]
	require.False(t, ok)
	_, ok = svc.Properties["facility"]
	require.True(t, ok)
	require.Len(t, svc.Metrics, 2)
	names := []string{svc.Metrics[0].MetricName, svc.Metrics[1].MetricName}
	require.ElementsMatch(t, []string{"value", "load"}, names)
}

func TestValidateRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   []rule
		wantErr string
	}{
		{
			name:  "valid rule with tag and pattern",
			rules: []rule{{Target: "resource", Tag: "host", Pattern: ".*"}},
		},
		{
			name:  "valid rule with template only",
			rules: []rule{{Target: "group", Template: `{{ .Tag "cluster" }}`}},
		},
		{
			name:  "valid rule with field only",
			rules: []rule{{Target: "message", Field: "message"}},
		},
		{
			name:    "invalid target",
			rules:   []rule{{Target: "host", Tag: "host"}},
			wantErr: "invalid target",
		},
		{
			name:    "tag and field are mutually exclusive",
			rules:   []rule{{Target: "resource", Tag: "a", Field: "b"}},
			wantErr: "only one of",
		},
		{
			name:    "pattern requires a source",
			rules:   []rule{{Target: "resource", Pattern: ".*"}},
			wantErr: "\"pattern\" requires",
		},
		{
			name:    "rule without source and template",
			rules:   []rule{{Target: "resource"}},
			wantErr: "at least one of",
		},
		{
			name:    "invalid pattern",
			rules:   []rule{{Target: "resource", Tag: "host", Pattern: "["}},
			wantErr: "invalid pattern",
		},
		{
			name:    "invalid template",
			rules:   []rule{{Target: "resource", Tag: "host", Template: "{{ .Nope"}},
			wantErr: "invalid template",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRules(tt.rules)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSampleConfig(t *testing.T) {
	// The sample config is embedded into the README and
	// must stay loadable by the plugin.
	conf := config.NewConfig()
	require.NoError(t, conf.LoadConfig("sample.conf"))
	require.Len(t, conf.Outputs, 1)
	output, ok := conf.Outputs[0].Output.(*Groundwork)
	require.True(t, ok)
	require.Empty(t, output.Rules) // all the rule examples are commented out

	// The example rules shown in the sample config must decode correctly
	// once uncommented
	example := `
[[outputs.groundwork]]
  url = "https://groundwork.example.com"
  agent_id = "` + defaultTestAgentID + `"
  username = "u"
  password = "p"

  [[outputs.groundwork.rules]]
    target = "resource"
    tag = "node_name"

  [[outputs.groundwork.rules]]
    target = "group"
    template = '{{ .Tag "cluster" }}-{{ .Tag "namespace" }}'

  [[outputs.groundwork.rules]]
    target = "status"
    field = "message"
    pattern = "(?i)fatal|critical"
    template = "SERVICE_UNSCHEDULED_CRITICAL"

  [[outputs.groundwork.rules]]
    target = "skip"
    tag = "secret"
`
	f, err := os.CreateTemp(t.TempDir(), "*.conf")
	require.NoError(t, err)
	_, err = f.WriteString(example)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	conf2 := config.NewConfig()
	require.NoError(t, conf2.LoadConfig(f.Name()))
	require.Len(t, conf2.Outputs, 1)
	output2, ok := conf2.Outputs[0].Output.(*Groundwork)
	require.True(t, ok)
	require.Len(t, output2.Rules, 4)
	require.NoError(t, validateRules(output2.Rules))
}
