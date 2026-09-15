package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AIAssistantConfig struct {
	APIKey           string
	EmbeddingURL     string
	EmbeddingModel   string
	ChatURL          string
	ChatModel        string
	RerankURL        string
	RerankModel      string
	MilvusURL        string
	MilvusCollection string
}

type AIAssistantService struct {
	cfg    AIAssistantConfig
	client *http.Client
}

// IndexArticle asynchronously prepares fixed-size Markdown chunks for Milvus.
func (s *AIAssistantService) IndexArticle(ctx context.Context, articleID uint, title, content string) error {
	if s.cfg.APIKey == "" || s.cfg.EmbeddingURL == "" || s.cfg.MilvusURL == "" {
		return fmt.Errorf("AI 索引服务未配置")
	}
	chunks := markdownChunks(content, 1200)
	if len(chunks) == 0 {
		chunks = []string{content}
	}
	inputs := make([]string, len(chunks))
	for i, c := range chunks {
		inputs[i] = c
	}
	var emb struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := s.post(ctx, s.cfg.EmbeddingURL, map[string]any{"model": s.cfg.EmbeddingModel, "input": inputs}, &emb); err != nil {
		return err
	}
	if len(emb.Data) != len(chunks) {
		return fmt.Errorf("embedding 分片数量不一致")
	}
	rows := make([]map[string]any, len(chunks))
	for i, chunk := range chunks {
		rows[i] = map[string]any{"article_id": articleID, "title": title, "heading_path": "", "chunk_id": fmt.Sprintf("%d-%d", articleID, i), "text": chunk, "embedding": emb.Data[i].Embedding}
	}
	var out map[string]any
	// Re-indexing an edited article must be idempotent; remove stale chunks first.
	if err := s.post(ctx, strings.TrimRight(s.cfg.MilvusURL, "/")+"/v2/vectordb/entities/delete", map[string]any{
		"collectionName": s.cfg.MilvusCollection,
		"filter":         "article_id == " + strconv.FormatUint(uint64(articleID), 10),
	}, &out); err != nil {
		return err
	}
	return s.post(ctx, strings.TrimRight(s.cfg.MilvusURL, "/")+"/v2/vectordb/entities/insert", map[string]any{"collectionName": s.cfg.MilvusCollection, "data": rows}, &out)
}

func markdownChunks(text string, size int) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := []string{}
	cur := ""
	for _, line := range lines {
		if cur != "" && len(cur)+len(line)+1 > size {
			out = append(out, cur)
			cur = ""
		}
		if cur != "" {
			cur += "\n"
		}
		cur += line
	}
	if strings.TrimSpace(cur) != "" {
		out = append(out, cur)
	}
	return out
}

type AICitation struct {
	ArticleID   uint   `json:"article_id,omitempty"`
	Title       string `json:"title,omitempty"`
	HeadingPath string `json:"heading_path,omitempty"`
	ChunkID     string `json:"chunk_id,omitempty"`
	Text        string `json:"text,omitempty"`
}
type AIAssistantAnswer struct {
	Answer    string       `json:"answer"`
	Citations []AICitation `json:"citations,omitempty"`
}

func NewAIAssistantService(cfg AIAssistantConfig) *AIAssistantService {
	return &AIAssistantService{cfg: cfg, client: &http.Client{Timeout: 45 * time.Second}}
}

func (s *AIAssistantService) Ask(ctx context.Context, question string, articleID *uint) (*AIAssistantAnswer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("问题不能为空")
	}
	if s.cfg.APIKey == "" || s.cfg.EmbeddingURL == "" || s.cfg.ChatURL == "" || s.cfg.MilvusURL == "" {
		return nil, fmt.Errorf("AI 助手服务未配置")
	}
	vector, err := s.embedding(ctx, question)
	if err != nil {
		return nil, err
	}
	hits, err := s.search(ctx, vector, articleID)
	if err != nil {
		return nil, err
	}
	if s.cfg.RerankURL != "" && len(hits) > 1 {
		hits = s.rerank(ctx, question, hits)
	}
	answer, err := s.chat(ctx, question, hits)
	if err != nil {
		return nil, err
	}
	citations := make([]AICitation, 0, len(hits))
	for _, h := range hits {
		citations = append(citations, h.citation)
	}
	return &AIAssistantAnswer{Answer: answer, Citations: citations}, nil
}

