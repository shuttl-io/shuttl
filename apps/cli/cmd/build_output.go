package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/shuttl-ai/cli/config"
	"github.com/shuttl-ai/cli/ipc"
)

const buildOutputDirName = ".shuttl_build"

func resolveBuildCommands(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	return cfg.BuildCommands
}

func resolveLanguage(appPath string, cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Language) != "" {
		return strings.TrimSpace(cfg.Language)
	}
	return inferLanguageFromApp(appPath)
}

func inferLanguageFromApp(appPath string) string {
	parts := ipc.ParseCommand(appPath)
	if len(parts) == 0 {
		return ""
	}
	cmd := strings.ToLower(filepath.Base(parts[0]))
	switch cmd {
	case "python", "python3":
		return "python"
	case "node", "nodejs":
		return "node"
	case "bun":
		return "bun"
	case "deno":
		return "deno"
	case "go":
		return "go"
	}
	if len(parts) > 1 {
		ext := strings.ToLower(filepath.Ext(parts[1]))
		switch ext {
		case ".py":
			return "python"
		case ".ts", ".tsx":
			return "typescript"
		case ".js", ".mjs", ".cjs":
			return "javascript"
		case ".go":
			return "go"
		}
	}
	return ""
}

func writeBuildOutput(projectRoot string, manifest Manifest, manifestJSON []byte, noZip bool) error {
	outputRoot := filepath.Join(projectRoot, buildOutputDirName)
	if err := os.RemoveAll(outputRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(outputRoot, 0755); err != nil {
		return err
	}

	includedFiles, err := collectIncludedFiles(projectRoot, outputRoot)
	if err != nil {
		return err
	}

	dockerfileContent, err := dockerfileForBuild(projectRoot, manifest.BuildCommands)
	if err != nil {
		return err
	}

	triggers := manifest.Triggers
	if len(triggers) == 0 {
		for _, agent := range manifest.Agents {
			triggers = append(triggers, ipc.TriggerInfo{
				Name:        "api",
				TriggerType: "api",
				Description: "Default API trigger",
				AgentName:   agent.Name,
			})
		}
	}

	for _, trigger := range triggers {
		agentDir := sanitizeDir(trigger.AgentName)
		triggerDir := sanitizeDir(trigger.Name)
		if agentDir == "" || triggerDir == "" {
			return fmt.Errorf("invalid agent/trigger name for build output: %s/%s", trigger.AgentName, trigger.Name)
		}
		destRoot := filepath.Join(outputRoot, agentDir, triggerDir)
		if err := os.MkdirAll(destRoot, 0755); err != nil {
			return err
		}
		filtered, err := manifestForTrigger(manifest, trigger)
		if err != nil {
			return err
		}
		filteredJSON, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return err
		}
		for _, rel := range includedFiles {
			srcPath := filepath.Join(projectRoot, rel)
			destPath := filepath.Join(destRoot, rel)
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(destRoot, "DOCKERFILE"), []byte(dockerfileContent), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destRoot, "shuttl-manifest.json"), filteredJSON, 0644); err != nil {
			return err
		}

		if !noZip {
			tarPath := destRoot + ".tar.gz"
			if err := createTarGz(destRoot, tarPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func createTarGz(sourceDir, tarPath string) error {
	file, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == sourceDir {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if _, err := io.Copy(tarWriter, in); err != nil {
			return err
		}
		return nil
	})
}

func manifestForTrigger(manifest Manifest, trigger ipc.TriggerInfo) (Manifest, error) {
	agent, ok := findAgent(manifest.Agents, trigger.AgentName)
	if !ok {
		return Manifest{}, fmt.Errorf("agent not found for trigger %s/%s", trigger.AgentName, trigger.Name)
	}
	toolkitSet := make(map[string]struct{})
	for _, name := range agent.Toolkits {
		toolkitSet[name] = struct{}{}
	}
	var toolkits []ipc.ToolkitInfo
	for _, toolkit := range manifest.Toolkits {
		if _, ok := toolkitSet[toolkit.Name]; ok {
			toolkits = append(toolkits, toolkit)
		}
	}
	var tools []ipc.SingleToolInfo
	for _, tool := range manifest.Tools {
		if _, ok := toolkitSet[tool.ToolkitName]; ok {
			tools = append(tools, tool)
		}
	}
	var models []ipc.ModelInfo
	if agent.Model.Identifier != "" {
		for _, model := range manifest.Models {
			if model.Identifier == agent.Model.Identifier {
				models = append(models, model)
			}
		}
	}
	var prompts []ipc.PromptInfo
	for _, prompt := range manifest.Prompts {
		if prompt.AgentName == agent.Name {
			prompts = append(prompts, prompt)
		}
	}
	return Manifest{
		Version:       manifest.Version,
		BuildTime:     manifest.BuildTime,
		App:           manifest.App,
		Language:      manifest.Language,
		BuildCommands: manifest.BuildCommands,
		Agents:        []ipc.AgentInfo{agent},
		Toolkits:      toolkits,
		Tools:         tools,
		Triggers:      []ipc.TriggerInfo{trigger},
		Models:        models,
		Prompts:       prompts,
	}, nil
}

func findAgent(agents []ipc.AgentInfo, name string) (ipc.AgentInfo, bool) {
	for _, agent := range agents {
		if agent.Name == name {
			return agent, true
		}
	}
	return ipc.AgentInfo{}, false
}

func collectIncludedFiles(projectRoot, outputRoot string) ([]string, error) {
	var ignorer *ignore.GitIgnore
	ignorePath := filepath.Join(projectRoot, ".gitignore")
	if _, err := os.Stat(ignorePath); err == nil {
		compiled, err := ignore.CompileIgnoreFile(ignorePath)
		if err != nil {
			return nil, err
		}
		ignorer = compiled
	}

	var files []string
	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == buildOutputDirName || strings.HasPrefix(rel, buildOutputDirName+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == outputRoot || strings.HasPrefix(path, outputRoot+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ignorer != nil {
			relSlash := filepath.ToSlash(rel)
			if ignorer.MatchesPath(relSlash) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func dockerfileForBuild(projectRoot string, buildCommands []string) (string, error) {
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		data, err := os.ReadFile(dockerfilePath)
		if err != nil {
			return "", err
		}
		content := string(data)
		commands := formatBuildCommands(buildCommands)
		if strings.TrimSpace(commands) == "" {
			return content, nil
		}
		return content + "\n\n# Build commands from shuttl.json\n" + commands, nil
	}
	commands := formatBuildCommands(buildCommands)
	return renderDefaultDockerfile(commands), nil
}

func formatBuildCommands(commands []string) string {
	var cleaned []string
	for _, cmd := range commands {
		trimmed := strings.TrimSpace(cmd)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, cmd := range cleaned {
		builder.WriteString("RUN ")
		builder.WriteString(cmd)
		builder.WriteString("\n")
	}
	return builder.String()
}

func renderDefaultDockerfile(buildCommands string) string {
	template := `FROM node:20-slim

WORKDIR /app
COPY . .

RUN curl -fsSL https://shuttl.io/install.sh | bash

{{BUILD_COMMANDS}}

EXPOSE 8443
ENTRYPOINT ["shuttl", "serve"]
CMD ["--port", "8443", "--insecure"]
`
	if strings.TrimSpace(buildCommands) == "" {
		return strings.ReplaceAll(template, "{{BUILD_COMMANDS}}", "")
	}
	return strings.ReplaceAll(template, "{{BUILD_COMMANDS}}", strings.TrimRight(buildCommands, "\n"))
}

func sanitizeDir(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.ReplaceAll(clean, string(os.PathSeparator), "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	return clean
}

func copyFile(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("copyFile called with directory source")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
