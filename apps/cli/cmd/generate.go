package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate <name>",
	Short: "Generate a new Shuttl AI agent project",
	Long: `Generate a new Shuttl AI agent project with all necessary scaffolding.

This command creates a new directory with the specified name and sets up 
a complete Shuttl AI project with the chosen language, including:
  - Project configuration files
  - Dependency management files  
  - A basic agent example
  - The shuttl.json configuration file

Supported languages:
  - typescript (ts)  - TypeScript/Node.js project
  - python (py)      - Python project with uv/pip
  - go               - Go project with modules
  - java             - Java project with Maven
  - csharp (cs)      - C# project with .NET

Examples:
  shuttl generate my-agent --language typescript
  shuttl generate my-agent -l python
  shuttl generate my-agent -l go`,
	Args: cobra.ExactArgs(1),
	Run:  runGenerate,
}

func init() {
	generateCmd.Flags().StringP("language", "l", "", "Programming language (typescript, python, go, java, csharp)")
	generateCmd.Flags().Bool("skip-install", false, "Skip installing dependencies")
	generateCmd.MarkFlagRequired("language")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) {
	projectPath := args[0]
	language, _ := cmd.Flags().GetString("language")
	skipInstall, _ := cmd.Flags().GetBool("skip-install")

	// Normalize language aliases
	language = normalizeLanguage(language)

	if !isValidLanguage(language) {
		fmt.Fprintf(os.Stderr, "❌ Error: unsupported language '%s'. Supported languages: typescript, python, go, java, csharp\n", language)
		os.Exit(1)
	}

	// Create project directory
	projectDir, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error resolving project path: %v\n", err)
		os.Exit(1)
	}

	// Extract the base name as the project name
	projectName := filepath.Base(projectDir)

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Error: directory '%s' already exists\n", projectPath)
		os.Exit(1)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error creating project directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Generating %s Shuttl AI project: %s\n", language, projectName)

	var genErr error
	switch language {
	case "typescript":
		genErr = generateTypeScriptProject(projectDir, projectName, skipInstall)
	case "python":
		genErr = generatePythonProject(projectDir, projectName, skipInstall)
	case "go":
		genErr = generateGoProject(projectDir, projectName, skipInstall)
	case "java":
		genErr = generateJavaProject(projectDir, projectName, skipInstall)
	case "csharp":
		genErr = generateCSharpProject(projectDir, projectName, skipInstall)
	}

	if genErr != nil {
		fmt.Fprintf(os.Stderr, "❌ Error generating project: %v\n", genErr)
		// Cleanup on failure
		os.RemoveAll(projectDir)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Project '%s' created successfully!\n\n", projectName)
	printNextSteps(projectDir, language)
}

func normalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "ts", "node", "nodejs":
		return "typescript"
	case "py":
		return "python"
	case "golang":
		return "go"
	case "cs", "dotnet", ".net":
		return "csharp"
	default:
		return lang
	}
}

func isValidLanguage(lang string) bool {
	validLanguages := []string{"typescript", "python", "go", "java", "csharp"}
	for _, valid := range validLanguages {
		if lang == valid {
			return true
		}
	}
	return false
}

