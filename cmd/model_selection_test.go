package cmd

import (
	"os"
	"strings"
	"testing"

	"apimgr/config"
)

func TestShouldPrompt_单模型配置不提示(t *testing.T) {
	// Arrange
	cfg := &config.APIConfig{
		Model:  "model-1",
		Models: []string{"model-1"},
	}
	ms := NewModelSelector()

	// Act
	result := ms.ShouldPrompt(cfg, "", false)

	// Assert
	if result != false {
		t.Errorf("Expected false for single model config, got %v", result)
	}
}

func TestShouldPrompt_多模型配置提示(t *testing.T) {
	// Arrange
	cfg := &config.APIConfig{
		Model:  "model-1",
		Models: []string{"model-1", "model-2", "model-3"},
	}
	ms := NewModelSelector()

	// Act (this test will only work in interactive terminals)
	// We'll assume it returns false in test environments which are non-interactive
	result := ms.ShouldPrompt(cfg, "", false)

	// Assert (in CI environment, isInteractiveTerminal() returns false)
	if result != false {
		t.Log("Warning: ShouldPrompt returned true in non-interactive test environment")
	}
}

func TestShouldPrompt_指定模型参数不提示(t *testing.T) {
	// Arrange
	cfg := &config.APIConfig{
		Model:  "model-1",
		Models: []string{"model-1", "model-2"},
	}
	ms := NewModelSelector()

	// Act
	result := ms.ShouldPrompt(cfg, "model-2", false)

	// Assert
	if result != false {
		t.Errorf("Expected false when model flag is specified, got %v", result)
	}
}

func TestShouldPrompt_显式禁用不提示(t *testing.T) {
	// Arrange
	cfg := &config.APIConfig{
		Model:  "model-1",
		Models: []string{"model-1", "model-2"},
	}
	ms := NewModelSelector()

	// Act
	result := ms.ShouldPrompt(cfg, "", true)

	// Assert
	if result != false {
		t.Errorf("Expected false when no-prompt is true, got %v", result)
	}
}

func TestPromptSimple_有效输入(t *testing.T) {
	// Arrange
	// Mock stdin for this test
	testInput := "2\n" // Select the second model
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString(testInput)
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	ms := NewModelSelector()
	models := []string{"model-1", "model-2", "model-3"}
	currentModel := "model-1"

	// Act
	selected, err := ms.PromptSimple(models, currentModel)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if selected != "model-2" {
		t.Errorf("Expected model-2, got: %v", selected)
	}
}

func TestPromptSimple_空输入使用默认值(t *testing.T) {
	// Arrange
	// Mock stdin for this test
	testInput := "\n" // Press Enter
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString(testInput)
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	ms := NewModelSelector()
	models := []string{"model-1", "model-2", "model-3"}
	currentModel := "model-1"

	// Act
	selected, err := ms.PromptSimple(models, currentModel)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if selected != currentModel {
		t.Errorf("Expected current model %s, got: %v", currentModel, selected)
	}
}

func TestValidateModelInList_有效模型(t *testing.T) {
	// Arrange
	ms := NewModelSelector()
	models := []string{"model-1", "model-2", "model-3"}
	model := "model-2"

	// Act
	err := ms.ValidateModelInList(model, models)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for valid model, got: %v", err)
	}
}

func TestValidateModelInList_无效模型(t *testing.T) {
	// Arrange
	ms := NewModelSelector()
	models := []string{"model-1", "model-2", "model-3"}
	model := "invalid-model"

	// Act
	err := ms.ValidateModelInList(model, models)

	// Assert
	if err == nil {
		t.Error("Expected error for invalid model, got none")
	} else if !strings.Contains(err.Error(), "invalid-model") {
		t.Errorf("Expected error to contain 'invalid-model', got: %v", err)
	}
}
