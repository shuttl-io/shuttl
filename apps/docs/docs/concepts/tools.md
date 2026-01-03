# Tools & Toolkits

**Tools** are functions that agents can call to interact with the world. They're how agents search databases, call APIs, process files, and take actions.

---

## Anatomy of a Tool

Every tool has three parts:

```typescript
const myTool = {
    // 1. Identity: Name and description for the LLM
    name: "get_weather",
    description: "Get current weather for a location",
    
    // 2. Schema: What arguments the tool accepts
    schema: Schema.objectValue({
        location: Schema.stringValue("City name").isRequired(),
        unit: Schema.enumValue("Temperature unit", ["celsius", "fahrenheit"])
            .defaultTo("celsius"),
    }),
    
    // 3. Implementation: What the tool actually does
    execute: async (args: Record<string, unknown>) => {
        const { location, unit } = args;
        const weather = await fetchWeather(location, unit);
        return weather;
    },
};
```

---

## The ITool Interface

```typescript
interface ITool {
    name: string;           // Unique identifier
    description: string;    // What the tool does (for LLM)
    schema?: Schema;        // Input validation
    execute(args: Record<string, unknown>): unknown;  // Implementation
}
```

### Tool Names

- Use `snake_case` for consistency
- Be descriptive: `search_products` not `search`
- Avoid generic names: `get_data` is too vague

### Descriptions

The description tells the LLM when to use this tool:

```typescript
// ❌ Too vague
description: "Search things"

// ✅ Clear and actionable
description: `Search the product catalog by name, category, or price range.
Use this when users ask about products, availability, or prices.
Returns matching products with name, price, and stock status.`
```

---

## Schema Definition

Schemas define what arguments your tool accepts and validate LLM outputs.

### Basic Types

```typescript
import { Schema } from "@shuttl-io/core";

Schema.stringValue("A text value")
Schema.numberValue("A numeric value")
Schema.booleanValue("A true/false value")
Schema.enumValue("Pick one", ["option1", "option2", "option3"])
```

### Modifiers

```typescript
// Required field
Schema.stringValue("User's email").isRequired()

// Default value
Schema.numberValue("Results limit").defaultTo(10)

// Combine them
Schema.enumValue("Priority", ["low", "medium", "high"])
    .isRequired()
    .defaultTo("medium")
```

### Object Schema

Combine fields into an object:

```typescript
schema: Schema.objectValue({
    query: Schema.stringValue("Search query").isRequired(),
    filters: Schema.objectValue({
        category: Schema.stringValue("Product category"),
        minPrice: Schema.numberValue("Minimum price"),
        maxPrice: Schema.numberValue("Maximum price"),
        inStock: Schema.booleanValue("Only show in-stock items").defaultTo(true),
    }),
    limit: Schema.numberValue("Max results").defaultTo(20),
})
```

---

## Implementing execute()

The `execute` function receives validated arguments and returns results.

### Basic Implementation

```typescript
execute: async (args: Record<string, unknown>) => {
    const query = args.query as string;
    const limit = args.limit as number;
    
    const results = await database.search(query, { limit });
    return results;
}
```

### Return Values

Return any JSON-serializable value:

```typescript
// Object
return { temperature: 72, condition: "sunny" };

// Array
return [{ id: 1, name: "Product A" }, { id: 2, name: "Product B" }];

// Primitive
return "Success!";

// Structured response with metadata
return {
    success: true,
    data: results,
    count: results.length,
    hasMore: results.length === limit,
};
```

### Error Handling

Throw errors for the LLM to handle:

```typescript
execute: async (args) => {
    const user = await db.findUser(args.userId);
    
    if (!user) {
        throw new Error(`User ${args.userId} not found`);
    }
    
    return user;
}
```

The LLM receives the error message and can:
- Try a different approach
- Ask the user for clarification
- Report the error appropriately

---

## Toolkits

**Toolkits** group related tools together for organization and reuse.

### Creating a Toolkit

```typescript
import { Toolkit } from "@shuttl-io/core";

const weatherToolkit = new Toolkit("weather", "Tools for weather information");

// Add tools to the toolkit
weatherToolkit.addTool({
    name: "get_current_weather",
    description: "Get current weather conditions",
    schema: Schema.objectValue({
        location: Schema.stringValue("City name").isRequired(),
    }),
    execute: async ({ location }) => {
        return await weatherApi.current(location);
    },
});

weatherToolkit.addTool({
    name: "get_forecast",
    description: "Get weather forecast for upcoming days",
    schema: Schema.objectValue({
        location: Schema.stringValue("City name").isRequired(),
        days: Schema.numberValue("Number of days").defaultTo(5),
    }),
    execute: async ({ location, days }) => {
        return await weatherApi.forecast(location, days);
    },
});
```

### Using Toolkits with Agents

```typescript
export const weatherAgent = new Agent({
    name: "WeatherBot",
    systemPrompt: "You help users with weather information.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    toolkits: [weatherToolkit],  // All tools in toolkit are available
});
```

### Combining Toolkits

```typescript
export const assistantAgent = new Agent({
    name: "Assistant",
    systemPrompt: "You're a helpful assistant.",
    model: Model.openAI("gpt-4", Secret.fromEnv("KEY")),
    toolkits: [weatherToolkit, calendarToolkit, emailToolkit],
    tools: [customTool],  // Can also add individual tools
});
```

---

## Tool Patterns

### API Wrapper

Wrap an external API:

```typescript
const slackTool = {
    name: "send_slack_message",
    description: "Send a message to a Slack channel",
    schema: Schema.objectValue({
        channel: Schema.stringValue("Channel name or ID").isRequired(),
        message: Schema.stringValue("Message content").isRequired(),
        thread_ts: Schema.stringValue("Thread timestamp for replies"),
    }),
    execute: async ({ channel, message, thread_ts }) => {
        const response = await slack.chat.postMessage({
            channel,
            text: message,
            thread_ts,
        });
        return { 
            success: true, 
            messageId: response.ts,
            channel: response.channel,
        };
    },
};
```

