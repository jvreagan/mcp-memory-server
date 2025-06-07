// examples/keyword_extraction_demo.go
package main

import (
	"fmt"
	"mcp-memory-server/pkg/keywords"
)

func main() {
	extractor := keywords.NewExtractor()
	
	// Example 1: Technical content
	fmt.Println("=== Example 1: Technical Content ===")
	techText := `We're migrating our microservices from Docker to Kubernetes. 
	The API is built with Golang and uses PostgreSQL for data persistence. 
	Frontend uses React with TypeScript. CI/CD pipeline runs on Jenkins.`
	
	techKeywords := extractor.Extract(techText, 10)
	printKeywords(techKeywords)
	
	// Example 2: Project documentation
	fmt.Println("\n=== Example 2: Project Documentation ===")
	projectText := `The mcp-memory-server project was reviewed by John Smith and Sarah Chen. 
	It integrates with user-dashboard and analytics-engine components. 
	Contact alice.jones@company.com for deployment questions.`
	
	projectKeywords := extractor.Extract(projectText, 10)
	printKeywords(projectKeywords)
	
	// Example 3: Mixed technical discussion
	fmt.Println("\n=== Example 3: Mixed Technical Discussion ===")
	mixedText := `Bob Wilson implemented the new feature in main.go using the Django REST framework. 
	The config.yaml file needs updating before deployment to AWS. 
	The database.sql script creates tables in MongoDB for the DataProcessor service.`
	
	mixedKeywords := extractor.Extract(mixedText, 10)
	printKeywords(mixedKeywords)
	
	// Example 4: Demonstrating TF-IDF across multiple documents
	fmt.Println("\n=== Example 4: TF-IDF Analysis ===")
	documents := []string{
		"Python is great for machine learning and data science applications",
		"Golang excels at building high-performance microservices and APIs",
		"Both Python and Golang are popular choices for backend development",
		"Machine learning models can be deployed as microservices using Docker",
	}
	
	tfidfScores := keywords.CalculateTFIDF(documents, 5)
	
	for i, doc := range documents {
		fmt.Printf("\nDocument %d: %s\n", i+1, doc[:50]+"...")
		fmt.Println("Top keywords by TF-IDF:")
		
		// Get top keywords for this document
		type scoredTerm struct {
			term  string
			score float64
		}
		var terms []scoredTerm
		for term, score := range tfidfScores[doc] {
			terms = append(terms, scoredTerm{term, score})
		}
		
		// Sort by score
		for i := 0; i < len(terms); i++ {
			for j := i + 1; j < len(terms); j++ {
				if terms[j].score > terms[i].score {
					terms[i], terms[j] = terms[j], terms[i]
				}
			}
		}
		
		// Print top 5
		for i := 0; i < 5 && i < len(terms); i++ {
			fmt.Printf("  - %s (score: %.3f)\n", terms[i].term, terms[i].score)
		}
	}
}

func printKeywords(keywords []keywords.Keyword) {
	fmt.Println("Extracted keywords:")
	for i, kw := range keywords {
		fmt.Printf("%d. %s (type: %s, score: %.3f)\n", i+1, kw.Term, kw.Type, kw.Score)
	}
}