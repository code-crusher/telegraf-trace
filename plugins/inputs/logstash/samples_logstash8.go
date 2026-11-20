package logstash

const logstash8PipelinesJSON = `
{
  "host" : "node-8",
  "version" : "8.11.4",
  "http_address" : "127.0.0.1:9600",
  "id" : "4793d0bd-2b46-431f-8f10-7de3f743c5f8",
  "name" : "node-8-test",
  "ephemeral_id" : "847d5c1d-3e4e-4e48-8f94-e613fdba5c4b",
  "status" : "green",
  "snapshot" : false,
  "pipeline" : {
    "workers" : 4,
    "batch_size" : 125,
    "batch_delay" : 50
  },
  "pipelines" : {
    "main" : {
      "events" : {
        "duration_in_millis" : 6164877,
        "in" : 4342655,
        "out" : 4342655,
        "filtered" : 4342655,
        "queue_push_duration_in_millis" : 40829
      },
      "flow" : {
        "input_throughput" : {
          "current" : 594.3,
          "last_1_minute" : 588.1,
          "lifetime" : 575.4
        },
        "filter_throughput" : {
          "current" : 594.6,
          "last_1_minute" : 587.9,
          "lifetime" : 575.1
        },
        "output_throughput" : {
          "current" : 595.2,
          "last_1_minute" : 588.3,
          "lifetime" : 575.1
        },
        "queue_backpressure" : {
          "current" : 0.0214,
          "last_1_minute" : 0.0185,
          "lifetime" : 0.0349
        },
        "worker_concurrency" : {
          "current" : 0.941,
          "last_1_minute" : 0.937,
          "lifetime" : 0.9709
        },
        "worker_utilization" : {
          "current" : 93.092,
          "last_1_minute" : 93.495,
          "lifetime" : 92.187
        }
      },
      "pipeline" : {
        "workers" : 4,
        "batch_size" : 125,
        "batch_delay" : 50
      },
      "plugins" : {
        "inputs" : [ {
          "id" : "8526dc80bc2257ab08f96018f96b0c68dd03abc5695bb22fb9e96339a8dfb4f86",
          "events" : {
            "out" : 4342655,
            "queue_push_duration_in_millis" : 40829
          },
          "flow" : {
            "throughput" : {
              "current" : 594.3,
              "last_1_minute" : 588.1,
              "lifetime" : 575.4
            }
          },
          "peak_connections" : 1,
          "name" : "beats",
          "current_connections" : 1
        } ],
        "codecs" : [ {
          "id" : "plain_7312c097-1e7f-41db-983b-4f5a87a9eba2",
          "encode" : {
            "duration_in_millis" : 0,
            "writes_in" : 0
          },
          "name" : "plain",
          "decode" : {
            "out" : 0,
            "duration_in_millis" : 0,
            "writes_in" : 0
          }
        } ],
        "filters" : [ {
          "id" : "9e8297a6ee7b61864f77853317dccde83d29952ef869010c385dcfc9064ab8b8",
          "events" : {
            "in" : 4342655,
            "out" : 4342655,
            "duration_in_millis" : 25364
          },
          "flow" : {
            "worker_utilization" : {
              "current" : 16.71,
              "last_1_minute" : 16.4,
              "lifetime" : 15.27
            },
            "worker_millis_per_event" : {
              "current" : 0.2829,
              "last_1_minute" : 0.2815,
              "lifetime" : 0.2585
            }
          },
          "name" : "date",
          "matches" : 4342655
        }, {
          "id" : "bec0c77b3f53a78c7878449c72ec59f97be31c1f12f9621f61ed2d4563bad869",
          "events" : {
            "in" : 4342655,
            "out" : 4342655,
            "duration_in_millis" : 677620
          },
          "flow" : {
            "worker_utilization" : {
              "current" : 3.6,
              "last_1_minute" : 3.4,
              "lifetime" : 3.1
            },
            "worker_millis_per_event" : {
              "current" : 0.1005,
              "last_1_minute" : 0.0995,
              "lifetime" : 0.0907
            }
          },
          "name" : "fingerprint"
        } ],
        "outputs" : [ {
          "id" : "38967f09bbd2647a95aa00702b6b557bdbbab31da6a04f991d38abe5629779e3",
          "events" : {
            "in" : 4342655,
            "out" : 4342655,
            "duration_in_millis" : 5729481
          },
          "flow" : {
            "worker_utilization" : {
              "current" : 75.34,
              "last_1_minute" : 75.61,
              "lifetime" : 76.01
            },
            "worker_millis_per_event" : {
              "current" : 1.276,
              "last_1_minute" : 1.281,
              "lifetime" : 1.287
            }
          },
          "name" : "elasticsearch"
        }, {
          "id" : "df59066a933f038354c1845ba44de692f70dbd0d2009ab07a12b98b776be7e3f",
          "events" : {
            "in" : 4342655,
            "out" : 4342655,
            "duration_in_millis" : 32247
          },
          "flow" : {
            "worker_utilization" : {
              "current" : 0.42,
              "last_1_minute" : 0.44,
              "lifetime" : 0.41
            },
            "worker_millis_per_event" : {
              "current" : 0.01422,
              "last_1_minute" : 0.01425,
              "lifetime" : 0.0133
            }
          },
          "name" : "stdout"
        } ]
      },
      "reloads" : {
        "successes" : 4,
        "last_error" : "Pipeline \"main\" configured with 1 worker and batch size 1",
        "failures" : 2,
        "last_success_timestamp" : "2024-06-05T08:06:12.538Z",
        "last_failure_timestamp" : "2024-06-04T09:15:44.101Z"
      },
      "queue" : {
        "type" : "persisted",
        "events_count" : 0,
        "queue_size_in_bytes" : 33223044,
        "max_queue_size_in_bytes" : 8589934592
      },
      "hash" : "5bc589ae4b02cb3e436626429b50928b9d99360639c84dc7fc69268ac01a9fd0",
      "ephemeral_id" : "4bcacefa-6cbf-461e-b14e-184edd9ebdf3"
    }
  }
}`
