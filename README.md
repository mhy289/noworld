# Noworld Backend

个人展示网站（MyWorld）的后端服务，基于 Go 标准库 `net/http` 构建，无第三方依赖。前后端分离架构，本仓库为后端部分。

## 功能

- **B站视频代理**：代理 B站 用户视频接口，供前端嵌入播放（UA 池 / 限流 / Wbi 参数，绕过跨域与风控）
- **投票系统**：1-10 数字投票统计（内存存储，并发安全）
- **健康检查**：`/api/health` 服务探活
- **对外公开接口**：`/public/*` 只读开放数据接口，与前端 `/api/*` 区分
- **CORS 支持**：内置跨域中间件，支持前后端分离部署

## 技术栈

- Go 1.22+（标准库 `net/http`，无第三方依赖）
- 目录按职责分层：`main.go`（入口）+ `internal/`（middleware / handler / service）

## 架构（前后端分离）

```
浏览器 ── axios ──> /api/* ──> Go 后端 (端口 8080)
  │                    │
  └── 开发环境: 前端 Vite dev server 通过 proxy 转发 /api 到本地 Go 后端
  └── 生产环境: 可同源（Go 托管静态文件）或跨域（后端已内置 CORS）
```

前端对接方式（前端仓库 `src/api/`）：

- `src/api/request.js` — axios 实例（baseURL 默认 `/api`，可用 `VITE_API_BASE_URL` 覆盖）+ 拦截器
- `src/api/index.js` — 业务 API（健康检查 / 投票 / B站代理）

## 快速开始

```bash
# 启动后端（默认端口 8080，仓库根目录即 Go module）
go run main.go

# 自定义端口
PORT=9090 go run main.go

# 编译
go build -o myworld-backend .
```

前端开发服务器需在 `vite.config.js` 将 `/api` 代理到本后端地址（默认 `http://localhost:8080`）。

## 后端接口

### 前端接口（`/api/*`）

面向网站前端 UI，供浏览器页面调用（含写操作）。

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| GET | `/api/health` | 健康检查 |
| POST | `/api/vote` | 投票 `{ "option": <任意值> }`（内存存储） |
| GET | `/api/votes` | 获取投票数据 |
| GET | `/api/bilibili/user/videos?mid=xxx` | B站用户视频代理（UA 池 / 限流 / Wbi 参数） |

### 对外公开接口（`/public/*`）

供第三方/外部系统调用的**只读**开放数据接口，与前端接口在路由前缀、能力范围上明确区分：仅提供查询能力，不含任何写操作。

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| GET | `/public/health` | 对外服务状态（精简信息，不暴露内部细节） |
| GET | `/public/stats/votes` | 对外投票统计（只读） |
| GET | `/public/bilibili/videos?mid=xxx` | 对外 B站视频数据（只读，复用代理逻辑） |

## 项目结构

```
（仓库根目录即 Go module）
├── go.mod                      # Go 模块定义
├── main.go                     # 入口：启动服务器、注册路由
├── start-server.bat            # Windows 启动脚本
└── internal/
    ├── middleware/
    │   └── cors.go             # CORS 跨域中间件
    ├── handler/                # HTTP 接口处理器层
    │   ├── health.go           # GET /api/health
    │   ├── vote.go             # POST /api/vote, GET /api/votes
    │   ├── bilibili.go         # GET /api/bilibili/user/videos
    │   └── public.go           # 对外公开接口 /public/*（健康 / 投票统计 / B站视频）
    └── service/                # 业务逻辑层
        ├── vote.go             # 投票内存存储（并发安全）
        └── bilibili.go         # B站代理（UA池 / 限流 / Wbi 参数）
    └── store/                  # 数据库访问层
        ├── db.go               # MySQL 连接池 + 自动执行 SQL 迁移
        └── migrations/         # SQL 迁移脚本（服务启动时自动按序执行）
            └── 001_init.sql    # 初始化建表
```

## 部署

建议由前端仓库的 GitHub Actions（`.github/workflows/deploy.yml`）统一部署：

1. 构建前端 `dist/` 并通过 SSH 部署到服务器
2. 交叉编译 Go 后端（linux/amd64）并部署、重启

需要配置 secrets：

- `SSH_PRIVATE_KEY` / `SSH_HOST` / `SSH_USERNAME`
- `DEPLOY_DIR`：前端静态文件目录（如 `/var/www/myworld`）
- `BACKEND_DIR`：后端二进制目录（如 `/opt/myworld/backend`）

## 许可

MIT
