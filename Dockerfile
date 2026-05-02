FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/simple-license-server ./cmd/server

FROM node:22-alpine AS ui-builder

WORKDIR /ui

COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui ./
RUN npm run build

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /out/simple-license-server /app/simple-license-server
COPY --from=ui-builder /ui/dist /app/ui

EXPOSE 8080

ENTRYPOINT ["/app/simple-license-server"]
