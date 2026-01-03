# Your First Agent

Let's build a complete, useful AI agent from scratch. By the end of this guide, you'll have a working agent that can search a knowledge base and answer questions.

---

## What We're Building

A **Support Agent** that can:

- Answer questions using a knowledge base
- Handle multi-turn conversations
- Gracefully handle unknown queries

This is a realistic starting point for customer support bots, internal tools, and documentation assistants.

---

## Project Structure

```
my-support-agent/
├── src/
│   ├── main.ts          # Entry point
│   ├── agent.ts         # Agent definition
│   └── tools/
│       └── search.ts    # Knowledge base tool
├── knowledge/
│   └── articles.json    # Sample knowledge base
├── shuttl.json
├── package.json
└── tsconfig.json
```

---

## Step 1: Set Up the Project

```bash
mkdir my-support-agent && cd my-support-agent
npm init -y
npm install @shuttl-io/core typescript ts-node @types/node
npx tsc --init
mkdir -p src/tools knowledge
```

---

## Step 2: Create a Knowledge Base

Create `knowledge/articles.json`:

```json
{
    "articles": [
        {
            "id": "pricing-001",
            "title": "Pricing Plans",
            "content": "We offer three plans: Starter ($9/mo), Pro ($29/mo), and Enterprise (custom). All plans include unlimited agents. Pro adds priority support. Enterprise includes dedicated infrastructure.",
            "tags": ["pricing", "plans", "billing"]
        },
        {
            "id": "getting-started-001",
            "title": "Getting Started",
            "content": "To get started: 1) Install the SDK with npm install @shuttl-io/core, 2) Create an agent file, 3) Run shuttl dev to start the development server. Check our Quick Start guide for detailed steps.",
            "tags": ["setup", "installation", "quickstart"]
        },
        {
            "id": "api-limits-001",
            "title": "API Rate Limits",
            "content": "Starter plan: 1000 requests/day. Pro plan: 10,000 requests/day. Enterprise: unlimited. Rate limits reset at midnight UTC. Contact support to request limit increases.",
            "tags": ["api", "limits", "quotas"]
        },
        {
            "id": "support-001",
            "title": "Getting Help",
            "content": "For support: Starter users can use our community Discord. Pro users get email support with 24-hour response time. Enterprise users have dedicated Slack channels and phone support.",
            "tags": ["support", "help", "contact"]
        }
    ]
}
```

---

## Step 3: Build the Search Tool

Create `src/tools/search.ts`:

```typescript
import { Schema } from "@shuttl-io/core";
import * as fs from "fs";
import * as path from "path";

// Types for our knowledge base
interface Article {
    id: string;
    title: string;
    content: string;
    tags: string[];
}

interface KnowledgeBase {
    articles: Article[];
}

// Load the knowledge base
const loadKnowledgeBase = (): KnowledgeBase => {
    const filePath = path.join(process.cwd(), "knowledge", "articles.json");
    const data = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(data);
};

// Simple search implementation
const searchArticles = (query: string, limit: number): Article[] => {
    const kb = loadKnowledgeBase();
    const queryLower = query.toLowerCase();
    const queryTerms = queryLower.split(/\s+/);

    // Score each article based on matches
    const scored = kb.articles.map((article) => {
        let score = 0;
        const searchText = `${article.title} ${article.content} ${article.tags.join(" ")}`.toLowerCase();

        for (const term of queryTerms) {
            if (searchText.includes(term)) {
                score += 1;
            }
            // Boost for tag matches
            if (article.tags.some((tag) => tag.includes(term))) {
                score += 2;
            }
            // Boost for title matches
            if (article.title.toLowerCase().includes(term)) {
                score += 3;
            }
        }

        return { article, score };
    });

    // Sort by score and return top results
    return scored
        .filter((s) => s.score > 0)
        .sort((a, b) => b.score - a.score)
        .slice(0, limit)
        .map((s) => s.article);
};

// Export the tool definition
export const searchTool = {
    name: "search_knowledge_base",
    description: `Search the knowledge base for articles related to a query. 
        Use this to find information about pricing, features, setup, and support.
        Returns matching articles with their titles and content.`,
    schema: Schema.objectValue({
        query: Schema.stringValue(
            "The search query - use keywords related to the user's question"
        ).isRequired(),
        limit: Schema.numberValue(
            "Maximum number of results to return"
        ).defaultTo(3),
    }),
    execute: async (args: Record<string, unknown>) => {
        const query = args.query as string;
        const limit = (args.limit as number) || 3;

        const results = searchArticles(query, limit);

        if (results.length === 0) {
            return {
                found: false,
                message: "No relevant articles found",
                suggestion: "Try different keywords or ask the user for clarification",
            };
        }

        return {
            found: true,
            count: results.length,
            articles: results.map((a) => ({
                title: a.title,
                content: a.content,
            })),
        };
    },
};
```

