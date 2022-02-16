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

package jicofo

const (
	loggingLevel     = "LOGGING_LEVEL"
	loggingLevelInfo = "INFO"
)

const jicofoCustomLogging = `handlers= java.util.logging.ConsoleHandler

java.util.logging.ConsoleHandler.level = ALL
java.util.logging.ConsoleHandler.formatter = net.java.sip.communicator.util.ScLogFormatter

net.java.sip.communicator.util.ScLogFormatter.programname=Jicofo

.level={{ . }}
net.sf.level=SEVERE
net.java.sip.communicator.plugin.reconnectplugin.level=FINE
org.ice4j.level=SEVERE
org.jitsi.impl.neomedia.level=SEVERE

# Do not worry about missing strings
net.java.sip.communicator.service.resources.AbstractResourcesService.level=SEVERE

#net.java.sip.communicator.service.protocol.level=ALL

# Enable debug packets logging
#org.jitsi.impl.protocol.xmpp.level=FINE`
