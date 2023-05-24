FROM golang:1.20-alpine AS builder

WORKDIR /app

ADD go.mod go.sum ./
RUN go mod download

ADD . ./
RUN go build -o backend

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /app/backend /backend

HEALTHCHECK --interval=15s --timeout=3s CMD curl -f http://localhost:8080/v1/ || exit 1
CMD ["/backend"]
