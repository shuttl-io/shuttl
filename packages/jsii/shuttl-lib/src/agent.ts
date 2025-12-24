import { IModelFactory, ModelContent, ModelResponse, ToolCallResponse } from "./models/types";
import { Toolkit } from "./tools/toolkit";
import { ITrigger } from "./Triggers";
import { IOutcome } from "./Outcomes";
import { IModelStreamer, ModelResponseData } from "./models/types";
import { IModel } from "./models/types";
import { ITool } from "./tools/tool";
import { stdout } from "process";

export interface AgentProps {
    readonly name: string;
    readonly toolkits: Toolkit[];
    readonly systemPrompt: string;
    readonly model: IModelFactory;
    readonly triggers?: ITrigger[];
    readonly outcomes?: IOutcome[];
    readonly tools?: ITool[];
}

export class AgentStreamer implements IModelStreamer {
    private resultPromise: Promise<{ callId: string, result: unknown }>[] = [];
    private jobs: number = 0;
    private calls: Record<string, Promise<{ callId: string, result: unknown }>> = {};

    public constructor(
        private readonly agent: Agent,
        private readonly controlID: string,
    ) {
    }

    private write(type: string, data: any, success: boolean): void {
        let body:any = {
            result: data,
        }
        if (!success) {
            body = {
                errorObj: {
                    code: "ERROR",
                    message: "An error occurred",
                },
            };
        }
        const json = JSON.stringify({
            id: this.controlID,
            type: type,
            success,
            ...body,
        });
        stdout.write(json + "\n");
    }

    public async recieve(model: IModel, content: ModelResponse): Promise<void> {
        this.jobs++;
        try {
            if (content.data === undefined) {
                return;
            }
            if (content.eventName === "response.completed") {
                this.jobs--;
                setImmediate(async () => {
                    if (this.resultPromise.length > 0) {
                        const results = await Promise.all(this.resultPromise);
                        const toolCalls = results.map(result => this.agent.getToolCallResult(result.callId, result.result));
                        this.write("tool_calls_completed", toolCalls, true);
                        this.resultPromise = [];
                        this.calls = {};
                        await this.invokeWithExpenantialBackoff(model, toolCalls, this, 0);
                    }
                });
            }

            if (Array.isArray(content.data)) {
                for (const data of content.data) {
                    this.processData(data);
                }
            } else {
                this.processData(content.data);
            }

        }
        finally {
            this.jobs--;
        }
    }

    private invokeWithExpenantialBackoff(model: IModel, toolCalls: (ModelContent | ToolCallResponse)[], streamer: IModelStreamer, attempts: number = 0): Promise<void> {
        return new Promise((resolve, reject) => {
            setTimeout(async () => {
                try {
                    await model.invoke(toolCalls, streamer);
                    resolve();
                } catch (e: any) {
                    if (e.isRetryable === true && attempts < 3) {
                        await this.invokeWithExpenantialBackoff(model, toolCalls, streamer, attempts + 1);
                        resolve();
                    }
                    reject(e);
                }
            }, 500 * Math.pow(2, attempts));
        });
    }

    private processData(data: ModelResponseData): void {
            switch (data.typeName) {
                case "tool_call":
                    this.write("tool_call", data, true);
                    if (data.toolCall) {
                        const tool = this.agent.getTool(data.toolCall.name);
                        const toolCallExecutor = async () => {
                            const result = await tool.execute(data.toolCall!.arguments);
                            return {
                                callId: data.toolCall!.callId,
                                result: result,
                            };
                        }
                        this.resultPromise.push(toolCallExecutor());
                        this.calls[data.toolCall!.callId] = toolCallExecutor();
                    }
                    break;
                case "output_text": {
                    this.write("output_text", data, true);
                    break;
                }
                case "output_text.part.done": {
                    this.write("output_text.part.done", data, true);
                    break;
                }
                case "output_text_delta": {
                    this.write("output_text_delta", data, true);
                    break;
                }
                case "response.requested": {
                    this.write("response.requested", data, true);
                    break;
                }
            }
        }
    }

export class Agent {
    public readonly name: string;
    public readonly toolkits: Toolkit[];
    public readonly tools: ITool[];
    public readonly systemPrompt: string;
    public readonly model: IModelFactory;
    public readonly triggers: ITrigger[];
    public readonly outcomes: IOutcome[];
    private modelInstances: Record<string, IModel> = {};
    private readonly toolRecords: Record<string, ITool> = {};

    public constructor(props: AgentProps) {
        this.name = props.name;
        this.toolkits = props.toolkits;
        this.systemPrompt = props.systemPrompt;
        this.model = props.model;
        this.triggers = props.triggers || [];
        this.outcomes = props.outcomes || [];
        this.tools = props.tools || [];
        this.tools = this.tools.concat(this.toolkits.flatMap(
            toolkit => toolkit.tools
        ));
        this.tools.forEach(tool => {
            this.toolRecords[tool.name] = tool;
        });
    }

    public async invoke(prompt: string | (ModelContent | ToolCallResponse)[], threadId?: string, streamer?: IModelStreamer): Promise<IModel> {
        if (!streamer) {
            streamer = new AgentStreamer(this, "NOTSET");
        }
        if (!threadId) {
            const model = await this.model.create({ 
                systemPrompt: this.systemPrompt,
                tools: this.tools,
            });
            await model.invoke([{ content: prompt, role: "user" }], streamer);
            // We have to wait for the first invoke bc that is where the threadId is set
            this.modelInstances[model.threadId!] = model;
            return model;
        } else {
            if (!this.modelInstances[threadId]) {
                throw new Error(`Model instance with threadId ${threadId} not found`);
            }
            const model = this.modelInstances[threadId];
            if (typeof prompt === "string") {
                await model.invoke([{ content: prompt, role: "user" }], streamer);
            } else {
                await model.invoke(prompt, streamer);
            }
            return model;
        }
    }

    public getTool(name: string): ITool {
        const tool = this.toolRecords[name];
        if (!tool) {
            throw new Error(`Tool ${name} not found`);
        }
        return tool;
    }

    public  getToolCallResult(
        callID: string, 
        result: unknown,
    ): Record<string, unknown>  {
        const toolResult = JSON.stringify(result);
        return {
            output: toolResult,
            type: "function_call_output",
            call_id: callID,
        };
    }

    public async respondWithToolCall(
        modelInstance: IModel, 
        callID: string, 
        result: unknown,
        streamer: IModelStreamer): Promise<void> {
        await modelInstance.invoke([this.getToolCallResult(callID, result)], streamer);
    }
}