import { BaseTrigger, TriggerOutput } from "./ITrigger";
import { IOutcome } from "../outcomes/IOutcomes";

export interface EmailTriggerConfig {
    readonly inboxName: string;
    readonly domain: string;
    readonly fromAddress: string;
    readonly subjectPattern: string;
    readonly folder: string;
}

export class EmailTrigger extends BaseTrigger {
    public triggerType: string = "email";
    public outcome?: IOutcome;

    public constructor(config: EmailTriggerConfig) {
        super("email", config as any);
    }

    public parseArgs(_: any): Promise<TriggerOutput> {
        return Promise.resolve({ input: [{
            typeName: "text",
            text: "The email trigger is activated",
        }] });
    }
}