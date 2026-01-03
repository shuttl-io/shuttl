# Core Concepts

Shuttl is built around four composable primitives that work together to create powerful AI agents.

---

## The Shuttl Architecture

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

---

## The Four Primitives

<div class="grid cards" markdown>

-   :brain: **Agents**

    ---

    The core unit of intelligence. An agent combines a language model with a system prompt and tools to perform tasks.

    [:octicons-arrow-right-24: Learn more](agents.md)

-   :wrench: **Tools & Toolkits**

    ---

    Give agents the ability to take actions: search databases, call APIs, process data, and more.

    [:octicons-arrow-right-24: Learn more](tools.md)

-   :zap: **Triggers**

    ---

    Define how and when agents are activated: HTTP requests, schedules, emails, file changes.

    [:octicons-arrow-right-24: Learn more](triggers.md)

-   :outbox_tray: **Outcomes**

    ---

    Route agent responses to destinations: stream to users, post to Slack, call webhooks.

    [:octicons-arrow-right-24: Learn more](outcomes.md)

</div>

---

## How They Work Together

### Example: A Scheduled Reporting Agent

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome } from "@shuttl-io/core";

const reportTool = {
    name: "generate_report",
    description: "Generate a daily metrics report",
    schema: Schema.objectValue({
        date: Schema.stringValue("Report date in YYYY-MM-DD format"),
    }),
    execute: async ({ date }) => {
        // Fetch metrics from your database
        return { users: 1523, revenue: 42000, churn: 0.02 };
    },
};

export const reportingAgent = new Agent({
    name: "DailyReporter",
    systemPrompt: `Generate concise daily reports. Format as bullet points.`,
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

**What happens:**

1. **Trigger** fires at 9 AM EST every day
2. **Agent** receives the trigger and uses its tools
3. **Tool** fetches metrics and returns data
4. **Agent** formats the response using GPT-4
5. **Outcome** posts the report to Slack

---

## Design Principles

### Composition Over Configuration

Instead of complex YAML configs, Shuttl uses code. Combine primitives like Lego blocks:

```typescript
// One agent, multiple triggers, one outcome
triggers: [new ApiTrigger(), Rate.hours(1)]

// One trigger, multiple agents (coming soon)
// Combine outcomes
outcomes: [new StreamingOutcome(), new SlackOutcome("#logs")]
```

### Type Safety

Full TypeScript support means your IDE catches errors before runtime:

```typescript
// Tool schema defines what the LLM can pass
schema: Schema.objectValue({
    location: Schema.stringValue("City name").isRequired(),
    //        ^^^^^^^^^^^^^ IDE knows this is a string builder
})
```

### Secrets Management

Never hardcode API keys:

```typescript
// Bad
model: Model.openAI("gpt-4", "sk-abc123")

// Good
model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY"))
```

### Observable by Default

Every tool call, LLM response, and outcome is streamed in real-time, making debugging straightforward.

---

## Common Patterns

### The Helper Agent

A simple agent that answers questions:

```typescript
new Agent({
    name: "Helper",
    systemPrompt: "You answer questions helpfully.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [searchTool],
    // Uses default ApiTrigger and StreamingOutcome
});
```

### The Worker Agent

A scheduled agent that performs background tasks:

```typescript
new Agent({
    name: "Janitor",
    systemPrompt: "Clean up stale data and generate reports.",
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY")),
    tools: [cleanupTool, reportTool],
    triggers: [Rate.hours(6)],
    outcomes: [new SlackOutcome("#ops")],
});
```

### The Reactive Agent

An agent that responds to external events:

```typescript
new Agent({
    name: "EmailResponder",
    systemPrompt: "Draft responses to customer emails.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [emailTool, crmTool],
    triggers: [new EmailTrigger("support@company.com")],
    outcomes: [new EmailOutcome()],
});
```

---

## Next Steps

Dive deeper into each primitive:

1. [:brain: Agents](agents.md) - The intelligence layer
2. [:wrench: Tools](tools.md) - The action layer
3. [:zap: Triggers](triggers.md) - The activation layer
4. [:outbox_tray: Outcomes](outcomes.md) - The output layer
5. [:robot_face: Models](models.md) - LLM configuration