type aiHit struct {
	citation AICitation
	score    float64
}

func (s *AIAssistantService) post(ctx context.Context, url string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("AI 服务返回 HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *AIAssistantService) embedding(ctx context.Context, text string) ([]float64, error) {
	var out struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	err := s.post(ctx, s.cfg.EmbeddingURL, map[string]any{"model": s.cfg.EmbeddingModel, "input": []string{text}}, &out)
	if err != nil || len(out.Data) == 0 {
		if err == nil {
			err = fmt.Errorf("embedding 响应为空")
		}
		return nil, err
	}
	return out.Data[0].Embedding, nil
}

func (s *AIAssistantService) search(ctx context.Context, vector []float64, articleID *uint) ([]aiHit, error) {
	filter := ""
	if articleID != nil {
		filter = "article_id == " + strconv.FormatUint(uint64(*articleID), 10)
	}
	body := map[string]any{"collectionName": s.cfg.MilvusCollection, "data": [][]float64{vector}, "annsField": "embedding", "limit": 8, "outputFields": []string{"article_id", "title", "heading_path", "chunk_id", "text"}}
	if filter != "" {
		body["filter"] = filter
	}
	var out struct {
		Data []struct {
			ID       any            `json:"id"`
			Distance float64        `json:"distance"`
			Entity   map[string]any `json:"entity"`
		} `json:"data"`
	}
	if err := s.post(ctx, strings.TrimRight(s.cfg.MilvusURL, "/")+"/v2/vectordb/entities/search", body, &out); err != nil {
		return nil, err
	}
	hits := make([]aiHit, 0, len(out.Data))
	for _, row := range out.Data {
		e := row.Entity
		h := aiHit{score: row.Distance}
		h.citation.ArticleID = number(e["article_id"])
		h.citation.Title = stringValue(e["title"])
		h.citation.HeadingPath = stringValue(e["heading_path"])
		h.citation.ChunkID = stringValue(e["chunk_id"])
		h.citation.Text = stringValue(e["text"])
		if h.citation.Text != "" {
			hits = append(hits, h)
		}
	}
	return hits, nil
}

func (s *AIAssistantService) rerank(ctx context.Context, query string, hits []aiHit) []aiHit {
	docs := make([]string, len(hits))
	for i, h := range hits {
		docs[i] = h.citation.Text
	}
	var out struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if s.post(ctx, s.cfg.RerankURL, map[string]any{"model": s.cfg.RerankModel, "query": query, "documents": docs}, &out) != nil || len(out.Results) == 0 {
		return hits
	}
	ranked := make([]aiHit, 0, len(out.Results))
	for _, r := range out.Results {
		if r.Index >= 0 && r.Index < len(hits) {
			h := hits[r.Index]
			h.score = r.RelevanceScore
			ranked = append(ranked, h)
		}
	}
	return ranked
}

func (s *AIAssistantService) chat(ctx context.Context, question string, hits []aiHit) (string, error) {
	var b strings.Builder
	for i, h := range hits {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, h.citation.Text)
	}
	prompt := "请仅依据下方帖子内容回答问题。如果资料不足，请明确说不知道。回答简洁并保留关键步骤。\n\n资料：\n" + b.String()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err := s.post(ctx, s.cfg.ChatURL, map[string]any{"model": s.cfg.ChatModel, "messages": []map[string]string{{"role": "system", "content": prompt}, {"role": "user", "content": question}}, "temperature": 0.2}, &out)
	if err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("大模型响应为空")
	}
	return out.Choices[0].Message.Content, nil
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func number(v any) uint {
	switch n := v.(type) {
	case float64:
		return uint(n)
	case json.Number:
		i, _ := strconv.ParseUint(string(n), 10, 64)
		return uint(i)
	case string:
		i, _ := strconv.ParseUint(n, 10, 64)
		return uint(i)
	}
	return 0
}
