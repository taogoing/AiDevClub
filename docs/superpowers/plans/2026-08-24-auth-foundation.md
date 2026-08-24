# P0+P1 基础设施骨架 + 用户与认证 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 搭建可运行的 Go/Gin 服务骨架，并实现用户注册、登录、登出、Token 刷新、资料管理、注销与头像上传。

**架构：** 扁平技术分层（`internal/handler` / `service` / `repo` / `model`），领域边界保留在 service 层。JWT Access Token（HS256，15 分钟）+ Redis 存储的不透明 Refresh Token（30 天，可吊销/轮换）。生产用 MySQL 8 + Redis；测试用内存 SQLite + miniredis。

**技术栈：** Go 1.21+、Gin、GORM（MySQL 驱动）、go-redis/v9、golang-jwt/v5、bcrypt、viper、slog。测试额外用 gorm sqlite 驱动 + miniredis。

**设计文档：** [2026-08-24-auth-foundation-design.md](../specs/2026-08-24-auth-foundation-design.md)

**依赖关系（避免循环 import）：**
- `model`：无依赖。
- `platform`：仅 stdlib + 第三方库（含 JWT、Redis、GORM、viper、gin），**不依赖** service/repo。
- `repo`：依赖 `model`。
- `service`：依赖 `model` + `platform` + `repo`。
- `handler`：依赖 `platform` + `service`。
- 因此 JWT 纯函数放在 `platform`，service 与中间件都调用它。

**约定：**
- 模块名 `aidevclub`，导入路径形如 `aidevclub/internal/...`。
- 统一响应 `{"code":0,"message":"ok","data":...}`。
- 业务错误码：`40001` 参数错误、`40101` 未认证、`40401` 用户不存在、`40901` 邮箱已存在、`42901` 限流、`50000` 服务器错误。
- 每个任务以可独立测试的交付物结束；先写测试、确认失败、再实现、确认通过、最后 commit。

---

### 任务 1：项目骨架、配置、日志与健康检查

**文件：**
- 创建：`go.mod`
- 创建：`internal/platform/config.go`
- 创建：`internal/platform/logger.go`
- 创建：`cmd/server/main.go`

- [ ] **步骤 1：初始化 Go 模块**

```bash
cd e:/goProject/library/src/GoStudy/AIDevClub
go mod init aidevclub
```

- [ ] **步骤 2：编写配置加载**

`internal/platform/config.go`：

```go
package platform

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr         string
	MySQLDSN         string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	JWTSecret        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	RateLimitPerMin  int
	AvatarDir        string
	DefaultAvatarURL string
	MaxAvatarBytes   int64
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("mysql.dsn", "root:root@tcp(localhost:3306)/aidevclub?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.secret", "dev-secret-change-me")
	v.SetDefault("token.access_ttl", "15m")
	v.SetDefault("token.refresh_ttl", "720h")
	v.SetDefault("ratelimit.per_minute", 10)
	v.SetDefault("avatar.dir", "storage/avatars")
	v.SetDefault("avatar.default_url", "/static/avatars/default.png")
	v.SetDefault("avatar.max_bytes", int64(2<<20))

	v.AutomaticEnv()
	v.SetEnvPrefix("AIDEVCLUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	accessTTL, err := time.ParseDuration(v.GetString("token.access_ttl"))
	if err != nil {
		return nil, err
	}
	refreshTTL, err := time.ParseDuration(v.GetString("token.refresh_ttl"))
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPAddr:         v.GetString("http.addr"),
		MySQLDSN:         v.GetString("mysql.dsn"),
		RedisAddr:        v.GetString("redis.addr"),
		RedisPassword:    v.GetString("redis.password"),
		RedisDB:          v.GetInt("redis.db"),
		JWTSecret:        v.GetString("jwt.secret"),
		AccessTokenTTL:   accessTTL,
		RefreshTokenTTL:  refreshTTL,
		RateLimitPerMin:  v.GetInt("ratelimit.per_minute"),
		AvatarDir:        v.GetString("avatar.dir"),
		DefaultAvatarURL: v.GetString("avatar.default_url"),
		MaxAvatarBytes:   v.GetInt64("avatar.max_bytes"),
	}, nil
}
```

- [ ] **步骤 3：编写日志初始化**

`internal/platform/logger.go`：

```go
package platform

import (
	"log/slog"
	"os"
)

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
```

- [ ] **步骤 4：编写最小 main + 健康检查**

`cmd/server/main.go`：

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
)

