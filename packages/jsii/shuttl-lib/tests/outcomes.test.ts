import { IOutcome, SlackOutcome } from "../src/outcomes/IOutcomes";
import { StreamingOutcome } from "../src/outcomes/StreamingOutcome";
import { IModelResponseStream, ModelResponseStreamValue } from "../src/models/types";

// Helper to create a properly typed mock streamer
function createMockStreamer(): IModelResponseStream {
    let called = false;
    return {
        async next(): Promise<ModelResponseStreamValue> {
            if (called) {
                return { value: undefined, done: true };
            }
            called = true;
            return { 
                value: { 
                    eventName: "test", 
                    data: { 
                        typeName: "output_text" as const,
                        outputText: { outputType: "output_text" as const, text: "test" }
                    } 
                },
                done: false
            };
        }
    };
}

describe("IOutcome interface", () => {
    it("should define send method that returns Promise<void>", () => {
        const outcome: IOutcome = {
            send: jest.fn().mockResolvedValue(undefined),
            bindToRequest: jest.fn().mockResolvedValue(undefined),
        };

        expect(typeof outcome.send).toBe("function");
    });

    it("should define bindToRequest method that returns Promise<void>", () => {
        const outcome: IOutcome = {
            send: jest.fn().mockResolvedValue(undefined),
            bindToRequest: jest.fn().mockResolvedValue(undefined),
        };

        expect(typeof outcome.bindToRequest).toBe("function");
    });

    it("should allow custom IOutcome implementations", async () => {
        class CustomOutcome implements IOutcome {
            public sentMessages: any[] = [];
            public boundRequests: any[] = [];

            async send(message: any): Promise<void> {
                this.sentMessages.push(message);
            }

            async bindToRequest(request: any): Promise<void> {
                this.boundRequests.push(request);
            }
        }

        const outcome = new CustomOutcome();
        await outcome.send({ test: "message" });
        await outcome.bindToRequest({ requestId: "123" });

        expect(outcome.sentMessages).toHaveLength(1);
        expect(outcome.sentMessages[0]).toEqual({ test: "message" });
        expect(outcome.boundRequests).toHaveLength(1);
        expect(outcome.boundRequests[0]).toEqual({ requestId: "123" });
    });
});

describe("SlackOutcome", () => {
    let consoleSpy: jest.SpyInstance;

    beforeEach(() => {
        consoleSpy = jest.spyOn(console, "log").mockImplementation();
    });

    afterEach(() => {
        consoleSpy.mockRestore();
    });

    describe("constructor", () => {
        it("should create a SlackOutcome instance", () => {
            const outcome = new SlackOutcome();

            expect(outcome).toBeInstanceOf(SlackOutcome);
        });
    });

    describe("send()", () => {
        it("should log the message to console", async () => {
            const outcome = new SlackOutcome();
            const message = { text: "Hello, Slack!" };

            await outcome.send(message as any);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                message
            );
        });

        it("should return a resolved promise", async () => {
            const outcome = new SlackOutcome();

            const result = outcome.send({ text: "Test" } as any);

            expect(result).toBeInstanceOf(Promise);
            await expect(result).resolves.toBeUndefined();
        });

        it("should handle async generator messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send(createMockStreamer());

            expect(consoleSpy).toHaveBeenCalled();
        });
    });

    describe("bindToRequest()", () => {
        it("should log the request to console", async () => {
            const outcome = new SlackOutcome();
            const request = { requestId: "123" };

            await outcome.bindToRequest(request);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Binding request to Slack:",
                request
            );
        });

        it("should return a resolved promise", async () => {
            const outcome = new SlackOutcome();

            const result = outcome.bindToRequest({ test: "data" });

            expect(result).toBeInstanceOf(Promise);
            await expect(result).resolves.toBeUndefined();
        });
    });

    describe("implements IOutcome", () => {
        it("should satisfy IOutcome interface", () => {
            const outcome: IOutcome = new SlackOutcome();

            expect(typeof outcome.send).toBe("function");
            expect(typeof outcome.bindToRequest).toBe("function");
        });

        it("should be usable as IOutcome", async () => {
            const outcome: IOutcome = new SlackOutcome();

            await expect(outcome.send({ test: "data" } as any)).resolves.toBeUndefined();
            await expect(outcome.bindToRequest({ test: "data" })).resolves.toBeUndefined();
        });
    });

    describe("multiple sends", () => {
        it("should handle multiple sequential sends", async () => {
            const outcome = new SlackOutcome();

            await outcome.send("First message" as any);
            await outcome.send("Second message" as any);
            await outcome.send("Third message" as any);

            expect(consoleSpy).toHaveBeenCalledTimes(3);
            expect(consoleSpy).toHaveBeenNthCalledWith(
                1,
                "Sending message to Slack:",
                "First message"
            );
            expect(consoleSpy).toHaveBeenNthCalledWith(
                2,
                "Sending message to Slack:",
                "Second message"
            );
            expect(consoleSpy).toHaveBeenNthCalledWith(
                3,
                "Sending message to Slack:",
                "Third message"
            );
        });

        it("should handle concurrent sends", async () => {
            const outcome = new SlackOutcome();

            await Promise.all([
                outcome.send("Message 1" as any),
                outcome.send("Message 2" as any),
                outcome.send("Message 3" as any),
            ]);

            expect(consoleSpy).toHaveBeenCalledTimes(3);
        });
    });
});

