# Xray Web Panel - 内嵌式 Web 管理面板

Xray Web Panel 是 Xray-core 的内嵌式 Web 管理面板，编译后生成**单一二进制文件**，无需额外部署前端服务，开箱即用。

当前这个 fork 的 WebPanel 不只是基础 CRUD 面板，还覆盖了订阅导入、节点池生命周期、透明代理 / TUN 控制、DNS 分流说明和上游版本更新发现。

## 特性

- **单一二进制部署** - 前端通过 Go `embed.FS` 嵌入，一个文件搞定一切
- **完整的 gRPC 桥接** - 通过 REST API 代理 Xray Commander 的核心 gRPC 接口
- **双通道控制** - Runtime API 即时生效 + Config File 持久化修改
- **版本更新发现** - 仪表盘可检查当前版本与上游 Xray-core 发布状态
- **订阅导入** - 支持远程 URL、手工粘贴、本地文件上传
- **节点池生命周期** - 候选、验证中、活跃、隔离、已移除五个池，支持排序和批量操作
- **节点情报悬停明细** - 默认只显示干净度 / 网络类型标签，详情通过悬停浮层查看
- **透明代理 / TUN 管理** - 提供启停、恢复干净系统、严格全隧道接管、加密远端 DNS、直连白名单、节点选择策略
- **赞赏支持入口** - 内置爱发电链接与二维码，适合中国大陆用户直接支持维护
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
| `configPath` | string | - | Xray 配置文件路径，用于配置编辑、订阅、节点池与备份功能 |
| `certFile` | string | - | TLS 证书文件路径（可选） |
| `keyFile` | string | - | TLS 私钥文件路径（可选） |

---

## 页面与功能

### 仪表盘 (`/dashboard`)

- 系统信息：运行时间、Goroutine 数、内存占用、GC 次数
- 流量概览：总上传/下载
- 在线用户数
- 当前版本与上游发布状态检查
- 快捷入口：就绪中心、更新检查、赞赏支持
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

- 这是一个**解释页**，不是独立 DNS 编辑器
- 展示主配置文件里的 `dns` 段、静态 Hosts 和查询策略
- 展示透明模式下显式保护直连 / 默认代理的 DNS 分流逻辑
- 基础 DNS 配置仍在 `/config` 页面维护
- 透明模式使用的远端 DNS 列表在 `/node-pool` 页面维护，运行时会把裸 IP 或 tcp/udp DNS 自动规范化为 DoH 加密解析器

### 隐私诊断 (`/privacy`)

- 对照浏览器侧证据与当前 TUN、DNS、节点池状态，检查 IP 定位、WebRTC、DNS、指纹、IP 纯净度和节点池去重
- 页面本身负责检测和解释；网络层防护依赖严格透明 TUN，全隧道模式会接管非旁路 IPv4、启用期间禁用 IPv6，并把远端 DNS 规范化为 DoH 加密解析
- 日常 Chrome/Chromium-family 浏览器需要先安装托管策略：`sudo ./scripts/install-browser-privacy-policy.sh`，再完全重启浏览器并在 `chrome://policy` 检查策略生效
- IPPure 验收以仓库脚本为准：在仓库根目录运行 `node scripts/verify-ippure.mjs`
- 该脚本默认从 `IPPURE_CONFIG`、当前仓库配置或正在运行的 `xray run -c ...` 配置中发现本地 SOCKS 入站，并启用加固浏览器策略、关闭 WebRTC API、随机化指纹 profile；需要可见浏览器时运行 `IPPURE_HEADLESS=0 IPPURE_KEEP_OPEN=1 node scripts/verify-ippure.mjs`
- 普通浏览器在透明 TUN 开启后不应暴露直连 IPv4、IPv6、DNS 或 WebRTC/STUN 网络路径；若仍暴露直连 IP，应按 TUN/helper 回归处理。浏览器指纹唯一性仍属于浏览器配置或 profile 策略问题

### 实时监控 (`/monitor`)

