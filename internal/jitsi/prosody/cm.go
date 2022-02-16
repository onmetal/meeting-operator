// /*
// Copyright (c) 2021 T-Systems International GmbH, SAP SE or an SAP affiliate company. All right reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
