############################
# STEP 1 build executable binary
############################
FROM golang:1.15-alpine AS builder

WORKDIR /go/src/app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -extldflags -static"

############################
# STEP 2 build a small image
############################
FROM alpine:3.12

RUN apk add --no-cache e2fsprogs util-linux

COPY --from=builder /go/src/app/restvirt-csi restvirt-csi

ENTRYPOINT ["./restvirt-csi"]