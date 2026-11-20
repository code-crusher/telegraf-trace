# Logstash Input Plugin

This plugin gathers metrics from a [Logstash][logstash] endpoint using the
[Monitoring API][logstash_api].

> [!NOTE]
> This plugin supports Logstash 5+.

⭐ Telegraf v1.12.0
🏷️ server
💻 all

[logstash]: https://www.elastic.co/logstash
[logstash_api]: https://www.elastic.co/guide/en/logstash/current/monitoring-logstash.html

## Global configuration options <!-- @/docs/includes/plugin_config.md -->

Plugins support additional global and plugin configuration settings for tasks
such as modifying metrics, tags, and fields, creating aliases, and configuring
plugin ordering. See [CONFIGURATION.md][CONFIGURATION.md] for more details.

[CONFIGURATION.md]: ../../../docs/CONFIGURATION.md#plugins

## Configuration

```toml @sample.conf
# Read metrics exposed by Logstash
[[inputs.logstash]]
  ## The URL of the exposed Logstash API endpoint.
  url = "http://127.0.0.1:9600"

  ## Use Logstash 5 single pipeline API, set to true when monitoring
  ## Logstash 5.
  # single_pipeline = false

  ## Enable optional collection components.  Can contain
  ## "pipelines", "process", and "jvm".
  # collect = ["pipelines", "process", "jvm"]

  ## Timeout for HTTP requests.
  # timeout = "5s"

  ## Optional HTTP Basic Auth credentials.
  # username = "username"
  # password = "pa$$word"

  ## Optional TLS Config.
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"

  ## Use TLS but skip chain & host verification.
  # insecure_skip_verify = false

  ## If 'use_system_proxy' is set to true, Telegraf will check env vars such as
  ## HTTP_PROXY, HTTPS_PROXY, and NO_PROXY (or their lowercase counterparts).
  ## If 'use_system_proxy' is set to false (default) and 'http_proxy_url' is
  ## provided, Telegraf will use the specified URL as HTTP proxy.
  # use_system_proxy = false
  # http_proxy_url = "http://localhost:8888"

  ## Optional HTTP headers.
  # [inputs.logstash.headers]
  #   "X-Special-Header" = "Special-Value"
```

## Metrics

Additional plugin stats may be collected (because logstash doesn't consistently
expose all stats)

- logstash_jvm
  - tags:
    - node_id
    - node_name
    - node_host
    - node_version
  - fields:
    - threads_peak_count
    - mem_pools_survivor_peak_max_in_bytes
    - mem_pools_survivor_max_in_bytes
    - mem_pools_old_peak_used_in_bytes
    - mem_pools_young_used_in_bytes
    - mem_non_heap_committed_in_bytes
    - threads_count
    - mem_pools_old_committed_in_bytes
    - mem_pools_young_peak_max_in_bytes
    - mem_heap_used_percent
    - gc_collectors_young_collection_time_in_millis
    - mem_pools_survivor_peak_used_in_bytes
    - mem_pools_young_committed_in_bytes
    - gc_collectors_old_collection_time_in_millis
    - gc_collectors_old_collection_count
    - mem_pools_survivor_used_in_bytes
    - mem_pools_old_used_in_bytes
    - mem_pools_young_max_in_bytes
    - mem_heap_max_in_bytes
    - mem_non_heap_used_in_bytes
    - mem_pools_survivor_committed_in_bytes
    - mem_pools_old_max_in_bytes
    - mem_heap_committed_in_bytes
    - mem_pools_old_peak_max_in_bytes
    - mem_pools_young_peak_used_in_bytes
    - mem_heap_used_in_bytes
    - gc_collectors_young_collection_count
    - uptime_in_millis

- logstash_process
  - tags:
    - node_id
    - node_name
    - source
    - node_version
  - fields:
    - open_file_descriptors
    - cpu_load_average_1m
    - cpu_load_average_5m
    - cpu_load_average_15m
    - cpu_total_in_millis
    - cpu_percent
    - peak_open_file_descriptors
    - max_file_descriptors
    - mem_total_virtual_in_bytes
    - mem_total_virtual_in_bytes

- logstash_events
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
  - fields:
    - queue_push_duration_in_millis
    - duration_in_millis
    - in
    - filtered
    - out

- logstash_pipeline
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
  - fields:
    - workers (for Logstash 8+)
    - batch_size (for Logstash 8+)
    - batch_delay (for Logstash 8+)

- logstash_flow
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
  - fields (for Logstash 8+):
    - input_throughput_current
    - input_throughput_last_1_minute
    - input_throughput_lifetime
    - filter_throughput_current
    - filter_throughput_lifetime
    - output_throughput_current
    - output_throughput_lifetime
    - queue_backpressure_current
    - queue_backpressure_lifetime
    - worker_concurrency_current
    - worker_concurrency_lifetime
    - worker_utilization_current
    - worker_utilization_lifetime

- logstash_reloads
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
  - fields:
    - successes
    - failures

- logstash_plugins
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
    - plugin_id
    - plugin_name
    - plugin_type
  - fields:
    - queue_push_duration_in_millis (for input plugins only)
    - duration_in_millis
    - in
    - out
    - failures(if exists)
    - bulk_requests_failures (for Logstash 7+)
    - bulk_requests_with_errors (for Logstash 7+)
    - documents_successes (for logstash 7+)
    - documents_retryable_failures (for logstash 7+)
    - flow_throughput_current (for Logstash 8+)
    - flow_throughput_last_1_minute (for Logstash 8+)
    - flow_throughput_lifetime (for Logstash 8+)
    - flow_worker_utilization_current (for Logstash 8+)
    - flow_worker_utilization_lifetime (for Logstash 8+)
    - flow_worker_millis_per_event_current (for Logstash 8+)
    - flow_worker_millis_per_event_lifetime (for Logstash 8+)

