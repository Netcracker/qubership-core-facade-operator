FROM golang:1.25 AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:2.2.1 AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

USER 10001:10001

CMD ["/app/facade-operator-service"]