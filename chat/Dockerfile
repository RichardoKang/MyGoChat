FROM golang:1.24 AS builder
LABEL authors="conway"

WORKDIR /app

# 复制共享模块
COPY pkg/ ./pkg/

# 复制 chat 模块
COPY chat/ ./chat/

# 不使用 go.work，使用 go.mod 的 replace 指令
WORKDIR /app/chat
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o chat ./cmd

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/chat/chat .
COPY --from=builder /app/chat/configs/config.docker.yaml ./configs/config.docker.yaml
EXPOSE 8080

ENV CONFIG_PATH=/app/configs/config.docker.yaml
ENTRYPOINT ["./chat"]
