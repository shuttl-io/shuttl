import { Secret, EnvSecret, ISecret } from "../src/secrets";

describe("ISecret interface", () => {
    it("should define resolveSecret method that returns Promise<string>", () => {
        const secret: ISecret = {
            resolveSecret: jest.fn().mockResolvedValue("secret-value"),
        };

        expect(typeof secret.resolveSecret).toBe("function");
    });

    it("should allow custom ISecret implementations", async () => {
        class CustomSecret implements ISecret {
            constructor(private value: string) {}

            async resolveSecret(): Promise<string> {
                return this.value;
            }
        }

        const secret = new CustomSecret("my-custom-secret");
        const resolved = await secret.resolveSecret();

        expect(resolved).toBe("my-custom-secret");
    });
});

describe("EnvSecret", () => {
    const originalEnv = process.env;

    beforeEach(() => {
        jest.resetModules();
        process.env = { ...originalEnv };
    });

    afterAll(() => {
        process.env = originalEnv;
    });

    describe("constructor", () => {
        it("should create an EnvSecret with environment variable name", () => {
            const secret = new EnvSecret("MY_API_KEY");

            expect(secret.envVarName).toBe("MY_API_KEY");
        });

        it("should store the env var name as readonly", () => {
            const secret = new EnvSecret("SECRET_KEY");

            expect(secret.envVarName).toBe("SECRET_KEY");
        });
    });

    describe("resolveSecret()", () => {
        it("should return the value of the environment variable", async () => {
            process.env.TEST_SECRET = "my-secret-value";
            const secret = new EnvSecret("TEST_SECRET");

            const result = await secret.resolveSecret();

            expect(result).toBe("my-secret-value");
        });

        it("should return empty string if env var is not set", async () => {
            delete process.env.NON_EXISTENT_VAR;
            const secret = new EnvSecret("NON_EXISTENT_VAR");

            const result = await secret.resolveSecret();

            expect(result).toBe("");
        });

        it("should return empty string if env var is undefined", async () => {
            process.env.UNDEFINED_VAR = undefined as any;
            const secret = new EnvSecret("UNDEFINED_VAR");

            const result = await secret.resolveSecret();

            expect(result).toBe("");
        });

        it("should return actual empty string if env var is set to empty", async () => {
            process.env.EMPTY_VAR = "";
            const secret = new EnvSecret("EMPTY_VAR");

            const result = await secret.resolveSecret();

            expect(result).toBe("");
        });

        it("should handle special characters in env var values", async () => {
            process.env.SPECIAL_CHARS = "secret!@#$%^&*()_+-=[]{}|;':\",./<>?";
            const secret = new EnvSecret("SPECIAL_CHARS");

            const result = await secret.resolveSecret();

            expect(result).toBe("secret!@#$%^&*()_+-=[]{}|;':\",./<>?");
        });

        it("should handle long env var values", async () => {
            const longValue = "a".repeat(10000);
            process.env.LONG_VAR = longValue;
            const secret = new EnvSecret("LONG_VAR");

            const result = await secret.resolveSecret();

            expect(result).toBe(longValue);
        });

        it("should resolve multiple times from the same instance", async () => {
            process.env.REUSE_VAR = "initial-value";
            const secret = new EnvSecret("REUSE_VAR");

            const result1 = await secret.resolveSecret();
            process.env.REUSE_VAR = "updated-value";
            const result2 = await secret.resolveSecret();

            expect(result1).toBe("initial-value");
            expect(result2).toBe("updated-value");
        });

        it("should be async", () => {
            process.env.ASYNC_TEST = "value";
            const secret = new EnvSecret("ASYNC_TEST");

            const result = secret.resolveSecret();

            expect(result).toBeInstanceOf(Promise);
        });
    });

    describe("implements ISecret", () => {
        it("should satisfy ISecret interface", () => {
            const secret: ISecret = new EnvSecret("MY_VAR");

            expect(typeof secret.resolveSecret).toBe("function");
        });

        it("should be usable as ISecret", async () => {
            process.env.INTERFACE_VAR = "interface-value";
            const secret: ISecret = new EnvSecret("INTERFACE_VAR");

            const result = await secret.resolveSecret();

            expect(result).toBe("interface-value");
        });
    });
});

