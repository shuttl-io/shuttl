# Scheduled Tasks Example

Build agents that run on a schedule to automate recurring tasks.

---

## What We're Building

A set of scheduled agents that:

- Generate daily reports at 9 AM
- Monitor systems every 15 minutes
- Send weekly summaries on Friday afternoons

---

## Use Cases

Scheduled agents are perfect for:

- **Reporting**: Daily/weekly summaries
- **Monitoring**: System health checks
- **Data Processing**: ETL pipelines
- **Notifications**: Digest emails
- **Maintenance**: Cleanup tasks

---

## Basic Scheduled Agent

The simplest scheduled agent:

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome } from "@shuttl-io/core";

export const heartbeatAgent = new Agent({
    name: "Heartbeat",
    systemPrompt: "Report that the system is healthy.",
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [],
    triggers: [Rate.hours(1)],
    outcomes: [new SlackOutcome("#monitoring")],
});
```

This agent runs every hour and posts a simple health check to Slack.

---

## Daily Report Agent

A more sophisticated agent that generates daily reports:

### Project Structure

```
daily-reporter/
├── src/
│   ├── agents/
│   │   └── reporter.ts
│   ├── tools/
│   │   ├── metrics.ts
│   │   └── database.ts
│   └── main.ts
├── shuttl.json
└── package.json
```

### Metrics Tool

Create `src/tools/metrics.ts`:

```typescript
import { Schema } from "@shuttl-io/core";

export const fetchMetricsTool = {
    name: "fetch_metrics",
    description: "Fetch business metrics for a date range",
    schema: Schema.objectValue({
        startDate: Schema.stringValue("Start date (YYYY-MM-DD)").isRequired(),
        endDate: Schema.stringValue("End date (YYYY-MM-DD)").isRequired(),
        metrics: Schema.stringValue("Comma-separated metric names"),
    }),
    execute: async ({ startDate, endDate, metrics }) => {
        // In production, fetch from your data warehouse
        // This is sample data for demonstration
        
        const metricList = metrics?.split(",").map(m => m.trim()) || [
            "revenue",
            "users",
            "orders",
            "churn",
        ];
        
        const results: Record<string, any> = {};
        
        for (const metric of metricList) {
            switch (metric) {
                case "revenue":
                    results.revenue = {
                        value: 45230.50,
                        previousPeriod: 42100.00,
                        change: "+7.4%",
                    };
                    break;
                case "users":
                    results.users = {
                        active: 1523,
                        new: 127,
                        churned: 23,
                    };
                    break;
                case "orders":
                    results.orders = {
                        total: 342,
                        avgValue: 132.25,
                        fulfillmentRate: "98.5%",
                    };
                    break;
                case "churn":
                    results.churn = {
                        rate: "1.5%",
                        atRisk: 45,
                    };
                    break;
            }
        }
        
        return {
            period: { startDate, endDate },
            metrics: results,
        };
    },
};

export const fetchTopProductsTool = {
    name: "fetch_top_products",
    description: "Get the top-selling products",
    schema: Schema.objectValue({
        limit: Schema.numberValue("Number of products").defaultTo(5),
        period: Schema.enumValue("Time period", ["day", "week", "month"]).defaultTo("day"),
    }),
    execute: async ({ limit, period }) => {
        // Sample data
        return {
            period,
            products: [
                { name: "Pro Plan Subscription", units: 45, revenue: 1305.00 },
                { name: "Enterprise Add-on", units: 12, revenue: 2388.00 },
                { name: "API Credits Pack", units: 89, revenue: 890.00 },
                { name: "Support Upgrade", units: 23, revenue: 575.00 },
                { name: "Training Session", units: 8, revenue: 1200.00 },
            ].slice(0, limit),
        };
    },
};
```

### Reporter Agent

Create `src/agents/reporter.ts`:

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome } from "@shuttl-io/core";
import { fetchMetricsTool, fetchTopProductsTool } from "../tools/metrics";

const REPORT_PROMPT = `You are a business analyst generating daily reports.

Your task:
1. Fetch yesterday's metrics using fetch_metrics
2. Get top products using fetch_top_products
3. Generate a concise, insightful report

Report format:
📊 **Daily Report - [Date]**

