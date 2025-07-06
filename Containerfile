FROM golang:1.23-alpine AS builder
RUN apk add --no-cache build-base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-w -s" -o /app/main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main /app/main
COPY templates/ ./templates/
COPY static/ ./static/
COPY extratos.db .
EXPOSE 8080
CMD ["/app/main"]