import { ITrigger, TriggerOutput } from "./ITrigger";

export interface EmailTriggerConfig {
    readonly inboxName: string;
    readonly domain: string;
    readonly fromAddress: string;
    readonly subjectPattern: string;
    readonly folder: string;
}

export class EmailTrigger implements ITrigger {
    public triggerType: string = "email";
    public triggerConfig: Record<string, unknown> = {};

    public constructor(config: EmailTriggerConfig) {
        this.triggerConfig = config as any;
    }

    public activate(_: any): Promise<TriggerOutput> {
        return Promise.resolve({ input: [{
            typeName: "text",
            text: "The email trigger is activated",
        }] });
    }
}