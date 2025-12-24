export interface ITrigger {
    name: string;
    triggerType: string;
    description: string;
    args?: any;
}

export class Rate {
    public value: number;
    private constructor(value: number) {
        this.value = value;
    }
    
    public static milliseconds(value: number): Rate {
        return new Rate(value);
    }
    
    public static seconds(value: number): Rate {
        return new Rate(value * 1000);
    }

    public static minutes(value: number): Rate {
        return new Rate(value * 60 * 1000);
    }
    
    public static hours(value: number): Rate {
        return new Rate(value * 60 * 60 * 1000);
    }

    public static days(value: number): Rate {
        return new Rate(value * 24 * 60 * 60 * 1000);
    }
    
    public static weeks(value: number): Rate {
        return new Rate(value * 7 * 24 * 60 * 60 * 1000);
    }

    public static months(value: number): Rate {
        return new Rate(value * 30 * 24 * 60 * 60 * 1000);
    }
    
}


export class Trigger implements ITrigger {
    public name: string;
    public triggerType: string;
    public description: string;
    public args?: any;

    private constructor(props: ITrigger) {
        this.name = props.name;
        this.triggerType = props.triggerType;
        this.description = props.description;
        this.args = props.args;
    }

    public static onCron(cronExpression: string, name?: string): Trigger {
        return new Trigger({
            name: name || "Cron Trigger",
            triggerType: "cron",
            description: "Trigger that activates on a cron schedule",
            args: {
                cronExpression: cronExpression,
            },
        });
    }

    public static onRate(rate: Rate, name?: string): Trigger {
        return new Trigger({
            name: name || "Rate Trigger",
            triggerType: "rate",
            description: "Trigger that activates on a rate schedule",
            args: {
                rate: rate,
            },
        });
    }

    public static onEvent(name: string): Trigger {
        return new Trigger({
            name: name,
            triggerType: "event",
            description: "Trigger that activates on an event",
        });
    }

    public static onFileUpload(name: string): Trigger {
        return new Trigger({
            name: name,
            triggerType: "file_upload",
            description: "Trigger that activates on a file upload",
        });
    }

    public static onSlack(name: string, channel: string): Trigger {
        return new Trigger({
            name: name,
            triggerType: "slack",
            description: "Trigger that activates on a slack message",
            args: {
                channel: channel,
            },
        });
    }
}