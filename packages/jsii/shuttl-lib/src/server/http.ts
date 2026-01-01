import { App } from "../app";
import { IServer } from "../Server";
import { stdin, stdout } from "process";
import { createInterface, Interface } from "readline";
import { Agent, AgentStreamer, IAgentStreamerWriter } from "../agent";
import { FileAttachment, InputContent, isFileAttachmentArray, IModelResponseStream, ModelResponse, ModelResponseStreamValue } from "../models/types";
import { ITriggerInvoker, SerializedHTTPRequest } from "../trigger/ITrigger";

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
                                args: tool.schema?.properties,
                            })),
                        })),
                    });
                    break;

                case "listTriggers":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: this.app!.agents.flatMap((agent) => 
                            agent.triggers.map((trigger) => ({
                                name: trigger.name,
                                triggerType: trigger.triggerType,
                                agentName: agent.name,
                            }))
                        ),
                    });
                    break;

                case "listModels":
                    // Get unique models from all agents
                    const modelsMap = new Map<string, { identifier: string; key: { source: string; name: string } }>();
                    for (const agent of this.app!.agents) {
                        const model = agent.model as any;
                        if (model && model.identifier) {
                            modelsMap.set(model.identifier, {
                                identifier: model.identifier,
                                key: model.key,
                            });
                        }
                    }
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: Array.from(modelsMap.values()),
                    });
                    break;

                case "listPrompts":
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: this.app!.agents.map((agent) => ({
                            agentName: agent.name,
                            systemPrompt: agent.systemPrompt,
                        })),
                    });
                    break;

                case "listTools":
                    // Get all tools from all toolkits
                    const allTools: Array<{
                        name: string;
                        description: string;
                        args: Record<string, unknown> | undefined;
                        toolkitName: string;
                    }> = [];
                    for (const toolkit of this.app!.toolkits) {
                        for (const tool of toolkit.tools) {
                            allTools.push({
                                name: tool.name,
                                description: tool.description,
                                args: tool.schema?.properties,
                                toolkitName: toolkit.name,
                            });
                        }
                    }
                    this.sendResponse({
                        id: request.id,
                        success: true,
                        result: allTools,
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

                case "invokeAgent":
                    this.handleInvokeAgent(request);
                    break;

                case "invokeTrigger":
                    this.handleInvokeTrigger(request);
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

    private async handleInvokeAgent(request: IPCRequest): Promise<void> {
        const params = request.body ?? {};
        const agentName = params.agent as string | undefined;
        const prompt = (params.prompt as string) ?? "";
        const threadId = params.threadId as string | undefined ?? undefined;
        const rawAttachments = params.attachments;
        
        // Parse and validate attachments
        let attachments: FileAttachment[] | undefined;
        if (rawAttachments !== undefined) {
            if (isFileAttachmentArray(rawAttachments)) {
                attachments = rawAttachments;
            } else {
                this.sendResponse({
                    id: request.id,
                    success: false,
                    errorObj: {
                        code: "INVALID_PARAMS",
                        message: "attachments must be an array of FileAttachment objects",
                    },
                });
                return;
            }
        }

        if (!agentName) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INVALID_PARAMS",
                    message: "invokeAgent requires 'agent' param",
                },
            });
            return;
        }

        const agent = this.app!.agents.find((a) => a.name === agentName);
        if (!agent) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "NOT_FOUND",
                    message: `Agent not found: ${agentName}`,
                },
            });
            return;
        }
        const streamer = new AgentStreamer(agent, request.id);
        try {
            const model = await agent.invoke(prompt, threadId, streamer, attachments);
            this.sendResponse({
                id: request.id,
                success: true,
                result: { threadId: model.threadId, status: "invoked" },
            });
        } catch (e) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INTERNAL_ERROR",
                    message: JSON.stringify({
                        message: (e as Error).message,
                        stack: (e as Error).stack,
                        name: (e as Error).name,
                        type_of_error: typeof e,
                        error_name: e instanceof Error ? e.name : undefined,
                        error_object: e,
                        error_constructor: e instanceof Error ? e.constructor.name : undefined,
                    }),
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
     * Handle trigger invocation request from the serve command
     */
    private async handleInvokeTrigger(request: IPCRequest): Promise<void> {
        const params = request.body ?? {};
        const agentName = params.agentName as string | undefined;
        const triggerName = params.triggerName as string | undefined;
        const triggerType = params.triggerType as string | undefined;
        const threadId = params.threadId as string | undefined;
        const httpRequest = params.httpRequest as SerializedHTTPRequest | undefined;

        // Validate required parameters
        if (!agentName) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INVALID_PARAMS",
                    message: "invokeTrigger requires 'agentName' param",
                },
            });
            return;
        }

        if (!triggerName && !triggerType) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "INVALID_PARAMS",
                    message: "invokeTrigger requires 'triggerName' or 'triggerType' param",
                },
            });
            return;
        }

        // Find the agent
        const agent = this.app!.agents.find((a) => a.name === agentName);
        if (!agent) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "NOT_FOUND",
                    message: `Agent not found: ${agentName}`,
                },
            });
            return;
        }

        // Find the trigger by name first, then by type
        let trigger = agent.triggers.find((t) => t.name === triggerName);
        if (!trigger && triggerType) {
            trigger = agent.triggers.find((t) => t.triggerType === triggerType);
        }

        if (!trigger) {
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "NOT_FOUND",
                    message: `Trigger not found: ${triggerName || triggerType} on agent ${agentName}`,
                },
            });
            return;
        }

        // Validate the trigger arguments if validation is available
        if (trigger.validate) {
            try {
                const validationResult = await trigger.validate(httpRequest);
                if (validationResult.error) {
                    this.sendResponse({
                        id: request.id,
                        success: false,
                        errorObj: {
                            code: "VALIDATION_ERROR",
                            message: validationResult.error as string,
                        },
                    });
                    return;
                }
            } catch (e) {
                this.sendResponse({
                    id: request.id,
                    success: false,
                    errorObj: {
                        code: "VALIDATION_ERROR",
                        message: (e as Error).message,
                    },
                });
                return;
            }
        }

        // Create the trigger invoker with optional thread ID
        const invoker = new ServerTriggerInvoker(agent, request.id, threadId);

        try {
            // Activate the trigger with the HTTP request
            const triggerFunc = async () => {
                await trigger.activate(httpRequest, invoker);

                this.sendResponse({
                    id: request.id,
                    success: true,
                    result: {
                        agentName: agentName,
                        triggerName: trigger.name,
                        triggerType: trigger.triggerType,
                        threadId: invoker.getThreadId(),
                        status: "completed",
                        output: invoker.getOutput(),
                    },
                });
            }
            triggerFunc();

            // Send success response with the result
            this.sendResponse({
                id: request.id,
                success: true,
                result: {
                    agentName: agentName,
                    triggerName: trigger.name,
                    triggerType: trigger.triggerType,
                    threadId: invoker.getThreadId(),
                    status: "acknowledged",
                    output: invoker.getOutput(),
                },
            });
        } catch (e) {
            
            this.sendResponse({
                id: request.id,
                success: false,
                errorObj: {
                    code: "TRIGGER_ERROR",
                    message: JSON.stringify({
                        message: (e as Error).message,
                        stack: (e as Error).stack,
                        name: (e as Error).name,
                        type_of_error: typeof e,
                        error_name: e instanceof Error ? e.name : undefined,
                        error_object: e,
                        error_constructor: e instanceof Error ? e.constructor.name : undefined,
                    }),
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

interface TriggerResponse extends ModelResponse {
    id: string;
    type: string;
    success: boolean;
}

class TriggerStreamerWriter implements IAgentStreamerWriter, IModelResponseStream {
    private buffer: ModelResponse[] = [];
    private done: boolean = false;

    write(value: string): void {
        this.writeObject(JSON.parse(value));
    }

    writeObject(value: TriggerResponse): void {
        if (!this.done) {
            this.buffer.push(value);
        }
        if (value.type === "overall.completed") {
            this.done = true;
        }
    }

    async next(): Promise<ModelResponseStreamValue> {
        if (this.buffer.length > 0) {
            return { done: false, value: this.buffer.shift() };
        }
        if (this.done && this.buffer.length === 0) {
            return { done: true, value: undefined };
        } else {
            return new Promise((resolve) => {
                setImmediate((() => {
                    resolve(this.next());
                }).bind(this));
            });
        }
    }
}   

/**
 * Trigger invoker that invokes the agent via the server
 */
class ServerTriggerInvoker implements ITriggerInvoker {
    private output: unknown = null;
    private currentThreadId: string | undefined;

    constructor(
        private readonly agent: Agent,
        private readonly requestId: string,
        private readonly threadId?: string,
    ) {
        this.currentThreadId = threadId;
    }

    public async invoke(prompt: InputContent[]): Promise<IModelResponseStream> {
        // Create a streamer for this invocation
        const writer = new TriggerStreamerWriter();
        const streamer = new AgentStreamer(this.agent, this.requestId, writer);
        
        // Build the model content from the input
        const modelContent = prompt.map((input) => ({
            role: "user" as const,
            content: input,
        }));

        // Use the agent's invoke method which handles thread management
        const startStream = async () => {
            const model = await this.agent.invoke(modelContent, this.threadId, streamer);
            
            // Store the thread ID for the response
            this.currentThreadId = model.threadId;

        }

        startStream();

        // Return a completed response stream since the actual streaming is handled by AgentStreamer
        return writer;
    }

    public async defaultOutcome(_stream: IModelResponseStream): Promise<void> {
        // The default outcome just stores the stream result
        // The actual streaming has already been handled by AgentStreamer
        this.output = { completed: true };
    }

    public getOutput(): unknown {
        return this.output;
    }

    public getThreadId(): string | undefined {
        return this.currentThreadId;
    }
}
