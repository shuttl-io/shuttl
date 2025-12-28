import { InputContent } from "../models/types";

export interface TriggerOutput {
    readonly input: InputContent[];
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

    /**
     * Activates the trigger and returns the input for the agent.
     * @param args - The arguments for the trigger.
     * @returns The input for the agent.
     */
    activate(args: any): Promise<TriggerOutput>;

    /**
     * Validates the arguments for the trigger.
     * @param args - The arguments for the trigger.
     * @returns The validation result.
     */
    validate?(args: any): Promise<Record<string, unknown>>;
}