describe("StreamingOutcome", () => {
    let consoleSpy: jest.SpyInstance;

    beforeEach(() => {
        consoleSpy = jest.spyOn(console, "log").mockImplementation();
    });

    afterEach(() => {
        consoleSpy.mockRestore();
    });

    describe("constructor", () => {
        it("should create a StreamingOutcome instance", () => {
            const outcome = new StreamingOutcome();

            expect(outcome).toBeInstanceOf(StreamingOutcome);
        });
    });

    describe("send()", () => {
        it("should log the message to console", async () => {
            const outcome = new StreamingOutcome();

            await outcome.send(createMockStreamer());

            expect(consoleSpy).toHaveBeenCalled();
        });

        it("should return a resolved promise", async () => {
            const outcome = new StreamingOutcome();

            const result = outcome.send(createMockStreamer());

            expect(result).toBeInstanceOf(Promise);
            await expect(result).resolves.toBeUndefined();
        });
    });

    describe("bindToRequest()", () => {
        it("should log the request to console", async () => {
            const outcome = new StreamingOutcome();
            const request = { requestId: "123" };

            await outcome.bindToRequest(request);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Binding request to Slack:",
                request
            );
        });

        it("should return a resolved promise", async () => {
            const outcome = new StreamingOutcome();

            const result = outcome.bindToRequest({ test: "data" });

            expect(result).toBeInstanceOf(Promise);
            await expect(result).resolves.toBeUndefined();
        });
    });

    describe("implements IOutcome", () => {
        it("should satisfy IOutcome interface", () => {
            const outcome: IOutcome = new StreamingOutcome();

            expect(typeof outcome.send).toBe("function");
            expect(typeof outcome.bindToRequest).toBe("function");
        });

        it("should be usable as IOutcome", async () => {
            const outcome: IOutcome = new StreamingOutcome();

            await expect(outcome.send(createMockStreamer())).resolves.toBeUndefined();
            await expect(outcome.bindToRequest({ test: "data" })).resolves.toBeUndefined();
        });
    });
});

describe("IOutcome usage patterns", () => {
    it("should work with multiple outcome types", async () => {
        class EmailOutcome implements IOutcome {
            public sent: any[] = [];
            async send(message: any): Promise<void> {
                this.sent.push({ type: "email", message });
            }
            async bindToRequest(_request: any): Promise<void> {}
        }

        class WebhookOutcome implements IOutcome {
            public sent: any[] = [];
            async send(message: any): Promise<void> {
                this.sent.push({ type: "webhook", message });
            }
            async bindToRequest(_request: any): Promise<void> {}
        }

        const outcomes: IOutcome[] = [
            new SlackOutcome(),
            new EmailOutcome(),
            new WebhookOutcome(),
        ];

        const message = { content: "Test notification" };

        for (const outcome of outcomes) {
            await outcome.send(message as any);
        }

        expect(outcomes.length).toBe(3);
    });

    it("should allow async operations in custom outcomes", async () => {
        class AsyncOutcome implements IOutcome {
            public completed = false;

            async send(_message: any): Promise<void> {
                await new Promise((resolve) => setTimeout(resolve, 10));
                this.completed = true;
            }

            async bindToRequest(_request: any): Promise<void> {}
        }

        const outcome = new AsyncOutcome();

        expect(outcome.completed).toBe(false);
        await outcome.send("async message");
        expect(outcome.completed).toBe(true);
    });

    it("should allow throwing outcomes for error handling", async () => {
        class FailingOutcome implements IOutcome {
            async send(_message: any): Promise<void> {
                throw new Error("Send failed");
            }
            async bindToRequest(_request: any): Promise<void> {}
        }

        const outcome = new FailingOutcome();

        await expect(outcome.send("test")).rejects.toThrow("Send failed");
    });
});

