FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/templr-linux-${TARGETARCH} /usr/local/bin/templr

RUN set +x /usr/local/bin/templr

ENTRYPOINT [ "/usr/local/bin/templr" ]