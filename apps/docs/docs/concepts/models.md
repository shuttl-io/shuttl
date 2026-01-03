# Models

**Models** are the language models (LLMs) that power your agents' intelligence. Shuttl provides a unified interface across providers.

---

## Supported Providers

| Provider | Models | Status |
|----------|--------|--------|
| OpenAI | GPT-4, GPT-4o, GPT-4o-mini, GPT-3.5 | ✅ Supported |
| Anthropic | Claude 3, Claude 3.5 | 🚧 Coming Soon |
| Google | Gemini Pro, Gemini Ultra | 🚧 Coming Soon |
| Local | Ollama, LM Studio | 🚧 Coming Soon |

---

## OpenAI Models

### Basic Usage

```typescript
import { Model, Secret } from "@shuttl-io/core";

const model = Model.openAI(
    "gpt-4",                          // Model name
    Secret.fromEnv("OPENAI_API_KEY")  // API key
);

export const myAgent = new Agent({
    name: "MyAgent",
    systemPrompt: "You're helpful.",
    model: model,
    tools: [],
});
```

### Available Models

| Model | Best For | Context | Cost |
|-------|----------|---------|------|
| `gpt-4` | Complex reasoning, nuanced tasks | 8K | $$$ |
| `gpt-4-turbo` | Long context, code generation | 128K | $$ |
| `gpt-4o` | Multimodal (images, audio) | 128K | $$ |
| `gpt-4o-mini` | Fast, cost-effective tasks | 128K | $ |
| `gpt-3.5-turbo` | Simple tasks, high volume | 16K | $ |

### Model Selection Guide

```typescript
// Complex analysis, important decisions
Model.openAI("gpt-4", Secret.fromEnv("KEY"))

// General purpose, good balance
Model.openAI("gpt-4o", Secret.fromEnv("KEY"))

// High volume, simple tasks
Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY"))

// Image/document analysis
Model.openAI("gpt-4o", Secret.fromEnv("KEY"))  // Supports vision
```

---

## Secrets Management

Never hardcode API keys. Use the `Secret` class for secure key management.

### From Environment Variables

```typescript
// Reads from process.env.OPENAI_API_KEY
Secret.fromEnv("OPENAI_API_KEY")
```

### Setting Environment Variables

=== "Bash/Zsh"

    ```bash
    export OPENAI_API_KEY="sk-..."
    ```

=== ".env file"

    ```bash
    # .env
    OPENAI_API_KEY=sk-...
    ```
    
    Then use `dotenv`:
    
    ```typescript
    import "dotenv/config";
    ```

=== "Production"

    Use your cloud provider's secrets manager:
    
    - AWS Secrets Manager
    - GCP Secret Manager  
    - Azure Key Vault
    - HashiCorp Vault

---

## Model Configuration

### Model Factory Pattern

The `Model.openAI()` function returns an `IModelFactory` that creates model instances per conversation:

```typescript
const modelFactory = Model.openAI("gpt-4", Secret.fromEnv("KEY"));

// Each agent invocation gets a fresh model instance
// with its own conversation thread
```

### Configuration Options

```typescript
Model.openAI("gpt-4", Secret.fromEnv("KEY"), {
    temperature: 0.7,      // Creativity (0-2, default 1)
    maxTokens: 4096,       // Max response length
    topP: 0.9,            // Nucleus sampling
    frequencyPenalty: 0,   // Reduce repetition
    presencePenalty: 0,    // Encourage new topics
});
```

### Temperature Guide

| Temperature | Behavior | Use Case |
|-------------|----------|----------|
| 0.0 | Deterministic, focused | Code, math, facts |
| 0.3 | Slightly creative | Business writing |
| 0.7 | Balanced | General conversation |
| 1.0 | Creative | Brainstorming, stories |
| 1.5+ | Very random | Experimental |

```typescript
// Precise, deterministic responses
Model.openAI("gpt-4", key, { temperature: 0 })

// Creative content generation
Model.openAI("gpt-4", key, { temperature: 0.9 })
```

---

## Multi-turn Conversations

Models maintain conversation state through threads:

```typescript
// First message creates a thread
const model = await agent.invoke("Hello!");
const threadId = model.threadId;

// Subsequent messages use the same thread
await agent.invoke("What did I just say?", threadId);
// The model remembers the conversation context
```

