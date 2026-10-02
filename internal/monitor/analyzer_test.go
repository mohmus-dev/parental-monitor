package monitor

import (
	"strconv"
	"testing"
)

func TestContainsBlockedKeyword(t *testing.T) {
	analyzer := NewAnalyzer([]string{"", " porn ", "adult"})

	keyword, matched := analyzer.ContainsBlockedKeyword("Searching for PORN filters")
	if !matched || keyword != "porn" {
		t.Fatalf("expected case-insensitive porn match, got keyword %q, matched %t", keyword, matched)
	}

	if keyword, matched := analyzer.ContainsBlockedKeyword("ordinary homework search"); matched {
		t.Fatalf("unexpected match for keyword %q", keyword)
	}
}

func TestFindMatchesReturnsEachCategoryAndPreservesQuery(t *testing.T) {
	analyzer := NewGroupedAnalyzer(
		[]string{"drugs", "beaten"},
		map[string][]string{
			"drugs":    {"drug", "drugs", "morphine"},
			"violence": {"beaten", "violence"},
		},
	)
	query := "wwe drugs beaten"
	matches := analyzer.FindMatches(query)

	if len(matches) != 2 {
		t.Fatalf("expected two category matches, got %#v", matches)
	}
	if matches[0] != (KeywordMatch{Category: "drugs", Keyword: "drugs"}) {
		t.Fatalf("unexpected first match: %#v", matches[0])
	}
	if matches[1] != (KeywordMatch{Category: "violence", Keyword: "beaten"}) {
		t.Fatalf("unexpected second match: %#v", matches[1])
	}

	matches = analyzer.FindMatches("morphine for pain")
	if len(matches) != 1 || matches[0].Category != "drugs" || matches[0].Keyword != "morphine" {
		t.Fatalf("expected morphine to map to drugs, got %#v", matches)
	}
}

func TestMergeKeywordGroupsAddsCustomTermsWithoutReplacingDefaults(t *testing.T) {
	merged := MergeKeywordGroups(
		map[string][]string{"drugs": {"morphine", "cocaine"}},
		map[string][]string{"drugs": {"custom medicine", "morphine"}, "school": {"custom phrase"}},
	)
	analyzer := NewGroupedAnalyzer(nil, merged)

	matches := analyzer.FindMatches("morphine and custom medicine")
	if len(matches) != 1 || matches[0].Category != "drugs" || matches[0].Keyword != "morphine" {
		t.Fatalf("expected default drug term to remain available, got %#v", matches)
	}
	matches = analyzer.FindMatches("custom phrase")
	if len(matches) != 1 || matches[0].Category != "school" {
		t.Fatalf("expected custom category to be added, got %#v", matches)
	}
}

func TestFindMatchesHandlesOverlappingPhrasesAndFailureLinks(t *testing.T) {
	analyzer := NewGroupedAnalyzer(nil, map[string][]string{
		"long phrase":   {"alpha beta gamma"},
		"suffix phrase": {"beta gamma"},
		"fallback":      {"gamma delta"},
	})

	matches := analyzer.FindMatches("alpha beta gamma delta")
	if len(matches) != 3 {
		t.Fatalf("expected all overlapping phrase matches, got %#v", matches)
	}
	if matches[0].Category != "fallback" || matches[1].Category != "long phrase" || matches[2].Category != "suffix phrase" {
		t.Fatalf("unexpected category order or missing fallback match: %#v", matches)
	}
}

func BenchmarkFindMatchesLargeDictionary(b *testing.B) {
	for _, size := range []int{10_000, 100_000, 1_000_000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			terms := make([]string, size)
			for i := range terms {
				terms[i] = "term" + strconv.Itoa(i)
			}
			analyzer := NewGroupedAnalyzer(nil, map[string][]string{"large": terms})
			query := "unrelated " + terms[size-1]

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				analyzer.FindMatches(query)
			}
		})
	}
}
