// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

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
