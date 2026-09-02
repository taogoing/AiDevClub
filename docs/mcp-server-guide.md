# AIDevClub MCP Server 使用指南

## 1. 概述

AIDevClub 提供了符合 MCP (Model Context Protocol) 标准的 Streamable HTTP 服务器，让 AI Agent（如 Claude Code、Cursor、Windsurf 等）能够直接检索和操作平台内容。

**生产环境端点**: `https://aidevclub.xyz/mcp`

## 2. 架构设计

### 2.1 核心设计原则

| 原则 | 说明 |
|---|---|
| **只读设计** | 所有工具均为只读，不会修改数据，适合公开访问 |
| **无状态** | 每次请求创建独立的 MCP 服务器实例，无会话状态 |
| **共享服务层** | 与 REST API 共享相同的业务服务、数据库和缓存 |
| **官方 SDK** | 使用 Go 官方 MCP SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) |

### 2.2 请求处理流程

```
MCP Client → HTTP POST /mcp → Middleware Chain → MCP SDK Handler
                                                    ↓
                                          工具路由 → 服务层 → 数据库/Redis
                                                    ↓
                                            JSON-RPC 响应返回
```

### 2.3 中间件链（按顺序执行）

1. **Request ID** - 生成或复用 `X-Request-ID`
2. **Panic Recovery** - 捕获 panic，返回 500
3. **Origin Validation** - 校验 Origin 头（可选）
4. **Bearer Authentication** - JWT 认证（可选）
5. **Rate Limiting** - Redis 限流（60次/分钟/IP）
6. **Request Timeout** - 请求超时（30秒）
7. **MCP SDK Handler** - 处理 JSON-RPC 请求

## 3. 可用工具

### 3.1 公开工具（6个，无需认证）

#### `search_content` - 搜索内容

```json
{
  "name": "search_content",
  "arguments": {
    "query": "Go MCP",           // 搜索关键词
    "content_type": "all",       // all | article | skill | mcp_server
    "tag_id": null,              // 标签ID过滤
    "sort": "relevance",         // relevance | latest
    "page": 1,
    "page_size": 10
  }
}
```

#### `browse_content` - 浏览内容

```json
{
  "name": "browse_content",
  "arguments": {
    "content_type": "article",   // all | article | skill | mcp_server
    "sort": "latest",            // latest | hot
    "page": 1,
    "page_size": 10
  }
}
```

#### `get_article` - 获取文章详情

```json
{
  "name": "get_article",
  "arguments": {
    "id": 1,                     // 文章ID
    "content_offset": 0,         // 内容起始偏移（用于分页读取长文章）
    "content_limit": 30000       // 内容长度限制（最大50000字符）
  }
}
```

#### `get_skill` - 获取 Skill 详情

```json
{
  "name": "get_skill",
  "arguments": {
    "id": 1,
    "content_offset": 0,
    "content_limit": 30000
  }
}
```

#### `get_mcp_server` - 获取 MCP Server 详情

```json
{
  "name": "get_mcp_server",
  "arguments": {
    "id": 1,
    "content_offset": 0,
    "content_limit": 30000
  }
}
```

#### `list_taxonomy` - 列出标签

```json
{
  "name": "list_taxonomy",
  "arguments": {
    "kind": "tags",              // 仅支持 tags
    "keyword": "Go",             // 可选关键词过滤
    "limit": 50                  // 1-100
  }
}
```

### 3.2 认证工具（3个，需要 Bearer Token）

#### `get_my_profile` - 获取用户资料

```json
{
  "name": "get_my_profile",
  "arguments": {}
}
```

#### `list_my_content` - 列出我的内容

```json
{
  "name": "list_my_content",
  "arguments": {
    "content_type": "article",   // article | skill | mcp_server
    "status": "published",       // draft | published | pending_review | rejected | archived
    "page": 1,
    "page_size": 10
  }
}
```

#### `list_my_notifications` - 列出通知

```json
{
  "name": "list_my_notifications",
  "arguments": {
    "type": "comment_article",   // 通知类型过滤
    "unread_only": true,         // 仅未读
    "page": 1,
    "page_size": 20
  }
}
```

## 4. Agent 连接配置

### 4.1 Claude Code 配置

在 `~/.claude/claude_desktop_config.json` 或项目 `.mcp.json` 中添加：

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "https://aidevclub.xyz/mcp"
    }
  }
}
```

### 4.2 Cursor 配置

在项目根目录创建 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "https://aidevclub.xyz/mcp"
    }
  }
}
```

