import { ITool } from "../tools/tool";

export interface InputContent {
    readonly typeName: "text" | "image" | "file";
    readonly text?: string;
    readonly image?: string;
    readonly file?: string;
}
export const isInputContent = (content: any): content is InputContent => {
    return typeof(content) !== "string" 
        && !Array.isArray(content) 
        && content !== null 
        && typeof content === "object"
        && "typeName" in content;
}

export const isInputContentArray = (content: any): content is InputContent[] => {
    return Array.isArray(content) && content.every(isInputContent);
}

export interface ModelContent {
    readonly content: string | InputContent | InputContent[];
    readonly role: "user" | "assistant" | "system";
}

/**
 * @internal
 */
export interface GenericArguments {
    [key: string]: any;
}

export interface ModelToolOutput {
    readonly outputType: "tool_call";
    readonly name: string;
    readonly arguments: Record<string, unknown>;
    readonly callId: string;
}

export interface ModelTextOutput {
    readonly outputType: "output_text";
    readonly text: string;
}

export interface InputTokensDetails {
    readonly cachedTokens: number;
}

export interface OutputTokensDetails {
    readonly reasoningTokens: number;
}

export interface ModelDeltaOutput {
    readonly outputType: "output_text_delta";
    readonly delta: string;
    readonly sequenceNumber: number;
}

export interface Usage {
    readonly inputTokens: number;
    readonly inputTokensDetails: InputTokensDetails;
    readonly outputTokens: number;
    readonly outputTokensDetails: OutputTokensDetails;
    readonly totalTokens: number;
}
export interface ModelResponseData {
    readonly typeName: "response.requested" | "tool_call" | "output_text" | "output_text_delta" | "response.completed" | "output_text.part.done";
    readonly toolCall?: ModelToolOutput;
    readonly outputText?: ModelTextOutput;
    readonly outputTextDelta?: ModelDeltaOutput;
    readonly requested?: any
}

export interface ModelResponse {
    readonly eventName: string;
    readonly data: ModelResponseData[] | ModelResponseData;
    readonly usage?: Usage;
    readonly modelInstance?: IModel;
}

export type ToolCallResponse = Record<string, unknown>;

export interface IModelStreamer {
    recieve(model: IModel, content: ModelResponse): Promise<void>;
}

export interface IModel {
    readonly threadId?: string;
    // isBlocked(): boolean;
    invoke(prompt: (ModelContent | ToolCallResponse)[], streamer: IModelStreamer): Promise<void>;
}

export interface IModelFactoryProps {
    readonly systemPrompt: string;
    readonly tools?: ITool[];
}

export interface IModelFactory {
    create(props: IModelFactoryProps): Promise<IModel>;
}