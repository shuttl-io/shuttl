import { Model } from "./model";
import { Toolkit } from "./tools/toolkit";

export interface AgentProps {
    readonly name: string;
    readonly toolkits: Toolkit[];
    readonly systemPrompt: string;
    readonly model: Model;
}

export class Agent {
    public readonly name: string;
    public readonly toolkits: Toolkit[];
    public readonly systemPrompt: string;
    public readonly model: Model;

    public constructor(props: AgentProps) {
        this.name = props.name;
        this.toolkits = props.toolkits;
        this.systemPrompt = props.systemPrompt;
        this.model = props.model;
    }
   
}