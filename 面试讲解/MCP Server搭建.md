# MCP Server 搭建：面试讲解

## 对应的简历内容

> **MCP Server 搭建：**基于 **MCP Go SDK** 搭建 MCP Server，将平台搜索、热榜、资源浏览、个人数据等能力封装为标准化 **MCP Tools**，支持主流 AI Agent 通过 MCP 协议自主检索和调用平台数据。

这段经历的核心不是“写了几个接口”，而是把原本只供网页或 REST API 使用的平台能力，包装成 AI Agent 可以**自动发现、理解和调用**的标准工具。

## 30 秒回答版本

我使用官方 MCP Go SDK 为平台搭建了一个基于 Streamable HTTP 的 MCP Server。服务端把搜索、内容浏览、文章/Skill/MCP Server 详情、标签查询，以及登录用户的资料、内容和通知等能力，注册成 9 个只读 MCP Tool。

Claude Code、Cursor 等 Agent 作为 MCP Client 连接 `/mcp` 后，会先获取 Tool 列表和参数 Schema；模型根据用户意图选择工具，通过 JSON-RPC 传递参数。服务端经由鉴权、限流和超时等中间件后，将调用路由到 Tool Handler；Handler 不直接访问数据库，而是复用已有 Service 层和权限规则查询 MySQL、Redis，最后返回结构化结果给 Agent。

## 一句话理解 MCP

```text
MCP 是 AI Agent 与外部能力之间的统一协议。
```

本项目中的角色：

```text
Claude Code / Cursor      = MCP Client（Agent）
AIDevClub MCP Server      = 能力提供方
search_content 等 Tool    = Agent 可调用的业务函数
JSON-RPC                  = 消息格式
Streamable HTTP           = 消息传输方式
MySQL / Redis             = 平台数据与缓存
```

最重要的区分：

```text
Streamable HTTP：消息怎么传输（HTTP POST /mcp）
JSON-RPC：消息长什么样（调用哪个方法、参数是什么、返回什么）
MCP：Agent 发现和调用 Tool 的完整标准
```

## 系统架构

```text
用户自然语言请求
        ↓
Claude Code / Cursor（MCP Client）
        ↓  tools/list：发现工具
        ↓  tools/call：调用工具
HTTP POST /mcp（Streamable HTTP，JSON-RPC）
        ↓
MCP 中间件：Request ID → 异常恢复 → Origin → JWT → 限流 → 超时
        ↓
MCP Go SDK：解析协议、校验 Schema、分发 Tool
        ↓
internal/mcpserver：参数适配、输出整形、错误映射
        ↓
既有 Service 层：搜索/文章/Skill/MCP资源/用户/通知
        ↓
Repository 层 → MySQL / Redis
        ↓
结构化 JSON 结果 → Agent → 自然语言回答
```

## 为什么选择 Streamable HTTP

MCP 常见传输方式有两种：

| 方式 | 适用场景 | 本项目是否采用 |
|---|---|---|
| stdio | MCP Client 在本地拉起一个子进程，例如本地文件工具 | 否 |
| Streamable HTTP | 服务远程部署，多个 Agent 客户端通过 URL 访问 | 是 |

本项目服务端监听 `:8081`，对外端点是 `/mcp`。选择 Streamable HTTP 的原因：平台能力需要让 Claude Code、Cursor、Windsurf 等远程客户端访问，并且 HTTP 容易接入网关、鉴权、限流、健康检查和部署体系。

SDK 初始化在 `internal/mcpserver/handler.go`：

```go
sdkHandler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server {
        actor := actorFromRequest(r)
        return newMCPServer(deps, actor, cfg)
    },
    &mcp.StreamableHTTPOptions{
        Stateless: true,
        JSONResponse: true,
        PropagateRequestCancellation: true,
        MaxRequestBodyBytes: cfg.MCPMaxBodyBytes,
    },
)
```

重点解释：

- `Stateless: true`：不保存 MCP 会话状态，每个请求可独立处理，易于水平扩容。
- `JSONResponse: true`：查询型工具直接返回 JSON，不需要持续推送结果。
- `PropagateRequestCancellation: true`：Agent 取消请求时，Go 的 `context` 也会取消，底层查询可以尽快停止。
- `MaxRequestBodyBytes`：限制请求体，避免恶意或异常大请求。

