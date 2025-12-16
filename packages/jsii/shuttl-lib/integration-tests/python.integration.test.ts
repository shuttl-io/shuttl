/**
 * Integration tests for Python JSII target
 * 
 * These tests spawn the Python test app as a separate process
 * and communicate with it via the IPC protocol over STDIN/STDOUT.
 */

import * as path from "path";
import * as fs from "fs";
import { execSync } from "child_process";
import { IPCClient, createIPCClient } from "./helpers/ipc-client";

const FIXTURE_DIR = path.join(__dirname, "fixtures", "python");
const APP_PATH = path.join(FIXTURE_DIR, "app.py");
const VENV_DIR = path.join(FIXTURE_DIR, ".venv");
const DIST_DIR = path.join(__dirname, "..", "dist", "python");

describe("Python Integration Tests", () => {
    let client: IPCClient;
    let pythonPath: string;
    let setupComplete = false;

    beforeAll(async () => {
        // Check if Python wheel exists
        const wheelFiles = fs.readdirSync(DIST_DIR).filter(f => f.endsWith('.whl'));
        if (wheelFiles.length === 0) {
            console.warn("Python wheel not found in dist/python. Skipping Python integration tests.");
            return;
        }

        try {
            // Create virtual environment if it doesn't exist
            if (!fs.existsSync(VENV_DIR)) {
                console.log("Creating Python virtual environment...");
                execSync(`python3 -m venv ${VENV_DIR}`, { cwd: FIXTURE_DIR, stdio: 'pipe' });
            }

            // Determine python path
            pythonPath = path.join(VENV_DIR, "bin", "python");
            if (!fs.existsSync(pythonPath)) {
                pythonPath = path.join(VENV_DIR, "Scripts", "python.exe"); // Windows
            }

            // Install the wheel
            const wheelPath = path.join(DIST_DIR, wheelFiles[0]);
            console.log(`Installing wheel: ${wheelPath}`);
            execSync(`${pythonPath} -m pip install "${wheelPath}" --force-reinstall --quiet`, {
                cwd: FIXTURE_DIR,
                stdio: 'pipe',
            });

            setupComplete = true;

            // Spawn the Python app
            client = await createIPCClient({
                command: pythonPath,
                args: [APP_PATH],
                cwd: FIXTURE_DIR,
            });
        } catch (error) {
            console.warn("Failed to set up Python environment:", error);
        }
    }, 120000); // 2 minute timeout for setup

    afterAll(async () => {
        if (client) {
            await client.close();
        }
    });

    // Skip tests if setup failed
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
                name: "PythonTestApp",
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
        });
    });

    describeIfSetup()("invokeTool", () => {
        it("should invoke echo tool and return result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "echo", {
                message: "Hello from Python!",
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.echoed).toBe("Hello from Python!");
        });

        it("should invoke math tool and return calculation result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "add", {
                a: 7,
                b: 13,
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.result).toBe(20);
        });

        it("should return error for non-existent toolkit", async () => {
            const response = await client.invokeTool("NonExistentToolkit", "echo", {});

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
        });
    });
});













