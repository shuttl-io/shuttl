# Agents

An **Agent** is the core unit of intelligence in Shuttl. It combines a language model, system prompt, and tools to create an AI that can understand requests and take actions.

---

## Creating an Agent

```typescript
import { Agent, Model, Secret } from "@shuttl-io/core";

export const myAgent = new Agent({
    name: "MyAgent",
    systemPrompt: "You are a helpful assistant.",
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY")),
    tools: [],
    triggers: [],
    outcomes: [],
});
```

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | `string` | Unique identifier for your agent |
| `systemPrompt` | `string` | Instructions that define the agent's behavior |
| `model` | `IModelFactory` | The LLM to use (see [Models](models.md)) |

### Optional Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `tools` | `ITool[]` | `[]` | Tools the agent can use |
| `toolkits` | `Toolkit[]` | `[]` | Groups of related tools |
| `triggers` | `ITrigger[]` | `[ApiTrigger]` | How the agent is activated |
| `outcomes` | `IOutcome[]` | `[StreamingOutcome]` | Where responses are sent |

---

## The System Prompt

The system prompt is the most important part of your agent. It defines:

- **Identity**: Who is the agent?
- **Capabilities**: What can it do?
- **Behavior**: How should it respond?
- **Constraints**: What shouldn't it do?

### Best Practices

#### Be Specific

```typescript
// ❌ Too vague
systemPrompt: "You are helpful."

// ✅ Specific and actionable
systemPrompt: `You are a customer support agent for Acme Corp.

Your responsibilities:
- Answer questions about our products
- Help troubleshoot issues
- Escalate complex problems to humans

Guidelines:
- Be friendly but professional
- Always search the knowledge base before answering
- If unsure, say "I don't know" rather than guessing`
```

#### Include Tool Instructions

```typescript
systemPrompt: `You help users find restaurants.

Available tools:
- search_restaurants: Use this to find restaurants matching criteria
- get_reviews: Use this to get reviews for a specific restaurant

Always search for restaurants before making recommendations.
When the user asks for details, fetch reviews.`
```

#### Define Output Format

```typescript
systemPrompt: `You analyze sales data and provide insights.

Format your responses as:
1. Key Metric Summary (2-3 bullet points)
2. Notable Trends
3. Recommended Actions

Use specific numbers and percentages when available.`
```

---

## Multi-turn Conversations

Agents maintain conversation context through **threads**. Each thread represents a separate conversation.

```typescript
// First message creates a thread
const response1 = await agent.invoke("What's the weather in NYC?");
const threadId = response1.threadId;

// Continue the conversation
const response2 = await agent.invoke(
    "What about tomorrow?",
    threadId  // Pass the thread ID
);
```

### How It Works

```
Thread: abc123
├── User: "What's the weather in NYC?"
├── Agent: [calls weather_tool({location: "NYC"})]
├── Tool: {temp: 72, condition: "sunny"}
├── Agent: "It's 72°F and sunny in NYC!"
├── User: "What about tomorrow?"
├── Agent: [calls weather_tool({location: "NYC", day: "tomorrow"})]
├── Tool: {temp: 68, condition: "cloudy"}
└── Agent: "Tomorrow will be 68°F and cloudy."
```

The agent remembers the previous context (NYC) and applies it to the follow-up question.

---

## Invoking Agents

### Direct Invocation

```typescript
// Simple string prompt
await agent.invoke("Hello, how are you?");

// With thread ID for conversation continuity
await agent.invoke("Follow up question", threadId);

// With file attachments
await agent.invoke("Analyze this image", threadId, undefined, [
    { name: "photo.jpg", data: imageBuffer, mimeType: "image/jpeg" }
]);
```

### With Custom Streamer

Control how responses are handled:

```typescript
const customStreamer = {
    async receive(model, content) {
        if (content.data?.typeName === "output_text_delta") {
            console.log(content.data.text);
        }
    }
};

await agent.invoke("Hello", undefined, customStreamer);
```

---

## Tool Execution Flow

When an agent decides to use a tool:

```
1. LLM generates tool call with arguments
2. Shuttl validates arguments against schema
3. Tool's execute() function runs
4. Result is sent back to LLM
5. LLM incorporates result into response
6. Steps 1-5 may repeat for multiple tools
```

### Automatic Retries

Shuttl includes exponential backoff for transient failures:

```typescript
// Built-in retry logic
// Attempt 1: immediate
// Attempt 2: 500ms delay
// Attempt 3: 1000ms delay
// Attempt 4: 2000ms delay (max)
```

---

## Agent Patterns

### The Specialist

An agent focused on one domain:

```typescript
export const sqlAgent = new Agent({
    name: "SQLExpert",
    systemPrompt: `You are a SQL expert. You help users:
    - Write SQL queries
    - Optimize existing queries
    - Explain query execution plans
    
    Always validate queries before suggesting them.
    Never suggest destructive operations without confirmation.`,
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [executeSqlTool, explainQueryTool],
});
```

### The Coordinator

An agent that routes to specialists (coming soon):

```typescript
export const routerAgent = new Agent({
    name: "Router",
    systemPrompt: `Route user requests to the appropriate specialist:
    - SQL questions → SQLExpert
    - Code review → CodeReviewer
    - General questions → Helper`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY")),
    tools: [routeToAgent],
});
```

### The Background Worker

An agent that runs on a schedule:

```typescript
export const cleanupAgent = new Agent({
    name: "Cleanup",
    systemPrompt: `Perform daily maintenance tasks:
    1. Archive old records
    2. Generate summary reports
    3. Alert on anomalies`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY")),
    tools: [archiveTool, reportTool, alertTool],
    triggers: [Rate.days(1)],
    outcomes: [new SlackOutcome("#ops")],
});
```

---

## Error Handling

### Tool Errors

If a tool throws an error, the agent receives it and can decide how to proceed:

```typescript
const riskyTool = {
    name: "risky_operation",
    execute: async (args) => {
        if (args.invalid) {
            throw new Error("Invalid input provided");
        }
        return { success: true };
    },
};
```

The LLM sees the error and can:
- Try again with different arguments
- Ask the user for clarification
- Fall back to a different approach

### Model Errors

Shuttl handles rate limits and transient API errors automatically with retries.

---

## Best Practices

### 1. Keep Agents Focused

One agent, one job. Create multiple specialized agents rather than one that does everything.

### 2. Test Your System Prompts

The system prompt is code. Test it:

```typescript
describe("SupportAgent", () => {
    it("searches knowledge base before answering", async () => {
        const response = await agent.invoke("What are your prices?");
        expect(response.toolCalls).toContain("search_knowledge_base");
    });
});
```

### 3. Use Appropriate Models

- **GPT-4**: Complex reasoning, nuanced responses
- **GPT-4o-mini**: Fast, cost-effective for simpler tasks
- **GPT-4o**: Multimodal capabilities (images, audio)

### 4. Monitor Tool Usage

Track which tools are called and how often:

```typescript
execute: async (args) => {
    metrics.increment("tool.search.called");
    const start = Date.now();
    const result = await doSearch(args);
    metrics.timing("tool.search.duration", Date.now() - start);
    return result;
}
```

---

## Next Steps

- [:wrench: Learn about Tools](tools.md) - Give your agent capabilities
- [:zap: Add Triggers](triggers.md) - Control how agents are activated
- [:outbox_tray: Configure Outcomes](outcomes.md) - Route responses

