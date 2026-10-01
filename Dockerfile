FROM --platform=$BUILDPLATFORM golang:1.27.1@sha256:e0174e51e81218523251d85d248a90d24c3d5e81543b4f07a5d66229397db190 AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.5@sha256:1e207c43e907165f9c55838bb853465357a28bfd50799cf4aa66b959fe72a836 AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

CMD ["/app/facade-operator-service"]