describe("Secret", () => {
    const originalEnv = process.env;

    beforeEach(() => {
        jest.resetModules();
        process.env = { ...originalEnv };
    });

    afterAll(() => {
        process.env = originalEnv;
    });

    describe("fromEnv()", () => {
        it("should create an ISecret from environment variable name", () => {
            const secret = Secret.fromEnv("MY_API_KEY");

            expect(secret).toBeDefined();
            expect(typeof secret.resolveSecret).toBe("function");
        });

        it("should return an EnvSecret instance", () => {
            const secret = Secret.fromEnv("TEST_KEY");

            expect(secret).toBeInstanceOf(EnvSecret);
        });

        it("should create a working secret that resolves env vars", async () => {
            process.env.FROM_ENV_TEST = "test-secret-value";
            const secret = Secret.fromEnv("FROM_ENV_TEST");

            const result = await secret.resolveSecret();

            expect(result).toBe("test-secret-value");
        });

        it("should create independent secret instances", () => {
            const secret1 = Secret.fromEnv("VAR_1");
            const secret2 = Secret.fromEnv("VAR_2");

            expect(secret1).not.toBe(secret2);
        });

        it("should handle different env var names", async () => {
            process.env.API_KEY = "api-key-value";
            process.env.DB_PASSWORD = "db-password-value";

            const apiSecret = Secret.fromEnv("API_KEY");
            const dbSecret = Secret.fromEnv("DB_PASSWORD");

            expect(await apiSecret.resolveSecret()).toBe("api-key-value");
            expect(await dbSecret.resolveSecret()).toBe("db-password-value");
        });

        it("should return ISecret interface type", () => {
            const secret: ISecret = Secret.fromEnv("ANY_VAR");

            expect(secret.resolveSecret).toBeDefined();
        });
    });

    describe("integration patterns", () => {
        it("should work with async/await pattern", async () => {
            process.env.ASYNC_SECRET = "async-value";
            const secret = Secret.fromEnv("ASYNC_SECRET");

            const value = await secret.resolveSecret();

            expect(value).toBe("async-value");
        });

        it("should work with Promise.then pattern", (done) => {
            process.env.PROMISE_SECRET = "promise-value";
            const secret = Secret.fromEnv("PROMISE_SECRET");

            secret.resolveSecret().then((value) => {
                expect(value).toBe("promise-value");
                done();
            });
        });

        it("should work with multiple concurrent resolutions", async () => {
            process.env.CONCURRENT_1 = "value-1";
            process.env.CONCURRENT_2 = "value-2";
            process.env.CONCURRENT_3 = "value-3";

            const secrets = [
                Secret.fromEnv("CONCURRENT_1"),
                Secret.fromEnv("CONCURRENT_2"),
                Secret.fromEnv("CONCURRENT_3"),
            ];

            const results = await Promise.all(
                secrets.map((s) => s.resolveSecret())
            );

            expect(results).toEqual(["value-1", "value-2", "value-3"]);
        });
    });
});

describe("ISecret custom implementations", () => {
    it("should support file-based secrets", async () => {
        class FileSecret implements ISecret {
            constructor(private content: string) {}

            async resolveSecret(): Promise<string> {
                // Simulate file read
                return this.content;
            }
        }

        const secret: ISecret = new FileSecret("file-based-secret");
        const result = await secret.resolveSecret();

        expect(result).toBe("file-based-secret");
    });

    it("should support vault-style secrets", async () => {
        class VaultSecret implements ISecret {
            private cached: string | null = null;

            constructor(private path: string) {}

            async resolveSecret(): Promise<string> {
                if (this.cached) {
                    return this.cached;
                }
                // Simulate vault fetch
                this.cached = `vault-secret-from-${this.path}`;
                return this.cached;
            }
        }

        const secret: ISecret = new VaultSecret("secret/my-app/api-key");
        const result = await secret.resolveSecret();

        expect(result).toBe("vault-secret-from-secret/my-app/api-key");
    });

    it("should support caching secrets", async () => {
        let fetchCount = 0;

        class CachingSecret implements ISecret {
            private cached: string | null = null;

            async resolveSecret(): Promise<string> {
                if (!this.cached) {
                    fetchCount++;
                    this.cached = "cached-value";
                }
                return this.cached;
            }
        }

        const secret: ISecret = new CachingSecret();

        await secret.resolveSecret();
        await secret.resolveSecret();
        await secret.resolveSecret();

        expect(fetchCount).toBe(1);
    });

    it("should support rotating secrets", async () => {
        class RotatingSecret implements ISecret {
            private counter = 0;

            async resolveSecret(): Promise<string> {
                return `rotating-secret-${this.counter++}`;
            }
        }

        const secret: ISecret = new RotatingSecret();

        expect(await secret.resolveSecret()).toBe("rotating-secret-0");
        expect(await secret.resolveSecret()).toBe("rotating-secret-1");
        expect(await secret.resolveSecret()).toBe("rotating-secret-2");
    });

    it("should support async secrets with delays", async () => {
        class DelayedSecret implements ISecret {
            async resolveSecret(): Promise<string> {
                await new Promise((resolve) => setTimeout(resolve, 10));
                return "delayed-secret";
            }
        }

        const secret: ISecret = new DelayedSecret();
        const result = await secret.resolveSecret();

        expect(result).toBe("delayed-secret");
    });
});


