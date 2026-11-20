package elasticcommonschema

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/toml"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/parsers/influx"
	"github.com/influxdata/telegraf/testutil"
	"github.com/influxdata/telegraf/testutil/serializers"
)

func TestSerialize(t *testing.T) {
	m := metric.New(
		"up",
		map[string]string{
			"host":                      "Hugin",
			"orchestrator.cluster.name": "cluster",
			"service.name":              "telegraf",
		},
		map[string]any{
			"state": "up",
			"gauge": 1,
		},
		time.Unix(1682613051, 0),
	)

	s := &Serializer{}
	require.NoError(t, s.Init())
	buf, err := s.Serialize(m)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"@timestamp": "2023-04-27T16:30:51Z",
		"ecs": {"version": "8.11.0"},
		"event": {"dataset": "up"},
		"host": "Hugin",
		"orchestrator": {"cluster": {"name": "cluster"}},
		"service": {"name": "telegraf"},
		"up": {"state": "up", "gauge": 1}
	}`, string(buf))
	require.True(t, strings.HasSuffix(string(buf), "\n"), "document must be newline terminated")
}

func TestSerializeSchemaVersion(t *testing.T) {
	m := metric.New(
		"up",
		map[string]string{},
		map[string]any{"gauge": 1},
		time.Unix(0, 0),
	)

	s := &Serializer{ECSVersion: "9.0.0"}
	require.NoError(t, s.Init())
	buf, err := s.Serialize(m)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"@timestamp": "1970-01-01T00:00:00Z",
		"ecs": {"version": "9.0.0"},
		"event": {"dataset": "up"},
		"up": {"gauge": 1}
	}`, string(buf))
}

func TestSerializeBatch(t *testing.T) {
	metrics := []telegraf.Metric{
		metric.New(
			"up",
			map[string]string{"host": "Hugin"},
			map[string]any{"state": "up"},
			time.Unix(1682613051, 0),
		),
		metric.New(
			"up",
			map[string]string{"host": "Munin"},
			map[string]any{"state": "down"},
			time.Unix(1682613052, 0),
		),
	}

	s := &Serializer{}
	require.NoError(t, s.Init())
	buf, err := s.SerializeBatch(metrics)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(buf), "\n"), "\n")
	require.Len(t, lines, 2)
	require.JSONEq(t, `{
		"@timestamp": "2023-04-27T16:30:51Z",
		"ecs": {"version": "8.11.0"},
		"event": {"dataset": "up"},
		"host": "Hugin",
		"up": {"state": "up"}
	}`, lines[0])
	require.JSONEq(t, `{
		"@timestamp": "2023-04-27T16:30:52Z",
		"ecs": {"version": "8.11.0"},
		"event": {"dataset": "up"},
		"host": "Munin",
		"up": {"state": "down"}
	}`, lines[1])
}

func TestSerializeBatchEmpty(t *testing.T) {
	s := &Serializer{}
	require.NoError(t, s.Init())
	buf, err := s.SerializeBatch(nil)
	require.NoError(t, err)
	require.Empty(t, buf)
}

func TestSerializeNoFields(t *testing.T) {
	// A metric without fields emits an empty object named after the
	// measurement
	m := metric.New(
		"up",
		map[string]string{"host": "Hugin"},
		map[string]any{},
		time.Unix(0, 0),
	)

	s := &Serializer{}
	require.NoError(t, s.Init())
	buf, err := s.Serialize(m)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"@timestamp": "1970-01-01T00:00:00Z",
		"ecs": {"version": "8.11.0"},
		"event": {"dataset": "up"},
		"host": "Hugin",
		"up": {}
	}`, string(buf))
}

func TestInit(t *testing.T) {
	tests := []struct {
		name     string
		plugin   *Serializer
		expected string
		err      string
	}{
		{
			name:     "defaults",
			plugin:   &Serializer{},
			expected: "8.11.0",
		},
		{
			name:     "schema version",
			plugin:   &Serializer{ECSVersion: "9.0.0"},
			expected: "9.0.0",
		},
		{
			name:   "invalid schema version",
			plugin: &Serializer{ECSVersion: "banana"},
			err:    `invalid 'ecs_schema_version' "banana"`,
		},
		{
			name:     "float handling drop",
			plugin:   &Serializer{FloatHandling: "drop"},
			expected: "8.11.0",
		},
		{
			name:     "float handling replace",
			plugin:   &Serializer{FloatHandling: "replace"},
			expected: "8.11.0",
		},
		{
			name:   "invalid float handling",
			plugin: &Serializer{FloatHandling: "banana"},
			err:    `invalid 'ecs_float_handling' "banana"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Init()
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, tt.plugin.ECSVersion)
		})
	}
}

