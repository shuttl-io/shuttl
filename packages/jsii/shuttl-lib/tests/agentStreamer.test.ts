import { EventEmitter } from "events";

// Create mock stdout before imports
const mockStdoutWrite = jest.fn();

// Mock process module
jest.mock("process", () => ({
    stdin: new EventEmitter(),
    stdout: {
        write: (data: string) => mockStdoutWrite(data),
    },
}));

import { Agent, AgentStreamer } from "../src/agent";
import { IModel, IModelFactory, IModelStreamer, ModelContent, ModelResponse, ToolCallResponse } from "../src/models/types";
import { ITool, Schema } from "../src/tools/tool";
import { Toolkit } from "../src/tools/toolkit";

// Mock tool implementation for testing
class MockTool implements ITool {
    public name: string;
    public description: string;
    public schema: Schema;
    public executeResult: unknown = { result: "success" };

    constructor(name: string, description: string) {
        this.name = name;
        this.description = description;
        this.schema = Schema.objectValue({});
    }

    execute(_args: Record<string, unknown>): unknown {
        return this.executeResult;
    }
}

// Mock model for testing
class MockModel implements IModel {
    public readonly threadId?: string = "mock-thread-id";
    public invokedWith: (ModelContent | ToolCallResponse)[][] = [];

    async invoke(prompt: (ModelContent | ToolCallResponse)[], _streamer: IModelStreamer): Promise<void> {
        this.invokedWith.push(prompt);
    }
}

// Mock model factory
class MockModelFactory implements IModelFactory {
    private mockModel: MockModel;

    constructor(model?: MockModel) {
        this.mockModel = model ?? new MockModel();
    }

    async create(_props: { systemPrompt: string }): Promise<IModel> {
        return this.mockModel;
    }
}

