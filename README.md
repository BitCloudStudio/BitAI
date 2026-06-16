# BitAPI

中文 AI API Gateway — 统一管理和分发 AI 模型接口，支持多上游账号、用户配额、计费支付、品牌定制。

后端 Go · Gin · GORM · SQLite ｜ 前端 Vue 3 · TypeScript · Arco Design

## 功能

- **OpenAI 兼容接口** — `/v1/chat/completions`、`/v1/responses`、`/v1/models`，任何 OpenAI SDK 直接接入
- **多上游管理** — 配置多个 API 账号，按分组分配模型，负载均衡、故障切换
- **用户体系** — 注册/登录、API Key 管理、用量统计、充值兑换
- **计费 & 支付** — 用量计费、余额扣除、多支付通道
- **管理后台** — 用户管理、分组配置、订单审核、兑换码、数据统计
- **品牌定制** — 站点名称、Logo（浅色/深色/侧边栏）、版权信息全部可换
- **安全** — 无默认密码，首次部署强制手动初始化；JWT 鉴权、API Key 认证

## 快速开始

### 方式一：Docker（推荐）

```bash
# 1. 初始化（生成密钥和配置）
bash docker-init.sh

# 2. 启动
docker compose up -d --build

# 3. 访问
# http://localhost:8091/auth/login
```

数据库和上传文件通过 Docker volume 持久化，删除容器不会丢数据。配置文件在 `.env`，按需修改端口、管理员信息等。

### 方式二：手动部署

**要求：** Go ≥1.22、Node.js ≥20、npm、Nginx

```bash
# 1. 上传整个项目目录到服务器
# 2. 进入目录
cd /path/to/BitAPI

# 3. 赋权
chmod +x 一键初始化.sh 一键运行.sh scripts/init.sh scripts/run.sh

# 4. 初始化（会提示填写站点名、管理员账号/密码等）
bash 一键初始化.sh

# 5. 启动
bash 一键运行.sh
```

启动后服务监听 `:8091`，前端静态文件 + API 都在同端口。日志在 `logs/` 目录。

## 访问地址

| 地址 | 说明 |
|---|---|
| `/` | 前台首页 |
| `/auth/login` | 管理后台登录 |
| `/health` | 健康检查 |
| `/v1/chat/completions` | OpenAI 兼容接口 |
| `/v1/models` | 模型列表 |

网关 base URL 直接填域名或 IP + 端口即可。

## 品牌定制

登录后台 → `系统设置 → 产品品牌配置`：

- 站点名称
- 浅色 Logo（导航栏、登录页、控制台）
- 深色 Logo（底部栏等深色区域）
- 侧边栏收缩 Logo（建议方形图标）
- 版权信息

Logo 上传后保存到 `/uploads/branding/`，Nginx 需保留 `/uploads/` 转发。

## 常见问题

**页面空白 / JS / CSS 403**
- 检查宝塔是否开了整站反代（删除或注释 `proxy/*.conf`）
- 防盗链放行 `/bitapi-assets/`，WAF 临时关闭测试
- CORS 配置：`BITAPI_CORS_ORIGINS` 需包含你的域名

**初始化后登录失败**
- 确认管理员邮箱和密码是否与初始化时一致
- 查看 `logs/init-backend.err.log`
- 确认 `backend/data/bitapi.db` 已生成

**反向代理慢**
- 不要整站反代。Nginx 直出 `frontend/dist`，Go 只处理 API
- 确认代理目标是 `127.0.0.1:8091`，不是 `:5181`

**重新初始化**
```bash
bash 一键初始化.sh  # 会清空数据库
```

## 技术架构

```
请求 → Nginx (HTTPS + 静态文件) → Go Backend (:8091)
        静态文件由 frontend/dist 直出                ├─ /api/*  管理 API
        API 请求反代到 Go 后端                       ├─ /v1/*   OpenAI 兼容接口
                                                     ├─ /uploads 文件服务
                                                     └─ SQLite 数据存储
```

后端目录：

```
backend/
├── cmd/server/main.go      入口
└── internal/
    ├── config/             环境变量
    ├── db/                 SQLite 初始化与迁移
    ├── http/               路由、Handler、中间件、响应封装
    ├── models/             GORM 数据模型
    ├── pkg/crypto/         密码哈希
    └── services/
        ├── adapters/       AI 厂商适配器 (OpenAI)
        ├── auth/           认证与 JWT
        ├── billing/        计费
        ├── gateway/        核心转发
        ├── keys/           API Key 管理
        ├── monitor/        统计监控
        ├── payments/       支付通道
        └── ratelimit/      限流
```

## Nginx 配置参考

**Docker 部署无需 Nginx**，后端已内置静态文件服务。以下仅供手动部署 + 宝塔面板参考。

完整示例见 [`deploy/nginx-bitapi.conf`](deploy/nginx-bitapi.conf)。

核心规则：
- 网站根目录设为 `frontend/dist`
- 只反代以下路径到 `127.0.0.1:8091`：`/api/`、`/v1/`、`/responses`、`/uploads/`、`/health`
- 不要开整站反代
- 防盗链放行 `/bitapi-assets/`