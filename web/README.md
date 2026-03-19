# Xray Web Panel - 内嵌式 Web 管理面板

Xray Web Panel 是 Xray-core 的内嵌式 Web 管理面板，编译后生成**单一二进制文件**，无需额外部署前端服务，开箱即用。

## 特性

- **单一二进制部署** - 前端通过 Go `embed.FS` 嵌入，一个文件搞定一切
- **完整的 gRPC 桥接** - 通过 REST API 代理 Xray Commander 的全部 gRPC 接口
- **双通道控制** - Runtime API 即时生效 + Config File 持久化修改
- **JWT 认证** - 登录限流防暴力破解，token 24 小时过期
- **实时监控** - WebSocket 推送流量速率和连接信息
- **分享链接** - 一键生成 VLESS/VMess/Trojan/Shadowsocks 分享 URI
- **配置管理** - JSON 编辑器 + 导入导出 + 自动备份/恢复
- **国际化** - 中文 / English 双语支持
- **暗色主题** - 支持亮色/暗色主题切换
- **响应式布局** - 适配桌面和移动端

---

## 快速开始

### 1. 构建

```bash
# 完整构建（前端 + Go）
cd web && npm install && npm run build && cd ..
cp -r web/dist app/webpanel/dist
go build -o xray ./main

# 或者使用 Makefile
make build
```

```bash
# 仅构建 Go（使用占位前端，用于开发调试）
make build-dev
```

### 2. 配置

在 Xray 配置文件中添加 `webpanel` 段：

```json
{
  "log": { ... },
  "api": {
    "tag": "api",
    "listen": "127.0.0.1:10085",
    "services": [
      "HandlerService",
      "StatsService",
      "LoggerService",
      "RoutingService",
      "ObservatoryService"
    ]
  },
  "stats": {},
  "policy": {
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true,
      "statsOutboundUplink": true,
      "statsOutboundDownlink": true
    }
  },
  "webpanel": {
    "listen": "127.0.0.1:9527",
    "apiEndpoint": "127.0.0.1:10085",
    "username": "admin",
    "password": "admin123",
    "jwtSecret": "请修改为随机字符串",
    "configPath": "/etc/xray/config.json"
  },
  "inbounds": [ ... ],
  "outbounds": [ ... ]
}
```

> **重要**：Web Panel 依赖 `api`、`stats`、`policy` 配置段正常工作。请确保 `apiEndpoint` 与 `api.listen` 指向同一地址。

### 3. 运行

```bash
./xray run -c /etc/xray/config.json
```

浏览器访问 `http://127.0.0.1:9527`，使用配置的用户名密码登录。

---

## 配置说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `listen` | string | `127.0.0.1:9527` | Web 面板监听地址 |
| `apiEndpoint` | string | `127.0.0.1:10085` | Xray gRPC API 地址（对应 `api.listen`） |
| `username` | string | `admin` | 登录用户名 |
| `password` | string | `admin123` | 登录密码 |
| `jwtSecret` | string | `xray-webpanel-secret` | JWT 签名密钥，**务必修改** |
| `configPath` | string | - | Xray 配置文件路径，用于配置编辑/备份功能 |
| `certFile` | string | - | TLS 证书文件路径（可选） |
| `keyFile` | string | - | TLS 私钥文件路径（可选） |

---

## 功能模块

### 仪表盘 (`/dashboard`)

- 系统信息：运行时间、Goroutine 数、内存占用、GC 次数
- 流量概览：总上传/下载
- 在线用户数
- 实时流量折线图（5 秒轮询）
- Top 10 用户流量排行

### 入站管理 (`/inbounds`)

- 入站列表：Tag、协议、操作
- 新增入站：选择协议 + JSON 配置
- 删除入站
- 支持协议：VLESS、VMess、Trojan、Shadowsocks、SOCKS、HTTP、Dokodemo-door、Hysteria、WireGuard

### 出站管理 (`/outbounds`)

- 出站列表与增删
- 支持协议：Freedom、Blackhole、VLESS、VMess、Trojan、Shadowsocks、SOCKS、HTTP、DNS、Loopback、WireGuard、Hysteria

