import { IOutcome, SlackOutcome } from "../src/Outcomes";

describe("IOutcome interface", () => {
    it("should define send method that returns Promise<void>", () => {
        const outcome: IOutcome = {
            send: jest.fn().mockResolvedValue(undefined),
        };

        expect(typeof outcome.send).toBe("function");
    });

    it("should allow custom IOutcome implementations", async () => {
        class CustomOutcome implements IOutcome {
            public sentMessages: any[] = [];

            async send(message: any): Promise<void> {
                this.sentMessages.push(message);
            }
        }

        const outcome = new CustomOutcome();
        await outcome.send({ test: "message" });

        expect(outcome.sentMessages).toHaveLength(1);
        expect(outcome.sentMessages[0]).toEqual({ test: "message" });
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

            await outcome.send(message);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                message
            );
        });

        it("should return a resolved promise", async () => {
            const outcome = new SlackOutcome();

            const result = outcome.send({ text: "Test" });

            expect(result).toBeInstanceOf(Promise);
            await expect(result).resolves.toBeUndefined();
        });

        it("should handle string messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send("Simple string message");

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                "Simple string message"
            );
        });

        it("should handle object messages", async () => {
            const outcome = new SlackOutcome();
            const message = {
                channel: "#general",
                text: "Hello!",
                attachments: [{ title: "Attachment" }],
            };

            await outcome.send(message);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                message
            );
        });

        it("should handle null messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send(null);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                null
            );
        });

        it("should handle undefined messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send(undefined);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                undefined
            );
        });

        it("should handle array messages", async () => {
            const outcome = new SlackOutcome();
            const messages = ["message1", "message2", "message3"];

            await outcome.send(messages);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                messages
            );
        });

        it("should handle number messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send(42);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                42
            );
        });

        it("should handle boolean messages", async () => {
            const outcome = new SlackOutcome();

            await outcome.send(true);

            expect(consoleSpy).toHaveBeenCalledWith(
                "Sending message to Slack:",
                true
            );
        });
    });

    describe("implements IOutcome", () => {
        it("should satisfy IOutcome interface", () => {
            const outcome: IOutcome = new SlackOutcome();

            expect(typeof outcome.send).toBe("function");
        });

        it("should be usable as IOutcome", async () => {
            const outcome: IOutcome = new SlackOutcome();

            await expect(outcome.send({ test: "data" })).resolves.toBeUndefined();
        });
    });

    describe("multiple sends", () => {
        it("should handle multiple sequential sends", async () => {
            const outcome = new SlackOutcome();

            await outcome.send("First message");
            await outcome.send("Second message");
            await outcome.send("Third message");

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
                outcome.send("Message 1"),
                outcome.send("Message 2"),
                outcome.send("Message 3"),
            ]);

            expect(consoleSpy).toHaveBeenCalledTimes(3);
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
        }

        class WebhookOutcome implements IOutcome {
            public sent: any[] = [];
            async send(message: any): Promise<void> {
                this.sent.push({ type: "webhook", message });
            }
        }

        const outcomes: IOutcome[] = [
            new SlackOutcome(),
            new EmailOutcome(),
            new WebhookOutcome(),
        ];

        const message = { content: "Test notification" };

        for (const outcome of outcomes) {
            await outcome.send(message);
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
        }

        const outcome = new FailingOutcome();

        await expect(outcome.send("test")).rejects.toThrow("Send failed");
    });
});

