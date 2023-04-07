FROM golang:1.20 AS builder

WORKDIR /app

ADD go.mod go.sum ./
RUN go mod download

ADD . ./
RUN CGO_ENABLED=0 go build -o backend

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/backend /backend
CMD ["/backend"]
