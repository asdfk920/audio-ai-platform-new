# 构建阶段
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git make

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建用户服务
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./services/user

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/user-service .

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8001

CMD ["./user-service"]