describe("AgentStreamer", () => {
    beforeEach(() => {
        mockStdoutWrite.mockClear();
    });

    // Helper to parse stdout writes
    const getWrittenMessages = () => {
        return mockStdoutWrite.mock.calls.map((call) => {
            const json = call[0].replace(/\n$/, "");
            return JSON.parse(json);
        });
    };

    const getLastMessage = () => {
        const messages = getWrittenMessages();
        return messages[messages.length - 1];
    };

    describe("constructor", () => {
        it("should create an AgentStreamer with agent and controlID", () => {
            const mockFactory = new MockModelFactory();
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: mockFactory,
            });

            const streamer = new AgentStreamer(agent, "control-123");

            expect(streamer).toBeInstanceOf(AgentStreamer);
        });
    });

    describe("recieve()", () => {
        let agent: Agent;
        let streamer: AgentStreamer;
        let mockModel: MockModel;

        beforeEach(() => {
            mockModel = new MockModel();
            const mockFactory = new MockModelFactory(mockModel);
            const tool = new MockTool("test_tool", "A test tool");
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [tool],
            });

            agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit],
                systemPrompt: "Test prompt",
                model: mockFactory,
            });

            streamer = new AgentStreamer(agent, "test-control-id");
        });

        it("should handle undefined data", async () => {
            const response: ModelResponse = {
                eventName: "response.something",
                data: undefined as any,
            };

            await streamer.recieve(mockModel, response);

            // Should not write anything for undefined data
            expect(mockStdoutWrite).not.toHaveBeenCalled();
        });

        describe("tool_call handling", () => {
            it("should write tool_call event to stdout", async () => {
                const response: ModelResponse = {
                    eventName: "response.function_call",
                    data: {
                        typeName: "tool_call",
                        toolCall: {
                            outputType: "tool_call",
                            name: "test_tool",
                            arguments: { arg: "value" },
                            callId: "call-1",
                        },
                    },
                };

                await streamer.recieve(mockModel, response);

                const message = getLastMessage();
                expect(message.id).toBe("test-control-id");
                expect(message.type).toBe("tool_call");
                expect(message.success).toBe(true);
                expect(message.result.typeName).toBe("tool_call");
            });
        });

        describe("output_text handling", () => {
            it("should write output_text event to stdout", async () => {
                const response: ModelResponse = {
                    eventName: "response.output_text.done",
                    data: {
                        typeName: "output_text",
                        outputText: {
                            outputType: "output_text",
                            text: "Hello, world!",
                        },
                    },
                };

                await streamer.recieve(mockModel, response);

                const message = getLastMessage();
                expect(message.id).toBe("test-control-id");
                expect(message.type).toBe("output_text");
                expect(message.success).toBe(true);
            });
        });

        describe("output_text_delta handling", () => {
            it("should write output_text_delta event to stdout", async () => {
                const response: ModelResponse = {
                    eventName: "response.output_text.delta",
                    data: {
                        typeName: "output_text_delta",
                        outputTextDelta: {
                            outputType: "output_text_delta",
                            delta: "Hello",
                            sequenceNumber: 1,
                        },
                    },
                };

                await streamer.recieve(mockModel, response);

                const message = getLastMessage();
                expect(message.type).toBe("output_text_delta");
                expect(message.success).toBe(true);
            });
        });

        describe("output_text.part.done handling", () => {
            it("should write output_text.part.done event to stdout", async () => {
                const response: ModelResponse = {
                    eventName: "response.output_item.done",
                    data: {
                        typeName: "output_text.part.done",
                        outputText: {
                            outputType: "output_text",
                            text: "Complete text",
                        },
                    },
                };

                await streamer.recieve(mockModel, response);

                const message = getLastMessage();
                expect(message.type).toBe("output_text.part.done");
                expect(message.success).toBe(true);
            });
        });

        describe("response.requested handling", () => {
            it("should write response.requested event to stdout", async () => {
                const response: ModelResponse = {
                    eventName: "response.requested",
                    data: {
                        typeName: "response.requested",
                        requested: { model: "gpt-4" },
                    },
                };

                await streamer.recieve(mockModel, response);

                const message = getLastMessage();
                expect(message.type).toBe("response.requested");
                expect(message.success).toBe(true);
            });
        });

        describe("array data handling", () => {
            it("should process each item in data array", async () => {
                const response: ModelResponse = {
                    eventName: "response.completed",
                    data: [
                        {
                            typeName: "output_text",
                            outputText: {
                                outputType: "output_text",
                                text: "First",
                            },
                        },
                        {
                            typeName: "output_text",
                            outputText: {
                                outputType: "output_text",
                                text: "Second",
                            },
                        },
                    ],
                };

                await streamer.recieve(mockModel, response);

                const messages = getWrittenMessages();
                // Should have written twice for the two data items
                expect(messages.filter(m => m.type === "output_text").length).toBe(2);
            });
        });
    });

    describe("write() output format", () => {
        let agent: Agent;
        let streamer: AgentStreamer;
        let mockModel: MockModel;

        beforeEach(() => {
            mockModel = new MockModel();
            const mockFactory = new MockModelFactory(mockModel);

            agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: mockFactory,
            });

            streamer = new AgentStreamer(agent, "control-id-456");
        });

        it("should output valid JSON with newline", async () => {
            const response: ModelResponse = {
                eventName: "test",
                data: {
                    typeName: "output_text",
                    outputText: { outputType: "output_text", text: "test" },
                },
            };

            await streamer.recieve(mockModel, response);

            const call = mockStdoutWrite.mock.calls[0][0];
            expect(call.endsWith("\n")).toBe(true);
            expect(() => JSON.parse(call)).not.toThrow();
        });

        it("should include controlID as id in output", async () => {
            const response: ModelResponse = {
                eventName: "test",
                data: {
                    typeName: "output_text",
                    outputText: { outputType: "output_text", text: "test" },
                },
            };

            await streamer.recieve(mockModel, response);

            const message = getLastMessage();
            expect(message.id).toBe("control-id-456");
        });

        it("should include success: true for successful messages", async () => {
            const response: ModelResponse = {
                eventName: "test",
                data: {
                    typeName: "output_text",
                    outputText: { outputType: "output_text", text: "test" },
                },
            };

            await streamer.recieve(mockModel, response);

            const message = getLastMessage();
            expect(message.success).toBe(true);
        });

        it("should include result field with data", async () => {
            const response: ModelResponse = {
                eventName: "test",
                data: {
                    typeName: "output_text",
                    outputText: { outputType: "output_text", text: "the text" },
                },
            };

            await streamer.recieve(mockModel, response);

            const message = getLastMessage();
            expect(message.result).toBeDefined();
            expect(message.result.typeName).toBe("output_text");
        });
    });

    describe("integration with Agent", () => {
        it("should be created by Agent.invoke when no streamer provided", async () => {
            const mockModel = new MockModel();
            const mockFactory = new MockModelFactory(mockModel);

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: mockFactory,
            });

            // When invoke is called without a streamer, a default AgentStreamer should be created
            await agent.invoke("Test prompt");

            expect(mockModel.invokedWith.length).toBeGreaterThan(0);
        });

        it("should be used by Agent.invoke when provided", async () => {
            const mockModel = new MockModel();
            const mockFactory = new MockModelFactory(mockModel);

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: mockFactory,
            });

            const customStreamer = new AgentStreamer(agent, "custom-control");

            await agent.invoke("Test prompt", undefined, customStreamer);

            expect(mockModel.invokedWith.length).toBeGreaterThan(0);
        });
    });
});

