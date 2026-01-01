import { ISecret } from "../secrets";
import { 
    IModel, 
    IModelFactory, 
    IModelFactoryProps,
    ModelContent,
    IModelStreamer,
    ModelResponse,
    ToolCallResponse,
    ModelResponseData,
    isInputContentArray,
    isInputContent,
    InputContent,
} from "./types";
import { ITool } from "../tools/tool";

export class OpenAIError extends Error {
    public readonly isRetryable: boolean = true;
    constructor(
        message: string, 
        public readonly statusCode: number, 
        public readonly error: any,
    ) {
        super(message);
        this.isRetryable = true;
    }
}
export class OpenAIBadKeyError extends Error {
    public readonly isRetryable: boolean = false;
    constructor(message: string, public readonly statusCode: number, public readonly error: any) {
        super(message);
        this.isRetryable = false;
    }
}

export class OpenAI implements IModel {
    private readonly messages: ModelContent[] = [];
    private threadIDPromise: Promise<string>;
    public threadId?: string;
    private inputs: (ModelContent | ToolCallResponse)[] = [];
    public isDoneReceiving: boolean = true;

    public constructor(
        public readonly identifier: string,
        public readonly apiKey: string,
        public readonly systemPrompt: string,
        public readonly tools?: ITool[],
    ){

        // this.threadId = crypto.randomUUID();
        this.threadIDPromise = this.getThreadId();
        this.messages.push({
            content: systemPrompt,
            role: "system",
        });
    }

    private async getThreadId(): Promise<string> {
        // Make a request to the OpenAI API to create a new conversation. Use Node primitives to make the request.
        const response = await fetch("https://api.openai.com/v1/conversations", {
            method: "POST",
            headers: {
                "Authorization": `Bearer ${this.apiKey}`,
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                items: [
                    {
                        role: "system",
                        content: this.systemPrompt,
                    }
                ]
            }),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new OpenAIError("failed to create thread", response.status, error);
        }
        const data = await response.json() as { id: string, object: string };
        const threadId = data.id;
        return threadId;
    }

