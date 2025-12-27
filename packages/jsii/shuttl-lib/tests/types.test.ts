import {
    isInputContent,
    isInputContentArray,
    InputContent,
    ModelContent,
    ModelResponse,
    ModelResponseData,
    ModelToolOutput,
    ModelTextOutput,
    ModelDeltaOutput,
    Usage,
    ToolCallResponse,
} from "../src/models/types";

describe("isInputContent()", () => {
    describe("returns true for valid InputContent", () => {
        it("should return true for text InputContent", () => {
            const content: InputContent = {
                typeName: "text",
                text: "Hello world",
            };

            expect(isInputContent(content)).toBe(true);
        });

        it("should return true for image InputContent", () => {
            const content: InputContent = {
                typeName: "image",
                image: "https://example.com/image.png",
            };

            expect(isInputContent(content)).toBe(true);
        });

        it("should return true for file InputContent", () => {
            const content: InputContent = {
                typeName: "file",
                file: "https://example.com/document.pdf",
            };

            expect(isInputContent(content)).toBe(true);
        });

        it("should return true for InputContent with minimal required fields", () => {
            const content = { typeName: "text" as const };

            expect(isInputContent(content)).toBe(true);
        });

        it("should return true for InputContent with all optional fields", () => {
            const content: InputContent = {
                typeName: "text",
                text: "text content",
                image: "image url",
                file: "file url",
            };

            expect(isInputContent(content)).toBe(true);
        });
    });

    describe("returns false for invalid inputs", () => {
        it("should return false for string", () => {
            expect(isInputContent("Hello world")).toBe(false);
        });

        it("should return false for number", () => {
            expect(isInputContent(42)).toBe(false);
        });

        it("should return false for boolean", () => {
            expect(isInputContent(true)).toBe(false);
        });

        it("should return false for null", () => {
            expect(isInputContent(null)).toBe(false);
        });

        it("should return false for undefined", () => {
            expect(isInputContent(undefined)).toBe(false);
        });

        it("should return false for array", () => {
            expect(isInputContent([])).toBe(false);
            expect(isInputContent([{ typeName: "text" }])).toBe(false);
        });

        it("should return false for object without typeName", () => {
            expect(isInputContent({ text: "Hello" })).toBe(false);
        });

        it("should return false for empty object", () => {
            expect(isInputContent({})).toBe(false);
        });

        it("should return false for function", () => {
            expect(isInputContent(() => {})).toBe(false);
        });

        it("should return false for Date object", () => {
            expect(isInputContent(new Date())).toBe(false);
        });
    });

    describe("edge cases", () => {
        it("should handle object with typeName property of wrong type", () => {
            const content = { typeName: 123 };
            // Still returns true because it just checks for "typeName" in content
            expect(isInputContent(content)).toBe(true);
        });

        it("should handle deeply nested objects with typeName", () => {
            const content = {
                typeName: "text",
                nested: { also: { typeName: "nested" } },
            };

            expect(isInputContent(content)).toBe(true);
        });
    });
});

describe("isInputContentArray()", () => {
    describe("returns true for valid InputContent arrays", () => {
        it("should return true for array with single text InputContent", () => {
            const contents: InputContent[] = [{ typeName: "text", text: "Hello" }];

            expect(isInputContentArray(contents)).toBe(true);
        });

        it("should return true for array with multiple InputContents", () => {
            const contents: InputContent[] = [
                { typeName: "text", text: "Hello" },
                { typeName: "image", image: "https://example.com/img.png" },
                { typeName: "file", file: "https://example.com/doc.pdf" },
            ];

            expect(isInputContentArray(contents)).toBe(true);
        });

        it("should return true for empty array", () => {
            expect(isInputContentArray([])).toBe(true);
        });

        it("should return true for array with mixed content types", () => {
            const contents: InputContent[] = [
                { typeName: "text", text: "Describe this:" },
                { typeName: "image", image: "https://example.com/photo.jpg" },
            ];

            expect(isInputContentArray(contents)).toBe(true);
        });
    });

    describe("returns false for invalid inputs", () => {
        it("should return false for string", () => {
            expect(isInputContentArray("Hello")).toBe(false);
        });

        it("should return false for number", () => {
            expect(isInputContentArray(42)).toBe(false);
        });

        it("should return false for null", () => {
            expect(isInputContentArray(null)).toBe(false);
        });

        it("should return false for undefined", () => {
            expect(isInputContentArray(undefined)).toBe(false);
        });

        it("should return false for object", () => {
            expect(isInputContentArray({ typeName: "text" })).toBe(false);
        });

        it("should return false for array with non-InputContent elements", () => {
            expect(isInputContentArray(["Hello", "World"])).toBe(false);
        });

        it("should return false for array with mixed valid and invalid elements", () => {
            const mixed = [
                { typeName: "text", text: "Valid" },
                "Invalid string",
            ];

            expect(isInputContentArray(mixed)).toBe(false);
        });

        it("should return false for array of numbers", () => {
            expect(isInputContentArray([1, 2, 3])).toBe(false);
        });

        it("should return false for array of objects without typeName", () => {
            expect(isInputContentArray([{ text: "Hello" }, { image: "url" }])).toBe(
                false
            );
        });

        it("should return false for array of null", () => {
            expect(isInputContentArray([null, null])).toBe(false);
        });
    });

    describe("edge cases", () => {
        it("should handle large arrays", () => {
            const largeArray: InputContent[] = Array(1000)
                .fill(null)
                .map((_, i) => ({
                    typeName: "text" as const,
                    text: `Item ${i}`,
                }));

            expect(isInputContentArray(largeArray)).toBe(true);
        });

        it("should return false if one element fails validation", () => {
            const mostlyValid = [
                { typeName: "text", text: "Valid 1" },
                { typeName: "text", text: "Valid 2" },
                { noTypeName: "invalid" },
                { typeName: "text", text: "Valid 3" },
            ];

            expect(isInputContentArray(mostlyValid)).toBe(false);
        });
    });
});

