# Build stage
FROM us-docker.pkg.dev/opsramp-registry/base/golang:1.24.13-alpine3.23 AS builder
WORKDIR /app
COPY . .
RUN cd cmd/otelcorecol && CGO_ENABLED=0 go build -trimpath -o ../../bin/otelcorecol .



# Final image
FROM us-docker.pkg.dev/opsramp-registry/base/alpine:3.23.3
COPY --from=builder /app/bin/otelcorecol ./otelcollector
CMD ["./otelcollector"]

