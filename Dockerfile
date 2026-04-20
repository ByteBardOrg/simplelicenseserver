FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/simple-license-server ./cmd/server

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /out/simple-license-server /app/simple-license-server

EXPOSE 8080

ENTRYPOINT ["/app/simple-license-server"]
