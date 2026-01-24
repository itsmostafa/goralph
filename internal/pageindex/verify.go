package pageindex

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Verifier handles verification and correction of TOC entries.
type Verifier struct {
	llm    LLMProvider
	config *Config
}

// NewVerifier creates a new Verifier with the given LLM provider.
func NewVerifier(llm LLMProvider, config *Config) *Verifier {
	if config == nil {
		config = DefaultConfig()
	}
	return &Verifier{llm: llm, config: config}
}

// VerificationReport contains the results of verifying a TOC.
type VerificationReport struct {
	TotalEntries     int
	CorrectEntries   int
	IncorrectEntries []int // Indices of incorrect entries
	Accuracy         float64
}

// VerifyTOCEntries verifies that TOC entries appear on their expected pages.
// Returns a report with accuracy and list of incorrect entries.
func (v *Verifier) VerifyTOCEntries(ctx context.Context, items []TOCItem, pages []PageContent, maxWorkers int) (*VerificationReport, error) {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	// Filter items with physical indices
	var toVerify []int
	for i, item := range items {
		if item.PhysicalIndex != nil && *item.PhysicalIndex >= 0 && *item.PhysicalIndex < len(pages) {
			toVerify = append(toVerify, i)
		}
	}

	if len(toVerify) == 0 {
		return &VerificationReport{
			TotalEntries:   len(items),
			CorrectEntries: 0,
			Accuracy:       0,
		}, nil
	}

	type result struct {
		index   int
		correct bool
		err     error
	}

	results := make(chan result, len(toVerify))

	// Create worker pool
	jobs := make(chan int, len(toVerify))

	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				item := items[idx]
				pageIdx := *item.PhysicalIndex
				correct, err := v.verifyEntry(ctx, item.Title, pageIdx, pages[pageIdx].Text)
				results <- result{index: idx, correct: correct, err: err}
			}
		}()
	}

	// Queue jobs
	for _, idx := range toVerify {
		jobs <- idx
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	correctCount := 0
	var incorrect []int

	for r := range results {
		if r.err != nil {
			// Treat errors as unverified (not incorrect)
			continue
		}
		if r.correct {
			correctCount++
		} else {
			incorrect = append(incorrect, r.index)
		}
	}

	accuracy := float64(correctCount) / float64(len(toVerify))

	return &VerificationReport{
		TotalEntries:     len(items),
		CorrectEntries:   correctCount,
		IncorrectEntries: incorrect,
		Accuracy:         accuracy,
	}, nil
}

// verifyEntry checks if a title appears on the expected page.
func (v *Verifier) verifyEntry(ctx context.Context, title string, pageNum int, pageText string) (bool, error) {
	// First, try simple string matching with fuzzy comparison
	if containsFuzzy(pageText, title) {
		return true, nil
	}

	// Fall back to LLM verification
	prompt := fmt.Sprintf(VerifyTOCEntryPrompt, title, pageNum, truncateForPrompt(pageText, 3000))

	response, err := v.llm.Complete(ctx, prompt)
	if err != nil {
		return false, err
	}

	result, err := ExtractJSON[LLMResponse](response)
	if err != nil {
		return false, err
	}

	return strings.ToLower(result.Answer) == "yes", nil
}

// containsFuzzy checks if the page contains the title with fuzzy matching.
// Handles common OCR and formatting issues like extra spaces.
func containsFuzzy(pageText, title string) bool {
	// Normalize both strings
	normalizeStr := func(s string) string {
		// Collapse multiple spaces
		s = strings.Join(strings.Fields(s), " ")
		// Lowercase
		s = strings.ToLower(s)
		return s
	}

	pageNorm := normalizeStr(pageText)
	titleNorm := normalizeStr(title)

	return strings.Contains(pageNorm, titleNorm)
}

// FixIncorrectEntries attempts to fix incorrect TOC entries by searching nearby pages.
func (v *Verifier) FixIncorrectEntries(ctx context.Context, items []TOCItem, pages []PageContent, incorrectIndices []int, searchRadius int) ([]TOCItem, error) {
	if searchRadius <= 0 {
		searchRadius = 5
	}

	result := make([]TOCItem, len(items))
	copy(result, items)

	for _, idx := range incorrectIndices {
		item := items[idx]
		if item.PhysicalIndex == nil {
			continue
		}

		originalPage := *item.PhysicalIndex

		// Search nearby pages
		found := false
		for offset := -searchRadius; offset <= searchRadius && !found; offset++ {
			if offset == 0 {
				continue // Already checked this page
			}

			searchPage := originalPage + offset
			if searchPage < 0 || searchPage >= len(pages) {
				continue
			}

			if containsFuzzy(pages[searchPage].Text, item.Title) {
				result[idx].PhysicalIndex = &searchPage
				found = true
			}
		}

		if !found {
			// Use LLM to search if simple matching fails
			correctedPage, err := v.searchForEntry(ctx, item.Title, pages, originalPage, searchRadius)
			if err == nil && correctedPage >= 0 {
				result[idx].PhysicalIndex = &correctedPage
			}
		}
	}

	return result, nil
}

