FROM golang:1.27.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mimir ./cmd/mimir

FROM alpine:3.22
RUN apk add --no-cache su-exec \
    && addgroup -S -g 10001 mimir \
    && adduser -S -D -H -u 10001 -G mimir mimir \
    && mkdir -p /var/lib/mimir /data/sources \
    && chown -R mimir:mimir /var/lib/mimir
COPY --from=build /out/mimir /usr/local/bin/mimir
COPY deploy/backend/entrypoint.sh /usr/local/bin/entrypoint
RUN chmod 0555 /usr/local/bin/entrypoint
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/entrypoint"]
