FROM alpine:3.9.5

COPY ./bin/biz /usr/bin/biz

ENTRYPOINT ["/usr/bin/biz"]
