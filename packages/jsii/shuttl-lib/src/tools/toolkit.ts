import { ITool } from "./tool";

export interface ToolkitProps {
    readonly name: string;
    readonly description?: string;
    readonly tools?: ITool[];
}

export class Toolkit {
    public readonly name: string;
    public readonly description: string | undefined;
    public readonly tools: ITool[];

    public constructor(props: ToolkitProps) {
        this.name = props.name;
        this.description = props.description;
        this.tools = props.tools ?? [];
    }

    public addTool(tool: ITool): void {
        this.tools.push(tool);
    }
}