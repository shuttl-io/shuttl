# Shuttl

## Build AI Agents That Actually Work

Shuttl is a developer-first framework for building, deploying, and managing AI agents. Stop wrestling with complex infrastructure and start shipping intelligent agents in minutes.

---

<div class="grid cards" markdown>

-   :rocket: **Get Started in 5 Minutes**

    ---

    Install the CLI, define your agent, and deploy. It's that simple.

    [:octicons-arrow-right-24: Quick Start](getting-started/quickstart.md)

-   :brain: **Powerful Agent Primitives**

    ---

    Agents, Tools, Triggers, and Outcomes—composable building blocks for any AI workflow.

    [:octicons-arrow-right-24: Core Concepts](concepts/index.md)

-   :wrench: **Batteries Included**

    ---

    Built-in support for OpenAI, scheduled tasks, webhooks, and more. Extend with custom tools.

    [:octicons-arrow-right-24: Examples](examples/index.md)

-   :terminal: **Developer Experience First**

    ---

    Hot reload, streaming responses, and a TUI for real-time debugging. Build with confidence.

    [:octicons-arrow-right-24: CLI Reference](cli/index.md)

</div>

---

## Why Shuttl?

Building AI agents shouldn't require a PhD in distributed systems. Shuttl abstracts away the complexity so you can focus on what matters: **making your agents smart**.

### :zap: From Zero to Agent in Minutes

```typescript
import { Agent, Model, Secret, Schema } from "@shuttl-io/core";

const weatherTool = {
    name: "get_weather",
    description: "Get current weather for a location",
    schema: Schema.objectValue({
        location: Schema.stringValue("City name").isRequired(),
    }),
    execute: async (args) => {
        // Your logic here
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

That's it. Run `shuttl dev` and your agent is live.

### :repeat: Triggers for Every Use Case

Agents that respond to the world, not just API calls:

| Trigger | Description |
|---------|-------------|
| **API** | Expose your agent as an HTTP endpoint |
| **Rate** | Schedule agents with cron or intervals |
| **Email** | React to incoming emails |
| **File** | Watch for file system changes |

```typescript
import { Rate, StreamingOutcome } from "@shuttl-io/core";

// Run every hour
triggers: [Rate.hours(1).bindOutcome(new StreamingOutcome())]

// Or use cron for precise scheduling
triggers: [Rate.cron("0 9 * * MON-FRI", "America/New_York")]
```

### :package: Tools & Toolkits

Give your agents superpowers with reusable tools:

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

Group related tools into **Toolkits** for clean, modular agent design.

### :chart_with_upwards_trend: Production-Ready

- **Streaming responses** for real-time UX
- **Automatic retries** with exponential backoff
- **Thread management** for multi-turn conversations
- **Outcome routing** to Slack, webhooks, or custom destinations

---

## Quick Install

=== "npm"

    ```bash
    npm install @shuttl-io/core
    ```

=== "pnpm"

    ```bash
    pnpm add @shuttl-io/core
    ```

=== "yarn"

    ```bash
    yarn add @shuttl-io/core
    ```

Install the CLI:

```bash
# Download the latest release
curl -fsSL https://shuttl.dev/install.sh | bash

# Or build from source
git clone https://github.com/shuttl-io/shuttl
cd shuttl/apps/cli && go build -o shuttl
```

---

## What Can You Build?

<div class="grid cards" markdown>

-   **Customer Support Bots**

    ---

    Agents that understand context, access your knowledge base, and escalate when needed.

-   **Automated Workflows**

    ---

    Schedule agents to process data, generate reports, or sync systems on a cadence.

-   **Internal Tools**

    ---

    Give your team AI-powered assistants that integrate with your existing stack.

-   **Content Pipelines**

    ---

    Agents that create, review, and publish content across channels.

</div>

---

## Ready to Build?

<div class="grid cards" markdown>

-   [:rocket: **Get Started**](getting-started/quickstart.md)

    Jump into the quick start guide and deploy your first agent.

-   [:book: **Read the Docs**](concepts/index.md)

    Deep dive into Shuttl's architecture and capabilities.

-   [:star: **Star on GitHub**](https://github.com/shuttl-io)

    Show your support and stay updated on new releases.

</div>

