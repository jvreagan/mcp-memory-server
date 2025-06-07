// pkg/keywords/extractor.go
package keywords

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Extractor handles keyword extraction from text
type Extractor struct {
	// Common words to exclude from keyword extraction
	stopWords map[string]bool
	// Technical term patterns
	technicalPatterns []*regexp.Regexp
	// Project name patterns
	projectPatterns []*regexp.Regexp
	// Person name patterns
	personPatterns []*regexp.Regexp
}

// Keyword represents an extracted keyword with its score
type Keyword struct {
	Term  string
	Score float64
	Type  string // "technical", "project", "person", "concept"
}

// NewExtractor creates a new keyword extractor
func NewExtractor() *Extractor {
	extractor := &Extractor{
		stopWords: makeStopWords(),
		technicalPatterns: []*regexp.Regexp{
			// Programming languages
			regexp.MustCompile(`\b(golang|python|javascript|typescript|java|rust|cpp|c\+\+|ruby|php|swift|kotlin|scala)\b`),
			// Frameworks and libraries
			regexp.MustCompile(`\b(react|angular|vue|django|flask|spring|express|nextjs|rails|laravel)\b`),
			// Technologies
			regexp.MustCompile(`\b(docker|kubernetes|k8s|aws|gcp|azure|terraform|ansible|jenkins|gitlab|github)\b`),
			// Databases
			regexp.MustCompile(`\b(postgresql|postgres|mysql|mongodb|redis|elasticsearch|cassandra|dynamodb)\b`),
			// Technical concepts
			regexp.MustCompile(`\b(api|rest|graphql|grpc|microservice|serverless|ci\/cd|devops|agile|scrum)\b`),
			// File extensions and formats
			regexp.MustCompile(`\b\w+\.(go|py|js|ts|java|rs|cpp|rb|php|swift|kt|json|yaml|yml|xml|html|css|scss|sql)\b`),
		},
		projectPatterns: []*regexp.Regexp{
			// GitHub/GitLab style project names
			regexp.MustCompile(`\b[a-zA-Z0-9]+[-_][a-zA-Z0-9]+(?:[-_][a-zA-Z0-9]+)*\b`),
			// CamelCase project names
			regexp.MustCompile(`\b[A-Z][a-z]+(?:[A-Z][a-z]+)+\b`),
			// Package names with dots
			regexp.MustCompile(`\b[a-z]+(?:\.[a-z]+)+\b`),
		},
		personPatterns: []*regexp.Regexp{
			// Full names (First Last)
			regexp.MustCompile(`\b[A-Z][a-z]+\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)?\b`),
			// Email addresses (extract name part)
			regexp.MustCompile(`\b([a-zA-Z]+(?:[._-][a-zA-Z]+)*)@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`),
			// GitHub/GitLab usernames with @
			regexp.MustCompile(`@[a-zA-Z0-9][a-zA-Z0-9-]{0,38}`),
		},
	}
	
	return extractor
}

