
FROM golang:1.25-bookworm AS builder

WORKDIR /src

ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -o /out/existencias-puerto ./cmd/api


FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        libaio1 \
        libnsl2 && \
    rm -rf /var/lib/apt/lists/*

ENV ORACLE_HOME=/opt/oracle/instantclient_21_12
ENV LD_LIBRARY_PATH=/opt/oracle/instantclient_21_12
ENV PATH="${PATH}:/opt/oracle/instantclient_21_12"

COPY --from=builder /out/existencias-puerto /usr/local/bin/existencias-puerto

EXPOSE 9042

ENTRYPOINT ["/usr/local/bin/existencias-puerto"]