### Database Query

Query your database safely:

```typescript
const searchUsersTool = {
    name: "search_users",
    description: "Search users by name or email",
    schema: Schema.objectValue({
        query: Schema.stringValue("Search term").isRequired(),
        role: Schema.enumValue("Filter by role", ["admin", "user", "guest"]),
        limit: Schema.numberValue("Max results").defaultTo(10),
    }),
    execute: async ({ query, role, limit }) => {
        let queryBuilder = db.users
            .where("name", "ilike", `%${query}%`)
            .orWhere("email", "ilike", `%${query}%`)
            .limit(limit);
        
        if (role) {
            queryBuilder = queryBuilder.andWhere("role", role);
        }
        
        const users = await queryBuilder.select("id", "name", "email", "role");
        return { users, count: users.length };
    },
};
```

### File Operations

Work with files:

```typescript
const readFileTool = {
    name: "read_file",
    description: "Read contents of a file",
    schema: Schema.objectValue({
        path: Schema.stringValue("File path").isRequired(),
        encoding: Schema.enumValue("File encoding", ["utf8", "base64"])
            .defaultTo("utf8"),
    }),
    execute: async ({ path, encoding }) => {
        // Validate path is within allowed directory
        const safePath = resolveSafePath(path);
        const content = await fs.readFile(safePath, encoding);
        return { content, path: safePath };
    },
};
```

### Multi-step Operations

Combine multiple operations:

```typescript
const createOrderTool = {
    name: "create_order",
    description: "Create a new order with items",
    schema: Schema.objectValue({
        customerId: Schema.stringValue("Customer ID").isRequired(),
        items: Schema.stringValue("JSON array of {productId, quantity}").isRequired(),
    }),
    execute: async ({ customerId, items }) => {
        const parsedItems = JSON.parse(items);
        
        // 1. Validate customer
        const customer = await db.customers.find(customerId);
        if (!customer) throw new Error("Customer not found");
        
        // 2. Check inventory
        for (const item of parsedItems) {
            const product = await db.products.find(item.productId);
            if (product.stock < item.quantity) {
                throw new Error(`Insufficient stock for ${product.name}`);
            }
        }
        
        // 3. Create order
        const order = await db.orders.create({
            customerId,
            items: parsedItems,
            status: "pending",
        });
        
        // 4. Update inventory
        for (const item of parsedItems) {
            await db.products.decrement(item.productId, "stock", item.quantity);
        }
        
        return { orderId: order.id, status: "created" };
    },
};
```

---

## Best Practices

### 1. Keep Tools Focused

One tool, one action:

```typescript
// ❌ Too many responsibilities
const userTool = {
    name: "manage_user",
    execute: async ({ action, userId, data }) => {
        if (action === "create") { ... }
        if (action === "update") { ... }
        if (action === "delete") { ... }
    }
};

// ✅ Separate, focused tools
const createUserTool = { name: "create_user", ... };
const updateUserTool = { name: "update_user", ... };
const deleteUserTool = { name: "delete_user", ... };
```

### 2. Return Useful Information

Help the LLM understand what happened:

```typescript
// ❌ Minimal response
return true;

// ✅ Rich response
return {
    success: true,
    orderId: "ORD-123",
    itemCount: 3,
    totalAmount: 149.99,
    estimatedDelivery: "2025-01-10",
};
```

### 3. Handle Errors Gracefully

Provide actionable error messages:

```typescript
execute: async ({ email }) => {
    const user = await db.users.findByEmail(email);
    
    if (!user) {
        // ❌ Generic error
        throw new Error("Not found");
        
        // ✅ Helpful error
        throw new Error(
            `No user found with email "${email}". ` +
            `Try searching by name instead.`
        );
    }
    
    return user;
}
```

### 4. Validate Dangerous Operations

Add confirmation for destructive actions:

```typescript
const deleteAccountTool = {
    name: "delete_account",
    description: "Permanently delete a user account. Requires confirmation.",
    schema: Schema.objectValue({
        userId: Schema.stringValue("User ID").isRequired(),
        confirmPhrase: Schema.stringValue(
            "Must be exactly 'DELETE MY ACCOUNT'"
        ).isRequired(),
    }),
    execute: async ({ userId, confirmPhrase }) => {
        if (confirmPhrase !== "DELETE MY ACCOUNT") {
            throw new Error(
                "Confirmation phrase doesn't match. " +
                "User must type exactly: DELETE MY ACCOUNT"
            );
        }
        
        await db.users.delete(userId);
        return { deleted: true, userId };
    },
};
```

---

## Testing Tools

Test your tools in isolation:

```typescript
import { describe, it, expect } from "vitest";
import { searchTool } from "./tools/search";

describe("searchTool", () => {
    it("returns matching results", async () => {
        const result = await searchTool.execute({
            query: "pricing",
            limit: 5,
        });
        
        expect(result.found).toBe(true);
        expect(result.articles.length).toBeGreaterThan(0);
    });
    
    it("handles no matches gracefully", async () => {
        const result = await searchTool.execute({
            query: "xyznonexistent123",
            limit: 5,
        });
        
        expect(result.found).toBe(false);
        expect(result.suggestion).toBeDefined();
    });
});
```

---

## Next Steps

- [:zap: Add Triggers](triggers.md) - Control when your agent runs
- [:outbox_tray: Configure Outcomes](outcomes.md) - Route responses
- [:mag: See Examples](../examples/index.md) - Real-world tool implementations