// Extract extracts keywords from the given text
func (e *Extractor) Extract(text string, maxKeywords int) []Keyword {
	if maxKeywords <= 0 {
		maxKeywords = 10
	}
	
	// Tokenize and count word frequencies
	words := e.tokenize(text)
	wordFreq := make(map[string]int)
	totalWords := 0
	
	for _, word := range words {
		if !e.isStopWord(word) && len(word) > 2 {
			wordFreq[strings.ToLower(word)]++
			totalWords++
		}
	}
	
	// Calculate TF scores
	tfScores := make(map[string]float64)
	for word, freq := range wordFreq {
		tfScores[word] = float64(freq) / float64(totalWords)
	}
	
	// Extract special terms
	technicalTerms := e.extractTechnicalTerms(text)
	projectNames := e.extractProjectNames(text)
	personNames := e.extractPersonNames(text)
	
	// Create keyword map with scores
	keywordMap := make(map[string]*Keyword)
	
	// Add technical terms with boost
	for term := range technicalTerms {
		termLower := strings.ToLower(term)
		score := tfScores[termLower] * 2.0 // Boost technical terms
		if score == 0 {
			score = 0.5 // Give a base score even if not in word frequency
		}
		keywordMap[termLower] = &Keyword{
			Term:  term,
			Score: score,
			Type:  "technical",
		}
	}
	
	// Add project names with boost
	for name := range projectNames {
		nameLower := strings.ToLower(name)
		score := tfScores[nameLower] * 1.8
		if score == 0 {
			score = 0.4
		}
		if existing, ok := keywordMap[nameLower]; !ok || existing.Score < score {
			keywordMap[nameLower] = &Keyword{
				Term:  name,
				Score: score,
				Type:  "project",
			}
		}
	}
	
	// Add person names with boost
	for name := range personNames {
		nameLower := strings.ToLower(name)
		score := tfScores[nameLower] * 1.5
		if score == 0 {
			score = 0.3
		}
		if existing, ok := keywordMap[nameLower]; !ok || existing.Score < score {
			keywordMap[nameLower] = &Keyword{
				Term:  name,
				Score: score,
				Type:  "person",
			}
		}
	}
	
	// Add high-frequency terms as concepts
	for word, tfScore := range tfScores {
		if _, exists := keywordMap[word]; !exists && tfScore > 0.01 {
			// Check if it's a meaningful concept (not just a common word)
			if e.isMeaningfulConcept(word) {
				keywordMap[word] = &Keyword{
					Term:  word,
					Score: tfScore,
					Type:  "concept",
				}
			}
		}
	}
	
	// Convert to slice and sort by score
	keywords := make([]Keyword, 0, len(keywordMap))
	for _, kw := range keywordMap {
		keywords = append(keywords, *kw)
	}
	
	sort.Slice(keywords, func(i, j int) bool {
		return keywords[i].Score > keywords[j].Score
	})
	
	// Return top keywords
	if len(keywords) > maxKeywords {
		keywords = keywords[:maxKeywords]
	}
	
	return keywords
}

// ExtractAll extracts all types of keywords without limit
func (e *Extractor) ExtractAll(text string) []Keyword {
	return e.Extract(text, 50) // Extract up to 50 keywords
}

// tokenize splits text into words
func (e *Extractor) tokenize(text string) []string {
	var words []string
	var currentWord strings.Builder
	
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}
	
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}
	
	return words
}

// isStopWord checks if a word is a stop word
func (e *Extractor) isStopWord(word string) bool {
	return e.stopWords[strings.ToLower(word)]
}

// isMeaningfulConcept checks if a word is a meaningful concept
func (e *Extractor) isMeaningfulConcept(word string) bool {
	// Check minimum length
	if len(word) < 4 {
		return false
	}
	
	// Check if it contains at least one vowel (filters out acronyms without meaning)
	hasVowel := false
	for _, r := range word {
		if strings.ContainsRune("aeiouAEIOU", r) {
			hasVowel = true
			break
		}
	}
	
	return hasVowel
}

// extractTechnicalTerms extracts technical terms from text
func (e *Extractor) extractTechnicalTerms(text string) map[string]bool {
	terms := make(map[string]bool)
	textLower := strings.ToLower(text)
	
	for _, pattern := range e.technicalPatterns {
		matches := pattern.FindAllString(textLower, -1)
		for _, match := range matches {
			terms[match] = true
		}
	}
	
	return terms
}

// extractProjectNames extracts project names from text
func (e *Extractor) extractProjectNames(text string) map[string]bool {
	names := make(map[string]bool)
	
	for _, pattern := range e.projectPatterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			// Filter out common false positives
			if !e.isCommonWord(match) && len(match) > 3 {
				names[match] = true
			}
		}
	}
	
	return names
}

// extractPersonNames extracts person names from text
func (e *Extractor) extractPersonNames(text string) map[string]bool {
	names := make(map[string]bool)
	
	for _, pattern := range e.personPatterns {
		if pattern.String() == `\b([a-zA-Z]+(?:[._-][a-zA-Z]+)*)@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b` {
			// Special handling for email addresses
			matches := pattern.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) > 1 {
					name := strings.ReplaceAll(match[1], ".", " ")
					name = strings.ReplaceAll(name, "_", " ")
					name = strings.ReplaceAll(name, "-", " ")
					names[strings.Title(name)] = true
				}
			}
		} else {
			matches := pattern.FindAllString(text, -1)
			for _, match := range matches {
				names[match] = true
			}
		}
	}
	
	return names
}

// isCommonWord checks if a word is too common to be a project name
func (e *Extractor) isCommonWord(word string) bool {
	common := []string{
		"the", "and", "for", "with", "from", "this", "that", "what",
		"when", "where", "which", "while", "about", "after", "before",
		"between", "during", "under", "over", "through", "into",
	}
	
	wordLower := strings.ToLower(word)
	for _, c := range common {
		if wordLower == c {
			return true
		}
	}
	
	return false
}

