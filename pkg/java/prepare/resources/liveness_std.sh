#!/usr/bin/env bash

# Dette er et placeholder script for liveness probem. Det eneste den gjør er å returnere 0.
# Før et prosjekt bestemmer seg for å implementere et liveness script er det viktig å forstå
# forskjellen mellom readiness. For mer informasjon se http://kubernetes.io/docs/user-guide/pod-states/
# Eks. DC config
# - containers:
# .....
#     livenessProbe:
#       exec:
#         command: [/u01/application/bin/liveness.sh]
#       initialDelaySeconds: 15
#       timeoutSeconds: 2

exit 0
