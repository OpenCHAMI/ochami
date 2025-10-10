FROM cgr.dev/chainguard/wolfi-base

RUN apk add --no-cache tini

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/ochami /usr/bin/ochami

# Make dir for config file
RUN mkdir -p /.config/ochami
RUN chown -R 65534:65534 /.config

# nobody 65534:65534
USER 65534:65534

CMD [ "/usr/bin/ochami" ]
ENTRYPOINT [ "/sbin/tini", "--" ]
