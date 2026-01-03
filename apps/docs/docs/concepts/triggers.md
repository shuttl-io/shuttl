# Triggers

**Triggers** define how and when your agents are activated. Beyond simple API calls, agents can respond to schedules, emails, file changes, and more.

---

## Available Triggers

| Trigger | Description | Use Case |
|---------|-------------|----------|
| `ApiTrigger` | HTTP endpoint | Chat interfaces, webhooks |
| `Rate` | Schedule-based | Cron jobs, periodic tasks |
| `EmailTrigger` | Incoming emails | Email automation |
| `FileTrigger` | File system changes | Document processing |

---

## API Trigger

The default trigger. Exposes your agent as an HTTP endpoint.

```typescript
import { Agent, ApiTrigger } from "@shuttl-io/core";

export const chatAgent = new Agent({
    name: "ChatBot",
    systemPrompt: "You're a helpful assistant.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    triggers: [new ApiTrigger()],  // This is the default
});
```

### Endpoints

When you run `shuttl serve`, these HTTP endpoints are created:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/chat` | Send a message |
| `POST` | `/chat/:threadId` | Continue a conversation |
| `GET` | `/agents` | List available agents |
| `GET` | `/health` | Health check |

!!! tip "Development vs Production"
    Use `shuttl dev` for interactive development with the TUI. Use `shuttl serve` to expose HTTP endpoints for production or integration testing.

### Request Format

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello!",
    "agentName": "ChatBot"
  }'
```

With attachments:

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: multipart/form-data" \
  -F "message=Analyze this image" \
  -F "file=@photo.jpg"
```

---

## Rate Trigger

Schedule agents to run at specific intervals or times.

### Simple Intervals

```typescript
import { Rate, StreamingOutcome } from "@shuttl-io/core";

// Every 5 minutes
triggers: [Rate.minutes(5).bindOutcome(new StreamingOutcome())]

// Every hour
triggers: [Rate.hours(1).bindOutcome(new SlackOutcome("#channel"))]

// Every day
triggers: [Rate.days(1).bindOutcome(new WebhookOutcome("https://..."))]
```

### Available Interval Methods

```typescript
Rate.milliseconds(500)  // Every 500ms
Rate.seconds(30)        // Every 30 seconds
Rate.minutes(15)        // Every 15 minutes
Rate.hours(6)           // Every 6 hours
Rate.days(1)            // Every day
Rate.weeks(1)           // Every week
Rate.months(1)          // Every month (30 days)
```

### Cron Expressions

For precise scheduling, use cron syntax:

```typescript
// 9 AM every weekday
Rate.cron("0 9 * * MON-FRI", "America/New_York")

// Every hour on the hour
Rate.cron("0 * * * *")

// First day of every month at midnight
Rate.cron("0 0 1 * *")

// Every 15 minutes during business hours
Rate.cron("*/15 9-17 * * MON-FRI", "America/New_York")
```

### Cron Syntax Reference

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, SUN-SAT)
│ │ │ │ │
* * * * *
```

### Custom Input on Trigger

Customize what the agent receives when triggered:

```typescript
Rate.hours(1)
    .withOnTrigger({
        onTrigger: async () => {
            const stats = await fetchDailyStats();
            return [{
                typeName: "text",
                text: `Generate a report for: ${JSON.stringify(stats)}`,
            }];
        },
    })
    .bindOutcome(new SlackOutcome("#reports"))
```

---

## Email Trigger

React to incoming emails.

```typescript
import { EmailTrigger, EmailOutcome } from "@shuttl-io/core";

export const emailAgent = new Agent({
    name: "EmailResponder",
    systemPrompt: `You handle customer support emails.
    
    For each email:
    1. Understand the customer's issue
    2. Search the knowledge base for solutions
    3. Draft a helpful response`,
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [searchKnowledgeTool, draftEmailTool],
    triggers: [
        new EmailTrigger({
            address: "support@company.com",
            filters: {
                excludeSpam: true,
                subjectContains: ["help", "support", "issue"],
            },
        }),
    ],
    outcomes: [new EmailOutcome()],
});
```

### Email Input Format

When triggered by an email, the agent receives:

```typescript
{
    from: "customer@example.com",
    to: "support@company.com",
    subject: "Help with my order",
    body: "I placed an order yesterday but...",
    attachments: [/* any attached files */],
    receivedAt: "2025-01-03T10:30:00Z",
}
```

---

## File Trigger

Watch for file system changes.

```typescript
import { FileTrigger } from "@shuttl-io/core";

export const documentProcessor = new Agent({
    name: "DocProcessor",
    systemPrompt: "Process incoming documents and extract key information.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [extractTextTool, classifyDocTool, saveToDbTool],
    triggers: [
        new FileTrigger({
            path: "/incoming/documents",
            patterns: ["*.pdf", "*.docx"],
            events: ["created"],
        }),
    ],
});
```

### Configuration Options

```typescript
new FileTrigger({
    path: "/path/to/watch",      // Directory to monitor
    patterns: ["*.pdf"],          // Glob patterns to match
    events: ["created", "modified", "deleted"],  // Events to trigger on
    recursive: true,              // Watch subdirectories
    debounceMs: 1000,            // Wait for file to stabilize
})
```

