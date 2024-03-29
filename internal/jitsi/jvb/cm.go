// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package jvb

type SIP struct {
	Options []string
}

type TurnConfig struct {
	XMPPDomain, TurnCredentials, TurnHost, StunHost, TurnPort, StunPort, TurnsPort string
	StunEnabled, TurnUDPEnabled                                                    bool
}

const (
	loggingLevel     = "LOGGING_LEVEL"
	loggingLevelInfo = "INFO"
)

const jvbGracefulShutdown = `
	#!/bin/bash
    #
    # 1. The script issues shutdown command to the bridge over REST API.
    #    If HTTP status code other than 200 is returned then it exits with 1.
    # 2. If the code is ok then it checks if the bridge has exited.
    # 3. If not then it polls bridge statistics until conference count drops to 0.
    # 4. Gives some time for the bridge to shutdown. If it does not quit after that
    #    time then it kills the process. If the process was successfully killed 0 is
    #    returned and 1 otherwise.
    #
    #   Arguments:
    #   "-p"(mandatory) the PID of jitsi Videobridge process
    #   "-h"("http://localhost:8080" by default) REST requests host URI part
    #   "-t"("25" by default) number of second we we for the bridge to shutdown
    #       gracefully after conference count drops to 0
    #   "-s"(disabled by default) enable silent mode - no info output
    #
    #   NOTE: script depends on the tools jq, used to parse json, and curl
    #
    
    # Initialize arguments
    hostUrl="http://localhost:8080"
    timeout=25
    verbose=1
    
    # Parse arguments
    OPTIND=1
    while getopts "p:h:t:s" opt; do
        case "$opt" in
        p)
            pid=$OPTARG
            ;;
        h)
            hostUrl=$OPTARG
            ;;
        t)
            timeout=$OPTARG
            ;;
        s)
            verbose=0
            ;;
        *)
            echo "usage: $0 [-p] [-h] [-t] [-s]" >&2
            exit 1
            ;;
        esac
    done
    shift "$((OPTIND-1))"
    
    # Get PID from supervisor if no PID was provided
    if [ "$pid" = "" ] ;then
        pid=` + "`s6-svstat -o pid /var/run/s6/services/jvb`\n" +
	`
	fi
    
    # Check if PID is a number
    re='^[0-9]+$'
    if ! [[ $pid =~ $re ]] ; then
       echo "error: PID is not a number" >&2; exit 1
    fi
    
    # Returns conference count by calling JVB REST statistics API and extracting
    # conference count from JSON stats text returned.
    function getConferenceCount {
        # Total number of conferences minus the empty conferences
        curl -s "$hostUrl/colibri/stats"| jq '.conferences - .conference_sizes[0]'
    }
    
    # Prints info messages
    function printInfo {
      if [ "$verbose" == "1" ]
      then
        echo "$@"
      fi
    }
    
    # Prints errors
    function printError {
      echo "$@" 1>&2
    }
    
    shutdownStatus=` + "`curl -s -o /dev/null -H \"Content-MonitoringType: application/json\"" +
	" -d '{ \"graceful-shutdown\": \"true\" }' -w \"%{http_code}\" \"$hostUrl/colibri/shutdown\"`\n" +
	`	
	if [ "$shutdownStatus" == "200" ]
    then
      printInfo "Graceful shutdown started"
    
      # turn off automatic restart of JVB service
      s6-svc -O /var/run/s6/services/jvb
    
      confCount=` + "`getConferenceCount`\n" +
	`
	  while [[ $confCount -gt 0 ]] ; do
        printInfo "There are still $confCount conferences"
        sleep 10
        confCount=` + "`getConferenceCount`\n" +
	`
	  done
    
      sleep 5
    
      jvbAvailable=` + "`curl -s -o /dev/null -w \"%{http_code}\" \"$hostUrl/colibri/stats\"`\n" +
	`
	  if [ "$jvbAvailable" == "200" ]
      then
        printInfo "It is still running, lets give it $timeout seconds"
        sleep $timeout
        jvbAvailable=` + "`curl -s -o /dev/null -w \"%{http_code}\" \"$hostUrl/colibri/stats\"`\n" +
	`
		if [ "$jvbAvailable" == "200" ]
        then
          printError "Bridge did not exit after $timeout sec - killing $pid"
        fi
      fi
      kill $pid
    
      # check for 3 seconds if we managed to kill
      for I in 1 2 3
      do
        if ps -p $pid > /dev/null 2>&1
        then
          sleep 1
        fi
      done
      if ps -p $pid > /dev/null 2>&1
      then
        printError "Failed to kill $pid"
        printError "Sending force kill to $pid"
        kill -9 $pid
        if ps -p $pid > /dev/null 2>&1
        then
          printError "Failed to force kill $pid, giving up."
          exit 1
        fi
      fi
      printInfo "Bridge shutdown OK"
      exit 0
    else
      printError "Invalid HTTP status for shutdown request: $shutdownStatus"
      exit 1
    fi
`

const jvbCustomSIP = `{{"{{ if .Env.DOCKER_HOST_ADDRESS }}"}}
org.ice4j.ice.harvest.NAT_HARVESTER_LOCAL_ADDRESS={{"{{ .Env.LOCAL_ADDRESS }}"}}
org.ice4j.ice.harvest.NAT_HARVESTER_PUBLIC_ADDRESS={{"{{ .Env.DOCKER_HOST_ADDRESS }}"}}
org.jitsi.videobridge.ENABLE_REST_SHUTDOWN=true
{{"{{ end }}"}}
{{ range $s := .Options }}{{ printf "%s\n" $s }} {{ end }}`

const jvbCustomLogging = `handlers= java.util.logging.ConsoleHandler

java.util.logging.ConsoleHandler.level = ALL
java.util.logging.ConsoleHandler.formatter = net.java.sip.communicator.util.ScLogFormatter

net.java.sip.communicator.util.ScLogFormatter.programname=JVB

.level={{ . }}

org.jitsi.videobridge.xmpp.ComponentImpl.level=FINE

# All of the INFO level logs from MediaStreamImpl are unnecessary in the context of jitsi-videobridge.
org.jitsi.impl.neomedia.MediaStreamImpl.level=WARNING`
