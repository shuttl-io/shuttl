import { InputContent, IModelResponseStream } from "../models/types";
import { IOutcome } from "../outcomes/IOutcomes";

export interface TriggerOutput {
    readonly input: InputContent[];
}

export interface ITriggerInvoker {
    invoke(prompt: InputContent[]): Promise<IModelResponseStream>;
    defaultOutcome(stream: IModelResponseStream): Promise<void>;
}

/**
 * Represents a trigger that can activate an agent. Triggers can take any arguments and then return what the input should be for the agent.
 * Triggers also can validate the arguments and return an error if the arguments are invalid.
 */
export interface ITrigger {
    /**
     * The type of trigger.
     */
    triggerType: string;
    /**
     * The configuration for the trigger.
     */
    triggerConfig: Record<string, unknown>;

    outcome?: IOutcome;

    /**
     * Activates the trigger and returns the input for the agent.
     * @param args - The arguments for the trigger.
     * @returns The input for the agent.
     */
    activate(args: any, invoker: ITriggerInvoker): Promise<void>;

    /**
     * Validates the arguments for the trigger.
     * @param args - The arguments for the trigger.
     * @returns The validation result.
     */
    validate?(args: any): Promise<Record<string, unknown>>;

    /**
     * binds the outcome to the trigger
     * @param outcome - The outcome to bind to the trigger.
     * @returns The bound outcome.
     */
    bindOutcome(outcome: IOutcome): ITrigger;
}

export abstract class BaseTrigger implements ITrigger {
    public triggerType: string;
    public triggerConfig: Record<string, unknown> = {};
    public outcome?: IOutcome;

    public constructor(triggerType: string, config: Record<string, unknown>) {
        this.triggerType = triggerType;
        this.triggerConfig = config;
    }

    public abstract parseArgs(args: any): Promise<TriggerOutput>;

    public async activate(args: any, invoker: ITriggerInvoker): Promise<void> {
        const parsedArgs = await this.parseArgs(args);
        const response = await invoker.invoke(parsedArgs.input);
        if (this.outcome) {
            await this.outcome.send(response);
        } else {
            await invoker.defaultOutcome(response);
        }
    }

    public bindOutcome(outcome: IOutcome): ITrigger {
        this.outcome = outcome;
        return this;
    }
}