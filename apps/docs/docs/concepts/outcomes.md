# Outcomes

**Outcomes** define where agent responses go. Stream to users in real-time, post to Slack, call webhooks, or build custom integrations.

---

## Available Outcomes

| Outcome | Description | Use Case |
|---------|-------------|----------|
| `StreamingOutcome` | Real-time streaming | Chat interfaces, live feedback |
| `SlackOutcome` | Post to Slack | Notifications, reports |
| `CombinationOutcome` | Multiple destinations | Broadcast to several channels |

---

## Streaming Outcome

The default outcome. Streams agent responses in real-time via Server-Sent Events (SSE) or WebSocket.

```typescript
import { Agent, StreamingOutcome } from "@shuttl-io/core";

export const chatAgent = new Agent({
    name: "ChatBot",
    systemPrompt: "You're a helpful assistant.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    outcomes: [new StreamingOutcome()],  // This is the default
});
```

### Stream Events

When streaming, clients receive events like:

```json
{"type": "response.requested", "data": {...}}
{"type": "output_text_delta", "data": {"text": "Hello"}}
{"type": "output_text_delta", "data": {"text": "! How"}}
{"type": "output_text_delta", "data": {"text": " can I help?"}}
{"type": "tool_call", "data": {"name": "search", "arguments": {...}}}
{"type": "tool_calls_completed", "data": {...}}
{"type": "output_text", "data": {"text": "Based on the search results..."}}
{"type": "response.completed", "data": {...}}
{"type": "overall.completed", "data": {...}}
```

### Consuming Streams (JavaScript)

When using `shuttl serve` to expose HTTP endpoints, you can consume streams like this:

```typescript
const response = await fetch("http://localhost:8080/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello!" }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    
    const lines = decoder.decode(value).split("\n");
    for (const line of lines) {
        if (line.trim()) {
            const event = JSON.parse(line);
            if (event.type === "output_text_delta") {
                process.stdout.write(event.data.text);
            }
        }
    }
}
```

---

## Slack Outcome

Post agent responses to Slack channels.

```typescript
import { SlackOutcome } from "@shuttl-io/core";

export const reportAgent = new Agent({
    name: "Reporter",
    systemPrompt: "Generate concise reports.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    triggers: [Rate.days(1)],
    outcomes: [new SlackOutcome("#reports")],
});
```

### Configuration

```typescript
new SlackOutcome(
    "#channel-name",   // Channel name or ID
    {
        token: Secret.fromEnv("SLACK_BOT_TOKEN"),  // Bot token
        username: "Shuttl Bot",                     // Display name
        iconEmoji: ":robot_face:",                  // Custom icon
    }
)
```

### Message Formatting

The agent's text response is posted directly. For rich formatting, include markdown in the response:

```typescript
systemPrompt: `Generate reports with this format:

*Daily Report - ${date}*

📊 *Key Metrics*
• Users: {count}
• Revenue: ${amount}

📈 *Trends*
{analysis}

✅ *Recommendations*
{actions}`
```

---

## Combination Outcome

Send responses to multiple destinations simultaneously.

```typescript
import { CombinationOutcome, StreamingOutcome, SlackOutcome } from "@shuttl-io/core";

export const broadcastAgent = new Agent({
    name: "Broadcaster",
    systemPrompt: "Announce important updates.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    outcomes: [
        new CombinationOutcome([
            new StreamingOutcome(),           // Real-time to API caller
            new SlackOutcome("#announcements"), // Also post to Slack
        ]),
    ],
});
```

### Use Cases

```typescript
// Log + Notify
new CombinationOutcome([
    new StreamingOutcome(),
    new WebhookOutcome("https://logging.example.com/ingest"),
])

// Multi-channel notification
new CombinationOutcome([
    new SlackOutcome("#team-a"),
    new SlackOutcome("#team-b"),
    new EmailOutcome("stakeholders@company.com"),
])
```

---

## Binding Outcomes to Triggers

Different triggers can have different outcomes:

```typescript
export const flexAgent = new Agent({
    name: "FlexBot",
    systemPrompt: "Handle various requests.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    triggers: [
        // API calls stream back to the caller
        new ApiTrigger().bindOutcome(new StreamingOutcome()),
        
        // Scheduled runs post to Slack
        Rate.hours(1).bindOutcome(new SlackOutcome("#hourly-updates")),
        
        // Email triggers reply via email
        new EmailTrigger("bot@company.com").bindOutcome(new EmailOutcome()),
    ],
});
```

---

## Custom Outcomes

Build your own outcomes by implementing the `IOutcome` interface:

```typescript
import { IOutcome, IModelResponseStream } from "@shuttl-io/core";

interface IOutcome {
    send(messageStream: IModelResponseStream): Promise<void>;
    bindToRequest(request: any): Promise<void>;
}
```

The `IModelResponseStream` provides an async iterator over model responses:

```typescript
interface ModelResponseStreamValue {
    readonly value: ModelResponse | undefined;
    readonly done: boolean;
}

interface IModelResponseStream {
    next(): Promise<ModelResponseStreamValue>;
}
```

### Webhook Outcome Example