---

## Step 4: Define the Agent

Create `src/agent.ts`:

```typescript
import { Agent, Model, Secret } from "@shuttl-io/core";
import { searchTool } from "./tools/search";

const SYSTEM_PROMPT = `You are a helpful support agent for Shuttl, an AI agent framework.

Your responsibilities:
1. Answer user questions accurately using the knowledge base
2. Be friendly and professional
3. If you can't find an answer, be honest and suggest alternatives

Guidelines:
- Always search the knowledge base before answering questions about pricing, features, or setup
- Provide concise but complete answers
- Include relevant details like specific numbers or steps
- If the user's question is unclear, ask for clarification

Remember: You represent Shuttl. Be helpful, accurate, and friendly.`;

export const supportAgent = new Agent({
    name: "SupportAgent",
    systemPrompt: SYSTEM_PROMPT,
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [searchTool],
});
```

---

## Step 5: Create the Entry Point

Create `src/main.ts`:

```typescript
// Export all agents for Shuttl to discover
export { supportAgent } from "./agent";
```

---

## Step 6: Configure Shuttl

Create `shuttl.json`:

```json
{
    "app": "node --require ts-node/register ./src/main.ts"
}
```

---

## Step 7: Run Your Agent

Set your API key and start the development TUI:

```bash
export OPENAI_API_KEY="sk-your-key-here"
shuttl dev
```

---

## Step 8: Test It Out

The TUI launches with four screens. Use the **Agent Select** screen to choose `SupportAgent`, then navigate to the **Chat** screen.

### Ask about pricing:

Type in the chat:

```
What pricing plans do you offer?
```

**Expected behavior:** Watch the **Agent Debug** screen as the agent searches the knowledge base, finds the pricing article, and responds with plan details.

### Ask a follow-up:

```
What are the API limits for Pro?
```

**Expected behavior:** The agent remembers the conversation context and provides specific Pro plan limits.

### Ask something not in the knowledge base:

```
Do you support Claude models?
```

**Expected behavior:** The agent searches, finds no results, and honestly says it doesn't have that information.

Use the **Agent Debug** screen to watch the full execution flow—tool calls, LLM responses, and final output—in real-time.

---

## Understanding the Flow

```
User Message
     ↓
┌─────────────┐
│   Agent     │ ← System prompt defines behavior
└─────────────┘
     ↓
┌─────────────┐
│    LLM      │ ← Decides to use search tool
└─────────────┘
     ↓
┌─────────────┐
│ Search Tool │ ← Executes with args from LLM
└─────────────┘
     ↓
┌─────────────┐
│    LLM      │ ← Synthesizes response from results
└─────────────┘
     ↓
Final Response
```

---

## Improving Your Agent

Now that you have a working agent, here are ways to make it better:

### Add More Tools

```typescript
const escalateTool = {
    name: "escalate_to_human",
    description: "Escalate complex issues to human support",
    schema: Schema.objectValue({
        reason: Schema.stringValue("Why this needs human attention").isRequired(),
        priority: Schema.enumValue("Urgency level", ["low", "medium", "high"]),
    }),
    execute: async (args) => {
        // In production: create support ticket, notify team, etc.
        return { ticketId: "SUP-" + Date.now(), status: "escalated" };
    },
};
```

### Add Triggers

Make your agent respond to schedules or webhooks:

```typescript
import { Rate, ApiTrigger } from "@shuttl-io/core";

export const supportAgent = new Agent({
    // ... existing config
    triggers: [
        new ApiTrigger(), // HTTP endpoint (default)
        Rate.hours(1),    // Also run every hour for proactive tasks
    ],
});
```

### Use Better Search

Replace the simple search with a vector database for semantic search:

```typescript
// With a vector DB like Pinecone or Weaviate
execute: async ({ query }) => {
    const embedding = await openai.embeddings.create({ input: query });
    const results = await vectorDB.query(embedding, { topK: 5 });
    return results;
};
```

---

## Next Steps

<div class="grid cards" markdown>

-   **:brain: Deep Dive into Agents**

    ---

    Learn advanced agent patterns and best practices.

    [:octicons-arrow-right-24: Agents Guide](../concepts/agents.md)

-   **:wrench: Master Tools**

    ---

    Build powerful tools and toolkits.

    [:octicons-arrow-right-24: Tools Guide](../concepts/tools.md)

-   **:zap: Add Triggers**

    ---

    Connect your agent to schedules and events.

    [:octicons-arrow-right-24: Triggers Guide](../concepts/triggers.md)

</div>

