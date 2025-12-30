import { IOutcome } from "../outcomes/IOutcomes";
import { BaseTrigger, TriggerOutput } from "./ITrigger";
import { z } from "zod";

export interface FileTriggerConfig {
    readonly allowedExtensions: string[];
    readonly maxFileSize: number;
    readonly uploadPath: string;
    readonly s3Bucket: string;
}

const FileTriggerSchema = z.object({
    content: z.string(),
    file_url: z.string(),
    mimeType: z.string().optional(),
    name: z.string(),
});

export class FileTrigger extends BaseTrigger {
    public triggerType: string = "file";

    public outcome?: IOutcome;

    public constructor(config: FileTriggerConfig) {
        super("file", config as any);
    }

    public parseArgs(args: any): Promise<TriggerOutput> {
        const parsedArgs = FileTriggerSchema.safeParse(args);
        if (!parsedArgs.success) {
            throw new Error(parsedArgs.error.message);
        }
        return Promise.resolve({ input: [{
            typeName: "file",
            fileData: {
                content: parsedArgs.data.content,
                name: parsedArgs.data.name,
                mimeType: parsedArgs.data.mimeType,
            },
        }] });
    }
}