describe("Agent additional tests", () => {
    describe("getTool()", () => {
        it("should return a tool by name", () => {
            const tool = new MockTool("my_tool", "My tool");
            const toolkit = new Toolkit({
                name: "MyToolkit",
                tools: [tool],
            });

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            const foundTool = agent.getTool("my_tool");

            expect(foundTool).toBe(tool);
        });

        it("should throw error if tool not found", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            expect(() => agent.getTool("nonexistent")).toThrow(
                "Tool nonexistent not found"
            );
        });

        it("should find tools from multiple toolkits", () => {
            const tool1 = new MockTool("tool1", "Tool 1");
            const tool2 = new MockTool("tool2", "Tool 2");
            const toolkit1 = new Toolkit({ name: "Toolkit1", tools: [tool1] });
            const toolkit2 = new Toolkit({ name: "Toolkit2", tools: [tool2] });

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit1, toolkit2],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            expect(agent.getTool("tool1")).toBe(tool1);
            expect(agent.getTool("tool2")).toBe(tool2);
        });
    });

    describe("getToolCallResult()", () => {
        it("should format tool call result correctly", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            const result = agent.getToolCallResult("call-123", { data: "value" });

            expect(result.call_id).toBe("call-123");
            expect(result.type).toBe("function_call_output");
            expect(result.output).toBe(JSON.stringify({ data: "value" }));
        });

        it("should stringify complex results", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            const complexResult = {
                items: [1, 2, 3],
                nested: { key: "value" },
                bool: true,
            };

            const result = agent.getToolCallResult("call-456", complexResult);

            expect(JSON.parse(result.output as string)).toEqual(complexResult);
        });

        it("should handle null results", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            const result = agent.getToolCallResult("call-null", null);

            expect(result.output).toBe("null");
        });

        it("should handle string results", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            const result = agent.getToolCallResult("call-str", "simple string");

            expect(result.output).toBe('"simple string"');
        });
    });

    describe("invoke() with threadId", () => {
        it("should throw error for non-existent threadId", async () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            await expect(
                agent.invoke("Hello", "nonexistent-thread")
            ).rejects.toThrow("Model instance with threadId nonexistent-thread not found");
        });

        it("should reuse model instance for existing threadId", async () => {
            const mockModel = new MockModel();
            const mockFactory = new MockModelFactory(mockModel);

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: mockFactory,
            });

            // First invoke creates the model
            const model1 = await agent.invoke("First message");
            const threadId = model1.threadId;

            // Second invoke with same threadId should reuse
            const model2 = await agent.invoke("Second message", threadId);

            expect(model1).toBe(model2);
            expect(mockModel.invokedWith.length).toBe(2);
        });
    });

    describe("triggers and outcomes", () => {
        it("should initialize with default ApiTrigger when no triggers provided", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            expect(agent.triggers).toHaveLength(1);
            expect(agent.triggers[0].triggerType).toBe("api");
        });

        it("should initialize with default StreamingOutcome when no outcomes provided", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            expect(agent.outcomes).toHaveLength(1);
            expect(agent.outcomes[0]).toBeDefined();
        });

        it("should store provided triggers", () => {
            const trigger = { 
                name: "test", 
                triggerType: "event", 
                triggerConfig: {}, 
                activate: jest.fn(), 
                validate: jest.fn(),
                outcome: undefined,
                bindOutcome: jest.fn().mockReturnThis(),
            };

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
                triggers: [trigger],
            });

            expect(agent.triggers).toHaveLength(1);
            expect(agent.triggers[0]).toBe(trigger);
        });

        it("should store provided outcomes", () => {
            const outcome = { send: jest.fn(), bindToRequest: jest.fn() };

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
                outcomes: [outcome],
            });

            expect(agent.outcomes).toHaveLength(1);
            expect(agent.outcomes[0]).toBe(outcome);
        });
    });

    describe("tools from props", () => {
        it("should accept standalone tools in addition to toolkit tools", () => {
            const toolkitTool = new MockTool("toolkit_tool", "From toolkit");
            const standaloneTool = new MockTool("standalone_tool", "Standalone");
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [toolkitTool],
            });

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit],
                systemPrompt: "Test",
                model: new MockModelFactory(),
                tools: [standaloneTool],
            });

            expect(agent.tools).toContain(standaloneTool);
            expect(agent.tools).toContain(toolkitTool);
        });

        it("should be able to find both toolkit and standalone tools", () => {
            const toolkitTool = new MockTool("toolkit_tool", "From toolkit");
            const standaloneTool = new MockTool("standalone_tool", "Standalone");
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [toolkitTool],
            });

            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit],
                systemPrompt: "Test",
                model: new MockModelFactory(),
                tools: [standaloneTool],
            });

            expect(agent.getTool("toolkit_tool")).toBe(toolkitTool);
            expect(agent.getTool("standalone_tool")).toBe(standaloneTool);
        });
    });
});

