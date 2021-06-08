Resource name could be:
1. jitsi_conference - Metrics based on active JVB conference count for 15m.
2. cpu - Metrics based on "Container_Cpu_Usage" (not working with influx right now).

Prometheus example:
```
apiVersion: meeting.ko/v1alpha1
kind: AutoScaler
metadata:
  name: jas-sample
spec:
  type: "prometheus"
  host: "http://172.28.174.90:9090/"
  interval: "60s"
  scaleTargetRef:
    name: jitsi-sample
  minReplicas: 1
  maxReplicas: 3
  metrics:
    - type: "Resource"
      resource:
        name: jitsi_conference
        targetAverageUtilization: 40
```

InfluxDB example:
```
apiVersion: meeting.ko/v1alpha1
kind: AutoScaler
metadata:
  name: jas-influx-sample
  annotations:
    jas.influxdb/token: "e9XEamUiqMeoa0HmsXiZ8zIe5BDr2p8D"
#    # This could be either the organization name or the ID.
    jas.influxdb/org: "influxdata"
    jas.influxdb/bucket: "jitsi"
spec:
  type: "influxdb"
  host: "http://influx-influxdb2:80/"
  interval: "60s"
  scaleTargetRef:
    name: jitsi-sample
  minReplicas: 1
  maxReplicas: 3
  metrics:
  - type: "Resource"
    resource:
      name: jitsi_conference
      targetAverageUtilization: 40

```

For InfluxDB, you can set up next fields in annotations:
1. jas.influxdb/token = InfluxDB auth token. Field is being required.
2. jas.influxdb/org = InfluxDB organization name. If field not provided, then it would be equal to "influxdata".
3. jas.influxdb/bucket = InfluxDB bucket with jitsi metrics. If field not provided, then it would be equal to "jitsi".