FROM scratch
COPY api-service /api-service
ENTRYPOINT ["/api-service"]