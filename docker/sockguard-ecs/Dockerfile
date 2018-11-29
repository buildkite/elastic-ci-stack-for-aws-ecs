FROM alpine:latest

RUN apk add --update jq curl && rm -rf /var/cache/apk/*

COPY ./sockguard /usr/bin/sockguard
COPY ./sockguard.sh /usr/bin/sockguard.sh

CMD ["/usr/bin/sockguard.sh"]
