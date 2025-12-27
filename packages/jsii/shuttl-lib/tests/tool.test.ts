import { Schema, ToolArgBuilder, ITool } from "../src/tools/tool";

describe("ToolArgBuilder", () => {
    describe("constructor", () => {
        it("should create a ToolArgBuilder with all required properties", () => {
            const builder = new ToolArgBuilder("string", "A description", false);

            expect(builder.argType).toBe("string");
            expect(builder.description).toBe("A description");
            expect(builder.required).toBe(false);
            expect(builder.defaultValue).toBeUndefined();
            expect(builder.enumValues).toBeUndefined();
        });

        it("should create a ToolArgBuilder with required set to true", () => {
            const builder = new ToolArgBuilder("number", "A number field", true);

            expect(builder.argType).toBe("number");
            expect(builder.required).toBe(true);
        });

        it("should create a ToolArgBuilder with a default value", () => {
            const builder = new ToolArgBuilder("string", "With default", false, "default_value");

            expect(builder.defaultValue).toBe("default_value");
        });

        it("should create a ToolArgBuilder with enum values", () => {
            const enumVals = ["option1", "option2", "option3"];
            const builder = new ToolArgBuilder("string", "Enum field", false, undefined, enumVals);

            expect(builder.enumValues).toEqual(enumVals);
        });

        it("should support boolean argType", () => {
            const builder = new ToolArgBuilder("boolean", "A boolean field", false);

            expect(builder.argType).toBe("boolean");
        });

        it("should support object argType", () => {
            const builder = new ToolArgBuilder("object", "An object field", false);

            expect(builder.argType).toBe("object");
        });

        it("should support array argType", () => {
            const builder = new ToolArgBuilder("array", "An array field", false);

            expect(builder.argType).toBe("array");
        });
    });

    describe("isRequired()", () => {
        it("should set required to true and return the builder", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            const result = builder.isRequired();

            expect(result).toBe(builder);
            expect(builder.required).toBe(true);
        });

        it("should be chainable", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            const result = builder.isRequired();

            expect(result).toBeInstanceOf(ToolArgBuilder);
        });

        it("should work when already required", () => {
            const builder = new ToolArgBuilder("string", "Description", true);

            builder.isRequired();

            expect(builder.required).toBe(true);
        });
    });

    describe("defaultTo()", () => {
        it("should set defaultValue and return the builder", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            const result = builder.defaultTo("my_default");

            expect(result).toBe(builder);
            expect(builder.defaultValue).toBe("my_default");
        });

        it("should be chainable", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            const result = builder.defaultTo("default");

            expect(result).toBeInstanceOf(ToolArgBuilder);
        });

        it("should accept number default values", () => {
            const builder = new ToolArgBuilder("number", "Description", false);

            builder.defaultTo(42);

            expect(builder.defaultValue).toBe(42);
        });

        it("should accept boolean default values", () => {
            const builder = new ToolArgBuilder("boolean", "Description", false);

            builder.defaultTo(true);

            expect(builder.defaultValue).toBe(true);
        });

        it("should accept object default values", () => {
            const defaultObj = { key: "value" };
            const builder = new ToolArgBuilder("object", "Description", false);

            builder.defaultTo(defaultObj);

            expect(builder.defaultValue).toEqual(defaultObj);
        });

        it("should accept null as a default value", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            builder.defaultTo(null);

            expect(builder.defaultValue).toBeNull();
        });

        it("should accept array default values", () => {
            const builder = new ToolArgBuilder("array", "Description", false);

            builder.defaultTo([1, 2, 3]);

            expect(builder.defaultValue).toEqual([1, 2, 3]);
        });
    });

    describe("method chaining", () => {
        it("should allow chaining isRequired() and defaultTo()", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            builder.isRequired().defaultTo("default");

            expect(builder.required).toBe(true);
            expect(builder.defaultValue).toBe("default");
        });

        it("should allow chaining in any order", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            builder.defaultTo("value").isRequired();

            expect(builder.required).toBe(true);
            expect(builder.defaultValue).toBe("value");
        });

        it("should allow multiple defaultTo calls (last one wins)", () => {
            const builder = new ToolArgBuilder("string", "Description", false);

            builder.defaultTo("first").defaultTo("second");

            expect(builder.defaultValue).toBe("second");
        });
    });
});

