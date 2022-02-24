module github.com/onmetal/meeting-operator

go 1.17

require (
	github.com/go-logr/logr v1.2.0
	github.com/influxdata/influxdb-client-go/v2 v2.4.0
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/common v0.32.1
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/controller-runtime v0.11.1
)