- logstash_queue
  - tags:
    - node_id
    - node_name
    - source
    - node_version
    - pipeline (for Logstash 6+)
    - queue_type
  - fields:
    - events
    - free_space_in_bytes
    - max_queue_size_in_bytes
    - max_unread_events
    - page_capacity_in_bytes
    - queue_size_in_bytes

## Example Output

```text
logstash_jvm,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,source=debian-stretch-logstash6.virt gc_collectors_old_collection_count=2,gc_collectors_old_collection_time_in_millis=100,gc_collectors_young_collection_count=26,gc_collectors_young_collection_time_in_millis=1028,mem_heap_committed_in_bytes=1056309248,mem_heap_max_in_bytes=1056309248,mem_heap_used_in_bytes=207216328,mem_heap_used_percent=19,mem_non_heap_committed_in_bytes=160878592,mem_non_heap_used_in_bytes=140838184,mem_pools_old_committed_in_bytes=899284992,mem_pools_old_max_in_bytes=899284992,mem_pools_old_peak_max_in_bytes=899284992,mem_pools_old_peak_used_in_bytes=189468088,mem_pools_old_used_in_bytes=189468088,mem_pools_survivor_committed_in_bytes=17432576,mem_pools_survivor_max_in_bytes=17432576,mem_pools_survivor_peak_max_in_bytes=17432576,mem_pools_survivor_peak_used_in_bytes=17432576,mem_pools_survivor_used_in_bytes=12572640,mem_pools_young_committed_in_bytes=139591680,mem_pools_young_max_in_bytes=139591680,mem_pools_young_peak_max_in_bytes=139591680,mem_pools_young_peak_used_in_bytes=139591680,mem_pools_young_used_in_bytes=5175600,threads_count=20,threads_peak_count=24,uptime_in_millis=739089 1566425244000000000
logstash_process,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,source=debian-stretch-logstash6.virt cpu_load_average_15m=0.03,cpu_load_average_1m=0.01,cpu_load_average_5m=0.04,cpu_percent=0,cpu_total_in_millis=83230,max_file_descriptors=16384,mem_total_virtual_in_bytes=3689132032,open_file_descriptors=118,peak_open_file_descriptors=118 1566425244000000000
logstash_events,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,pipeline=main,source=debian-stretch-logstash6.virt duration_in_millis=0,filtered=0,in=0,out=0,queue_push_duration_in_millis=0 1566425244000000000
logstash_pipeline,node_id=4793d0bd-2b46-431f-8f10-7de3f743c5f8,node_name=node-8-test,node_version=8.11.4,pipeline=main,source=node-8 batch_delay=50,batch_size=125,workers=4 1566425244000000000
logstash_flow,node_id=4793d0bd-2b46-431f-8f10-7de3f743c5f8,node_name=node-8-test,node_version=8.11.4,pipeline=main,source=node-8 filter_throughput_current=594.6,filter_throughput_last_1_minute=587.9,filter_throughput_lifetime=575.1,input_throughput_current=594.3,input_throughput_last_1_minute=588.1,input_throughput_lifetime=575.4,output_throughput_current=595.2,output_throughput_last_1_minute=588.3,output_throughput_lifetime=575.1,queue_backpressure_current=0.0214,queue_backpressure_last_1_minute=0.0185,queue_backpressure_lifetime=0.0349,worker_concurrency_current=0.941,worker_concurrency_last_1_minute=0.937,worker_concurrency_lifetime=0.9709,worker_utilization_current=93.092,worker_utilization_last_1_minute=93.495,worker_utilization_lifetime=92.187 1566425244000000000
logstash_reloads,node_id=4793d0bd-2b46-431f-8f10-7de3f743c5f8,node_name=node-8-test,node_version=8.11.4,pipeline=main,source=node-8 failures=2,successes=4 1566425244000000000
logstash_plugins,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,pipeline=main,plugin_id=2807cb8610ba7854efa9159814fcf44c3dda762b43bd088403b30d42c88e69ab,plugin_name=beats,plugin_type=input,source=debian-stretch-logstash6.virt out=0,queue_push_duration_in_millis=0 1566425244000000000
logstash_plugins,node_id=4793d0bd-2b46-431f-8f10-7de3f743c5f8,node_name=node-8-test,node_version=8.11.4,pipeline=main,plugin_id=38967f09bbd2647a95aa00702b6b557bdbbab31da6a04f991d38abe5629779e3,plugin_name=elasticsearch,plugin_type=output,source=node-8 flow_worker_millis_per_event_current=1.276,flow_worker_millis_per_event_last_1_minute=1.281,flow_worker_millis_per_event_lifetime=1.287,flow_worker_utilization_current=75.34,flow_worker_utilization_last_1_minute=75.61,flow_worker_utilization_lifetime=76.01 1566425244000000000
logstash_plugins,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,pipeline=main,plugin_id=7a6c973366186a695727c73935634a00bccd52fceedf30d0746983fce572d50c,plugin_name=file,plugin_type=output,source=debian-stretch-logstash6.virt duration_in_millis=0,in=0,out=0 1566425244000000000
logstash_queue,node_id=3da53ed0-a946-4a33-9cdb-33013f2273f6,node_name=debian-stretch-logstash6.virt,node_version=6.8.1,pipeline=main,queue_type=memory,source=debian-stretch-logstash6.virt events=0 1566425244000000000
```
