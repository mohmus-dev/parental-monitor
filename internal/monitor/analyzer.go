package monitor

import (
	"sort"
	"strings"
	"unicode"
)

type KeywordMatch struct {
	Category string
	Keyword  string
}

type analyzerGroup struct {
	category string
	keywords []string
}

type patternOutput struct {
	categoryIndex int
	keywordIndex  int
	keyword       string
}

type trieNode struct {
	next    map[int]int
	fail    int
	output  int
	outputs []patternOutput
}

type Analyzer struct {
	nodes      []trieNode
	tokenIDs   map[string]int
	categories []string
}

func MergeKeywordGroups(groupSets ...map[string][]string) map[string][]string {
	merged := make(map[string][]string)
	seen := make(map[string]map[string]struct{})
	for _, groupSet := range groupSets {
		for category, terms := range groupSet {
			category = strings.ToLower(strings.TrimSpace(category))
			if category == "" {
				continue
			}
			if seen[category] == nil {
				seen[category] = make(map[string]struct{})
			}
			for _, term := range terms {
				term = strings.TrimSpace(term)
				key := strings.ToLower(term)
				if term == "" {
					continue
				}
				if _, exists := seen[category][key]; exists {
					continue
				}
				seen[category][key] = struct{}{}
				merged[category] = append(merged[category], term)
			}
		}
	}
	return merged
}

func NewAnalyzer(keywords []string) *Analyzer {
	return NewGroupedAnalyzer(keywords, nil)
}

func NewGroupedAnalyzer(keywords []string, configuredGroups map[string][]string) *Analyzer {
	groupsByCategory := make(map[string][]string)
	configuredTerms := make(map[string]struct{})

	for category, terms := range configuredGroups {
		category = strings.TrimSpace(category)
		if category == "" {
			continue
		}

		seenInGroup := make(map[string]struct{})
		for _, term := range terms {
			term = strings.TrimSpace(term)
			key := strings.ToLower(term)
			if term == "" {
				continue
			}
			if _, exists := seenInGroup[key]; exists {
				continue
			}
			seenInGroup[key] = struct{}{}
			configuredTerms[key] = struct{}{}
			groupsByCategory[category] = append(groupsByCategory[category], term)
		}
	}

	seenFallback := make(map[string]struct{})
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		key := strings.ToLower(keyword)
		if keyword == "" {
			continue
		}
		if _, exists := configuredTerms[key]; exists {
			continue
		}
		if _, exists := seenFallback[key]; exists {
			continue
		}
		seenFallback[key] = struct{}{}
		groupsByCategory[keyword] = append(groupsByCategory[keyword], keyword)
	}

	categories := make([]string, 0, len(groupsByCategory))
	for category := range groupsByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	analyzer := &Analyzer{
		nodes:      []trieNode{{next: make(map[int]int)}},
		tokenIDs:   make(map[string]int),
		categories: categories,
	}
	for categoryIndex, category := range categories {
		for keywordIndex, keyword := range groupsByCategory[category] {
			words := keywordWords(keyword)
			if len(words) == 0 {
				continue
			}

			nodeIndex := 0
			for _, word := range words {
				tokenID := analyzer.internToken(word)
				nextIndex, exists := analyzer.nodes[nodeIndex].next[tokenID]
				if !exists {
					nextIndex = len(analyzer.nodes)
					analyzer.nodes = append(analyzer.nodes, trieNode{})
					if analyzer.nodes[nodeIndex].next == nil {
						analyzer.nodes[nodeIndex].next = make(map[int]int)
					}
					analyzer.nodes[nodeIndex].next[tokenID] = nextIndex
				}
				nodeIndex = nextIndex
			}
			analyzer.nodes[nodeIndex].outputs = append(analyzer.nodes[nodeIndex].outputs, patternOutput{
				categoryIndex: categoryIndex,
				keywordIndex:  keywordIndex,
				keyword:       keyword,
			})
		}
	}
	analyzer.buildFailureLinks()
	return analyzer
}

func (a *Analyzer) FindMatches(query string) []KeywordMatch {
	if len(a.nodes) == 0 {
		return nil
	}
	matched := make([]bool, len(a.categories))
	bestMatches := make([]patternOutput, len(a.categories))
	state := 0

	for _, word := range keywordWords(query) {
		tokenID := a.tokenIDs[word]
		if tokenID == 0 {
			state = 0
			continue
		}

		for {
			if next, exists := a.nodes[state].next[tokenID]; exists {
				state = next
				break
			}
			if state == 0 {
				break
			}
			state = a.nodes[state].fail
		}

		for outputState := state; outputState != 0; outputState = a.nodes[outputState].output {
			for _, output := range a.nodes[outputState].outputs {
				categoryIndex := output.categoryIndex
				if !matched[categoryIndex] || output.keywordIndex < bestMatches[categoryIndex].keywordIndex {
					matched[categoryIndex] = true
					bestMatches[categoryIndex] = output
				}
			}
		}
	}

	matches := make([]KeywordMatch, 0)
	for categoryIndex, found := range matched {
		if found {
			matches = append(matches, KeywordMatch{
				Category: a.categories[categoryIndex],
				Keyword:  bestMatches[categoryIndex].keyword,
			})
		}
	}
	return matches
}

func (a *Analyzer) internToken(word string) int {
	if tokenID, exists := a.tokenIDs[word]; exists {
		return tokenID
	}
	tokenID := len(a.tokenIDs) + 1
	a.tokenIDs[word] = tokenID
	return tokenID
}

func (a *Analyzer) buildFailureLinks() {
	queue := make([]int, 0, len(a.nodes)-1)
	for _, child := range a.nodes[0].next {
		queue = append(queue, child)
	}

	for head := 0; head < len(queue); head++ {
		state := queue[head]
		for tokenID, child := range a.nodes[state].next {
			fallback := a.nodes[state].fail
			for {
				if next, exists := a.nodes[fallback].next[tokenID]; exists {
					fallback = next
					break
				}
				if fallback == 0 {
					break
				}
				fallback = a.nodes[fallback].fail
			}

			a.nodes[child].fail = fallback
			if len(a.nodes[fallback].outputs) > 0 {
				a.nodes[child].output = fallback
			} else {
				a.nodes[child].output = a.nodes[fallback].output
			}
			queue = append(queue, child)
		}
	}
}

func keywordWords(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func (a *Analyzer) ContainsBlockedKeyword(query string) (string, bool) {
	matches := a.FindMatches(query)
	if len(matches) == 0 {
		return "", false
	}
	return matches[0].Keyword, true
}