### 用户管理 (`/users`)

- 跨入站聚合视图：Email、入站 Tag、等级、在线状态、上下行流量
- 添加/删除用户
- 生成分享链接（VLESS/VMess/Trojan/SS URI）
- 支持自定义传输方式、TLS/REALITY 等参数

### 路由规则 (`/routing`)

- **规则列表**：查看/添加/删除路由规则
- **均衡器**：查看均衡器信息，覆盖均衡器目标
- **路由测试**：输入域名/IP/端口等参数，测试路由匹配结果

### DNS 配置 (`/dns`)

- 查看 DNS 服务器列表
- 查看静态 Hosts 映射
- 显示查询策略等通用设置

### 实时监控 (`/monitor`)

- **流量监控**：WebSocket 实时速度折线图
- **连接监控**：WebSocket 推送活跃连接（入站、出站、目标、用户等）
- **系统监控**：Goroutine / 内存时间线图（2 秒轮询）

### 系统设置 (`/settings`)

- 日志配置查看 + 重启日志按钮
- 策略配置查看
- Observatory 出站健康状态表
- API 设置查看

### 配置管理 (`/config`)

- **原始编辑器**：直接编辑 JSON 配置
- **验证配置**：保存前校验 JSON 合法性
- **导入/导出**：JSON 文件上传/下载
- **备份/恢复**：自动时间戳备份（保留最近 20 个），一键恢复

---

## REST API

所有 API 均需 JWT 认证（`Authorization: Bearer <token>`），除登录和订阅端点外。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录，返回 JWT token |

### 统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sys/stats` | 系统状态（Goroutine、内存、GC 等） |
| GET | `/api/v1/stats/query?pattern=&reset=` | 查询流量统计 |
| GET | `/api/v1/stats/online-users` | 获取在线用户列表 |
| GET | `/api/v1/stats/online-ips?email=` | 获取用户在线 IP |

### 入站/出站

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/inbounds` | 列出所有入站 |
| POST | `/api/v1/inbounds` | 添加入站 |
| DELETE | `/api/v1/inbounds/:tag` | 删除入站 |
| GET | `/api/v1/inbounds/:tag/users` | 获取入站用户 |
| POST | `/api/v1/inbounds/:tag/users` | 添加用户到入站 |
| DELETE | `/api/v1/inbounds/:tag/users/:email` | 从入站删除用户 |
| GET | `/api/v1/outbounds` | 列出所有出站 |
| POST | `/api/v1/outbounds` | 添加出站 |
| DELETE | `/api/v1/outbounds/:tag` | 删除出站 |

### 用户

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/` | 列出所有用户（跨入站聚合） |
| DELETE | `/api/v1/users/:email` | 从所有入站删除用户 |

### 路由

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/routing/rules` | 列出路由规则 |
| POST | `/api/v1/routing/rules` | 添加路由规则 |
| DELETE | `/api/v1/routing/rules/:tag` | 删除路由规则 |
| POST | `/api/v1/routing/test` | 测试路由匹配 |
| GET | `/api/v1/routing/balancers/:tag` | 获取均衡器信息 |
| PUT | `/api/v1/routing/balancers/:tag` | 覆盖均衡器目标 |

### 其他

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/observatory/status` | 出站健康状态 |
| POST | `/api/v1/logger/restart` | 重启日志 |
| GET | `/api/v1/config` | 读取配置文件 |
| PUT | `/api/v1/config` | 保存配置文件 |
| POST | `/api/v1/config/reload` | 重载配置 |
| POST | `/api/v1/config/validate` | 校验配置 |
| GET/POST | `/api/v1/config/backups` | 备份管理 |
| POST | `/api/v1/share/generate` | 生成分享链接 |

### WebSocket

| 路径 | 说明 |
|------|------|
| `/api/v1/ws/routing-stats` | 连接流实时推送（gRPC Stream 代理） |
| `/api/v1/ws/traffic` | 流量速率实时推送（2 秒轮询） |

WebSocket 通过 URL 参数 `?token=<jwt>` 传递认证。

