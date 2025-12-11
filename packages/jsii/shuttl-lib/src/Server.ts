
export interface IServer {
    accept(app: unknown): void;
    start(): Promise<void>;
    stop(): Promise<void>;
}

