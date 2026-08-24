package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"aidevclub/internal/model"
)

func main() {
	dsn := "root:root@tcp(localhost:3306)/aidevclub?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{}, &model.Category{}, &model.Tag{}, &model.Article{},
		&model.ArticleTag{}, &model.ArticleLike{}, &model.ArticleFavorite{},
		&model.Comment{}, &model.CommentLike{},
		&model.Skill{}, &model.SkillTag{},
		&model.McpServer{}, &model.McpServerTag{},
		&model.SkillLike{}, &model.SkillFavorite{},
		&model.McpServerLike{}, &model.McpServerFavorite{},
		&model.ResourceComment{}, &model.ResourceCommentLike{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	cleanData(db)
	seedCategories(db)
	userIDs := createUsers(db)
	tagIDs, tagMap := createTags(db)
	createArticles(db, r, userIDs, tagIDs, tagMap)
	createComments(db, r, userIDs)
	createMcpServers(db, r, userIDs, tagIDs, tagMap)
	createSkills(db, r, userIDs, tagIDs, tagMap)
	createResourceComments(db, r, userIDs)
	printSummary(db)
}

func cleanData(db *gorm.DB) {
	fmt.Println("清理旧数据...")
	db.Unscoped().Where("1 = 1").Delete(&model.ResourceCommentLike{})
	db.Unscoped().Where("1 = 1").Delete(&model.ResourceComment{})
	db.Unscoped().Where("1 = 1").Delete(&model.McpServerFavorite{})
	db.Unscoped().Where("1 = 1").Delete(&model.McpServerLike{})
	db.Unscoped().Where("1 = 1").Delete(&model.McpServerTag{})
	db.Unscoped().Where("1 = 1").Delete(&model.McpServer{})
	db.Unscoped().Where("1 = 1").Delete(&model.SkillFavorite{})
	db.Unscoped().Where("1 = 1").Delete(&model.SkillLike{})
	db.Unscoped().Where("1 = 1").Delete(&model.SkillTag{})
	db.Unscoped().Where("1 = 1").Delete(&model.Skill{})
	db.Unscoped().Where("1 = 1").Delete(&model.CommentLike{})
	db.Unscoped().Where("1 = 1").Delete(&model.Comment{})
	db.Unscoped().Where("1 = 1").Delete(&model.ArticleFavorite{})
	db.Unscoped().Where("1 = 1").Delete(&model.ArticleLike{})
	db.Unscoped().Where("1 = 1").Delete(&model.ArticleTag{})
	db.Unscoped().Where("1 = 1").Delete(&model.Article{})
	db.Unscoped().Where("1 = 1").Delete(&model.Tag{})
	db.Unscoped().Where("1 = 1").Delete(&model.Category{})
	db.Unscoped().Where("1 = 1").Delete(&model.User{})
}

func seedCategories(db *gorm.DB) {
	fmt.Println("创建分类...")
	cats := []model.Category{
		{Name: "Go", Slug: "go", SortOrder: 1},
		{Name: "后端", Slug: "backend", SortOrder: 2},
		{Name: "前端", Slug: "frontend", SortOrder: 3},
		{Name: "AI/LLM", Slug: "ai-llm", SortOrder: 4},
		{Name: "DevOps", Slug: "devops", SortOrder: 5},
		{Name: "数据库", Slug: "database", SortOrder: 6},
		{Name: "移动端", Slug: "mobile", SortOrder: 7},
		{Name: "安全", Slug: "security", SortOrder: 8},
		{Name: "其他", Slug: "other", SortOrder: 9},
	}
	db.Create(&cats)
}

func createUsers(db *gorm.DB) []uint {
	fmt.Println("创建用户...")
	type userData struct{ email, nick, bio string }
	data := []userData{
		{"linxm@example.com", "林小明", "Go 后端工程师，5 年经验，热爱开源"},
		{"wanglh@example.com", "王丽华", "前端架构师，专注 Vue/React 生态"},
		{"zhangjg@example.com", "张建国", "AI/ML 研究员，博士，研究方向为 NLP"},
		{"liumq@example.com", "刘美琪", "DevOps 工程师，K8s CKA/CKAD 认证"},
		{"chenzq@example.com", "陈志强", "全栈开发者，创业公司 CTO"},
		{"lixd@example.com", "李晓东", "数据库专家，MySQL OCP，PostgreSQL 贡献者"},
		{"zhaoxm@example.com", "赵雪梅", "移动端开发，Flutter/React Native 实战经验"},
		{"sundp@example.com", "孙大鹏", "安全工程师，专注渗透测试与安全审计"},
		{"zhouwj@example.com", "周文静", "技术写作者，开源贡献者，社区运营"},
		{"wuhr@example.com", "吴浩然", "云架构师，AWS SAP/GCP Professional 认证"},
	}
	hash := hashPassword("123456")
	var ids []uint
	for _, u := range data {
		user := model.User{Email: u.email, Nickname: u.nick, PasswordHash: hash, Bio: u.bio, AvatarURL: fmt.Sprintf("https://picsum.photos/200/200?random=%d", len(ids)+1)}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("创建用户 %s 失败: %v", u.email, err)
		}
		ids = append(ids, user.ID)
	}
	return ids
}

func createTags(db *gorm.DB) ([]uint, map[string]uint) {
	fmt.Println("创建标签...")
	names := []string{"Go", "Vue", "React", "AI", "LLM", "Docker", "Kubernetes", "MySQL", "Redis", "TypeScript", "Python", "Gin", "DevOps", "Cloud", "Microservices", "GraphQL", "Rust", "Java", "Frontend", "Backend"}
	tm := make(map[string]uint)
	var ids []uint
	for _, n := range names {
		t := model.Tag{Name: n, Enabled: true}
		db.Create(&t)
		tm[n] = t.ID
		ids = append(ids, t.ID)
	}
	return ids, tm
}

