/**
 * Integration tests for TypeScript/JavaScript JSII target
 * 
 * These tests spawn the TypeScript test app as a separate process
 * and communicate with it via the IPC protocol over STDIN/STDOUT.
 */

import * as path from "path";
import { IPCClient, createIPCClient } from "./helpers/ipc-client";

const FIXTURE_DIR = path.join(__dirname, "fixtures", "typescript");
const APP_PATH = path.join(FIXTURE_DIR, "app.ts");

describe("TypeScript Integration Tests", () => {
    let client: IPCClient;

    beforeAll(async () => {
        // Spawn the TypeScript app using ts-node
        client = await createIPCClient({
            command: "npx",
            args: ["ts-node", APP_PATH],
            cwd: path.join(__dirname, ".."),
        });
    });

    afterAll(async () => {
        if (client) {
            await client.close();
        }
    });

    describe("ready signal", () => {
        it("should receive ready signal on startup", async () => {
            // The ready signal is received during spawn, verify client is connected
            expect(client).toBeDefined();
        });
    });

    describe("ping", () => {
        it("should respond to ping request", async () => {
            const response = await client.ping();

            expect(response.success).toBe(true);
            expect(response.result).toMatchObject({
                pong: true,
                protocol_version: "1.0",
            });
            expect((response.result as Record<string, unknown>).timestamp).toBeDefined();
        });

        it("should respond to multiple ping requests", async () => {
            const response1 = await client.ping();
            const response2 = await client.ping();
            const response3 = await client.ping();

            expect(response1.success).toBe(true);
            expect(response2.success).toBe(true);
            expect(response3.success).toBe(true);
        });
    });

    describe("getAppInfo", () => {
        it("should return app information", async () => {
            const response = await client.getAppInfo();

            expect(response.success).toBe(true);
            expect(response.result).toMatchObject({
                name: "TypeScriptTestApp",
                agentCount: 1,
                toolkitCount: 2,
            });
        });
    });

    describe("listAgents", () => {
        it("should list all agents", async () => {
            const response = await client.listAgents();

            expect(response.success).toBe(true);
            const agents = response.result as Array<Record<string, unknown>>;
            expect(agents).toHaveLength(1);
            expect(agents[0]).toMatchObject({
                name: "TestAgent",
                systemPrompt: "You are a helpful test agent for integration testing.",
            });
            expect(agents[0].toolkits).toEqual(["UtilityToolkit", "AsyncToolkit"]);
        });
    });

    describe("listToolkits", () => {
        it("should list all toolkits with their tools", async () => {
            const response = await client.listToolkits();

            expect(response.success).toBe(true);
            const toolkits = response.result as Array<Record<string, unknown>>;
            expect(toolkits).toHaveLength(2);

            // Find utility toolkit
            const utilToolkit = toolkits.find(t => t.name === "UtilityToolkit");
            expect(utilToolkit).toBeDefined();
            expect(utilToolkit!.description).toBe("A toolkit with utility functions");
            
            const tools = utilToolkit!.tools as Array<Record<string, unknown>>;
            expect(tools).toHaveLength(2);
            
            const echoTool = tools.find(t => t.name === "echo");
            expect(echoTool).toBeDefined();
            expect(echoTool!.description).toBe("Echoes the input message back");
        });
    });

    describe("invokeTool", () => {
        it("should invoke echo tool and return result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "echo", {
                message: "Hello, Integration Test!",
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.echoed).toBe("Hello, Integration Test!");
            expect(result.timestamp).toBeDefined();
        });

        it("should invoke math tool and return calculation result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "add", {
                a: 10,
                b: 25,
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.result).toBe(35);
            expect(result.operation).toBe("add");
        });

        it("should invoke async tool and wait for result", async () => {
            const response = await client.invokeTool("AsyncToolkit", "delay", {
                ms: 50,
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.waited).toBe(50);
            expect(result.completed).toBe(true);
        });

        it("should return error for non-existent toolkit", async () => {
            const response = await client.invokeTool("NonExistentToolkit", "echo", {});

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
            expect(response.errorObj?.message).toContain("Toolkit not found");
        });

        it("should return error for non-existent tool", async () => {
            const response = await client.invokeTool("UtilityToolkit", "nonexistent", {});

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
            expect(response.errorObj?.message).toContain("Tool not found");
        });
    });

    describe("unknown method", () => {
        it("should return error for unknown method", async () => {
            const response = await client.send("unknownMethod");

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("UNKNOWN_METHOD");
        });
    });
});