// searchForEntry uses LLM to find where a title actually appears.
func (v *Verifier) searchForEntry(ctx context.Context, title string, pages []PageContent, centerPage, radius int) (int, error) {
	// Build context of pages around the expected location
	var pagesContext strings.Builder
	startPage := centerPage - radius
	if startPage < 0 {
		startPage = 0
	}
	endPage := centerPage + radius
	if endPage >= len(pages) {
		endPage = len(pages) - 1
	}

	for i := startPage; i <= endPage; i++ {
		pagesContext.WriteString(fmt.Sprintf("\n=== Page %d ===\n%s\n", i, truncateForPrompt(pages[i].Text, 500)))
	}

	prompt := fmt.Sprintf(`You are an expert in analyzing documents. You are looking for where a section title appears in a document.

Section Title: %s

Pages to search:
%s

Which page number does this section title appear on? If not found, respond with -1.

Respond in JSON format:
{
  "thinking": "<your reasoning>",
  "page_number": <page number or -1>
}`, title, pagesContext.String())

	response, err := v.llm.Complete(ctx, prompt)
	if err != nil {
		return -1, err
	}

	type pageResult struct {
		PageNumber int `json:"page_number"`
	}

	result, err := ExtractJSON[pageResult](response)
	if err != nil {
		return -1, err
	}

	return result.PageNumber, nil
}

// FixIncorrectTOCWithRetries iteratively fixes incorrect entries with multiple attempts.
func (v *Verifier) FixIncorrectTOCWithRetries(ctx context.Context, items []TOCItem, pages []PageContent, maxRetries int) ([]TOCItem, *VerificationReport, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	current := make([]TOCItem, len(items))
	copy(current, items)

	var lastReport *VerificationReport

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Verify current state
		report, err := v.VerifyTOCEntries(ctx, current, pages, 10)
		if err != nil {
			return current, report, err
		}

		lastReport = report

		// Check if we've reached acceptable accuracy
		if report.Accuracy >= 1.0 {
			return current, report, nil
		}

		if len(report.IncorrectEntries) == 0 {
			return current, report, nil
		}

		// Attempt to fix incorrect entries
		// Increase search radius with each attempt
		searchRadius := 5 + attempt*3

		fixed, err := v.FixIncorrectEntries(ctx, current, pages, report.IncorrectEntries, searchRadius)
		if err != nil {
			return current, report, err
		}

		current = fixed
	}

	return current, lastReport, nil
}

// DetermineProcessingMode decides which TOC processing mode to use.
// Returns: "toc_with_pages", "toc_without_pages", or "no_toc"
func DetermineProcessingMode(tocResult *TOCCheckResult) string {
	if tocResult == nil || len(tocResult.TOCPageList) == 0 {
		return "no_toc"
	}

	if tocResult.PageIndexGivenTOC {
		return "toc_with_pages"
	}

	return "toc_without_pages"
}

// VerificationThreshold defines the accuracy thresholds for TOC verification.
type VerificationThreshold struct {
	Complete    float64 // Accuracy for considering complete (default 1.0)
	Fixable     float64 // Minimum accuracy to attempt fixing (default 0.6)
	FallbackMin float64 // Below this, fallback to generation (default 0.6)
}

// DefaultVerificationThreshold returns the default threshold values.
func DefaultVerificationThreshold() *VerificationThreshold {
	return &VerificationThreshold{
		Complete:    1.0,
		Fixable:     0.6,
		FallbackMin: 0.6,
	}
}

// EvaluateVerification determines the next action based on verification results.
// Returns: "complete", "fix", or "fallback"
func EvaluateVerification(report *VerificationReport, threshold *VerificationThreshold) string {
	if threshold == nil {
		threshold = DefaultVerificationThreshold()
	}

	if report.Accuracy >= threshold.Complete {
		return "complete"
	}

	if report.Accuracy >= threshold.Fixable {
		return "fix"
	}

	return "fallback"
}
