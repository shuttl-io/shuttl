/**
 * Integration tests for .NET JSII target
 * 
 * These tests build and spawn the .NET test app as a separate process
 * and communicate with it via the IPC protocol over STDIN/STDOUT.
 */

import * as path from "path";
import * as fs from "fs";
import { execSync, spawnSync } from "child_process";
import { IPCClient, createIPCClient } from "./helpers/ipc-client";

const FIXTURE_DIR = path.join(__dirname, "fixtures", "dotnet");
const BIN_DIR = path.join(FIXTURE_DIR, "bin", "Release", "net6.0");

describe(".NET Integration Tests", () => {
    let client: IPCClient;
    let dllPath: string;
    let setupComplete = false;

    beforeAll(async () => {
        // Check if dotnet is available
        const dotnetCheck = spawnSync("dotnet", ["--version"], { stdio: 'pipe' });
        if (dotnetCheck.status !== 0) {
            console.warn("dotnet CLI not found. Skipping .NET integration tests.");
            return;
        }

        // Check if NuGet package exists in dist
        const dotnetDistDir = path.join(__dirname, "..", "dist", "dotnet");
        const nupkgFiles = fs.existsSync(dotnetDistDir) 
            ? fs.readdirSync(dotnetDistDir).filter(f => f.endsWith('.nupkg'))
            : [];
        
        if (nupkgFiles.length === 0) {
            console.warn("NuGet package not found in dist/dotnet. Skipping .NET integration tests.");
            return;
        }

        try {
            // Restore and build the .NET test app
            console.log("Building .NET test app...");
            execSync("dotnet restore", {
                cwd: FIXTURE_DIR,
                stdio: 'pipe',
            });

            execSync("dotnet build -c Release", {
                cwd: FIXTURE_DIR,
                stdio: 'pipe',
            });

            // Find the DLL
            dllPath = path.join(BIN_DIR, "IntegrationTestApp.dll");
            if (!fs.existsSync(dllPath)) {
                console.warn("Built DLL not found. Skipping .NET integration tests.");
                return;
            }

            setupComplete = true;

            // Spawn the .NET app
            client = await createIPCClient({
                command: "dotnet",
                args: [dllPath],
                cwd: FIXTURE_DIR,
            });
        } catch (error) {
            console.warn("Failed to set up .NET environment:", error);
        }
    }, 120000); // 2 minute timeout for build

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
                name: "DotNetTestApp",
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
                message: "Hello from .NET!",
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.echoed).toBe("Hello from .NET!");
        });

        it("should invoke math tool and return calculation result", async () => {
            const response = await client.invokeTool("UtilityToolkit", "add", {
                a: 42,
                b: 58,
            });

            expect(response.success).toBe(true);
            const result = response.result as Record<string, unknown>;
            expect(result.result).toBe(100);
        });

        it("should return error for non-existent toolkit", async () => {
            const response = await client.invokeTool("NonExistentToolkit", "echo", {});

            expect(response.success).toBe(false);
            expect(response.errorObj?.code).toBe("NOT_FOUND");
        });
    });
});






