func createArticles(db *gorm.DB, r *rand.Rand, userIDs, tagIDs []uint, tagMap map[string]uint) {
	fmt.Println("创建文章...")
	var firstCat model.Category
	db.First(&firstCat)

	type artDef struct {
		title, summary, content string
		tags                    []string
	}
	arts := []artDef{
		{
			"Kubernetes 生产环境部署清单：从零到上线的 10 个关键步骤",
			"本文总结了在生产环境部署 Kubernetes 集群时需要关注的 10 个关键步骤，涵盖资源限制、健康检查、安全策略、日志管理等方面，帮助你避免常见的生产事故，确保服务稳定运行。",
			`# Kubernetes 生产环境部署清单

在生产环境部署 Kubernetes 不是一件简单的事情，需要经过周密的规划和准备。本文将分享我在多个生产项目中总结的 10 个关键步骤。

## 1. 资源限制与请求

每个容器都必须设置 resources requests 和 limits，这是防止资源争抢的基础：

` + "```yaml" + `
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myapp:v1.2.0
    resources:
      requests:
        memory: "256Mi"
        cpu: "250m"
      limits:
        memory: "512Mi"
        cpu: "500m"
` + "```" + `

## 2. 健康检查配置

![Kubernetes 架构示意图](https://picsum.photos/800/400?random=101)

Liveness、Readiness 和 Startup 三种探针缺一不可：

` + "```yaml" + `
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
` + "```" + `

## 3. Pod 反亲和性与高可用

通过 podAntiAffinity 确保同一应用的多个副本分布在不同节点上，避免单点故障。

## 4. 网络策略

默认情况下 Pod 之间可以自由通信，使用 NetworkPolicy 限制流量，实现微隔离。

## 5. 密钥与配置管理

使用 Secret 和 ConfigMap 管理敏感信息，配合 External Secrets Operator 对接 Vault 等外部密钥管理系统。

## 总结

以上 10 个步骤是生产环境部署的基本清单，实际场景中还需要考虑监控告警、灾备恢复等因素。`,
			[]string{"Kubernetes", "DevOps", "Cloud"},
		},
		{
			"深入理解 Kubernetes HPA：自定义指标与弹性伸缩实战",
			"HPA 是 Kubernetes 中实现自动弹性伸缩的核心组件。本文深入讲解 HPA 的工作原理，并通过自定义指标实现基于业务负载的弹性伸缩，让你的集群真正智能化。",
			`# 深入理解 Kubernetes HPA

HPA（Horizontal Pod Autoscaler）根据观测到的指标自动调整 Pod 副本数。

## 基本用法

` + "```yaml" + `
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
` + "```" + `

## 自定义指标

![HPA 工作流程](https://picsum.photos/800/400?random=102)

基于 QPS 的弹性伸缩更贴近业务需求，需要部署 Prometheus Adapter：

` + "```yaml" + `
metrics:
- type: Pods
  pods:
    metric:
      name: http_requests_per_second
    target:
      type: AverageValue
      averageValue: "100"
` + "```" + `

## 冷却窗口

设置 stabilizationWindowSeconds 防止频繁伸缩带来的抖动。

## 总结

合理使用 HPA 可以显著降低资源成本，同时保障服务质量。`,
			[]string{"Kubernetes", "DevOps"},
		},
		{
			"Kubernetes 多环境管理：用 Kustomize 和 Helm 管理 dev/staging/prod",
			"多环境管理是 DevOps 的核心挑战之一。本文介绍如何结合 Kustomize 和 Helm 管理 dev、staging、prod 三套环境的 Kubernetes 配置，实现配置复用与环境差异化。",
			`# Kubernetes 多环境管理

## 为什么需要多环境管理

在实际项目中，dev、staging、prod 环境的配置存在差异，如副本数、资源限制、镜像标签等。

## Kustomize 方案

` + "```yaml" + `
# base/kustomization.yaml
resources:
- deployment.yaml
- service.yaml

# overlays/prod/kustomization.yaml
bases:
- ../../base
patchesStrategicMerge:
- replica-patch.yaml
namePrefix: prod-
` + "```" + `

![多环境管理流程](https://picsum.photos/800/400?random=103)

## Helm 方案

` + "```yaml" + `
# values-prod.yaml
replicaCount: 3
resources:
  limits:
    memory: 1Gi
    cpu: 500m
image:
  tag: "v1.2.0"
` + "```" + `

## 选择建议

小项目用 Kustomize，复杂项目用 Helm，也可以两者结合使用。

## 总结

良好的多环境管理能减少配置错误，提升发布效率。`,
			[]string{"Kubernetes", "DevOps", "Docker"},
		},
		{
			"Kubernetes 日志管理最佳实践：EFK Stack 与结构化日志",
			"日志是排查问题的关键。本文介绍如何在 Kubernetes 中搭建 EFK（Elasticsearch + Fluentd + Kibana）日志系统，并规范应用的结构化日志输出。",
			`# Kubernetes 日志管理最佳实践

## 日志架构

在 Kubernetes 中，推荐采用节点级日志收集方案。

` + "```yaml" + `
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/*.log
      pos_file /var/log/fluentd-containers.log.pos
      tag kubernetes.*
      format json
      read_from_head true
    </source>
    <match kubernetes.**>
      @type elasticsearch
      host elasticsearch
      port 9200
      logstash_format true
    </match>
` + "```" + `

![EFK 架构图](https://picsum.photos/800/400?random=104)

## 结构化日志

应用输出 JSON 格式日志，便于解析和检索：

` + "```go" + `
log.Printf(` + "`" + `{"level":"info","msg":"request handled","path":"%s","status":%d,"latency":"%v"}` + "`" + `, r.URL.Path, status, latency)
` + "```" + `

## 日志轮转与保留

配置日志索引生命周期管理（ILM），自动清理过期日志。

## 总结

完善的日志系统能大幅提升问题排查效率。`,
			[]string{"Kubernetes", "DevOps"},
		},
		{
			"Kubernetes 网络策略详解：用 Calico 实现微隔离安全",
			"Kubernetes 默认允许所有 Pod 自由通信，这在安全敏感的场景下是不够的。本文详解如何使用 NetworkPolicy 和 Calico 实现 Pod 间的微隔离。",
			`# Kubernetes 网络策略详解

## 默认行为

K8s 默认采用扁平网络模型，所有 Pod 可以自由通信。

## NetworkPolicy 基础

` + "```yaml" + `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-server-policy
spec:
  podSelector:
    matchLabels:
      app: api-server
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - port: 8080
` + "```" + `

![网络策略示意](https://picsum.photos/800/400?random=105)

## Calico 高级特性

Calico 支持全局网络策略、服务账户选择器、CIDR 匹配等高级功能。

## 默认拒绝策略

推荐为每个命名空间设置默认拒绝所有入站流量的策略，再按需开放。

## 总结

网络策略是零信任安全架构的重要组成部分。`,
			[]string{"Kubernetes", "Cloud", "Microservices"},
		},
		{
			"Docker 多阶段构建实战：将 Go 应用镜像从 1GB 压缩到 10MB",
			"镜像体积直接影响构建速度、存储成本和安全攻击面。本文通过一个真实的 Go 应用案例，展示如何使用多阶段构建将镜像从 1GB 压缩到 10MB 以内。",
			`# Docker 多阶段构建实战

## 问题

一个普通的 Go 应用 Dockerfile 往往会生成超过 1GB 的镜像。

## 优化前

` + "```dockerfile" + `
FROM golang:1.21
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]
` + "```" + `

## 多阶段构建

` + "```dockerfile" + `
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

FROM scratch
COPY --from=builder /app/main /main
EXPOSE 8080
CMD ["/main"]
` + "```" + `

![镜像大小对比](https://picsum.photos/800/400?random=201)

## 进一步优化

使用 UPX 压缩、去除调试信息等手段可以继续缩小镜像。

## 安全扫描

使用 Trivy 扫描镜像漏洞：` + "`" + `trivy image myapp:latest` + "`" + `

## 总结

多阶段构建是 Docker 镜像优化的第一步，效果立竿见影。`,
			[]string{"Docker", "Go", "DevOps"},
		},
		{
			"Docker Compose V2 实战：本地开发环境一键搭建与热重载",
			"Docker Compose V2 带来了更好的性能和新的特性。本文展示如何用 Compose V2 搭建包含应用、数据库、缓存、消息队列的完整本地开发环境。",
			`# Docker Compose V2 实战

## 完整开发环境

` + "```yaml" + `
version: "3.9"
services:
  app:
    build:
      context: .
      target: dev
    volumes:
    - .:/app
    ports:
    - "8080:8080"
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
    environment:
    - DB_HOST=db
    - REDIS_HOST=redis

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: aidevclub
    volumes:
    - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
    - "6379:6379"

volumes:
  db_data:
` + "```" + `

![开发环境架构](https://picsum.photos/800/400?random=202)

## 热重载

使用 air 实现 Go 应用热重载，配合 volume mount 实现代码实时同步。

## Profile 管理

使用 profiles 按需启动可选服务，如 Prometheus、Grafana。

## 总结

Docker Compose 是本地开发环境的标准工具。`,
			[]string{"Docker", "DevOps", "MySQL"},
		},
		{
			"Docker 日志管理：从 docker logs 到生产级日志收集方案",
			"日志管理是容器化应用的重要环节。本文从 docker logs 基础出发，介绍生产环境中的日志收集方案，包括 Fluentd、Loki 等工具的使用。",
			`# Docker 日志管理

## 默认日志驱动

Docker 默认使用 json-file 日志驱动，日志存储在节点本地。

## 日志驱动选择

` + "```json" + `
{
  "log-driver": "journald",
  "log-opts": {
    "tag": "{{.Name}}/{{.ID}}"
  }
}
` + "```" + `

![日志收集架构](https://picsum.photos/800/400?random=203)

## Loki + Grafana

Loki 是轻量级的日志聚合系统，与 Grafana 无缝集成：

` + "```yaml" + `
clients:
  - url: http://loki:3100/loki/api/v1/push
scrape_configs:
  - job_name: docker
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
    relabel_configs:
      - source_labels: [container_name]
        target_label: container
` + "```" + `

## 日志轮转

配置 max-size 和 max-file 防止磁盘撑满。

## 总结

选择合适的日志方案对运维至关重要。`,
			[]string{"Docker", "DevOps"},
		},
		{
			"Docker 安全加固指南：镜像扫描、Rootless 模式与 Seccomp",
			"容器安全不容忽视。本文介绍 Docker 安全加固的多个维度，包括镜像漏洞扫描、Rootless 模式运行、Seccomp 配置文件等实践。",
			`# Docker 安全加固指南

## 镜像安全

使用 Trivy 进行镜像漏洞扫描：

` + "```bash" + `
trivy image --severity HIGH,CRITICAL myapp:latest
` + "```" + `

![安全扫描结果](https://picsum.photos/800/400?random=204)

## Rootless 模式

Rootless Docker 以非 root 用户运行守护进程和容器，降低提权风险。

## Seccomp 配置

通过 Seccomp Profile 限制容器可使用的系统调用：

` + "```json" + `
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {
      "names": ["accept", "bind", "connect"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
` + "```" + `

## 其他加固措施

限制 capabilities、使用只读文件系统、设置资源限制等。

## 总结

安全是一个持续的过程，需要定期审计和更新。`,
			[]string{"Docker", "DevOps", "Cloud"},
		},
		{
			"从 Docker Compose 到 Kubernetes：平滑迁移的完整指南",
			"很多团队从 Docker Compose 起步，随着业务增长需要迁移到 Kubernetes。本文提供一套完整的迁移方案，包括工具使用、配置转换和渐进式迁移策略。",
			`# 从 Docker Compose 到 Kubernetes

## 迁移策略

推荐渐进式迁移，先在 K8s 上运行无状态服务，再逐步迁移有状态服务。

## Kompose 工具

Kompose 可以将 Compose 文件转换为 K8s 资源：

` + "```bash" + `
kompose convert -f docker-compose.yaml -o k8s/
kompose up
` + "```" + `

![迁移流程](https://picsum.photos/800/400?random=205)

## 手动转换示例

Compose 中的 services 对应 K8s Deployment + Service：

` + "```yaml" + `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: app
        image: myapp:latest
        envFrom:
        - configMapRef:
            name: app-config
` + "```" + `

## 注意事项

数据卷、网络、密钥管理等方面需要额外处理。

## 总结

迁移是渐进过程，不要试图一步到位。`,
			[]string{"Docker", "Kubernetes", "DevOps"},
		},
		{
			"Go 语言 Channel 高级模式：fan-in、fan-out 与 pipeline 实战",
			"Channel 是 Go 并发编程的核心。本文深入讲解 Channel 的高级使用模式，包括 fan-in、fan-out、pipeline 等经典并发模式，附带完整的代码示例。",
			`# Go 语言 Channel 高级模式

## Fan-Out 模式

多个 goroutine 同时从同一个 channel 读取数据：

` + "```go" + `
func fanOut(input <-chan int, workers int) []<-chan string {
    channels := make([]<-chan string, workers)
    for i := 0; i < workers; i++ {
        channels[i] = process(input, i)
    }
    return channels
}

func process(input <-chan int, id int) <-chan string {
    out := make(chan string)
    go func() {
        defer close(out)
        for v := range input {
            out <- fmt.Sprintf("worker-%d: %d", id, v*2)
        }
    }()
    return out
}
` + "```" + `

## Fan-In 模式

![Channel 模式示意](https://picsum.photos/800/400?random=301)

将多个 channel 的输出合并到一个 channel：

` + "```go" + `
func fanIn(channels ...<-chan string) <-chan string {
    var wg sync.WaitGroup
    out := make(chan string)
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan string) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(ch)
    }
    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
` + "```" + `

## Pipeline 模式

将多个处理步骤串联成流水线，每一步通过 channel 传递数据。

## 总结

掌握这些模式能写出更优雅的并发代码。`,
			[]string{"Go", "Backend", "Microservices"},
		},
		{
			"Go 泛型实战：用类型约束构建通用数据结构和算法库",
			"Go 1.18 引入泛型后，我们可以编写更通用的代码。本文通过实现通用数据结构（Set、Map、Queue）和算法（Sort、Search、Map/Filter），展示泛型的实际应用。",
			`# Go 泛型实战

## 类型约束

` + "```go" + `
type Number interface {
    ~int | ~int32 | ~int64 | ~float32 | ~float64
}

type Ordered interface {
    ~int | ~int32 | ~string | ~float64
}
` + "```" + `

## 通用 Set

` + "```go" + `
type Set[T comparable] struct {
    items map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
    return &Set[T]{items: make(map[T]struct{})}
}

func (s *Set[T]) Add(item T) {
    s.items[item] = struct{}{}
}

func (s *Set[T]) Contains(item T) bool {
    _, ok := s.items[item]
    return ok
}
` + "```" + `

![泛型类型约束](https://picsum.photos/800/400?random=302)

## 通用算法

` + "```go" + `
func Map[T any, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

func Filter[T any](s []T, pred func(T) bool) []T {
    var result []T
    for _, v := range s {
        if pred(v) {
            result = append(result, v)
        }
    }
    return result
}
` + "```" + `

## 总结

泛型让 Go 代码更加通用和可复用。`,
			[]string{"Go", "Backend"},
		},
		{
			"Go 并发模式：Context 超时控制与优雅关闭的完整方案",
			"在 Go 服务中，正确处理超时和优雅关闭是保证服务稳定性的关键。本文介绍如何使用 context 包实现超时控制、级联取消和优雅关闭。",
			`# Go 并发模式：Context 与优雅关闭

## Context 超时控制

` + "```go" + `
func fetchData(ctx context.Context, url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, fmt.Errorf("请求超时: %w", err)
        }
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
` + "```" + `

![Context 传播链](https://picsum.photos/800/400?random=303)

## 优雅关闭

` + "```go" + `
func main() {
    srv := &http.Server{Addr: ":8080"}

    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
` + "```" + `

## 总结

良好的超时和关闭机制是生产级 Go 服务的基础。`,
			[]string{"Go", "Backend", "Gin"},
		},
		{
			"Go 错误处理最佳实践：从 if err != nil 到结构化错误管理",
			"Go 的 error 处理一直被讨论。本文总结 Go 错误处理的最佳实践，包括 sentinel errors、自定义错误类型、errors.Is/As 的正确使用，以及如何构建清晰的错误链。",
			`# Go 错误处理最佳实践

## Sentinel Errors

` + "```go" + `
var (
    ErrNotFound    = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrConflict    = errors.New("conflict")
)
` + "```" + `

## 自定义错误类型

` + "```go" + `
type AppError struct {
    Code    int
    Message string
    Err     error
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *AppError) Unwrap() error {
    return e.Err
}
` + "```" + `

![错误处理流程](https://picsum.photos/800/400?random=304)

## 错误判断

` + "```go" + `
if errors.Is(err, ErrNotFound) {
    return c.JSON(404, gin.H{"error": "resource not found"})
}

var appErr *AppError
if errors.As(err, &appErr) {
    return c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
` + "```" + `

## 总结

好的错误处理让代码更健壮、更易调试。`,
			[]string{"Go", "Backend"},
		},
		{
			"Go 性能优化实战：pprof 分析、内存泄漏排查与 GC 调优",
			"性能优化是 Go 开发者的必备技能。本文通过真实案例，展示如何使用 pprof 进行 CPU 和内存分析，排查内存泄漏，以及通过 GOGC 等参数调优 GC。",
			`# Go 性能优化实战

## pprof 基础

` + "```go" + `
import _ "net/http/pprof"

func main() {
    go func() {
        http.ListenAndServe(":6060", nil)
    }()
}
` + "```" + `

` + "```bash" + `
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof -http=:8080 cpu.prof
` + "```" + `

## 内存分析

![pprof 火焰图](https://picsum.photos/800/400?random=305)

` + "```bash" + `
go tool pprof http://localhost:6060/debug/pprof/heap
` + "```" + `

## 常见内存泄漏场景

- goroutine 泄漏：channel 未关闭、context 未取消
- 全局 map 未清理
- time.After 在循环中使用

## GC 调优

` + "```bash" + `
GOGC=200 GOMEMLIMIT=2GiB ./myapp
` + "```" + `

## 总结

性能优化要基于数据，不要凭直觉。`,
			[]string{"Go", "Backend", "DevOps"},
		},
		{
			"Vue 3 组合式函数（Composables）设计模式与实战",
			"组合式函数是 Vue 3 Composition API 的核心复用机制。本文总结常用的 Composable 设计模式，包括 useFetch、useForm、usePermission 等实用案例。",
			`# Vue 3 组合式函数设计模式

## 什么是 Composable

Composable 是利用 Vue Composition API 封装和复用有状态逻辑的函数。

## useFetch

` + "```typescript" + `
export function useFetch<T>(url: MaybeRef<string>) {
  const data = ref<T>() as Ref<T | undefined>
  const error = ref<Error>()
  const loading = ref(false)

  async function execute() {
    loading.value = true
    try {
      const res = await fetch(unref(url))
      data.value = await res.json()
    } catch (e) {
      error.value = e as Error
    } finally {
      loading.value = false
    }
  }

  watchEffect(() => execute())

  return { data, error, loading, refresh: execute }
}
` + "```" + `

![Composable 模式](https://picsum.photos/800/400?random=401)

## useForm

封装表单验证、提交、重置逻辑。

## usePermission

基于角色和权限控制 UI 元素的显示。

## 总结

好的 Composable 设计能大幅提升代码复用性。`,
			[]string{"Vue", "Frontend", "TypeScript"},
		},
		{
			"React 18 并发特性深度解析：Suspense、Transitions 与 useDeferredValue",
			"React 18 引入了并发渲染机制，带来了 Suspense、useTransition、useDeferredValue 等新特性。本文深入解析这些特性的工作原理和使用场景。",
			`# React 18 并发特性

## 并发渲染

React 18 的并发渲染允许中断和恢复渲染工作。

## Suspense

` + "```jsx" + `
function App() {
  return (
    <Suspense fallback={<Skeleton />}>
      <UserProfile />
      <Suspense fallback={<CommentsSkeleton />}>
        <Comments />
      </Suspense>
    </Suspense>
  )
}
` + "```" + `

![并发渲染流程](https://picsum.photos/800/400?random=402)

## useTransition

` + "```jsx" + `
function SearchPage() {
  const [query, setQuery] = useState('')
  const [isPending, startTransition] = useTransition()

  const handleChange = (e) => {
    setQuery(e.target.value)
    startTransition(() => {
      setSearchResults(query)
    })
  }

  return (
    <>
      <input value={query} onChange={handleChange} />
      {isPending && <Spinner />}
      <Results />
    </>
  )
}
` + "```" + `

## useDeferredValue

延迟更新非关键 UI，保持输入响应性。

## 总结

并发特性让 React 应用更加流畅。`,
			[]string{"React", "Frontend", "TypeScript"},
		},
		{
			"Vue 3 + TypeScript 大型项目架构设计：从目录结构到状态管理",
			"大型 Vue 3 项目需要良好的架构设计。本文分享一个经过实战验证的项目架构方案，涵盖目录结构、状态管理、路由设计、API 层封装等核心内容。",
			`# Vue 3 大型项目架构设计

## 目录结构

` + "```text" + `
src/
├── api/          # API 请求层
├── assets/       # 静态资源
├── components/   # 通用组件
├── composables/  # 组合式函数
├── layouts/      # 布局组件
├── pages/        # 页面组件
├── router/       # 路由配置
├── stores/       # Pinia 状态管理
├── types/        # TypeScript 类型
└── utils/        # 工具函数
` + "```" + `

## API 层封装

` + "```typescript" + `
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE,
  timeout: 10000,
})

request.interceptors.request.use((config) => {
  const token = useUserStore().token
  if (token) config.headers.Authorization = ` + "`" + `Bearer ${token}` + "`" + `
  return config
})
` + "```" + `

![项目架构图](https://picsum.photos/800/400?random=403)

## 状态管理

使用 Pinia 按功能模块拆分 store。

## 总结

好的架构是项目可持续发展的基础。`,
			[]string{"Vue", "Frontend", "TypeScript"},
		},
		{
			"React 状态管理 2024：Zustand vs Jotai vs Redux Toolkit 对比",
			"React 状态管理方案众多，本文从 API 设计、性能、TypeScript 支持、学习曲线等维度对比 Zustand、Jotai 和 Redux Toolkit，帮助你选择最适合的方案。",
			`# React 状态管理方案对比

## Zustand

` + "```typescript" + `
import { create } from 'zustand'

const useStore = create<CounterState>((set) => ({
  count: 0,
  increment: () => set((s) => ({ count: s.count + 1 })),
  decrement: () => set((s) => ({ count: s.count - 1 })),
}))
` + "```" + `

## Jotai

` + "```typescript" + `
import { atom, useAtom } from 'jotai'

const countAtom = atom(0)
const doubleAtom = atom((get) => get(countAtom) * 2)

function Counter() {
  const [count, setCount] = useAtom(countAtom)
  return <button onClick={() => setCount(c => c + 1)}>{count}</button>
}
` + "```" + `

![状态管理对比](https://picsum.photos/800/400?random=404)

## 对比总结

- Zustand：简单直接，适合中小型项目
- Jotai：原子化状态，适合细粒度更新
- Redux Toolkit：功能完善，适合大型项目

## 总结

没有最好的方案，只有最适合的方案。`,
			[]string{"React", "Frontend", "TypeScript"},
		},
		{
			"Vue 3 性能优化完全指南：虚拟列表、懒加载与编译优化",
			"Vue 3 在性能方面有很多优化空间。本文介绍虚拟列表、组件懒加载、v-once/v-memo 指令、编译时优化等实用性能优化技巧。",
			`# Vue 3 性能优化完全指南

## 虚拟列表

当列表数据量大时，使用虚拟列表只渲染可见区域的元素：

` + "```vue" + `
<template>
  <VirtualList
    :items="items"
    :item-height="50"
    :container-height="600"
  >
    <template #default="{ item }">
      <div class="list-item">{{ item.name }}</div>
    </template>
  </VirtualList>
</template>
` + "```" + `

## 组件懒加载

` + "```typescript" + `
const HeavyChart = defineAsyncComponent(() =>
  import('./components/HeavyChart.vue')
)
` + "```" + `

![性能优化效果](https://picsum.photos/800/400?random=405)

## v-memo 指令

` + "```vue" + `
<div v-memo="[selectedItem.id]">
  <DetailPanel :item="selectedItem" />
</div>
` + "```" + `

## 编译优化

使用 defineModel、响应式解构等新特性减少运行时开销。

## 总结

性能优化要循序渐进，先测量再优化。`,
			[]string{"Vue", "Frontend", "TypeScript"},
		},
		{
			"LLM 应用开发入门：用 LangChain 构建你的第一个 AI 应用",
			"LangChain 是构建 LLM 应用的主流框架。本文从零开始，带你用 LangChain 构建一个包含对话记忆、工具调用、知识库检索的完整 AI 应用。",
			`# LLM 应用开发入门

## LangChain 基础

` + "```python" + `
from langchain.chat_models import ChatOpenAI
from langchain.prompts import ChatPromptTemplate
from langchain.chains import LLMChain

llm = ChatOpenAI(model="gpt-4", temperature=0)
prompt = ChatPromptTemplate.from_messages([
    ("system", "你是一个专业的技术助手。"),
    ("user", "{input}"),
])
chain = LLMChain(llm=llm, prompt=prompt)
result = chain.run(input="解释 Go 的 goroutine")
` + "```" + `

## RAG 知识库

![RAG 架构](https://picsum.photos/800/400?random=501)

` + "```python" + `
from langchain.vectorstores import FAISS
from langchain.embeddings import OpenAIEmbeddings

vectorstore = FAISS.from_texts(docs, OpenAIEmbeddings())
retriever = vectorstore.as_retriever()
` + "```" + `

## 工具调用

通过 Agent 让 LLM 调用外部工具，如搜索引擎、代码执行器等。

## 总结

LangChain 让 LLM 应用开发变得简单。`,
			[]string{"AI", "LLM", "Python"},
		},
		{
			"RAG 系统优化实战：从 60% 到 95% 的检索准确率提升之路",
			"RAG（检索增强生成）是当前 LLM 应用的主流范式。本文分享在实际项目中优化 RAG 系统的经验，包括分块策略、Embedding 模型选择、重排序等技巧。",
			`# RAG 系统优化实战

## 问题

默认 RAG 系统的检索准确率往往不理想。

## 分块策略

` + "```python" + `
from langchain.text_splitter import RecursiveCharacterTextSplitter

splitter = RecursiveCharacterTextSplitter(
    chunk_size=500,
    chunk_overlap=50,
    separators=["\n\n", "\n", "。", "！", "？"]
)
chunks = splitter.split_documents(docs)
` + "```" + `

![RAG 优化流程](https://picsum.photos/800/400?random=502)

## Embedding 模型

选择领域适配的 Embedding 模型，中文场景可考虑 bge-large-zh。

## 重排序

使用 Cross-Encoder 对检索结果重排序：

` + "```python" + `
from sentence_transformers import CrossEncoder

reranker = CrossEncoder('BAAI/bge-reranker-large')
pairs = [[query, doc.page_content] for doc in results]
scores = reranker.predict(pairs)
` + "```" + `

## 总结

RAG 优化是一个系统工程，需要持续迭代。`,
			[]string{"AI", "LLM", "Python"},
		},
		{
			"大语言模型微调入门：从 LoRA 到 QLoRA 的完整实践指南",
			"大语言模型微调成本高昂，LoRA 和 QLoRA 技术让普通开发者也能在消费级 GPU 上微调模型。本文从原理到实践，完整讲解微调流程。",
			`# 大语言模型微调入门

## LoRA 原理

LoRA 通过低秩矩阵分解减少可训练参数量：

` + "```python" + `
from peft import LoraConfig, get_peft_model

config = LoraConfig(
    r=16,
    lora_alpha=32,
    target_modules=["q_proj", "v_proj"],
    lora_dropout=0.05,
    task_type="CAUSAL_LM"
)
model = get_peft_model(base_model, config)
model.print_trainable_parameters()
` + "```" + `

## QLoRA

![LoRA 原理图](https://picsum.photos/800/400?random=503)

QLoRA 引入 4-bit 量化，进一步降低显存需求：

` + "```python" + `
from transformers import BitsAndBytesConfig

bnb_config = BitsAndBytesConfig(
    load_in_4bit=True,
    bnb_4bit_quant_type="nf4",
    bnb_4bit_compute_dtype=torch.bfloat16
)
model = AutoModelForCausalLM.from_pretrained(
    model_name, quantization_config=bnb_config
)
` + "```" + `

## 训练配置

使用 gradient checkpointing 和 gradient accumulation 优化训练效率。

## 总结

QLoRA 让 LLM 微调触手可及。`,
			[]string{"AI", "LLM", "Python"},
		},
		{
			"AI Agent 架构设计：ReAct、Plan-and-Execute 与多 Agent 协作",
			"AI Agent 是 LLM 应用的高级形态。本文介绍 ReAct、Plan-and-Execute 等主流 Agent 架构，以及如何设计多 Agent 协作系统。",
			`# AI Agent 架构设计

## ReAct 模式

ReAct 将推理和行动交替进行：

` + "```python" + `
from langchain.agents import initialize_agent, Tool

tools = [
    Tool(name="Search", func=search, description="搜索最新信息"),
    Tool(name="Calculator", func=calculate, description="数学计算"),
]
agent = initialize_agent(tools, llm, agent="zero-shot-react-description")
agent.run("2024年Go语言最受欢迎的框架是什么？")
` + "```" + `

## Plan-and-Execute

![Agent 架构](https://picsum.photos/800/400?random=504)

先制定计划，再逐步执行，适合复杂任务。

## 多 Agent 协作

` + "```python" + `
from langgraph.graph import StateGraph

workflow = StateGraph(AgentState)
workflow.add_node("planner", planner_agent)
workflow.add_node("executor", executor_agent)
workflow.add_node("reviewer", reviewer_agent)
workflow.add_edge("planner", "executor")
workflow.add_edge("executor", "reviewer")
` + "```" + `

## 总结

选择合适的 Agent 架构是成功的关键。`,
			[]string{"AI", "LLM", "Python"},
		},
		{
			"Prompt Engineering 实战：10 个提升 LLM 输出质量的技巧",
			"Prompt Engineering 是与 LLM 交互的核心技能。本文总结 10 个实用的 Prompt 技巧，包括 Few-shot、Chain-of-Thought、角色扮演等，附带大量示例。",
			`# Prompt Engineering 实战

## 1. 明确角色和任务

` + "```text" + `
你是一位资深的 Go 后端工程师，有 10 年经验。
请 review 以下代码，指出潜在的性能问题和安全隐患。
` + "```" + `

## 2. Few-shot 示例

` + "```text" + `
将以下技术术语翻译成中文：
Kubernetes -> Kubernetes（K8s）
Docker -> Docker
LLM -> 大语言模型
Goroutine -> 协程
` + "```" + `

![Prompt 技巧](https://picsum.photos/800/400?random=505)

## 3. Chain-of-Thought

让模型逐步推理，提升复杂问题的准确率。

## 4. 输出格式控制

` + "```text" + `
请以 JSON 格式输出，包含以下字段：
- title: 文章标题
- summary: 200字以内的摘要
- tags: 标签数组（2-3个）
` + "```" + `

## 其他技巧

温度调节、分隔符使用、负面指令等。

## 总结

好的 Prompt 是获得高质量输出的前提。`,
			[]string{"AI", "LLM"},
		},
		{
			"MySQL 索引优化实战：从慢查询到毫秒级响应的完整案例",
			"索引是 MySQL 性能优化的核心。本文通过多个真实案例，讲解索引设计原则、慢查询分析、执行计划解读和索引优化技巧。",
			`# MySQL 索引优化实战

## 慢查询定位

` + "```sql" + `
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1;

SELECT * FROM orders 
WHERE user_id = 123 
AND status = 'paid' 
AND created_at > '2024-01-01'
ORDER BY created_at DESC
LIMIT 20;
` + "```" + `

## 执行计划分析

` + "```sql" + `
EXPLAIN SELECT * FROM orders 
WHERE user_id = 123 AND status = 'paid';
` + "```" + `

![EXPLAIN 结果分析](https://picsum.photos/800/400?random=601)

## 索引设计

` + "```sql" + `
ALTER TABLE orders ADD INDEX idx_user_status_created 
(user_id, status, created_at);
` + "```" + `

## 覆盖索引

查询字段全部在索引中时，无需回表：

` + "```sql" + `
ALTER TABLE orders ADD INDEX idx_cover 
(user_id, status, created_at, order_no);

SELECT order_no FROM orders 
WHERE user_id = 123 AND status = 'paid';
` + "```" + `

## 总结

索引优化是 DBA 的基本功。`,
			[]string{"MySQL", "Backend"},
		},
		{
			"Redis 缓存设计模式：Cache Aside、Write Through 与 Write Behind",
			"缓存是高性能系统的标配。本文深入讲解三种主流缓存设计模式，以及缓存穿透、击穿、雪崩的解决方案。",
			`# Redis 缓存设计模式

## Cache Aside

` + "```go" + `
func GetUser(id uint) (*User, error) {
    key := fmt.Sprintf("user:%d", id)
    data, err := redis.Get(ctx, key).Result()
    if err == nil {
        var user User
        json.Unmarshal([]byte(data), &user)
        return &user, nil
    }

    user, err := db.FindUser(id)
    if err != nil {
        return nil, err
    }
    data, _ = json.Marshal(user)
    redis.Set(ctx, key, data, time.Hour)
    return user, nil
}
` + "```" + `

![缓存设计模式](https://picsum.photos/800/400?random=602)

## Write Through

写入时同步更新缓存和数据库。

## 缓存穿透

使用布隆过滤器或缓存空值。

## 缓存雪崩

设置随机过期时间，使用多级缓存。

## 总结

选择合适的缓存模式要基于业务特点。`,
			[]string{"Redis", "Backend", "Go"},
		},
		{
			"PostgreSQL vs MySQL 2024：功能、性能与选型全面对比",
			"PostgreSQL 和 MySQL 是最流行的开源数据库。本文从功能特性、性能表现、生态工具、云支持等维度进行全面对比，帮助你做出正确的技术选型。",
			`# PostgreSQL vs MySQL 2024

## 功能对比

| 特性 | PostgreSQL | MySQL |
|------|-----------|-------|
| JSON 支持 | 原生 JSONB | JSON 类型 |
| 全文搜索 | 内置 | 内置 |
| 窗口函数 | 完整支持 | 8.0 支持 |
| CTE | 递归 CTE | 8.0 支持 |

## 性能对比

![性能对比图](https://picsum.photos/800/400?random=603)

OLTP 场景两者差距不大，OLAP 场景 PostgreSQL 更优。

## 生态工具

MySQL 的工具链更成熟，PostgreSQL 的扩展更丰富。

## 云支持

主流云平台均提供托管服务，如 RDS、Cloud SQL。

## 选型建议

- 传统 Web 应用：MySQL
- 复杂查询/数据分析：PostgreSQL
- GIS 应用：PostgreSQL + PostGIS

## 总结

两者都是优秀的数据库，选择取决于具体需求。`,
			[]string{"MySQL", "Backend", "Cloud"},
		},
		{
			"MySQL 事务与锁深度解析：MVCC、间隙锁与死锁排查",
			"事务和锁是数据库的核心概念。本文深入讲解 MySQL InnoDB 的 MVCC 实现、锁类型（记录锁、间隙锁、临键锁）以及死锁排查方法。",
			`# MySQL 事务与锁深度解析

## MVCC 实现

InnoDB 通过 undo log 实现多版本并发控制：

` + "```sql" + `
BEGIN;
SELECT * FROM accounts WHERE id = 1;
-- 此时其他事务修改 id=1 的记录
-- 当前事务仍能看到修改前的数据
COMMIT;
` + "```" + `

## 锁类型

![锁类型关系](https://picsum.photos/800/400?random=604)

` + "```sql" + `
-- 记录锁
SELECT * FROM users WHERE id = 1 FOR UPDATE;

-- 间隙锁
SELECT * FROM users WHERE id > 10 FOR UPDATE;

-- 临键锁（记录锁 + 间隙锁）
SELECT * FROM users WHERE id BETWEEN 5 AND 15 FOR UPDATE;
` + "```" + `

## 死锁排查

` + "```sql" + `
SHOW ENGINE INNODB STATUS;
-- 查看死锁日志
SELECT * FROM information_schema.innodb_lock_waits;
` + "```" + `

## 总结

理解锁机制是避免并发问题的关键。`,
			[]string{"MySQL", "Backend"},
		},
		{
			"Redis 高可用架构：Sentinel、Cluster 与 Codis 方案对比",
			"Redis 高可用是生产环境的必备能力。本文对比 Sentinel、Redis Cluster 和 Codis 三种高可用方案的架构、优缺点和适用场景。",
			`# Redis 高可用架构

## Sentinel

` + "```conf" + `
sentinel monitor mymaster 127.0.0.1 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 60000
` + "```" + `

适用场景：读写分离、自动故障转移。

## Redis Cluster

![Cluster 架构](https://picsum.photos/800/400?random=605)

` + "```bash" + `
redis-cli --cluster create \
  192.168.1.1:6379 192.168.1.2:6379 \
  192.168.1.3:6379 192.168.1.4:6379 \
  192.168.1.5:6379 192.168.1.6:6379 \
  --cluster-replicas 1
` + "```" + `

## Codis

豌豆荚开源的 Redis 集群方案，通过 Proxy 层实现透明路由。

## 方案对比

- Sentinel：简单，适合中小规模
- Cluster：原生支持，适合大规模
- Codis：兼容性好，但已停止维护

## 总结

根据业务规模选择合适的高可用方案。`,
			[]string{"Redis", "Backend", "DevOps"},
		},
	}

	var allArticles []model.Article
	for _, a := range arts {
		now := time.Now()
		publishedAt := now.Add(-time.Duration(r.Intn(60)+1) * 24 * time.Hour)
		article := model.Article{
			AuthorID:       userIDs[r.Intn(len(userIDs))],
			CategoryID:     firstCat.ID,
			Title:          a.title,
			Summary:        a.summary,
			Content:        a.content,
			Status:         model.ArticleStatusPublished,
			Views:          r.Intn(9900) + 100,
			LikesCount:     r.Intn(490) + 10,
			FavoritesCount: r.Intn(195) + 5,
			CommentsCount:  0,
			PublishedAt:    &publishedAt,
		}
		if err := db.Create(&article).Error; err != nil {
			log.Printf("创建文章 %s 失败: %v", a.title, err)
			continue
		}
		allArticles = append(allArticles, article)

		for _, tagName := range a.tags {
			if tagID, ok := tagMap[tagName]; ok {
				db.Create(&model.ArticleTag{ArticleID: article.ID, TagID: tagID})
				db.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", gorm.Expr("usage_count + 1"))
			}
		}
	}
}