```typescript
import { IOutcome, IModelResponseStream, ModelResponse } from "@shuttl-io/core";

class WebhookOutcome implements IOutcome {
    private requestContext: any;
    
    constructor(private webhookUrl: string) {}
    
    async bindToRequest(request: any): Promise<void> {
        // Store any request context you need
        this.requestContext = request;
    }
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        const chunks: string[] = [];
        
        // Consume the stream
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            
            if (value?.data?.typeName === "output_text_delta") {
                chunks.push(value.data.text);
            }
        }
        
        // Send the complete response
        await fetch(this.webhookUrl, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                response: chunks.join(""),
                timestamp: new Date().toISOString(),
            }),
        });
    }
}

// Usage
outcomes: [new WebhookOutcome("https://api.example.com/webhook")]
```

### Database Logging Outcome

```typescript
class DatabaseLogOutcome implements IOutcome {
    constructor(private db: Database) {}
    
    async bindToRequest(request: any): Promise<void> {
        // No request context needed
    }
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        const events: ModelResponse[] = [];
        
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            if (value) events.push(value);
        }
        
        await this.db.agentLogs.insert({
            events,
            createdAt: new Date(),
        });
    }
}
```

### Real-time Forwarding Outcome

Forward events as they stream (don't wait for completion):

```typescript
class RealtimeForwardOutcome implements IOutcome {
    constructor(private websocket: WebSocket) {}
    
    async bindToRequest(request: any): Promise<void> {}
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            
            // Forward each event immediately
            if (value) {
                this.websocket.send(JSON.stringify(value));
            }
        }
    }
}
```

---

## Outcome Patterns

### The Audit Trail

Log all responses while still streaming:

```typescript
outcomes: [
    new CombinationOutcome([
        new StreamingOutcome(),
        new DatabaseLogOutcome(db),
    ]),
]
```

### The Notification Hub

Route based on content:

```typescript
class SmartRouterOutcome implements IOutcome {
    async bindToRequest(request: any): Promise<void> {}
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        const chunks: string[] = [];
        
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            
            if (value?.data?.typeName === "output_text_delta") {
                chunks.push(value.data.text);
            }
        }
        
        const response = chunks.join("");
        const isUrgent = response.includes("URGENT") || response.includes("ERROR");
        const isReport = response.includes("Report") || response.includes("Summary");
        
        if (isUrgent) {
            await slackClient.post("#alerts", response);
            await pagerDuty.alert(response);
        } else if (isReport) {
            await slackClient.post("#reports", response);
        } else {
            await slackClient.post("#general", response);
        }
    }
}
```

### The Transformer

Modify output before sending:

```typescript
class FormattedSlackOutcome implements IOutcome {
    constructor(private channel: string) {}
    
    async bindToRequest(request: any): Promise<void> {}
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        const chunks: string[] = [];
        
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            
            if (value?.data?.typeName === "output_text_delta") {
                chunks.push(value.data.text);
            }
        }
        
        const response = chunks.join("");
        
        // Add header and footer
        const formatted = `
🤖 *Agent Response*
───────────────
${response}
───────────────
_Generated at ${new Date().toLocaleTimeString()}_
        `.trim();
        
        await slackClient.post(this.channel, formatted);
    }
}
```

---

## Best Practices

### 1. Match Outcome to Use Case

| Scenario | Recommended Outcome |
|----------|---------------------|
| Interactive chat | `StreamingOutcome` |
| Scheduled reports | `SlackOutcome` |
| Background tasks | `WebhookOutcome` or `DatabaseLogOutcome` |
| Critical alerts | `CombinationOutcome` with multiple channels |

### 2. Handle Failures Gracefully

```typescript
class ResilientWebhookOutcome implements IOutcome {
    async bindToRequest(request: any): Promise<void> {}
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        // Collect the response first
        const chunks: string[] = [];
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            if (value?.data?.typeName === "output_text_delta") {
                chunks.push(value.data.text);
            }
        }
        const response = chunks.join("");
        
        // Retry with exponential backoff
        const maxRetries = 3;
        for (let attempt = 1; attempt <= maxRetries; attempt++) {
            try {
                await this.postToWebhook(response);
                return;
            } catch (error) {
                if (attempt === maxRetries) {
                    console.error("Webhook failed after retries:", error);
                    await this.fallbackToDatabase(response);
                }
                await this.sleep(1000 * attempt);
            }
        }
    }
}
```

### 3. Include Context in Outputs

```typescript
class ContextualSlackOutcome implements IOutcome {
    constructor(private channel: string) {}
    
    async bindToRequest(request: any): Promise<void> {}
    
    async send(messageStream: IModelResponseStream): Promise<void> {
        const chunks: string[] = [];
        while (true) {
            const { value, done } = await messageStream.next();
            if (done) break;
            if (value?.data?.typeName === "output_text_delta") {
                chunks.push(value.data.text);
            }
        }
        const response = chunks.join("");
        
        await slackClient.post(this.channel, {
            blocks: [
                {
                    type: "header",
                    text: { type: "plain_text", text: "Agent Report" },
                },
                {
                    type: "section",
                    text: { type: "mrkdwn", text: response },
                },
                {
                    type: "context",
                    elements: [
                        { type: "mrkdwn", text: `Generated: ${new Date().toISOString()}` },
                    ],
                },
            ],
        });
    }
}
```

---

## Next Steps

- [:robot_face: Learn about Models](models.md) - Configure LLM providers
- [:zap: Review Triggers](triggers.md) - Connect triggers to outcomes
- [:mag: See Examples](../examples/index.md) - Real-world outcome usage