func writeShuttlJSON(projectDir, appCommand string) error {
	config := map[string]string{
		"app": appCommand,
	}
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, "shuttl.json"), data, 0644)
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandSilent(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

// ============================================================================
// TypeScript Project Generator
// ============================================================================

func generateTypeScriptProject(projectDir, projectName string, skipInstall bool) error {
	fmt.Println("📦 Creating TypeScript project structure...")

	// Create package.json
	packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "description": "A Shuttl AI agent project",
  "main": "src/main.ts",
  "type": "module",
  "scripts": {
    "start": "npx tsx src/main.ts",
    "dev": "shuttl dev",
    "build": "tsc"
  },
  "dependencies": {
    "@shuttl-io/core": "^0.1.5"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "tsx": "^4.19.0",
    "typescript": "^5.6.0"
  }
}
`, projectName)
	if err := writeFile(filepath.Join(projectDir, "package.json"), packageJSON); err != nil {
		return err
	}

	// Create tsconfig.json
	tsConfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "esModuleInterop": true,
    "strict": true,
    "skipLibCheck": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "declaration": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`
	if err := writeFile(filepath.Join(projectDir, "tsconfig.json"), tsConfig); err != nil {
		return err
	}

	// Create src directory and main.ts
	mainTS := `import { App, StdInServer, Agent, Model, Secret, Schema } from "@shuttl-io/core";

// Create a model using OpenAI (you can change this to other providers)
const model = Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY"));

// Define a simple tool for your agent
const greetTool = {
  name: "greet",
  description: "Greet someone by name",
  schema: Schema.objectValue({
    name: Schema.stringValue("The name of the person to greet").isRequired(),
  }),
  execute: async (args: Record<string, unknown>): Promise<unknown> => {
    return { message: ` + "`Hello, ${args.name}!`" + ` };
  },
};

// Create your agent
const myAgent = new Agent({
  name: "MyAgent",
  systemPrompt: "You are a helpful assistant that can greet people.",
  model: model,
  tools: [greetTool],
  toolkits: [],
  triggers: [],
});

// Main function to start the server
async function main() {
  const server = new StdInServer();
  const app = new App("` + projectName + `", server);
  
  app.addAgent(myAgent);
  app.serve();
}

main();
`
	if err := writeFile(filepath.Join(projectDir, "src", "main.ts"), mainTS); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `node_modules/
dist/
.env
*.log
`
	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// Create .env.example
	envExample := `# Add your API keys here
OPENAI_API_KEY=your-api-key-here
`
	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// Create shuttl.json
	if err := writeShuttlJSON(projectDir, "npx tsx src/main.ts"); err != nil {
		return err
	}

	// Install dependencies
	if !skipInstall {
		fmt.Println("📥 Installing dependencies...")
		if err := runCommand(projectDir, "npm", "install"); err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipping dependency installation (run 'npm install' to install)")
	}

	return nil
}

// ============================================================================
// Python Project Generator
// ============================================================================

func generatePythonProject(projectDir, projectName string, skipInstall bool) error {
	fmt.Println("📦 Creating Python project structure...")

	// Normalize project name for Python (replace hyphens with underscores)
	moduleName := strings.ReplaceAll(projectName, "-", "_")

	// Create pyproject.toml
	pyprojectTOML := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
description = "A Shuttl AI agent project"
requires-python = ">=3.10"
readme = "README.md"
dependencies = [
    "shuttl-core>=0.1.5",
]

[project.scripts]
%s = "%s.main:main"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["%s"]
`, projectName, moduleName, moduleName, moduleName)
	if err := writeFile(filepath.Join(projectDir, "pyproject.toml"), pyprojectTOML); err != nil {
		return err
	}

	// Create module directory and files
	moduleDir := filepath.Join(projectDir, moduleName)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	// Create __init__.py
	if err := writeFile(filepath.Join(moduleDir, "__init__.py"), `"""Shuttl AI Agent Project"""`+"\n"); err != nil {
		return err
	}

	// Create main.py
	mainPy := `"""Main entry point for the Shuttl AI agent."""
import os
from shuttl_core import App, StdInServer, Agent, Model, Secret, Schema


def create_greet_tool():
    """Create a simple greeting tool."""
    return {
        "name": "greet",
        "description": "Greet someone by name",
        "schema": Schema.object_value({
            "name": Schema.string_value("The name of the person to greet").is_required(),
        }),
        "execute": lambda args: {"message": f"Hello, {args['name']}!"},
    }


def create_agent():
    """Create the main agent."""
    model = Model.open_ai("gpt-4", Secret.from_env("OPENAI_API_KEY"))
    
    return Agent(
        name="MyAgent",
        system_prompt="You are a helpful assistant that can greet people.",
        model=model,
        tools=[create_greet_tool()],
        toolkits=[],
        triggers=[],
    )


def main():
    """Start the Shuttl AI server."""
    server = StdInServer()
    app = App("` + projectName + `", server)
    
    app.add_agent(create_agent())
    app.serve()


if __name__ == "__main__":
    main()
`
	if err := writeFile(filepath.Join(moduleDir, "main.py"), mainPy); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `__pycache__/
*.py[cod]
*$py.class
.Python
.venv/
venv/
.env
*.egg-info/
dist/
build/
`
	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// Create .env.example
	envExample := `# Add your API keys here
OPENAI_API_KEY=your-api-key-here
`
	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// Create README.md
	readme := fmt.Sprintf("# %s\n\nA Shuttl AI agent project.\n\n## Setup\n\n"+
		"1. Create a virtual environment:\n```bash\n"+
		"python -m venv .venv\nsource .venv/bin/activate  # On Windows: .venv\\Scripts\\activate\n```\n\n"+
		"2. Install dependencies:\n```bash\npip install -e .\n```\n\n"+
		"3. Copy `.env.example` to `.env` and add your API keys.\n\n"+
		"4. Run the agent:\n```bash\nshuttl dev\n```\n", projectName)
	if err := writeFile(filepath.Join(projectDir, "README.md"), readme); err != nil {
		return err
	}

	// Create shuttl.json
	if err := writeShuttlJSON(projectDir, fmt.Sprintf("python -m %s.main", moduleName)); err != nil {
		return err
	}

	// Try to install dependencies with uv, fallback to pip
	if !skipInstall {
		fmt.Println("📥 Installing dependencies...")
		if err := runCommandSilent(projectDir, "uv", "--version"); err == nil {
			// uv is available
			if err := runCommand(projectDir, "uv", "venv"); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w", err)
			}
			if err := runCommand(projectDir, "uv", "pip", "install", "-e", "."); err != nil {
				return fmt.Errorf("failed to install dependencies: %w", err)
			}
		} else {
			// Fallback to pip
			fmt.Println("   (uv not found, using pip)")
			if err := runCommand(projectDir, "python", "-m", "venv", ".venv"); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w", err)
			}
			// Install with pip in venv
			pipPath := filepath.Join(projectDir, ".venv", "bin", "pip")
			if _, err := os.Stat(pipPath); os.IsNotExist(err) {
				pipPath = filepath.Join(projectDir, ".venv", "Scripts", "pip.exe") // Windows
			}
			if err := runCommand(projectDir, pipPath, "install", "-e", "."); err != nil {
				return fmt.Errorf("failed to install dependencies: %w", err)
			}
		}
	} else {
		fmt.Println("⏭️  Skipping dependency installation (run 'pip install -e .' to install)")
	}

	return nil
}

