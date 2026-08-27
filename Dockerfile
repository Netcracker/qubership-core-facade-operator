FROM --platform=$BUILDPLATFORM golang:1.27@sha256:0ecdc2a9f6156af6451080bfe3d8382a662fcc4e209608c6f919e643453514c1 AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:2.4.1@sha256:c668333b6b03d897bfea3a7345bcff14a3b9224fffe62024202b2a125a6b0171 AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

CMD ["/app/facade-operator-service"]