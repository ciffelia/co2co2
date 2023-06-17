# syntax=docker/dockerfile:1.5
FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG TARGETPLATFORM
RUN <<eot
    set -eux

    case $TARGETPLATFORM in
      linux/amd64  ) export GOOS=linux GOARCH=amd64;;
      linux/arm/v7 ) export GOOS=linux GOARCH=arm GOARM=7;;
      linux/arm64  ) export GOOS=linux GOARCH=arm64;;
      *            ) echo "unsupported platform $TARGETPLATFORM"; exit 1;;
    esac

    CGO_ENABLED=0 go build -o ./out/ ./...
eot

FROM gcr.io/distroless/static-debian11:latest

COPY --from=builder /build/out/co2co2 /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/co2co2"]