### Thread Lifecycle

```
Thread Created (invoke without threadId)
    ↓
Messages Accumulate (invoke with threadId)
    ↓
Thread Expires (provider-dependent, usually 30-60 minutes of inactivity)
```

---

## Streaming Responses

Models support streaming for real-time UI updates:

```typescript
// The agent handles streaming automatically
await agent.invoke("Write a long story", threadId, customStreamer);
```

### Stream Event Types

| Event | Description |
|-------|-------------|
| `response.requested` | Model is processing |
| `output_text_delta` | Partial text chunk |
| `output_text` | Complete text segment |
| `tool_call` | Model wants to use a tool |
| `tool_calls_completed` | Tool execution finished |
| `response.completed` | Single response done |
| `overall.completed` | All processing complete |

---

## Tool Integration

Models automatically receive tool definitions:

```typescript
const agent = new Agent({
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [searchTool, calculateTool],
});

// Under the hood, tools are converted to OpenAI function format:
// {
//   type: "function",
//   function: {
//     name: "search",
//     description: "Search the database",
//     parameters: { type: "object", properties: {...} }
//   }
// }
```

---

## Best Practices

### 1. Choose the Right Model

```typescript
// ❌ Using GPT-4 for simple classification
Model.openAI("gpt-4", key)  // Expensive, overkill

// ✅ Use GPT-4o-mini for simple tasks
Model.openAI("gpt-4o-mini", key)  // Fast, cheap, sufficient
```

### 2. Handle Rate Limits

Shuttl includes automatic retry with exponential backoff:

```typescript
// Built-in retry logic handles:
// - 429 Rate Limit Exceeded
// - 500 Server Errors
// - Network timeouts
```

### 3. Monitor Usage

Track token usage in production:

```typescript
const analyticsStreamer = {
    async receive(model, content) {
        if (content.eventName === "response.completed") {
            analytics.track("llm_usage", {
                model: "gpt-4",
                inputTokens: content.usage?.input_tokens,
                outputTokens: content.usage?.output_tokens,
            });
        }
    }
};
```

### 4. Use System Prompts Effectively

The system prompt significantly impacts model behavior:

```typescript
// ❌ Vague
systemPrompt: "Be helpful"

// ✅ Specific and structured
systemPrompt: `You are a customer support agent for TechCorp.

ROLE: Answer product questions and troubleshoot issues.

GUIDELINES:
- Search the knowledge base before answering
- Be concise but thorough
- Escalate billing issues to humans

FORMAT: Use bullet points for lists. Keep responses under 200 words.`
```

---

## Choosing Between Models

### Decision Tree

```
Is the task complex reasoning or analysis?
├── Yes → GPT-4 or GPT-4-turbo
└── No → Does it involve images/audio?
    ├── Yes → GPT-4o
    └── No → Is latency critical?
        ├── Yes → GPT-4o-mini
        └── No → Is cost critical?
            ├── Yes → GPT-4o-mini or GPT-3.5-turbo
            └── No → GPT-4o (good balance)
```

### Cost Optimization

```typescript
// Tier 1: Premium (complex analysis)
const analysisModel = Model.openAI("gpt-4", key);

// Tier 2: Standard (general tasks)
const standardModel = Model.openAI("gpt-4o", key);

// Tier 3: Economy (high volume, simple tasks)
const economyModel = Model.openAI("gpt-4o-mini", key);
```

---

## Custom Models

Integrate any LLM provider by implementing the model interfaces.

### Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐
│  IModelFactory  │────▶│     IModel      │
│    (creates)    │     │   (instance)    │
└─────────────────┘     └─────────────────┘
        │                       │
        ▼                       ▼
   Passed to Agent         Handles invoke()
```

You create two things:

1. **IModel** - The model instance that handles conversations
2. **IModelFactory** - A factory that creates model instances

### The IModel Interface

```typescript
interface IModel {
    /** Thread ID for conversation continuity */
    readonly threadId?: string;
    
    /** Invoke the model with messages and stream responses */
    invoke(
        prompt: (ModelContent | ToolCallResponse)[],
        streamer: IModelStreamer
    ): Promise<void>;
}
```

### The IModelFactory Interface

```typescript
interface IModelFactory {
    create(props: IModelFactoryProps): Promise<IModel>;
}

