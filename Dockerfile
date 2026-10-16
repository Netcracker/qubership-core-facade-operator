FROM --platform=$BUILDPLATFORM golang:1.27.2@sha256:820153096fda4de6ac1b7194d12e416b7dd8f4e9a33d17e0345b111b945162d8 AS build

WORKDIR /app

COPY facade-operator-service/ .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o facade-operator-service .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.7@sha256:f0836415633bc2a63c2a0431c4c0701d17bd7b31132176b26651b49927d9ad7e AS run

EXPOSE 8080 15010

COPY --chown=10001:0 --chmod=555 --from=build app/facade-operator-service /app/facade-operator-service
COPY --chown=10001:0 --chmod=444 --from=build app/application.yaml /app/

WORKDIR /app

CMD ["/app/facade-operator-service"]