func createComments(db *gorm.DB, r *rand.Rand, userIDs []uint) {
	fmt.Println("创建文章评论...")
	var articles []model.Article
	db.Find(&articles)
	if len(articles) == 0 {
		return
	}

	topComments := []string{
		"写得很详细，收藏了！",
		"正好在找这方面的资料，感谢分享。",
		"请问这个方案在生产环境中验证过吗？",
		"代码示例很清晰，一下就理解了。",
		"有没有遇到性能瓶颈的情况？",
		"这个思路很棒，不过我觉得还可以考虑用另一种方案。",
		"终于有人把这个讲清楚了，之前看官方文档一头雾水。",
		"建议补充一下错误处理的部分，实际开发中很重要。",
		"我们团队正在用这个方案，确实效果不错。",
		"图片加载不出来，能重新上传一下吗？",
		"版本兼容性问题怎么解决？我在旧版本上跑不通。",
		"这篇比市面上大部分教程都讲得好。",
		"想问下作者在实际项目中用的哪个版本？",
		"配置部分的参数能详细解释一下吗？",
		"已按照教程操作，成功跑通了，感谢！",
		"这个坑我之前也踩过，当时折腾了好久。",
		"有没有对应的 GitHub 仓库可以参考？",
		"期待后续文章，这个系列很有价值。",
		"安全方面有什么建议吗？生产环境需要注意什么？",
		"和之前的方案相比，优势在哪里？",
		"测试覆盖了吗？单元测试怎么写？",
		"这个方案的扩展性怎么样？",
		"能否分享一下监控和告警的配置？",
		"小白表示看懂了，写得通俗易懂。",
		"生产环境建议加上限流和熔断。",
		"有没有 Kubernetes 上的部署示例？",
		"这个工具确实好用，我们团队也在用。",
		"性能数据能分享一下吗？QPS 大概多少？",
		"文章中的依赖版本有点旧，新版有些 API 变了。",
		"总结得很全面，适合收藏反复看。",
		"请问这个方案支持水平扩展吗？",
		"踩过这个坑，后来发现是配置问题。",
		"代码风格很规范，值得学习。",
		"建议加上 Docker 部署的部分，更完整。",
		"这个架构设计很巧妙，学到了。",
		"有没有遇到并发问题？怎么解决的？",
		"文档写得很用心，点赞！",
		"这个方案的成本大概是多少？",
		"能对比一下其他方案吗？",
		"正在做技术选型，这篇文章帮了大忙。",
		"生产环境跑了半年，没出过问题。",
		"有个小错误，第三段代码少了一个 import。",
		"这个优化效果太明显了，性能提升了 10 倍。",
		"新手建议先看官方文档再来读这篇文章。",
		"请问有视频教程吗？",
		"这个系列可以出一本书了。",
		"补充一下：最新版本已经支持热更新了。",
		"我们线上用的就是这个方案，稳定可靠。",
		"这个设计模式在其他语言中也适用吗？",
		"作者是在什么业务场景下使用这个方案的？",
		"配置热更新是怎么实现的？",
		"有没有压测数据？想看看极限性能。",
		"这个方案的运维成本高吗？",
		"非常实用的分享，已转发到团队群。",
		"建议补充一下回滚方案。",
		"这个思路打开了我的视野，原来还能这样做。",
		"代码可以直接复制使用吗？有许可证吗？",
		"这个方案在高并发场景下表现如何？",
		"写得很系统，从原理到实践都覆盖了。",
		"期待作者分享更多实战经验。",
		"这个方案在微服务架构下怎么使用？",
		"踩过坑提醒一下：注意内存泄漏问题。",
		"这篇文章解决了我困扰好几天的问题。",
		"有没有 Windows 环境下的部署指南？",
		"这个工具的学习曲线陡不陡？",
		"生产环境建议多做灰度发布。",
		"这个方案的容灾能力怎么样？",
		"代码中的注释很详细，好评。",
		"这个架构的瓶颈在哪里？",
		"终于等到这个主题的文章了。",
		"补充一点：记得开启日志审计功能。",
		"这个方案的成本效益比很高。",
		"请问作者对这个领域的新趋势怎么看？",
		"这个系列的文章质量都很高。",
		"有没有遇到数据一致性的问题？",
		"这个方案支持多租户吗？",
		"感谢分享，对我帮助很大。",
		"这个方案的监控告警怎么做？",
		"建议加上安全加固的部分。",
		"这个设计很优雅，值得借鉴。",
	}

	replies := []string{
		"是的，我们线上跑了大半年，很稳定。",
		"可以参考官方文档的迁移指南。",
		"你说得对，我补充一下：还需要考虑兼容性。",
		"性能方面我们做过压测，QPS 可以到 5000+。",
		"图片已经更新，感谢反馈。",
		"版本是 v1.21，建议用最新版。",
		"GitHub 仓库我整理一下发出来。",
		"配置参数的详细说明我后续会补充。",
		"安全方面建议开启 TLS 和访问控制。",
		"扩展性没问题，我们一直在加新功能。",
		"是的，新版 API 有 breaking changes。",
		"单元测试用的是 testify，后续会出教程。",
		"成本主要看云服务商，自建的话服务器成本不高。",
		"水平扩展没问题，加节点就行。",
		"并发问题用锁解决了，后续文章会详细讲。",
		"微服务场景下建议用服务网格来管理。",
		"是的，注意 goroutine 泄漏。",
		"压测数据后续会补充到文章中。",
		"运维成本不高，自动化程度很高。",
		"新趋势我比较看好 AI 辅助开发。",
	}

	var allTopComments []model.Comment
	for i, content := range topComments {
		article := articles[i%len(articles)]
		c := model.Comment{
			ArticleID:  article.ID,
			AuthorID:   userIDs[r.Intn(len(userIDs))],
			Content:    content,
			LikesCount: r.Intn(50),
		}
		db.Create(&c)
		allTopComments = append(allTopComments, c)
		db.Model(&model.Article{}).Where("id = ?", article.ID).Update("comments_count", gorm.Expr("comments_count + 1"))
	}

	replyIndices := r.Perm(len(allTopComments))[:20]
	for _, idx := range replyIndices {
		parent := allTopComments[idx]
		reply := model.Comment{
			ArticleID:  parent.ArticleID,
			AuthorID:   userIDs[r.Intn(len(userIDs))],
			ParentID:   &parent.ID,
			Content:    replies[r.Intn(len(replies))],
			LikesCount: r.Intn(20),
		}
		db.Create(&reply)
		db.Model(&model.Article{}).Where("id = ?", parent.ArticleID).Update("comments_count", gorm.Expr("comments_count + 1"))
	}
}

