import { App } from "../app";
import { IServer } from "../Server";
import { stdin, stdout } from "process";
import { createInterface, Interface } from "readline";

/**
 * Request message format from the host CLI
 */
export interface IPCRequest {
    /** Unique request ID for correlation */
    id: string;
    /** The method/action to invoke */
    method: string;
    /** Optional parameters for the method */
    body?: Record<string, unknown>;
}

export interface IPCResponseError {
    code: string;
    message: string;
}
/**
 * Response message format sent back to the host CLI
 */
export interface IPCResponse {
    /** Correlates to the request ID */
    id: string;
    /** Whether the request was successful */
    success: boolean;
    /** The result data (on success) */
    result?: unknown;
    /** Error information (on failure) */
    errorObj?: IPCResponseError;
}

/**
 * A server that communicates via STDIN/STDOUT using JSON messages.
 * Uses newline-delimited JSON (NDJSON) protocol where each message
 * is a single line of JSON followed by a newline character.
 * 
 * Host CLI writes requests to the process's STDIN.
 * This server writes responses to STDOUT.
 * Debug/log messages go to STDERR to avoid polluting the protocol.
 */
export class StdInServer implements IServer {
    private app?: App;
    private running: boolean = false;
    private rl?: Interface;

    public constructor() {}

    public accept(app: any): void {
        this.app = app;
    }

    public async start(): Promise<void> {
        if (!this.app) {
            throw new Error("No app registered. Call accept() before start().");
        }

        this.running = true;

        // Create readline interface for line-by-line processing
        this.rl = createInterface({
            input: stdin,
            output: undefined, // We write to stdout manually
            terminal: false,
        });

        // Handle each line as a JSON message
        this.rl.on("line", (line: string) => {
            this.handleLine(line);
        });

        // Handle stdin close
        this.rl.on("close", () => {
            this.running = false;
        });

        // Send ready signal
        this.sendResponse({
            id: "__ready__",
            success: true,
            result: {
                name: this.app.name,
                protocol: "ndjson",
                version: "1.0",
            },
        });

        // Keep the process alive while running
        while (this.running) {
            await new Promise((resolve) => setTimeout(resolve, 100));
        }
    }

    public async stop(): Promise<void> {
        this.running = false;
        if (this.rl) {
            this.rl.close();
        }
    }

    /**
     * Process a single line of input
     */
    private handleLine(line: string): void {
        const trimmed = line.trim();
        if (!trimmed) {
            return; // Skip empty lines
        }

        let request: IPCRequest;

        try {
            request = JSON.parse(trimmed) as IPCRequest;
        } catch (e) {
            this.sendResponse({
                id: "__parse_error__",
                success: false,
                errorObj: {
                    code: "PARSE_ERROR",
                    message: `Invalid JSON: ${(e as Error).message}`,
                },
            });
            return;
        }

        // Validate request format
        if (!request.id || typeof request.id !== "string") {
            this.sendResponse({
                id: request.id ?? "__invalid__",
                success: false,
                errorObj: {
                    code: "INVALID_REQUEST",
                    message: "Request must have a string 'id' field",
                },
            });
            return;
        }

        if (!request.method || typeof request.method !== "string") {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INVALID_REQUEST",
                    message: "Request must have a string 'method' field",
                },
            });
            return;
        }

        // Route the request
        this.handleRequest(request);
    }

    /**
     * Route and handle a validated request
     */
    private handleRequest(request: IPCRequest): void {
        try {
            switch (request.method) {
                case "ping":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: { pong: true, timestamp: Date.now(), protocol_version: "1.0" },
                    });
                    break;

                case "getAppInfo":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: {
                            name: this.app!.name,
                            agentCount: this.app!.agents.length,
                            toolkitCount: this.app!.toolkits.size,
                        },
                    });
                    break;

                case "listAgents":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: this.app!.agents.map((agent) => ({
                            name: agent.name,
                            systemPrompt: agent.systemPrompt,
                            model: agent.model,
                            toolkits: agent.toolkits.map((tk) => tk.name),
                        })),
                    });
                    break;

                case "listToolkits":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: Array.from(this.app!.toolkits).map((toolkit) => ({
                            name: toolkit.name,
                            description: toolkit.description,
                            tools: toolkit.tools.map((tool) => ({
                                name: tool.name,
                                description: tool.description,
                                args: tool.produceArgs(),
                            })),
                        })),
                    });
                    break;

                case "invokeTool":
                    this.handleInvokeTool(request);
                    break;

                case "shutdown":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: { shutting_down: true },
                    });
                    this.stop();
                    break;

                default:
                    this.sendResponse({
                        id: request.id,
                        success: false,
                        errorObj: {
                            code: "UNKNOWN_METHOD",
                            message: `Unknown method: ${request.method}`,
                        },
                    });
            }
        } catch (e) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INTERNAL_ERROR",
                    message: (e as Error).message,
                },
            });
        }
    }

    /**
     * Handle tool invocation request
     */
    private handleInvokeTool(request: IPCRequest): void {
        const params = request.body ?? {};
        const toolkitName = params.toolkit as string | undefined;
        const toolName = params.tool as string | undefined;
        const toolArgs = (params.args as Record<string, unknown>) ?? {};

        if (!toolkitName || !toolName) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INVALID_PARAMS",
                    message: "invokeTool requires 'toolkit' and 'tool' params",
                },
            });
            return;
        }

        // Find the toolkit
        const toolkit = Array.from(this.app!.toolkits).find(
            (tk) => tk.name === toolkitName
        );
        if (!toolkit) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "NOT_FOUND",
                    message: `Toolkit not found: ${toolkitName}`,
                },
            });
            return;
        }

        // Find the tool
        const tool = toolkit.tools.find((t) => t.name === toolName);
        if (!tool) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "NOT_FOUND",
                    message: `Tool not found: ${toolName} in toolkit ${toolkitName}`,
                },
            });
            return;
        }

        // Execute the tool
        try {
            const result = tool.execute(toolArgs);
            
            // Handle async tools
            if (result instanceof Promise) {
                result
                    .then((asyncResult) => {
                        this.sendResponse({
                            id: request.id,
                            success: true,
                            result: asyncResult,
                        });
                    })
                    .catch((e) => {
                        this.sendResponse({
                            id: request.id,
                            success: false,
                            errorObj: {
                                code: "TOOL_ERROR",
                                message: (e as Error).message,
                            },
                        });
                    });
            } else {
                this.sendResponse({
                    id: request.id,
                    success: true,
                    result,
                });
            }
        } catch (e) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "TOOL_ERROR",
                    message: (e as Error).message,
                },
            });
        }
    }

    /**
     * Send a JSON response to STDOUT
     */
    private sendResponse(response: IPCResponse): void {
        const json = JSON.stringify(response);
        stdout.write(json + "\n");
    }

}
