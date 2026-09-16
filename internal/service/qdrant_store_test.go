package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeLexicalKeepsChineseBigramsAndTechnicalIdentifiers(t *testing.T) {
	vector := EncodeLexical("使用 singleflight.Group 防止缓存击穿")

	require.NotEmpty(t, vector.Indices)
	require.Len(t, vector.Indices, len(vector.Values))
	for i := 1; i < len(vector.Indices); i++ {
		require.Less(t, vector.Indices[i-1], vector.Indices[i])
	}
}

func TestChunkMarkdownKeepsHeadingContextAndWholeCodeBlock(t *testing.T) {
	chunks := ChunkMarkdown("# Redis 热榜\n\n```go\nvar group singleflight.Group\n```", ChunkOptions{TargetTokens: 450, MaxTokens: 800, OverlapTokens: 80})

	require.Len(t, chunks, 1)
	require.Equal(t, []string{"Redis 热榜"}, chunks[0].HeadingPath)
	require.Contains(t, chunks[0].Text, "singleflight.Group")
}
