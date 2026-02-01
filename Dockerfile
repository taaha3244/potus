FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY potus /usr/local/bin/potus

RUN chmod +x /usr/local/bin/potus

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/potus"]
