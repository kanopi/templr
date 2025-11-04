FROM debian:bookworm-slim

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/templr /usr/local/bin/templr

RUN set +x /usr/local/bin/templr

ENTRYPOINT [ "/usr/local/bin/templr" ]