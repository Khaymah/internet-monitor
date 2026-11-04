FROM oven/bun:1.3.14 AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=docker
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION}" -o /internet-monitor ./cmd/headless

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /internet-monitor /usr/local/bin/internet-monitor
VOLUME /data
EXPOSE 8765
ENV IM_DIR=/data IM_LISTEN=0.0.0.0
ENTRYPOINT ["/usr/local/bin/internet-monitor"]