func main() {
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := platform.NewLogger()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	logger.Info("server starting", "addr", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		logger.Error("server exited", "err", err)
	}
}
```

- [ ] **步骤 5：拉取依赖并编译**

```bash
go mod tidy
go build ./...
```

预期：编译通过。

- [ ] **步骤 6：运行确认健康检查**

```bash
go run ./cmd/server &
sleep 2
curl -s http://localhost:8080/healthz
```

预期：返回 `{"status":"ok"}`。然后停掉进程。

- [ ] **步骤 7：Commit**

```bash
git add go.mod go.sum cmd internal
git commit -m "chore: 初始化 Go 服务骨架（配置/日志/健康检查）"
```

---

### 任务 2：Docker Compose、数据库与 Redis 连接、User 模型

**文件：**
- 创建：`docker-compose.yml`
- 创建：`internal/platform/database.go`
- 创建：`internal/platform/redis.go`
- 创建：`internal/model/user.go`
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：编写 Docker Compose（MySQL 8 + Redis）**

`docker-compose.yml`：

```yaml
services:
  mysql:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: aidevclub
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
  redis:
    image: redis:7
    ports:
      - "6379:6379"

volumes:
  mysql_data:
```

- [ ] **步骤 2：编写数据库连接与 Redis 客户端**

`internal/platform/database.go`：

```go
package platform

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
```

`internal/platform/redis.go`：

```go
package platform

import "github.com/redis/go-redis/v9"

func OpenRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
```

- [ ] **步骤 3：编写 User 模型**

`internal/model/user.go`：

```go
package model

import "time"

type User struct {
	ID           uint       `gorm:"primaryKey"`
	Email        string     `gorm:"size:191;uniqueIndex;not null"`
	PasswordHash string     `gorm:"size:255;not null"`
	Nickname     string     `gorm:"size:64;not null"`
	AvatarURL    string     `gorm:"size:255;not null"`
	Bio          string     `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time `gorm:"index"`
}
```

- [ ] **步骤 4：在 main 中装配 DB/Redis 并迁移**

`cmd/server/main.go`（在健康检查路由前新增）：

```go
db, err := platform.OpenMySQL(cfg.MySQLDSN)
if err != nil {
	logger.Error("open mysql", "err", err)
	return
}
if err := db.AutoMigrate(&model.User{}); err != nil {
	logger.Error("migrate", "err", err)
	return
}
rdb := platform.OpenRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
```

并在 import 中加入 `"aidevclub/internal/model"`。

- [ ] **步骤 5：编译确认**

```bash
go mod tidy
go build ./...
```

预期：编译通过。

- [ ] **步骤 6：Commit**

```bash
git add docker-compose.yml internal cmd
git commit -m "chore: 接入 MySQL/Redis 并新增 User 模型"
```

---

### 任务 3：统一响应与错误处理中间件

**文件：**
- 创建：`internal/platform/response.go`
- 创建：`internal/platform/errors.go`
- 创建：`internal/platform/middleware.go`
- 创建：`internal/platform/response_test.go`

- [ ] **步骤 1：编写失败测试**

`internal/platform/response_test.go`：

```go
package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOKWritesUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) { OK(c, gin.H{"id": 1}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != `{"code":0,"message":"ok","data":{"id":1}}` {
		t.Fatalf("body = %s", got)
	}
}

func TestErrorMiddlewareMapsBizError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RecoverMiddleware())
	r.GET("/err", func(c *gin.Context) {
		_ = c.Error(NewBizError(http.StatusConflict, 40901, "邮箱已存在"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/err", nil))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if got := w.Body.String(); got != `{"code":40901,"message":"邮箱已存在"}` {
		t.Fatalf("body = %s", got)
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

```bash
go test ./internal/platform/ -run TestOK -v
```

预期：编译失败（`OK`、`RecoverMiddleware`、`NewBizError` 未定义）。

- [ ] **步骤 3：编写实现**

`internal/platform/response.go`：

```go
package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, status, code int, message string) {
	c.JSON(status, Response{Code: code, Message: message})
}
```

`internal/platform/errors.go`：

```go
package platform

type BizError struct {
	Status  int
	Code    int
	Message string
}

func (e *BizError) Error() string { return e.Message }

