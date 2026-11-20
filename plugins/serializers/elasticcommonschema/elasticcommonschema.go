package elasticcommonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/serializers"
)

const defaultECSVersion = "8.11.0"

type Serializer struct {
	// ECSVersion is the version of the Elastic Common Schema the generated
	// documents conform to. It is written to the `ecs.version` field of
	// every document.
	ECSVersion       string          `toml:"ecs_schema_version"`
	FloatHandling    string          `toml:"ecs_float_handling"`
	FloatReplacement float64         `toml:"ecs_float_replacement_value"`
	Log              telegraf.Logger `toml:"-"`
}

func (s *Serializer) Init() error {
	// Determine the ECS version to use
	if s.ECSVersion == "" {
		s.ECSVersion = defaultECSVersion
	} else if _, err := semver.NewVersion(s.ECSVersion); err != nil {
		return fmt.Errorf("invalid 'ecs_schema_version' %q: %w", s.ECSVersion, err)
	}

	// Determine how to handle NaN and Inf values, as those cannot be
	// represented in JSON and would make the documents invalid.
	switch s.FloatHandling {
	case "":
		s.FloatHandling = "none"
	case "none", "drop", "replace":
		// Do nothing, those are valid
	default:
		return fmt.Errorf("invalid 'ecs_float_handling' %q", s.FloatHandling)
	}

	return nil
}

// Serialize writes the given metric as a single ECS-compliant JSON document
// terminated by a newline.
func (s *Serializer) Serialize(metric telegraf.Metric) ([]byte, error) {
	return s.appendDocument(nil, metric)
}

// SerializeBatch writes the given metrics as a stream of newline-delimited
// JSON documents (NDJSON), one ECS-compliant document per metric. This
// matches the document part of the payload the Elasticsearch `_bulk`
// endpoint expects, so an output supporting that protocol can add the
// necessary action lines around the documents.
func (s *Serializer) SerializeBatch(metrics []telegraf.Metric) ([]byte, error) {
	var buf []byte
	for _, metric := range metrics {
		var err error
		buf, err = s.appendDocument(buf, metric)
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func (s *Serializer) appendDocument(buf []byte, metric telegraf.Metric) ([]byte, error) {
	doc := s.createObject(metric)

	encoded, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("creating document for %q failed: %w", metric.Name(), err)
	}
	buf = append(buf, encoded...)
	buf = append(buf, '\n')
	return buf, nil
}

// createObject converts the given metric into an ECS-compliant document:
//   - the metric time is written to `@timestamp`
//   - the configured schema version is written to `ecs.version`
//   - the measurement name is written to `event.dataset`
//   - the tags are written as top-level fields, with dotted keys expanded
//     into nested objects (e.g. the tag `orchestrator.cluster.name` is
//     written as {"orchestrator":{"cluster":{"name":...}}})
//   - the fields are nested under the measurement name
func (s *Serializer) createObject(metric telegraf.Metric) map[string]any {
	doc := map[string]any{
		"@timestamp": metric.Time(),
		"ecs":        map[string]any{"version": s.ECSVersion},
		"event":      map[string]any{"dataset": metric.Name()},
	}

	// Add the fields nested under the measurement name
	fields := make(map[string]any, len(metric.FieldList()))
	for _, field := range metric.FieldList() {
		if v, ok := s.fieldValue(field.Value); ok {
			fields[field.Key] = v
		}
	}
	doc[metric.Name()] = fields

	// Add the tags as top-level fields, expanding dotted keys into nested
	// objects. Tags conflicting with a field of the document, e.g. a tag
	// `up` on a measurement `up` or a tag `host` shadowing the object built
	// for a tag `host.name`, are skipped.
	for _, tag := range metric.TagList() {
		if err := setNestedValue(doc, strings.Split(tag.Key, "."), tag.Value); err != nil {
			s.warnf("skipping tag %q of metric %q: %v", tag.Key, metric.Name(), err)
		}
	}

	return doc
}

// fieldValue handles the special float values NaN and Inf, which cannot be
// represented in JSON. Depending on the configured handling, the value is
// returned unchanged, dropped or replaced.
func (s *Serializer) fieldValue(value any) (any, bool) {
	v, ok := value.(float64)
	if !ok || s.FloatHandling == "none" || !(math.IsNaN(v) || math.IsInf(v, 0)) {
		return value, true
	}

	switch s.FloatHandling {
	case "drop":
		return nil, false
	case "replace":
		if math.IsNaN(v) || math.IsInf(v, 1) {
			return s.FloatReplacement, true
		}
		return -s.FloatReplacement, true
	}
	return value, true
}

// setNestedValue stores the given value in the document under the given key
// path, creating the intermediate objects on the way. It returns an error if
// the key path cannot be used, e.g. because a segment is empty or because a
// segment is already occupied by a non-object value. The document is left
// unchanged in that case.
func setNestedValue(doc map[string]any, keys []string, value any) error {
	// Check the key path upfront to keep the document unchanged on error
	for _, key := range keys {
		if key == "" {
			return errors.New("empty key segment")
		}
	}

	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		if _, found := doc[key]; !found {
			doc[key] = map[string]any{}
		}
		m, ok := doc[key].(map[string]any)
		if !ok {
			return fmt.Errorf("key %q is already used by a non-object field", strings.Join(keys[:i+1], "."))
		}
		doc = m
	}

	leaf := keys[len(keys)-1]
	if _, found := doc[leaf]; found {
		return fmt.Errorf("key %q is already used", strings.Join(keys, "."))
	}
	doc[leaf] = value
	return nil
}

// warnf logs the given warning. The logger is not available if the serializer
// is used outside of an agent, e.g. in unit-tests, in which case the
// warning is dropped.
func (s *Serializer) warnf(format string, a ...any) {
	if s.Log != nil {
		s.Log.Warnf(format, a...)
	}
}

func init() {
	serializers.Add("elasticcommonschema",
		func() telegraf.Serializer {
			return &Serializer{}
		},
	)
}
