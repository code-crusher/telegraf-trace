# GroundWork Output Plugin

This plugin writes metrics to a [GroundWork Monitor][groundwork] instance.

> [!IMPORTANT]
> Plugin only supports GroundWork v8 or later.

⭐ Telegraf v1.21.0
🏷️ applications, messaging
💻 all

[groundwork]: https://www.gwos.com/product/groundwork-monitor/

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

Plugins support additional global and plugin configuration settings for tasks
such as modifying metrics, tags, and fields, creating aliases, and configuring
plugin ordering. See [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Secret store support

This plugin supports secrets from secret stores for the `username` and
`password` option.
See the [secret store documentation][SECRETSTORE] for more details on how
to use them.

[SECRETSTORE]: ../../../docs/CONFIGURATION.md#secret-store-secrets

## Configuration

```toml @sample.conf
# Send telegraf metrics to GroundWork Monitor
[[outputs.groundwork]]
  ## URL of your groundwork instance.
  url = "https://groundwork.example.com"

  ## Agent uuid for GroundWork API Server.
  agent_id = ""

  ## Username and password to access GroundWork API.
  username = ""
  password = ""

  ## Default application type to use in GroundWork client
  # default_app_type = "TELEGRAF"

  ## Default display name for the host with services(metrics).
  # default_host = "telegraf"

  ## Service state used when the status is neither given by a "status" tag or
  ## field, nor computed from thresholds.
  # default_service_state = "SERVICE_OK"

  ## The name of the tag that contains the hostname.
  # resource_tag = "host"

  ## The name of the tag that contains the host group name.
  # group_tag = "group"

  ## Rules to build the resource, group, service, status and message from
  ## the metric, instead of the fixed tags and fields above. This way the
  ## other outputs are not affected. Rules are evaluated in the order they
  ## are defined and the first matching rule wins. A rule matches when the
  ## configured "tag" or "field" exists, the "pattern" (if any) matches its
  ## value and the resulting value is not empty. When no rule matches, the
  ## default handling described above is used.
  ##
  ## A rule can set the "target" to one of "resource", "group", "service",
  ## "status", "message" or "skip". Metrics matching a "skip" rule are not
  ## sent to GroundWork and are not retried later.
  ##
  ## The "pattern" is a regular expression matched against the value of the
  ## configured "tag" or "field". With "template" unset, the matched text is
  ## used as the value. "template" is a Go template with the Sprig functions
  ## available, with the metric as "." (e.g. {{ .Tag "cluster" }},
  ## {{ .Field "load" }} or {{ .Name }}), and the following extra keys:
  ##   .Value  - the raw value of the configured tag or field
  ##   .Match  - the part of the value matched by the regular expression
  ##   .Groups - the named capture groups of the regular expression
  ##
  ## Example: build the host from the "node_name" tag, fall back to "host":
  # [[outputs.groundwork.rules]]
  #   target = "resource"
  #   tag = "node_name"
  ##
  ## Example: build the group from the "cluster" and "namespace" tags:
  # [[outputs.groundwork.rules]]
  #   target = "group"
  #   template = "{{ .Tag \"cluster\" }}-{{ .Tag \"namespace\" }}"
  ##
  ## Example: set the status from a log message field:
  # [[outputs.groundwork.rules]]
  #   target = "status"
  #   field = "message"
  #   pattern = "(?i)fatal|critical"
  #   template = "SERVICE_UNSCHEDULED_CRITICAL"
  ##
  ## Example: skip metrics with a "secret" tag for this output only:
  # [[outputs.groundwork.rules]]
  #   target = "skip"
  #   tag = "secret"
```

## Rules

The host, group, service, status and message can each be built from the
metric instead of using the fixed tags and fields described above. Rules are
defined with `[[outputs.groundwork.rules]]` tables and apply only to this
output, so other outputs still see the original metrics.

Rules are evaluated in the order they are defined and the first matching rule
wins. A rule matches when:

* the configured `tag` or `field` exists on the metric (or neither is set),
* the configured `pattern` (a regular expression) matches the value, if set,
* and the resulting value is not empty.

The `target` of a rule is one of the following:

* `resource` - the host the service belongs to (default: the `resource_tag`
  tag, or `default_host` if it is not set)
* `group` - the group the host belongs to (default: the `group_tag` tag)
* `service` - the name of the service (default: the `service` tag, or the
  metric name)
* `status` - the service status (default: the `status` tag or field, the
  status computed from thresholds, or `default_service_state`)
* `message` - the output message (default: the `message` tag or field)
* `skip` - the metric is not sent to GroundWork and is not retried later

If no rule matches for a target, the default handling is used.

The `pattern` is a regular expression matched against the value of the
configured `tag` or `field`. Without a `template`, the part of the value that
the pattern matched is used as the resulting value. With a `template`,
the value is rendered with a [Go template][template] that has the Sprig
functions available. The metric is available as `.` with the usual `.Tag`,
`.Field`, `.Name`, ... accessors, and these extra keys are available:

* `.Value` - the raw value of the configured tag or field
* `.Match` - the part of the value matched by the regular expression
* `.Groups` - the named capture groups of the regular expression

The metrics skipped with a `skip` rule are simply not sent to GroundWork and
are not retried. The other metrics of the same batch are sent as usual. Note
that "skipped" only applies to this output, the metrics can still be written
by other outputs.

Tags and fields used as the source of a rule are not sent as properties or
metrics, similar to the fixed tags and fields used by the plugin (e.g.
`service`, `status`, `host`, ...).

Examples:

Build the host from `node_name` with a shortened form of the `host` tag as
fallback, and fall back to `default_host` otherwise:

```toml
[[outputs.groundwork.rules]]
  target = "resource"
  tag = "node_name"

[[outputs.groundwork.rules]]
  target = "resource"
  tag = "host"
  pattern = "^(?P<short>[^.]+)"
  template = "{{ .Groups.short }}"
```

Build the group from the `cluster` and `namespace` tags:

```toml
[[outputs.groundwork.rules]]
  target = "group"
  template = "{{ .Tag \"cluster\" }}-{{ .Tag \"namespace\" }}"
```

Set the status of a log metric based on its message field:

```toml
[[outputs.groundwork.rules]]
  target = "status"
  field = "message"
  pattern = "(?i)fatal|critical"
  template = "SERVICE_UNSCHEDULED_CRITICAL"
```

Skip metrics whose `kubernetes_namespace` tag starts with `kube-`:

```toml
[[outputs.groundwork.rules]]
  target = "skip"
  tag = "kubernetes_namespace"
  pattern = "^kube-"
```

[template]: https://pkg.go.dev/text/template

## List of tags used by the plugin

* __group__ - to define the name of the group you want to monitor,
  can be changed with config.
* __host__ - to define the name of the host you want to monitor,
  can be changed with config.
* __service__ - to define the name of the service you want to monitor.
* __status__ - to define the status of the service. Supported statuses:
  "SERVICE_OK", "SERVICE_WARNING", "SERVICE_UNSCHEDULED_CRITICAL",
  "SERVICE_PENDING", "SERVICE_SCHEDULED_CRITICAL", "SERVICE_UNKNOWN".
* __message__ - to provide any message you want,
  it overrides __message__ field value.
* __unitType__ - to use in monitoring contexts (subset of The Unified Code for
  Units of Measure standard). Supported types: "1", "%cpu", "KB", "GB", "MB".
* __critical__ - to define the default critical threshold value,
  it overrides value_cr field value.
* __warning__ - to define the default warning threshold value,
  it overrides value_wn field value.
* __value_cr__ - to define critical threshold value,
  it overrides __critical__ tag value and __value_cr__ field value.
* __value_wn__ - to define warning threshold value,
  it overrides __warning__ tag value and __value_wn__ field value.

## NOTE

The current version of GroundWork Monitor does not support metrics whose values
are strings. Such metrics will be skipped and will not be added to the final
payload. You can find more context in this pull request: [#10255][].

[#10255]: https://github.com/influxdata/telegraf/pull/10255