func NewBizError(status, code int, message string) *BizError {
	return &BizError{Status: status, Code: code, Message: message}
}
```

`internal/platform/middleware.go`：

```go
package platform

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		for _, e := range c.Errors {
			if be, ok := e.Err.(*BizError); ok {
				Fail(c, be.Status, be.Code, be.Message)
				return
			}
		}
		Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
	}
}
```

- [ ] **步骤 4：运行测试确认通过**

```bash
go test ./internal/platform/ -v
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/platform
git commit -m "feat: 统一响应格式与错误处理中间件"
```

---

### 任务 4：共享测试辅助 + User Repository（CRUD + 软删除）

**文件：**
- 创建：`internal/testutil/testutil.go`
- 创建：`internal/repo/user.go`
- 创建：`internal/repo/user_test.go`

- [ ] **步骤 1：编写共享测试辅助**

`internal/testutil/testutil.go`：

```go
package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

// NewTestDB 返回内存 SQLite，单连接避免 "no such table" 竞态。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// NewTestRedis 返回内存 Redis（miniredis）。
func NewTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}
```

- [ ] **步骤 2：编写失败测试**

`internal/repo/user_test.go`：

```go
package repo

import (
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/testutil"
)

func TestUserRepoCreateAndFindByEmail(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))

	u := &model.User{Email: "a@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}

	got, err := r.FindByEmail("a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("id = %d, want %d", got.ID, u.ID)
	}
}

func TestUserRepoSoftDelete(t *testing.T) {
	r := NewUserRepo(testutil.NewTestDB(t))

	u := &model.User{Email: "b@example.com", PasswordHash: "x", Nickname: "用户_abc", AvatarURL: "/static/avatars/default.png"}
	if err := r.Create(u); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.FindByEmail("b@example.com"); err != gorm.ErrRecordNotFound {
		t.Fatalf("err = %v, want ErrRecordNotFound", err)
	}
}
```

- [ ] **步骤 3：运行测试确认失败**

```bash
go test ./internal/repo/ -v
```

预期：编译失败（`NewUserRepo` 未定义）。

- [ ] **步骤 4：编写实现**

`internal/repo/user.go`：

```go
package repo

import (
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *model.User) error { return r.db.Create(u).Error }

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Update(u *model.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}
```

- [ ] **步骤 5：运行测试确认通过**

```bash
go test ./internal/repo/ -v
```

预期：PASS。

- [ ] **步骤 6：Commit**

```bash
git add internal/testutil internal/repo
git commit -m "feat: 共享测试辅助与用户仓库（CRUD + 软删除）"
```

---

### 任务 5：密码哈希、JWT 与 Refresh Token 存储

> JWT 纯函数放在 `platform` 包，避免 platform↔service 循环依赖；密码哈希留在 service。

**文件：**
- 创建：`internal/platform/jwt.go`
- 创建：`internal/platform/jwt_test.go`
- 创建：`internal/repo/token.go`
- 创建：`internal/repo/token_test.go`
- 创建：`internal/service/auth.go`（本任务只含密码哈希函数）
- 创建：`internal/service/auth_test.go`

- [ ] **步骤 1：编写 JWT 失败测试**

`internal/platform/jwt_test.go`：

```go
package platform

