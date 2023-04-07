FROM golang:1.20-alpine AS builder

WORKDIR /app

ADD go.mod go.sum ./
RUN go mod download

ADD . ./
RUN go build -o backend

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/backend /backend
CMD ["/backend"]
