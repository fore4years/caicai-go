# ---------- 构建阶段 ----------
FROM golang:1.27.1-alpine AS builder

# 设置工作目录
WORKDIR /app

# 先拷贝依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./

# 配置国内代理（你之前拉包卡过，这步很关键）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

# 下载依赖
RUN go mod download

# 拷贝源码
COPY . .

# 编译（关键参数看下面说明）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o caicai-go .

# ---------- 运行阶段 ----------
FROM alpine:latest

# 时区（日志时间会用到）
RUN apk add --no-cache tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 从构建阶段拷贝二进制
COPY --from=builder /app/caicai-go .

# 如果你的 conf.yaml 想打进镜像（不推荐，但简单）：
# COPY --from=builder /app/conf.yaml .
ENV GIN_MODE=release

# 暴露端口（跟你配置里的 6666 一致）
EXPOSE 6666

# 启动
CMD ["./caicai-go"]