---

## Multiple Triggers

Agents can have multiple triggers:

```typescript
export const flexibleAgent = new Agent({
    name: "FlexBot",
    systemPrompt: "You respond to various inputs.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [searchTool],
    triggers: [
        new ApiTrigger(),                    // HTTP endpoint
        Rate.hours(1),                       // Hourly check-in
        new EmailTrigger("bot@company.com"), // Email responses
    ],
});
```

Each trigger can have its own outcome:

```typescript
triggers: [
    new ApiTrigger().bindOutcome(new StreamingOutcome()),
    Rate.hours(1).bindOutcome(new SlackOutcome("#updates")),
]
```

---

## Binding Outcomes

Connect triggers to specific outcomes:

```typescript
// Rate trigger → Slack
Rate.hours(1).bindOutcome(new SlackOutcome("#channel"))

// Rate trigger → Webhook
Rate.minutes(30).bindOutcome(new WebhookOutcome("https://api.example.com/hook"))

// Rate trigger → Multiple outcomes
Rate.days(1).bindOutcome(
    new CombinationOutcome([
        new SlackOutcome("#reports"),
        new WebhookOutcome("https://backup.example.com"),
    ])
)
```

---

## Trigger Patterns

### The Poller

Check for updates periodically:

```typescript
const pollerAgent = new Agent({
    name: "DataPoller",
    systemPrompt: "Check for new data and process it.",
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("KEY")),
    tools: [checkForUpdatesTool, processUpdateTool],
    triggers: [
        Rate.minutes(5).withOnTrigger({
            onTrigger: async () => [{
                typeName: "text",
                text: "Check for new data and process any updates.",
            }],
        }),
    ],
});
```

### The Reporter

Generate scheduled reports:

```typescript
const reporterAgent = new Agent({
    name: "DailyReporter",
    systemPrompt: "Generate comprehensive daily reports.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [fetchMetricsTool, generateChartTool],
    triggers: [
        Rate.cron("0 9 * * MON-FRI", "America/New_York")
            .withOnTrigger({
                onTrigger: async () => {
                    const date = new Date().toISOString().split("T")[0];
                    return [{
                        typeName: "text",
                        text: `Generate the daily report for ${date}.`,
                    }];
                },
            })
            .bindOutcome(new SlackOutcome("#reports")),
    ],
});
```

### The Hybrid

Respond to both API calls and schedules:

```typescript
const hybridAgent = new Agent({
    name: "HybridBot",
    systemPrompt: `You perform analysis tasks.
    
    If invoked via API: Respond to the user's specific request.
    If invoked on schedule: Generate the standard daily summary.`,
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    tools: [analyzeTool, summarizeTool],
    triggers: [
        new ApiTrigger(),
        Rate.cron("0 18 * * *").bindOutcome(new SlackOutcome("#eod")),
    ],
});
```

---

## Custom Triggers

Build your own triggers by implementing the `ITrigger` interface or extending `BaseTrigger`.

### The ITrigger Interface

```typescript
import { ITrigger, IOutcome, ITriggerInvoker } from "@shuttl-io/core";

interface ITrigger {
    /** Unique name of this trigger instance */
    name: string;
    
    /** The type of trigger (e.g., "webhook", "queue") */
    triggerType: string;
    
    /** Configuration for the trigger */
    triggerConfig: Record<string, unknown>;
    
    /** Optional bound outcome */
    outcome?: IOutcome;
    
    /** Activates the trigger and invokes the agent */
    activate(args: any, invoker: ITriggerInvoker): Promise<void>;
    
    /** Validates trigger arguments (optional) */
    validate?(args: any): Promise<Record<string, unknown>>;
    
    /** Binds an outcome to this trigger */
    bindOutcome(outcome: IOutcome): ITrigger;
    
    /** Sets the trigger name */
    withName(name: string): ITrigger;
}
```

### Extending BaseTrigger (Recommended)

The `BaseTrigger` abstract class handles most boilerplate:

```typescript
import { BaseTrigger, TriggerOutput, ITriggerInvoker, IOutcome } from "@shuttl-io/core";

class WebhookTrigger extends BaseTrigger {
    constructor(private webhookSecret: string) {
        super("webhook", { secret: webhookSecret });
    }
    
    // Parse incoming webhook payload into agent input
    async parseArgs(args: any): Promise<TriggerOutput> {
        const { event, data } = args;
        
        return {
            input: [{
                typeName: "text",
                text: `Webhook event: ${event}\nData: ${JSON.stringify(data)}`,
            }],
        };
    }
    
    // Optional: validate the webhook signature
    async validate(args: any): Promise<Record<string, unknown>> {
        const signature = args.headers?.["x-webhook-signature"];
        if (!this.verifySignature(signature, args.body)) {
            throw new Error("Invalid webhook signature");
        }
        return {};
    }
    
    private verifySignature(signature: string, body: any): boolean {
        // Your signature verification logic
        return true;
    }
}
```

### TriggerOutput Format

