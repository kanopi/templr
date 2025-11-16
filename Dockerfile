# Ultra-minimal Docker image using scratch base
# Final image size: ~9.5MB (just the binary + ca-certificates)
FROM alpine:3.22 AS certs

# Get ca-certificates
RUN apk add --no-cache ca-certificates

# Final stage - scratch (empty) base
FROM scratch

# Use the targetplatform to copy the file from
ARG TARGETPLATFORM

# Copy ca-certificates from alpine
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the statically-linked binary from GoReleaser
COPY $TARGETPLATFORM/templr /usr/local/bin/templr

# Note: No shell, no package manager, just the binary
# Cannot use USER directive with scratch (no /etc/passwd)

WORKDIR /work

ENTRYPOINT ["/usr/local/bin/templr"]
CMD ["--help"]