- **流量监控**：WebSocket 实时速度折线图
- **连接监控**：WebSocket 推送活跃连接（入站、出站、目标、用户等）
- **系统监控**：Goroutine / 内存时间线图（2 秒轮询）

### 系统设置 (`/settings`)

- 日志配置查看 + 重启日志按钮
- 策略配置查看
- Observatory 出站健康状态表
- API 设置查看
- TUN 诊断信息查看，包括 direct / proxy egress 语义和结构化 DNS / 路由决策
- 这里主要保留诊断信息；透明模式主控制入口已经转移到节点池页

### 配置管理 (`/config`)

- **原始编辑器**：直接编辑 JSON 配置
- **验证配置**：保存前校验 JSON 合法性
- **导入/导出**：JSON 文件上传/下载
- **备份/恢复**：自动时间戳备份（保留最近 20 个），一键恢复

### 订阅管理 (`/subscriptions`)

- 支持三种导入方式：远程 URL、手工粘贴、本地文件上传
- 可设置备注、自动刷新和刷新间隔
- 支持手工立即刷新订阅
- 删除订阅时，该订阅下节点会转入生命周期管理，不会直接抹掉全部记录
- 本地上传文件只在导入当次读取内容，不会持续监听本地文件变化

### 赞赏支持 (`/support`)

- 内置爱发电主页入口：`https://ifdian.net/a/abc678`
- 提供二维码，方便手机扫码直接进入赞赏页
- 入口同时出现在侧边栏和仪表盘快捷操作中
- 仅在用户主动打开页面时展示，不影响任何代理或管理功能

### 节点池 (`/node-pool`)

- 显示五个池：候选、验证中、活跃、隔离、已移除
- 候选池**不会自动探测**；只有手工加入验证池后才开始探测
- 各池支持按质量、最后探测时间、失败率、平均延迟排序
- 支持单个和批量操作：加入验证、晋升活跃、移出活跃、移入已移除、恢复到候选、彻底删除已移除节点
- 支持批量移除不稳定节点，以及清理 100% 失败率的已移除记录
- 节点情报默认压缩成标签展示，出口 IP、判断原因和技术细节改为悬停查看
- 透明模式主控制入口在这里：启用严格全隧道透明模式、恢复干净系统、配置最少活跃节点、探测参数、远端 DNS、直连白名单、分流策略、节点选择策略
- 实验性 UDP/QUIC 聚合脚手架也在这里维护：默认关闭；即使打开，也会明确显示当前是否仍回退到稳定单路径模式以及原因
- 当实验性 UDP/QUIC 聚合打开时，NodePool 还会展示本地 scheduler prototype 的路径快照、预览会话和选择原因，方便在不切换实际转发路径的前提下调试 #41 这条本地主线
- 当 relayEndpoint 已配置时，NodePool 还会展示 relay assembler 预览和两组可重复 synthetic benchmark，对比稳定单路径与实验调度在健康路径/主路径退化场景下的启动时延、卡顿、吞吐和丢包变化

### 节点生命周期速览

- **候选池**：新发现节点、订阅中暂时消失的节点、或从已移除手工恢复的节点；默认不自动探测
- **验证池**：正在按探测地址统计失败率和平均延迟
- **活跃池**：探测达标，可参与透明模式和节点选择
- **隔离池**：连续失败过多或被手工移出活跃池，暂不参与活跃选择
- **已移除池**：手工移除或订阅整体删除后的节点；可手工恢复到候选池，或彻底删除记录

---

## REST API

所有 API 均需 JWT 认证（`Authorization: Bearer <token>`），除登录和公开订阅端点外。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录，返回 JWT token |

### 统计

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sys/stats` | 系统状态（Goroutine、内存、GC 等） |
| GET | `/api/v1/sys/update` | 检查当前版本与上游发布状态 |
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

### 透明模式 / TUN

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tun/status` | 获取透明模式状态与诊断信息 |
| GET | `/api/v1/tun/settings` | 读取透明模式可编辑设置 |
| PUT | `/api/v1/tun/settings` | 保存透明模式可编辑设置 |
| POST | `/api/v1/tun/start` | 启动透明模式 |
| POST | `/api/v1/tun/stop` | 停止透明模式 |
| POST | `/api/v1/tun/restore-clean` | 恢复干净系统 |
| POST | `/api/v1/tun/toggle` | 切换透明模式开关 |
| POST | `/api/v1/tun/install-privilege` | 安装或修复提权组件 |

