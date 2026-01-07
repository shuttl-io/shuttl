# @shuttl-io/core

A developer-first framework for building, deploying, and managing AI agents. Stop wrestling with complex infrastructure and start shipping intelligent agents in minutes.

## Installation

**TypeScript/JavaScript:**

```bash
npm install @shuttl-io/core
# or
pnpm add @shuttl-io/core
# or
yarn add @shuttl-io/core
```

**Python:**

```bash
pip install shuttl-core
```

**Java (Maven):**

```xml
<dependency>
  <groupId>io.shuttl.module</groupId>
  <artifactId>core</artifactId>
  <version>0.1.5</version>
</dependency>
```

**Go:**

```bash
go get github.com/shuttl-io/shuttl-core-go
```

**.NET:**

```bash
dotnet add package shuttl.core
```

## Quick Start

### TypeScript

```typescript
import { Agent, Model, Secret, Schema } from "@shuttl-io/core";

const weatherTool = {
    name: "get_weather",
    description: "Get current weather for a location",
    schema: Schema.objectValue({
        location: Schema.stringValue("City name").isRequired(),
    }),
    execute: async (args) => {
        return { temperature: 72, condition: "sunny" };
    },
};

export const weatherAgent = new Agent({
    name: "WeatherBot",
    systemPrompt: "You help users check the weather.",
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY")),
    tools: [weatherTool],
});
```

### Python

In Python, tools must be implemented using the `@jsii.implements()` decorator. **Do not inherit directly from the interface** - this will cause metaclass conflicts. See the [JSII Python documentation](https://aws.github.io/jsii/user-guides/lib-user/language-specific/python/#implementing-interfaces) for details.

```python
import jsii
from shuttl.core import App, Agent, Model, Secret
from shuttl.core.tools import ITool, ToolArg


@jsii.implements(ITool)
class WeatherTool:
    """A tool that gets weather information."""
    
    def __init__(self):
        self._name = "get_weather"
        self._description = "Get current weather for a location"
    
    @property
    def name(self) -> str:
        return self._name
    
    @name.setter
    def name(self, value: str):
        self._name = value
    
    @property
    def description(self) -> str:
        return self._description
    
    @description.setter
    def description(self, value: str):
        self._description = value
    
    def execute(self, args):
        location = args.get("location", "Unknown")
        return {"temperature": 72, "condition": "sunny", "location": location}
    
    def produce_args(self):
        return {
            "location": ToolArg(
                name="location",
                arg_type="string",
                description="City name",
                required=True,
                default_value=None,
            ),
        }


def main():
    app = App("weather-bot")
    
    model = Model.open_ai("gpt-4", Secret.from_env("OPENAI_KEY"))
    agent = Agent(
        name="WeatherBot",
        system_prompt="You help users check the weather.",
        model=model,
        tools=[WeatherTool()],
    )
    
    app.add_agent(agent)
    app.serve()


if __name__ == "__main__":
    main()
```

Run `shuttl dev` and your agent is live.

## Core Concepts

Shuttl is built around four composable primitives:

```
┌─────────────────────────────────────────────────────────────┐
│                         TRIGGERS                            │
│   (API, Rate/Cron, Email, File, Webhook)                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                          AGENT                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   System    │  │    Model    │  │   Tools & Toolkits  │ │
│  │   Prompt    │  │  (GPT, etc) │  │                     │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        OUTCOMES                             │
│   (Streaming, Slack, Webhook, Custom)                       │
└─────────────────────────────────────────────────────────────┘
```

### Agents

The core unit of intelligence. An agent combines a language model with a system prompt and tools to perform tasks.

```typescript
export const myAgent = new Agent({
    name: "MyAgent",
    systemPrompt: "You are a helpful assistant.",
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY")),
    tools: [],
    triggers: [],
    outcomes: [],
});
```

**Key features:**
- Multi-turn conversations with thread management
- Automatic retries with exponential backoff
- Full TypeScript support for type safety

### Tools & Toolkits

Give agents the ability to take actions: search databases, call APIs, process data, and more.

```typescript
const searchTool = {
    name: "search",
    description: "Search the knowledge base",
    schema: Schema.objectValue({
        query: Schema.stringValue("Search query").isRequired(),
        limit: Schema.numberValue("Max results").defaultTo(10),
    }),
    execute: async ({ query, limit }) => {
        return await searchKnowledgeBase(query, limit);
    },
};
```

Group related tools into **Toolkits** for clean, modular agent design:

```typescript
const weatherToolkit = new Toolkit("weather", "Tools for weather information");
weatherToolkit.addTool(getCurrentWeatherTool);
weatherToolkit.addTool(getForecastTool);
```

### Triggers

Define how and when agents are activated:

| Trigger | Description | Use Case |
|---------|-------------|----------|
| `ApiTrigger` | HTTP endpoint | Chat interfaces, webhooks |
| `Rate` | Schedule-based | Cron jobs, periodic tasks |
| `EmailTrigger` | Incoming emails | Email automation |
| `FileTrigger` | File system changes | Document processing |

```typescript
// Run every hour
triggers: [Rate.hours(1).bindOutcome(new StreamingOutcome())]

// Or use cron for precise scheduling
triggers: [Rate.cron("0 9 * * MON-FRI", "America/New_York")]
```

### Outcomes

Route agent responses to destinations:

| Outcome | Description | Use Case |
|---------|-------------|----------|
| `StreamingOutcome` | Real-time streaming | Chat interfaces, live feedback |
| `SlackOutcome` | Post to Slack | Notifications, reports |
| `CombinationOutcome` | Multiple destinations | Broadcast to several channels |

```typescript
outcomes: [
    new CombinationOutcome([
        new StreamingOutcome(),
        new SlackOutcome("#announcements"),
    ]),
]
```

### Models

Language models that power your agents. Currently supports OpenAI with more providers coming soon.

```typescript
// Complex analysis, important decisions
Model.openAI("gpt-4", Secret.fromEnv("KEY"))

// General purpose, good balance
Model.openAI("gpt-4o", Secret.fromEnv("KEY"))

// High volume, simple tasks
Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY"))
```

**Supported providers:**
- ✅ OpenAI (GPT-4, GPT-4o, GPT-4o-mini, GPT-3.5)
- 🚧 Anthropic (Claude 3, Claude 3.5) - Coming Soon
- 🚧 Google (Gemini Pro, Gemini Ultra) - Coming Soon
- 🚧 Local (Ollama, LM Studio) - Coming Soon

## Example: Scheduled Reporting Agent

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome, Schema } from "@shuttl-io/core";

