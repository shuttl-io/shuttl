# Examples

Learn by example. These guides show real-world Shuttl agents you can build and adapt.

---

## Quick Examples

### Minimal Chat Agent

The simplest possible agent—great for testing:

```typescript
import { Agent, Model, Secret } from "@shuttl-io/core";

export const chatBot = new Agent({
    name: "ChatBot",
    systemPrompt: "You are a helpful assistant. Be concise.",
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [],
});
```

### Agent with Tools

Add capabilities with tools:

```typescript
import { Agent, Model, Secret, Schema } from "@shuttl-io/core";

const calculateTool = {
    name: "calculate",
    description: "Perform mathematical calculations",
    schema: Schema.objectValue({
        expression: Schema.stringValue("Math expression like '2 + 2'").isRequired(),
    }),
    execute: async ({ expression }) => {
        const result = eval(expression); // Use a proper math parser in production!
        return { result };
    },
};

export const mathBot = new Agent({
    name: "MathBot",
    systemPrompt: "You help with math. Use the calculate tool for computations.",
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [calculateTool],
});
```

### Scheduled Agent

Run on a schedule instead of API calls:

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome } from "@shuttl-io/core";

export const dailyDigest = new Agent({
    name: "DailyDigest",
    systemPrompt: "Summarize the day's key metrics.",
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [fetchMetricsTool],
    triggers: [Rate.cron("0 9 * * *", "America/New_York")],
    outcomes: [new SlackOutcome("#daily-digest")],
});
```

---

## Featured Examples

<div class="grid cards" markdown>

-   :sunny: **Weather Agent**

    ---

    A conversational weather bot with API integration.

    [:octicons-arrow-right-24: View example](weather-agent.md)

-   :clock: **Scheduled Tasks**

    ---

    Background agents that run on a schedule.

    [:octicons-arrow-right-24: View example](scheduled-tasks.md)

</div>

---

## Example Patterns

### Customer Support Bot

```typescript
const supportAgent = new Agent({
    name: "SupportBot",
    systemPrompt: `You are a customer support agent.
    
    Guidelines:
    - Search the knowledge base before answering
    - Be friendly but professional
    - Escalate complex issues to humans`,
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [
        searchKnowledgeBaseTool,
        createTicketTool,
        escalateTool,
    ],
});
```

### Document Analyzer

```typescript
const docAnalyzer = new Agent({
    name: "DocAnalyzer",
    systemPrompt: `Analyze documents and extract key information.
    
    For each document:
    1. Identify document type
    2. Extract key entities
    3. Summarize main points`,
    model: Model.openAI("gpt-4o", Secret.fromEnv("KEY")), // Vision model
    tools: [
        extractTextTool,
        classifyDocTool,
        extractEntitiesTool,
    ],
});
```

### Code Review Assistant

```typescript
const codeReviewer = new Agent({
    name: "CodeReviewer",
    systemPrompt: `Review code for:
    - Bugs and potential issues
    - Performance concerns
    - Style and best practices
    
    Be constructive and specific.`,
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [
        analyzeCodeTool,
        searchDocsTool,
        suggestFixTool,
    ],
});
```

### Data Pipeline Agent

```typescript
const dataPipeline = new Agent({
    name: "DataPipeline",
    systemPrompt: `Process incoming data:
    1. Validate format
    2. Clean and normalize
    3. Store in database
    4. Generate summary report`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY")),
    tools: [
        validateDataTool,
        transformDataTool,
        saveToDbTool,
        generateReportTool,
    ],
    triggers: [Rate.minutes(15)],
    outcomes: [new SlackOutcome("#data-ops")],
});
```

---

## Tool Recipes

### Database Query Tool

```typescript
const queryDbTool = {
    name: "query_database",
    description: "Query the database with SQL",
    schema: Schema.objectValue({
        query: Schema.stringValue("SQL SELECT query").isRequired(),
    }),
    execute: async ({ query }) => {
        // Validate it's a SELECT query
        if (!query.trim().toUpperCase().startsWith("SELECT")) {
            throw new Error("Only SELECT queries allowed");
        }
        
        const results = await db.query(query);
        return { 
            rows: results.rows,
            count: results.rowCount,
        };
    },
};
```

### HTTP API Tool

```typescript
const apiCallTool = {
    name: "call_api",
    description: "Make an HTTP API request",
    schema: Schema.objectValue({
        url: Schema.stringValue("API endpoint URL").isRequired(),
        method: Schema.enumValue("HTTP method", ["GET", "POST"]).defaultTo("GET"),
        body: Schema.stringValue("Request body (JSON)"),
    }),
    execute: async ({ url, method, body }) => {
        const response = await fetch(url, {
            method,
            headers: { "Content-Type": "application/json" },
            body: body ? JSON.stringify(JSON.parse(body)) : undefined,
        });
        
        return {
            status: response.status,
            data: await response.json(),
        };
    },
};
```

### File Operations Tool

```typescript
const readFileTool = {
    name: "read_file",
    description: "Read a file from the workspace",
    schema: Schema.objectValue({
        path: Schema.stringValue("File path").isRequired(),
    }),
    execute: async ({ path }) => {
        const safePath = sanitizePath(path); // Prevent directory traversal
        const content = await fs.readFile(safePath, "utf-8");
        return { content, path: safePath };
    },
};
```

### Slack Notification Tool

```typescript
const notifySlackTool = {
    name: "notify_slack",
    description: "Send a message to Slack",
    schema: Schema.objectValue({
        channel: Schema.stringValue("Channel name").isRequired(),
        message: Schema.stringValue("Message content").isRequired(),
        urgency: Schema.enumValue("Priority", ["low", "medium", "high"]).defaultTo("low"),
    }),
    execute: async ({ channel, message, urgency }) => {
        const emoji = { low: "ℹ️", medium: "⚠️", high: "🚨" };
        
        await slack.chat.postMessage({
            channel,
            text: `${emoji[urgency]} ${message}`,
        });
        
        return { sent: true, channel };
    },
};
```

---

## Complete Project Structure

Here's a recommended structure for larger Shuttl projects:

```
my-shuttl-project/
├── src/
│   ├── agents/
│   │   ├── support.ts
│   │   ├── analytics.ts
│   │   └── index.ts
│   ├── tools/
│   │   ├── database.ts
│   │   ├── slack.ts
│   │   ├── email.ts
│   │   └── index.ts
│   ├── toolkits/
│   │   ├── crm.ts
│   │   └── reporting.ts
│   └── main.ts
├── knowledge/
│   └── articles.json
├── shuttl.json
├── package.json
└── tsconfig.json
```

### src/main.ts

```typescript
// Export all agents for Shuttl to discover
export * from "./agents";
```

### src/agents/index.ts

```typescript
export { supportAgent } from "./support";
export { analyticsAgent } from "./analytics";
```

---

## Testing Agents

### Unit Testing Tools

```typescript
import { describe, it, expect } from "vitest";
import { searchTool } from "../src/tools/search";

describe("searchTool", () => {
    it("returns results for valid queries", async () => {
        const result = await searchTool.execute({ query: "pricing" });
        expect(result.found).toBe(true);
    });
    
    it("handles empty results gracefully", async () => {
        const result = await searchTool.execute({ query: "nonexistent" });
        expect(result.found).toBe(false);
        expect(result.suggestion).toBeDefined();
    });
});
```

### Integration Testing Agents

```typescript
import { describe, it, expect } from "vitest";
import { supportAgent } from "../src/agents/support";

describe("supportAgent", () => {
    it("uses search tool for product questions", async () => {
        const toolCalls: string[] = [];
        
        const mockStreamer = {
            async receive(_, content) {
                if (content.data?.typeName === "tool_call") {
                    toolCalls.push(content.data.toolCall.name);
                }
            },
        };
        
        await supportAgent.invoke(
            "What are your prices?",
            undefined,
            mockStreamer
        );
        
        expect(toolCalls).toContain("search_knowledge_base");
    });
});
```

---

## Next Steps

- [:sunny: Weather Agent Example](weather-agent.md) - Full walkthrough
- [:clock: Scheduled Tasks Example](scheduled-tasks.md) - Background agents
- [:book: Core Concepts](../concepts/index.md) - Deep dive

