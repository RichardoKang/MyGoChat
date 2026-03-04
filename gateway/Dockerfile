FROM golang:1.24 AS builder
LABEL authors="conway"

WORKDIR /app

# 复制共享模块
COPY pkg/ ./pkg/

# 复制 gateway 模块
COPY gateway/ ./gateway/

# 不使用 go.work，使用 go.mod 的 replace 指令
WORKDIR /app/gateway
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway/gateway .
COPY --from=builder /app/gateway/configs/config.docker.yaml ./configs/config.docker.yaml
EXPOSE 8081

ENV CONFIG_PATH=/app/configs/config.docker.yaml
ENV GATEWAY_ID=gateway-01
ENTRYPOINT ["./gateway"]
