package slm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// CategorizationResult is the parsed result of SLM categorization.
type CategorizationResult struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// Categorizer uses the SLM to classify tasks into categories.
type Categorizer struct {
	client *Client
}

// NewCategorizer creates a new Categorizer.
func NewCategorizer(client *Client) *Categorizer {
	return &Categorizer{client: client}
}

// Categorize classifies a task into one of the provided categories.
func (c *Categorizer) Categorize(ctx context.Context, title, description string, categories []string) (*CategorizationResult, error) {
	catList := strings.Join(categories, ", ")
	prompt := fmt.Sprintf("Available categories: %s\n\nTask title: %s\nTask description: %s", catList, title, description)

	response, err := c.client.Generate(ctx, CategorizationSystemPrompt, prompt)
	if err != nil {
		log.Printf("SLM categorization failed: %v", err)
		return nil, fmt.Errorf("SLM categorization failed: %w", err)
	}

	var result CategorizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("SLM returned unparseable categorization response: %s", response)
		return nil, fmt.Errorf("unparseable SLM response: %w", err)
	}

	return &result, nil
}
