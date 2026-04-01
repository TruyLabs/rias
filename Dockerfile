FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/tinhvqbk/kai.Version=${VERSION} -X github.com/tinhvqbk/kai.Commit=${COMMIT}" \
    -o /kai ./cmd/kai

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /kai /usr/local/bin/kai

WORKDIR /app

ENTRYPOINT ["kai"]
