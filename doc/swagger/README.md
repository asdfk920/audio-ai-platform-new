# Swagger / OpenAPI 文档生成

本文档支持两类 Swagger 生成方式：

- **微服务（user/device/content）**：go-zero **`goctl api swagger`** 从 **`*.api`** 生成 **Swagger 2.0 JSON**
- **go-admin 后台**：swaggo **`swag init`** 从 Go 注解生成 **Swagger JSON/YAML**

## 前置条件

- 已安装 **Go**（用于 `go run` 拉取 goctl）
- 可选：安装全局 goctl（`make goctl-install` 或 `go install github.com/zeromicro/go-zero/tools/goctl@latest`）
- 可选：安装 swag（`go install github.com/swaggo/swag/cmd/swag@latest`）

## 一键生成全部 API 文档

在**仓库根目录**执行：

**Linux / macOS（有 Make）**

```bash
make swagger
```

**Windows（PowerShell，无需 Make）**

```powershell
.\scripts\gen-swagger.ps1
```

**仅生成某一个服务**

```bash
make swagger-user
make swagger-device
make swagger-content
make swagger-admin
```

或手动：

```bash
go run github.com/zeromicro/go-zero/tools/goctl@latest api swagger ^
  --api services/user/user.api --dir doc/swagger/user --filename user
```

（Linux/macOS 将 `^` 换为行末 `\`。）

### 生成 YAML

在对应命令中增加 `--yaml`，并自行指定输出目录与文件名习惯（goctl 会生成 `.yaml`）。

## 输出位置

| 服务   | 源文件                      | 生成物                    |
|--------|-----------------------------|---------------------------|
| user   | `services/user/user.api`    | `doc/swagger/user/user.json`    |
| device | `services/device/device.api`| `doc/swagger/device/device.json` |
| content| `services/content/content.api` | `doc/swagger/content/content.json` |
| admin | `admin`（swag 注解） | `doc/swagger/admin/admin.json` |

## 在浏览器访问 Swagger UI（本仓库）

1. 安装 **Node.js**。
2. 在**仓库根目录**执行：

```powershell
.\scripts\serve-swagger-ui.ps1
```

已生成好 `doc/swagger/**/*.json`、只想起服务时加 **`-SkipGen`**。

3. 在浏览器打开（默认端口 **8090**）：

| 文档 | URL |
|------|-----|
| user（默认） | http://127.0.0.1:8090/ 或 http://127.0.0.1:8090/?spec=user |
| device | http://127.0.0.1:8090/?spec=device |
| content | http://127.0.0.1:8090/?spec=content |
| admin | http://127.0.0.1:8090/?spec=admin |

**ReDoc 单页**（离线查看）：双击 `doc/swagger/export/*-api.html`，或见下文「导出 ReDoc」。

**说明**：这是**文档静态站**，与 user/device 等业务服务是否启动无关；在 Swagger UI 里「Try it out」能否打通，取决于你填的服务器地址与 CORS。

## 如何查看 / 调试

1. **Swagger Editor（在线）**  
   打开 [editor.swagger.io](https://editor.swagger.io/)，将对应 `*.json` 内容粘贴或 **File → Import file**。

2. **Apifox / Postman / Apipost**  
   选择「导入」→ OpenAPI / Swagger → 选中 `doc/swagger/**/*.json`。

3. **本机 Swagger UI（Docker 示例）**

   ```bash
   docker run --rm -p 8080:8080 -e SWAGGER_JSON=/spec/user.json \
     -v "$(pwd)/doc/swagger/user/user.json:/spec/user.json" \
     swaggerapi/swagger-ui
   ```

   浏览器访问 `http://localhost:8080`。

4. **VS Code**  
   安装 OpenAPI / Swagger 类扩展，直接打开 `user.json` 预览。

5. **本机 Swagger UI（本仓库内置静态页）**

Windows（PowerShell）：

```powershell
.\scripts\serve-swagger-ui.ps1
```

打开后可用参数切换：

- `?spec=user|device|content|admin`

6. **导出 ReDoc 单页 HTML / PDF（便于交付、打印、归档）**

先保证已生成 `doc/swagger/**/*.json`（见上文「一键生成」）。在**仓库根目录**执行：

**Windows（PowerShell，推荐）**

```powershell
.\scripts\export-swagger-redoc.ps1
```

默认会先执行 `gen-swagger.ps1` 再导出。常用参数：

- `-SkipGen`：跳过重新生成 JSON，仅用当前 `doc/swagger` 下的文件
- `-HtmlOnly`：只生成 `.html`
- `-PdfOnly`：只生成 `.pdf`

**与 redoc-cli 命令等价示例（单份 user）**

```bash
npx --yes redoc-cli bundle doc/swagger/user/user.json -o doc/swagger/export/user-api.html
npx --yes redoc-cli bundle doc/swagger/user/user.json -o doc/swagger/export/user-api.pdf
```

**输出目录**：`doc/swagger/export/`

| 文件 | 说明 |
|------|------|
| `user-api.html` / `.pdf` | 用户服务 API |
| `device-api.html` / `.pdf` | 设备服务 API |
| `content-api.html` / `.pdf` | 内容服务 API |
| `admin-api.html` / `.pdf` | 管理后台 API |

**Linux / macOS（有 Make + Node）**

```bash
make swagger-export
```

会先执行 `make swagger` 与 `make swagger-admin`（admin 需本机已装 `swag`），再导出八个文件到 `doc/swagger/export/`。

说明：`redoc-cli` 会通过 `npx` 临时拉取，一般**无需**全局 `npm install -g redoc-cli`。工具提示 deprecated 时，可后续改用 `npx @redocly/cli build-docs`（命令与参数不同，需单独查阅 Redocly 文档）。

## 维护说明

- 新增或修改 HTTP 接口时，优先改 **`services/*/xxx.api`**，再执行 **`make swagger`** 或 **`scripts/gen-swagger.ps1`** 更新文档。
- 修改 go-admin 接口注解后，执行 `make swagger-admin`（或 Windows 直接跑 `scripts/gen-swagger.ps1`）更新 `doc/swagger/admin/admin.json`。
- `@doc` 注释会进入生成的 `summary`/`description`，建议在 `.api` 里写清楚。
