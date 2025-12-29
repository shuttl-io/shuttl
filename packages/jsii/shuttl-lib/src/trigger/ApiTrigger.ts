import { z } from "zod";
import { InputContent } from "../models/types";
import { BaseTrigger, TriggerOutput } from "./ITrigger";
import { IOutcome } from "../outcomes/IOutcomes";

function assertNever(value: never): never {
    throw new Error(`Unexpected value: ${value}`);
}

export interface IApiAuthenticator {
    authenticate(args: ApiTriggerArgs): Promise<boolean>;
}

export interface ApiTriggerConfig {
    readonly cors?: string[];
    readonly authenticator?: IApiAuthenticator;
}

const ApiTriggerSchema = z.object({
    method: z.enum(["POST", "GET", "PUT", "PATCH", "DELETE"]),
    hostName: z.string(),
    headers: z.record(z.string(), z.string()).optional(),
    body: z.any().optional(),
    queryParams: z.record(z.string(), z.string()).optional(),
    pathParams: z.record(z.string(), z.string()).optional(),
    cookies: z.record(z.string(), z.string()).optional(),
    auth: z.object({
        authType: z.enum(["bearer", "basic", "api_key"]),
        value: z.string(),
    }).optional(),
});

const fileAttachmentSchema = z.object({
    type: z.literal("file"),
    name: z.string(),
    content: z.string(),
    mimeType: z.string().optional(),
});

const imageSchema = z.object({
    type: z.literal("image"),
    content: z.string(),
    name: z.string(),
    mimeType: z.string().optional(),
});

const textSchema = z.object({
    type: z.literal("text"),
    content: z.string(),
});

const bodySchema1 = z.discriminatedUnion("type", [fileAttachmentSchema, imageSchema, textSchema]);

const bodySchema = z.union([bodySchema1, z.array(bodySchema1)]);

export interface ApiAuth {
    readonly authType: "bearer" | "basic" | "api_key";
    readonly value: string;
}

export interface ApiTriggerArgs {
    readonly method: "POST" | "GET" | "PUT" | "PATCH" | "DELETE";
    readonly hostName: string;
    readonly headers?: Record<string, string> | undefined;
    readonly body?: any;
    readonly queryParams?: Record<string, string> | undefined;
    readonly pathParams?: Record<string, string> | undefined;
    readonly cookies?: Record<string, string> | undefined;
    readonly auth?: ApiAuth | undefined;
}

/**
 * Represents a trigger that can activate an agent via an API call.
 * This API trigger is the default trigger for agents.
 */
export class ApiTrigger extends BaseTrigger  {
    public triggerType: string = "api";
    public triggerConfig: Record<string, unknown> = {};
    public outcome?: IOutcome;
    private authenticator?: IApiAuthenticator;

    public constructor(config?: ApiTriggerConfig) {
        super("api", config ?? {
            cors: ["*"],
            authenticator: async (_: ApiTriggerArgs) => true,
        } as any);
        this.authenticator = config?.authenticator ?? undefined;
    }

    public parseArgs(rawArgs: any): Promise<TriggerOutput> {
        const args = ApiTriggerSchema.parse(rawArgs);
        const body = bodySchema.parse(args.body);
        if (!Array.isArray(body)) {
            return Promise.resolve({ input: [this.createInputContent(body)] });
        }
        return Promise.resolve({ input: body.map((b) => this.createInputContent(b)) });
    }

    private createInputContent(body: z.infer<typeof bodySchema1>): InputContent {
        if (body.type === "text") {
            return { typeName: "text", text: body.content };
        }
        if (body.type === "file") {
            return { typeName: "file", fileData: {
                content: body.content,
                name: body.name,
                mimeType: body.mimeType,
            } };
        }
        if (body.type === "image") {
            return { 
                typeName: "image", 
                fileData: {
                    content: body.content,
                    name: body.name,
                    mimeType: body.mimeType,
                }
            };
        }
        assertNever(body);
    }

    public async validate(rawArgs: any): Promise<Record<string, unknown>> {
        const args = ApiTriggerSchema.safeParse(rawArgs);
        if (!args.success) {
            return Promise.resolve({ error: "Invalid arguments", details: args.error.issues });
        }
        if (!this.authenticator) {
            return Promise.resolve({});
        }
        const authenticator = this.authenticator;
        const authenticated = await authenticator.authenticate(args.data);
        if (!authenticated) {
            return Promise.resolve({ error: "Unauthorized" });
        }
        return Promise.resolve({});
    }
}