### 4.3 Windsurf 配置

在 `~/.windsurf/mcp.json` 中添加：

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "https://aidevclub.xyz/mcp"
    }
  }
}
```

### 4.4 认证访问（获取完整功能）

如果需要访问认证工具（`get_my_profile`、`list_my_content`、`list_my_notifications`），需要提供 Bearer Token：

```json
{
  "mcpServers": {
    "aidevclub": {
      "url": "https://aidevclub.xyz/mcp",
      "headers": {
        "Authorization": "Bearer <your-access-token>"
      }
    }
  }
}
```

**获取 Access Token**：
1. 在 AIDevClub 平台注册/登录
2. 调用 `/api/v1/auth/login` 获取 access_token
3. 将 token 填入配置

## 5. 使用示例

### 5.1 搜索 Go 相关文章

Agent 会自动调用 `search_content` 工具：

```
用户: 搜索 AIDevClub 上关于 Go 语言的文章

Agent 调用:
{
  "name": "search_content",
  "arguments": {
    "query": "Go",
    "content_type": "article",
    "sort": "relevance"
  }
}
```

### 5.2 获取文章详情

```
用户: 读取文章 ID 为 1 的内容

Agent 调用:
{
  "name": "get_article",
  "arguments": {
    "id": 1,
    "content_limit": 5000
  }
}
```

### 5.3 浏览热门 Skill

```
用户: 查看热门的 Skill

Agent 调用:
{
  "name": "browse_content",
  "arguments": {
    "content_type": "skill",
    "sort": "hot"
  }
}
```

### 5.4 查看 MCP Server 安装配置

```
用户: 获取 MCP Server 的安装方法

Agent 调用:
{
  "name": "get_mcp_server",
  "arguments": {
    "id": 1
  }
}
```

返回包含 `installations` 数组，包含各客户端的安装命令和配置。

### 5.5 分页读取长文章

对于长文章，使用 `content_offset` 和 `content_limit` 分页读取：

```json
// 第一次请求
{
  "name": "get_article",
  "arguments": {
    "id": 1,
    "content_offset": 0,
    "content_limit": 5000
  }
}

// 返回 has_more: true, next_offset: 5000

// 第二次请求
{
  "name": "get_article",
  "arguments": {
    "id": 1,
    "content_offset": 5000,
    "content_limit": 5000
  }
}
```

## 6. 错误处理

| 错误码 | HTTP 状态 | 说明 |
|---|---|---|
| `invalid_argument` | 200 (工具结果) | 参数错误 |
| `content_not_found` | 200 (工具结果) | 内容不存在或无权访问 |
| `not_authenticated` | 200 (工具结果) | 需要认证但未提供 |
| - | 401 | 无效/过期的 Bearer Token |
| - | 429 | 超出速率限制（60次/分钟） |
| - | 413 | 请求体过大（最大1MB） |

## 7. 环境变量配置

| 变量 | 默认值 | 说明 |
|---|---|---|
| `AIDEVCLUB_MCP_ADDR` | `:8081` | MCP 服务器监听地址 |
| `AIDEVCLUB_MCP_RATE_LIMIT_PER_MINUTE` | `60` | 每分钟请求限制 |
| `AIDEVCLUB_MCP_REQUEST_TIMEOUT` | `30s` | 单次请求超时 |
| `AIDEVCLUB_MCP_MAX_BODY_BYTES` | `1048576` | 最大请求体大小（1MB） |

## 8. 健康检查

```bash
# 存活探针
curl https://aidevclub.xyz/healthz
# 返回: {"status":"ok"}

# 就绪探针（检查 MySQL 和 Redis 连接）
curl https://aidevclub.xyz/readyz
# 返回: {"status":"ok"} 或 {"status":"error"}
```

## 9. 最佳实践

1. **使用内容窗口** - 对于长内容，使用 `content_offset` 和 `content_limit` 分页读取，避免单次返回过大
2. **合理设置 page_size** - 列表查询时根据需要设置每页数量，避免一次获取过多数据
3. **利用缓存** - 热门内容通过 Redis 缓存，频繁查询可获得更好的性能
4. **错误重试** - 对于临时不可用错误，可适当重试
5. **认证访问** - 如果需要查看个人内容，确保配置有效的 access_token

## 10. 相关链接

- **生产环境**: https://aidevclub.xyz
- **MCP 端点**: https://aidevclub.xyz/mcp
- **GitHub**: https://github.com/taogoing/AiDevClub
- **MCP 协议规范**: https://modelcontextprotocol.io