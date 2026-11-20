# Elastic Common Schema Serializer

The `elasticcommonschema` output data format converts metrics into JSON
documents following the [Elastic Common Schema (ECS)][ECS].

The serializer only builds the documents. It does not talk to Elasticsearch
and it does not generate the `_bulk` framing or index routing, as those belong
to the output. Combine it with any output able to send the documents, e.g. to
an ingest gateway in front of Elasticsearch. An output implementing the
`_bulk` protocol can add the necessary action lines around the documents.

## Configuration

```toml
[[outputs.file]]
  ## Files to write to, "stdout" is a specially handled file.
  files = ["stdout"]

  ## Data format to output.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_OUTPUT.md
  data_format = "elasticcommonschema"

  ## Version of the Elastic Common Schema the documents conform to. The value
  ## is written to the `ecs.version` field of the generated documents.
  ## It should be set to the ECS version the target deployment uses.
  # ecs_schema_version = "8.11.0"

  ## Specifies the handling of NaN and Inf values, as those cannot be
  ## represented in JSON. This option can have the following values:
  ##    none    -- do not modify field-values (default); will produce an error if NaNs or infs are encountered
  ##    drop    -- drop fields containing NaNs or infs
  ##    replace -- replace with the value in "ecs_float_replacement_value" (default: 0.0)
  ##               NaNs and inf will be replaced with the given number, -inf with the negative of that number
  # ecs_float_handling = "none"
  # ecs_float_replacement_value = 0.0
```

## Metrics

For every metric the serializer generates a single ECS document with:

| Document field  | Content                                       |
| --------------- | --------------------------------------------- |
| `@timestamp`    | timestamp of the metric                       |
| `ecs.version`   | configured schema version                    |
| `event.dataset` | measurement name of the metric               |
| `<tag>`         | tag values, see below                        |
| `<measurement>` | object holding the metric fields              |

Tags are written as top-level fields. Dotted tag keys are expanded into
nested objects, so a tag `orchestrator.cluster.name` is written as
`{"orchestrator":{"cluster":{"name":...}}}`. This way tags already following
the ECS field naming, e.g. `host.name`, `service.name` or
`orchestrator.cluster.name`, end up at their proper ECS field.

Fields are nested under an object named after the measurement, so field names
cannot clash with the ECS fields above.

## Example

Here is an example conversion, using InfluxDB Line Protocol as the reference
format:

```influx
up,host=Hugin,orchestrator.cluster.name=cluster,service.name=telegraf state="up",gauge=1i 1682613051000000000
```

which the serializer converts to:

```json
{
  "@timestamp": "2023-04-27T16:30:51Z",
  "ecs": {"version": "8.11.0"},
  "event": {"dataset": "up"},
  "host": "Hugin",
  "orchestrator": {"cluster": {"name": "cluster"}},
  "service": {"name": "telegraf"},
  "up": {"state": "up", "gauge": 1}
}
```

In batch mode the serializer emits the documents as a stream of
newline-delimited JSON (NDJSON), one document per metric.

[ECS]: https://www.elastic.co/guide/en/ecs/current/index.html
