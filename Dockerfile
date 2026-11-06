FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /out/api ./cmd/api
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libaio1 libnsl2 && rm -rf /var/lib/apt/lists/*
ENV LD_LIBRARY_PATH=/opt/oracle/instantclient
WORKDIR /app
COPY --from=builder /out/api /app/api
USER 65532:65532
EXPOSE 9042
ENTRYPOINT ["/app/api"]