describe("Schema", () => {
    describe("objectValue()", () => {
        it("should create a Schema with empty properties", () => {
            const schema = Schema.objectValue({});

            expect(schema.properties).toEqual({});
        });

        it("should create a Schema with single property", () => {
            const props = {
                name: new ToolArgBuilder("string", "User name", true),
            };

            const schema = Schema.objectValue(props);

            expect(schema.properties).toBe(props);
            expect(schema.properties.name.argType).toBe("string");
            expect(schema.properties.name.required).toBe(true);
        });

        it("should create a Schema with multiple properties", () => {
            const props = {
                name: new ToolArgBuilder("string", "User name", true),
                age: new ToolArgBuilder("number", "User age", false),
                active: new ToolArgBuilder("boolean", "Is active", false),
            };

            const schema = Schema.objectValue(props);

            expect(Object.keys(schema.properties)).toHaveLength(3);
            expect(schema.properties.name.argType).toBe("string");
            expect(schema.properties.age.argType).toBe("number");
            expect(schema.properties.active.argType).toBe("boolean");
        });

        it("should maintain reference to properties", () => {
            const nameBuilder = new ToolArgBuilder("string", "Name", false);
            const props = { name: nameBuilder };

            const schema = Schema.objectValue(props);

            expect(schema.properties.name).toBe(nameBuilder);
        });
    });

    describe("stringValue()", () => {
        it("should create a ToolArgBuilder with string type", () => {
            const builder = Schema.stringValue("A string field");

            expect(builder).toBeInstanceOf(ToolArgBuilder);
            expect(builder.argType).toBe("string");
            expect(builder.description).toBe("A string field");
            expect(builder.required).toBe(false);
        });

        it("should return a chainable ToolArgBuilder", () => {
            const builder = Schema.stringValue("Chainable string");

            const result = builder.isRequired();

            expect(result).toBe(builder);
            expect(builder.required).toBe(true);
        });

        it("should allow setting default value", () => {
            const builder = Schema.stringValue("With default").defaultTo("hello");

            expect(builder.defaultValue).toBe("hello");
        });
    });

    describe("numberValue()", () => {
        it("should create a ToolArgBuilder with number type", () => {
            const builder = Schema.numberValue("A number field");

            expect(builder).toBeInstanceOf(ToolArgBuilder);
            expect(builder.argType).toBe("number");
            expect(builder.description).toBe("A number field");
            expect(builder.required).toBe(false);
        });

        it("should return a chainable ToolArgBuilder", () => {
            const builder = Schema.numberValue("Chainable number");

            builder.isRequired().defaultTo(100);

            expect(builder.required).toBe(true);
            expect(builder.defaultValue).toBe(100);
        });
    });

    describe("booleanValue()", () => {
        it("should create a ToolArgBuilder with boolean type", () => {
            const builder = Schema.booleanValue("A boolean field");

            expect(builder).toBeInstanceOf(ToolArgBuilder);
            expect(builder.argType).toBe("boolean");
            expect(builder.description).toBe("A boolean field");
            expect(builder.required).toBe(false);
        });

        it("should return a chainable ToolArgBuilder", () => {
            const builder = Schema.booleanValue("Chainable boolean");

            builder.isRequired().defaultTo(false);

            expect(builder.required).toBe(true);
            expect(builder.defaultValue).toBe(false);
        });
    });

    describe("enumValue()", () => {
        it("should create a ToolArgBuilder with string type and enum values", () => {
            const enumVals = ["option1", "option2", "option3"];
            const builder = Schema.enumValue("Select an option", enumVals);

            expect(builder).toBeInstanceOf(ToolArgBuilder);
            expect(builder.argType).toBe("string");
            expect(builder.description).toBe("Select an option");
            expect(builder.enumValues).toEqual(enumVals);
            expect(builder.required).toBe(false);
        });

        it("should return a chainable ToolArgBuilder", () => {
            const builder = Schema.enumValue("Choice", ["a", "b"]);

            builder.isRequired();

            expect(builder.required).toBe(true);
        });

        it("should handle empty enum array", () => {
            const builder = Schema.enumValue("Empty enum", []);

            expect(builder.enumValues).toEqual([]);
        });

        it("should handle single enum value", () => {
            const builder = Schema.enumValue("Single option", ["only_one"]);

            expect(builder.enumValues).toEqual(["only_one"]);
        });

        it("should allow setting a default from enum values", () => {
            const builder = Schema.enumValue("Format", ["json", "xml", "csv"]).defaultTo("json");

            expect(builder.defaultValue).toBe("json");
        });
    });

    describe("integration: building complete tool schemas", () => {
        it("should create a schema for a search tool", () => {
            const schema = Schema.objectValue({
                query: Schema.stringValue("The search query").isRequired(),
                limit: Schema.numberValue("Maximum number of results").defaultTo(10),
                includeArchived: Schema.booleanValue("Include archived items"),
            });

            expect(schema.properties.query.required).toBe(true);
            expect(schema.properties.limit.defaultValue).toBe(10);
            expect(schema.properties.includeArchived.required).toBe(false);
        });

        it("should create a schema for a format converter", () => {
            const schema = Schema.objectValue({
                input: Schema.stringValue("The input data").isRequired(),
                format: Schema.enumValue("Output format", ["json", "xml", "csv"]).isRequired(),
                pretty: Schema.booleanValue("Pretty print output").defaultTo(true),
            });

            expect(schema.properties.format.enumValues).toEqual(["json", "xml", "csv"]);
            expect(schema.properties.format.required).toBe(true);
            expect(schema.properties.pretty.defaultValue).toBe(true);
        });

        it("should create a schema with nested complexity", () => {
            const schema = Schema.objectValue({
                name: Schema.stringValue("User name").isRequired(),
                email: Schema.stringValue("Email address").isRequired(),
                age: Schema.numberValue("User age"),
                role: Schema.enumValue("User role", ["admin", "user", "guest"]).defaultTo("user"),
                notifications: Schema.booleanValue("Enable notifications").defaultTo(true),
            });

            expect(Object.keys(schema.properties)).toHaveLength(5);
            expect(schema.properties.name.required).toBe(true);
            expect(schema.properties.email.required).toBe(true);
            expect(schema.properties.age.required).toBe(false);
            expect(schema.properties.role.defaultValue).toBe("user");
            expect(schema.properties.notifications.defaultValue).toBe(true);
        });
    });
});

