# Task 11 Report: Integrate Notification Triggers + ReplyToID + Route Registration + Initial Admin

## Status: DONE

## Changes Summary

### 1. CommentService (`internal/service/comment.go`)
- Injected `NotificationService` into struct and constructor
- `Create()`: saves original `parentID` as `ReplyToID`; sends `comment_article` or `reply_comment` notification asynchronously
- `ToggleLike()`: sends `like_comment` notification asynchronously on like
- Self-actions skip notifications (handled by `NotificationService.Create` check)

### 2. ResourceCommentService (`internal/service/resource_comment.go`)
- Injected `NotificationService` into struct and constructor
- `Create()`: saves original `parentID` as `ReplyToID`; sends notifications for resource comments
- `ToggleLike()`: sends `like_resource_comment` notification on like

### 3. ArticleService (`internal/service/article.go`)
- Injected `NotificationService` into struct and constructor
- `ToggleLike()`: sends `like_article` notification on like

### 4. SkillService (`internal/service/skill.go`)
- Injected `NotificationService` into struct and constructor
- `ToggleLike()`: sends `like_skill` notification on like

### 5. McpServerService (`internal/service/mcp_server.go`)
- Injected `NotificationService` into struct and constructor
- `ToggleLike()`: sends `like_mcp_server` notification on like

### 6. main.go (`cmd/server/main.go`)
- Added `Notification`, `Report`, `AdminLog`, `Announcement` to AutoMigrate
- Initialized `notifRepo` and `notifSvc` early in startup
- Updated all service constructor calls with `notifSvc` parameter
- Initialized new repos: `adminLogRepo`, `announcementRepo`, `reportRepo`
- Initialized new services: `adminLogSvc`, `adminSvc`, `reportSvc`
- Registered notification routes (`/api/v1/notifications`)
- Registered report routes (`/api/v1/reports`)
- Registered admin routes (`/api/v1/admin`) with `AdminMiddleware`
- Added initial admin seeding from `cfg.AdminEmails`

## Verification
- `go build ./...` passes with zero errors