func TestFloatHandling(t *testing.T) {
	tests := []struct {
		name        string
		plugin      *Serializer
		measurement string
		expectError bool
	}{
		{
			name:        "none produces invalid JSON",
			plugin:      &Serializer{FloatHandling: "none"},
			expectError: true,
		},
		{
			name:        "drop",
			plugin:      &Serializer{FloatHandling: "drop"},
			measurement: `{"gauge": 1}`,
		},
		{
			name:        "replace",
			plugin:      &Serializer{FloatHandling: "replace"},
			measurement: `{"gauge": 1, "nan": 0, "inf": 0, "ninf": -0}`,
		},
		{
			name:        "replace with custom value",
			plugin:      &Serializer{FloatHandling: "replace", FloatReplacement: -1},
			measurement: `{"gauge": 1, "nan": -1, "inf": -1, "ninf": 1}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metric.New(
				"up",
				map[string]string{},
				map[string]any{
					"gauge": 1,
					"nan":   math.NaN(),
					"inf":   math.Inf(1),
					"ninf":  math.Inf(-1),
				},
				time.Unix(0, 0),
			)

			require.NoError(t, tt.plugin.Init())
			buf, err := tt.plugin.Serialize(m)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Compare only the measurement object, the fixed fields of the
			// document are covered by the other tests
			var doc map[string]any
			require.NoError(t, json.Unmarshal(buf, &doc))
			require.JSONEq(t, tt.measurement, string(mustJSON(t, doc["up"])))
		})
	}
}

func TestTagConflicts(t *testing.T) {
	tests := []struct {
		name     string
		tags     []struct{ key, value string }
		expected string
		full     bool
	}{
		{
			// The insertion order is irrelevant as the tags are processed in
			// the sorted order of the metric, so the scalar 'host' wins over
			// the nested 'host.name' in both directions
			name: "scalar and object",
			tags: []struct{ key, value string }{
				{"host", "Hugin"},
				{"host.name", "hugin.example.org"},
			},
			// The tag 'host.name' is skipped as the scalar for 'host' already
			// exists
			expected: `{"host": "Hugin"}`,
		},
		{
			name: "object and scalar",
			tags: []struct{ key, value string }{
				{"host.name", "hugin.example.org"},
				{"host", "Hugin"},
			},
			// The insertion order is irrelevant, the scalar 'host' is
			// processed first due to the sorted tag order and therefore the
			// nested tag is skipped
			expected: `{"host": "Hugin"}`,
		},
		{
			name: "empty key segment",
			tags: []struct{ key, value string }{
				{"host..name", "hugin.example.org"},
			},
			// The tag is skipped as the key contains an empty segment
			expected: `{}`,
		},
		{
			name: "conflict with ECS field",
			tags: []struct{ key, value string }{
				{"ecs.version", "banana"},
				{"event.dataset", "banana"},
				{"@timestamp", "banana"},
			},
			// Those tags are skipped as the fields are owned by the serializer
			// and keep the values set by it
			expected: `{"@timestamp": "1970-01-01T00:00:00Z", "ecs": {"version": "8.11.0"}, "event": {"dataset": "up"}, "up": {"gauge": 1}}`,
			full:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metric.New(
				"up",
				nil,
				map[string]any{"gauge": 1},
				time.Unix(0, 0),
			)
			for _, tag := range tt.tags {
				m.AddTag(tag.key, tag.value)
			}

			s := &Serializer{}
			require.NoError(t, s.Init())
			buf, err := s.Serialize(m)
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(buf, &doc))
			if !tt.full {
				// Only compare the tag part of the document, the fixed ECS
				// fields and the measurement fields are covered by the other
				// tests
				for _, key := range []string{"@timestamp", "ecs", "event", "up"} {
					delete(doc, key)
				}
			}
			require.JSONEq(t, tt.expected, string(mustJSON(t, doc)))
		})
	}
}

func TestCases(t *testing.T) {
	// Get all sub-directories
	folders, err := os.ReadDir("testcases")
	require.NoError(t, err)

	parser := &influx.Parser{}
	require.NoError(t, parser.Init())

	for _, f := range folders {
		// Only handle directories
		if !f.IsDir() {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			// Configure the serializer from the testcase
			cfg, err := os.ReadFile(filepath.Join("testcases", f.Name(), "telegraf.conf"))
			require.NoError(t, err)
			s := &Serializer{}
			require.NoError(t, toml.Unmarshal(cfg, s))
			require.NoError(t, s.Init())

			// Read the input metrics and the expected documents
			input, err := testutil.ParseMetricsFromFile(filepath.Join("testcases", f.Name(), "input.influx"), parser)
			require.NoError(t, err)
			expected, err := os.ReadFile(filepath.Join("testcases", f.Name(), "expected.ndjson"))
			require.NoError(t, err)

			// Batch mode must produce the expected NDJSON stream
			actual, err := s.SerializeBatch(input)
			require.NoError(t, err)
			require.Equal(t, string(expected), string(actual))

			// Non-batch mode must produce the same documents
			var single []byte
			for _, m := range input {
				buf, err := s.Serialize(m)
				require.NoError(t, err)
				single = append(single, buf...)
			}
			require.Equal(t, string(expected), string(single))
		})
	}
}

func BenchmarkSerialize(b *testing.B) {
	s := &Serializer{}
	require.NoError(b, s.Init())
	metrics := serializers.BenchmarkMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Serialize(metrics[i%len(metrics)])
		require.NoError(b, err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	buf, err := json.Marshal(v)
	require.NoError(t, err)
	return buf
}
