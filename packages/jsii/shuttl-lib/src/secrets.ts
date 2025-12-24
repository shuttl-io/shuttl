export interface ISecret {
    resolveSecret(): Promise<string>;
}

export class EnvSecret implements ISecret {
    public constructor(public readonly envVarName: string){}
    public async resolveSecret(): Promise<string> {
        return process.env[this.envVarName] || "";
    }
}

export class Secret  {
    public source: "env" | "file" | "shuttl";
    public name: string;
    private constructor(source: "env" | "file" | "shuttl", name: string) {
        this.source = source;
        this.name = name;
    }

    /**
     * Create a Secret from an environment variable.    
     * @param envVarName The name of the environment variable.
     * @returns A new Secret.
     */
    public static fromEnv(envVarName: string): ISecret {
        return new EnvSecret(envVarName);
    }
}