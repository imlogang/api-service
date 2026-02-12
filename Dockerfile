FROM scratch
COPY api-service /api-service
EXPOSE 8080
ENTRYPOINT ["/api-service"]