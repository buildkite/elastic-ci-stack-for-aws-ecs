FROM buildkite/agent:3

RUN apk -v --update add \
        python \
        py-pip \
        groff \
        less \
        mailcap \
        && \
    pip install --upgrade awscli s3cmd==2.0.1 python-magic && \
    apk -v --purge del py-pip && \
    rm /var/cache/apk/*

RUN curl -Lfs -o /usr/bin/sockguard \
    https://github.com/buildkite/sockguard/releases/download/v1.0.0/sockguard-linux-amd64

ADD bootstrap.sh /buildkite/bootstrap.sh
