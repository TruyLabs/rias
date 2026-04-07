FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/norenis/kai.Version=${VERSION} -X github.com/norenis/kai.Commit=${COMMIT}" \
    -o /kai ./cmd/kai

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /kai /usr/local/bin/kai

# Brain and sessions are stored in ~/.kai (i.e. /root/.kai in this container).
# Mount volumes here to persist data across container restarts.
VOLUME ["/root/.kai/brain", "/root/.kai/sessions"]

WORKDIR /root/.kai

ENTRYPOINT ["kai"]