### 公开端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/sub/:token` | 订阅端点（无需认证） |

---

## 技术栈

### 后端（Go）

- 纯 Go 标准库 `net/http` 路由
- 复用 Xray 已有的 gRPC client stubs（HandlerService / StatsService / RoutingService / LoggerService / ObservatoryService）
- Go `embed.FS` 嵌入前端编译产物
- `gorilla/websocket`（项目已有依赖）处理实时数据流
- JWT 认证（纯 Go 实现，无外部依赖）

### 前端（Vue 3）

| 依赖 | 版本 | 用途 |
|------|------|------|
| Vue 3 | ^3.5 | 框架（Composition API） |
| Naive UI | ^2.39 | 组件库 |
| Pinia | ^2.2 | 状态管理 |
| Vue Router | ^4.4 | 路由 |
| Axios | ^1.7 | HTTP 客户端 |
| ECharts | ^5.5 | 图表 |
| vue-echarts | ^7.0 | ECharts Vue 封装 |
| vue-i18n | ^10.0 | 国际化 |
| qrcode.vue | ^3.4 | 二维码 |
| Vite | ^6.0 | 构建工具 |
| TypeScript | ^5.6 | 类型系统 |

---

## 项目结构

```
app/webpanel/                    # Go 后端
    config.proto                 # Protobuf 配置定义
    config.pb.go                 # Protobuf 生成代码
    webpanel.go                  # Feature 注册 + HTTP 服务器 + 路由
    auth.go                      # JWT 认证 + 登录限流
    grpc_client.go               # gRPC 连接管理（5 个服务客户端）
    handler_auth.go              # 登录 API
    handler_stats.go             # 统计 API
    handler_handler.go           # 入站/出站管理 API
    handler_users.go             # 用户管理 API
    handler_routing.go           # 路由 API
    handler_config.go            # 配置文件 CRUD API
    handler_logger.go            # 日志 API
    handler_observatory.go       # Observatory API
    handler_subscription.go      # 订阅端点
    ws_routing.go                # WebSocket 路由流 + 流量推送
    config_file.go               # 配置文件读写/备份逻辑
    share_link.go                # 分享链接生成（VLESS/VMess/Trojan/SS）
    embed.go                     # //go:embed dist
    dist/                        # 前端编译产物（嵌入）

web/                             # Vue 3 前端源码
    src/
        main.ts                  # 入口
        App.vue                  # 根组件（主题 + 语言 Provider）
        api/
            client.ts            # Axios 实例 + JWT 拦截器 + 全部 API
            types.ts             # TypeScript 类型定义
        stores/
            app.ts               # 主题/语言/侧边栏状态
            auth.ts              # 认证状态
            stats.ts             # 统计数据
        router/index.ts          # 路由定义 + 导航守卫
        views/                   # 10 个页面组件
        components/layout/       # AppShell 布局
        composables/             # usePolling / useWebSocket
        i18n/locales/            # zh-CN.json / en.json
        utils/format.ts          # 格式化工具函数

infra/conf/webpanel.go           # JSON 配置解析
```

---

## 安全建议

1. **修改默认密码** - 配置中的 `username` 和 `password` 务必修改
2. **修改 JWT 密钥** - `jwtSecret` 使用随机字符串
3. **默认仅本地** - `listen` 默认 `127.0.0.1`，不暴露到公网
4. **使用反向代理** - 如需公网访问，建议通过 Nginx 反向代理并配置 HTTPS
5. **或配置 TLS** - 设置 `certFile` 和 `keyFile` 启用内置 HTTPS
6. **自动备份** - 配置文件写入前自动创建时间戳备份，保留最近 20 个

---

## 前端开发

```bash
# 启动前端开发服务器（热重载）
cd web
npm install
npm run dev

# Vite 代理配置已指向 127.0.0.1:9527
# 确保 Xray + Web Panel 在后台运行
```

---

## 构建命令

```bash
make build      # 完整构建：前端 + Go 二进制
make web        # 仅构建前端
make build-dev  # 仅构建 Go（占位前端）
make clean      # 清理构建产物
```