describe("Type definitions", () => {
    describe("InputContent", () => {
        it("should accept text type", () => {
            const content: InputContent = {
                typeName: "text",
                text: "Hello world",
            };

            expect(content.typeName).toBe("text");
            expect(content.text).toBe("Hello world");
        });

        it("should accept image type", () => {
            const content: InputContent = {
                typeName: "image",
                image: "https://example.com/image.png",
            };

            expect(content.typeName).toBe("image");
            expect(content.image).toBe("https://example.com/image.png");
        });

        it("should accept file type", () => {
            const content: InputContent = {
                typeName: "file",
                file: "https://example.com/document.pdf",
            };

            expect(content.typeName).toBe("file");
            expect(content.file).toBe("https://example.com/document.pdf");
        });
    });

    describe("ModelContent", () => {
        it("should accept user role", () => {
            const content: ModelContent = {
                content: "Hello",
                role: "user",
            };

            expect(content.role).toBe("user");
        });

        it("should accept assistant role", () => {
            const content: ModelContent = {
                content: "Hello",
                role: "assistant",
            };

            expect(content.role).toBe("assistant");
        });

        it("should accept system role", () => {
            const content: ModelContent = {
                content: "You are helpful",
                role: "system",
            };

            expect(content.role).toBe("system");
        });

        it("should accept string content", () => {
            const content: ModelContent = {
                content: "String message",
                role: "user",
            };

            expect(content.content).toBe("String message");
        });

        it("should accept InputContent content", () => {
            const content: ModelContent = {
                content: { typeName: "text", text: "InputContent" },
                role: "user",
            };

            expect((content.content as InputContent).typeName).toBe("text");
        });

        it("should accept InputContent array content", () => {
            const content: ModelContent = {
                content: [
                    { typeName: "text", text: "First" },
                    { typeName: "image", image: "url" },
                ],
                role: "user",
            };

            expect(Array.isArray(content.content)).toBe(true);
            expect((content.content as InputContent[]).length).toBe(2);
        });
    });

    describe("ModelToolOutput", () => {
        it("should have required properties", () => {
            const output: ModelToolOutput = {
                outputType: "tool_call",
                name: "get_weather",
                arguments: { location: "San Francisco" },
                callId: "call-123",
            };

            expect(output.outputType).toBe("tool_call");
            expect(output.name).toBe("get_weather");
            expect(output.arguments).toEqual({ location: "San Francisco" });
            expect(output.callId).toBe("call-123");
        });
    });

    describe("ModelTextOutput", () => {
        it("should have required properties", () => {
            const output: ModelTextOutput = {
                outputType: "output_text",
                text: "Hello, world!",
            };

            expect(output.outputType).toBe("output_text");
            expect(output.text).toBe("Hello, world!");
        });
    });

    describe("ModelDeltaOutput", () => {
        it("should have required properties", () => {
            const delta: ModelDeltaOutput = {
                outputType: "output_text_delta",
                delta: "Hello",
                sequenceNumber: 1,
            };

            expect(delta.outputType).toBe("output_text_delta");
            expect(delta.delta).toBe("Hello");
            expect(delta.sequenceNumber).toBe(1);
        });
    });

    describe("Usage", () => {
        it("should have all token fields", () => {
            const usage: Usage = {
                inputTokens: 100,
                inputTokensDetails: { cachedTokens: 10 },
                outputTokens: 50,
                outputTokensDetails: { reasoningTokens: 5 },
                totalTokens: 150,
            };

            expect(usage.inputTokens).toBe(100);
            expect(usage.inputTokensDetails.cachedTokens).toBe(10);
            expect(usage.outputTokens).toBe(50);
            expect(usage.outputTokensDetails.reasoningTokens).toBe(5);
            expect(usage.totalTokens).toBe(150);
        });
    });

    describe("ModelResponseData", () => {
        it("should accept tool_call type", () => {
            const data: ModelResponseData = {
                typeName: "tool_call",
                toolCall: {
                    outputType: "tool_call",
                    name: "search",
                    arguments: { query: "test" },
                    callId: "call-1",
                },
            };

            expect(data.typeName).toBe("tool_call");
            expect(data.toolCall?.name).toBe("search");
        });

        it("should accept output_text type", () => {
            const data: ModelResponseData = {
                typeName: "output_text",
                outputText: {
                    outputType: "output_text",
                    text: "Response text",
                },
            };

            expect(data.typeName).toBe("output_text");
            expect(data.outputText?.text).toBe("Response text");
        });

        it("should accept output_text_delta type", () => {
            const data: ModelResponseData = {
                typeName: "output_text_delta",
                outputTextDelta: {
                    outputType: "output_text_delta",
                    delta: "chunk",
                    sequenceNumber: 3,
                },
            };

            expect(data.typeName).toBe("output_text_delta");
            expect(data.outputTextDelta?.delta).toBe("chunk");
        });

        it("should accept response.completed type", () => {
            const data: ModelResponseData = {
                typeName: "response.completed",
            };

            expect(data.typeName).toBe("response.completed");
        });

        it("should accept response.requested type", () => {
            const data: ModelResponseData = {
                typeName: "response.requested",
                requested: { model: "gpt-4" },
            };

            expect(data.typeName).toBe("response.requested");
            expect(data.requested.model).toBe("gpt-4");
        });

        it("should accept output_text.part.done type", () => {
            const data: ModelResponseData = {
                typeName: "output_text.part.done",
                outputText: {
                    outputType: "output_text",
                    text: "Part done",
                },
            };

            expect(data.typeName).toBe("output_text.part.done");
        });
    });

    describe("ModelResponse", () => {
        it("should have eventName and data", () => {
            const response: ModelResponse = {
                eventName: "response.output_text.done",
                data: {
                    typeName: "output_text",
                    outputText: {
                        outputType: "output_text",
                        text: "Complete",
                    },
                },
            };

            expect(response.eventName).toBe("response.output_text.done");
            expect(response.data).toBeDefined();
        });

        it("should accept data as array", () => {
            const response: ModelResponse = {
                eventName: "response.completed",
                data: [
                    {
                        typeName: "output_text",
                        outputText: { outputType: "output_text", text: "First" },
                    },
                    {
                        typeName: "output_text",
                        outputText: { outputType: "output_text", text: "Second" },
                    },
                ],
            };

            expect(Array.isArray(response.data)).toBe(true);
            expect((response.data as ModelResponseData[]).length).toBe(2);
        });

        it("should have optional usage field", () => {
            const response: ModelResponse = {
                eventName: "response.done",
                data: { typeName: "response.completed" },
                usage: {
                    inputTokens: 100,
                    inputTokensDetails: { cachedTokens: 0 },
                    outputTokens: 50,
                    outputTokensDetails: { reasoningTokens: 0 },
                    totalTokens: 150,
                },
            };

            expect(response.usage?.totalTokens).toBe(150);
        });
    });

    describe("ToolCallResponse", () => {
        it("should be a generic record type", () => {
            const response: ToolCallResponse = {
                output: '{"result": "success"}',
                type: "function_call_output",
                call_id: "call-123",
            };

            expect(response.output).toBe('{"result": "success"}');
            expect(response.type).toBe("function_call_output");
            expect(response.call_id).toBe("call-123");
        });

        it("should accept any keys", () => {
            const response: ToolCallResponse = {
                customField: "custom value",
                numericField: 42,
                nested: { key: "value" },
            };

            expect(response.customField).toBe("custom value");
            expect(response.numericField).toBe(42);
        });
    });
});

