export interface ToolArg {
    readonly name?: string;
    readonly argType: string;
    readonly description: string;
    readonly required: boolean;
    readonly defaultValue?: unknown;
    readonly enumValues?: string[];
}
export interface ITool {
    name: string;
    description: string;
    schema?: Schema;
    execute(args: Record<string, unknown>): unknown;
}

export class ToolArgBuilder {
    public argType: string;
    public description: string;
    public required: boolean;
    public defaultValue?: unknown;
    public enumValues?: string[];

    public constructor(argType: string, description: string, required: boolean, defaultValue?: unknown, enumValues?: string[]) {
        this.argType = argType;
        this.description = description;
        this.required = required;
        this.defaultValue = defaultValue;
        this.enumValues = enumValues;
    }

    public isRequired(): ToolArgBuilder {
        this.required = true;
        return this;
    }

    public defaultTo(defaultValue: unknown): ToolArgBuilder {
        this.defaultValue = defaultValue;
        return this;
    }
}

export class Schema {
    private constructor(public readonly properties: Record<string, ToolArgBuilder>) {}
    public static objectValue(properties: Record<string, ToolArgBuilder>): Schema {
        return new Schema(properties);
    }
    public static stringValue(description: string): ToolArgBuilder {
        return new ToolArgBuilder( "string", description, false);
    }
    public static numberValue(description: string): ToolArgBuilder {
        return new ToolArgBuilder( "number", description, false);
    }
    public static booleanValue(description: string): ToolArgBuilder {
        return new ToolArgBuilder( "boolean", description, false);
    }

    public static enumValue(description: string, enumValues: string[]): ToolArgBuilder {
        return new ToolArgBuilder("string", description, false, undefined, enumValues);
    }
}