import (
	"testing"
	"time"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	tok, err := GenerateAccessToken("s", time.Minute, 42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseAccessToken("s", tok)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
}

func TestParseAccessTokenRejectsBadSecret(t *testing.T) {
	tok, _ := GenerateAccessToken("a", time.Minute, 1)
	if _, err := ParseAccessToken("b", tok); err == nil {
		t.Fatal("bad secret accepted")
	}
}
```

- [ ] **步骤 2：编写密码哈希失败测试**

`internal/service/auth_test.go`：

```go
package service

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	h, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if h == "secret123" {
		t.Fatal("password not hashed")
	}
	if err := checkPassword(h, "secret123"); err != nil {
		t.Fatalf("check = %v, want nil", err)
	}
	if err := checkPassword(h, "wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
}
```

- [ ] **步骤 3：运行测试确认失败**

```bash
go test ./internal/platform/ ./internal/service/ -v
```

预期：编译失败（`GenerateAccessToken`、`hashPassword` 未定义）。

- [ ] **步骤 4：编写实现**

`internal/platform/jwt.go`：

```go
package platform

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	UserID uint `json:"uid"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret string, ttl time.Duration, userID uint) (string, error) {
	now := time.Now()
	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

var ErrInvalidToken = errors.New("invalid token")

func ParseAccessToken(secret, token string) (uint, error) {
	t, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	claims, ok := t.Claims.(*accessClaims)
	if !ok || !t.Valid {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
```

`internal/service/auth.go`（本任务片段）：

```go
package service

import "golang.org/x/crypto/bcrypt"

func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
```

- [ ] **步骤 5：运行测试确认通过**

```bash
go test ./internal/platform/ ./internal/service/ -v
```

预期：PASS。

- [ ] **步骤 6：编写 Refresh Token 仓库失败测试**

`internal/repo/token_test.go`：

```go
package repo

import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/testutil"
)

func TestTokenRepoIssueValidateRevoke(t *testing.T) {
	ctx := context.Background()
	r := NewTokenRepo(testutil.NewTestRedis(t), time.Hour)

	tok, err := r.Issue(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}

	uid, err := r.Validate(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 7 {
		t.Fatalf("uid = %d, want 7", uid)
	}

	if err := r.Revoke(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Validate(ctx, tok); err == nil {
		t.Fatal("token still valid after revoke")
	}
}

func TestTokenRepoRevokeAllForUser(t *testing.T) {
	ctx := context.Background()
	r := NewTokenRepo(testutil.NewTestRedis(t), time.Hour)

	t1, _ := r.Issue(ctx, 9)
	t2, _ := r.Issue(ctx, 9)
	if err := r.RevokeAllForUser(ctx, 9); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Validate(ctx, t1); err == nil {
		t.Fatal("t1 still valid")
	}
	if _, err := r.Validate(ctx, t2); err == nil {
		t.Fatal("t2 still valid")
	}
}
```

- [ ] **步骤 7：运行测试确认失败**

```bash
go test ./internal/repo/ -run TestTokenRepo -v
```

预期：编译失败（`NewTokenRepo` 未定义）。

- [ ] **步骤 8：编写实现**

`internal/repo/token.go`：

```go
package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTokenNotFound = errors.New("refresh token not found")

type TokenRepo struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewTokenRepo(rdb *redis.Client, ttl time.Duration) *TokenRepo {
	return &TokenRepo{rdb: rdb, ttl: ttl}
}

func refreshKey(token string) string { return "refresh:" + token }
func sessionsKey(userID uint) string { return "user_sessions:" + strconv.FormatUint(uint64(userID), 10) }

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *TokenRepo) Issue(ctx context.Context, userID uint) (string, error) {
	tok, err := newToken()
	if err != nil {
		return "", err
	}
	pipe := r.rdb.TxPipeline()
	pipe.Set(ctx, refreshKey(tok), userID, r.ttl)
	pipe.SAdd(ctx, sessionsKey(userID), tok)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return tok, nil
}

func (r *TokenRepo) Validate(ctx context.Context, token string) (uint, error) {
	v, err := r.rdb.Get(ctx, refreshKey(token)).Result()
	if err != nil {
		return 0, ErrTokenNotFound
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, ErrTokenNotFound
	}
	return uint(id), nil
}

func (r *TokenRepo) Revoke(ctx context.Context, token string) error {
	uid, err := r.Validate(ctx, token)
	if err != nil {
		return err
	}
	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, refreshKey(token))
	pipe.SRem(ctx, sessionsKey(uid), token)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *TokenRepo) RevokeAllForUser(ctx context.Context, userID uint) error {
	tokens, err := r.rdb.SMembers(ctx, sessionsKey(userID)).Result()
	if err != nil {
		return err
	}
	pipe := r.rdb.TxPipeline()
	for _, tok := range tokens {
		pipe.Del(ctx, refreshKey(tok))
	}
	pipe.Del(ctx, sessionsKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}
```

- [ ] **步骤 9：运行测试确认通过**

```bash
go test ./internal/repo/ -v
```

预期：PASS。

- [ ] **步骤 10：Commit**

```bash
git add internal/platform internal/service internal/repo
git commit -m "feat: 密码哈希、JWT 与 Refresh Token 存储"
```

---

### 任务 6：Auth Service —— 注册与登录

**文件：**
- 修改：`internal/service/auth.go`（新增 AuthService 与注册/登录方法）
- 修改：`internal/service/auth_test.go`

- [ ] **步骤 1：编写失败测试**

`internal/service/auth_test.go` 追加（并补 import）：

```go
import (
	"context"
	"testing"
	"time"

	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	cfg := &platform.Config{
		DefaultAvatarURL: "/static/avatars/default.png",
		JWTSecret:        "s",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
	}
	return NewAuthService(
		repo.NewUserRepo(testutil.NewTestDB(t)),
		repo.NewTokenRepo(testutil.NewTestRedis(t), time.Hour),
		cfg,
	)
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err == nil {
		t.Fatal("duplicate email accepted")
	}

	out, err := svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("empty tokens")
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

```bash
go test ./internal/service/ -v
```

预期：编译失败（`AuthService`、`RegisterInput` 等未定义）。

- [ ] **步骤 3：编写实现**

`internal/service/auth.go` 追加：

```go
import (
	"context"
	"math/rand"
	"net/http"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrEmailExists   = platform.NewBizError(http.StatusConflict, 40901, "邮箱已存在")
	ErrUserNotFound  = platform.NewBizError(http.StatusNotFound, 40401, "用户不存在")
	ErrBadCredential = platform.NewBizError(http.StatusBadRequest, 40001, "邮箱或密码错误")
	ErrInvalidParam  = platform.NewBizError(http.StatusBadRequest, 40001, "参数错误")
)

type RegisterInput struct {
	Email    string
	Password string
	Nickname string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AuthService struct {
	users  *repo.UserRepo
	tokens *repo.TokenRepo
	cfg    *platform.Config
}

func NewAuthService(users *repo.UserRepo, tokens *repo.TokenRepo, cfg *platform.Config) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

func defaultNickname() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "用户_" + string(b)
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) error {
	if in.Email == "" || in.Password == "" {
		return ErrInvalidParam
	}
	if _, err := s.users.FindByEmail(in.Email); err == nil {
		return ErrEmailExists
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return err
	}
	nickname := in.Nickname
	if nickname == "" {
		nickname = defaultNickname()
	}
	return s.users.Create(&model.User{
		Email:        in.Email,
		PasswordHash: hash,
		Nickname:     nickname,
		AvatarURL:    s.cfg.DefaultAvatarURL,
	})
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	if in.Email == "" || in.Password == "" {
		return nil, ErrInvalidParam
	}
	u, err := s.users.FindByEmail(in.Email)
	if err != nil {
		return nil, ErrBadCredential
	}
	if err := checkPassword(u.PasswordHash, in.Password); err != nil {
		return nil, ErrBadCredential
	}
	return s.issueTokens(ctx, u.ID)
}

func (s *AuthService) issueTokens(ctx context.Context, userID uint) (*TokenPair, error) {
	access, err := platform.GenerateAccessToken(s.cfg.JWTSecret, s.cfg.AccessTokenTTL, userID)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.Issue(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
```

- [ ] **步骤 4：运行测试确认通过**

```bash
go test ./internal/service/ -v
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/service
git commit -m "feat: 用户注册与登录服务"
```

---

### 任务 7：Auth Service —— 刷新与登出

**文件：**
- 修改：`internal/service/auth.go`（新增 Refresh、Logout）
- 修改：`internal/service/auth_test.go`

- [ ] **步骤 1：编写失败测试**

`internal/service/auth_test.go` 追加：

```go
func TestRefreshRotatesToken(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	if err := svc.Register(ctx, RegisterInput{Email: "a@example.com", Password: "secret123"}); err != nil {
		t.Fatal(err)
	}
	pair, err := svc.Login(ctx, LoginInput{Email: "a@example.com", Password: "secret123"})
	if err != nil {
		t.Fatal(err)
	}

	newPair, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token not rotated")
	}
	// 旧 refresh 已作废
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("old refresh token still valid")
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	_ = svc.Register(ctx, RegisterInput{Email: "b@example.com", Password: "secret123"})
	pair, _ := svc.Login(ctx, LoginInput{Email: "b@example.com", Password: "secret123"})

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("refresh token still valid after logout")
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

预期：编译失败（`Refresh`、`Logout` 未定义）。

- [ ] **步骤 3：编写实现**

`internal/service/auth.go` 追加：

```go
func (s *AuthService) Refresh(ctx context.Context, refresh string) (*TokenPair, error) {
	userID, err := s.tokens.Validate(ctx, refresh)
	if err != nil {
		return nil, ErrInvalidParam
	}
	if err := s.tokens.Revoke(ctx, refresh); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, userID)
}

func (s *AuthService) Logout(ctx context.Context, refresh string) error {
	return s.tokens.Revoke(ctx, refresh)
}
```

- [ ] **步骤 4：运行测试确认通过**

```bash
go test ./internal/service/ -v
```

预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add internal/service
git commit -m "feat: Token 刷新与登出"
```

---

### 任务 8：鉴权中间件、User Service 与用户资料接口

**文件：**
- 创建：`internal/platform/auth_middleware.go`
- 创建：`internal/platform/auth_middleware_test.go`
- 创建：`internal/service/user.go`
- 创建：`internal/service/user_test.go`
- 创建：`internal/handler/errors.go`
- 创建：`internal/handler/auth.go`
- 创建：`internal/handler/user.go`
- 创建：`internal/handler/handler_test.go`
- 修改：`cmd/server/main.go`

- [ ] **步骤 1：编写鉴权中间件失败测试**

`internal/platform/auth_middleware_test.go`：

```go
package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", AuthMiddleware("s"), func(c *gin.Context) {
		c.JSON(200, gin.H{"uid": c.GetUint("user_id")})
	})

	// 无 token → 401
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d, want 401", w.Code)
	}

	// 有效 token → 200，user_id 注入
	tok, _ := GenerateAccessToken("s", time.Minute, 42)
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("valid-token status = %d, want 200", w.Code)
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

