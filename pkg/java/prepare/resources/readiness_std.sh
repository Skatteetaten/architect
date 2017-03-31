#!/usr/bin/env bash

# Denne readiness probem gir beskjed til endepunkt kontrolleren når den er klar til å motta forespørsler.
# Leveransepakke byggeren legger opp til at prosjekter selv kan levere sine egnse prober som denne. Den
# må hete readiness.sh og ligge under /bin i leveransepakken. 
# Proben må exit med 0 dersom success. Andre verdier vil melde containeren ut av tjeneste (service endepunkte)
# For mer infirmasjon se http://kubernetes.io/docs/user-guide/pod-states/
# Eks. DC config
# - containers:
# .....
#     readinessProbe:
#       exec:
#         command: [/u01/application/bin/readiness.sh]
#       initialDelaySeconds: 15
#       timeoutSeconds: 2


if [ -z "$HTTP_PORT" ]; then
  HTTP_PORT=8080
fi

if [ -z "$MANAGEMENT_HTTP_PORT" ]; then
  MANAGEMENT_HTTP_PORT=8081
fi

PORT=${HTTP_PORT}

if [ ! -z "$READINESS_ON_MANAGEMENT_PORT" ]; then
  PORT=$MANAGEMENT_HTTP_PORT
fi

if [ ! -z "$READINESS_CHECK_URL" ]; then
  HEALTH_URL="http://localhost:${PORT}${READINESS_CHECK_URL}"
  wget -s "$HEALTH_URL" &> /dev/null

## Sample code using CURL with http code
#HTTP_CODE=`curl -m 10 -sL -w "%{http_code}" "${HEALTH_URL}" -o /dev/null`
#if [[ "$HTTP_CODE" -ge 200 && "$HTTP_CODE" -le 399 ]]; then
#  exit 0
#else
#  exit 1
#fi

  exit $?
fi

sleep 0.5 | telnet localhost $HTTP_PORT &> /dev/null
exit $?
