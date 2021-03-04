FROM alpine:3.9.5

COPY ./bin/dist /usr/bin/dist

ENTRYPOINT ["/usr/bin/dist"]
