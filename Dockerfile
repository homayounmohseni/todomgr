FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/todomgr .

FROM alpine:3.21
RUN addgroup -S app && adduser -S app -G app && apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /app/todomgr /todomgr
EXPOSE 8080
USER app
ENTRYPOINT ["/todomgr"]
