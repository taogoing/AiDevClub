# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

AIDevClub 是一个面向开发者和 AI Agent 的技术内容与 AI 资源分享社区，包含三个核心板块：

1. **技术社区**：发布和交流技术文章。
2. **Skills Hub**：发布和分享 Skill（以 ZIP 资源包形式，内含 `SKILL.md`）。
3. **MCP Hub**：发布和分享 MCP Server。

平台同时提供整站 MCP Server，让 Claude Code、Codex 等 MCP 客户端检索和操作平台内容。完整需求见 [doc/AIDevClub需求文档.md](doc/AIDevClub需求文档.md)。

## 技术栈

- 后端：Go、Gin、GORM
- 数据库：MySQL 8
- 缓存与限流：Redis
- 前端：Vue 3、TypeScript、Vite
- 网关：Nginx
- 部署：Docker、Docker Compose
- 接口：REST API、MCP Server

## 架构

整体采用**模块化单体（Modular Monolith）**架构：

- REST API 与整站 MCP Server **共用同一套领域服务、权限规则和数据**，不重复实现业务逻辑。
- MySQL 持久化业务数据；Redis 负责缓存、限流、Token 状态与热门排行。
- 文章使用独立分类体系；Skill 与 MCP Server 不设分类，主要通过**统一标签（tag）**组织——文章、Skill、MCP Server 共用同一套标签。
- 内容与资源采用**软删除**；Skill 与 MCP Server 需管理员审核（草稿 → 待审核 → 已发布 / 已拒绝 / 已下架）后才公开。
- 用户无状态机，仅有登录 / 登出 / 注销账号（软删除）。认证采用 Access Token + Refresh Token，注册与登录接口限流。

## 开发工作流

使用 superpowers-zh 技能框架（见下文），走「brainstorming → writing-plans → 实现 → finishing-a-development-branch」流程。设计文档存 `docs/superpowers/specs/`，实现计划存 `docs/superpowers/plans/`。

## 现状与路线

P0+P1（基础设施骨架 + 用户认证）已完成并合并到 master。下一步是 P2（技术社区）。总体路线见 [docs/roadmap.md](docs/roadmap.md)，阶段一总结与遗留跟进项见 [docs/phase1-summary.md](docs/phase1-summary.md)。

标准命令：`go build ./...`、`go test ./...` 及 `go test ./<package>`（单测）。测试依赖真实 MySQL/Redis（先 `docker compose up -d`，Redis 宿主机端口为 16379）。

---

<!-- superpowers-zh:begin (do not edit between these markers) -->
# Superpowers-ZH 中文增强版

本项目已安装 superpowers-zh 技能框架（20 个 skills）。

## 核心规则

1. **收到任务时，先检查是否有匹配的 skill** — 哪怕只有 1% 的可能性也要检查
2. **设计先于编码** — 收到功能需求时，先用 brainstorming skill 做需求分析
3. **测试先于实现** — 写代码前先写测试（TDD）
4. **验证先于完成** — 声称完成前必须运行验证命令

## 可用 Skills

Skills 位于 `.claude/skills/` 目录，每个 skill 有独立的 `SKILL.md` 文件。

- **brainstorming**: 在任何创造性工作之前必须使用此技能——创建功能、构建组件、添加功能或修改行为。在实现之前先探索用户意图、需求和设计。
- **chinese-code-review**: 中文 review 沟通参考——话术模板、分级标注（必须修复/建议修改/仅供参考）、国内团队常见反模式应对。仅在用户显式 /chinese-code-review 时调用，不要根据上下文自动触发。
- **chinese-commit-conventions**: 中文 commit 与 changelog 配置参考——Conventional Commits 中文适配、commitlint/husky/commitizen 中文模板、conventional-changelog 中文配置。仅在用户显式 /chinese-commit-conventions 时调用，不要根据上下文自动触发。
- **chinese-documentation**: 中文文档排版参考——中英文空格、全半角标点、术语保留、链接格式、中文文案排版指北约定。仅在用户显式 /chinese-documentation 时调用，不要根据上下文自动触发。
- **chinese-git-workflow**: 国内 Git 平台配置参考——Gitee、Coding.net、极狐 GitLab、CNB 的 SSH/HTTPS/凭据/CI 接入差异与镜像同步配置。仅在用户显式 /chinese-git-workflow 时调用，不要根据上下文自动触发。
- **dispatching-parallel-agents**: 当面对 2 个以上可以独立进行、无共享状态或顺序依赖的任务时使用
- **executing-plans**: 当你有一份书面实现计划需要在单独的会话中执行，并设有审查检查点时使用
- **finishing-a-development-branch**: 当实现完成、所有测试通过、需要决定如何集成这份工作时使用
- **mcp-builder**: MCP 服务器构建方法论 — 系统化构建生产级 MCP 工具，让 AI 助手连接外部能力
- **receiving-code-review**: 收到代码审查反馈后、实施建议之前使用，尤其当反馈不明确或技术上有疑问时——需要技术严谨性和验证，而非敷衍附和或盲目执行
- **requesting-code-review**: 完成任务、实现重要功能或合并前使用，用于验证工作成果是否符合要求
- **subagent-driven-development**: 当在当前会话中执行包含独立任务的实现计划时使用
- **systematic-debugging**: 遇到任何 bug、测试失败或异常行为时使用，在提出修复方案之前执行
- **test-driven-development**: 在实现任何功能或修复 bug 时使用，在编写实现代码之前
- **using-git-worktrees**: 当需要开始与当前工作区隔离的功能开发，或在执行实现计划之前使用——通过原生工具或 git worktree 回退机制确保隔离工作区存在
- **using-superpowers**: 在开始任何对话时使用——确立如何查找和使用技能，要求在任何响应（包括澄清性问题）之前调用 Skill 工具
- **verification-before-completion**: 在宣称工作完成、已修复或测试通过之前使用，在提交或创建 PR 之前——必须运行验证命令并确认输出后才能声称成功；始终用证据支撑断言
- **workflow-runner**: 在 Claude Code / OpenClaw / Cursor 中直接运行 agency-orchestrator YAML 工作流——无需 API key，使用当前会话的 LLM 作为执行引擎。当用户提供 .yaml 工作流文件或要求多角色协作完成任务时触发。
- **writing-plans**: 当你有规格说明或需求用于多步骤任务时使用，在动手写代码之前
- **writing-skills**: 当创建新技能、编辑现有技能或在部署前验证技能是否有效时使用

## 如何使用

当任务匹配某个 skill 时，使用 `Skill` 工具加载对应 skill 并严格遵循其流程。绝不要用 Read 工具读取 SKILL.md 文件。

<!-- superpowers-zh:end -->