```bash
go test ./internal/platform/ -run TestAuthMiddleware -v
```

预期：编译失败（`AuthMiddleware` 未定义）。

- [ ] **步骤 3：编写鉴权中间件实现**

`internal/platform/auth_middleware.go`：

```go
package platform

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(h, prefix) {
			Fail(c, http.StatusUnauthorized, 40101, "未认证")
			c.Abort()
			return
		}
		uid, err := ParseAccessToken(secret, strings.TrimPrefix(h, prefix))
		if err != nil {
			Fail(c, http.StatusUnauthorized, 40101, "未认证")
			c.Abort()
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}
```

- [ ] **步骤 4：编写 User Service 失败测试**

`internal/service/user_test.go`：

```go
package service

import (
	"context"
	"testing"
)

func TestUserProfileUpdateChangePasswordDelete(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(t)

	_ = svc.Register(ctx, RegisterInput{Email: "u@example.com", Password: "secret123"})
	u, _ := svc.users.FindByEmail("u@example.com")

	userSvc := NewUserService(svc.users, svc.tokens, svc.cfg)

	if err := userSvc.UpdateProfile(ctx, u.ID, UpdateProfileInput{Nickname: "新昵称", Bio: "你好"}); err != nil {
		t.Fatal(err)
	}
	got, _ := userSvc.Get(ctx, u.ID)
	if got.Nickname != "新昵称" {
		t.Fatalf("nickname = %s", got.Nickname)
	}

	if err := userSvc.ChangePassword(ctx, u.ID, "newpass123"); err != nil {
		t.Fatal(err)
	}

	if err := userSvc.Delete(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := userSvc.Get(ctx, u.ID); err == nil {
		t.Fatal("user still visible after delete")
	}
}
```

