package testutil

import (
	"context"
	"os"
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

// 测试后端与生产隔离：独立 MySQL 库 aidevclub_test 与 Redis DB 15。
var (
	testMySQLDSN  = envOr("AIDEVCLUB_TEST_MYSQL_DSN", "root:root@tcp(localhost:3306)/aidevclub_test?charset=utf8mb4&parseTime=True&loc=Local")
	testRedisAddr = envOr("AIDEVCLUB_TEST_REDIS_ADDR", "localhost:16379")
)

const testRedisDB = 15

// ensureTestDB 用无库名的引导连接创建测试库（幂等，不依赖 docker init 脚本时序）。
func ensureTestDB(t *testing.T) {
	t.Helper()
	bootDSN := "root:root@tcp(localhost:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	b, err := gorm.Open(mysql.Open(bootDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect mysql bootstrap: %v", err)
	}
	if err := b.Exec("CREATE DATABASE IF NOT EXISTS aidevclub_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		t.Fatalf("create test db: %v", err)
	}
}

// NewTestDB 连接测试 MySQL 并重置 users 表，测试结束后清理。
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

// NewTestRedis 连接测试 Redis（DB 15），测试结束后 FlushDB。
func NewTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: testRedisAddr, DB: testRedisDB})
	t.Cleanup(func() {
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	})
	return rdb
}
