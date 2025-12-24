import { Agent, AgentProps } from "../src/agent";
import { Model } from "../src/model";
import { Secret } from "../src/secrets";
import { IModelFactory, IModelStreamer, ModelContent, IModel, ModelResponse, ModelResponseData } from "../src/models/types";
import { Toolkit } from "../src/tools/toolkit";
import { ITool, Schema, ToolArgBuilder } from "../src/tools/tool";

// Mock tool implementation for testing
class MockTool implements ITool {
    public name: string;
    public description: string;
    public schema: Schema;

    constructor(name: string, description: string, args: Record<string, ToolArgBuilder> = {}) {
        this.name = name;
        this.description = description;
        this.schema = Schema.objectValue(args);
    }

    execute(_args: Record<string, unknown>): unknown {
        return { executed: true };
    }
}

// Mock streamer for testing
class MockStreamer implements IModelStreamer {
    public receivedResponses: ModelResponse[] = [];
    public receivedModels: IModel[] = [];

    async recieve(model: IModel, content: ModelResponse): Promise<void> {
        this.receivedModels.push(model);
        this.receivedResponses.push(content);
    }
}

// Mock model for testing without real API calls
class MockModel implements IModel {
    public readonly threadId?: string = "mock-thread-id";

    async invoke(prompt: ModelContent[], streamer: IModelStreamer): Promise<void> {
        await streamer.recieve(this, {
            eventName: "response.output_text.done",
            data: {
                typeName: "output_text",
                outputText: {
                    outputType: "output_text",
                    text: `Mock response to: ${prompt[0]?.content || "empty"}`,
                },
            },
        });
    }
}

// Mock factory for testing
class MockModelFactory implements IModelFactory {
    async create(_props: { systemPrompt: string }): Promise<IModel> {
        return new MockModel();
    }
}

