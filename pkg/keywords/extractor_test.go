// pkg/keywords/extractor_test.go
package keywords

import (
	"strings"
	"testing"
)

func TestExtractor_Extract(t *testing.T) {
	extractor := NewExtractor()
	
	tests := []struct {
		name         string
		text         string
		maxKeywords  int
		wantKeywords []string // Keywords we expect to find (not exhaustive)
		wantTypes    map[string]string // Expected types for specific keywords
	}{
		{
			name: "Technical content",
			text: "We're using Golang and PostgreSQL for the backend API. The frontend is built with React and TypeScript. Everything runs on Docker containers in AWS.",
			maxKeywords: 10,
			wantKeywords: []string{"golang", "postgresql", "react", "typescript", "docker", "aws"},
			wantTypes: map[string]string{
				"golang":     "technical",
				"postgresql": "technical",
				"docker":     "technical",
			},
		},
		{
			name: "Project names",
			text: "The mcp-memory-server project integrates with MyAwesomeApp and handles data from the user-dashboard component.",
			maxKeywords: 10,
			wantKeywords: []string{"mcp-memory-server", "MyAwesomeApp", "user-dashboard"},
			wantTypes: map[string]string{
				"mcp-memory-server": "project",
				"MyAwesomeApp":      "project",
			},
		},
		{
			name: "Person names",
			text: "John Smith reviewed the PR. Sarah Johnson from the DevOps team helped with deployment. Contact: john.smith@example.com",
			maxKeywords: 10,
			wantKeywords: []string{"John Smith", "Sarah Johnson"},
			wantTypes: map[string]string{
				"John Smith":    "person",
				"Sarah Johnson": "person",
			},
		},
		{
			name: "Mixed content",
			text: "Alice Cooper implemented the Python script for data-processor using Django framework. The script connects to MongoDB and runs on kubernetes cluster.",
			maxKeywords: 10,
			wantKeywords: []string{"Alice Cooper", "python", "data-processor", "django", "mongodb", "kubernetes"},
			wantTypes: map[string]string{
				"Alice Cooper":   "person",
				"python":         "technical",
				"data-processor": "project",
			},
		},
		{
			name: "File paths and extensions",
			text: "Edit the config.yaml file and update main.go. The styles.scss needs refactoring. Check database.sql for schema.",
			maxKeywords: 10,
			wantKeywords: []string{"config.yaml", "main.go", "styles.scss", "database.sql"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := extractor.Extract(tt.text, tt.maxKeywords)
			
			// Check that we got some keywords
			if len(keywords) == 0 {
				t.Errorf("Expected keywords but got none")
			}
			
			// Check that specific expected keywords are present
			foundKeywords := make(map[string]bool)
			keywordTypes := make(map[string]string)
			for _, kw := range keywords {
				foundKeywords[strings.ToLower(kw.Term)] = true
				keywordTypes[strings.ToLower(kw.Term)] = kw.Type
			}
			
			for _, want := range tt.wantKeywords {
				if !foundKeywords[strings.ToLower(want)] {
					t.Errorf("Expected keyword %q not found in results", want)
				}
			}
			
			// Check types if specified
			for keyword, expectedType := range tt.wantTypes {
				if actualType, ok := keywordTypes[strings.ToLower(keyword)]; ok {
					if actualType != expectedType {
						t.Errorf("Keyword %q has type %q, want %q", keyword, actualType, expectedType)
					}
				}
			}
			
			// Check that we respect the max keywords limit
			if len(keywords) > tt.maxKeywords {
				t.Errorf("Got %d keywords, want at most %d", len(keywords), tt.maxKeywords)
			}
		})
	}
}

func TestExtractor_StopWords(t *testing.T) {
	extractor := NewExtractor()
	
	text := "The quick brown fox jumps over the lazy dog and runs through the forest"
	keywords := extractor.Extract(text, 10)
	
	// Check that stop words are not included
	stopWords := []string{"the", "and", "over", "through"}
	for _, kw := range keywords {
		for _, stopWord := range stopWords {
			if strings.EqualFold(kw.Term, stopWord) {
				t.Errorf("Stop word %q should not be in keywords", stopWord)
			}
		}
	}
}

func TestExtractor_Scoring(t *testing.T) {
	extractor := NewExtractor()
	
	// Text with repeated technical terms
	text := "Kubernetes is great. We use kubernetes for deployment. Our kubernetes cluster has many nodes. Docker containers run in the kubernetes environment."
	
	keywords := extractor.Extract(text, 5)
	
	// kubernetes should be the top keyword due to frequency and being technical
	if len(keywords) > 0 && !strings.EqualFold(keywords[0].Term, "kubernetes") {
		t.Errorf("Expected 'kubernetes' to be the top keyword, got %q", keywords[0].Term)
	}
	
	// Check that scores are in descending order
	for i := 1; i < len(keywords); i++ {
		if keywords[i].Score > keywords[i-1].Score {
			t.Errorf("Keywords not properly sorted by score")
		}
	}
}

func TestCalculateTFIDF(t *testing.T) {
	documents := []string{
		"golang is a great programming language for building APIs",
		"python is popular for machine learning and data science", 
		"golang and python are both used for backend development",
		"machine learning models can be deployed as APIs",
	}
	
	tfidfScores := CalculateTFIDF(documents, 5)
	
	// Check that we got scores for all documents
	if len(tfidfScores) != len(documents) {
		t.Errorf("Expected scores for %d documents, got %d", len(documents), len(tfidfScores))
	}
	
	// Check that unique terms have higher scores
	// "golang" appears in docs 0 and 2, "machine" appears in docs 1 and 3
	// Terms that appear in fewer documents should have higher IDF scores
	doc0Scores := tfidfScores[documents[0]]
	doc1Scores := tfidfScores[documents[1]]
	
	if doc0Scores["golang"] <= 0 {
		t.Errorf("Expected positive TF-IDF score for 'golang' in document 0")
	}
	
	if doc1Scores["python"] <= 0 {
		t.Errorf("Expected positive TF-IDF score for 'python' in document 1")
	}
}