type mcpRaw struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RepoURL     string   `json:"repo_url"`
	Stars       int      `json:"stars"`
	Author      string   `json:"author"`
	Tools       []string `json:"tools"`
}

func createMcpServers(db *gorm.DB, r *rand.Rand, userIDs, tagIDs []uint, tagMap map[string]uint) {
	fmt.Println("创建 MCP Server...")
	raw, err := os.ReadFile("scripts/seed/mcp_servers_raw.json")
	if err != nil {
		log.Fatalf("读取 mcp_servers_raw.json 失败: %v", err)
	}
	var servers []mcpRaw
	if err := json.Unmarshal(raw, &servers); err != nil {
		log.Fatalf("解析 mcp_servers_raw.json 失败: %v", err)
	}

	serverTagMap := map[string][]string{
		"filesystem":     {"Backend", "DevOps"},
		"github":         {"Backend", "DevOps", "Microservices"},
		"postgres":       {"MySQL", "Backend"},
		"sqlite":         {"MySQL", "Backend"},
		"puppeteer":      {"Frontend", "DevOps"},
		"memory":         {"Backend", "AI"},
		"brave-search":   {"AI", "Backend"},
		"git":            {"DevOps", "Backend"},
		"google-maps":    {"Backend", "Cloud"},
		"slack":          {"Backend", "Microservices"},
		"fetch":          {"Backend", "Cloud"},
		"redis":          {"Redis", "Backend"},
		"docker":         {"Docker", "DevOps"},
		"kubernetes":     {"Kubernetes", "DevOps", "Cloud"},
		"notion":         {"Backend", "Frontend"},
		"obsidian":       {"Backend"},
		"linear":         {"Backend", "Microservices"},
		"jira":           {"Backend", "DevOps"},
		"discord":        {"Backend", "Microservices"},
		"twitter":        {"Backend", "Cloud"},
		"openai":         {"AI", "LLM", "Python"},
		"anthropic":      {"AI", "LLM"},
		"sentry":         {"DevOps", "Backend"},
		"datadog":        {"DevOps", "Cloud"},
		"aws":            {"Cloud", "DevOps"},
		"gcp":            {"Cloud", "DevOps"},
		"azure":          {"Cloud", "DevOps"},
		"mongodb":        {"Backend", "MySQL"},
		"elasticsearch":  {"Backend", "DevOps"},
		"mysql":          {"MySQL", "Backend"},
	}

	for _, s := range servers {
		toolsJSON, _ := json.Marshal(s.Tools)
		now := time.Now()
		publishedAt := now.Add(-time.Duration(r.Intn(60)+1) * 24 * time.Hour)
		mcp := model.McpServer{
			AuthorID:       userIDs[r.Intn(len(userIDs))],
			Name:           s.Name,
			Description:    s.Description,
			RepoURL:        s.RepoURL,
			ToolsJSON:      string(toolsJSON),
			Readme:         genMcpReadme(s.Name, s.Description, s.Tools),
			Status:         model.ResourceStatusPublished,
			Views:          r.Intn(9900) + 100,
			Downloads:      r.Intn(4900) + 100,
			LikesCount:     r.Intn(490) + 10,
			FavoritesCount: r.Intn(195) + 5,
			CommentsCount:  0,
			PublishedAt:    &publishedAt,
		}
		if err := db.Create(&mcp).Error; err != nil {
			log.Printf("创建 MCP Server %s 失败: %v", s.Name, err)
			continue
		}

		for _, tagName := range serverTagMap[s.Name] {
			if tagID, ok := tagMap[tagName]; ok {
				db.Create(&model.McpServerTag{McpServerID: mcp.ID, TagID: tagID})
				db.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", gorm.Expr("usage_count + 1"))
			}
		}
	}
}

