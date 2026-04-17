package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// aiTerms are lowercase substrings matched against normalized story titles.
// Titles are lowercased and have hyphens/slashes replaced with spaces before
// matching, so single-word terms catch hyphenated variants (e.g. "llm-based").
var aiTerms = []string{
	// Generic concepts
	"artificial intelligence",
	"machine learning",
	"deep learning",
	"neural network",
	"large language model",
	"generative ai",
	"natural language processing",
	"foundation model",
	"diffusion model",
	"retrieval augmented",
	"prompt engineering",
	"fine tuning",
	"finetuning",
	"vector database",
	"embedding model",
	"agentic",
	"ai agent",
	"ai model",
	"ai system",
	"ai tool",
	"vibe coding",
	"attention mechanism",
	// Acronyms – matched as whole words via the surrounding-space trick
	" ai ",
	" llm ",
	" llms ",
	" agi ",
	" rlhf ",
	" nlp ",
	" genai ",
	// Companies
	"openai",
	"anthropic",
	"deepmind",
	"mistral",
	"cohere",
	"hugging face",
	"stability ai",
	"inflection ai",
	"xai",
	// Products / models
	"chatgpt",
	"gpt 4",
	"gpt 3",
	"gpt4",
	"gpt3",
	"claude",
	"gemini",
	"llama",
	"copilot",
	"midjourney",
	"dall e",
	"stable diffusion",
	"whisper",
	"grok",
	"mixtral",
	"perplexity",
	"devin",
	"cursor",
	"phi 3",
	"phi 4",
	"o1",
	"o3",
}

// matchesAI reports whether title contains an AI-related term.
// Matching is case-insensitive; hyphens, slashes, and dots are treated as
// word boundaries so "LLM-based" still matches "llm".
func matchesAI(title string) bool {
	normalized := " " + strings.ToLower(strings.NewReplacer(
		"-", " ", "/", " ", ":", " ", "·", " ", ".", " ",
	).Replace(title)) + " "
	for _, term := range aiTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

// generateFiltered fetches up to 3× the requested count, applies keep to each
// hit, trims to the requested count, and renders the feed.
func generateFiltered(c *gin.Context, sp *SearchParams, op *OutputParams, keep func(AlgoliaSearchHit) bool) {
	if op.Format == "" {
		op.Format = "rss"
	}

	requested := 20
	if sp.Count != "" {
		if n, err := strconv.Atoi(sp.Count); err == nil && n > 0 {
			requested = n
		}
	}
	fetch := requested * 3
	if fetch > HitsPerPageLimit {
		fetch = HitsPerPageLimit
	}
	sp.Count = strconv.Itoa(fetch)

	results, err := GetResults(sp.Values())
	if err != nil {
		c.Error(err)
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	c.Header("X-Algolia-URL", algoliaSearchURL+sp.Values().Encode())

	var filtered []AlgoliaSearchHit
	for _, hit := range results.Hits {
		if keep(hit) {
			filtered = append(filtered, hit)
		}
	}
	if len(filtered) > requested {
		filtered = filtered[:requested]
	}
	results.Hits = filtered

	if len(results.Hits) > 0 {
		c.Header("Last-Modified", Timestamp("http", results.Hits[0].GetCreatedAt()))
	}

	switch op.Format {
	case "rss":
		c.XML(http.StatusOK, NewRSS(results, op))
	case "atom":
		c.XML(http.StatusOK, NewAtom(results, op))
	case "jsonfeed":
		c.JSON(http.StatusOK, NewJSONFeed(results, op))
	}
}

// algoliaAIPreFilter is a small set of high-signal AI keywords sent to Algolia
// as optionalWords so the API pre-filters before client-side matchesAI runs.
const algoliaAIPreFilter = "AI LLM ChatGPT OpenAI Anthropic Claude Gemini Llama DeepMind Mistral Grok Copilot"

func NewestAI(c *gin.Context) {
	var sp SearchParams
	var op OutputParams
	ParseRequest(c, &sp, &op)

	sp.Tags = "(story,poll)"
	op.Title = "Hacker News: Newest – AI"
	op.Link = "https://news.ycombinator.com/newest"

	// Pre-filter at the Algolia level: any story matching at least one keyword
	// in algoliaAIPreFilter is returned (optionalWords = OR semantics).
	// matchesAI() still runs client-side for precision.
	sp.Query = algoliaAIPreFilter
	sp.OptionalWords = algoliaAIPreFilter

	generateFiltered(c, &sp, &op, func(hit AlgoliaSearchHit) bool {
		return matchesAI(hit.GetTitle())
	})
}

func NewestNoAI(c *gin.Context) {
	var sp SearchParams
	var op OutputParams
	ParseRequest(c, &sp, &op)

	sp.Tags = "(story,poll)"
	op.Title = "Hacker News: Newest – No AI"
	op.Link = "https://news.ycombinator.com/newest"

	generateFiltered(c, &sp, &op, func(hit AlgoliaSearchHit) bool {
		return !matchesAI(hit.GetTitle())
	})
}
