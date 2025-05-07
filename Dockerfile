FROM ghcr.io/netcracker/qubership/core-base:main-20250331175012-25

EXPOSE 8080 15010

COPY --chown=10001:0 facade-operator-service/bin/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 ["facade-operator-service/application.yaml", "facade-operator-service/policies.conf", "/app/"]