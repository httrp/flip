package commands

import (
	"testing"

	"github.com/httrp/flip/internal/ai"
)

// ---------------------------------------------------------------------------
// ollamaListContainsModel
// ---------------------------------------------------------------------------

func TestOllamaListContainsModel_ExactMatch(t *testing.T) {
	output := "NAME            ID              SIZE    MODIFIED\nllama3.2        abc123          2.0 GB  2 hours ago\n"
	if !ollamaListContainsModel(output, "llama3.2") {
		t.Error("should find exact model name 'llama3.2'")
	}
}

func TestOllamaListContainsModel_WithTag(t *testing.T) {
	// Ollama often stores models as "name:latest"
	output := "NAME                ID              SIZE    MODIFIED\nllama3.2:latest     abc123          2.0 GB  2 hours ago\n"
	if !ollamaListContainsModel(output, "llama3.2") {
		t.Error("should find model 'llama3.2' when stored as 'llama3.2:latest'")
	}
}

func TestOllamaListContainsModel_CaseInsensitive(t *testing.T) {
	output := "NAME            ID\nLlama3.2:latest abc\n"
	if !ollamaListContainsModel(output, "llama3.2") {
		t.Error("matching should be case-insensitive")
	}
}

func TestOllamaListContainsModel_NotPresent(t *testing.T) {
	output := "NAME            ID\nllama3.2:latest abc\n"
	if ollamaListContainsModel(output, "mistral") {
		t.Error("'mistral' should not be found in output that only has llama3.2")
	}
}

func TestOllamaListContainsModel_EmptyOutput(t *testing.T) {
	if ollamaListContainsModel("", "llama3.2") {
		t.Error("empty output should return false")
	}
}

func TestOllamaListContainsModel_EmptyModel(t *testing.T) {
	output := "NAME\nllama3.2:latest\n"
	if ollamaListContainsModel(output, "") {
		t.Error("empty model name should return false")
	}
}

func TestOllamaListContainsModel_HeaderLineSkipped(t *testing.T) {
	// The header line "NAME" should not match a model named "name"
	output := "NAME    ID\n"
	if ollamaListContainsModel(output, "name") {
		t.Error("header line should be skipped")
	}
}

func TestOllamaListContainsModel_MultipleModels(t *testing.T) {
	output := "NAME              ID\nllama3.2:latest   abc\nmistral:latest    def\ngemma3:latest     ghi\n"

	cases := []struct {
		model string
		want  bool
	}{
		{"llama3.2", true},
		{"mistral", true},
		{"gemma3", true},
		{"phi3", false},
	}
	for _, tc := range cases {
		got := ollamaListContainsModel(output, tc.model)
		if got != tc.want {
			t.Errorf("ollamaListContainsModel(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestOllamaListContainsModel_PrefixNotSuffix(t *testing.T) {
	// "llama" should not match "llama3.2"
	output := "NAME\nllama3.2:latest\n"
	if ollamaListContainsModel(output, "llama") {
		t.Error("'llama' should not match 'llama3.2:latest'")
	}
}

// ---------------------------------------------------------------------------
// firstModelID
// ---------------------------------------------------------------------------

func TestFirstModelID_ReturnsFirstID(t *testing.T) {
	models := []ai.ModelInfo{
		{ID: "llama3.2:latest", Name: "Llama 3.2"},
		{ID: "mistral:latest", Name: "Mistral"},
	}
	got := firstModelID(models)
	if got != "llama3.2:latest" {
		t.Errorf("expected 'llama3.2:latest', got %q", got)
	}
}

func TestFirstModelID_EmptySlice(t *testing.T) {
	got := firstModelID([]ai.ModelInfo{})
	if got != "" {
		t.Errorf("empty slice should return empty string, got %q", got)
	}
}

func TestFirstModelID_SingleModel(t *testing.T) {
	models := []ai.ModelInfo{{ID: "gemma3:latest"}}
	got := firstModelID(models)
	if got != "gemma3:latest" {
		t.Errorf("expected 'gemma3:latest', got %q", got)
	}
}
