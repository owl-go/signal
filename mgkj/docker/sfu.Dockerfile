FROM alpine:3.9.5

COPY ./bin/sfu /usr/bin/sfu

ENTRYPOINT ["/usr/bin/sfu"]