type skillRaw struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RepoURL     string   `json:"repo_url"`
	Stars       int      `json:"stars"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

func createSkills(db *gorm.DB, r *rand.Rand, userIDs, tagIDs []uint, tagMap map[string]uint) {
	fmt.Println("创建 Skills...")
	raw, err := os.ReadFile("scripts/seed/skills_raw.json")
	if err != nil {
		log.Fatalf("读取 skills_raw.json 失败: %v", err)
	}
	var skills []skillRaw
	if err := json.Unmarshal(raw, &skills); err != nil {
		log.Fatalf("解析 skills_raw.json 失败: %v", err)
	}

	skillTagMap := map[string][]string{
		"code-reviewer":       {"Backend", "Go"},
		"test-generator":      {"Backend", "DevOps"},
		"refactor-helper":     {"Backend", "Go"},
		"documentation-writer":{"Backend"},
		"bug-finder":          {"Backend", "DevOps"},
		"performance-optimizer":{"Go", "Backend"},
		"security-auditor":    {"Backend", "Cloud"},
		"api-designer":        {"GraphQL", "Backend", "Microservices"},
		"database-designer":   {"MySQL", "Backend"},
		"git-workflow":        {"DevOps"},
		"docker-expert":       {"Docker", "DevOps"},
		"kubernetes-expert":   {"Kubernetes", "DevOps", "Cloud"},
		"ci-cd-pipeline":      {"DevOps", "Cloud"},
		"frontend-developer":  {"Vue", "React", "Frontend"},
		"backend-developer":   {"Go", "Backend", "Python"},
		"fullstack-developer": {"Vue", "React", "TypeScript"},
		"mobile-developer":    {"Frontend", "React"},
		"data-scientist":      {"AI", "LLM", "Python"},
		"devops-engineer":     {"DevOps", "Cloud", "Docker"},
		"cloud-architect":     {"Cloud", "DevOps"},
		"microservices-expert":{"Microservices", "Go", "Backend"},
		"graphql-expert":      {"GraphQL", "Backend"},
		"typescript-expert":   {"TypeScript", "Frontend"},
		"rust-developer":      {"Rust", "Backend"},
		"go-developer":        {"Go", "Backend", "Gin"},
		"python-expert":       {"Python", "Backend", "AI"},
		"java-developer":      {"Java", "Backend"},
		"react-expert":        {"React", "Frontend", "TypeScript"},
		"vue-expert":          {"Vue", "Frontend", "TypeScript"},
		"angular-expert":      {"Frontend", "TypeScript"},
	}

	for _, s := range skills {
		now := time.Now()
		publishedAt := now.Add(-time.Duration(r.Intn(60)+1) * 24 * time.Hour)
		skill := model.Skill{
			AuthorID:       userIDs[r.Intn(len(userIDs))],
			Name:           s.Name,
			Description:    s.Description,
			RepoURL:        s.RepoURL,
			Status:         model.ResourceStatusPublished,
			Views:          r.Intn(9900) + 100,
			Downloads:      r.Intn(4900) + 100,
			LikesCount:     r.Intn(490) + 10,
			FavoritesCount: r.Intn(195) + 5,
			CommentsCount:  0,
			PublishedAt:    &publishedAt,
		}
		if err := db.Create(&skill).Error; err != nil {
			log.Printf("创建 Skill %s 失败: %v", s.Name, err)
			continue
		}

		for _, tagName := range skillTagMap[s.Name] {
			if tagID, ok := tagMap[tagName]; ok {
				db.Create(&model.SkillTag{SkillID: skill.ID, TagID: tagID})
				db.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", gorm.Expr("usage_count + 1"))
			}
		}
	}
}

func createResourceComments(db *gorm.DB, r *rand.Rand, userIDs []uint) {
	fmt.Println("创建资源评论...")
	var mcpServers []model.McpServer
	db.Find(&mcpServers)
	var skills []model.Skill
	db.Find(&skills)

	mcpComments := []string{
		"这个 MCP Server 非常好用，推荐！",
		"配置简单，几分钟就跑通了。",
		"和 Claude Code 配合使用效果很好。",
		"有没有 Python 版本的客户端？",
		"工具列表很丰富，覆盖了大部分场景。",
		"性能不错，响应很快。",
		"文档写得很清楚，上手容易。",
		"建议加上认证功能，生产环境需要。",
		"和官方版本相比有什么优势？",
		"已经用了一个月，很稳定。",
		"安装过程遇到了一些问题，希望能完善文档。",
		"这个工具的返回格式能自定义吗？",
		"和其他同类工具相比，这个更轻量。",
		"支持批量操作吗？",
		"错误处理做得不错，提示信息很明确。",
		"希望能支持更多的协议。",
		"社区活跃度怎么样？有微信群吗？",
		"这个方案的安全性如何？",
		"配合 LangChain 使用效果很好。",
		"版本更新频率怎么样？",
		"内存占用大吗？",
		"有没有 Docker 镜像可以直接用？",
		"这个工具的并发性能如何？",
		"建议加上日志输出功能。",
		"已经集成到我们的项目中了，感谢开源。",
	}

	skillComments := []string{
		"这个 Skill 太实用了，效率提升明显。",
		"代码质量检查很到位，发现了不少问题。",
		"配置灵活，可以根据项目需求调整。",
		"有没有视频教程？",
		"和现有的 CI/CD 流程集成很方便。",
		"生成的代码质量很高。",
		"建议加上自定义规则的功能。",
		"学习曲线不陡，很快上手了。",
		"这个 Skill 的文档很完善。",
		"团队已经全面采用了，好评。",
		"支持自定义模板吗？",
		"和其他类似工具相比，这个更全面。",
		"性能优化建议：可以加缓存。",
		"错误提示很友好，容易定位问题。",
		"希望能支持更多的编程语言。",
		"这个 Skill 帮我节省了很多时间。",
		"有没有企业版？需要更多功能。",
		"集成测试覆盖了吗？",
		"和 IDE 的集成做得很好。",
		"社区支持很及时，问题很快得到解决。",
		"这个方案的可扩展性很强。",
		"建议加上性能分析功能。",
		"已经推荐给同事了。",
		"更新很及时，bug 修复很快。",
		"这个 Skill 的设计很优雅。",
	}

	for i, mcp := range mcpServers {
		if i >= len(mcpComments) {
			break
		}
		rc := model.ResourceComment{
			ResourceType: "mcp_server",
			ResourceID:   mcp.ID,
			AuthorID:     userIDs[r.Intn(len(userIDs))],
			Content:      mcpComments[i],
			LikesCount:   r.Intn(30),
		}
		db.Create(&rc)
		db.Model(&model.McpServer{}).Where("id = ?", mcp.ID).Update("comments_count", gorm.Expr("comments_count + 1"))
	}

	for i, skill := range skills {
		if i >= len(skillComments) {
			break
		}
		rc := model.ResourceComment{
			ResourceType: "skill",
			ResourceID:   skill.ID,
			AuthorID:     userIDs[r.Intn(len(userIDs))],
			Content:      skillComments[i],
			LikesCount:   r.Intn(30),
		}
		db.Create(&rc)
		db.Model(&model.Skill{}).Where("id = ?", skill.ID).Update("comments_count", gorm.Expr("comments_count + 1"))
	}
}

func genMcpReadme(name, desc string, tools []string) string {
	readme := fmt.Sprintf("# %s MCP Server\n\n%s\n\n", name, desc)
	readme += "## Installation\n\n"
	readme += "```bash\n"
	readme += fmt.Sprintf("npx @mcp/server-%s\n", name)
	readme += "```\n\n"
	readme += "## Configuration\n\n"
	readme += "```json\n"
	readme += "{\n"
	readme += "  \"mcpServers\": {\n"
	readme += fmt.Sprintf("    \"%s\": {\n", name)
	readme += fmt.Sprintf("      \"command\": \"npx\",\n")
	readme += fmt.Sprintf("      \"args\": [\"-y\", \"@mcp/server-%s\"]\n", name)
	readme += "    }\n"
	readme += "  }\n"
	readme += "}\n"
	readme += "```\n\n"
	readme += "## Tools\n\n"
	for _, t := range tools {
		readme += fmt.Sprintf("- `%s`\n", t)
	}
	return readme
}