const reportTool = {
    name: "generate_report",
    description: "Generate a daily metrics report",
    schema: Schema.objectValue({
        date: Schema.stringValue("Report date in YYYY-MM-DD format"),
    }),
    execute: async ({ date }) => {
        return { users: 1523, revenue: 42000, churn: 0.02 };
    },
};

export const reportingAgent = new Agent({
    name: "DailyReporter",
    systemPrompt: "Generate concise daily reports. Format as bullet points.",
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY")),
    tools: [reportTool],
    triggers: [
        Rate.cron("0 9 * * *", "America/New_York")  // 9 AM EST daily
    ],
    outcomes: [
        new SlackOutcome("#metrics-channel")
    ],
});
```

## CLI Commands

Install the Shuttl CLI:

```bash
curl -fsSL https://shuttl.dev/install.sh | bash
```

| Command | Description |
|---------|-------------|
| `shuttl dev` | Interactive development with TUI |
| `shuttl serve` | Expose agents as HTTP endpoints |
| `shuttl build` | Build your agent for deployment |

## Configuration

Create `shuttl.json` in your project root:

```json
{
    "app": "node --require ts-node/register ./src/main.ts"
}
```

## Secrets Management

Never hardcode API keys. Use the `Secret` class:

```typescript
// Good
model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY"))

// Bad - never do this
model: Model.openAI("gpt-4", "sk-abc123...")
```

## What Can You Build?

- **Customer Support Bots** - Agents that understand context, access your knowledge base, and escalate when needed
- **Automated Workflows** - Schedule agents to process data, generate reports, or sync systems
- **Internal Tools** - AI-powered assistants that integrate with your existing stack
- **Content Pipelines** - Agents that create, review, and publish content

## License

MIT

## Links

- [Documentation](https://docs.shuttl.io)
- [GitHub](https://github.com/shuttl-io/shuttl_ai)
- [Website](https://www.shuttl.io)

