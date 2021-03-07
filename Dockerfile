############################
# STEP 1 build executable binary
############################
FROM golang:1.16-alpine AS builder

WORKDIR /go/src/app

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -extldflags -static"

############################
# STEP 2 build a small image
############################
FROM alpine:3.12

LABEL org.opencontainers.image.source="https://github.com/verbit/restvirt-csi"

RUN apk add --no-cache e2fsprogs util-linux

COPY --from=builder /go/src/app/restvirt-csi restvirt-csi

ENTRYPOINT ["./restvirt-csi"]
