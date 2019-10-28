#!/bin/bash

read -p "Sikker? Dette scriptet har NULL feilsjekking! Pass på å stå i nytt prosjekt, og helst på QA" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    echo "Installing...."
    oc apply -f refapp.json
    oc apply -f refapp-svc.json
    oc apply -f refapp-cm.json
    oc expose svc refapp
    sleep 2
    oc tag container-registry-internal-private-pull.aurora.skead.no/aurora/webleveransetest-static:0 refapp-static:default --scheduled=true --insecure=true
    oc tag container-registry-internal-private-pull.aurora.skead.no/aurora/webleveransetest-app:0 refapp-app:default --scheduled=true --insecure=true
fi
