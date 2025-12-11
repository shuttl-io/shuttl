import { Model, ModelProps } from "../src/model";

describe("Model", () => {
    describe("constructor", () => {
        it("should create a Model with identifier and key", () => {
            const props: ModelProps = {
                identifier: "gpt-4",
                key: "sk-test-key-12345",
            };

            const model = new Model(props);

            expect(model.identifier).toBe("gpt-4");
            expect(model.key).toBe("sk-test-key-12345");
        });

        it("should create a Model with different identifiers", () => {
            const props: ModelProps = {
                identifier: "claude-3-opus",
                key: "anthropic-key-abc",
            };

            const model = new Model(props);

            expect(model.identifier).toBe("claude-3-opus");
            expect(model.key).toBe("anthropic-key-abc");
        });

        it("should create a Model with empty strings", () => {
            const props: ModelProps = {
                identifier: "",
                key: "",
            };

            const model = new Model(props);

            expect(model.identifier).toBe("");
            expect(model.key).toBe("");
        });

        it("should create multiple independent Model instances", () => {
            const model1 = new Model({ identifier: "model-1", key: "key-1" });
            const model2 = new Model({ identifier: "model-2", key: "key-2" });

            expect(model1.identifier).toBe("model-1");
            expect(model1.key).toBe("key-1");
            expect(model2.identifier).toBe("model-2");
            expect(model2.key).toBe("key-2");
        });
    });

    describe("readonly properties", () => {
        it("should have readonly identifier property", () => {
            const model = new Model({ identifier: "test-id", key: "test-key" });

            // TypeScript will prevent assignment at compile time
            // At runtime, the property is still writable (JS limitation)
            // This test verifies the property exists and holds correct value
            expect(model.identifier).toBeDefined();
            expect(typeof model.identifier).toBe("string");
        });

        it("should have readonly key property", () => {
            const model = new Model({ identifier: "test-id", key: "test-key" });

            expect(model.key).toBeDefined();
            expect(typeof model.key).toBe("string");
        });
    });

    describe("ModelProps interface", () => {
        it("should accept valid ModelProps", () => {
            const props: ModelProps = {
                identifier: "gpt-3.5-turbo",
                key: "openai-api-key",
            };

            // Verify the interface works correctly
            expect(props.identifier).toBe("gpt-3.5-turbo");
            expect(props.key).toBe("openai-api-key");
        });
    });
});

