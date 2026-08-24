package platform

import (
	"log/slog"
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
	v.SetDefault("redis.addr", "localhost:16379")
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

	cfg := &Config{
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
	}

	// 生产环境若忘记设置 AIDEVCLUB_JWT_SECRET，会使用众所周知的默认值，
	// 任何人都可伪造 JWT。这里只告警、不返回错误，避免破坏开发环境。
	if cfg.JWTSecret == "dev-secret-change-me" {
		slog.Warn("JWT secret 仍为默认值，生产环境请务必通过 AIDEVCLUB_JWT_SECRET 覆盖",
			"jwt_secret", "dev-secret-change-me")
	}

	return cfg, nil
}
