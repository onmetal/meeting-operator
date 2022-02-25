module github.com/onmetal/meeting-operator

go 1.16

require (
	github.com/go-logr/logr v1.2.2
	github.com/influxdata/influxdb-client-go/v2 v2.8.0
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/common v0.32.1
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/controller-runtime v0.11.1
)
