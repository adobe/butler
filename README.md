# butler

## Butler Configuration Management System (BCMS) Overview
The butler tool is designed to grab any configuration files, defined in its configuration file, from a remote location/repository via http(s) and side load them onto another locally running container.

The butler configuration file is a [TOML](https://github.com/toml-lang/toml) formatted file. You can store the file locally (using a mounted filesystem), or on a remote server. The proper formatting for the config file can be found [here](https://git.corp.adobe.com/TechOps-IAO/butler/tree/master/contrib)

### Butler at 30,000 feet
Here is a quick diagram that contains all the elements of what butler does, and how it is intended to interact with other systems.

![Butler Elements][elements-diagram]

### Butler Workflow
To help understand more of how butler functions, here is a work flow diagram showing, in more detail, the work butler does.

![Butler Workflow][workflow-diagram]

## Usage
There are various ways that you can run butler. We will ultimately deploy butler via DCOS, and you can run this on your local machine to do some testing.

### Command Line Usage
```
[14:22]pts/12:16(stegen@woden):[~/git/ethos/butler]% ./butler -h
Usage of ./butler:
  -config.path string
    	Full remote path to butler configuration file (eg: full URL scheme://path).
  -config.retrieve-interval int
    	The interval, in seconds, to retrieve new butler configuration files. (default 300)
  -http.retries int
    	The number of http retries for GET requests to obtain the butler configuration files (default 4)
  -http.retry_wait_max int
    	The maximum amount of time to wait before attemping to retry the http config get operation. (default 10)
  -http.retry_wait_min int
    	The minimum amount of time to wait before attemping to retry the http config get operation. (default 5)
  -http.timeout int
    	The http timeout, in seconds, for GET requests to obtain the butler configuration file. (default 10)
  -log.level string
    	The butler log level. Log levels are: debug, info, warn, error, fatal, panic. (default "info")
  -s3.region string
    	The S3 Region that the config file resides.
  -version
    	Print version information.

[master]
[14:22]pts/12:17(stegen@woden):[~/git/ethos/butler]%

```

### Example Command Line Usage
#### HTTP/HTTPS CLI
```
[14:24]pts/12:21(stegen@woden):[~/git/ethos/butler]% ./butler -config.path http://localhost/butler/config/butler.toml -config.retrieve-interval 10 -log.level info
INFO[2017-10-11T14:24:29+01:00] Starting butler version v1.0.0
^C

[master]
[14:24]pts/12:22(stegen@woden):[~/git/ethos/butler]%
```
When you execute butler with the above arguments, you are asking butler to grab its configuration file from http://localhost/butler/config/butler.toml, and try to re-retrieve and refresh it every 10 seconds. It will also use the default log level of INFO. If you need more verbosity to your output, specify `debug` as the logging level argument.

#### S3 CLI
```
[14:24]pts/12:21(stegen@woden):[~/git/ethos/butler]% ./butler -config.path s3://s3-bucket/config/butler.toml -config.retrieve-interval 10 -log.level info -s3.region <aws-region>
INFO[2017-10-11T14:24:29+01:00] Starting butler version v1.0.0
^C

[master]
[14:24]pts/12:22(stegen@woden):[~/git/ethos/butler]%
```
When you execute butler with the above arguments, you are asking butler to grab its configuration file from S3 storage using bucket `s3-bucket`, file key `config/butler.toml` and the aws-region as specified by `s3.region`, and try to re-retrieve and refresh it every 10 seconds. It will also use the default log level of INFO. If you need more verbosity to your output, specify `debug` as the logging level argument.

### DCOS Deployment JSON
```
{
  "volumes": null,
  "id": "/prometheus-server-butler2",
  "cmd": null,
  "args": [
    "-config.path",
    "http://10.14.210.14/cgit/adobe-platform/ethos-monitoring/plain/oncluster/butler.toml"
  ],
  "user": null,
  "env": null,
  "instances": 1,
  "cpus": 0.05,
  "mem": 20,
  "disk": 0,
  "gpus": 0,
  "executor": null,
  "constraints": [
    [
      "hostname",
      "LIKE",
      "10.14.211.16"
    ],
    [
      "hostname",
      "UNIQUE"
    ]
  ],
  "fetch": null,
  "storeUrls": null,
  "backoffSeconds": 1,
  "backoffFactor": 1.15,
  "maxLaunchDelaySeconds": 3600,
  "container": {
    "docker": {
      "image": "docker-ethos-core-univ-dev.dr-uw2.adobeitc.com/ethos/butler:x.y.z",
      "forcePullImage": false,
      "privileged": false,
      "parameters": [
        {
          "key": "volume",
          "value": "/opt/prometheus:/opt/prometheus:z,rw"
        }
      ],
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp",
          "servicePort": 10057
        }
      ],
      "network": "BRIDGE"
    }
  },
  "healthChecks": [
    {
      "protocol": "HTTP",
      "path": "/health-check",
      "gracePeriodSeconds": 5,
      "intervalSeconds": 20,
      "timeoutSeconds": 20,
      "maxConsecutiveFailures": 3,
      "ignoreHttp1xx": false
    }
  ],
  "readinessChecks": null,
  "dependencies": null,
  "upgradeStrategy": {
    "minimumHealthCapacity": 1,
    "maximumOverCapacity": 1
  },
  "labels": null,
  "acceptedResourceRoles": [
    "*",
    "slave_public"
  ],
  "residency": null,
  "secrets": null,
  "taskKillGracePeriodSeconds": null,
  "portDefinitions": [
    {
      "port": 10057,
      "protocol": "tcp",
      "labels": {
      }
    }
  ],
  "requirePorts": false
}
```

## Butler Configuration File
Refer to the contrib/ directory for more information about the butler.toml configuration file, and all its features.

## Building
## Pushing
You need to `export ARTIFACTORY_USER="<YOUR ARTIFACTORY USERNAME>"` prior to doing your push. You may want to just have this environment variable inside of your dot.profile.  Make push-butler-<whatever> requires this environment variable to be set to whatever your artifactory login is.

## Health Checks
butler provides DCOS health checks by exposing an http service with a /health-check endpoint. It exposes various configuration, and realtime information in JSON format regarding the butler process.
```
[12:54]pts/11:13(stegen@woden):[~/git/ethos/butler]% http get localhost:8080/health-check
HTTP/1.1 200 OK
Content-Type: application/json
Date: Thu, 12 Oct 2017 10:44:50 GMT
Transfer-Encoding: chunked

{
    "config-path": "http://localhost/butler/config/butler.toml", 
    "config-settings": {
        "globals": {
            "exit-on-failure": false, 
            "scheduler-interval": 10
        }, 
        "managers": {
            "alertmanager": {
                "cache-path": "/opt/cache/alertmanager", 
                "clean-files": true, 
                "dest-path": "/opt/alertmanager", 
                "enable-cache": true, 
                "last-run": "2017-10-12T11:44:50.29930327+01:00", 
                "mustache-subs": {
                    "endpoint": "external", 
                    "ethos-cluster-id": "ethos01-dev-or1"
                }, 
                "name": "alertmanager", 
                "opts": {
                    "alertmanager.repo3.domain.com": {
                        "additional-config": null, 
                        "method": "http", 
                        "opts": {
                            "retries": 5, 
                            "retry-wait-max": 10, 
                            "retry-wait-min": 5, 
                            "timeout": 10
                        }, 
                        "primary-config": [
                            "alertmanager.yml"
                        ], 
                        "repo": "repo3.domain.com", 
                        "uri-path": "/butler/configs/alertmanager"
                    }, 
                    "alertmanager.repo4.domain.com": {
                        "additional-config": null, 
                        "method": "http", 
                        "opts": {
                            "retries": 5, 
                            "retry-wait-max": 10, 
                            "retry-wait-min": 5, 
                            "timeout": 10
                        }, 
                        "primary-config": [
                            "alertmanager-2.yml"
                        ], 
                        "repo": "repo4.domain.com", 
                        "uri-path": "/butler/configs/alertmanager"
                    }
                }, 
                "primary-config-name": "alertmanager.yml", 
                "reloader": {
                    "method": "http", 
                    "opts": {
                        "content-type": "application/json", 
                        "host": "localhost", 
                        "method": "post", 
                        "payload": "{}", 
                        "port": 9093, 
                        "retries": 5, 
                        "retry-wait-max": 10, 
                        "retry-wait-min": 5, 
                        "timeout": 10, 
                        "uri": "/-/reload"
                    }
                }, 
                "urls": [
                    "repo3.domain.com", 
                    "repo4.domain.com"
                ]
            }, 
            "prometheus": {
                "cache-path": "/opt/cache/prometheus", 
                "clean-files": true, 
                "dest-path": "/opt/prometheus", 
                "enable-cache": true, 
                "last-run": "2017-10-12T11:44:50.29659399+01:00", 
                "mustache-subs": {
                    "endpoint": "external", 
                    "ethos-cluster-id": "ethos01-dev-or1"
                }, 
                "name": "prometheus", 
                "opts": {
                    "prometheus.repo1.domain.com": {
                        "additional-config": [
                            "alerts/commonalerts.yml", 
                            "butler/butler.yml"
                        ], 
                        "method": "http", 
                        "opts": {
                            "retries": 5, 
                            "retry-wait-max": 10, 
                            "retry-wait-min": 5, 
                            "timeout": 10
                        }, 
                        "primary-config": [
                            "prometheus.yml", 
                            "prometheus-other.yml"
                        ], 
                        "repo": "repo1.domain.com", 
                        "uri-path": "/butler/configs/prometheus"
                    }, 
                    "prometheus.repo2.domain.com": {
                        "additional-config": [
                            "alerts/tenant.yml", 
                            "butler/butler-repo2.yml"
                        ], 
                        "method": "http", 
                        "opts": {
                            "retries": 5, 
                            "retry-wait-max": 10, 
                            "retry-wait-min": 5, 
                            "timeout": 10
                        }, 
                        "primary-config": [
                            "prometheus-repo2.yml", 
                            "prometheus-repo2-other.yml"
                        ], 
                        "repo": "repo2.domain.com", 
                        "uri-path": "/butler/configs/prometheus"
                    }
                }, 
                "primary-config-name": "prometheus.yml", 
                "reloader": {
                    "method": "http", 
                    "opts": {
                        "content-type": "application/json", 
                        "host": "localhost", 
                        "method": "post", 
                        "payload": "{}", 
                        "port": 9090, 
                        "retries": 5, 
                        "retry-wait-max": 10, 
                        "retry-wait-min": 5, 
                        "timeout": 10, 
                        "uri": "/-/reload"
                    }
                }, 
                "urls": [
                    "repo1.domain.com", 
                    "repo2.domain.com"
                ]
            }
        }
    }, 
    "log-level": 5, 
    "retrieve-interval": 10, 
    "version": "v1.0.0"
}

[master]
[13:02]pts/11:14(stegen@woden):[~/git/ethos/butler]% 
```
## Prometheus Metrics
butler provides native Prometheus of the butler go binary by exposing an http service with a /metrics endpoint. This includes both butler specific metric information (prefixed with `butler_`), and internal go and process related metrics (prefixed with `go_` and `process_`)
```
[13:04]pts/11:15(stegen@woden):[~/git/ethos/butler]% http get localhost:8080/metrics 
HTTP/1.1 200 OK
Content-Encoding: gzip
Content-Length: 1381
Content-Type: text/plain; version=0.0.4
Date: Wed, 26 Jul 2017 12:04:57 GMT

# HELP butler_localconfig_reload_success Did butler successfully reload prometheus
# TYPE butler_localconfig_reload_success gauge
butler_localconfig_reload_success 1
# HELP butler_localconfig_reload_time Time that butler successfully reload prometheus
# TYPE butler_localconfig_reload_time gauge
butler_localconfig_reload_time 1.501070697e+09
# HELP butler_localconfig_render_success Did butler successfully render the prometheus.yml
# TYPE butler_localconfig_render_success gauge
butler_localconfig_render_success 1
# HELP butler_localconfig_render_time Time that butler successfully rendered the prometheus.yml
# TYPE butler_localconfig_render_time gauge
butler_localconfig_render_time 1.501070527e+09
# HELP butler_remoterepo_config_valid Is the butler configuration valid
# TYPE butler_remoterepo_config_valid gauge
butler_remoterepo_config_valid{config_file="commonalerts.yml"} 1
butler_remoterepo_config_valid{config_file="prometheus.yml"} 1
butler_remoterepo_config_valid{config_file="tenant.yml"} 1
# HELP butler_remoterepo_contact_success Did butler succesfully contact the remote repository
# TYPE butler_remoterepo_contact_success gauge
butler_remoterepo_contact_success{config_file="commonalerts.yml"} 1
butler_remoterepo_contact_success{config_file="prometheus.yml"} 1
butler_remoterepo_contact_success{config_file="tenant.yml"} 1
# HELP butler_remoterepo_contact_time Time that butler succesfully contacted the remote repository
# TYPE butler_remoterepo_contact_time gauge
butler_remoterepo_contact_time{config_file="commonalerts.yml"} 1.501070685e+09
butler_remoterepo_contact_time{config_file="prometheus.yml"} 1.501070697e+09
butler_remoterepo_contact_time{config_file="tenant.yml"} 1.501070685e+09
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0.000236024
go_gc_duration_seconds{quantile="0.25"} 0.000236024
go_gc_duration_seconds{quantile="0.5"} 0.000236024
go_gc_duration_seconds{quantile="0.75"} 0.000236024
go_gc_duration_seconds{quantile="1"} 0.000236024
go_gc_duration_seconds_sum 0.000236024
go_gc_duration_seconds_count 1
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 18
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 3.3794e+06
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 6.070552e+06
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.445366e+06
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 15629
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 333824
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 3.3794e+06
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 106496
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 4.251648e+06
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 17404
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 0
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 4.358144e+06
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.5010706071801429e+09
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 466
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 33033
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 9600
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 16384
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 45144
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 49152
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 4.194304e+06
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 1.012482e+06
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 884736
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 884736
# HELP go_memstats_sys_bytes Number of bytes obtained from system.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 8.100088e+06
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.12
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1024
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 31
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 1.0821632e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.50107052601e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.39341568e+08


[master]
[13:04]pts/11:16(stegen@woden):[~/git/ethos/butler]% 
```
[workflow-diagram]: https://git.corp.adobe.com/TechOps-IAO/butler/raw/more_docs/contrib/diagrams/png/Butler%20Workflow.png
[elements-diagram]: https://git.corp.adobe.com/TechOps-IAO/butler/raw/more_docs/contrib/diagrams/png/Butler%20Elements.png
