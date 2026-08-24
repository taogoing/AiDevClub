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
	AvatarDir            string
	DefaultAvatarURL     string
	MaxAvatarBytes       int64
	DefaultPageSize      int
	MaxPageSize          int
	HotCacheTTL          time.Duration
	ArticleImageDir      string
	MaxArticleImageBytes int64
	SkillZipDir         string
	McpServerZipDir     string
	MaxResourceZipBytes int64
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
	v.SetDefault("article.hot_cache_ttl", "60s")
	v.SetDefault("article.page_size_default", 20)
	v.SetDefault("article.page_size_max", 50)
	v.SetDefault("article_image.dir", "storage/articles")
	v.SetDefault("article_image.max_bytes", int64(5<<20))
	v.SetDefault("skill_zip.dir", "storage/skills")
	v.SetDefault("mcp_server_zip.dir", "storage/mcp_servers")
	v.SetDefault("resource_zip.max_bytes", int64(50<<20))

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
	hotCacheTTL, err := time.ParseDuration(v.GetString("article.hot_cache_ttl"))
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
		AvatarDir:            v.GetString("avatar.dir"),
		DefaultAvatarURL:     v.GetString("avatar.default_url"),
		MaxAvatarBytes:       v.GetInt64("avatar.max_bytes"),
		DefaultPageSize:      v.GetInt("article.page_size_default"),
		MaxPageSize:          v.GetInt("article.page_size_max"),
		HotCacheTTL:          hotCacheTTL,
		ArticleImageDir:      v.GetString("article_image.dir"),
		MaxArticleImageBytes: v.GetInt64("article_image.max_bytes"),
		SkillZipDir:         v.GetString("skill_zip.dir"),
		McpServerZipDir:     v.GetString("mcp_server_zip.dir"),
		MaxResourceZipBytes: v.GetInt64("resource_zip.max_bytes"),
	}

	// 生产环境若忘记设置 AIDEVCLUB_JWT_SECRET，会使用众所周知的默认值，
	// 任何人都可伪造 JWT。这里只告警、不返回错误，避免破坏开发环境。
	if cfg.JWTSecret == "dev-secret-change-me" {
		slog.Warn("JWT secret 仍为默认值，生产环境请务必通过 AIDEVCLUB_JWT_SECRET 覆盖",
			"jwt_secret", "dev-secret-change-me")
	}

	return cfg, nil
}
