# VANTIQOR single-container deployment.
# The frontend is built and served by the Go API process.
FROM node:22-alpine AS web-build
WORKDIR /app/web
COPY web/package.json web/bun.lock* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS go-build
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -trimpath -o /out/vantiqor ./cmd/clawtrade

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-build /out/vantiqor ./vantiqor
COPY --from=go-build /app/config ./config
COPY --from=go-build /app/web/dist ./web/dist
ENV PORT=8080
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["./vantiqor", "serve"]
