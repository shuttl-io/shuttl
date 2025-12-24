import { Model } from "../src/model";
import { Secret } from "../src/secrets";
import { ModelContent, IModelStreamer, IModel, IModelFactory, ModelResponse, ModelResponseData } from "../src/models/types";

// Mock streamer for testing
class MockStreamer implements IModelStreamer {
    public receivedResponses: ModelResponse[] = [];
    public receivedModels: IModel[] = [];

    async recieve(model: IModel, content: ModelResponse): Promise<void> {
        this.receivedModels.push(model);
        this.receivedResponses.push(content);
    }
}

// Mock model for testing the streaming interface without real API calls
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

describe("Model", () => {
    describe("openAI factory", () => {
        it("should create an OpenAI model factory with identifier and apiKey", () => {
            const secret = Secret.fromEnv("OPENAI_API_KEY");
            const factory = Model.openAI("gpt-4", secret);

            expect(factory).toBeDefined();
        });

        it("should create factories with different identifiers", () => {
            const secret = Secret.fromEnv("OPENAI_API_KEY");
            const gpt4 = Model.openAI("gpt-4", secret);
            const gpt35 = Model.openAI("gpt-3.5-turbo", secret);

            expect(gpt4).toBeDefined();
            expect(gpt35).toBeDefined();
        });

        it("should return an IModelFactory implementation", () => {
            const secret = Secret.fromEnv("OPENAI_API_KEY");
            const factory = Model.openAI("gpt-4", secret);

            // IModelFactory should have a create method
            expect(typeof factory.create).toBe("function");
        });

        it("should create multiple independent factory instances", () => {
            const key1 = Secret.fromEnv("KEY_1");
            const key2 = Secret.fromEnv("KEY_2");
            const factory1 = Model.openAI("model-1", key1);
            const factory2 = Model.openAI("model-2", key2);

            expect(factory1).not.toBe(factory2);
        });
    });

    describe("IModelFactory interface", () => {
        it("should have a create method that accepts IModelFactoryProps", async () => {
            const factory = new MockModelFactory();

            const model = await factory.create({
                systemPrompt: "You are a helpful assistant.",
            });

            expect(model).toBeDefined();
        });

        it("should create IModel instances with invoke method", async () => {
            const factory = new MockModelFactory();

            const model = await factory.create({
                systemPrompt: "You are a helpful assistant.",
            });

            expect(typeof model.invoke).toBe("function");
        });

        it("should create IModel instances with threadId property", async () => {
            const factory = new MockModelFactory();

            const model = await factory.create({
                systemPrompt: "You are a helpful assistant.",
            });

            expect(model.threadId).toBeDefined();
        });

        it("should create multiple independent model instances", async () => {
            const factory = new MockModelFactory();

            const model1 = await factory.create({
                systemPrompt: "You are assistant 1.",
            });

            const model2 = await factory.create({
                systemPrompt: "You are assistant 2.",
            });

            expect(model1).not.toBe(model2);
        });
    });

    describe("IModel interface with streaming", () => {
        it("should invoke with streamer and return void", async () => {
            const model = new MockModel();
            const streamer = new MockStreamer();
            const prompt: ModelContent[] = [
                { content: "Test message", role: "user" },
            ];

            const result = await model.invoke(prompt, streamer);

            expect(result).toBeUndefined();
        });

        it("should send ModelResponse to streamer via recieve method", async () => {
            const model = new MockModel();
            const streamer = new MockStreamer();
            const prompt: ModelContent[] = [
                { content: "Hello", role: "user" },
            ];

            await model.invoke(prompt, streamer);

            expect(streamer.receivedResponses.length).toBeGreaterThan(0);
        });

        it("should send response with event and data", async () => {
            const model = new MockModel();
            const streamer = new MockStreamer();
            const prompt: ModelContent[] = [
                { content: "Hello", role: "user" },
            ];

            await model.invoke(prompt, streamer);

            expect(streamer.receivedResponses[0].eventName).toBeDefined();
            expect(streamer.receivedResponses[0].data).toBeDefined();
        });

        it("should pass model reference to streamer recieve method", async () => {
            const model = new MockModel();
            const streamer = new MockStreamer();
            const prompt: ModelContent[] = [
                { content: "Hello", role: "user" },
            ];

            await model.invoke(prompt, streamer);

            expect(streamer.receivedModels[0]).toBe(model);
        });

        it("should have threadId property on model", async () => {
            const model = new MockModel();

            expect(model.threadId).toBe("mock-thread-id");
        });
    });

    describe("IModelStreamer interface", () => {
        it("should implement recieve method with model and ModelResponse parameters", () => {
            const streamer = new MockStreamer();

            expect(typeof streamer.recieve).toBe("function");
        });

        it("should accumulate received responses", async () => {
            const streamer = new MockStreamer();
            const mockModel = new MockModel();

            await streamer.recieve(mockModel, {
                eventName: "response.output_text.done",
                data: { typeName: "output_text", outputText: { outputType: "output_text", text: "First" } },
            });
            await streamer.recieve(mockModel, {
                eventName: "response.output_text.done",
                data: { typeName: "output_text", outputText: { outputType: "output_text", text: "Second" } },
            });

            expect(streamer.receivedResponses).toHaveLength(2);
        });

        it("should track which model sent each response", async () => {
            const streamer = new MockStreamer();
            const model1 = new MockModel();
            const model2 = new MockModel();

            await streamer.recieve(model1, {
                eventName: "response.output_text.done",
                data: { typeName: "output_text", outputText: { outputType: "output_text", text: "From model 1" } },
            });
            await streamer.recieve(model2, {
                eventName: "response.output_text.done",
                data: { typeName: "output_text", outputText: { outputType: "output_text", text: "From model 2" } },
            });

            expect(streamer.receivedModels[0]).toBe(model1);
            expect(streamer.receivedModels[1]).toBe(model2);
        });
    });

    describe("ModelResponse types", () => {
        it("should support output_text type", async () => {
            const response: ModelResponse = {
                eventName: "response.output_text.done",
                data: {
                    typeName: "output_text",
                    outputText: {
                        outputType: "output_text",
                        text: "Hello world",
                    },
                },
            };

            expect(Array.isArray(response.data)).toBe(false);
            const data = response.data as ModelResponseData;
            expect(data.typeName).toBe("output_text");
            expect(data.outputText?.text).toBe("Hello world");
        });

        it("should support tool_call type", async () => {
            const response: ModelResponse = {
                eventName: "response.function_call_arguments.done",
                data: {
                    typeName: "tool_call",
                    toolCall: {
                        outputType: "tool_call",
                        name: "get_weather",
                        arguments: { location: "San Francisco" },
                        callId: "123",
                    },
                },
            };

            expect(Array.isArray(response.data)).toBe(false);
            const data = response.data as ModelResponseData;
            expect(data.typeName).toBe("tool_call");
            expect(data.toolCall?.name).toBe("get_weather");
        });

        it("should support output_text_delta type", async () => {
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

            expect(Array.isArray(response.data)).toBe(false);
            const data = response.data as ModelResponseData;
            expect(data.typeName).toBe("output_text_delta");
            expect(data.outputTextDelta?.delta).toBe("Hello");
        });

        it("should support data as an array", async () => {
            const response: ModelResponse = {
                eventName: "response.completed",
                data: [
                    {
                        typeName: "output_text",
                        outputText: {
                            outputType: "output_text",
                            text: "First message",
                        },
                    },
                    {
                        typeName: "output_text",
                        outputText: {
                            outputType: "output_text",
                            text: "Second message",
                        },
                    },
                ],
            };

            expect(Array.isArray(response.data)).toBe(true);
            const dataArray = response.data as ModelResponseData[];
            expect(dataArray).toHaveLength(2);
            expect(dataArray[0].typeName).toBe("output_text");
            expect(dataArray[0].outputText?.text).toBe("First message");
            expect(dataArray[1].outputText?.text).toBe("Second message");
        });

        it("should support optional usage field", async () => {
            const response: ModelResponse = {
                eventName: "response.done",
                data: {
                    typeName: "output_text",
                    outputText: { outputType: "output_text", text: "Done" },
                },
                usage: {
                    inputTokens: 10,
                    inputTokensDetails: { cachedTokens: 0 },
                    outputTokens: 20,
                    outputTokensDetails: { reasoningTokens: 5 },
                    totalTokens: 30,
                },
            };

            expect(response.usage?.totalTokens).toBe(30);
        });
    });

    describe("ModelContent type", () => {
        it("should accept user role", () => {
            const content: ModelContent = {
                content: "User message",
                role: "user",
            };

            expect(content.role).toBe("user");
            expect(content.content).toBe("User message");
        });

        it("should accept assistant role", () => {
            const content: ModelContent = {
                content: "Assistant response",
                role: "assistant",
            };

            expect(content.role).toBe("assistant");
            expect(content.content).toBe("Assistant response");
        });

        it("should accept system role", () => {
            const content: ModelContent = {
                content: "System instructions",
                role: "system",
            };

            expect(content.role).toBe("system");
            expect(content.content).toBe("System instructions");
        });
    });

    describe("different secret types with openAI", () => {
        it("should work with env secrets", () => {
            const secret = Secret.fromEnv("MY_API_KEY");
            const factory = Model.openAI("gpt-4", secret);

            expect(factory).toBeDefined();
            expect(typeof factory.create).toBe("function");
        });
    });
});
