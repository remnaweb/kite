FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web ./
RUN npm run build

FROM golang:1.23-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./internal/webui/dist
RUN CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o /panel ./cmd/panel

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates unzip curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /panel /usr/local/bin/panel
RUN mkdir -p /app/bin /app/data
ARG XRAY_VERSION=v25.8.3
RUN curl -fsSL -o /tmp/xray.zip https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-64.zip \
 && unzip /tmp/xray.zip xray -d /app/bin \
 && chmod +x /app/bin/xray \
 && rm /tmp/xray.zip
ENV PANEL_LISTEN=0.0.0.0:2053 PANEL_DATA=/app/data XRAY_BIN=/app/bin/xray
VOLUME ["/app/data"]
EXPOSE 2053
CMD ["panel"]