The `parseArgs` method returns a `TriggerOutput` with the input for the agent:

```typescript
interface TriggerOutput {
    input: InputContent[];
}

// InputContent can be text or file attachments
type InputContent = 
    | { typeName: "text"; text: string }
    | { typeName: "file"; file: string; fileData: FileAttachment };
```

### Example: Queue Trigger

A trigger that processes messages from a queue:

```typescript
class QueueTrigger extends BaseTrigger {
    constructor(private queueUrl: string) {
        super("queue", { queueUrl });
    }
    
    async parseArgs(args: any): Promise<TriggerOutput> {
        const message = args.message;
        const metadata = args.metadata || {};
        
        return {
            input: [{
                typeName: "text",
                text: `Process this queue message:
                
Message ID: ${metadata.messageId}
Received: ${metadata.timestamp}
Content: ${JSON.stringify(message)}`,
            }],
        };
    }
}

// Usage
export const queueAgent = new Agent({
    name: "QueueProcessor",
    systemPrompt: "Process incoming queue messages.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    triggers: [
        new QueueTrigger("https://sqs.us-east-1.amazonaws.com/queue")
            .withName("order-queue")
            .bindOutcome(new SlackOutcome("#orders")),
    ],
});
```

### Example: GitHub Webhook Trigger

```typescript
class GitHubWebhookTrigger extends BaseTrigger {
    constructor(private webhookSecret: string) {
        super("github-webhook", { secret: webhookSecret });
    }
    
    async parseArgs(args: any): Promise<TriggerOutput> {
        const event = args.headers["x-github-event"];
        const payload = args.body;
        
        let prompt: string;
        
        switch (event) {
            case "pull_request":
                prompt = `Review this PR:
                    Title: ${payload.pull_request.title}
                    Author: ${payload.pull_request.user.login}
                    Description: ${payload.pull_request.body}
                    Changes: ${payload.pull_request.diff_url}`;
                break;
            case "issues":
                prompt = `Triage this issue:
                    Title: ${payload.issue.title}
                    Body: ${payload.issue.body}
                    Labels: ${payload.issue.labels.map(l => l.name).join(", ")}`;
                break;
            default:
                prompt = `GitHub ${event} event received: ${JSON.stringify(payload)}`;
        }
        
        return {
            input: [{ typeName: "text", text: prompt }],
        };
    }
    
    async validate(args: any): Promise<Record<string, unknown>> {
        const signature = args.headers["x-hub-signature-256"];
        if (!this.verifyGitHubSignature(signature, args.rawBody)) {
            throw new Error("Invalid GitHub webhook signature");
        }
        return { event: args.headers["x-github-event"] };
    }
}
```

### The ITriggerInvoker

When `activate` is called, you receive an `ITriggerInvoker` that invokes the agent:

```typescript
interface ITriggerInvoker {
    invoke(input: InputContent[]): Promise<IModelResponseStream>;
    defaultOutcome(response: IModelResponseStream): Promise<void>;
}
```

If you need full control over activation, override `activate` directly:

```typescript
class CustomTrigger extends BaseTrigger {
    async parseArgs(args: any): Promise<TriggerOutput> {
        return { input: [{ typeName: "text", text: args.message }] };
    }
    
    // Override for custom activation logic
    async activate(args: any, invoker: ITriggerInvoker): Promise<void> {
        // Pre-processing
        console.log("Trigger activated:", this.name);
        
        const { input } = await this.parseArgs(args);
        const response = await invoker.invoke(input);
        
        // Route to outcome or default
        if (this.outcome) {
            await this.outcome.send(response);
        } else {
            await invoker.defaultOutcome(response);
        }
        
        // Post-processing
        console.log("Trigger completed:", this.name);
    }
}
```

---

## Best Practices

### 1. Use Appropriate Intervals

Don't poll more often than necessary:

```typescript
// ❌ Too aggressive
Rate.seconds(1)

// ✅ Reasonable for most use cases
Rate.minutes(5)
```

### 2. Handle Trigger Context

Use `onTrigger` to provide context:

```typescript
Rate.hours(1).withOnTrigger({
    onTrigger: async () => {
        const context = await gatherContext();
        return [{
            typeName: "text",
            text: `Current context: ${JSON.stringify(context)}. 
                   Analyze and report any anomalies.`,
        }];
    },
})
```

### 3. Match Outcomes to Triggers

API triggers usually stream; scheduled triggers usually notify:

```typescript
// API → Stream to user
new ApiTrigger().bindOutcome(new StreamingOutcome())

// Schedule → Post to Slack
Rate.days(1).bindOutcome(new SlackOutcome("#channel"))
```

### 4. Consider Timezones

Always specify timezone for cron expressions:

```typescript
// ❌ Ambiguous
Rate.cron("0 9 * * *")

// ✅ Explicit
Rate.cron("0 9 * * *", "America/New_York")
```

---

## Next Steps

- [:outbox_tray: Configure Outcomes](outcomes.md) - Route trigger responses
- [:robot_face: Learn about Models](models.md) - LLM configuration
- [:mag: See Examples](../examples/scheduled-tasks.md) - Real scheduled agents

