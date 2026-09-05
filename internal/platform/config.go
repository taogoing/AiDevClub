package platform

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr             string
	MySQLDSN             string
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	JWTSecret            string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	RateLimitPerMin      int
	RankingSingleflight  bool
	MCPAddr              string
	PublicBaseURL        string
	MCPAllowedOrigins    []string
	MCPRateLimitPerMin   int
	MCPMaxBodyBytes      int64
	MCPRequestTimeout    time.Duration
	AvatarDir            string
	DefaultAvatarURL     string
	MaxAvatarBytes       int64
	DefaultPageSize      int
	MaxPageSize          int
	ArticleImageDir      string
	MaxArticleImageBytes int64
	AdminEmails          []string
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
	v.SetDefault("ranking.singleflight", true)
	v.SetDefault("mcp.addr", ":8081")
	v.SetDefault("public.base_url", "http://localhost:5173")
	v.SetDefault("mcp.allowed_origins", "")
	v.SetDefault("mcp.ratelimit_per_minute", 60)
	v.SetDefault("mcp.max_body_bytes", int64(1<<20))
	v.SetDefault("mcp.request_timeout", "30s")
	v.SetDefault("avatar.dir", "storage/avatars")
	v.SetDefault("avatar.default_url", "/static/avatars/default.png")
	v.SetDefault("avatar.max_bytes", int64(2<<20))
	v.SetDefault("article.page_size_default", 20)
	v.SetDefault("article.page_size_max", 50)
	v.SetDefault("article_image.dir", "storage/articles")
	v.SetDefault("article_image.max_bytes", int64(5<<20))
	v.SetDefault("admin.emails", "")

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
	mcpRequestTimeout, err := time.ParseDuration(v.GetString("mcp.request_timeout"))
	if err != nil {
		return nil, err
	}
	publicBaseURL, err := validatePublicBaseURL(v.GetString("public.base_url"))
	if err != nil {
		return nil, err
	}
	mcpAllowedOrigins, err := parseMCPAllowedOrigins(v.GetString("mcp.allowed_origins"))
	if err != nil {
		return nil, err
	}

	var adminEmails []string
	if emails := v.GetString("admin.emails"); emails != "" {
		for _, e := range strings.Split(emails, ",") {
			if trimmed := strings.TrimSpace(e); trimmed != "" {
				adminEmails = append(adminEmails, trimmed)
			}
		}
	}

	cfg := &Config{
		HTTPAddr:             v.GetString("http.addr"),
		MySQLDSN:             v.GetString("mysql.dsn"),
		RedisAddr:            v.GetString("redis.addr"),
		RedisPassword:        v.GetString("redis.password"),
		RedisDB:              v.GetInt("redis.db"),
		JWTSecret:            v.GetString("jwt.secret"),
		AccessTokenTTL:       accessTTL,
		RefreshTokenTTL:      refreshTTL,
		RateLimitPerMin:      v.GetInt("ratelimit.per_minute"),
		RankingSingleflight:  v.GetBool("ranking.singleflight"),
		MCPAddr:              v.GetString("mcp.addr"),
		PublicBaseURL:        publicBaseURL,
		MCPAllowedOrigins:    mcpAllowedOrigins,
		MCPRateLimitPerMin:   v.GetInt("mcp.ratelimit_per_minute"),
		MCPMaxBodyBytes:      v.GetInt64("mcp.max_body_bytes"),
		MCPRequestTimeout:    mcpRequestTimeout,
		AvatarDir:            v.GetString("avatar.dir"),
		DefaultAvatarURL:     v.GetString("avatar.default_url"),
		MaxAvatarBytes:       v.GetInt64("avatar.max_bytes"),
		DefaultPageSize:      v.GetInt("article.page_size_default"),
		MaxPageSize:          v.GetInt("article.page_size_max"),
		ArticleImageDir:      v.GetString("article_image.dir"),
		MaxArticleImageBytes: v.GetInt64("article_image.max_bytes"),
		AdminEmails:          adminEmails,
	}

	// 生产环境若忘记设置 AIDEVCLUB_JWT_SECRET，会使用众所周知的默认值，
	// 任何人都可伪造 JWT。这里只告警、不返回错误，避免破坏开发环境。
	if cfg.JWTSecret == "dev-secret-change-me" {
		slog.Warn("JWT secret 仍为默认值，生产环境请务必通过 AIDEVCLUB_JWT_SECRET 覆盖",
			"jwt_secret", "dev-secret-change-me")
	}

	return cfg, nil
}

func validatePublicBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid public base URL %q", value)
	}
	return value, nil
}

func parseMCPAllowedOrigins(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}

	origins := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(value)
		parsed, err := url.Parse(origin)
		if origin == "" || strings.Contains(origin, "*") || err != nil ||
			parsed.Host == "" || parsed.User != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Path != "" || parsed.ForceQuery || parsed.RawQuery != "" ||
			strings.Contains(origin, "#") || parsed.Fragment != "" {
			return nil, fmt.Errorf("invalid MCP allowed origin %q", origin)
		}
		origins = append(origins, origin)
	}
	return origins, nil
}
