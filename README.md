# butler
![Butler Logo][butler-logo]

## Butler Configuration Management System (BCMS) Overview
The butler tool is designed to grab any configuration files, defined in its configuration file, from a remote location/repository via http(s)/s3(AWS)/blob(Azure)/file/etcd and side load them onto another locally running container.

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
  -etcd.endpoints string
    	The endpoints to connect to etcd.
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

Valid schemes are: blob (Azure), etcd, file, http (or https), and s3 (AWS)

### Use of Environment Variables
Butler supports the usre of environment variables. Any field that is prefixed with `env:` will be looked up in the environment. This will work for all command line options, and MOST configuration file options.

There are only a few places in the configuration file where environment variables will not be used. Any value that is used which defines a new section/nest for the butler configuration will not look up any environment variables. This is due to how the configuration file is nested.

For example, the following settings will not do environment variable lookups.
1. In the `config-managers` section of the `butler.toml` where configuration managers are defined.
1. As the definition for the configuration manager.
1. In the `repos` section in the configuration manager section.
1. As the definition for the configuration manager repository.
1. As the `method` in the configuration manager repository.
1. As the definition for the configuration manager repository.

You should get the gist at this point. Refer to the butler.toml.sample configuration for additional examples.

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

#### etcd CLI
```
[12:34]pts/16:3(stegen@woden):[~/git/ethos/butler]% ./butler -config.path etcd://etcd.mesos/butler/butler.toml -etcd.endpoints http://etcd.mesos:1026 -log.level info
INFO[2018-05-10T11:34:05Z] Starting butler version v1.2.0
INFO[2018-05-10T11:34:05Z] Config::Init(): initializing butler config.
WARN[2018-05-10T11:34:05Z] ButlerConfig::Init() Above \"NewHttpMethod(): could not convert\" warnings may be safely disregarded.
INFO[2018-05-10T11:34:05Z] Config::Init(): butler config initialized.
INFO[2018-05-10T11:34:05Z] ButlerConfig::Handler(): entering.
INFO[2018-05-10T11:34:05Z] Config::RunCMHandler(): entering
^C
[12:34]pts/16:4(stegen@woden):[~/git/ethos/butler]%
```

You can grab the butler.toml directly from etcd, and you can also create a repo which utilizes etcd within the butler.toml. Refer to [this example](https://git.corp.adobe.com/copernicus/butler/blob/master/contrib/butler.toml.etcdtest)

You can easily add the files into etcd by the following commands:
```
etcdctl --endpoint http://etcd.mesos:1026 mkdir /butler
etcdctl --endpoint http://etcd.mesos:1026 set /butler/butler.toml "$(cat /tmp/butler.toml)"
```
Note that this should support both etcd v2 and v3.

#### S3 CLI
```
[14:24]pts/12:21(stegen@woden):[~/git/ethos/butler]% ./butler -config.path s3://s3-bucket/config/butler.toml -config.retrieve-interval 10 -log.level info -s3.region <aws-region>
INFO[2017-10-11T14:24:29+01:00] Starting butler version v1.0.0
^C

[master]
[14:24]pts/12:22(stegen@woden):[~/git/ethos/butler]%
```
When you execute butler with the above arguments, you are asking butler to grab its configuration file from S3 storage using bucket `s3-bucket`, file key `config/butler.toml` and the aws-region as specified by `s3.region`, and try to re-retrieve and refresh it every 10 seconds. It will also use the default log level of INFO. If you need more verbosity to your output, specify `debug` as the logging level argument.

#### Azure CLI and Usage
In order to use the butler Azure CLI, you must set the appropriate environment variables.
1. `BUTLER_STORAGE_TOKEN` - This is the API Token to your Azure Storage Container resource

The following environment variable is optional
1. `BUTLER_STORAGE_ACCOUNT`- This is the name of the Azure Storage Account. You can either specify this in the environment, or you can specify it in the butler.toml configuration file. See the example file for reference under `./contrib/butler.toml.sample`.

The command line option looks like this:

`[14:24]pts/12:21(stegen@woden):[~/git/ethos/butler]% ./butler -config.path blob://azure-storage-account/azure-blob-container/butler.toml -config.retrieve-interval 10 -log.level info`

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

## Testing
Butler has some unit testing, and some acceptance testing.

The unit testing is using the check.v1 testing package (gopkg.in/check.v1). The code coverage is not very impressive, but we continue to add test cases as we go. If you want to run the unit tests, just run `make test-unit`.

The acceptance testing tries to do some tests of how butler operates overall. You can provide a the butler binary with a configuration file, and run it with the `-test` flag. What this tells butler to do is to just perform a full config operation once. If there are percieved failures, it'll quit out with unix status code 1. For example, if it's unable to parse a configuration, or get some variables that it needs, it will exit out. It should also, hopefully, catch bugs which aren't caught in the unit testing, where panics may get invoked from calls that are made from functions that cannot be easily unit tested, but could be caught when running against actual configuration.

Out of the box, it tests some http:// https:// file:// endoints, which it can handle internally.

There are two additional scripts which can test against s3:// and blob:// storage. For both of these, you must set the appropriate environment variables for authenticating to the respective AWS or Azure storage service.
### Blob Testing
To test against Blob storage, you will need to export two environment variables:
1. `BUTLER_BLOB_TEST_CONFIGS`
1. `BUTLER_BLOB_TEST_RESPONSES`

`BUTLER_BLOB_TEST_CONFIGS` is a space delimited list of urls to test against.
`BUTLER_BLOB_TEST_RESPONSES` is a space delimited list of return codes which are expected against the list of delimited urls.

Here is an example:
```
export BUTLER_BLOB_TEST_CONFIGS="blob://stegentestblobva7/butler/butler1.toml blob://stegentestblobva7/butler/butler2.toml blob://stegentestblobva7/butler/butler3.toml"
export BUTLER_BLOB_TEST_RESPONSES="0 0 1"
```

The actual script that gets executed is `./files/tests/scripts/azure.sh`

### S3 Testing
To test against S3 storage, you will need to export two environment variables:
1. `BUTLER_S3_TEST_CONFIGS`
1. `BUTLER_S3_TEST_RESPONSES`

`BUTLER_S3_TEST_CONFIGS` is a space delimited list of urls to test against.
`BUTLER_S3_TEST_RESPONSES` is a space delimited list of return codes which are expected against the list of delimited urls.

Here is an example:
```
export BUTLER_S3_TEST_CONFIGS="s3://stegen-test-bucket/butler1.toml s3://stegen-test-bucket/butler2.toml s3://stegen-test-bucket/butler3.toml"
export BUTLER_S3_TEST_RESPONSES="0 1 1"
```

The actual script that gets executed is `./files/tests/scripts/s3.sh`

If you want to run the acceptance testing, just run `make test-accept`.

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
### Contributing

Contributions are welcomed! Read the [Contributing Guide](CONTRIBUTING.md) for more information.

### Licensing

This project is licensed under the Apache V2 License. See [LICENSE](LICENSE) for more information.

[workflow-diagram]: https://git.corp.adobe.com/TechOps-IAO/butler/raw/master/contrib/diagrams/png/Butler%20Workflow.png
[elements-diagram]: https://git.corp.adobe.com/TechOps-IAO/butler/raw/master/contrib/diagrams/png/Butler%20Elements.png
[butler-logo]: https://git.corp.adobe.com/TechOps-IAO/butler/raw/master/contrib/images/butler.png
