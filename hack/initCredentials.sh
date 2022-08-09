#!/bin/bash

NEXUS_PASS=$(oc get secrets jenkins-slave-nexus-mount -o json -n aup-build| jq '.data["nexus.json"]' -r |base64 -d | jq . -r)

export NEXUS_USERNAME=$(echo $NEXUS_PASS | jq .username -r)
export NEXUS_PASSWORD=$(echo $NEXUS_PASS | jq .password -r)
export NEXUS_URL=$(echo $NEXUS_PASS | jq .nexusUrl -r)
