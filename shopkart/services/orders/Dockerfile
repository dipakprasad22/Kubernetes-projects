# Build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY go.* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server .

# Runtime stage: distroless, non-root, minimal attack surface
FROM gcr.io/distroless/static-debian12 AS runtime
COPY --from=build /app/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
