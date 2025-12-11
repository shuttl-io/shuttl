export interface ToolArg {
    readonly name: string;
    readonly argType: string;
    readonly description: string;
    readonly required: boolean;
    readonly defaultValue: unknown;
}
export interface ITool {
    name: string;
    description: string;
    execute(args: Record<string, unknown>): unknown;
    produceArgs(): Record<string, ToolArg>;
}
