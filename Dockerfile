# syntax=docker/dockerfile:1

# Use official MTG image as source for the binary
FROM nineseconds/mtg:2 as mtg

# Final runtime image  
FROM gcr.io/distroless/static:latest

# Copy official MTG binary
COPY --from=mtg /mtg /usr/local/bin/mtg

# Run as non-root user
USER 65534:65534

EXPOSE 3128

ENTRYPOINT ["/usr/local/bin/mtg"] 