describe("ITool interface usage", () => {
    // Test that the ITool interface works correctly with Schema
    class TestTool implements ITool {
        name = "test_tool";
        description = "A test tool";
        schema = Schema.objectValue({
            input: Schema.stringValue("Input text").isRequired(),
        });

        execute(args: Record<string, unknown>): unknown {
            return { received: args };
        }
    }

    it("should create a valid ITool implementation", () => {
        const tool = new TestTool();

        expect(tool.name).toBe("test_tool");
        expect(tool.description).toBe("A test tool");
        expect(tool.schema).toBeDefined();
    });

    it("should have accessible schema properties", () => {
        const tool = new TestTool();

        expect(tool.schema?.properties.input).toBeDefined();
        expect(tool.schema?.properties.input.argType).toBe("string");
        expect(tool.schema?.properties.input.required).toBe(true);
    });

    it("should execute with args", () => {
        const tool = new TestTool();
        const args = { input: "test value" };

        const result = tool.execute(args);

        expect(result).toEqual({ received: { input: "test value" } });
    });

    it("should work without schema (optional)", () => {
        class SchemalesssTool implements ITool {
            name = "simple_tool";
            description = "No schema needed";
            schema?: Schema = undefined;

            execute(args: Record<string, unknown>): unknown {
                return args;
            }
        }

        const tool = new SchemalesssTool();

        expect(tool.schema).toBeUndefined();
        expect(tool.execute({ foo: "bar" })).toEqual({ foo: "bar" });
    });
});

