package golitecron

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRejectsTaskOnlyOptionsAtCompileTime(t *testing.T) {
	output, err := buildExternalProgram(t, `package main

import (
	"time"

	cron "github.com/hansir-hsj/GoLiteCron"
)

func main() {
	_, _ = cron.Parse("* * * * *", cron.WithTimeout(time.Second))
}
`)
	if err == nil {
		t.Fatalf("go build succeeded; Parse should reject task-only options")
	}
	if !strings.Contains(string(output), "does not implement golitecron.ParseOption") {
		t.Fatalf("go build failed for an unexpected reason:\n%s", output)
	}
}

func TestParserFieldTypesAreInternal(t *testing.T) {
	output, err := buildExternalProgram(t, `package main

import cron "github.com/hansir-hsj/GoLiteCron"

func main() {
	var _ cron.FieldType
}
`)
	if err == nil {
		t.Fatalf("go build succeeded; parser field types should be internal")
	}
	if !strings.Contains(string(output), "undefined: cron.FieldType") {
		t.Fatalf("go build failed for an unexpected reason:\n%s", output)
	}
}

func buildExternalProgram(t *testing.T, mainGo string) (string, error) {
	t.Helper()

	moduleDir := t.TempDir()
	goMod := `module apicompilecheck

go 1.23.6

require (
	github.com/hansir-hsj/GoLiteCron v0.0.0
	github.com/robfig/cron/v3 v3.0.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/hansir-hsj/GoLiteCron => ` + mustAbs(t, ".") + `
`

	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return "", err
	}
	goSum, err := os.ReadFile("go.sum")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.sum"), goSum, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command("go", "build", ".")
	cmd.Dir = moduleDir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOCACHE="+filepath.Join(moduleDir, ".gocache"))
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
