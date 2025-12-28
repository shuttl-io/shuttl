import { EventEmitter } from "events";

// Create mock instances before jest.mock calls
const mockStdin = new EventEmitter();
const mockStdoutWrite = jest.fn();
const mockReadlineInterface = new EventEmitter() as EventEmitter & { close: jest.Mock };
mockReadlineInterface.close = jest.fn();

// Increase max listeners to avoid warnings during tests
mockReadlineInterface.setMaxListeners(50);

// Mock process module
jest.mock("process", () => ({
    stdin: mockStdin,
    stdout: {
        write: (data: string) => mockStdoutWrite(data),
    },
}));

// Mock readline module
jest.mock("readline", () => ({
    createInterface: jest.fn(() => mockReadlineInterface),
}));

import { StdInServer, IPCRequest, IPCResponse } from "../../src/server/http";
import { Schema } from "../../src/tools/tool";

describe("StdInServer", () => {
    let server: StdInServer;

    // Helper to parse captured responses
    const getResponses = (): IPCResponse[] => {
        return mockStdoutWrite.mock.calls.map((call) => {
            const json = call[0].replace(/\n$/, "");
            return JSON.parse(json) as IPCResponse;
        });
    };

    // Helper to get the last response
    const getLastResponse = (): IPCResponse => {
        const responses = getResponses();
        return responses[responses.length - 1];
    };

    // Helper to simulate sending a line of input
    const sendLine = (line: string) => {
        mockReadlineInterface.emit("line", line);
    };

    // Helper to send a request object
    const sendRequest = (request: IPCRequest) => {
        sendLine(JSON.stringify(request));
    };

    beforeEach(() => {
        jest.clearAllMocks();
        server = new StdInServer();
    });

    afterEach(async () => {
        await server.stop();
    });

    describe("accept()", () => {
        it("should accept an app", () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            expect(() => server.accept(mockApp)).not.toThrow();
        });
    });

    describe("start()", () => {
        it("should throw if no app is registered", async () => {
            await expect(server.start()).rejects.toThrow(
                "No app registered. Call accept() before start()."
            );
        });

        it("should send ready signal on start", async () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            server.accept(mockApp);

            // Start in background (don't await, it runs indefinitely)
            void server.start();

            // Give it a tick to send ready signal
            await new Promise((resolve) => setTimeout(resolve, 10));

            const readyResponse = getResponses().find((r) => r.id === "__ready__");
            expect(readyResponse).toBeDefined();
            expect(readyResponse!.success).toBe(true);
            expect(readyResponse!.result).toEqual({
                name: "TestApp",
                protocol: "ndjson",
                version: "1.0",
            });

            await server.stop();
        });
    });

    describe("handleLine()", () => {
        beforeEach(async () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            server.accept(mockApp);
            void server.start(); // Start but don't await
            await new Promise((resolve) => setTimeout(resolve, 10));
            jest.clearAllMocks(); // Clear ready signal
        });

        it("should skip empty lines", () => {
            sendLine("");
            sendLine("   ");
            sendLine("\t");
            expect(mockStdoutWrite).not.toHaveBeenCalled();
        });

        it("should return parse error for invalid JSON", () => {
            sendLine("not valid json");

            const response = getLastResponse();
            expect(response.id).toBe("__parse_error__");
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("PARSE_ERROR");
        });

        it("should return error for missing id field", () => {
            sendLine(JSON.stringify({ method: "ping" }));

            const response = getLastResponse();
            expect(response.id).toBe("__invalid__");
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_REQUEST");
            expect(response.errorObj?.message).toContain("'id' field");
        });

        it("should return error for non-string id field", () => {
            sendLine(JSON.stringify({ id: 123, method: "ping" }));

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_REQUEST");
        });

        it("should return error for missing method field", () => {
            sendLine(JSON.stringify({ id: "1" }));

            const response = getLastResponse();
            expect(response.id).toBe("1");
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_REQUEST");
            expect(response.errorObj?.message).toContain("'method' field");
        });

        it("should return error for non-string method field", () => {
            sendLine(JSON.stringify({ id: "1", method: 123 }));

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_REQUEST");
        });
    });

    describe("handleRequest()", () => {
        beforeEach(async () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            server.accept(mockApp);
            void server.start();
            await new Promise((resolve) => setTimeout(resolve, 10));
            jest.clearAllMocks();
        });

        describe("ping", () => {
            it("should respond with pong and timestamp", () => {
                const now = Date.now();
                sendRequest({ id: "1", method: "ping" });

                const response = getLastResponse();
                expect(response.id).toBe("1");
                expect(response.success).toBe(true);
                expect(response.result).toMatchObject({
                    pong: true,
                    protocol_version: "1.0",
                });
                expect((response.result as Record<string, unknown>).timestamp).toBeGreaterThanOrEqual(now);
            });
        });

        describe("getAppInfo", () => {
            it("should return app information", () => {
                sendRequest({ id: "2", method: "getAppInfo" });

                const response = getLastResponse();
                expect(response.id).toBe("2");
                expect(response.success).toBe(true);
                expect(response.result).toEqual({
                    name: "TestApp",
                    agentCount: 0,
                    toolkitCount: 0,
                });
            });
        });

        describe("listAgents", () => {
            it("should return empty array when no agents", () => {
                sendRequest({ id: "3", method: "listAgents" });

                const response = getLastResponse();
                expect(response.id).toBe("3");
                expect(response.success).toBe(true);
                expect(response.result).toEqual([]);
            });

            it("should return agent information", async () => {
                const mockAgent = {
                    name: "TestAgent",
                    systemPrompt: "You are a test agent",
                    model: { provider: "openai", name: "gpt-4" },
                    toolkits: [{ name: "toolkit1" }],
                };
                const mockApp = {
                    name: "TestApp",
                    agents: [mockAgent],
                    toolkits: new Set(),
                };

                await server.stop();
                server = new StdInServer();
                server.accept(mockApp);
                void server.start();
                await new Promise((resolve) => setTimeout(resolve, 10));
                jest.clearAllMocks();

                sendRequest({ id: "4", method: "listAgents" });

                const response = getLastResponse();
                expect(response.success).toBe(true);
                expect(response.result).toEqual([
                    {
                        name: "TestAgent",
                        systemPrompt: "You are a test agent",
                        model: { provider: "openai", name: "gpt-4" },
                        toolkits: ["toolkit1"],
                    },
                ]);
            });
        });

        describe("listToolkits", () => {
            it("should return empty array when no toolkits", () => {
                sendRequest({ id: "5", method: "listToolkits" });

                const response = getLastResponse();
                
                expect(response.id).toBe("5");
                expect(response.success).toEqual(true);
                expect(response.result).toEqual([]);
            });

            it("should return toolkit information with tools", async () => {
                const mockTool = {
                    name: "testTool",
                    description: "A test tool",
                    schema: Schema.objectValue({
                        arg1: Schema.stringValue("A test argument").isRequired(),
                    }),
                    execute: jest.fn(),
                };
                const mockToolkit = {
                    name: "TestToolkit",
                    description: "A test toolkit",
                    tools: [mockTool],
                };
                const mockApp = {
                    name: "TestApp",
                    agents: [],
                    toolkits: new Set([mockToolkit]),
                };

                await server.stop();
                server = new StdInServer();
                server.accept(mockApp);
                void server.start();
                await new Promise((resolve) => setTimeout(resolve, 10));
                jest.clearAllMocks();

                sendRequest({ id: "6", method: "listToolkits" });

                const response = getLastResponse();
                console.log(response);
                expect(response.success).toBe(true);
                expect(response.result).toEqual([
                    {
                        name: "TestToolkit",
                        description: "A test toolkit",
                        tools: [
                            {
                                name: "testTool",
                                description: "A test tool",
                                args: { 
                                    arg1: { argType: "string", required: true, description: "A test argument" } },
                            },
                        ],
                    },
                ]);
            });
        });

        describe("shutdown", () => {
            it("should respond and stop the server", async () => {
                sendRequest({ id: "7", method: "shutdown" });

                const response = getLastResponse();
                expect(response.id).toBe("7");
                expect(response.success).toBe(true);
                expect(response.result).toEqual({ shutting_down: true });
            });
        });

        describe("unknown method", () => {
            it("should return error for unknown method", () => {
                sendRequest({ id: "8", method: "unknownMethod" });

                const response = getLastResponse();
                expect(response.id).toBe("8");
                expect(response.success).toBe(false);
                expect(response.errorObj?.code).toBe("UNKNOWN_METHOD");
                expect(response.errorObj?.message).toContain("unknownMethod");
            });
        });
    });

    describe("handleInvokeTool()", () => {
        let mockTool: {
            name: string;
            description: string;
            execute: jest.Mock;
            produceArgs: jest.Mock;
        };
        let mockToolkit: {
            name: string;
            description: string;
            tools: typeof mockTool[];
        };

        beforeEach(async () => {
            mockTool = {
                name: "testTool",
                description: "A test tool",
                execute: jest.fn().mockReturnValue({ result: "success" }),
                produceArgs: jest.fn().mockReturnValue({}),
            };
            mockToolkit = {
                name: "TestToolkit",
                description: "A test toolkit",
                tools: [mockTool],
            };
            const mockApp = {
                name: "TestApp",
                agents: [],
                toolkits: new Set([mockToolkit]),
            };

            server.accept(mockApp);
            void server.start();
            await new Promise((resolve) => setTimeout(resolve, 10));
            jest.clearAllMocks();
        });

        it("should return error if toolkit param is missing", () => {
            sendRequest({
                id: "10",
                method: "invokeTool",
                body: { tool: "testTool" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_PARAMS");
        });

        it("should return error if tool param is missing", () => {
            sendRequest({
                id: "11",
                method: "invokeTool",
                body: { toolkit: "TestToolkit" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_PARAMS");
        });

        it("should return error if toolkit not found", () => {
            sendRequest({
                id: "12",
                method: "invokeTool",
                body: { toolkit: "NonExistent", tool: "testTool" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
            expect(response.errorObj?.message).toContain("Toolkit not found");
        });

        it("should return error if tool not found in toolkit", () => {
            sendRequest({
                id: "13",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "nonExistentTool" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
            expect(response.errorObj?.message).toContain("Tool not found");
        });

        it("should execute tool and return result", () => {
            sendRequest({
                id: "14",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "testTool", args: { foo: "bar" } },
            });

            expect(mockTool.execute).toHaveBeenCalledWith({ foo: "bar" });
            const response = getLastResponse();
            expect(response.success).toBe(true);
            expect(response.result).toEqual({ result: "success" });
        });

        it("should handle async tool execution", async () => {
            mockTool.execute.mockResolvedValue({ asyncResult: "done" });

            sendRequest({
                id: "15",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "testTool" },
            });

            // Wait for async resolution
            await new Promise((resolve) => setTimeout(resolve, 10));

            const response = getLastResponse();
            expect(response.id).toBe("15");
            expect(response.success).toBe(true);
            expect(response.result).toEqual({ asyncResult: "done" });
        });

        it("should handle async tool error", async () => {
            mockTool.execute.mockRejectedValue(new Error("Async error"));

            sendRequest({
                id: "16",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "testTool" },
            });

            // Wait for async rejection
            await new Promise((resolve) => setTimeout(resolve, 10));

            const response = getLastResponse();
            expect(response.id).toBe("16");
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("TOOL_ERROR");
            expect(response.errorObj?.message).toBe("Async error");
        });

        it("should handle sync tool error", () => {
            mockTool.execute.mockImplementation(() => {
                throw new Error("Sync error");
            });

            sendRequest({
                id: "17",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "testTool" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("TOOL_ERROR");
            expect(response.errorObj?.message).toBe("Sync error");
        });

        it("should use empty args if not provided", () => {
            sendRequest({
                id: "18",
                method: "invokeTool",
                body: { toolkit: "TestToolkit", tool: "testTool" },
            });

            expect(mockTool.execute).toHaveBeenCalledWith({});
        });
    });

    describe("handleInvokeAgent()", () => {
        let mockAgent: {
            name: string;
            systemPrompt: string;
            model: { provider: string; name: string };
            toolkits: { name: string }[];
            invoke: jest.Mock;
        };

        beforeEach(async () => {
            mockAgent = {
                name: "TestAgent",
                systemPrompt: "You are a test agent",
                model: { provider: "openai", name: "gpt-4" },
                toolkits: [{ name: "toolkit1" }],
                invoke: jest.fn().mockResolvedValue({ threadId: "thread-123" }),
            };
            const mockApp = {
                name: "TestApp",
                agents: [mockAgent],
                toolkits: new Set(),
            };

            server.accept(mockApp);
            void server.start();
            await new Promise((resolve) => setTimeout(resolve, 10));
            jest.clearAllMocks();
        });

        it("should return error if agent param is missing", () => {
            sendRequest({
                id: "20",
                method: "invokeAgent",
                body: { prompt: "Hello" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_PARAMS");
            expect(response.errorObj?.message).toContain("'agent'");
        });

        it("should return error if agent not found", () => {
            sendRequest({
                id: "21",
                method: "invokeAgent",
                body: { agent: "NonExistentAgent", prompt: "Hello" },
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
            expect(response.errorObj?.message).toContain("Agent not found");
        });

        it("should invoke agent and return thread info on success", async () => {
            sendRequest({
                id: "22",
                method: "invokeAgent",
                body: { agent: "TestAgent", prompt: "Hello, agent!" },
            });

            // Wait for async invoke
            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(mockAgent.invoke).toHaveBeenCalled();
            const response = getLastResponse();
            expect(response.id).toBe("22");
            expect(response.success).toBe(true);
            expect(response.result).toMatchObject({
                threadId: "thread-123",
                status: "invoked",
            });
        });

        it("should invoke agent with empty prompt if not provided", async () => {
            sendRequest({
                id: "23",
                method: "invokeAgent",
                body: { agent: "TestAgent" },
            });

            // Wait for async invoke
            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(mockAgent.invoke).toHaveBeenCalledWith(
                "",
                undefined,
                expect.anything(),
                undefined
            );
        });

        it("should pass threadId if provided", async () => {
            sendRequest({
                id: "24",
                method: "invokeAgent",
                body: { agent: "TestAgent", prompt: "Continue...", threadId: "existing-thread" },
            });

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(mockAgent.invoke).toHaveBeenCalledWith(
                "Continue...",
                "existing-thread",
                expect.anything(),
                undefined
            );
        });

        it("should pass attachments if provided", async () => {
            const attachments = [
                { name: "test.txt", content: "SGVsbG8gV29ybGQ=", mimeType: "text/plain" },
            ];
            sendRequest({
                id: "25a",
                method: "invokeAgent",
                body: { agent: "TestAgent", prompt: "Check this file", attachments },
            });

            await new Promise((resolve) => setTimeout(resolve, 10));

            expect(mockAgent.invoke).toHaveBeenCalledWith(
                "Check this file",
                undefined,
                expect.anything(),
                attachments
            );
        });

        it("should return error on agent invoke failure", async () => {
            mockAgent.invoke.mockRejectedValue(new Error("Agent failed"));

            sendRequest({
                id: "25",
                method: "invokeAgent",
                body: { agent: "TestAgent", prompt: "Hello" },
            });

            // Wait for async rejection
            await new Promise((resolve) => setTimeout(resolve, 10));

            const response = getLastResponse();
            expect(response.id).toBe("25");
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INTERNAL_ERROR");
        });

        it("should handle no body at all", () => {
            sendRequest({
                id: "26",
                method: "invokeAgent",
            });

            const response = getLastResponse();
            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("INVALID_PARAMS");
        });
    });

    describe("stop()", () => {
        it("should stop the server gracefully", async () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            server.accept(mockApp);

            const startPromise = server.start();
            await new Promise((resolve) => setTimeout(resolve, 10));

            await server.stop();

            // Server should exit the start loop
            await expect(
                Promise.race([
                    startPromise,
                    new Promise((_, reject) =>
                        setTimeout(() => reject(new Error("Timeout")), 500)
                    ),
                ])
            ).resolves.toBeUndefined();
        });
    });

    describe("readline close event", () => {
        it("should stop running when readline closes", async () => {
            const mockApp = { name: "TestApp", agents: [], toolkits: new Set() };
            server.accept(mockApp);

            const startPromise = server.start();
            await new Promise((resolve) => setTimeout(resolve, 10));

            // Simulate stdin close
            mockReadlineInterface.emit("close");

            // Server should exit
            await expect(
                Promise.race([
                    startPromise,
                    new Promise((_, reject) =>
                        setTimeout(() => reject(new Error("Timeout")), 500)
                    ),
                ])
            ).resolves.toBeUndefined();
        });
    });
});
