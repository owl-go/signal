FROM alpine:3.9.5

COPY ./bin/issr /usr/bin/issr

ENTRYPOINT ["/usr/bin/issr"]
