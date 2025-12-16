/**
 * TypeScript test application for integration testing
 * 
 * This app creates a simple agent with a toolkit and tool,
 * then starts the StdInServer to accept IPC commands.
 */

import { App, Agent, Model, Toolkit, ITool, ToolArg, StdInServer } from "../../../src";

// A simple test tool that echoes its input
class EchoTool implements ITool {
    public name = "echo";
    public description = "Echoes the input message back";

    execute(args: Record<string, unknown>): unknown {
        const message = args.message as string || "no message";
        return {
            echoed: message,
            timestamp: Date.now(),
        };
    }

    produceArgs(): Record<string, ToolArg> {
        return {
            message: {
                name: "message",
                argType: "string",
                description: "The message to echo",
                required: true,
                defaultValue: undefined,
            },
        };
    }
}

// A tool that performs simple math
class MathTool implements ITool {
    public name = "add";
    public description = "Adds two numbers together";

    execute(args: Record<string, unknown>): unknown {
        const a = args.a as number || 0;
        const b = args.b as number || 0;
        return {
            result: a + b,
            operation: "add",
        };
    }

    produceArgs(): Record<string, ToolArg> {
        return {
            a: {
                name: "a",
                argType: "number",
                description: "First number",
                required: true,
                defaultValue: undefined,
            },
            b: {
                name: "b",
                argType: "number",
                description: "Second number",
                required: true,
                defaultValue: undefined,
            },
        };
    }
}

// An async tool that simulates a delay
class AsyncTool implements ITool {
    public name = "delay";
    public description = "Waits for a specified time then returns";

    async execute(args: Record<string, unknown>): Promise<unknown> {
        const ms = args.ms as number || 100;
        await new Promise(resolve => setTimeout(resolve, ms));
        return {
            waited: ms,
            completed: true,
        };
    }

    produceArgs(): Record<string, ToolArg> {
        return {
            ms: {
                name: "ms",
                argType: "number",
                description: "Milliseconds to wait",
                required: false,
                defaultValue: 100,
            },
        };
    }
}

// Create the application
function main() {
    // Create server
    const server = new StdInServer();
    
    // Create app
    const app = new App("TypeScriptTestApp", server);
    
    // Create model
    const model = new Model({
        identifier: "test-model",
        key: "test-key-12345",
    });
    
    // Create toolkit with tools
    const utilToolkit = new Toolkit({
        name: "UtilityToolkit",
        description: "A toolkit with utility functions",
        tools: [new EchoTool(), new MathTool()],
    });
    
    const asyncToolkit = new Toolkit({
        name: "AsyncToolkit", 
        description: "A toolkit with async tools",
        tools: [new AsyncTool()],
    });
    
    // Create agent
    const agent = new Agent({
        name: "TestAgent",
        systemPrompt: "You are a helpful test agent for integration testing.",
        model: model,
        toolkits: [utilToolkit, asyncToolkit],
    });
    
    // Add agent to app
    app.addAgent(agent);
    
    // Start serving
    app.serve();
}

main();













