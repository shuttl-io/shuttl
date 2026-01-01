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
 * Represents a serialized HTTP request from the serve command
 */
export interface SerializedHTTPRequest {
    /** The HTTP method (POST, GET, etc.) */
    readonly method: string;
    /** The request path */
    readonly path: string;
    /** HTTP headers as key-value pairs with array values */
    readonly headers: Record<string, string[]>;
    /** Query parameters as key-value pairs with array values */
    readonly query: Record<string, string[]>;
    /** The request body (parsed JSON or raw) */
    readonly body?: unknown;
    /** The Content-Type header value */
    readonly contentType: string;
    /** The remote address of the client */
    readonly remoteAddr: string;
    /** The host header value */
    readonly host: string;
    /** The HTTP protocol version */
    readonly proto: string;
    /** Timestamp of when the request was received */
    readonly timestamp: string;
}

/**
 * Represents a trigger that can activate an agent. Triggers can take any arguments and then return what the input should be for the agent.
 * Triggers also can validate the arguments and return an error if the arguments are invalid.
 */
export interface ITrigger {
    /**
     * The unique name of this trigger instance.
     * If not set, defaults to triggerType.
     */
    name: string;

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

    /**
     * Sets the name of the trigger
     * @param name - The name to set for the trigger
     * @returns The trigger instance for chaining
     */
    withName(name: string): ITrigger;
}

export abstract class BaseTrigger implements ITrigger {
    public name: string;
    public triggerType: string;
    public triggerConfig: Record<string, unknown> = {};
    public outcome?: IOutcome;

    public constructor(triggerType: string, config: Record<string, unknown>) {
        this.triggerType = triggerType;
        this.name = triggerType; // Default name is the trigger type
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

    public withName(name: string): ITrigger {
        this.name = name;
        return this;
    }
}