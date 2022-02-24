module github.com/onmetal/meeting-operator

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.4.0
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.18.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/controller-runtime v0.7.2
)
