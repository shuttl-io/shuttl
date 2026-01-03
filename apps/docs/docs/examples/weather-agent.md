# Weather Agent Example

Build a conversational weather agent that can fetch current conditions and forecasts.

---

## What We're Building

A weather assistant that can:

- Get current weather for any location
- Provide multi-day forecasts
- Handle natural language queries
- Maintain conversation context

---

## Prerequisites

- Node.js 18+
- OpenAI API key
- OpenWeatherMap API key (free tier works)

---

## Project Setup

```bash
mkdir weather-agent && cd weather-agent
npm init -y
npm install @shuttl-io/core typescript ts-node @types/node
npx tsc --init
mkdir src
```

---

## Step 1: Create Weather Tools

Create `src/tools/weather.ts`:

```typescript
import { Schema } from "@shuttl-io/core";

const WEATHER_API_KEY = process.env.OPENWEATHER_API_KEY;
const BASE_URL = "https://api.openweathermap.org/data/2.5";

// Helper to fetch from OpenWeatherMap
async function fetchWeather(endpoint: string, params: Record<string, string>) {
    const url = new URL(`${BASE_URL}/${endpoint}`);
    url.searchParams.set("appid", WEATHER_API_KEY!);
    url.searchParams.set("units", "imperial"); // Fahrenheit
    
    for (const [key, value] of Object.entries(params)) {
        url.searchParams.set(key, value);
    }
    
    const response = await fetch(url.toString());
    if (!response.ok) {
        throw new Error(`Weather API error: ${response.statusText}`);
    }
    return response.json();
}

// Current weather tool
export const getCurrentWeatherTool = {
    name: "get_current_weather",
    description: `Get the current weather conditions for a city. 
        Returns temperature, conditions, humidity, and wind speed.`,
    schema: Schema.objectValue({
        city: Schema.stringValue("City name, e.g., 'New York' or 'London, UK'").isRequired(),
    }),
    execute: async (args: Record<string, unknown>) => {
        const city = args.city as string;
        
        try {
            const data = await fetchWeather("weather", { q: city });
            
            return {
                location: `${data.name}, ${data.sys.country}`,
                temperature: Math.round(data.main.temp),
                feelsLike: Math.round(data.main.feels_like),
                condition: data.weather[0].main,
                description: data.weather[0].description,
                humidity: data.main.humidity,
                windSpeed: Math.round(data.wind.speed),
                icon: data.weather[0].icon,
            };
        } catch (error) {
            return {
                error: true,
                message: `Could not find weather for "${city}". Try a different city name.`,
            };
        }
    },
};

// Forecast tool
export const getForecastTool = {
    name: "get_weather_forecast",
    description: `Get the weather forecast for the next several days.
        Returns daily high/low temperatures and conditions.`,
    schema: Schema.objectValue({
        city: Schema.stringValue("City name").isRequired(),
        days: Schema.numberValue("Number of days (1-5)").defaultTo(3),
    }),
    execute: async (args: Record<string, unknown>) => {
        const city = args.city as string;
        const days = Math.min(Math.max(args.days as number || 3, 1), 5);
        
        try {
            const data = await fetchWeather("forecast", { q: city, cnt: String(days * 8) });
            
            // Group by day and get daily summaries
            const dailyForecasts: Record<string, any[]> = {};
            
            for (const item of data.list) {
                const date = new Date(item.dt * 1000).toLocaleDateString("en-US", {
                    weekday: "short",
                    month: "short",
                    day: "numeric",
                });
                
                if (!dailyForecasts[date]) {
                    dailyForecasts[date] = [];
                }
                dailyForecasts[date].push(item);
            }
            
            const forecast = Object.entries(dailyForecasts).slice(0, days).map(([date, items]) => {
                const temps = items.map(i => i.main.temp);
                const conditions = items.map(i => i.weather[0].main);
                const mostCommon = conditions.sort((a, b) =>
                    conditions.filter(v => v === a).length - conditions.filter(v => v === b).length
                ).pop();
                
                return {
                    date,
                    high: Math.round(Math.max(...temps)),
                    low: Math.round(Math.min(...temps)),
                    condition: mostCommon,
                };
            });
            
            return {
                location: `${data.city.name}, ${data.city.country}`,
                forecast,
            };
        } catch (error) {
            return {
                error: true,
                message: `Could not get forecast for "${city}".`,
            };
        }
    },
};
```

---

## Step 2: Create the Agent

Create `src/agent.ts`:

```typescript
import { Agent, Model, Secret } from "@shuttl-io/core";
import { getCurrentWeatherTool, getForecastTool } from "./tools/weather";

const SYSTEM_PROMPT = `You are a friendly weather assistant. You help users check weather conditions and forecasts for any location.