- [ ] **步骤 5：编写 User Service 实现**

`internal/service/user.go`：

```go
package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

type UpdateProfileInput struct {
	Nickname  string
	AvatarURL string
	Bio       string
}

type UserService struct {
	users  *repo.UserRepo
	tokens *repo.TokenRepo
	cfg    *platform.Config
}

func NewUserService(users *repo.UserRepo, tokens *repo.TokenRepo, cfg *platform.Config) *UserService {
	return &UserService{users: users, tokens: tokens, cfg: cfg}
}

func (s *UserService) AvatarDir() string       { return s.cfg.AvatarDir }
func (s *UserService) MaxAvatarBytes() int64   { return s.cfg.MaxAvatarBytes }

func (s *UserService) Get(ctx context.Context, id uint) (*model.User, error) {
	u, err := s.users.FindByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id uint, in UpdateProfileInput) error {
	u, err := s.users.FindByID(id)
	if err != nil {
		return ErrUserNotFound
	}
	if in.Nickname != "" {
		u.Nickname = in.Nickname
	}
	if in.AvatarURL != "" {
		u.AvatarURL = in.AvatarURL
	}
	if in.Bio != "" {
		u.Bio = in.Bio
	}
	return s.users.Update(u)
}

func (s *UserService) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	u, err := s.users.FindByID(id)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := s.users.Update(u); err != nil {
		return err
	}
	return s.tokens.RevokeAllForUser(ctx, id)
}

func (s *UserService) Delete(ctx context.Context, id uint) error {
	if err := s.users.Delete(id); err != nil {
		return err
	}
	return s.tokens.RevokeAllForUser(ctx, id)
}
```

- [ ] **步骤 6：运行 service/platform 测试确认通过**

