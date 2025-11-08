FROM golang:1.25.3-bookworm AS builder
LABEL user=weicai

# 国内配置go源
ENV GO111MODULE=on
ENV GOPROXY='https://goproxy.cn,direct'

WORKDIR /app
RUN apt update && apt install file -y

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -linkmode external -extldflags '-static'" \
    -o main .

RUN file main && readelf -d /app/main | grep NEEDED || echo "静态编译成功"

FROM gcr.io/distroless/static:latest
WORKDIR /app
# 复制 CA 证书（用于 HTTPS 请求）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 复制时区数据（如果需要）
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 复制可执行文件
COPY --from=builder /app/main /app/main

# 环境变量
ENV TZ=UTC
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

COPY --from=builder /app/main /app/main
ENTRYPOINT ["/app/main"]