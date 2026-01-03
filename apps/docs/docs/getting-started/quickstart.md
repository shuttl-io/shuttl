# Quick Start

Get your first Shuttl agent running in under 5 minutes.

---

## Prerequisites

Before you begin, make sure you have:

- **Node.js 18+** or **Python 3.9+**
- An **OpenAI API key** (or another supported LLM provider)
- A terminal and your favorite code editor

---

## Step 1: Install the SDK

=== "TypeScript"

    ```bash
    # Create a new project
    mkdir my-agent && cd my-agent
    npm init -y

    # Install Shuttl
    npm install @shuttl-io/core typescript ts-node @types/node

    # Initialize TypeScript
    npx tsc --init
    ```

=== "Python"

    ```bash
    # Create a new project
    mkdir my-agent && cd my-agent

    # Create virtual environment
    python -m venv .venv
    source .venv/bin/activate  # On Windows: .venv\Scripts\activate

    # Install Shuttl
    pip install shuttl
    ```

---

## Step 2: Install the CLI

The Shuttl CLI provides development tools and deployment capabilities.

```bash
# Download for your platform
curl -fsSL https://shuttl.dev/install.sh | bash

# Verify installation
shuttl version
```

Or download directly from [GitHub Releases](https://github.com/shuttl-io/shuttl/releases).

---

## Step 3: Create Your Agent

=== "TypeScript"

    Create `agent.ts`:

    ```typescript
    import { Agent, Model, Secret, Schema } from "@shuttl-io/core";

    // Define a simple tool
    const greetTool = {
        name: "greet",
        description: "Generate a personalized greeting",
        schema: Schema.objectValue({
            name: Schema.stringValue("The person's name").isRequired(),
            style: Schema.enumValue("Greeting style", ["formal", "casual", "enthusiastic"])
                .defaultTo("casual"),
        }),
        execute: async (args: Record<string, unknown>) => {
            const { name, style } = args;
            const greetings = {
                formal: `Good day, ${name}. How may I assist you?`,
                casual: `Hey ${name}! What's up?`,
                enthusiastic: `${name}!!! So great to see you! 🎉`,
            };
            return { greeting: greetings[style as keyof typeof greetings] };
        },
    };

    // Create the agent
    export const greeterAgent = new Agent({
        name: "GreeterAgent",
        systemPrompt: `You are a friendly greeter. Use the greet tool to 
        welcome users based on their preferences.`,
        model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY")),
        tools: [greetTool],
    });
    ```

=== "Python"

    Create `agent.py`:

    ```python
    from shuttl import Application

    app = Application("my-agent")

    @app.toolkit("greetings", "Tools for greeting users")
    class GreetingToolkit:
        
        @tool(
            name="greet",
            description="Generate a personalized greeting",
            args={
                "name": {"type": "string", "required": True},
                "style": {"type": "enum", "values": ["formal", "casual"]},
            }
        )
        def greet(self, name: str, style: str = "casual") -> dict:
            greetings = {
                "formal": f"Good day, {name}. How may I assist you?",
                "casual": f"Hey {name}! What's up?",
            }
            return {"greeting": greetings[style]}
    ```

---

## Step 4: Configure Your App

Create `shuttl.json` in your project root:

```json
{
    "app": "node --require ts-node/register ./agent.ts"
}
```

!!! tip "For Python"
    ```json
    {
        "app": "python agent.py"
    }
    ```

---

## Step 5: Set Your API Key

```bash
export OPENAI_API_KEY="sk-your-key-here"
```

!!! warning "Keep it secret"
    Never commit API keys to version control. Use environment variables or a secrets manager.

---

## Step 6: Run Your Agent

```bash
shuttl dev
```

This launches the Shuttl TUI (Terminal User Interface) with four screens:

| Screen | Description |
|--------|-------------|
| **Agent Select** | Choose which agent to interact with |
| **Chat** | Conversational interface with your agent |
| **Agent Debug** | View agent state, tool calls, and responses |
| **Program Debug** | Monitor logs and system-level events |

---

## Step 7: Test Your Agent

1. Use the **Agent Select** screen to choose `GreeterAgent`
2. Navigate to the **Chat** screen
3. Type: `Say hello to Alex in an enthusiastic way!`

Watch the agent:

1. Process your message
2. Call the `greet` tool with `{"name": "Alex", "style": "enthusiastic"}`
3. Return: `Alex!!! So great to see you! 🎉`

Use the **Agent Debug** screen to see the full tool call flow and LLM responses in real-time.

---

## What's Next?

<div class="grid cards" markdown>

-   **:book: Learn the Concepts**

    ---

    Understand Agents, Tools, Triggers, and Outcomes.

    [:octicons-arrow-right-24: Core Concepts](../concepts/index.md)

-   **:wrench: Build More Tools**

    ---

    Create powerful tools that connect to APIs and databases.

    [:octicons-arrow-right-24: Tools Guide](../concepts/tools.md)

-   **:clock: Add Triggers**

    ---

    Schedule your agent or connect it to webhooks.

    [:octicons-arrow-right-24: Triggers](../concepts/triggers.md)

-   **:rocket: Deploy to Production**

    ---

    Ship your agent to the cloud.

    [:octicons-arrow-right-24: Deployment Guide](installation.md)

</div>

