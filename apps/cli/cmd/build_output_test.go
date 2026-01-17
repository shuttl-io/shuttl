package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuttl-ai/cli/ipc"
)

func TestWriteBuildOutputCreatesFilteredManifestsAndTar(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "main.ts"), []byte("console.log('hi')"), 0644); err != nil {
		t.Fatalf("write main.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("write ignored.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.txt\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}

	manifest := Manifest{
		Version:       "1.0",
		BuildTime:     "now",
		App:           "node ./main.ts",
		Language:      "node",
		BuildCommands: []string{"npm ci"},
		Agents: []ipc.AgentInfo{
			{Name: "AgentA", Toolkits: []string{"ToolkitA"}, Model: ipc.Model{Identifier: "model-a"}},
			{Name: "AgentB", Toolkits: []string{"ToolkitB"}, Model: ipc.Model{Identifier: "model-b"}},
		},
		Toolkits: []ipc.ToolkitInfo{
			{Name: "ToolkitA", Description: "A", Tools: []ipc.ToolInfo{{Name: "ToolA"}}},
			{Name: "ToolkitB", Description: "B", Tools: []ipc.ToolInfo{{Name: "ToolB"}}},
		},
		Tools: []ipc.SingleToolInfo{
			{Name: "ToolA", ToolkitName: "ToolkitA"},
			{Name: "ToolB", ToolkitName: "ToolkitB"},
		},
		Triggers: []ipc.TriggerInfo{
			{Name: "api", TriggerType: "api", AgentName: "AgentA"},
			{Name: "rate", TriggerType: "rate", AgentName: "AgentB"},
		},
		Models: []ipc.ModelInfo{
			{Identifier: "model-a"},
			{Identifier: "model-b"},
		},
		Prompts: []ipc.PromptInfo{
			{AgentName: "AgentA", SystemPrompt: "a"},
			{AgentName: "AgentB", SystemPrompt: "b"},
		},
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	if err := writeBuildOutput(root, manifest, manifestJSON, false, "amd64"); err != nil {
		t.Fatalf("writeBuildOutput: %v", err)
	}

	outputRoot := filepath.Join(root, ".shuttl_build")
	destRoot := filepath.Join(outputRoot, "AgentA", "api")
	if _, err := os.Stat(destRoot); err != nil {
		t.Fatalf("expected build output directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destRoot, "main.ts")); err != nil {
		t.Fatalf("expected main.ts in build output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destRoot, "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected ignored.txt to be excluded, got: %v", err)
	}

	dockerfileBytes, err := os.ReadFile(filepath.Join(destRoot, "Dockerfile"))
	if err != nil {
		t.Fatalf("read DOCKERFILE: %v", err)
	}
	if !strings.Contains(string(dockerfileBytes), "FROM scratch") {
		t.Fatalf("expected DOCKERFILE to include base dockerfile content")
	}
	if !strings.Contains(string(dockerfileBytes), "--agent=AgentA") || !strings.Contains(string(dockerfileBytes), "--trigger=api") {
		t.Fatalf("expected DOCKERFILE to include agent/trigger CMD")
	}

	manifestBytes, err := os.ReadFile(filepath.Join(destRoot, "shuttl-manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var filtered Manifest
	if err := json.Unmarshal(manifestBytes, &filtered); err != nil {
		t.Fatalf("unmarshal filtered manifest: %v", err)
	}
	if len(filtered.Agents) != 1 || filtered.Agents[0].Name != "AgentA" {
		t.Fatalf("expected filtered agent AgentA, got %+v", filtered.Agents)
	}
	if len(filtered.Triggers) != 1 || filtered.Triggers[0].Name != "api" {
		t.Fatalf("expected filtered trigger api, got %+v", filtered.Triggers)
	}
	if len(filtered.Toolkits) != 1 || filtered.Toolkits[0].Name != "ToolkitA" {
		t.Fatalf("expected filtered toolkit ToolkitA, got %+v", filtered.Toolkits)
	}
	if len(filtered.Tools) != 1 || filtered.Tools[0].Name != "ToolA" {
		t.Fatalf("expected filtered tool ToolA, got %+v", filtered.Tools)
	}
	if len(filtered.Models) != 1 || filtered.Models[0].Identifier != "model-a" {
		t.Fatalf("expected filtered model model-a, got %+v", filtered.Models)
	}
	if len(filtered.Prompts) != 1 || filtered.Prompts[0].AgentName != "AgentA" {
		t.Fatalf("expected filtered prompt AgentA, got %+v", filtered.Prompts)
	}

	tarPath := destRoot + ".tar.gz"
	entries, err := readTarGzEntries(tarPath)
	if err != nil {
		t.Fatalf("read tar: %v", err)
	}
	if !entries["main.ts"] || !entries["Dockerfile"] || !entries["shuttl-manifest.json"] {
		t.Fatalf("expected tar entries for main.ts, Dockerfile, shuttl-manifest.json, got %+v", entries)
	}
}

func TestWriteBuildOutputNoZipSkipsArchives(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.ts"), []byte("console.log('hi')"), 0644); err != nil {
		t.Fatalf("write main.ts: %v", err)
	}
	manifest := Manifest{
		Version:   "1.0",
		BuildTime: "now",
		App:       "node ./main.ts",
		Agents:    []ipc.AgentInfo{{Name: "AgentA"}},
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := writeBuildOutput(root, manifest, manifestJSON, true, "amd64"); err != nil {
		t.Fatalf("writeBuildOutput: %v", err)
	}
	outputRoot := filepath.Join(root, ".shuttl_build")
	destRoot := filepath.Join(outputRoot, "AgentA", "api")
	if _, err := os.Stat(destRoot + ".tar.gz"); !os.IsNotExist(err) {
		t.Fatalf("expected no tar.gz archive, got: %v", err)
	}
}

func TestWriteBuildOutputDefaultsTriggersWhenMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.ts"), []byte("console.log('hi')"), 0644); err != nil {
		t.Fatalf("write main.ts: %v", err)
	}
	manifest := Manifest{
		Version:   "1.0",
		BuildTime: "now",
		App:       "node ./main.ts",
		Agents:    []ipc.AgentInfo{{Name: "AgentA"}},
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := writeBuildOutput(root, manifest, manifestJSON, true, "amd64"); err != nil {
		t.Fatalf("writeBuildOutput: %v", err)
	}
	destRoot := filepath.Join(root, ".shuttl_build", "AgentA", "api")
	if _, err := os.Stat(destRoot); err != nil {
		t.Fatalf("expected default api trigger output, got: %v", err)
	}
}

func readTarGzEntries(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	entries := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entries[header.Name] = true
	}
	return entries, nil
}