interface IModelFactoryProps {
    readonly systemPrompt: string;
    readonly tools?: ITool[];
    readonly configuration?: Record<string, unknown>;
}
```

### The IModelStreamer Interface

Your model must emit events through the streamer:

```typescript
interface IModelStreamer {
    recieve(model: IModel, content: ModelResponse): Promise<void>;
}

interface ModelResponse {
    readonly eventName: string;
    readonly data: ModelResponseData[] | ModelResponseData;
    readonly usage?: Usage;
    readonly threadId?: string;
}

interface ModelResponseData {
    readonly typeName: 
        | "response.requested" 
        | "tool_call" 
        | "output_text" 
        | "output_text_delta" 
        | "response.completed"
        | "output_text.part.done"
        | "overall.completed";
    readonly toolCall?: ModelToolOutput;
    readonly outputText?: ModelTextOutput;
    readonly outputTextDelta?: ModelDeltaOutput;
    readonly threadId?: string;
}
```

### Example: Anthropic Claude Model

```typescript
import Anthropic from "@anthropic-ai/sdk";
import { 
    IModel, 
    IModelFactory, 
    IModelFactoryProps,
    IModelStreamer,
    ModelContent,
    ToolCallResponse,
    ITool 
} from "@shuttl-io/core";

// The Model class
class ClaudeModel implements IModel {
    public threadId?: string;
    private messages: any[] = [];
    
    constructor(
        private client: Anthropic,
        private modelName: string,
        private systemPrompt: string,
        private tools: ITool[]
    ) {
        this.threadId = `claude-${Date.now()}`;
    }
    
    async invoke(
        prompt: (ModelContent | ToolCallResponse)[],
        streamer: IModelStreamer
    ): Promise<void> {
        // Convert to Anthropic message format
        for (const msg of prompt) {
            if ("role" in msg) {
                this.messages.push({
                    role: msg.role === "user" ? "user" : "assistant",
                    content: typeof msg.content === "string" 
                        ? msg.content 
                        : this.formatContent(msg.content),
                });
            }
        }
        
        // Emit request started
        await streamer.recieve(this, {
            eventName: "response.requested",
            data: { typeName: "response.requested" },
        });
        
        // Stream from Claude
        const stream = await this.client.messages.stream({
            model: this.modelName,
            max_tokens: 4096,
            system: this.systemPrompt,
            messages: this.messages,
            tools: this.convertTools(),
        });
        
        for await (const event of stream) {
            if (event.type === "content_block_delta") {
                if (event.delta.type === "text_delta") {
                    await streamer.recieve(this, {
                        eventName: "output_text_delta",
                        data: {
                            typeName: "output_text_delta",
                            outputTextDelta: {
                                outputType: "output_text_delta",
                                delta: event.delta.text,
                                sequenceNumber: 0,
                            },
                        },
                    });
                }
            }
            
            if (event.type === "message_stop") {
                await streamer.recieve(this, {
                    eventName: "response.completed",
                    data: { typeName: "response.completed" },
                });
            }
        }
        
        // Emit overall complete
        await streamer.recieve(this, {
            eventName: "overall.completed",
            data: { typeName: "overall.completed", threadId: this.threadId },
        });
    }
    
    private convertTools(): any[] {
        return this.tools.map(tool => ({
            name: tool.name,
            description: tool.description,
            input_schema: this.schemaToJsonSchema(tool.schema),
        }));
    }
    
    private schemaToJsonSchema(schema: any): any {
        // Convert Shuttl schema to JSON Schema
        // Implementation depends on your schema format
        return { type: "object", properties: {} };
    }
    
    private formatContent(content: any): string {
        if (typeof content === "string") return content;
        if (Array.isArray(content)) {
            return content.map(c => c.text || "").join("");
        }
        return content.text || "";
    }
}

// The Factory class
class ClaudeModelFactory implements IModelFactory {
    private client: Anthropic;
    
    constructor(
        private modelName: string,
        apiKey: string
    ) {
        this.client = new Anthropic({ apiKey });
    }
    
    async create(props: IModelFactoryProps): Promise<IModel> {
        return new ClaudeModel(
            this.client,
            this.modelName,
            props.systemPrompt,
            props.tools || []
        );
    }
}

