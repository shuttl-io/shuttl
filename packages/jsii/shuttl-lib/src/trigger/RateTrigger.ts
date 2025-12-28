import { ITrigger, TriggerOutput } from "./ITrigger";
import { InputContent } from "../models/types";

interface RateTriggerConfig {
    /**
     * The cron expression for the rate trigger.
     */
    cronExpression?: string;
    /**
     * The timezone for the cron expression.
     * @default "UTC"
     */
    timezone?: string;
    ms_rate?: number;


    /**
     * The function to call when the trigger is activated.
     * @default null
     */
    onTrigger?: IRateTriggerOnTrigger;
}

export interface IRateTriggerOnTrigger {
    onTrigger(): Promise<InputContent[]>;
}

export class Rate implements ITrigger {
    public triggerType: string = "rate";
    public triggerConfig: Record<string, unknown> = {};
    private onTrigger: (() => Promise<InputContent[]>) | null;

    private constructor(config: RateTriggerConfig) {
        this.triggerConfig = config as any;
        this.onTrigger = config.onTrigger?.onTrigger ?? null;
    }

    public async activate(_: any): Promise<TriggerOutput> {
        if (this.onTrigger) {
            const input = await this.onTrigger();
            return { input };
        } else {
            return { input: [{
                typeName: "text",
                text: "The cron expression is " + this.triggerConfig.cronExpression + " and the timezone is " + this.triggerConfig.timezone,
            }] };
        }
    }

    public withOnTrigger(onTrigger: IRateTriggerOnTrigger): Rate {
        this.onTrigger = onTrigger.onTrigger;
        return this;
    }

    public static milliseconds(value: number): Rate {
        return new Rate({ ms_rate: value });
    }

    public static seconds(value: number): Rate {
        return new Rate({ ms_rate: value * 1000 });
    }

    public static minutes(value: number): Rate {
        return new Rate({ ms_rate: value * 60 * 1000 });
    }

    public static hours(value: number): Rate {
        return new Rate({ ms_rate: value * 60 * 60 * 1000 });
    }

    public static days(value: number): Rate {
        return new Rate({ ms_rate: value * 24 * 60 * 60 * 1000 });
    }
    
    public static weeks(value: number): Rate {
        return new Rate({ ms_rate: value * 7 * 24 * 60 * 60 * 1000 });
    }

    public static months(value: number): Rate {
        return new Rate({ ms_rate: value * 30 * 24 * 60 * 60 * 1000 });
    }

    public static cron(expression: string, timezone?: string): Rate {
        return new Rate({ cronExpression: expression, timezone: timezone ?? "UTC" });
    }
}