// makeStopWords creates the stop words set
func makeStopWords() map[string]bool {
	words := []string{
		// Articles
		"a", "an", "the",
		// Pronouns
		"i", "me", "my", "myself", "we", "our", "ours", "ourselves",
		"you", "your", "yours", "yourself", "yourselves",
		"he", "him", "his", "himself", "she", "her", "hers", "herself",
		"it", "its", "itself", "they", "them", "their", "theirs", "themselves",
		"what", "which", "who", "whom", "this", "that", "these", "those",
		// Prepositions
		"at", "by", "for", "from", "in", "of", "on", "to", "with",
		"about", "against", "between", "into", "through", "during",
		"before", "after", "above", "below", "up", "down", "out", "off",
		"over", "under", "again", "further", "then", "once",
		// Conjunctions
		"and", "but", "or", "nor", "so", "yet", "both", "either", "neither",
		// Auxiliary verbs
		"am", "is", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "having", "do", "does", "did", "doing",
		"will", "would", "shall", "should", "may", "might", "must",
		"can", "could", "ought",
		// Common verbs
		"get", "got", "gets", "getting", "make", "made", "making",
		"go", "goes", "went", "going", "take", "takes", "took", "taking",
		"come", "comes", "came", "coming", "want", "wants", "wanted",
		"use", "uses", "used", "using", "find", "finds", "found",
		"give", "gives", "gave", "giving", "tell", "tells", "told",
		"work", "works", "worked", "working", "call", "calls", "called",
		"try", "tries", "tried", "trying", "need", "needs", "needed",
		"feel", "feels", "felt", "feeling", "become", "becomes", "became",
		"leave", "leaves", "left", "leaving", "put", "puts", "putting",
		"mean", "means", "meant", "meaning", "keep", "keeps", "kept",
		"let", "lets", "letting", "begin", "begins", "began", "beginning",
		"seem", "seems", "seemed", "seeming", "help", "helps", "helped",
		"show", "shows", "showed", "showing", "hear", "hears", "heard",
		"play", "plays", "played", "playing", "run", "runs", "ran",
		"move", "moves", "moved", "moving", "live", "lives", "lived",
		"believe", "believes", "believed", "believing",
		// Other common words
		"here", "there", "when", "where", "why", "how", "all", "many",
		"some", "few", "more", "most", "other", "such", "no", "not",
		"only", "own", "same", "than", "too", "very", "just", "now",
		"also", "well", "even", "back", "still", "way", "because",
		"however", "around", "yet", "since", "while", "whether",
	}
	
	stopWords := make(map[string]bool)
	for _, word := range words {
		stopWords[word] = true
	}
	
	return stopWords
}

// CalculateTFIDF calculates TF-IDF scores for keywords across multiple documents
func CalculateTFIDF(documents []string, maxKeywordsPerDoc int) map[string]map[string]float64 {
	extractor := NewExtractor()
	
	// Document frequency map
	docFreq := make(map[string]int)
	// Term frequency per document
	docTermFreq := make(map[int]map[string]int)
	
	// First pass: collect term frequencies
	for i, doc := range documents {
		words := extractor.tokenize(doc)
		termFreq := make(map[string]int)
		seenTerms := make(map[string]bool)
		
		for _, word := range words {
			wordLower := strings.ToLower(word)
			if !extractor.isStopWord(wordLower) && len(word) > 2 {
				termFreq[wordLower]++
				if !seenTerms[wordLower] {
					docFreq[wordLower]++
					seenTerms[wordLower] = true
				}
			}
		}
		
		docTermFreq[i] = termFreq
	}
	
	// Calculate TF-IDF scores
	numDocs := float64(len(documents))
	tfidfScores := make(map[string]map[string]float64)
	
	for i, termFreq := range docTermFreq {
		scores := make(map[string]float64)
		totalTerms := 0
		for _, freq := range termFreq {
			totalTerms += freq
		}
		
		for term, freq := range termFreq {
			tf := float64(freq) / float64(totalTerms)
			idf := math.Log(numDocs / float64(docFreq[term]))
			scores[term] = tf * idf
		}
		
		tfidfScores[documents[i]] = scores
	}
	
	return tfidfScores
}