## Tool 是怎样设计的

项目共注册 9 个只读 Tool。

| 类别 | Tool | 作用 |
|---|---|---|
| 公开 | `search_content` | 搜索文章、Skill、MCP Server |
| 公开 | `browse_content` | 按最新或热度浏览内容 |
| 公开 | `get_article` | 读取已发布文章 |
| 公开 | `get_skill` | 读取 Skill 与 SKILL.md |
| 公开 | `get_mcp_server` | 读取 MCP Server、安装配置与 README |
| 公开 | `list_taxonomy` | 查询标签 |
| 登录后 | `get_my_profile` | 查询当前用户资料 |
| 登录后 | `list_my_content` | 查询当前用户自己的内容 |
| 登录后 | `list_my_notifications` | 查询当前用户通知 |

所有 Tool 都标记了：

```go
&mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
```

即 Tool 不改变平台数据，并且重复调用不会产生副作用。这样 Agent、客户端和权限系统能更准确地理解调用风险。

## 以 `search_content` 为例：从设计到调用

### 1. 定义输入契约

```go
type searchContentInput struct {
    Query       string
    ContentType string // all | article | skill | mcp_server
    TagID       *uint
    Sort        string // relevance | latest
    Page        int
    PageSize    int
}
```

项目使用 `jsonschema-go` 从 Go struct 推导 JSON Schema，再增加枚举、默认值和分页边界。例如：

- `content_type` 只能是 `all/article/skill/mcp_server`；
- 输入关键词时默认按 `relevance` 搜索；
- 没有关键词、只按标签筛选时默认按 `latest`；
- `all` 查询时每类默认返回 5 条，最大每类 10 条；单类型最大 20 条。

Schema 是 Agent 可读的工具说明书；但 Handler 里仍会做一遍运行时校验，不能完全信任外部调用方。