describe("Agent", () => {
    let defaultModelFactory: IModelFactory;
    let defaultToolkit: Toolkit;

    beforeEach(() => {
        defaultModelFactory = Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY"));
        defaultToolkit = new Toolkit({ name: "TestToolkit", description: "A test toolkit" });
    });

    describe("constructor", () => {
        it("should create an Agent with all required properties", () => {
            const props: AgentProps = {
                name: "TestAgent",
                toolkits: [defaultToolkit],
                systemPrompt: "You are a helpful assistant.",
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.name).toBe("TestAgent");
            expect(agent.toolkits).toEqual([defaultToolkit]);
            expect(agent.systemPrompt).toBe("You are a helpful assistant.");
            expect(agent.model).toBe(defaultModelFactory);
        });

        it("should create an Agent with empty toolkits array", () => {
            const props: AgentProps = {
                name: "EmptyToolkitsAgent",
                toolkits: [],
                systemPrompt: "No tools available.",
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.name).toBe("EmptyToolkitsAgent");
            expect(agent.toolkits).toEqual([]);
            expect(agent.toolkits).toHaveLength(0);
        });

        it("should create an Agent with multiple toolkits", () => {
            const toolkit1 = new Toolkit({ name: "Toolkit1" });
            const toolkit2 = new Toolkit({ name: "Toolkit2" });
            const toolkit3 = new Toolkit({ name: "Toolkit3" });

            const props: AgentProps = {
                name: "MultiToolkitAgent",
                toolkits: [toolkit1, toolkit2, toolkit3],
                systemPrompt: "You have many tools.",
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.toolkits).toHaveLength(3);
            expect(agent.toolkits[0]).toBe(toolkit1);
            expect(agent.toolkits[1]).toBe(toolkit2);
            expect(agent.toolkits[2]).toBe(toolkit3);
        });

        it("should create an Agent with empty string name", () => {
            const props: AgentProps = {
                name: "",
                toolkits: [],
                systemPrompt: "Empty name agent",
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.name).toBe("");
        });

        it("should create an Agent with empty system prompt", () => {
            const props: AgentProps = {
                name: "NoPromptAgent",
                toolkits: [],
                systemPrompt: "",
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.systemPrompt).toBe("");
        });

        it("should create an Agent with multiline system prompt", () => {
            const multilinePrompt = `You are a helpful assistant.
You should always be polite.
Never reveal sensitive information.`;

            const props: AgentProps = {
                name: "MultilinePromptAgent",
                toolkits: [],
                systemPrompt: multilinePrompt,
                model: defaultModelFactory,
            };

            const agent = new Agent(props);

            expect(agent.systemPrompt).toBe(multilinePrompt);
            expect(agent.systemPrompt).toContain("\n");
        });
    });

    describe("toolkits with tools", () => {
        it("should create an Agent with toolkits containing tools", () => {
            const tool1 = new MockTool("tool1", "First tool");
            const tool2 = new MockTool("tool2", "Second tool");
            const toolkit = new Toolkit({
                name: "ToolsToolkit",
                tools: [tool1, tool2],
            });

            const agent = new Agent({
                name: "ToolsAgent",
                toolkits: [toolkit],
                systemPrompt: "Agent with tools",
                model: defaultModelFactory,
            });

            expect(agent.toolkits).toHaveLength(1);
            expect(agent.toolkits[0].tools).toHaveLength(2);
            expect(agent.toolkits[0].tools[0].name).toBe("tool1");
            expect(agent.toolkits[0].tools[1].name).toBe("tool2");
        });

        it("should access tools through toolkit chain", () => {
            const tool = new MockTool("searchTool", "Search for items");
            const toolkit = new Toolkit({
                name: "SearchToolkit",
                tools: [tool],
            });

            const agent = new Agent({
                name: "SearchAgent",
                toolkits: [toolkit],
                systemPrompt: "Agent for searching",
                model: defaultModelFactory,
            });

            const result = agent.toolkits[0].tools[0].execute({});

            expect(result).toEqual({ executed: true });
        });
    });

    describe("model property", () => {
        it("should store the model factory reference correctly", () => {
            const secret = Secret.fromEnv("ANTHROPIC_API_KEY");
            const modelFactory = Model.openAI("claude-3-opus", secret);

            const agent = new Agent({
                name: "ClaudeAgent",
                toolkits: [],
                systemPrompt: "Using Claude",
                model: modelFactory,
            });

            expect(agent.model).toBe(modelFactory);
            expect(typeof agent.model.create).toBe("function");
        });

        it("should allow different agents to use different model factories", () => {
            const gptFactory = Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY"));
            const claudeFactory = Model.openAI("claude-3", Secret.fromEnv("ANTHROPIC_KEY"));

            const gptAgent = new Agent({
                name: "GPTAgent",
                toolkits: [],
                systemPrompt: "GPT based",
                model: gptFactory,
            });

            const claudeAgent = new Agent({
                name: "ClaudeAgent",
                toolkits: [],
                systemPrompt: "Claude based",
                model: claudeFactory,
            });

            expect(gptAgent.model).toBe(gptFactory);
            expect(claudeAgent.model).toBe(claudeFactory);
            expect(gptAgent.model).not.toBe(claudeAgent.model);
        });

        it("should allow multiple agents to share the same model factory", () => {
            const sharedFactory = Model.openAI("shared-model", Secret.fromEnv("SHARED_KEY"));

            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [],
                systemPrompt: "First agent",
                model: sharedFactory,
            });

            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [],
                systemPrompt: "Second agent",
                model: sharedFactory,
            });

            expect(agent1.model).toBe(agent2.model);
            expect(agent1.model).toBe(sharedFactory);
        });
    });

    describe("invoke method", () => {
        it("should have an invoke method", () => {
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: new MockModelFactory(),
            });

            expect(typeof agent.invoke).toBe("function");
        });

        it("should invoke with a prompt string", async () => {
            const model = new MockModel();
            class MockModelFactory2 implements IModelFactory {
                async create(_props: { systemPrompt: string }): Promise<IModel> {
                    return model;
                }
            }
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "You are helpful.",
                model: new MockModelFactory2(),
            });

            // Should not throw
            await expect(agent.invoke("Hello")).resolves.toEqual(model);
        });
    });

    describe("readonly properties", () => {
        it("should have readonly name property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: defaultModelFactory,
            });

            expect(agent.name).toBeDefined();
            expect(typeof agent.name).toBe("string");
        });

        it("should have readonly toolkits property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [defaultToolkit],
                systemPrompt: "Test",
                model: defaultModelFactory,
            });

            expect(agent.toolkits).toBeDefined();
            expect(Array.isArray(agent.toolkits)).toBe(true);
        });

        it("should have readonly systemPrompt property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Readonly prompt",
                model: defaultModelFactory,
            });

            expect(agent.systemPrompt).toBeDefined();
            expect(typeof agent.systemPrompt).toBe("string");
        });

        it("should have readonly model property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: defaultModelFactory,
            });

            expect(agent.model).toBeDefined();
            expect(typeof agent.model.create).toBe("function");
        });
    });

    describe("AgentProps interface", () => {
        it("should accept valid AgentProps", () => {
            const props: AgentProps = {
                name: "ValidAgent",
                toolkits: [],
                systemPrompt: "Valid prompt",
                model: defaultModelFactory,
            };

            expect(props.name).toBe("ValidAgent");
            expect(props.toolkits).toEqual([]);
            expect(props.systemPrompt).toBe("Valid prompt");
            expect(props.model).toBe(defaultModelFactory);
        });
    });

    describe("multiple agents independence", () => {
        it("should create independent agent instances", () => {
            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [defaultToolkit],
                systemPrompt: "Prompt 1",
                model: defaultModelFactory,
            });

            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [],
                systemPrompt: "Prompt 2",
                model: Model.openAI("other-model", Secret.fromEnv("OTHER_KEY")),
            });

            expect(agent1.name).not.toBe(agent2.name);
            expect(agent1.systemPrompt).not.toBe(agent2.systemPrompt);
            expect(agent1.toolkits.length).not.toBe(agent2.toolkits.length);
            expect(agent1.model).not.toBe(agent2.model);
        });
    });

    describe("model factory with IModelFactory interface", () => {
        it("should work with model factory created from Model.openAI", () => {
            const factory = Model.openAI("gpt-4", Secret.fromEnv("OPENAI_KEY"));
            const agent = new Agent({
                name: "FactoryModelAgent",
                toolkits: [],
                systemPrompt: "Uses factory model",
                model: factory,
            });

            expect(agent.model).toBe(factory);
            expect(typeof agent.model.create).toBe("function");
        });

        it("should work with any IModelFactory implementation", async () => {
            // Create a custom IModelFactory implementation for testing
            const customFactory: IModelFactory = {
                create: async (_props) => {
                    return {
                        threadId: "custom-thread-id",
                        invoke: async (_prompt, streamer) => {
                            const model: IModel = { threadId: "custom", invoke: async () => {} } as IModel;
                            await streamer.recieve(model, {
                                eventName: "response.output_text.done",
                                data: { typeName: "output_text", outputText: { outputType: "output_text", text: "Custom response" } },
                            });
                        },
                    };
                },
            };

            const agent = new Agent({
                name: "CustomModelAgent",
                toolkits: [],
                systemPrompt: "Uses custom model",
                model: customFactory,
            });

            const model = await agent.model.create({ systemPrompt: agent.systemPrompt });
            const streamer = new MockStreamer();
            await model.invoke([{ content: "Hello", role: "user" }], streamer);

            const response = streamer.receivedResponses[0];
            expect(Array.isArray(response.data)).toBe(false);
            const data = response.data as ModelResponseData;
            expect(data.typeName).toBe("output_text");
            if (data.typeName === "output_text") {
                expect(data.outputText?.text).toBe("Custom response");
            }
        });

        it("should be able to create model instances and invoke with streamer using mock factory", async () => {
            const mockFactory = new MockModelFactory();
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "You are a helpful assistant.",
                model: mockFactory,
            });

            const model = await agent.model.create({ systemPrompt: agent.systemPrompt });
            const streamer = new MockStreamer();

            await model.invoke([{ content: "Hello", role: "user" }], streamer);

            expect(streamer.receivedResponses.length).toBeGreaterThan(0);
            expect(streamer.receivedModels[0]).toBe(model);
            expect(model.threadId).toBe("mock-thread-id");
        });
    });
});
