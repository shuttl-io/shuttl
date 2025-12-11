import { Agent } from "./agent";
import { Toolkit } from "./tools/toolkit";
import { IServer } from "./Server";

export class App {
    readonly name: string;

    //@internal
    public readonly agents: Agent[];
    /**
     * @jsii ignore
     */
    public readonly toolkits: Set<Toolkit>;
    //@internal
    public readonly server: IServer;

    public constructor(name: string, server: IServer) {
        this.name = name;
        this.server = server;
        this.agents = [];
        this.toolkits = new Set();
        this.server.accept(this);
    }
    
    public addAgent(agent: Agent): void {
        this.agents.push(agent);
        agent.toolkits.forEach(toolkit => {
            this.toolkits.add(toolkit);
        });
    }

    public addToolkit(toolkit: Toolkit): void {
        this.toolkits.add(toolkit);
    }

    public serve(): void {
        this.server.start();
    }
   
}