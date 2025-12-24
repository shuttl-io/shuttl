import { App } from "../src/app";
import { Agent } from "../src/agent";
import { Model } from "../src/model";
import { Secret } from "../src/secrets";
import { IModelFactory, IModelStreamer, ModelContent, IModel, ModelResponse } from "../src/models/types";
import { Toolkit } from "../src/tools/toolkit";
import { IServer } from "../src/Server";
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

// Mock server implementation for testing
class MockServer implements IServer {
    public acceptedApp: unknown | null = null;
    public started: boolean = false;
    public stopped: boolean = false;
    public acceptCalls: number = 0;
    public startCalls: number = 0;
    public stopCalls: number = 0;

    accept(app: unknown): void {
        this.acceptedApp = app;
        this.acceptCalls++;
    }

    async start(): Promise<void> {
        this.started = true;
        this.startCalls++;
    }

    async stop(): Promise<void> {
        this.stopped = true;
        this.stopCalls++;
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

describe("App", () => {
    let mockServer: MockServer;
    let defaultModelFactory: IModelFactory;

    beforeEach(() => {
        mockServer = new MockServer();
        defaultModelFactory = Model.openAI("gpt-4", Secret.fromEnv("OPENAI_API_KEY"));
    });

    describe("constructor", () => {
        it("should create an App with name and server", () => {
            const app = new App("TestApp", mockServer);

            expect(app.name).toBe("TestApp");
            expect(app.server).toBe(mockServer);
            expect(app.agents).toEqual([]);
            expect(app.toolkits.size).toBe(0);
        });

        it("should call server.accept() during construction", () => {
            const app = new App("TestApp", mockServer);

            expect(mockServer.acceptCalls).toBe(1);
            expect(mockServer.acceptedApp).toBe(app);
        });

        it("should create an App with empty string name", () => {
            const app = new App("", mockServer);

            expect(app.name).toBe("");
        });

        it("should initialize agents as empty array", () => {
            const app = new App("TestApp", mockServer);

            expect(app.agents).toBeDefined();
            expect(Array.isArray(app.agents)).toBe(true);
            expect(app.agents).toHaveLength(0);
        });

        it("should initialize toolkits as empty Set", () => {
            const app = new App("TestApp", mockServer);

            expect(app.toolkits).toBeDefined();
            expect(app.toolkits).toBeInstanceOf(Set);
            expect(app.toolkits.size).toBe(0);
        });
    });

    describe("addAgent()", () => {
        it("should add an agent to the app", () => {
            const app = new App("TestApp", mockServer);
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: defaultModelFactory,
            });

            app.addAgent(agent);

            expect(app.agents).toHaveLength(1);
            expect(app.agents[0]).toBe(agent);
        });

        it("should add multiple agents", () => {
            const app = new App("TestApp", mockServer);
            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [],
                systemPrompt: "Prompt 1",
                model: defaultModelFactory,
            });
            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [],
                systemPrompt: "Prompt 2",
                model: defaultModelFactory,
            });

            app.addAgent(agent1);
            app.addAgent(agent2);

            expect(app.agents).toHaveLength(2);
            expect(app.agents[0]).toBe(agent1);
            expect(app.agents[1]).toBe(agent2);
        });

        it("should add agent toolkits to app toolkits", () => {
            const app = new App("TestApp", mockServer);
            const toolkit = new Toolkit({ name: "TestToolkit" });
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit],
                systemPrompt: "Test prompt",
                model: defaultModelFactory,
            });

            app.addAgent(agent);

            expect(app.toolkits.size).toBe(1);
            expect(app.toolkits.has(toolkit)).toBe(true);
        });

        it("should add multiple toolkits from agent", () => {
            const app = new App("TestApp", mockServer);
            const toolkit1 = new Toolkit({ name: "Toolkit1" });
            const toolkit2 = new Toolkit({ name: "Toolkit2" });
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [toolkit1, toolkit2],
                systemPrompt: "Test prompt",
                model: defaultModelFactory,
            });

            app.addAgent(agent);

            expect(app.toolkits.size).toBe(2);
            expect(app.toolkits.has(toolkit1)).toBe(true);
            expect(app.toolkits.has(toolkit2)).toBe(true);
        });

        it("should not duplicate toolkits when adding same toolkit from multiple agents", () => {
            const app = new App("TestApp", mockServer);
            const sharedToolkit = new Toolkit({ name: "SharedToolkit" });
            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [sharedToolkit],
                systemPrompt: "Prompt 1",
                model: defaultModelFactory,
            });
            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [sharedToolkit],
                systemPrompt: "Prompt 2",
                model: defaultModelFactory,
            });

            app.addAgent(agent1);
            app.addAgent(agent2);

            expect(app.toolkits.size).toBe(1);
            expect(app.toolkits.has(sharedToolkit)).toBe(true);
        });

        it("should allow adding the same agent twice", () => {
            const app = new App("TestApp", mockServer);
            const agent = new Agent({
                name: "TestAgent",
                toolkits: [],
                systemPrompt: "Test prompt",
                model: defaultModelFactory,
            });

            app.addAgent(agent);
            app.addAgent(agent);

            expect(app.agents).toHaveLength(2);
            expect(app.agents[0]).toBe(agent);
            expect(app.agents[1]).toBe(agent);
        });

        it("should handle agent with no toolkits", () => {
            const app = new App("TestApp", mockServer);
            const agent = new Agent({
                name: "NoToolkitAgent",
                toolkits: [],
                systemPrompt: "No toolkits",
                model: defaultModelFactory,
            });

            app.addAgent(agent);

            expect(app.agents).toHaveLength(1);
            expect(app.toolkits.size).toBe(0);
        });
    });

    describe("addToolkit()", () => {
        it("should add a toolkit to the app", () => {
            const app = new App("TestApp", mockServer);
            const toolkit = new Toolkit({ name: "TestToolkit" });

            app.addToolkit(toolkit);

            expect(app.toolkits.size).toBe(1);
            expect(app.toolkits.has(toolkit)).toBe(true);
        });

        it("should add multiple toolkits", () => {
            const app = new App("TestApp", mockServer);
            const toolkit1 = new Toolkit({ name: "Toolkit1" });
            const toolkit2 = new Toolkit({ name: "Toolkit2" });

            app.addToolkit(toolkit1);
            app.addToolkit(toolkit2);

            expect(app.toolkits.size).toBe(2);
            expect(app.toolkits.has(toolkit1)).toBe(true);
            expect(app.toolkits.has(toolkit2)).toBe(true);
        });

        it("should not duplicate toolkit when adding same toolkit twice", () => {
            const app = new App("TestApp", mockServer);
            const toolkit = new Toolkit({ name: "TestToolkit" });

            app.addToolkit(toolkit);
            app.addToolkit(toolkit);

            expect(app.toolkits.size).toBe(1);
        });

        it("should add toolkit with tools", () => {
            const app = new App("TestApp", mockServer);
            const tool = new MockTool("testTool", "A test tool");
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [tool],
            });

            app.addToolkit(toolkit);

            expect(app.toolkits.size).toBe(1);
            const addedToolkit = Array.from(app.toolkits)[0];
            expect(addedToolkit.tools).toHaveLength(1);
            expect(addedToolkit.tools[0].name).toBe("testTool");
        });
    });

    describe("serve()", () => {
        it("should call server.start()", () => {
            const app = new App("TestApp", mockServer);

            app.serve();

            expect(mockServer.startCalls).toBe(1);
            expect(mockServer.started).toBe(true);
        });

        it("should be callable multiple times", () => {
            const app = new App("TestApp", mockServer);

            app.serve();
            app.serve();
            app.serve();

            expect(mockServer.startCalls).toBe(3);
        });
    });

    describe("integration scenarios", () => {
        it("should work with complete app setup", () => {
            const app = new App("CompleteApp", mockServer);

            // Create tools
            const searchTool = new MockTool("search", "Search for items");
            const createTool = new MockTool("create", "Create new item");

            // Create toolkits
            const searchToolkit = new Toolkit({
                name: "SearchToolkit",
                description: "Tools for searching",
                tools: [searchTool],
            });
            const crudToolkit = new Toolkit({
                name: "CRUDToolkit",
                description: "CRUD operations",
                tools: [createTool],
            });

            // Create agents
            const searchAgent = new Agent({
                name: "SearchAgent",
                toolkits: [searchToolkit],
                systemPrompt: "You are a search agent",
                model: defaultModelFactory,
            });
            const crudAgent = new Agent({
                name: "CRUDAgent",
                toolkits: [crudToolkit],
                systemPrompt: "You are a CRUD agent",
                model: defaultModelFactory,
            });

            // Add agents to app
            app.addAgent(searchAgent);
            app.addAgent(crudAgent);

            // Add additional toolkit directly
            const utilToolkit = new Toolkit({ name: "UtilToolkit" });
            app.addToolkit(utilToolkit);

            // Verify state
            expect(app.name).toBe("CompleteApp");
            expect(app.agents).toHaveLength(2);
            expect(app.toolkits.size).toBe(3);
            expect(mockServer.acceptedApp).toBe(app);
        });

        it("should maintain references correctly", () => {
            const app = new App("RefApp", mockServer);
            const toolkit = new Toolkit({ name: "SharedToolkit" });
            const agent = new Agent({
                name: "RefAgent",
                toolkits: [toolkit],
                systemPrompt: "Test",
                model: defaultModelFactory,
            });

            app.addAgent(agent);
            app.addToolkit(toolkit);

            // Agent toolkit and app toolkit should be same reference
            expect(app.agents[0].toolkits[0]).toBe(toolkit);
            expect(Array.from(app.toolkits)[0]).toBe(toolkit);
        });

        it("should allow iteration over toolkits", () => {
            const app = new App("IterApp", mockServer);
            const toolkit1 = new Toolkit({ name: "Toolkit1" });
            const toolkit2 = new Toolkit({ name: "Toolkit2" });

            app.addToolkit(toolkit1);
            app.addToolkit(toolkit2);

            const toolkitNames: string[] = [];
            app.toolkits.forEach((toolkit) => {
                toolkitNames.push(toolkit.name);
            });

            expect(toolkitNames).toContain("Toolkit1");
            expect(toolkitNames).toContain("Toolkit2");
        });
    });

    describe("readonly properties", () => {
        it("should have readonly name property", () => {
            const app = new App("ReadonlyApp", mockServer);

            expect(app.name).toBeDefined();
            expect(typeof app.name).toBe("string");
        });

        it("should have agents array that can be read", () => {
            const app = new App("TestApp", mockServer);

            expect(app.agents).toBeDefined();
            expect(Array.isArray(app.agents)).toBe(true);
        });

        it("should have toolkits Set that can be read", () => {
            const app = new App("TestApp", mockServer);

            expect(app.toolkits).toBeDefined();
            expect(app.toolkits).toBeInstanceOf(Set);
        });

        it("should have server that can be read", () => {
            const app = new App("TestApp", mockServer);

            expect(app.server).toBeDefined();
            expect(app.server).toBe(mockServer);
        });
    });

    describe("IServer interface compliance", () => {
        it("should work with any IServer implementation", () => {
            class CustomServer implements IServer {
                public accepted = false;
                public running = false;

                accept(_app: unknown): void {
                    this.accepted = true;
                }

                async start(): Promise<void> {
                    this.running = true;
                }

                async stop(): Promise<void> {
                    this.running = false;
                }
            }

            const customServer = new CustomServer();
            const app = new App("CustomApp", customServer);

            expect(customServer.accepted).toBe(true);
            
            app.serve();
            expect(customServer.running).toBe(true);
        });
    });
});

describe("Model factory methods", () => {
    describe("Model.openAI()", () => {
        it("should create an OpenAI model factory with env secret", () => {
            const secret = Secret.fromEnv("OPENAI_API_KEY");
            const factory = Model.openAI("gpt-4", secret);

            expect(factory).toBeDefined();
            expect(typeof factory.create).toBe("function");
        });

        it("should create model factories with different identifiers", () => {
            const secret = Secret.fromEnv("OPENAI_API_KEY");
            const gpt4 = Model.openAI("gpt-4", secret);
            const gpt35 = Model.openAI("gpt-3.5-turbo", secret);

            expect(gpt4).toBeDefined();
            expect(gpt35).toBeDefined();
        });

        it("should create model instances from mock factory and invoke with streamer", async () => {
            const mockFactory = new MockModelFactory();

            const model = await mockFactory.create({
                systemPrompt: "You are a helpful assistant.",
            });

            const streamer = new MockStreamer();
            await model.invoke([{ content: "Hello", role: "user" }], streamer);

            expect(streamer.receivedResponses.length).toBeGreaterThan(0);
            expect(streamer.receivedModels[0]).toBe(model);
            expect(model.threadId).toBe("mock-thread-id");
        });
    });
});