// ============================================================================
// Go Project Generator
// ============================================================================

func generateGoProject(projectDir, projectName string, skipInstall bool) error {
	fmt.Println("📦 Creating Go project structure...")

	// Create go.mod
	goMod := fmt.Sprintf(`module %s

go 1.23

require github.com/shuttl-io/shuttl-core-go/core v0.1.5
`, projectName)
	if err := writeFile(filepath.Join(projectDir, "go.mod"), goMod); err != nil {
		return err
	}

	// Create main.go
	mainGo := `package main

import (
	"os"

	"github.com/shuttl-io/shuttl-core-go/core"
)

func main() {
	// Create the server
	server := core.NewStdInServer()

	// Create the app
	app := core.NewApp(ptr("` + projectName + `"), server)

	// Create and add the agent
	agent := createAgent()
	app.AddAgent(agent)

	// Start serving
	app.Serve()
}

func createAgent() core.Agent {
	// Create the model
	apiKey := os.Getenv("OPENAI_API_KEY")
	secret := core.NewEnvSecret(ptr("OPENAI_API_KEY"))
	model := core.Model_OpenAI(ptr("gpt-4"), secret)

	_ = apiKey // suppress unused warning

	// Create a simple greeting tool
	greetTool := &GreetTool{}

	// Create the agent
	agent := core.NewAgent(&core.AgentProps{
		Name:         ptr("MyAgent"),
		SystemPrompt: ptr("You are a helpful assistant that can greet people."),
		Model:        model,
		Tools:        &[]core.ITool{greetTool},
		Toolkits:     &[]core.Toolkit{},
		Triggers:     &[]core.ITrigger{},
	})

	return agent
}

// Helper function to create string pointers
func ptr(s string) *string {
	return &s
}

// GreetTool implements core.ITool
type GreetTool struct{}

func (t *GreetTool) Name() *string {
	return ptr("greet")
}

func (t *GreetTool) Description() *string {
	return ptr("Greet someone by name")
}

func (t *GreetTool) Schema() core.ToolArg {
	return core.Schema_ObjectValue(&map[string]core.ToolArg{
		"name": core.Schema_StringValue(ptr("The name of the person to greet")).IsRequired(),
	})
}

func (t *GreetTool) Execute(args *map[string]interface{}) interface{} {
	name := (*args)["name"].(string)
	return map[string]string{
		"message": "Hello, " + name + "!",
	}
}
`
	if err := writeFile(filepath.Join(projectDir, "main.go"), mainGo); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := fmt.Sprintf(`# Binaries
%s
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Environment
.env
`, projectName)
	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// Create .env.example
	envExample := `# Add your API keys here
OPENAI_API_KEY=your-api-key-here
`
	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// Create shuttl.json
	if err := writeShuttlJSON(projectDir, fmt.Sprintf("go run .")); err != nil {
		return err
	}

	// Download dependencies
	if !skipInstall {
		fmt.Println("📥 Downloading dependencies...")
		if err := runCommand(projectDir, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("failed to download dependencies: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipping dependency installation (run 'go mod tidy' to install)")
	}

	return nil
}

// ============================================================================
// Java Project Generator
// ============================================================================

func generateJavaProject(projectDir, projectName string, skipInstall bool) error {
	fmt.Println("📦 Creating Java project structure...")

	// Convert project name to valid Java package name
	packageName := strings.ReplaceAll(strings.ToLower(projectName), "-", "")
	groupId := "com.example"
	artifactId := projectName

	// Create pom.xml
	pomXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>%s</groupId>
    <artifactId>%s</artifactId>
    <version>0.1.0</version>
    <packaging>jar</packaging>

    <name>%s</name>
    <description>A Shuttl AI agent project</description>

    <properties>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <dependency>
            <groupId>io.shuttl.module</groupId>
            <artifactId>core</artifactId>
            <version>0.1.5</version>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-jar-plugin</artifactId>
                <version>3.3.0</version>
                <configuration>
                    <archive>
                        <manifest>
                            <mainClass>%s.%s.Main</mainClass>
                        </manifest>
                    </archive>
                </configuration>
            </plugin>
            <plugin>
                <groupId>org.codehaus.mojo</groupId>
                <artifactId>exec-maven-plugin</artifactId>
                <version>3.1.0</version>
                <configuration>
                    <mainClass>%s.%s.Main</mainClass>
                </configuration>
            </plugin>
        </plugins>
    </build>
</project>
`, groupId, artifactId, projectName, groupId, packageName, groupId, packageName)
	if err := writeFile(filepath.Join(projectDir, "pom.xml"), pomXML); err != nil {
		return err
	}

	// Create source directory structure
	srcDir := filepath.Join(projectDir, "src", "main", "java", "com", "example", packageName)

	// Create Main.java
	mainJava := fmt.Sprintf(`package %s.%s;

import io.shuttl.module.core.*;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Main {
    public static void main(String[] args) {
        // Create the server
        StdInServer server = new StdInServer();

        // Create the app
        App app = new App("%s", server);

        // Create and add the agent
        Agent agent = createAgent();
        app.addAgent(agent);

        // Start serving
        app.serve();
    }

    private static Agent createAgent() {
        // Create the model
        ISecret secret = Secret.fromEnv("OPENAI_API_KEY");
        IModel model = Model.openAI("gpt-4", secret);

        // Create tools list
        List<ITool> tools = new ArrayList<>();
        tools.add(new GreetTool());

        // Create the agent
        AgentProps props = AgentProps.builder()
            .name("MyAgent")
            .systemPrompt("You are a helpful assistant that can greet people.")
            .model(model)
            .tools(tools)
            .toolkits(new ArrayList<>())
            .triggers(new ArrayList<>())
            .build();

        return new Agent(props);
    }
}
`, groupId, packageName, projectName)
	if err := writeFile(filepath.Join(srcDir, "Main.java"), mainJava); err != nil {
		return err
	}

	// Create GreetTool.java
	greetToolJava := fmt.Sprintf(`package %s.%s;

import io.shuttl.module.core.*;
import java.util.HashMap;
import java.util.Map;

public class GreetTool implements ITool {
    @Override
    public String getName() {
        return "greet";
    }

    @Override
    public String getDescription() {
        return "Greet someone by name";
    }

    @Override
    public ToolArg getSchema() {
        Map<String, ToolArg> properties = new HashMap<>();
        properties.put("name", Schema.stringValue("The name of the person to greet").isRequired());
        return Schema.objectValue(properties);
    }

    @Override
    public Object execute(Map<String, Object> args) {
        String name = (String) args.get("name");
        Map<String, String> result = new HashMap<>();
        result.put("message", "Hello, " + name + "!");
        return result;
    }
}
`, groupId, packageName)
	if err := writeFile(filepath.Join(srcDir, "GreetTool.java"), greetToolJava); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `target/
*.class
*.jar
*.war
*.ear
.idea/
*.iml
.settings/
.project
.classpath
.env
`
	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// Create .env.example
	envExample := `# Add your API keys here
OPENAI_API_KEY=your-api-key-here
`
	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// Create shuttl.json
	if err := writeShuttlJSON(projectDir, "mvn exec:java"); err != nil {
		return err
	}

	// Install dependencies
	if !skipInstall {
		fmt.Println("📥 Installing dependencies...")
		if err := runCommand(projectDir, "mvn", "dependency:resolve"); err != nil {
			fmt.Println("   ⚠️  Maven not found or failed. Run 'mvn dependency:resolve' manually.")
		}
	} else {
		fmt.Println("⏭️  Skipping dependency installation (run 'mvn dependency:resolve' to install)")
	}

	return nil
}

// ============================================================================
// C# Project Generator
// ============================================================================

func generateCSharpProject(projectDir, projectName string, skipInstall bool) error {
	fmt.Println("📦 Creating C# project structure...")

	// Create .csproj file
	csproj := fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <RootNamespace>%s</RootNamespace>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="shuttl.core" Version="0.1.5" />
  </ItemGroup>

</Project>
`, strings.ReplaceAll(projectName, "-", "_"))
	if err := writeFile(filepath.Join(projectDir, projectName+".csproj"), csproj); err != nil {
		return err
	}

	// Create Program.cs
	programCS := fmt.Sprintf(`using shuttl.core;

namespace %s;

class Program
{
    static void Main(string[] args)
    {
        // Create the server
        var server = new StdInServer();

        // Create the app
        var app = new App("%s", server);

        // Create and add the agent
        var agent = CreateAgent();
        app.AddAgent(agent);

        // Start serving
        app.Serve();
    }

    static Agent CreateAgent()
    {
        // Create the model
        var secret = Secret.FromEnv("OPENAI_API_KEY");
        var model = Model.OpenAI("gpt-4", secret);

        // Create tools
        var tools = new ITool[] { new GreetTool() };

        // Create the agent
        var props = new AgentProps
        {
            Name = "MyAgent",
            SystemPrompt = "You are a helpful assistant that can greet people.",
            Model = model,
            Tools = tools,
            Toolkits = Array.Empty<Toolkit>(),
            Triggers = Array.Empty<ITrigger>()
        };

        return new Agent(props);
    }
}

class GreetTool : ITool
{
    public string Name => "greet";
    public string Description => "Greet someone by name";

    public ToolArg Schema => shuttl.core.Schema.ObjectValue(
        new Dictionary<string, ToolArg>
        {
            ["name"] = shuttl.core.Schema.StringValue("The name of the person to greet").IsRequired()
        }
    );

    public object Execute(IDictionary<string, object> args)
    {
        var name = args["name"]?.ToString() ?? "World";
        return new Dictionary<string, string>
        {
            ["message"] = $"Hello, {name}!"
        };
    }
}
`, strings.ReplaceAll(projectName, "-", "_"), projectName)
	if err := writeFile(filepath.Join(projectDir, "Program.cs"), programCS); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `bin/
obj/
.vs/
*.user
*.suo
.env
`
	if err := writeFile(filepath.Join(projectDir, ".gitignore"), gitignore); err != nil {
		return err
	}

	// Create .env.example
	envExample := `# Add your API keys here
OPENAI_API_KEY=your-api-key-here
`
	if err := writeFile(filepath.Join(projectDir, ".env.example"), envExample); err != nil {
		return err
	}

	// Create shuttl.json
	if err := writeShuttlJSON(projectDir, "dotnet run"); err != nil {
		return err
	}

	// Restore dependencies
	if !skipInstall {
		fmt.Println("📥 Installing dependencies...")
		if err := runCommand(projectDir, "dotnet", "restore"); err != nil {
			fmt.Println("   ⚠️  .NET SDK not found or failed. Run 'dotnet restore' manually.")
		}
	} else {
		fmt.Println("⏭️  Skipping dependency installation (run 'dotnet restore' to install)")
	}

	return nil
}

// ============================================================================
// Help Text
// ============================================================================

func printNextSteps(projectDir, language string) {
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s\n", projectDir)
	fmt.Println("  2. Copy .env.example to .env and add your API keys")

	switch language {
	case "typescript":
		fmt.Println("  3. Run: shuttl dev")
	case "python":
		fmt.Println("  3. Activate the virtual environment:")
		fmt.Println("       source .venv/bin/activate  # On Windows: .venv\\Scripts\\activate")
		fmt.Println("  4. Run: shuttl dev")
	case "go":
		fmt.Println("  3. Run: shuttl dev")
	case "java":
		fmt.Println("  3. Build: mvn compile")
		fmt.Println("  4. Run: shuttl dev")
	case "csharp":
		fmt.Println("  3. Run: shuttl dev")
	}

	fmt.Println("\nFor more information, visit: https://docs.shuttl.io")
}
