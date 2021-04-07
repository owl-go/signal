FROM alpine:3.9.5

COPY ./bin/logsvr /usr/bin/logsvr

ENTRYPOINT ["/usr/bin/logsvr"]
