FROM --platform=$BUILDPLATFORM golang:1.27.1@sha256:3680233e3204827fbdc66088528ae6d4b3d034f51d03a99d454f6de034888244 AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.6@sha256:e85e971e33cdb218bf1c23ccbad0d0ddaee7d3abfc442247d81081e3d0467467 AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

CMD ["/app/facade-operator-service"]