# butler
Prometheus Configuration Management System (PCMS)

`export ETHOS_CONFIG_URL=http://git1.dev.or1.adobe.net/cgit/adobe-platform/ethos-monitoring/plain/oncluster`
`export ARTIFACTORY_USER="matthsmi"` - Make push-butler-<whatever> requires this environment variable to be set to whatever your artifactory login is.

go run butler.go -config.url http://git1.dev.or1.adobe.net/cgit/adobe-platform/ethos-monitoring/plain/oncluster -config.cluster-id ethos01-dev-or1 -config.schedule-interval 1
