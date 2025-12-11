import { Toolkit, ToolkitProps } from "../src/tools/toolkit";
import { ITool, ToolArg } from "../src/tools/tool";

// Mock tool implementation for testing
class MockTool implements ITool {
    public name: string;
    public description: string;
    private args: Record<string, ToolArg>;
    private executeResult: unknown;

    constructor(
        name: string,
        description: string,
        args: Record<string, ToolArg> = {},
        executeResult: unknown = {}
    ) {
        this.name = name;
        this.description = description;
        this.args = args;
        this.executeResult = executeResult;
    }

    execute(_args: Record<string, unknown>): unknown {
        return this.executeResult;
    }

    produceArgs(): Record<string, ToolArg> {
        return this.args;
    }
}

describe("Toolkit", () => {
    describe("constructor", () => {
        it("should create a Toolkit with name only", () => {
            const props: ToolkitProps = {
                name: "TestToolkit",
            };

            const toolkit = new Toolkit(props);

            expect(toolkit.name).toBe("TestToolkit");
            expect(toolkit.description).toBeUndefined();
            expect(toolkit.tools).toEqual([]);
        });

        it("should create a Toolkit with name and description", () => {
            const props: ToolkitProps = {
                name: "MyToolkit",
                description: "A useful toolkit for testing",
            };

            const toolkit = new Toolkit(props);

            expect(toolkit.name).toBe("MyToolkit");
            expect(toolkit.description).toBe("A useful toolkit for testing");
            expect(toolkit.tools).toEqual([]);
        });

        it("should create a Toolkit with name, description, and tools", () => {
            const tool1 = new MockTool("tool1", "First tool");
            const tool2 = new MockTool("tool2", "Second tool");

            const props: ToolkitProps = {
                name: "FullToolkit",
                description: "A complete toolkit",
                tools: [tool1, tool2],
            };

            const toolkit = new Toolkit(props);

            expect(toolkit.name).toBe("FullToolkit");
            expect(toolkit.description).toBe("A complete toolkit");
            expect(toolkit.tools).toHaveLength(2);
            expect(toolkit.tools[0]).toBe(tool1);
            expect(toolkit.tools[1]).toBe(tool2);
        });

        it("should handle empty tools array", () => {
            const props: ToolkitProps = {
                name: "EmptyToolkit",
                tools: [],
            };

            const toolkit = new Toolkit(props);

            expect(toolkit.tools).toEqual([]);
            expect(toolkit.tools).toHaveLength(0);
        });

        it("should create toolkit with empty string name", () => {
            const toolkit = new Toolkit({ name: "" });

            expect(toolkit.name).toBe("");
        });

        it("should use empty array when tools is undefined", () => {
            const toolkit = new Toolkit({ name: "Test" });

            expect(toolkit.tools).toBeDefined();
            expect(Array.isArray(toolkit.tools)).toBe(true);
            expect(toolkit.tools).toHaveLength(0);
        });
    });

    describe("addTool()", () => {
        it("should add a tool to an empty toolkit", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });
            const tool = new MockTool("newTool", "A new tool");

            toolkit.addTool(tool);

            expect(toolkit.tools).toHaveLength(1);
            expect(toolkit.tools[0]).toBe(tool);
        });

        it("should add a tool to a toolkit with existing tools", () => {
            const existingTool = new MockTool("existing", "Existing tool");
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [existingTool],
            });
            const newTool = new MockTool("newTool", "A new tool");

            toolkit.addTool(newTool);

            expect(toolkit.tools).toHaveLength(2);
            expect(toolkit.tools[0]).toBe(existingTool);
            expect(toolkit.tools[1]).toBe(newTool);
        });

        it("should add multiple tools sequentially", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });
            const tool1 = new MockTool("tool1", "Tool 1");
            const tool2 = new MockTool("tool2", "Tool 2");
            const tool3 = new MockTool("tool3", "Tool 3");

            toolkit.addTool(tool1);
            toolkit.addTool(tool2);
            toolkit.addTool(tool3);

            expect(toolkit.tools).toHaveLength(3);
            expect(toolkit.tools).toEqual([tool1, tool2, tool3]);
        });

        it("should allow adding duplicate tools", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });
            const tool = new MockTool("duplicateTool", "A tool");

            toolkit.addTool(tool);
            toolkit.addTool(tool);

            expect(toolkit.tools).toHaveLength(2);
            expect(toolkit.tools[0]).toBe(tool);
            expect(toolkit.tools[1]).toBe(tool);
        });

        it("should allow adding tools with same name but different instances", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });
            const tool1 = new MockTool("sameName", "Tool 1");
            const tool2 = new MockTool("sameName", "Tool 2");

            toolkit.addTool(tool1);
            toolkit.addTool(tool2);

            expect(toolkit.tools).toHaveLength(2);
            expect(toolkit.tools[0].description).toBe("Tool 1");
            expect(toolkit.tools[1].description).toBe("Tool 2");
        });
    });

    describe("integration with ITool", () => {
        it("should work with tools that have complex args", () => {
            const complexArgs: Record<string, ToolArg> = {
                query: {
                    name: "query",
                    argType: "string",
                    description: "Search query",
                    required: true,
                    defaultValue: undefined,
                },
                limit: {
                    name: "limit",
                    argType: "number",
                    description: "Max results",
                    required: false,
                    defaultValue: 10,
                },
            };
            const tool = new MockTool("searchTool", "Search for items", complexArgs);
            const toolkit = new Toolkit({
                name: "SearchToolkit",
                tools: [tool],
            });

            expect(toolkit.tools[0].produceArgs()).toEqual(complexArgs);
        });

        it("should work with tools that return results", () => {
            const expectedResult = { found: true, items: ["a", "b", "c"] };
            const tool = new MockTool("searchTool", "Search tool", {}, expectedResult);
            const toolkit = new Toolkit({
                name: "TestToolkit",
                tools: [tool],
            });

            const result = toolkit.tools[0].execute({});

            expect(result).toEqual(expectedResult);
        });
    });

    describe("readonly properties", () => {
        it("should have readonly name property", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });

            expect(toolkit.name).toBeDefined();
            expect(typeof toolkit.name).toBe("string");
        });

        it("should have readonly description property", () => {
            const toolkit = new Toolkit({
                name: "TestToolkit",
                description: "Test description",
            });

            expect(toolkit.description).toBe("Test description");
        });

        it("should have readonly tools array reference", () => {
            const toolkit = new Toolkit({ name: "TestToolkit" });

            expect(toolkit.tools).toBeDefined();
            expect(Array.isArray(toolkit.tools)).toBe(true);
        });
    });

    describe("ToolkitProps interface", () => {
        it("should accept valid ToolkitProps with required fields only", () => {
            const props: ToolkitProps = {
                name: "MinimalToolkit",
            };

            expect(props.name).toBe("MinimalToolkit");
            expect(props.description).toBeUndefined();
            expect(props.tools).toBeUndefined();
        });

        it("should accept valid ToolkitProps with all fields", () => {
            const tool = new MockTool("tool", "A tool");
            const props: ToolkitProps = {
                name: "CompleteToolkit",
                description: "A complete toolkit",
                tools: [tool],
            };

            expect(props.name).toBe("CompleteToolkit");
            expect(props.description).toBe("A complete toolkit");
            expect(props.tools).toHaveLength(1);
        });
    });
});