func genSkillReadme(name, desc string) string {
	readme := fmt.Sprintf("# %s Skill\n\n%s\n\n", name, desc)
	readme += "## Installation\n\n"
	readme += fmt.Sprintf("1. Download the skill package\n")
	readme += fmt.Sprintf("2. Extract to your `.claude/skills/%s/` directory\n", name)
	readme += fmt.Sprintf("3. Ensure `SKILL.md` is present in the directory\n\n")
	readme += "## Usage\n\n"
	readme += fmt.Sprintf("Activate this skill by mentioning `%s` in your conversation.\n", name)
	return readme
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func printSummary(db *gorm.DB) {
	var userCount, articleCount, commentCount, mcpCount, skillCount, tagCount int64
	db.Model(&model.User{}).Count(&userCount)
	db.Model(&model.Article{}).Count(&articleCount)
	db.Model(&model.Comment{}).Count(&commentCount)
	db.Model(&model.McpServer{}).Count(&mcpCount)
	db.Model(&model.Skill{}).Count(&skillCount)
	db.Model(&model.Tag{}).Count(&tagCount)

	var resCommentCount int64
	db.Model(&model.ResourceComment{}).Count(&resCommentCount)

	fmt.Println("\n===== 种子数据统计 =====")
	fmt.Printf("用户: %d\n", userCount)
	fmt.Printf("标签: %d\n", tagCount)
	fmt.Printf("文章: %d\n", articleCount)
	fmt.Printf("文章评论: %d\n", commentCount)
	fmt.Printf("MCP Server: %d\n", mcpCount)
	fmt.Printf("Skills: %d\n", skillCount)
	fmt.Printf("资源评论: %d\n", resCommentCount)
	fmt.Println("\n===== 测试用户账号 =====")
	var users []model.User
	db.Find(&users)
	for _, u := range users {
		fmt.Printf("  %s / 123456 (%s)\n", u.Email, u.Nickname)
	}
	fmt.Println("\n测试数据创建完成！")
}