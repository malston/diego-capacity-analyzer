// ABOUTME: Tests for system prompt content requirements and composition function
// ABOUTME: Validates domain knowledge coverage, prompt budget, and context integration

package ai

import (
	"strings"
	"testing"
)

func TestSystemPromptContainsDomainKnowledge(t *testing.T) {
	requiredSections := []string{
		"<domain_knowledge>",
		"</domain_knowledge>",
		"<procurement_framing>",
		"</procurement_framing>",
		"<response_rules>",
		"</response_rules>",
		"<data_gap_handling>",
		"</data_gap_handling>",
	}
	for _, section := range requiredSections {
		if !strings.Contains(systemPrompt, section) {
			t.Errorf("system prompt missing required section tag: %s", section)
		}
	}
}

func TestSystemPromptContainsHeuristics(t *testing.T) {
	heuristics := []string{
		"N-1",
		"HA Admission Control",
		"vCPU:pCPU",
		"isolation segment",
		"cell sizing",
		"Diego",
	}
	lower := strings.ToLower(systemPrompt)
	for _, h := range heuristics {
		if !strings.Contains(lower, strings.ToLower(h)) {
			t.Errorf("system prompt missing required heuristic: %s", h)
		}
	}
}

func TestSystemPromptContainsProcurementFraming(t *testing.T) {
	terms := []string{
		"lead time",
		"budget",
		"procurement",
	}
	lower := strings.ToLower(systemPrompt)
	for _, term := range terms {
		if !strings.Contains(lower, term) {
			t.Errorf("system prompt missing procurement term: %s", term)
		}
	}
}

func TestSystemPromptContainsGapHandling(t *testing.T) {
	markers := []string{
		"NOT CONFIGURED",
		"UNAVAILABLE",
		"No scenario comparison has been run",
	}
	for _, marker := range markers {
		if !strings.Contains(systemPrompt, marker) {
			t.Errorf("system prompt missing data gap marker: %s", marker)
		}
	}
}

func TestSystemPromptContainsEvidenceRequirement(t *testing.T) {
	lower := strings.ToLower(systemPrompt)
	if !strings.Contains(lower, "cite") && !strings.Contains(lower, "reference") {
		t.Error("system prompt missing instruction to cite/reference data values from context")
	}
}

func TestSystemPromptTokenBudget(t *testing.T) {
	const maxChars = 10000
	if len(systemPrompt) > maxChars {
		t.Errorf("system prompt is %d chars (~%d tokens), exceeds budget of %d chars (~%d tokens)",
			len(systemPrompt), len(systemPrompt)/4, maxChars, maxChars/4)
	}
}

func TestBuildSystemPromptIncludesContext(t *testing.T) {
	ctx := "## Diego Cells\n**shared**: 6 cells, 196608 MB total"
	result := BuildSystemPrompt(ctx)

	if !strings.Contains(result, "<infrastructure_context>") {
		t.Error("composed prompt missing opening infrastructure_context tag")
	}
	if !strings.Contains(result, "</infrastructure_context>") {
		t.Error("composed prompt missing closing infrastructure_context tag")
	}
	if !strings.Contains(result, ctx) {
		t.Error("composed prompt missing context data")
	}
	if !strings.Contains(result, "N-1") {
		t.Error("composed prompt missing domain knowledge from static portion")
	}
}

func TestBuildSystemPromptEmptyContext(t *testing.T) {
	result := BuildSystemPrompt("")
	if !strings.Contains(result, "<infrastructure_context>") {
		t.Error("composed prompt missing infrastructure_context tag even with empty context")
	}
	if !strings.Contains(result, "<domain_knowledge>") {
		t.Error("composed prompt missing domain_knowledge section")
	}
}
