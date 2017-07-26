# butler - Prometheus Configuration Management System (PCMS)

The butler tool is designed to grab Prometheus configuration files from a remote location/repository via http and side load them onto another locally running Prometheus container.

## Usage
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

## Building
## Pushing
You need to `export ARTIFACTORY_USER="<YOUR ARTIFACTORY USERNAME>"` prior to doing your push. You may want to just have this environment variable inside of your dot.profile.  Make push-butler-<whatever> requires this environment variable to be set to whatever your artifactory login is.