**Key Metrics**
• Revenue: $X (↑/↓ Y% vs previous)
• Active Users: X (New: Y, Churned: Z)
• Orders: X (Avg: $Y)

**Top Products**
1. Product A - X units ($Y)
2. Product B - X units ($Y)
...

**Insights**
- Notable trends or anomalies
- Brief recommendations

Keep it concise but actionable.`;

function getYesterdayDate(): string {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    return yesterday.toISOString().split("T")[0];
}

export const dailyReportAgent = new Agent({
    name: "DailyReporter",
    systemPrompt: REPORT_PROMPT,
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [fetchMetricsTool, fetchTopProductsTool],
    triggers: [
        Rate.cron("0 9 * * MON-FRI", "America/New_York")
            .withOnTrigger({
                onTrigger: async () => {
                    const yesterday = getYesterdayDate();
                    return [{
                        typeName: "text",
                        text: `Generate the daily report for ${yesterday}.`,
                    }];
                },
            })
            .bindOutcome(new SlackOutcome("#daily-reports")),
    ],
});
```

---

## Monitoring Agent

An agent that checks system health regularly:

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome, Schema } from "@shuttl-io/core";

const checkHealthTool = {
    name: "check_system_health",
    description: "Check health of various system components",
    schema: Schema.objectValue({
        components: Schema.stringValue("Comma-separated component names"),
    }),
    execute: async ({ components }) => {
        const componentList = components?.split(",").map(c => c.trim()) || [
            "api",
            "database",
            "cache",
            "queue",
        ];
        
        const results: Record<string, any> = {};
        
        for (const component of componentList) {
            // In production, actually check each component
            const isHealthy = Math.random() > 0.1; // 90% healthy for demo
            results[component] = {
                status: isHealthy ? "healthy" : "degraded",
                latency: Math.floor(Math.random() * 100) + 20,
                lastCheck: new Date().toISOString(),
            };
        }
        
        const unhealthy = Object.entries(results)
            .filter(([_, v]) => v.status !== "healthy")
            .map(([k, _]) => k);
        
        return {
            overallStatus: unhealthy.length === 0 ? "healthy" : "degraded",
            components: results,
            issues: unhealthy,
        };
    },
};

const alertTool = {
    name: "create_alert",
    description: "Create an alert for unhealthy components",
    schema: Schema.objectValue({
        component: Schema.stringValue("Component name").isRequired(),
        severity: Schema.enumValue("Alert severity", ["warning", "critical"]).isRequired(),
        message: Schema.stringValue("Alert message").isRequired(),
    }),
    execute: async ({ component, severity, message }) => {
        // In production, create a PagerDuty/OpsGenie alert
        console.log(`ALERT [${severity}]: ${component} - ${message}`);
        return { alertId: `ALT-${Date.now()}`, status: "created" };
    },
};

export const monitoringAgent = new Agent({
    name: "SystemMonitor",
    systemPrompt: `You monitor system health.
    
Every check:
1. Check all component health
2. If any component is unhealthy, create an alert
3. Report status summary

Only report issues - if everything is healthy, just confirm briefly.`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [checkHealthTool, alertTool],
    triggers: [
        Rate.minutes(15)
            .withOnTrigger({
                onTrigger: async () => [{
                    typeName: "text",
                    text: "Run a health check on all system components.",
                }],
            })
            .bindOutcome(new SlackOutcome("#ops-monitoring")),
    ],
});
```

---

## Weekly Summary Agent

Generate comprehensive weekly summaries:

```typescript
import { Agent, Model, Secret, Rate, SlackOutcome, CombinationOutcome } from "@shuttl-io/core";

