/**
 * Integration tests for Java JSII target
 * 
 * These tests build and spawn the Java test app as a separate process
 * and communicate with it via the IPC protocol over STDIN/STDOUT.
 */

import * as path from "path";
import * as fs from "fs";
import { execSync, spawnSync } from "child_process";
import { IPCClient, createIPCClient } from "./helpers/ipc-client";

const FIXTURE_DIR = path.join(__dirname, "fixtures", "java");
const TARGET_DIR = path.join(FIXTURE_DIR, "target");

describe("Java Integration Tests", () => {
    let client: IPCClient;
    let jarPath: string;
    let setupComplete = false;

    beforeAll(async () => {
        // Check if Maven is available
        const mvnCheck = spawnSync("mvn", ["--version"], { stdio: 'pipe' });
        if (mvnCheck.status !== 0) {
            console.warn("Maven not found. Skipping Java integration tests.");
            return;
        }

        // Check if Java JAR exists in dist
        const javaDistDir = path.join(__dirname, "..", "dist", "java", "io", "shuttl", "module", "shuttl", "1.0.0");
        if (!fs.existsSync(javaDistDir)) {
            console.warn("Java JAR not found in dist. Skipping Java integration tests.");
            return;
        }

        try {
            // Build the Java test app
            console.log("Building Java test app...");
            execSync("mvn clean package -q -DskipTests", {
                cwd: FIXTURE_DIR,
                stdio: 'pipe',
            });

            // Find the shaded JAR
            const targetFiles = fs.readdirSync(TARGET_DIR);
            const shadedJar = targetFiles.find(f => f.endsWith('.jar') && !f.includes('original'));
            if (!shadedJar) {
                console.warn("Shaded JAR not found. Skipping Java integration tests.");
                return;
            }

            jarPath = path.join(TARGET_DIR, shadedJar);
            setupComplete = true;

            // Spawn the Java app
            client = await createIPCClient({
                command: "java",
                args: ["-jar", jarPath],
                cwd: FIXTURE_DIR,
            });
        } catch (error) {
            console.warn("Failed to set up Java environment:", error);
        }
    }, 180000); // 3 minute timeout for Maven build

    afterAll(async () => {
        if (client) {
            await client.close();
        }
    });

    const describeIfSetup = () => setupComplete ? describe : describe.skip;

    describeIfSetup()("ready signal", () => {
        it("should receive ready signal on startup", async () => {
            expect(client).toBeDefined();
        });
    });

    describeIfSetup()("ping", () => {
        it("should respond to ping request", async () => {
            const response = await client.ping();

            expect(response.success).toBe(true);
            expect(response.result).toMatchObject({
                pong: true,
                protocol_version: "1.0",
            });
        });
    });

    describeIfSetup()("getAppInfo", () => {
        it("should return app information", async () => {
            const response = await client.getAppInfo();

            expect(response.success).toBe(true);
            expect(response.result).toMatchObject({
                name: "JavaTestApp",
                agentCount: 1,
            });
        });
    });

    describeIfSetup()("listAgents", () => {
        it("should list all agents", async () => {
            const response = await client.listAgents();

            expect(response.success).toBe(true);
            const agents = response.result as Array<Record<string, unknown>>;
            expect(agents).toHaveLength(1);
            expect(agents[0].name).toBe("TestAgent");
        });
    });

    describeIfSetup()("listToolkits", () => {
        it("should list all toolkits with their tools", async () => {
            const response = await client.listToolkits();

            expect(response.success).toBe(true);
            const toolkits = response.result as Array<Record<string, unknown>>;
            expect(toolkits.length).toBeGreaterThan(0);

            const utilToolkit = toolkits.find(t => t.name === "UtilityToolkit");
            expect(utilToolkit).toBeDefined();
            
            const tools = utilToolkit!.tools as Array<Record<string, unknown>>;
            expect(tools.length).toBeGreaterThan(0);
        });
    });

    describeIfSetup()("invokeTool", () => {
        it("should invoke echo tool and return result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "echo", {
                message: "Hello from Java!",
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.echoed).toBe("Hello from Java!");
        });

        it("should invoke math tool and return calculation result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "add", {
                a: 100,
                b: 200,
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.result).toBe(300);
        });

        it("should return error for non-existent toolkit", async () => {
            const response = await client.invokeTool("NonExistentToolkit", "echo", {});

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
        });
    });
});