### 订阅与节点池

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/subscriptions` | 列出订阅源 |
| POST | `/api/v1/subscriptions` | 添加订阅源 |
| DELETE | `/api/v1/subscriptions/:id` | 删除订阅源 |
| POST | `/api/v1/subscriptions/:id/refresh` | 立即刷新订阅源 |
| GET | `/api/v1/node-pool` | 获取节点池概览、节点列表、最近事件 |
| GET | `/api/v1/node-pool/config` | 获取节点验证配置 |
| PUT | `/api/v1/node-pool/config` | 更新节点验证配置 |
| POST | `/api/v1/node-pool/:id/validate` | 将节点加入验证池 |
| POST | `/api/v1/node-pool/:id/promote` | 将节点手工晋升为活跃 |
| POST | `/api/v1/node-pool/:id/quarantine` | 将节点移出活跃池 |
| POST | `/api/v1/node-pool/:id/demote` | 将节点降级到隔离池 |
| POST | `/api/v1/node-pool/:id/remove` | 将节点移入已移除池 |
| POST | `/api/v1/node-pool/:id/restore` | 将已移除节点恢复到候选池 |
| DELETE | `/api/v1/node-pool/:id` | 彻底删除单个节点记录 |
| POST | `/api/v1/node-pool/bulk-validate` | 批量加入验证池 |
| POST | `/api/v1/node-pool/bulk-promote` | 批量晋升活跃 |
| POST | `/api/v1/node-pool/bulk-restore` | 批量恢复到候选池 |
| POST | `/api/v1/node-pool/bulk-remove` | 按筛选条件批量移除节点 |
| POST | `/api/v1/node-pool/bulk-purge-removed` | 批量彻底删除已移除节点 |

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
- `gorilla/websocket` 处理实时数据流
- JWT 认证（纯 Go 实现，无额外 Web 框架）

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

```text
app/webpanel/                    # Go 后端
    config.proto                 # Protobuf 配置定义
    config.pb.go                 # Protobuf 生成代码
    webpanel.go                  # Feature 注册 + HTTP 服务器 + 路由
    auth.go                      # JWT 认证 + 登录限流
    grpc_client.go               # gRPC 连接管理
    handler_auth.go              # 登录 API
    handler_stats.go             # 统计 API
    handler_handler.go           # 入站/出站管理 API
    handler_users.go             # 用户管理 API
    handler_routing.go           # 路由 API
    handler_config.go            # 配置文件 CRUD API
    handler_logger.go            # 日志 API
    handler_observatory.go       # Observatory API
    handler_update.go            # 版本更新检查 API
    handler_tun.go               # 透明模式 / TUN API
    handler_node_pool.go         # 订阅与节点池 API
    handler_subscription.go      # 公开订阅端点
    ws_routing.go                # WebSocket 路由流 + 流量推送
    config_file.go               # 配置文件读写/备份逻辑
    subscription_manager.go      # 订阅同步与节点池生命周期
    tun_manager.go               # 透明模式运行时管理
    update_checker.go            # 上游版本检查
    share_link.go                # 分享链接生成
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
        views/                   # 12 个页面组件（含订阅、节点池、登录）
        components/layout/       # AppShell 布局
        composables/             # usePolling / useWebSocket
        i18n/locales/            # zh-CN.json / en.json
        utils/format.ts          # 格式化工具函数

infra/conf/webpanel.go           # JSON 配置解析
```

---

## i18n 要求

1. 所有用户可见的文案变更，必须在同一个 PR 内同时更新 `web/src/i18n/locales/zh-CN.json` 和 `web/src/i18n/locales/en.json`。
2. 不允许只改一个语言版本后再补另一个版本。
3. 新增页面或交互提示时，优先走 i18n key，不要把文案硬编码在组件里。

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
