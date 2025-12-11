import { Agent, AgentProps } from "../src/agent";
import { Model } from "../src/model";
import { Toolkit } from "../src/tools/toolkit";
import { ITool, ToolArg } from "../src/tools/tool";

// Mock tool implementation for testing
class MockTool implements ITool {
    public name: string;
    public description: string;

    constructor(name: string, description: string) {
        this.name = name;
        this.description = description;
    }

    execute(_args: Record<string, unknown>): unknown {
        return { executed: true };
    }

    produceArgs(): Record<string, ToolArg> {
        return {};
    }
}

describe("Agent", () => {
    let defaultModel: Model;
    let defaultToolkit: Toolkit;

    beforeEach(() => {
        defaultModel = new Model({ identifier: "gpt-4", key: "test-key" });
        defaultToolkit = new Toolkit({ name: "TestToolkit", description: "A test toolkit" });
    });

    describe("constructor", () => {
        it("should create an Agent with all required properties", () => {
            const props: AgentProps = {
                name: "TestAgent",
                toolkits: [defaultToolkit],
                systemPrompt: "You are a helpful assistant.",
                model: defaultModel,
            };

            const agent = new Agent(props);

            expect(agent.name).toBe("TestAgent");
            expect(agent.toolkits).toEqual([defaultToolkit]);
            expect(agent.systemPrompt).toBe("You are a helpful assistant.");
            expect(agent.model).toBe(defaultModel);
        });

        it("should create an Agent with empty toolkits array", () => {
            const props: AgentProps = {
                name: "EmptyToolkitsAgent",
                toolkits: [],
                systemPrompt: "No tools available.",
                model: defaultModel,
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
                model: defaultModel,
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
                model: defaultModel,
            };

            const agent = new Agent(props);

            expect(agent.name).toBe("");
        });

        it("should create an Agent with empty system prompt", () => {
            const props: AgentProps = {
                name: "NoPromptAgent",
                toolkits: [],
                systemPrompt: "",
                model: defaultModel,
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
                model: defaultModel,
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
                model: defaultModel,
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
                model: defaultModel,
            });

            const result = agent.toolkits[0].tools[0].execute({});

            expect(result).toEqual({ executed: true });
        });
    });

    describe("model property", () => {
        it("should store the model reference correctly", () => {
            const model = new Model({ identifier: "claude-3-opus", key: "anthropic-key" });

            const agent = new Agent({
                name: "ClaudeAgent",
                toolkits: [],
                systemPrompt: "Using Claude",
                model,
            });

            expect(agent.model).toBe(model);
            expect(agent.model.identifier).toBe("claude-3-opus");
            expect(agent.model.key).toBe("anthropic-key");
        });

        it("should allow different agents to use different models", () => {
            const gptModel = new Model({ identifier: "gpt-4", key: "openai-key" });
            const claudeModel = new Model({ identifier: "claude-3", key: "anthropic-key" });

            const gptAgent = new Agent({
                name: "GPTAgent",
                toolkits: [],
                systemPrompt: "GPT based",
                model: gptModel,
            });

            const claudeAgent = new Agent({
                name: "ClaudeAgent",
                toolkits: [],
                systemPrompt: "Claude based",
                model: claudeModel,
            });

            expect(gptAgent.model.identifier).toBe("gpt-4");
            expect(claudeAgent.model.identifier).toBe("claude-3");
        });

        it("should allow multiple agents to share the same model", () => {
            const sharedModel = new Model({ identifier: "shared-model", key: "shared-key" });

            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [],
                systemPrompt: "First agent",
                model: sharedModel,
            });

            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [],
                systemPrompt: "Second agent",
                model: sharedModel,
            });

            expect(agent1.model).toBe(agent2.model);
            expect(agent1.model).toBe(sharedModel);
        });
    });

    describe("readonly properties", () => {
        it("should have readonly name property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: defaultModel,
            });

            expect(agent.name).toBeDefined();
            expect(typeof agent.name).toBe("string");
        });

        it("should have readonly toolkits property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [defaultToolkit],
                systemPrompt: "Test",
                model: defaultModel,
            });

            expect(agent.toolkits).toBeDefined();
            expect(Array.isArray(agent.toolkits)).toBe(true);
        });

        it("should have readonly systemPrompt property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Readonly prompt",
                model: defaultModel,
            });

            expect(agent.systemPrompt).toBeDefined();
            expect(typeof agent.systemPrompt).toBe("string");
        });

        it("should have readonly model property", () => {
            const agent = new Agent({
                name: "ReadonlyAgent",
                toolkits: [],
                systemPrompt: "Test",
                model: defaultModel,
            });

            expect(agent.model).toBeDefined();
            expect(agent.model).toBeInstanceOf(Model);
        });
    });

    describe("AgentProps interface", () => {
        it("should accept valid AgentProps", () => {
            const props: AgentProps = {
                name: "ValidAgent",
                toolkits: [],
                systemPrompt: "Valid prompt",
                model: defaultModel,
            };

            expect(props.name).toBe("ValidAgent");
            expect(props.toolkits).toEqual([]);
            expect(props.systemPrompt).toBe("Valid prompt");
            expect(props.model).toBe(defaultModel);
        });
    });

    describe("multiple agents independence", () => {
        it("should create independent agent instances", () => {
            const agent1 = new Agent({
                name: "Agent1",
                toolkits: [defaultToolkit],
                systemPrompt: "Prompt 1",
                model: defaultModel,
            });

            const agent2 = new Agent({
                name: "Agent2",
                toolkits: [],
                systemPrompt: "Prompt 2",
                model: new Model({ identifier: "other-model", key: "other-key" }),
            });

            expect(agent1.name).not.toBe(agent2.name);
            expect(agent1.systemPrompt).not.toBe(agent2.systemPrompt);
            expect(agent1.toolkits.length).not.toBe(agent2.toolkits.length);
            expect(agent1.model).not.toBe(agent2.model);
        });
    });
});

