package service

import (
	"sort"
	"strings"
	"unicode"
)

type ChunkOptions struct {
	TargetTokens  int
	MaxTokens     int
	OverlapTokens int
}

type MarkdownChunk struct {
	Text        string
	HeadingPath []string
}

// ChunkMarkdown groups Markdown by headings and only splits a section when it
// exceeds MaxTokens. Token estimation is deliberately deterministic and cheap:
// each Han rune is one token and a contiguous non-Han word is one token.
func ChunkMarkdown(markdown string, options ChunkOptions) []MarkdownChunk {
	if options.TargetTokens <= 0 {
		options.TargetTokens = 550
	}
	if options.MaxTokens <= 0 {
		options.MaxTokens = 800
	}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var chunks []MarkdownChunk
	path := make([]string, 0, 6)
	var section []string
	flush := func() {
		text := strings.TrimSpace(strings.Join(section, "\n"))
		if text == "" {
			return
		}
		for _, part := range splitChunk(text, options) {
			chunks = append(chunks, MarkdownChunk{Text: part, HeadingPath: append([]string(nil), path...)})
		}
		section = nil
	}
	for _, line := range lines {
		level, title, ok := markdownHeading(line)
		if ok {
			flush()
			for len(path) >= level {
				path = path[:len(path)-1]
			}
			path = append(path, title)
			section = append(section, line)
			continue
		}
		section = append(section, line)
	}
	flush()
	return chunks
}

func markdownHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || len(trimmed) == level || trimmed[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(trimmed[level:])
	return level, title, title != ""
}

func splitChunk(text string, options ChunkOptions) []string {
	if estimateTokens(text) <= options.MaxTokens {
		return []string{text}
	}
	paragraphs := strings.Split(text, "\n\n")
	var out []string
	var current []string
	currentTokens := 0
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		count := estimateTokens(paragraph)
		if currentTokens > 0 && currentTokens+count > options.TargetTokens {
			out = append(out, strings.Join(current, "\n\n"))
			current, currentTokens = nil, 0
		}
		if count > options.MaxTokens {
			for _, line := range strings.Split(paragraph, "\n") {
				if currentTokens > 0 && currentTokens+estimateTokens(line) > options.TargetTokens {
					out = append(out, strings.Join(current, "\n"))
					current, currentTokens = nil, 0
				}
				current = append(current, line)
				currentTokens += estimateTokens(line)
			}
			continue
		}
		current = append(current, paragraph)
		currentTokens += count
	}
	if len(current) > 0 {
		out = append(out, strings.Join(current, "\n\n"))
	}
	return out
}

func estimateTokens(text string) int {
	count, inWord := 0, false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			count++
			inWord = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			if !inWord {
				count++
				inWord = true
			}
			continue
		}
		inWord = false
	}
	return count
}

type SparseVector struct {
	Indices []uint32  `json:"indices"`
	Values  []float32 `json:"values"`
}

// EncodeLexical generates a stable sparse vector without a mutable vocabulary.
func EncodeLexical(text string) SparseVector {
	counts := map[uint32]int{}
	for _, token := range lexicalTokens(text) {
		counts[fnv32(token)]++
	}
	indices := make([]uint32, 0, len(counts))
	for index := range counts {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
	values := make([]float32, len(indices))
	for i, index := range indices {
		values[i] = 1
		if counts[index] > 1 {
			values[i] += float32(counts[index]-1) * 0.5
		}
	}
	return SparseVector{Indices: indices, Values: values}
}

func lexicalTokens(text string) []string {
	runes := []rune(strings.ToLower(text))
	var tokens []string
	for i := 0; i < len(runes); {
		if unicode.Is(unicode.Han, runes[i]) {
			start := i
			for i < len(runes) && unicode.Is(unicode.Han, runes[i]) {
				i++
			}
			for j := start; j < i; j++ {
				end := j + 2
				if end > i {
					end = i
				}
				tokens = append(tokens, string(runes[j:end]))
			}
			continue
		}
		if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_' || runes[i] == '-' || runes[i] == '.' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_' || runes[i] == '-' || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, string(runes[start:i]))
			continue
		}
		i++
	}
	return tokens
}

func fnv32(value string) uint32 {
	const offset uint32 = 2166136261
	const prime uint32 = 16777619
	hash := offset
	for _, b := range []byte(value) {
		hash ^= uint32(b)
		hash *= prime
	}
	return hash
}
