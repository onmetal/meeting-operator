Meeting-operator supports two exporters right now:

1. [prometheus-jitsi-meet-exporter](https://github.com/systemli/prometheus-jitsi-meet-exporter)  - Prometheus Exporter for Jitsi Meet written in Go.
2. [Telegraf](https://github.com/influxdata/telegraf) - is an agent for collecting, processing, aggregating, and writing metrics.

By default, operator will install that [exporter](https://github.com/systemli/prometheus-jitsi-meet-exporter).

If you want to use telegraf instead of prometheus:
```
  jvb:
    exporter:
      type: "telegraf"
      image: "telegraf:latest"
      image_pull_policy: "Always"
      config_map_name: telegraf-config
      security_context:
        runAsNonRoot: true
      resources:
        requests:
          cpu: "0.1"
          memory: "128Mi"
      environments:
        - name: INFLUX_HOST
          value: "http://localhost:8086"
```