# Deployment

Deploy your Shuttl agents to production. Choose between managed hosting with Shuttl Cloud or self-hosting on your own infrastructure.

---

## Shuttl Cloud (Recommended)

The easiest way to deploy agents is with Shuttl Cloud—no infrastructure to manage.

### Step 1: Create an Account

```bash
shuttl login
```

This opens your browser to authenticate with GitHub or Google.

### Step 2: Link Your Project

```bash
shuttl link
```

Follow the prompts to connect your project to Shuttl Cloud.

### Step 3: Deploy

```bash
shuttl deploy
```

That's it! Your agents are now live with:

- **Automatic scaling** based on demand
- **Built-in monitoring** and logging
- **Managed triggers** (cron, webhooks, etc.)
- **SSL/TLS** out of the box

### Environment Variables

Set secrets and configuration in the Shuttl Cloud dashboard or via CLI:

```bash
shuttl env set OPENAI_API_KEY=sk-...
shuttl env set SLACK_BOT_TOKEN=xoxb-...
```

---

## Self-Hosting

Prefer to run on your own infrastructure? Shuttl provides everything you need.

### Build Your Project

The `shuttl build` command generates a complete deployment manifest:

```bash
shuttl build --output ./dist
```

This creates:

```
dist/
├── manifest.json      # Full deployment manifest
├── agents.json        # Agent definitions
├── triggers.json      # Trigger configurations
├── outcomes.json      # Outcome configurations
└── bundle.js          # Compiled application (if applicable)
```

### Understanding the Manifest

The `manifest.json` contains everything needed to deploy:

```json
{
    "agents": [
        {
            "name": "SupportBot",
            "triggers": [
                { "type": "api", "config": {} },
                { "type": "rate", "config": { "cron": "0 9 * * *", "timezone": "UTC" } }
            ],
            "outcomes": [
                { "type": "streaming" },
                { "type": "slack", "config": { "channel": "#support" } }
            ]
        }
    ]
}
```

Use this to determine your infrastructure needs:

| Trigger Type | Infrastructure Needed |
|--------------|----------------------|
| `api` | HTTP server, load balancer |
| `rate` (cron) | Cron scheduler or Lambda/Cloud Functions |
| `email` | Email server integration (SES, SendGrid) |
| `file` | File system watcher or S3 triggers |

---

## The Serve Command

The `shuttl serve` command runs your agents in production mode with an HTTP server.

### Mode 1: Serve All Agents

Run all agents with all their triggers:

```bash
shuttl serve
```

With SSL:

```bash
shuttl serve --ssl-cert ./cert.pem --ssl-key ./key.pem
```

#### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `8080` | HTTP server port |
| `--ssl-cert` | | | Path to SSL certificate |
| `--ssl-key` | | | Path to SSL private key |
| `--config` | `-c` | `./shuttl.json` | Config file path |

#### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/chat` | POST | Send message to an agent |
| `/chat/:threadId` | POST | Continue a conversation |
| `/agents` | GET | List available agents |
| `/agents/:name` | GET | Get agent details |
| `/health` | GET | Health check |

---

### Mode 2: Serve Specific Agent or Trigger

Run only specific agents or triggers:

```bash
# Serve only the SupportBot agent
shuttl serve --agent SupportBot

# Serve only the API trigger for SupportBot
shuttl serve --agent SupportBot --trigger api

# Serve only rate-based triggers (for cron workers)
shuttl serve --trigger rate
```

#### Use Cases

| Command | Use Case |
|---------|----------|
| `--agent SupportBot` | Dedicated instance for one agent |
| `--trigger api` | API-only server (no scheduled tasks) |
| `--trigger rate` | Worker for scheduled/cron triggers |

This allows you to scale different parts of your system independently:

```yaml
# docker-compose.yml example
services:
  api:
    command: shuttl serve --trigger api
    ports:
      - "8080:8080"
    deploy:
      replicas: 3
  
  workers:
    command: shuttl serve --trigger rate
    deploy:
      replicas: 1
```

---

### Mode 3: Invoke a Specific Trigger

For event-based deployments (Lambda, Cloud Functions, Kubernetes CronJobs), invoke a specific agent and trigger directly:

```bash
shuttl serve --invoke --agent DailyReporter --trigger rate
```

This:

1. Loads the agent
2. Fires the specified trigger once
3. Processes the outcome
4. Exits

#### Perfect For

- **AWS Lambda**: Invoke on CloudWatch Events schedule
- **Google Cloud Functions**: Trigger from Cloud Scheduler
- **Kubernetes CronJobs**: Run as a one-shot container
- **GitHub Actions**: Scheduled workflows

#### Example: AWS Lambda

```yaml
# serverless.yml
functions:
  dailyReport:
    handler: handler.run
    events:
      - schedule: cron(0 9 * * ? *)
    environment:
      OPENAI_API_KEY: ${ssm:/shuttl/openai-key}
```

```javascript
// handler.js
const { execSync } = require('child_process');

exports.run = async (event) => {
    execSync('shuttl serve --invoke --agent DailyReporter --trigger rate');
    return { statusCode: 200 };
};
```

#### Example: Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-reporter
spec:
  schedule: "0 9 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: shuttl
              image: your-registry/shuttl-app:latest
              command: ["shuttl", "serve", "--invoke", "--agent", "DailyReporter", "--trigger", "rate"]
              env:
                - name: OPENAI_API_KEY
                  valueFrom:
                    secretKeyRef:
                      name: shuttl-secrets
                      key: openai-key
          restartPolicy: OnFailure
```

---

## Deployment Patterns

### Pattern 1: Simple HTTP Server

For small deployments, run everything on one server:

```bash
shuttl serve --port 80 --ssl-cert /etc/ssl/cert.pem --ssl-key /etc/ssl/key.pem
```

### Pattern 2: Separate API and Workers

Scale API and background workers independently:

```
┌─────────────────┐     ┌─────────────────┐
│   Load Balancer │     │   Cron Scheduler│
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  API Servers    │     │  Worker Nodes   │
│ (--trigger api) │     │ (--trigger rate)│
│   x3 replicas   │     │   x1 replica    │
└─────────────────┘     └─────────────────┘
```

### Pattern 3: Serverless

Use the invoke mode with cloud functions:

```
┌─────────────────┐
│  API Gateway    │──────▶ Lambda (--invoke --trigger api)
└─────────────────┘

┌─────────────────┐
│ CloudWatch/Cron │──────▶ Lambda (--invoke --trigger rate)
└─────────────────┘
```

---

## Docker Deployment

### Dockerfile

```dockerfile
FROM node:20-slim

WORKDIR /app

# Copy built application
COPY dist/ ./dist/
COPY shuttl.json ./

# Install Shuttl CLI
RUN curl -fsSL https://shuttl.dev/install.sh | bash

EXPOSE 8080

CMD ["shuttl", "serve"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  shuttl:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}
    restart: unless-stopped
```

---

## Health Checks

All serve modes expose a `/health` endpoint:

```bash
curl http://localhost:8080/health
```

Response:

```json
{
    "status": "healthy",
    "agents": 3,
    "uptime": 3600
}
```

Use this for:

- Load balancer health checks
- Kubernetes liveness probes
- Monitoring systems

---

## Next Steps

- [:book: CLI Commands](../cli/commands.md) - Full command reference
- [:zap: Triggers](../concepts/triggers.md) - Configure triggers for deployment
- [:outbox_tray: Outcomes](../concepts/outcomes.md) - Route outputs in production

