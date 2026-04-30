# Go 1.22 全栈依赖锁（仓库约定）

## 语言与工具链

- 所有 **`go.mod`**：`go 1.22` + **`toolchain go1.22.6`**
- 本地 / CI：`export GOTOOLCHAIN=go1.22.6`（PowerShell：`$env:GOTOOLCHAIN='go1.22.6'`）

## 微服务（go-zero）

- **go-zero**：**v1.9.3**（勿升到 v1.10+，其依赖链易要求 Go 1.23+）
- **根目录 + `services/user`**：在 `replace` 中锁定 `grpc` / `protobuf` / `x/crypto` / `redis` / 常用 `golang.org/x/*`

## Admin（go-admin）

- **go-admin-core / sdk**：**v1.5.2**
- **gin**：**v1.9.1**；**swag / gin-swagger**：**v1.16.2 / v1.5.0**（与生成文档匹配）
- **sentinel-golang**：**v1.0.4**；`pkg/adapters/gin` 使用代理可用的伪版本
- SDK **v1.5.2** 获取本机 IP 的函数名为 **`GetLocaHonst`**（代码中已对接）

## 通用 `replace`（防 `go get -u` 抬版本）

各模块按需包含（admin 最全）：

```text
github.com/redis/go-redis/v9 => v9.3.0
golang.org/x/crypto => v0.17.0
golang.org/x/net => v0.23.0
golang.org/x/sync => v0.11.0
golang.org/x/sys => v0.30.0
golang.org/x/text => v0.14.0
google.golang.org/grpc => v1.58.3
google.golang.org/protobuf => v1.31.0
```

## 改依赖后自检

```bash
GOTOOLCHAIN=go1.22.6 go mod tidy
GOTOOLCHAIN=go1.22.6 go build ./...
```

分别在：仓库根、`services/*` 独立模块、`admin` 下执行。
