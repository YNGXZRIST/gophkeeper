# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache module downloads separately from the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary so it runs on a minimal base image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.20

# Run as a non-root user.
RUN adduser -D -u 10001 app

COPY --from=builder /out/server /usr/local/bin/server

USER app
EXPOSE 8080

ENTRYPOINT ["server"]
