package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"aidevclub/internal/model"
)

func main() {
	dsn := "root:root@tcp(localhost:3306)/aidevclub?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	ctx := context.Background()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 创建测试用户
	users := []model.User{
		{Email: "test1@test.com", Nickname: "张三", PasswordHash: hashPassword("123456"), Bio: "Go 开发者，热爱开源"},
		{Email: "test2@test.com", Nickname: "李四", PasswordHash: hashPassword("123456"), Bio: "前端工程师，Vue/React"},
		{Email: "test3@test.com", Nickname: "王五", PasswordHash: hashPassword("123456"), Bio: "AI/LLM 研究员"},
		{Email: "test4@test.com", Nickname: "赵六", PasswordHash: hashPassword("123456"), Bio: "DevOps 工程师"},
		{Email: "test5@test.com", Nickname: "陈七", PasswordHash: hashPassword("123456"), Bio: "全栈开发者"},
	}

	for i := range users {
		if err := db.WithContext(ctx).Create(&users[i]).Error; err != nil {
			log.Printf("create user %s: %v (may already exist)", users[i].Email, err)
		}
	}

	// 获取用户 ID
	var userIDs []uint
	for _, u := range users {
		var user model.User
		db.Where("email = ?", u.Email).First(&user)
		userIDs = append(userIDs, user.ID)
	}

	// 获取分类
	var categories []model.Category
	db.Find(&categories)
	catMap := make(map[string]uint)
	for _, c := range categories {
		catMap[c.Slug] = c.ID
	}

	// 创建测试标签
	tags := []model.Tag{
		{Name: "Go"},
		{Name: "Vue"},
		{Name: "AI"},
		{Name: "LLM"},
		{Name: "Docker"},
		{Name: "Kubernetes"},
		{Name: "MySQL"},
		{Name: "Redis"},
		{Name: "TypeScript"},
		{Name: "React"},
		{Name: "Python"},
		{Name: "Gin"},
	}

	for i := range tags {
		if err := db.WithContext(ctx).Create(&tags[i]).Error; err != nil {
			log.Printf("create tag %s: %v (may already exist)", tags[i].Name, err)
		}
	}

	// 获取标签 ID
	var tagIDs []uint
	for _, t := range tags {
		var tag model.Tag
		db.Where("name = ?", t.Name).First(&tag)
		tagIDs = append(tagIDs, tag.ID)
	}

	// 创建测试文章
	articles := []struct {
		Title      string
		Summary    string
		Content    string
		CategoryID uint
		AuthorID   uint
		TagIDs     []uint
	}{
		{
			Title:      "Go 1.21 新特性：泛型实战指南",
			Summary:    "深入探讨 Go 1.21 泛型特性，通过实际案例展示如何在项目中使用泛型提高代码复用性。",
			Content:    "# Go 1.21 泛型实战\n\nGo 1.21 引入了泛型特性，这是 Go 语言发展史上的重要里程碑。\n\n## 基本语法\n\n```go\nfunc Min[T constraints.Ordered](a, b T) T {\n    if a < b {\n        return a\n    }\n    return b\n}\n```\n\n## 实际应用场景\n\n### 1. 通用数据结构\n\n```go\ntype Set[T comparable] struct {\n    items map[T]struct{}\n}\n```\n\n### 2. 算法抽象\n\n```go\nfunc Map[T, U any](s []T, f func(T) U) []U {\n    result := make([]U, len(s))\n    for i, v := range s {\n        result[i] = f(v)\n    }\n    return result\n}\n```\n\n## 总结\n\n泛型让 Go 代码更加简洁和可复用。",
			CategoryID: catMap["go"],
			AuthorID:   userIDs[0],
			TagIDs:     []uint{tagIDs[0], tagIDs[11]},
		},
		{
			Title:      "Vue 3 Composition API 最佳实践",
			Summary:    "分享 Vue 3 Composition API 在实际项目中的最佳实践，包括代码组织、状态管理和性能优化。",
			Content:    "# Vue 3 Composition API 最佳实践\n\n## 1. 使用 `<script setup>` 语法糖\n\n```vue\n<script setup lang=\"ts\">\nimport { ref, computed } from 'vue'\n\nconst count = ref(0)\nconst double = computed(() => count.value * 2)\n</script>\n```\n\n## 2. 组合式函数（Composables）\n\n```typescript\n// useCounter.ts\nexport function useCounter(initial = 0) {\n  const count = ref(initial)\n  const increment = () => count.value++\n  const decrement = () => count.value--\n  return { count, increment, decrement }\n}\n```\n\n## 3. 响应式状态管理\n\n使用 Pinia 进行全局状态管理。\n\n## 4. 性能优化\n\n- 使用 `shallowRef` 减少深层响应\n- 合理使用 `computed` 缓存\n- 使用 `v-memo` 指令",
			CategoryID: catMap["frontend"],
			AuthorID:   userIDs[1],
			TagIDs:     []uint{tagIDs[1], tagIDs[8]},
		},
		{
			Title:      "大语言模型微调入门：从 LoRA 到 QLoRA",
			Summary:    "介绍大语言模型微调技术，从 LoRA 到 QLoRA，让普通开发者也能在消费级 GPU 上微调自己的模型。",
			Content:    "# 大语言模型微调入门\n\n## 什么是 LoRA？\n\nLoRA (Low-Rank Adaptation) 是一种参数高效的微调方法。\n\n## LoRA 原理\n\n```python\n# 原始权重矩阵\nW = W_0 + BA\n\n# 其中 B ∈ R^(d×r), A ∈ R^(r×k), r << min(d, k)\n```\n\n## QLoRA：量化 + LoRA\n\nQLoRA 在 LoRA 基础上引入 4-bit 量化，大幅降低显存需求。\n\n## 实践步骤\n\n1. 准备数据集\n2. 选择基座模型\n3. 配置训练参数\n4. 开始微调\n\n## 总结\n\nQLoRA 让 LLM 微调更加平民化。",
			CategoryID: catMap["ai-llm"],
			AuthorID:   userIDs[2],
			TagIDs:     []uint{tagIDs[2], tagIDs[3], tagIDs[10]},
		},
		{
			Title:      "Docker + Kubernetes 生产环境部署指南",
			Summary:    "从零开始搭建 Docker + K8s 生产环境，涵盖镜像构建、服务编排、监控告警等核心内容。",
			Content:    "# Docker + K8s 生产部署指南\n\n## 1. Dockerfile 最佳实践\n\n```dockerfile\nFROM golang:1.21-alpine AS builder\nWORKDIR /app\nCOPY . .\nRUN go build -o main .\n\nFROM alpine:latest\nCOPY --from=builder /app/main /main\nCMD [\"/main\"]\n```\n\n## 2. Kubernetes 部署\n\n```yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: myapp\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: myapp\n  template:\n    spec:\n      containers:\n      - name: myapp\n        image: myapp:latest\n```\n\n## 3. 监控与告警\n\n- Prometheus + Grafana\n- 日志收集：EFK Stack",
			CategoryID: catMap["devops"],
			AuthorID:   userIDs[3],
			TagIDs:     []uint{tagIDs[4], tagIDs[5]},
		},
		{
			Title:      "MySQL 索引优化实战：从慢查询到毫秒级响应",
			Summary:    "通过实际案例讲解 MySQL 索引优化技巧，包括索引设计、查询优化和性能监控。",
			Content:    "# MySQL 索引优化实战\n\n## 1. 索引类型\n\n- B+Tree 索引\n- 哈希索引\n- 全文索引\n- 空间索引\n\n## 2. 索引设计原则\n\n```sql\n-- 最左前缀原则\nCREATE INDEX idx_user_status_created \nON users(status, created_at);\n\n-- 覆盖索引\nSELECT id, name FROM users WHERE status = 'active';\n```\n\n## 3. 慢查询分析\n\n```sql\n-- 开启慢查询日志\nSET GLOBAL slow_query_log = 'ON';\nSET GLOBAL long_query_time = 1;\n\n-- 分析执行计划\nEXPLAIN SELECT * FROM orders WHERE user_id = 1;\n```\n\n## 4. 优化案例\n\n优化前：查询耗时 5s\n优化后：查询耗时 50ms",
			CategoryID: catMap["database"],
			AuthorID:   userIDs[4],
			TagIDs:     []uint{tagIDs[6]},
		},
		{
			Title:      "Redis 缓存设计模式与常见陷阱",
			Summary:    "深入探讨 Redis 缓存设计模式，包括缓存穿透、击穿、雪崩的解决方案。",
			Content:    "# Redis 缓存设计模式\n\n## 1. Cache Aside 模式\n\n```go\nfunc GetUser(id uint) (*User, error) {\n    // 1. 先查缓存\n    user, err := cache.Get(id)\n    if err == nil {\n        return user, nil\n    }\n    \n    // 2. 查数据库\n    user, err = db.FindByID(id)\n    if err != nil {\n        return nil, err\n    }\n    \n    // 3. 写入缓存\n    cache.Set(id, user, time.Hour)\n    return user, nil\n}\n```\n\n## 2. 缓存穿透\n\n使用布隆过滤器或空值缓存。\n\n## 3. 缓存击穿\n\n使用互斥锁或永不过期策略。\n\n## 4. 缓存雪崩\n\n设置随机过期时间。",
			CategoryID: catMap["database"],
			AuthorID:   userIDs[0],
			TagIDs:     []uint{tagIDs[7], tagIDs[0]},
		},
		{
			Title:      "React 18 并发特性深度解析",
			Summary:    "React 18 引入了并发渲染，本文深入解析 Suspense、Transitions 等新特性。",
			Content:    "# React 18 并发特性\n\n## 1. 并发渲染\n\nReact 18 引入了并发渲染机制。\n\n## 2. Suspense\n\n```jsx\n<Suspense fallback={<Loading />}>\n  <Comments />\n</Suspense>\n```\n\n## 3. useTransition\n\n```jsx\nfunction SearchResults() {\n  const [isPending, startTransition] = useTransition();\n  \n  const handleChange = (e) => {\n    startTransition(() => {\n      setSearchQuery(e.target.value);\n    });\n  };\n}\n```\n\n## 4. useDeferredValue\n\n延迟更新非关键 UI。",
			CategoryID: catMap["frontend"],
			AuthorID:   userIDs[1],
			TagIDs:     []uint{tagIDs[9]},
		},
		{
			Title:      "Python 异步编程：asyncio 实战",
			Summary:    "Python asyncio 异步编程入门，通过实际案例掌握异步 IO 的使用。",
			Content:    "# Python asyncio 实战\n\n## 1. 基础概念\n\n```python\nimport asyncio\n\nasync def hello():\n    print('Hello')\n    await asyncio.sleep(1)\n    print('World')\n\nasyncio.run(hello())\n```\n\n## 2. 并发执行\n\n```python\nasync def main():\n    tasks = [\n        asyncio.create_task(task1()),\n        asyncio.create_task(task2()),\n    ]\n    await asyncio.gather(*tasks)\n```\n\n## 3. 异步 HTTP\n\n```python\nimport aiohttp\n\nasync def fetch(url):\n    async with aiohttp.ClientSession() as session:\n        async with session.get(url) as resp:\n            return await resp.json()\n```",
			CategoryID: catMap["backend"],
			AuthorID:   userIDs[2],
			TagIDs:     []uint{tagIDs[10]},
		},
		{
			Title:      "Gin 框架中间件开发指南",
			Summary:    "Gin 中间件机制详解，从原理到实践，手把手教你开发自定义中间件。",
			Content:    "# Gin 中间件开发指南\n\n## 1. 中间件原理\n\nGin 中间件本质是一个 `HandlerFunc`。\n\n```go\nfunc Logger() gin.HandlerFunc {\n    return func(c *gin.Context) {\n        start := time.Now()\n        c.Next()\n        latency := time.Since(start)\n        log.Printf(\"%s %s %v\", c.Request.Method, c.Request.URL, latency)\n    }\n}\n```\n\n## 2. 认证中间件\n\n```go\nfunc AuthRequired() gin.HandlerFunc {\n    return func(c *gin.Context) {\n        token := c.GetHeader(\"Authorization\")\n        if token == \"\" {\n            c.AbortWithStatusJSON(401, gin.H{\"error\": \"unauthorized\"})\n            return\n        }\n        c.Next()\n    }\n}\n```\n\n## 3. 限流中间件\n\n使用 Redis 实现分布式限流。",
			CategoryID: catMap["go"],
			AuthorID:   userIDs[0],
			TagIDs:     []uint{tagIDs[0], tagIDs[11]},
		},
		{
			Title:      "TypeScript 高级类型体操：从入门到精通",
			Summary:    "TypeScript 类型系统非常强大，本文通过实战案例讲解高级类型的使用。",
			Content:    "# TypeScript 高级类型\n\n## 1. 条件类型\n\n```typescript\ntype IsString<T> = T extends string ? true : false;\n\ntype A = IsString<'hello'>; // true\ntype B = IsString<123>;     // false\n```\n\n## 2. 映射类型\n\n```typescript\ntype Readonly<T> = {\n    readonly [P in keyof T]: T[P];\n};\n\ntype Partial<T> = {\n    [P in keyof T]?: T[P];\n};\n```\n\n## 3. 模板字面量类型\n\n```typescript\ntype EventName = `on${Capitalize<string>}`;\n```\n\n## 4. 实战：类型安全的 EventEmitter\n\n```typescript\ntype EventMap = {\n    click: { x: number; y: number };\n    focus: {};\n};\n\nclass EventEmitter<T extends Record<string, any>> {\n    on<K extends keyof T>(event: K, handler: (data: T[K]) => void) {}\n    emit<K extends keyof T>(event: K, data: T[K]) {}\n}\n```",
			CategoryID: catMap["frontend"],
			AuthorID:   userIDs[4],
			TagIDs:     []uint{tagIDs[8]},
		},
	}

	for _, a := range articles {
		article := model.Article{
			Title:          a.Title,
			Summary:        a.Summary,
			Content:        a.Content,
			CategoryID:     a.CategoryID,
			AuthorID:       a.AuthorID,
			Status:         model.ArticleStatusPublished,
			Views:          r.Intn(4900) + 100,
			LikesCount:     r.Intn(190) + 10,
			FavoritesCount: r.Intn(95) + 5,
			CommentsCount:  r.Intn(50),
			PublishedAt:    timePtr(time.Now().Add(-time.Duration(r.Intn(30)+1) * 24 * time.Hour)),
		}

		if err := db.WithContext(ctx).Create(&article).Error; err != nil {
			log.Printf("create article %s: %v", a.Title, err)
			continue
		}

		// 创建文章标签关联
		for _, tagID := range a.TagIDs {
			at := model.ArticleTag{
				ArticleID: article.ID,
				TagID:     tagID,
			}
			db.Create(&at)
			// 更新标签使用次数
			db.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", gorm.Expr("usage_count + 1"))
		}
	}

	// 创建测试评论
	var allArticles []model.Article
	db.Find(&allArticles)

	comments := []struct {
		ArticleID uint
		AuthorID  uint
		Content   string
	}{
		{allArticles[0].ID, userIDs[1], "写得很好，泛型确实让 Go 更加灵活了！"},
		{allArticles[0].ID, userIDs[2], "请问泛型在性能上有什么影响吗？"},
		{allArticles[0].ID, userIDs[0], "性能影响很小，编译器会做单态化处理。"},
		{allArticles[1].ID, userIDs[0], "Composition API 确实比 Options API 更灵活"},
		{allArticles[1].ID, userIDs[3], "Pinia 比 Vuex 好用多了"},
		{allArticles[2].ID, userIDs[4], "QLoRA 让普通开发者也能玩大模型了"},
		{allArticles[2].ID, userIDs[0], "请问微调数据集有什么要求吗？"},
		{allArticles[3].ID, userIDs[1], "K8s 学习曲线确实陡峭"},
		{allArticles[4].ID, userIDs[2], "索引优化是数据库性能的关键"},
		{allArticles[4].ID, userIDs[3], "慢查询日志是排查问题的利器"},
		{allArticles[5].ID, userIDs[4], "缓存雪崩问题确实很常见"},
		{allArticles[6].ID, userIDs[2], "React 18 的并发特性很强大"},
		{allArticles[7].ID, userIDs[0], "asyncio 比 threading 更适合 IO 密集型任务"},
		{allArticles[8].ID, userIDs[1], "Gin 中间件设计得很优雅"},
		{allArticles[9].ID, userIDs[3], "TypeScript 类型体操确实需要多练习"},
	}

	for _, c := range comments {
		comment := model.Comment{
			ArticleID:  c.ArticleID,
			AuthorID:   c.AuthorID,
			Content:    c.Content,
			LikesCount: r.Intn(20),
		}
		if err := db.WithContext(ctx).Create(&comment).Error; err != nil {
			log.Printf("create comment: %v", err)
		}
	}

	// 创建一些回复（二级评论）
	var topComments []model.Comment
	db.Where("parent_id IS NULL").Find(&topComments)

	replies := []struct {
		ParentID uint
		AuthorID uint
		Content  string
	}{
		{topComments[1].ID, userIDs[0], "编译器优化得很好，基本没有性能损失。"},
		{topComments[6].ID, userIDs[2], "数据集质量很重要，建议用高质量的数据。"},
	}

	for _, rep := range replies {
		parentID := rep.ParentID
		reply := model.Comment{
			ArticleID:  getArticleID(db, rep.ParentID),
			AuthorID:   rep.AuthorID,
			ParentID:   &parentID,
			Content:    rep.Content,
			LikesCount: r.Intn(10),
		}
		db.Create(&reply)
	}

	fmt.Println("测试数据创建完成！")
	fmt.Println("测试用户：")
	fmt.Println("  test1@test.com / 123456 (张三)")
	fmt.Println("  test2@test.com / 123456 (李四)")
	fmt.Println("  test3@test.com / 123456 (王五)")
	fmt.Println("  test4@test.com / 123456 (赵六)")
	fmt.Println("  test5@test.com / 123456 (陈七)")
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func getArticleID(db *gorm.DB, commentID uint) uint {
	var comment model.Comment
	db.First(&comment, commentID)
	return comment.ArticleID
}