### 2. 注册 Tool

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "search_content",
    Description: "Search published AIDevClub content.",
    InputSchema: searchContentInputSchema(),
}, searchContent(deps, publicBaseURL))
```

`Name` 用于调用；`Description` 帮模型判断何时该用它；`InputSchema` 描述参数；最后的 `Handler` 才是真正执行业务的 Go 函数。

### 3. Agent 调用

用户说“搜索平台上的 MCP Server”，Agent 会在工具发现后发出一条类似请求：

```http
POST /mcp
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": {
      "query": "MCP",
      "content_type": "mcp_server",
      "page": 1,
      "page_size": 5
    }
  }
}
```

### 4. Handler 复用 Service

MCP Handler 将输入归一化后，调用已有 SearchService：

```go
response, err := deps.Search.Search(ctx, service.SearchQuery{
    Keyword:     "MCP",
    ContentType: "mcp_server",
    Page:        1,
    PageSize:    5,
    Highlight:   false,
})
```

这里的关键设计是：**MCP Handler 不写 SQL，不复制搜索业务。**它使用与网页 REST API 同一个 `SearchService`。`Highlight: false` 是因为 HTML 高亮是网页展示需求，不应污染给模型的结构化结果。

底层查询会确保只暴露：

```sql
status = 'published' AND hidden = false
```

因此草稿、待审核、已隐藏资源不会因 MCP Tool 而泄露。

### 5. 返回给 Agent

结果会转换为 MCP 友好的稳定 DTO，例如：

```json
{
  "content_type": "mcp_server",
  "sort": "relevance",
  "mcp_servers": [
    {
      "id": 18,
      "title": "GitHub MCP Server",
      "summary": "用于查询仓库、Issue 和 Pull Request",
      "url": "https://aidevclub.xyz/mcps/18",
      "tags": [{ "id": 3, "name": "mcp" }]
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 5
}
```

Agent 用这份结构化结果，再生成自然语言回复给用户。

## 鉴权流程

公开 Tool 不需要登录；个人 Tool 需要用户在网站登录后，将 JWT access token 配置给 MCP Client。

```text
网站 REST 登录
        ↓
返回 JWT access_token
        ↓
用户在 Claude/Cursor MCP 配置中写 Header
        ↓
每次 /mcp 请求带 Authorization: Bearer <token>
        ↓
服务端 JWT 中间件验证签名和过期时间
        ↓
把 userID 写入 request Context
        ↓
当前请求动态注册个人 Tool，并按 userID 查询数据
```

客户端配置示例：

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "https://aidevclub.xyz/mcp",
      "headers": {
        "Authorization": "Bearer <access_token>"
      }
    }
  }
}
```

服务端鉴权的关键逻辑：

```go
userID, err := platform.ParseAccessToken(jwtSecret, token)
ctx := context.WithValue(r.Context(), actorContextKey{}, Actor{
    UserID: userID,
    Authenticated: true,
})
next.ServeHTTP(w, r.WithContext(ctx))
```

无 Token：匿名访问，可使用公开 Tool。Token 非法或过期：HTTP 401。Token 合法：除公开 Tool 外，再动态注册个人 Tool。

当前 access token 默认有效期为 15 分钟；过期后用户需要重新登录并更新 MCP Client 配置。若继续演进，可以实现 OAuth 或安全的自动 Token 刷新，避免在配置中手工维护短期 Token。

## 安全与稳定性

请求在进入 MCP SDK 前依次经过：

```text
Request ID → Panic Recovery → Origin 校验 → JWT → Redis 限流 → 请求超时
```

| 机制 | 作用 |
|---|---|
| Request ID | 将一次请求与日志关联，方便排障 |
| Panic Recovery | 避免单请求异常导致进程崩溃 |
| Origin 校验 | 限制浏览器来源 |
| JWT | 区分游客与具体登录用户 |
| Redis 限流 | 游客按 IP、登录用户按 userID 限流；默认 60 次/分钟 |
| 请求超时 | 默认 30 秒，防止慢查询长期占用资源 |
| 请求体限制 | 默认 1 MB，降低大包攻击风险 |
| 统一错误 | 对外返回 `invalid_argument`、`content_not_found` 等稳定错误，避免泄露内部错误 |

## 高频面试题与回答

### 1. MCP 与普通 REST API 有什么区别？

REST API 是给前端或开发者直接调用的业务接口；MCP 是面向 Agent 的标准工具协议。MCP 不只返回数据，还提供 Tool 名称、用途和 JSON Schema，让模型能够自主发现工具、判断何时调用、生成正确参数。底层业务仍可复用 REST API 的 Service 层。

### 2. 为什么不用 stdio？

stdio 适合 Client 启动本地子进程，例如本地文件系统 Tool。我们的能力是远程平台服务，需要多个 Agent 通过 URL 访问，所以选择 Streamable HTTP，便于部署、鉴权、限流和运维。

### 3. JSON-RPC 和 HTTP 是什么关系？

HTTP 是承载消息的传输通道；JSON-RPC 是消息格式。例如 HTTP POST 将一条 `tools/call` 的 JSON-RPC 请求发送到 `/mcp`，服务端在 HTTP 响应中返回 JSON-RPC 结果。

### 4. JSON-RPC 和普通 JSON 有什么区别？为什么不用自定义 JSON？

JSON 只是数据格式，规定对象、数组、字符串等数据怎么表示；JSON-RPC 是建立在 JSON 之上的远程调用规范。

仅有下面的 JSON：

```json
{ "query": "MCP", "content_type": "mcp_server" }
```

服务端无法从中确定：这是不是一次请求、要调用哪个功能、如何与并发响应关联、失败时错误以什么格式返回。JSON-RPC 统一规定了这些字段：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": { "query": "MCP" }
  }
}
```

| 字段 | 作用 |
|---|---|
| `method` | 要调用的远程方法 |
| `params` | 方法参数 |
| `id` | 关联请求与响应，支持并发调用 |
| `result` / `error` | 统一表达成功结果或失败结果 |

因此不是不能使用 JSON，而是 MCP 为了让不同 Agent 和服务端互通，选择 JSON-RPC 作为统一规范。自己设计普通 JSON 接口会变成私有 REST/HTTP 接口，Claude Code、Cursor 等客户端无法把它自动识别为 MCP Tool。

### 5. HTTP JSON 接口（REST）和 JSON-RPC 的核心区别是什么？

最简单的记忆方式：

> **RPC 更像是把服务端能力暴露成可远程调用的函数，也就是“动作”。**

调用方只需要知道函数（动作）名、参数和返回值，不需要关心服务端内部如何实现：

```text
本地函数：searchContent("MCP", "mcp_server")
远程 RPC：method = "search_content"，arguments = {"query": "MCP"}
```

| 对比项 | REST / HTTP JSON 接口 | JSON-RPC |
|---|---|---|
| 设计视角 | 资源导向：操作哪个资源 | 动作导向：调用哪个函数 |
| 功能定位 | HTTP Method + URL 路径 | JSON Body 中的 `method` |
| 参数位置 | URL path、query 或 body | `params` |
| 常见例子 | `GET /api/v1/articles/18` | `POST /rpc` + `method: "article.get"` |
| 错误格式 | 项目自行约定 | 有标准 `error` 结构；业务错误码仍可自定义 |
| 批量/通知 | 需自行设计 | 协议支持批量请求、无需响应的通知 |

一句话记忆：

```text
REST：我要操作哪个资源？
JSON-RPC：我要调用哪个函数（也就是执行哪个动作）？
```

MCP 中有两层方法名，面试时不要混淆：

```json
{
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": { "query": "MCP" }
  }
}
```

- `method: "tools/call"`：MCP 协议层的标准 JSON-RPC 方法，含义是“调用一个 Tool”。
- `params.name: "search_content"`：项目注册的具体业务 Tool。
- `arguments`：这个 Tool 的业务参数。

### 6. Tool 为什么要有 JSON Schema？

Schema 让模型和客户端知道工具参数、类型、必填项、枚举、默认值与边界。它是 AI 可调用性的契约；同时服务端仍要做运行时校验，保证安全性。

### 7. 如何保证 MCP 不能访问未发布内容？

公开 Tool 复用既有 Service/Repository 逻辑，底层查询带 `published` 和 `hidden=false` 条件；而个人 Tool 通过 JWT 获得当前 userID，只查询该用户拥有的数据。权限规则不在 MCP 层重复实现。

### 8. 为什么要按请求动态注册 Tool？

未认证请求根本不会发现个人 Tool，而不是“看得到但调用失败”。这减少了暴露面，也使 Tool 清单和当前身份的能力保持一致；无状态服务按请求创建 Server，正适合这种设计。

### 9. 如何控制大结果和模型上下文消耗？

列表 Tool 有分页和上限；详情 Tool 对长文章、SKILL.md、README 使用 `content_offset + content_limit` 窗口读取，默认 3 万字符、最大 5 万字符，并返回 `has_more` 和 `next_offset`。切片按 Unicode 字符而不是字节，避免截断中文或 Emoji。

## 可以主动说的改进点

1. 增加 MCP 协议级集成测试，覆盖 `tools/list`、`tools/call`、无 Token/合法 Token/过期 Token、限流和超时路径。
2. 当前短期 access token 需要用户更新客户端配置；生产化可升级为 OAuth 或安全的 Token 刷新机制。
3. 当前 Request ID 使用进程内计数器；高并发生产场景可改用 UUID 或 `sync/atomic`，避免并发数据竞争。
4. 继续新增写操作时，应将读写 Tool 分离；写 Tool 要增加显式用户确认、幂等键、审计日志及更细粒度授权。

## 最终可背诵版本

> 我基于 MCP Go SDK 搭建了 Streamable HTTP MCP Server，将平台搜索、浏览、内容详情和个人数据查询封装为标准 MCP Tools。Agent 先通过 `tools/list` 获取工具描述和 JSON Schema，模型再根据用户意图通过 JSON-RPC 调用 Tool。服务端在 MCP SDK 前完成 JWT 鉴权、Redis 限流、超时、Origin 校验等处理；Tool Handler 只做协议适配、参数校验和输出整形，底层复用已有 Service/Repository 层，保证 REST 和 MCP 的数据与权限逻辑一致。公开内容仅返回已发布且未隐藏的数据，认证后再动态暴露个人资料、我的内容和通知等 Tool。