export const weeklySummaryAgent = new Agent({
    name: "WeeklySummary",
    systemPrompt: `Generate a comprehensive weekly business summary.

Include:
1. Week-over-week metric comparisons
2. Top achievements
3. Areas needing attention
4. Recommendations for next week

Make it suitable for executive review - concise but insightful.`,
    model: Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [fetchMetricsTool, fetchTopProductsTool, fetchWeeklyGoalsTool],
    triggers: [
        Rate.cron("0 16 * * FRI", "America/New_York") // 4 PM on Fridays
            .withOnTrigger({
                onTrigger: async () => {
                    const endDate = new Date();
                    const startDate = new Date();
                    startDate.setDate(startDate.getDate() - 7);
                    
                    return [{
                        typeName: "text",
                        text: `Generate the weekly summary for ${startDate.toISOString().split("T")[0]} to ${endDate.toISOString().split("T")[0]}.`,
                    }];
                },
            })
            .bindOutcome(new CombinationOutcome([
                new SlackOutcome("#weekly-updates"),
                new SlackOutcome("#leadership"),
            ])),
    ],
});
```

---

## Combining Multiple Schedules

One agent, multiple schedules with different behaviors:

```typescript
export const flexibleAgent = new Agent({
    name: "FlexBot",
    systemPrompt: `You handle various scheduled tasks.
    Respond based on the input you receive.`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [fetchMetricsTool, checkHealthTool],
    triggers: [
        // Quick health check every 5 minutes
        Rate.minutes(5)
            .withOnTrigger({
                onTrigger: async () => [{
                    typeName: "text",
                    text: "Quick health check - only report if there are issues.",
                }],
            })
            .bindOutcome(new SlackOutcome("#alerts")),
        
        // Detailed report every hour
        Rate.hours(1)
            .withOnTrigger({
                onTrigger: async () => [{
                    typeName: "text",
                    text: "Generate hourly status report with key metrics.",
                }],
            })
            .bindOutcome(new SlackOutcome("#hourly-status")),
        
        // Still respond to API calls
        new ApiTrigger().bindOutcome(new StreamingOutcome()),
    ],
});
```

---

## Testing Scheduled Agents

### Local Testing

You can't easily wait for cron triggers, so test the agent logic directly:

```typescript
// test/reporter.test.ts
import { dailyReportAgent } from "../src/agents/reporter";

describe("DailyReportAgent", () => {
    it("generates a report with metrics", async () => {
        const toolCalls: string[] = [];
        
        const mockStreamer = {
            async receive(_, content) {
                if (content.data?.typeName === "tool_call") {
                    toolCalls.push(content.data.toolCall.name);
                }
            },
        };
        
        await dailyReportAgent.invoke(
            "Generate the daily report for 2025-01-02.",
            undefined,
            mockStreamer
        );
        
        expect(toolCalls).toContain("fetch_metrics");
    });
});
```

### Manual Trigger

Test using the TUI:

```bash
shuttl dev
```

1. Select `DailyReporter` from the **Agent Select** screen
2. Navigate to the **Chat** screen
3. Type: `Generate the daily report for 2025-01-02.`

Watch the **Agent Debug** screen to see the tool calls and report generation in real-time.

---

## Best Practices

### 1. Use Appropriate Intervals

```typescript
// ❌ Too frequent for expensive operations
Rate.seconds(30)  // Don't run GPT-4 every 30 seconds!

// ✅ Reasonable intervals
Rate.minutes(15)  // For monitoring
Rate.hours(1)     // For metrics
Rate.days(1)      // For reports
```

### 2. Handle Failures Gracefully

```typescript
const robustTool = {
    name: "fetch_data",
    execute: async (args) => {
        try {
            return await fetchFromApi(args);
        } catch (error) {
            // Return error info rather than throwing
            return {
                error: true,
                message: error.message,
                fallback: "Using cached data from last successful run.",
            };
        }
    },
};
```

### 3. Include Context in Triggers

```typescript
Rate.hours(1).withOnTrigger({
    onTrigger: async () => {
        const currentHour = new Date().getHours();
        const isBusinessHours = currentHour >= 9 && currentHour <= 17;
        
        return [{
            typeName: "text",
            text: isBusinessHours
                ? "Generate detailed report - business hours."
                : "Generate brief summary - after hours.",
        }];
    },
})
```

### 4. Use Different Outcomes for Different Urgencies

```typescript
triggers: [
    // Routine: post to general channel
    Rate.hours(1).bindOutcome(new SlackOutcome("#monitoring")),
    
    // Critical: alert on-call + create ticket
    // (handled in tool logic based on severity)
]
```

---

## Next Steps

- [:sunny: Weather Agent](weather-agent.md) - Conversational agent example
- [:zap: Triggers Guide](../concepts/triggers.md) - All trigger options
- [:outbox_tray: Outcomes Guide](../concepts/outcomes.md) - Routing responses