    public async invoke(prompt: ModelContent[], streamer: IModelStreamer): Promise<void> {
        // If the model is not done receiving, wait for it to be done
        if (!this.isDoneReceiving) {
            return new Promise((resolve) => {
                setTimeout(async () => {
                    await this.invoke(prompt, streamer);
                    resolve();
                }, 0);
            });
        }
        this.isDoneReceiving = false;
        this.inputs.push(...prompt);
        if (!this.threadId) {
            this.threadId = await this.threadIDPromise;
        }
        // this.inputs.push(...prompt);
        const stream = true;
        const tools = this.tools?.map(tool => ({
            type: "function",
            name: tool.name,
            description: tool.description,
            parameters: {
                type: "object",
                properties: Object.entries(tool.schema?.properties ?? {}).reduce((acc, [name, arg]) => {
                    const property: Record<string, any> = {
                        type: arg.argType,
                        description: arg.description,
                        default: arg.defaultValue,
                    };
                    if (arg.enumValues) {
                        property.enum = arg.enumValues;
                    }
                    acc[name] = property;
                    return acc;
                }, {} as Record<string, any>),
                required: Object.entries(tool.schema?.properties ?? {}).filter(([_, arg]) => arg.required).map(([name]) => name),
            }
        }));
        const body = {
                model: this.identifier,
                conversation: this.threadId,
                input: this.inputs.map(input => this.createInput(input)),
                parallel_tool_calls: false,
                stream,
                tools: tools ?? [],
        }
        streamer.recieve(this, {
            eventName: "response.requested",
            data: {
                typeName: "response.requested",
                requested: body,
                threadId: this.threadId,
            },
        });
        try {
            const response = await fetch(`https://api.openai.com/v1/responses`, {
                method: "POST",
                headers: {
                    "Authorization": `Bearer ${this.apiKey}`,
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(body),
            });
            if (!response.ok) {
                const error = await response.json() as any;
                if (response.status === 401 && error.error.code === "invalid_api_key") {
                    throw new OpenAIBadKeyError("bad API key", response.status, error);
                }
                throw new OpenAIError("failed to invoke model", response.status, error);
            }
            if (stream) {
                const reader = await response.body?.getReader();
                const decoder = new TextDecoder();
                let buffer = "";

                while (true) {
                    const result = await reader?.read();
                    if (!result) {
                        break;
                    }
                    const { done, value } = result;
                    if (done) {
                        streamer.recieve(this, {
                            eventName: "overall.completed",
                            data: {
                                typeName: "overall.completed",
                            },
                            threadId: this.threadId,
                        });
                        break;
                    }
                    
                    buffer += decoder.decode(value, { stream: true });
                    
                    // Process complete messages (delimited by double newlines)
                    let separatorIndex: number;
                    while ((separatorIndex = buffer.indexOf("\n\n")) !== -1) {
                        const rawMessage = buffer.slice(0, separatorIndex);
                        buffer = buffer.slice(separatorIndex + 2);
                        
                        if (!rawMessage.trim()) {
                            continue;
                        }
                        
                        // Raw data is a string in the format of `event: <event_name>\ndata: <event_data>`
                        const eventMatch = rawMessage.match(/^event: (.+)$/m);
                        const dataMatch = rawMessage.match(/^data: (.+)$/m);
                        
                        if (eventMatch && dataMatch) {
                            const eventName = eventMatch[1];
                            const eventData = dataMatch[1];
                            const data = JSON.parse(eventData) as any;
                            const response = this.createResponse(eventName, data);
                            streamer.recieve(this, response);
                        }
                    }
                }
               
            } else {
                const data = await response.json();
                streamer.recieve(this, data as any);
                streamer.recieve(this, {
                    eventName: "overall.completed",
                    data: {
                        typeName: "overall.completed",
                    },
                    threadId: this.threadId,
                });
            }

        }
        finally {
            this.isDoneReceiving = true;
        }
        
    }

    private createInputContent(content: InputContent) {
        
        if (content.typeName === "image") {
            return {
                type: "input_image",
                image_url: content.image,
            };
        }
        if (content.typeName === "file") {
            // Handle file attachments with base64 content
            if (content.fileData) {
                return {
                    type: "input_file",
                    filename: content.fileData.name,
                    file_data: `data:${content.fileData.mimeType || "application/octet-stream"};base64,${content.fileData.content}`,
                };
            }
            // Fallback to URL-based file reference
            return {
                type: "input_file",
                file_url: content.file,
            };
        }
        return {
            type: "input_text",
            text: content.text,
        };
    }

    private createInput(input: ModelContent | ToolCallResponse): ModelContent | ToolCallResponse {
        if ("role" in input) {
            const realInput = input as ModelContent;
            // if content is not a string, content should be transformed to a chat GPT input
            if (typeof realInput.content !== "string") {
                if (isInputContentArray(realInput.content)) {
                    const content = realInput.content.flatMap(c => this.createInputContent(c));
                    return {role: realInput.role, content: content};
                } else if (isInputContent(realInput.content)) {
                    let content = this.createInputContent(realInput.content);
                    if (Array.isArray(content)) {
                        return { role: realInput.role, content: content };
                    }
                    return { role: realInput.role, content: [content] };
                }
            }
            return { role: realInput.role, content: realInput.content };
        }
        return input as ToolCallResponse;
    }

    private createResponse(eventName: string, data: any): ModelResponse {
        switch(eventName) {
            case "response.completed":
                const response = data.response.output.reduce((acc: ModelResponseData[], output: any) => {
                    if (output.type === "function_call") {
                        acc.push({
                            typeName: "tool_call",
                            toolCall: output,
                        });
                    }
                    if (output.type === "message") {
                        output.content.forEach((content: any) => {
                            acc.push({
                                typeName: "output_text",
                                outputText: {
                                    outputType: "output_text",
                                    text: content.text,
                                },
                            });
                        });
                    }
                    return acc;
                }, []);
                return {
                    eventName: "response.completed",
                    modelInstance: this,
                    data: {
                        typeName: "response.completed",
                        ...response,
                    },
                    usage: data.response.usage,
                };
            case "response.output_text.delta":
                return {
                    eventName: "response.output_text_delta",
                    modelInstance: this,
                    data: {
                        typeName: "output_text_delta",
                        outputTextDelta: {
                            outputType: "output_text_delta",
                            delta: data.delta,
                            sequenceNumber: data.sequence_number,
                        },
                    },
                };
            case "response.content_part.done":
                return {
                    eventName: "response.content_part.done",
                    modelInstance: this,
                    data: {
                        typeName: "output_text",
                        outputText: {
                            outputType: "output_text",
                            text: data.part.text,
                        },
                    },
                };
            case "response.output_item.done":
                const output = data.item;
                if (output.type === "message") {
                    return {
                        eventName: "response.output_text.done",
                        modelInstance: this,
                        data: {
                            typeName: "output_text.part.done",
                            outputText: {
                                outputType: "output_text",
                                text: output.content[0].text,
                            }
                        },
                    };
                }
                return {
                    eventName: "response.completed",
                    modelInstance: this,
                    data: {
                        typeName: "tool_call",
                        toolCall: {
                            outputType: "tool_call",
                            name: output.name,
                            arguments: JSON.parse(output.arguments),
                            callId: output.call_id,
                        },
                    },
                };
            default:
                return data;
        }
    }
}

export class OpenAIFactory implements IModelFactory {
    constructor(
        public readonly identifier: string,
        public readonly apiKey: ISecret,
    ){
        this.create = this.create.bind(this);
    }

    public async create(props: IModelFactoryProps): Promise<IModel> {
        return new OpenAI(
            this.identifier,
            await this.apiKey.resolveSecret(),
            props.systemPrompt,
            props.tools,
        );
    }
}