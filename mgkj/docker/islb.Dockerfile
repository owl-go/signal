FROM alpine:3.9.5

COPY ./bin/islb /usr/bin/islb

ENTRYPOINT ["/usr/bin/islb"]
