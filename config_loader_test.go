package golitecron

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromYaml_Success(t *testing.T) {
	// Create temp YAML file
	content := `tasks:
  - id: "task1"
    cron_expr: "* * * * *"
    func_name: "testFunc"
    timeout: "1s"
    retry: 3
  - id: "task2"
    cron_expr: "0 * * * *"
    func_name: "testFunc2"
    enable_seconds: false
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := LoadFromYaml(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromYaml failed: %v", err)
	}

	if len(config.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(config.Tasks))
	}

	if config.Tasks[0].ID != "task1" {
		t.Errorf("expected task1 ID, got %s", config.Tasks[0].ID)
	}

	if config.Tasks[0].Timeout != "1s" {
		t.Errorf("expected timeout 1s, got %s", config.Tasks[0].Timeout)
	}

	if config.Tasks[0].Retry != 3 {
		t.Errorf("expected retry 3, got %d", config.Tasks[0].Retry)
	}
}

func TestLoadFromYaml_FileNotFound(t *testing.T) {
	_, err := LoadFromYaml("/non/existent/path.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadFromYaml_InvalidFormat(t *testing.T) {
	content := `this is not valid yaml: [[[`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadFromYaml(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid YAML format")
	}
}

func TestLoadFromJson_Success(t *testing.T) {
	content := `{
  "tasks": [
    {
      "id": "json-task1",
      "cron_expr": "*/5 * * * *",
      "func_name": "jsonFunc",
      "timeout": "2s",
      "retry": 2,
      "location": "UTC"
    }
  ]
}`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := LoadFromJson(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromJson failed: %v", err)
	}

	if len(config.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(config.Tasks))
	}

	if config.Tasks[0].ID != "json-task1" {
		t.Errorf("expected json-task1 ID, got %s", config.Tasks[0].ID)
	}

	if config.Tasks[0].Location != "UTC" {
		t.Errorf("expected UTC location, got %s", config.Tasks[0].Location)
	}
}

func TestLoadFromJson_FileNotFound(t *testing.T) {
	_, err := LoadFromJson("/non/existent/path.json")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadFromJson_InvalidFormat(t *testing.T) {
	content := `{invalid json}`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadFromJson(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON format")
	}
}

func TestLoadFromYaml_EmptyTasks(t *testing.T) {
	content := `tasks: []`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := LoadFromYaml(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromYaml failed: %v", err)
	}

	if len(config.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(config.Tasks))
	}
}

func TestLoadFromYaml_WithAllFields(t *testing.T) {
	content := `tasks:
  - id: "full-task"
    cron_expr: "0 0 * * * *"
    func_name: "fullFunc"
    timeout: "5s"
    retry: 5
    location: "America/New_York"
    enable_seconds: true
    enable_years: true
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "full.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := LoadFromYaml(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromYaml failed: %v", err)
	}

	task := config.Tasks[0]
	if !task.EnableSeconds {
		t.Error("expected EnableSeconds to be true")
	}
	if !task.EnableYears {
		t.Error("expected EnableYears to be true")
	}
	if task.Location != "America/New_York" {
		t.Errorf("expected America/New_York location, got %s", task.Location)
	}
}