Guidelines:
- Use get_current_weather for current conditions
- Use get_weather_forecast for multi-day forecasts
- Present information in a conversational, easy-to-read format
- Include relevant details like "feels like" temperature when notable
- Suggest appropriate clothing or activities based on conditions
- If a city isn't found, suggest alternative spellings or nearby cities

Format tips:
- Use temperature with units: "72°F"
- Describe conditions naturally: "partly cloudy" not just "Clouds"
- For forecasts, summarize trends: "warming up through the week"`;

export const weatherAgent = new Agent({
    name: "WeatherBot",
    systemPrompt: SYSTEM_PROMPT,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [getCurrentWeatherTool, getForecastTool],
});
```

---

## Step 3: Create Entry Point

Create `src/main.ts`:

```typescript
export { weatherAgent } from "./agent";
```

---

## Step 4: Configure

Create `shuttl.json`:

```json
{
    "app": "node --require ts-node/register ./src/main.ts"
}
```

---

## Step 5: Run

```bash
# Set API keys
export OPENAI_API_KEY="sk-..."
export OPENWEATHER_API_KEY="your-openweather-key"

# Start the agent
shuttl dev
```

---

## Testing the Agent

The TUI launches with four screens. Select `WeatherBot` from the **Agent Select** screen, then navigate to **Chat**.

### Current Weather

In the chat, type:

```
What's the weather like in Tokyo?
```

**Response:**

```
Currently in Tokyo, Japan it's 68°F and partly cloudy. 
The humidity is at 65% with light winds around 8 mph. 
It feels like 70°F - perfect weather for a walk!
```

### Forecast

```
What's the forecast for Paris this week?
```

**Response:**

```
Here's the forecast for Paris, France:

📅 **Fri, Jan 3**: High 52°F / Low 45°F - Cloudy
📅 **Sat, Jan 4**: High 48°F / Low 42°F - Rain
📅 **Sun, Jan 5**: High 50°F / Low 44°F - Partly Cloudy

Looks like Saturday might be rainy, so pack an umbrella if you're heading out!
```

### Follow-up Questions

```
Should I bring a jacket?
```

The agent remembers the previous context (Paris weather) and responds appropriately. Use the **Agent Debug** screen to watch the tool calls in real-time.

---

## Enhancements

### Add Weather Alerts

```typescript
export const getWeatherAlertsTool = {
    name: "get_weather_alerts",
    description: "Check for severe weather alerts in an area",
    schema: Schema.objectValue({
        city: Schema.stringValue("City name").isRequired(),
    }),
    execute: async ({ city }) => {
        // Fetch from weather alerts API
        const alerts = await fetchAlerts(city);
        return {
            hasAlerts: alerts.length > 0,
            alerts: alerts.map(a => ({
                type: a.event,
                severity: a.severity,
                description: a.description,
            })),
        };
    },
};
```

### Add Scheduled Weather Updates

```typescript
import { Rate, SlackOutcome } from "@shuttl-io/core";

export const morningWeatherAgent = new Agent({
    name: "MorningWeather",
    systemPrompt: `Generate a brief morning weather update for the team.
    Include: current conditions, today's forecast, and any alerts.`,
    model: Model.openAI("gpt-4o-mini", Secret.fromEnv("OPENAI_API_KEY")),
    tools: [getCurrentWeatherTool, getForecastTool, getWeatherAlertsTool],
    triggers: [
        Rate.cron("0 7 * * MON-FRI", "America/New_York")
            .withOnTrigger({
                onTrigger: async () => [{
                    typeName: "text",
                    text: "Generate the morning weather update for New York City.",
                }],
            }),
    ],
    outcomes: [new SlackOutcome("#general")],
});
```

### Add Unit Preferences

```typescript
const getCurrentWeatherTool = {
    name: "get_current_weather",
    description: "Get current weather with preferred units",
    schema: Schema.objectValue({
        city: Schema.stringValue("City name").isRequired(),
        units: Schema.enumValue("Temperature units", ["fahrenheit", "celsius"])
            .defaultTo("fahrenheit"),
    }),
    execute: async ({ city, units }) => {
        const apiUnits = units === "celsius" ? "metric" : "imperial";
        const data = await fetchWeather("weather", { q: city, units: apiUnits });
        
        const symbol = units === "celsius" ? "°C" : "°F";
        return {
            temperature: `${Math.round(data.main.temp)}${symbol}`,
            // ... rest of response
        };
    },
};
```

---

## Complete Code

The full project is available at: [github.com/shuttl-io/examples/weather-agent](https://github.com/shuttl-io/examples)

---

## Next Steps

- [:clock: Scheduled Tasks](scheduled-tasks.md) - Run agents on a schedule
- [:wrench: Tools Guide](../concepts/tools.md) - Build more tools
- [:brain: Agents Guide](../concepts/agents.md) - Advanced agent patterns