```bash
go test ./internal/service/ ./internal/platform/ -v
```

预期：PASS。

- [ ] **步骤 7：编写 handler 错误映射辅助**

`internal/handler/errors.go`：

```go
package handler

import (
	"errors"
	"net/http"

	"aidevclub/internal/platform"
)

func errStatus(err error) int {
	var be *platform.BizError
	if errors.As(err, &be) {
		return be.Status
	}
	return http.StatusInternalServerError
}

func errCode(err error) int {
	var be *platform.BizError
	if errors.As(err, &be) {
		return be.Code
	}
	return 50000
}
```

- [ ] **步骤 8：编写 handler 实现**

`internal/handler/auth.go`：

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

func (h *AuthHandler) Register(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.Register(c.Request.Context(), service.RegisterInput{Email: in.Email, Password: in.Password, Nickname: in.Nickname}); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), service.LoginInput{Email: in.Email, Password: in.Password})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), in.RefreshToken)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), in.RefreshToken); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}
```

`internal/handler/user.go`：

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc: svc} }

func (h *UserHandler) Me(c *gin.Context) {
	u, err := h.svc.Get(c.Request.Context(), c.GetUint("user_id"))
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname, "avatar_url": u.AvatarURL, "bio": u.Bio})
}

func (h *UserHandler) Update(c *gin.Context) {
	var in struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
		Bio       string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), c.GetUint("user_id"), service.UpdateProfileInput{Nickname: in.Nickname, AvatarURL: in.AvatarURL, Bio: in.Bio}); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var in struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Password == "" {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), c.GetUint("user_id"), in.Password); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.GetUint("user_id")); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}
```

- [ ] **步骤 9：编写 handler 集成测试**

`internal/handler/handler_test.go`：

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/service"
	"aidevclub/internal/testutil"
)

func setupRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &platform.Config{
		DefaultAvatarURL: "/static/avatars/default.png",
		JWTSecret:        "s",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
		RateLimitPerMin:  1000,
	}
	users := repo.NewUserRepo(testutil.NewTestDB(t))
	tokens := repo.NewTokenRepo(testutil.NewTestRedis(t), time.Hour)
	authSvc := service.NewAuthService(users, tokens, cfg)
	userSvc := service.NewUserService(users, tokens, cfg)

	r := gin.New()
	r.Use(platform.RecoverMiddleware())
	ah := NewAuthHandler(authSvc)
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", ah.Register)
	auth.POST("/login", ah.Login)
	auth.POST("/refresh", ah.Refresh)
	auth.POST("/logout", ah.Logout)

	uh := NewUserHandler(userSvc)
	me := r.Group("/api/v1/users", platform.AuthMiddleware(cfg.JWTSecret))
	me.GET("/me", uh.Me)
	me.PATCH("/me", uh.Update)
	me.PUT("/me/password", uh.ChangePassword)
	me.DELETE("/me", uh.Delete)
	return r
}

