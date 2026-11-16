FROM --platform=$BUILDPLATFORM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.13@sha256:891386ea91d90a1376e12f30b458fc23eb9793afd35f82817c90e4911cddd633 AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

CMD ["/app/facade-operator-service"]