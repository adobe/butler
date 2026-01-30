module github.com/adobe/butler

go 1.21

// Module path migrations for packages that have moved
replace (
	// The bouk/monkey package moved to bou.ke/monkey  
	github.com/bouk/monkey => bou.ke/monkey v1.0.2
	// The coreos/bbolt package moved to go.etcd.io/bbolt
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.8
	// Pin grpc to version that still has naming package
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)

require (
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible
	github.com/Jeffail/gabs v1.4.0
	github.com/aws/aws-sdk-go v1.55.8
	github.com/bouk/monkey v1.0.2
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/jasonlvhit/gocron v0.0.1
	github.com/mslocrian/mustache v0.0.0-20180126170304-0ba5e8ce9e20
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/client_model v0.6.1
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.18.2
	github.com/udhos/equalfile v0.3.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/yaml.v2 v2.4.0
)
