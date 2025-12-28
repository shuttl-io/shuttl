import { Agent } from "./agent";
import { Toolkit } from "./tools/toolkit";
import { IServer } from "./Server"
import { stderr } from "process";

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
        let originalConsoleLog: (...args: any[]) => void | undefined;
        if (process.env._SHUTTL_CONTROL === "true") {
            originalConsoleLog = console.log.bind(console);
            this.monkeyPatchConsoleLog();
            this.server.start();
            console.log = originalConsoleLog;
        } else {
            this.server.start();
        }
    }

    //Monkey patch console.log to log to stderr
    private monkeyPatchConsoleLog() {
        console.log = (...args: any[]) => {
            stderr.write(args.join(" ") + "\n");
        };
    }
}