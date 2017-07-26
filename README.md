# butler

## Prometheus Configuration Management System (PCMS) Overview
The butler tool is designed to grab Prometheus configuration files from a remote location/repository via http and side load them onto another locally running Prometheus container.

## Usage
There are various ways that you can run butler. We will ultimately deploy butler via DCOS, you can run this on your local machine to do some testing.
### Command Line Usage
```
[12:51]pts/11:11(stegen@woden):[~/git/ethos/butler]% go run butler.go -help
Usage of /tmp/go-build176186906/command-line-arguments/_obj/exe/butler:
  -config.additional-config string
    	The prometheus configuration files to grab in comma separated format. (default "alerts/commonalerts.yml,alerts/tenant.yml")
  -config.http-timeout-host int
    	The http timeout, in seconds, for GET requests to gather the configuration files (default 10)
  -config.mustache-subs string
    	prometheus.yml Mustache Substitutions.
  -config.prometheus-config string
    	The prometheus configuration file. (default "prometheus.yml")
  -config.prometheus-host string
    	The prometheus host to reload.
  -config.scheduler-interval int
    	The interval, in seconds, to run the scheduler. (default 300)
  -config.url string
    	The base url to grab prometheus configuration files
  -version
    	Print version information.
exit status 2

[master]
[12:51]pts/11:12(stegen@woden):[~/git/ethos/butler]% 
```
### DCOS Deployment JSON
```
{
  "volumes": null,
  "id": "/prometheus-server-butler2",
  "cmd": null,
  "args": [
    "-config.url",
    "http://10.14.210.14/cgit/adobe-platform/ethos-monitoring/plain/oncluster/",
    "-config.mustache-subs",
    "ethos-cluster-id=ethos01-dev-or1",
    "-config.additional-config",
    "alerts/commonalerts.yml,alerts/tenant.yml",
    "-config.prometheus-config",
    "prometheus.yml",
    "-config.scheduler-interval",
    "300"
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

## Building
## Pushing
You need to `export ARTIFACTORY_USER="<YOUR ARTIFACTORY USERNAME>"` prior to doing your push. You may want to just have this environment variable inside of your dot.profile.  Make push-butler-<whatever> requires this environment variable to be set to whatever your artifactory login is.