func TestRegisterLoginMe(t *testing.T) {
	r := setupRouter(t)

	body, _ := json.Marshal(map[string]string{"email": "a@example.com", "password": "secret123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register status = %d, body %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Data.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me status = %d, body %s", w.Code, w.Body.String())
	}
}
```

- [ ] **步骤 10：运行测试确认通过**

```bash
go test ./internal/handler/ -v
```

预期：PASS。

- [ ] **步骤 11：在 main 中装配路由**

`cmd/server/main.go` 用与 `setupRouter` 相同的装配逻辑（但不限流、连接真实 DB/Redis）构建路由，并注册 `/api/v1/auth/*` 与 `/api/v1/users/*`。

- [ ] **步骤 12：Commit**

```bash
git add internal cmd
git commit -m "feat: 鉴权中间件与用户资料接口"
```

---

### 任务 9：头像上传

**文件：**
- 修改：`internal/handler/user.go`（新增 UploadAvatar）
- 修改：`cmd/server/main.go`（静态路由 + 上传路由）

- [ ] **步骤 1：编写失败测试**

`internal/handler/handler_test.go` 追加：

```go
import (
	"mime/multipart"
	// ... 已有 import 不变
)

func validToken() string {
	tok, _ := platform.GenerateAccessToken("s", time.Minute, 1)
	return tok
}

func TestUploadAvatar(t *testing.T) {
	r := setupRouter(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "avatar.png")
	_, _ = fw.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3})
	_ = mw.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/users/me/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+validToken())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}
```

> 注意：`setupRouter` 需要先注册 `/api/v1/users/me/avatar` 路由（步骤 2 中一并完成）。

- [ ] **步骤 2：运行测试确认失败**

预期：编译失败（`UploadAvatar` 未定义）。

- [ ] **步骤 3：编写实现**

`internal/handler/user.go` 追加：

```go
import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		platform.Fail(c, http.StatusBadRequest, 40001, "不支持的图片格式")
		return
	}
	if file.Size > h.svc.MaxAvatarBytes() {
		platform.Fail(c, http.StatusBadRequest, 40001, "图片过大")
		return
	}
	if err := os.MkdirAll(h.svc.AvatarDir(), 0o755); err != nil {
		platform.Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
		return
	}
	name := randomHex(16) + ext
	if err := c.SaveUploadedFile(file, filepath.Join(h.svc.AvatarDir(), name)); err != nil {
		platform.Fail(c, http.StatusInternalServerError, 50000, "服务器内部错误")
		return
	}
	url := "/static/avatars/" + name
	if err := h.svc.UpdateProfile(c.Request.Context(), c.GetUint("user_id"), service.UpdateProfileInput{AvatarURL: url}); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"avatar_url": url})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **步骤 4：在 setupRouter 与 main 中注册路由 + 静态服务**

```go
me.POST("/me/avatar", uh.UploadAvatar)
r.Static("/static/avatars", cfg.AvatarDir)
```

- [ ] **步骤 5：运行测试确认通过**

```bash
go test ./internal/handler/ -v
```

预期：PASS。

- [ ] **步骤 6：Commit**

```bash
git add internal cmd
git commit -m "feat: 头像上传"
```

---

### 任务 10：注册/登录限流

**文件：**
- 创建：`internal/platform/ratelimit.go`
- 创建：`internal/platform/ratelimit_test.go`
- 修改：`cmd/server/main.go`（及 handler 测试的 setupRouter，接入限流）

- [ ] **步骤 1：编写失败测试**

`internal/platform/ratelimit_test.go`：

```go
package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/testutil"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testutil.NewTestRedis(t)
	r := gin.New()
	r.Use(RateLimitMiddleware(rdb, 2, time.Minute))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	do := func() int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		return w.Code
	}
	if do() != 200 || do() != 200 {
		t.Fatal("first two requests should pass")
	}
	if do() != http.StatusTooManyRequests {
		t.Fatal("third request should be rate limited")
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

预期：编译失败（`RateLimitMiddleware` 未定义）。

- [ ] **步骤 3：编写实现**

`internal/platform/ratelimit.go`：

```go
package platform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s:%s", c.FullPath(), c.ClientIP())
		ctx := context.Background()
		n, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if n == 1 {
			_ = rdb.Expire(ctx, key, window)
		}
		if n > int64(limit) {
			Fail(c, http.StatusTooManyRequests, 42901, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **步骤 4：运行测试确认通过**

```bash
go test ./internal/platform/ -v
```

预期：PASS。

- [ ] **步骤 5：在注册/登录路由接入限流**

`cmd/server/main.go`（及 handler 测试的 setupRouter）：

```go
rl := platform.RateLimitMiddleware(rdb, cfg.RateLimitPerMin, time.Minute)
auth.POST("/register", rl, ah.Register)
auth.POST("/login", rl, ah.Login)
```

- [ ] **步骤 6：运行全量测试与编译**

```bash
go test ./... -v
go build ./...
```

预期：全部 PASS，编译通过。

- [ ] **步骤 7：Commit**

```bash
git add internal cmd
git commit -m "feat: 注册/登录接口限流"
```

---

## 自检记录

- **规格覆盖度**：设计文档第 2.1 节所列能力（骨架、健康检查、配置、日志、错误处理、注册/登录/登出/刷新/资料/改密/注销、头像、限流）均已有对应任务。非目标项（状态机、邮箱验证、文章/资源）已排除。
- **占位符扫描**：无「待定/TODO/后续实现」；每个代码步骤均有代码块。
- **类型一致性**：`AuthService`/`UserService` 构造签名、`TokenRepo` 的 `Issue/Validate/Revoke/RevokeAllForUser`、`platform.GenerateAccessToken/ParseAccessToken`、`handler.errStatus/errCode` 在各任务间命名一致。
- **依赖一致性**：JWT 下沉至 `platform` 消除 platform↔service 循环依赖；service 依赖 platform/repo/model，handler 依赖 platform/service，符合依赖图。
