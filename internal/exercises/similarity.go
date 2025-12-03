package exercises

import (
	"strings"
)

// Similarity calculates similarity between two strings (0.0 - 1.0)
// Uses simple token-based Jaccard similarity
func Similarity(s1, s2 string) float64 {
	// Normalize
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))
	
	if s1 == s2 {
		return 1.0
	}
	
	// Tokenize
	tokens1 := tokenize(s1)
	tokens2 := tokenize(s2)
	
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}
	
	// Jaccard similarity: intersection / union
	intersection := 0
	union := make(map[string]bool)
	
	for token := range tokens1 {
		union[token] = true
	}
	for token := range tokens2 {
		union[token] = true
	}
	
	for token := range tokens1 {
		if tokens2[token] {
			intersection++
		}
	}
	
	return float64(intersection) / float64(len(union))
}

// tokenize splits string into word tokens
func tokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	words := strings.Fields(s)
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 0 {
			tokens[word] = true
		}
	}
	return tokens
}

// FindSimilarExercises finds exercises with similar names
func FindSimilarExercises(name string, exercises []*Exercise, threshold float64) []*Exercise {
	var similar []*Exercise
	
	for _, ex := range exercises {
		similarity := Similarity(name, ex.Name)
		if similarity >= threshold {
			similar = append(similar, ex)
		}
	}
	
	return similar
}
