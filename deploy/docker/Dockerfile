# Stage 1: Build the ShipSafe binary
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build static binary with version injection
COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/toyinlola/shipsafe/cmd.Version=${VERSION} \
      -X github.com/toyinlola/shipsafe/cmd.Commit=${COMMIT} \
      -X github.com/toyinlola/shipsafe/cmd.BuildDate=${BUILD_DATE}" \
    -o /shipsafe .

# Stage 2: Minimal runtime image
FROM alpine:3.20

# git is required for `shipsafe scan` on repositories
RUN apk add --no-cache git ca-certificates \
    && addgroup -S shipsafe \
    && adduser -S -G shipsafe shipsafe

COPY --from=builder /shipsafe /usr/local/bin/shipsafe

USER shipsafe
WORKDIR /workspace

ENTRYPOINT ["shipsafe"]
