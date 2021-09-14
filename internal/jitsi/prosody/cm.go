package prosody

type TurnConfig struct {
	XMPPDomain, TurnCredentials, TurnHost, StunHost, TurnPort, StunPort, TurnsPort string
	StunEnabled, TurnUDPEnabled                                                    bool
}

const prosodyTurnConfig = `muc_mapper_domain_base = "{{ .XMPPDomain }}"

turncredentials_secret = "{{ .TurnCredentials }}";

turncredentials = {
	{{- if .StunEnabled }}
    { type = "stun", host = "{{ .StunHost }}", port = "{{ .StunPort }}" },
	{{ end }}
	{{- if .TurnUDPEnabled }}
    { type = "turn", host = "{{ .TurnHost }}", port = "{{ .TurnPort }}", transport = "udp"},
	{{ end }}
    { type = "turns", host = "{{ .TurnHost }}", port = "{{ .TurnsPort }}", transport = "tcp" }
}

external_service_secret = "{{ .TurnCredentials }}";
external_services = {
	{{- if .StunEnabled }}
    {
        type = "stun",
        transport = "udp",
        host = "{{ .StunHost }}",
        port = {{ .StunPort }}
    }, 
	{{- end }}
	{{- if .TurnUDPEnabled }}
    {
        type = "turn",
        transport = "udp",
		ttl = 86400, 
		algorithm = "turn",
        host = "{{ .TurnHost }}",
        port = {{ .TurnPort }},
        secret = true
    }, 
	{{- end }}
	{
        type = "turn",
        transport = "tcp",
		ttl = 86400, 
		algorithm = "turn",
        host = "{{ .TurnHost }}",
        port = {{ .TurnsPort }},
        secret = true
    }
}`
