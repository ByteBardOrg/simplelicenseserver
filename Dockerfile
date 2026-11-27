# Build with a current Go release: standard library security fixes only ship in new Go versions.
FROM golang:1.27-alpine AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/simple-license-server /app/simple-license-server
COPY --from=ui-builder /ui/dist /app/ui

EXPOSE 8080

# Run as the unprivileged "nonroot" user (uid 65532) from the base image, not root.
USER nonroot:nonroot

ENTRYPOINT ["/app/simple-license-server"]