// Helper function (like Model.openAI)
function claude(modelName: string, apiKey: string): IModelFactory {
    return new ClaudeModelFactory(modelName, apiKey);
}

// Usage
export const myAgent = new Agent({
    name: "ClaudeAgent",
    systemPrompt: "You are a helpful assistant.",
    model: claude("claude-3-opus-20240229", process.env.ANTHROPIC_KEY!),
    tools: [searchTool],
});
```

### Example: Local Ollama Model

```typescript
class OllamaModel implements IModel {
    public threadId?: string;
    private history: any[] = [];
    
    constructor(
        private baseUrl: string,
        private modelName: string,
        private systemPrompt: string
    ) {
        this.threadId = `ollama-${Date.now()}`;
    }
    
    async invoke(
        prompt: (ModelContent | ToolCallResponse)[],
        streamer: IModelStreamer
    ): Promise<void> {
        // Build messages
        const messages = [
            { role: "system", content: this.systemPrompt },
            ...this.history,
        ];
        
        for (const msg of prompt) {
            if ("role" in msg && "content" in msg) {
                const content = typeof msg.content === "string" 
                    ? msg.content 
                    : JSON.stringify(msg.content);
                messages.push({ role: msg.role, content });
                this.history.push({ role: msg.role, content });
            }
        }
        
        // Stream from Ollama
        const response = await fetch(`${this.baseUrl}/api/chat`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                model: this.modelName,
                messages,
                stream: true,
            }),
        });
        
        const reader = response.body!.getReader();
        const decoder = new TextDecoder();
        
        await streamer.recieve(this, {
            eventName: "response.requested",
            data: { typeName: "response.requested" },
        });
        
        let fullResponse = "";
        
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            
            const chunk = decoder.decode(value);
            const lines = chunk.split("\n").filter(l => l.trim());
            
            for (const line of lines) {
                const data = JSON.parse(line);
                if (data.message?.content) {
                    fullResponse += data.message.content;
                    
                    await streamer.recieve(this, {
                        eventName: "output_text_delta",
                        data: {
                            typeName: "output_text_delta",
                            outputTextDelta: {
                                outputType: "output_text_delta",
                                delta: data.message.content,
                                sequenceNumber: 0,
                            },
                        },
                    });
                }
            }
        }
        
        // Store assistant response in history
        this.history.push({ role: "assistant", content: fullResponse });
        
        await streamer.recieve(this, {
            eventName: "response.completed",
            data: { typeName: "response.completed" },
        });
        
        await streamer.recieve(this, {
            eventName: "overall.completed",
            data: { typeName: "overall.completed", threadId: this.threadId },
        });
    }
}

class OllamaModelFactory implements IModelFactory {
    constructor(
        private baseUrl: string,
        private modelName: string
    ) {}
    
    async create(props: IModelFactoryProps): Promise<IModel> {
        return new OllamaModel(
            this.baseUrl,
            this.modelName,
            props.systemPrompt
        );
    }
}

// Helper function
function ollama(modelName: string, baseUrl = "http://localhost:11434"): IModelFactory {
    return new OllamaModelFactory(baseUrl, modelName);
}

// Usage
export const localAgent = new Agent({
    name: "LocalAgent",
    systemPrompt: "You are a helpful assistant running locally.",
    model: ollama("llama3"),
    tools: [],
});
```

### Key Events to Emit

Your model must emit these events through the streamer:

| Event | When | Required |
|-------|------|----------|
| `response.requested` | Processing started | Yes |
| `output_text_delta` | Each text chunk | For streaming |
| `tool_call` | Model wants to use a tool | If tools used |
| `response.completed` | Response finished | Yes |
| `overall.completed` | All processing done | Yes |

### Tool Call Format

When the model wants to call a tool:

```typescript
await streamer.recieve(this, {
    eventName: "tool_call",
    data: {
        typeName: "tool_call",
        toolCall: {
            outputType: "tool_call",
            name: "search",
            arguments: { query: "hello world" },
            callId: "call_123",
        },
    },
});
```

Shuttl will execute the tool and pass the result back via `invoke()` as a `ToolCallResponse`.

---

## Next Steps

- [:brain: Create Agents](agents.md) - Use models in agents
- [:wrench: Add Tools](tools.md) - Give models capabilities
- [:mag: Examples](../examples/index.md) - See models in action

