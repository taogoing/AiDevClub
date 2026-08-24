package testutil

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// 测试后端与生产隔离，且各测试包（repo/service/handler）按进程 PID 隔离，
// 使 `go test ./...` 并行运行多个测试二进制（各自独立进程）时互不干扰：
//   - MySQL：每个进程一个独立测试库 aidevclub_test_<pid>；
//   - Redis：每个进程使用派生 DB（pid % 16，假设并发测试包 ≤ 16，当前 3 个包满足）。
//
// 若显式设置 AIDEVCLUB_TEST_MYSQL_DSN，则原样使用并跳过自动建库（由用户自行负责库存在与隔离）；
// AIDEVCLUB_TEST_REDIS_ADDR 同样保留覆盖。
var (
	// customMySQLDSN 为 true 表示用户显式提供了 DSN（不自动建库、不按 PID 隔离）。
	customMySQLDSN = os.Getenv("AIDEVCLUB_TEST_MYSQL_DSN") != ""
)

// perPIDDBName 每个测试进程唯一的库名。
var perPIDDBName = fmt.Sprintf("aidevclub_test_%d", os.Getpid())

var (
	testMySQLDSN = envOr("AIDEVCLUB_TEST_MYSQL_DSN",
		"root:root@tcp(localhost:3306)/"+perPIDDBName+"?charset=utf8mb4&parseTime=True&loc=Local")
	testRedisAddr = envOr("AIDEVCLUB_TEST_REDIS_ADDR", "localhost:16379")
)

// testRedisDB 按进程派生（pid % 16），假设并发测试包 ≤ 16（当前 repo/service/handler 共 3 个）。
// 每个进程的 FlushDB 只清自己的 DB，互不影响。
var testRedisDB = os.Getpid() % 16

var (
	ensureDBOnce sync.Once
	ensureDBErr  error
)

// ensureTestDB 用无库名的引导连接创建测试库（每个进程仅执行一次，幂等）。
// DROP 用于清理 PID 复用时的残留。仅在未显式设置 AIDEVCLUB_TEST_MYSQL_DSN 时生效。
func ensureTestDB(t *testing.T) {
	t.Helper()
	if customMySQLDSN {
		return
	}
	ensureDBOnce.Do(func() {
		ensureDBErr = createPerPIDDB()
	})
	if ensureDBErr != nil {
		t.Fatalf("ensure test db: %v", ensureDBErr)
	}
}

func createPerPIDDB() error {
	bootDSN := "root:root@tcp(localhost:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	b, err := gorm.Open(mysql.Open(bootDSN), &gorm.Config{})
	if err != nil {
		return err
	}
	if err := b.Exec("DROP DATABASE IF EXISTS " + perPIDDBName).Error; err != nil {
		return err
	}
	if err := b.Exec("CREATE DATABASE IF NOT EXISTS " + perPIDDBName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		return err
	}
	return nil
}

// NewTestDB 连接测试 MySQL（按进程隔离的库）并重置 users 表，测试结束后清理。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ensureTestDB(t)
	db, err := gorm.Open(mysql.Open(testMySQLDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Migrator().DropTable(&model.User{}) // 首次可能不存在
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&model.User{})
	})
	return db
}

// NewTestRedis 连接测试 Redis（按进程派生的 DB），测试结束后 FlushDB 只清自己的 DB。
func NewTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: testRedisAddr, DB: testRedisDB})
	t.Cleanup(func() {
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	})
	return rdb
}
