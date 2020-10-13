#!/bin/bash

#if ! command -v xmlstarlet &> /dev/null
#then
#    echo "Installer xmlstarlet:)"
#    exit
#fi



#export NEXUS_USERNAME=$(xmlstarlet sel -N s="http://maven.apache.org/SETTINGS/1.0.0" -t -m "/s:settings/s:servers/s:server[s:id = 'nexus']" -v s:username  ~/.m2/settings.xml)
#export NEXUS_PASSWORD=$(xmlstarlet sel -N s="http://maven.apache.org/SETTINGS/1.0.0" -t -m "/s:settings/s:servers/s:server[s:id = 'nexus']" -v s:password  ~/.m2/settings.xml)
#export NEXUS_URL=$(xmlstarlet sel -N s="http://maven.apache.org/SETTINGS/1.0.0" -t -m "/s:settings/s:mirrors/s:mirror[s:id = 'nexus']" -v s:url  ~/.m2/settings.xml)

NEXUS_PASS=$(oc get secrets jenkins-slave-go-nexus-mount -o json -n aurora-build| jq '.data["nexus.json"]' -r |base64 -d | jq . -r)

export NEXUS_USERNAME=$(echo $NEXUS_PASS | jq .username -r)
export NEXUS_PASSWORD=$(echo $NEXUS_PASS | jq .password -r)
export NEXUS_URL=$(echo $NEXUS_PASS | jq .nexusUrl -r)
