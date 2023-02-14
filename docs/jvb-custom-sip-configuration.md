By default, JVB docker container has this sip configuration:

	{{ if .Env.DOCKER_HOST_ADDRESS }}
	org.ice4j.ice.harvest.NAT_HARVESTER_LOCAL_ADDRESS={{ .Env.LOCAL_ADDRESS }}
	org.ice4j.ice.harvest.NAT_HARVESTER_PUBLIC_ADDRESS={{ .Env.DOCKER_HOST_ADDRESS }}
	{{ end }}

That's mean we can't configure anything except "HARVESTER ADDRESS" for example:
```	org.jitsi.videobridge.ENABLE_REST_SHUTDOWN=true // enabled by default in operator```

But now we can simply set up additional configuration via "custom_sip"
field in Jitsi Custom Resource.
For instance:
```
    custom_sip:
      - org.jitsi.videobridge.ENABLE_STATISTICS=true
```
