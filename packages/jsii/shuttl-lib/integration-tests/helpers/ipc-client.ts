import { spawn, ChildProcess, SpawnOptions } from "child_process";
import { EventEmitter } from "events";
import * as readline from "readline";

/**
 * Request message format for IPC communication
 */
export interface IPCRequest {
    id: string;
    method: string;
    body?: Record<string, unknown>;
}

/**
 * Response message format from the server
 */
export interface IPCResponse {
    id: string;
    success: boolean;
    result?: unknown;
    errorObj?: {
        code: string;
        message: string;
    };
}

/**
 * Options for spawning a test app
 */
export interface SpawnAppOptions {
    /** The command to run */
    command: string;
    /** Arguments to pass to the command */
    args?: string[];
    /** Working directory */
    cwd?: string;
    /** Environment variables */
    env?: NodeJS.ProcessEnv;
    /** Additional spawn options */
    spawnOptions?: SpawnOptions;
}

/**
 * IPC Client for communicating with test apps via stdin/stdout
 */
export class IPCClient extends EventEmitter {
    private process: ChildProcess | null = null;
    private readline: readline.Interface | null = null;
    private pendingRequests: Map<string, {
        resolve: (response: IPCResponse) => void;
        reject: (error: Error) => void;
        timeout: NodeJS.Timeout;
    }> = new Map();
    private requestCounter = 0;
    private readyPromise: Promise<IPCResponse> | null = null;
    private readyResolve: ((response: IPCResponse) => void) | null = null;
    private stderrBuffer: string[] = [];

    /**
     * Spawn a new process and establish IPC communication
     */
    async spawn(options: SpawnAppOptions): Promise<IPCResponse> {
        const { command, args = [], cwd, env, spawnOptions = {} } = options;

        // Create the ready promise before spawning
        this.readyPromise = new Promise((resolve) => {
            this.readyResolve = resolve;
        });

        this.process = spawn(command, args, {
            cwd,
            env: { ...process.env, ...env },
            stdio: ['pipe', 'pipe', 'pipe'],
            ...spawnOptions,
        });

        // Set up readline for parsing NDJSON responses
        this.readline = readline.createInterface({
            input: this.process.stdout!,
            terminal: false,
        });

        this.readline.on('line', (line) => {
            this.handleLine(line);
        });

        // Capture stderr for debugging
        this.process.stderr?.on('data', (data) => {
            const str = data.toString();
            this.stderrBuffer.push(str);
            this.emit('stderr', str);
        });

        // Handle process exit
        this.process.on('exit', (code, signal) => {
            this.emit('exit', code, signal);
            this.cleanup();
        });

        this.process.on('error', (error) => {
            this.emit('error', error);
            this.rejectAllPending(error);
        });

        // Wait for the ready signal
        const readyResponse = await this.readyPromise;
        return readyResponse;
    }

    /**
     * Handle a line of output from the process
     */
    private handleLine(line: string): void {
        const trimmed = line.trim();
        if (!trimmed) return;

        try {
            const response = JSON.parse(trimmed) as IPCResponse;
            
            // Check if this is the ready signal
            if (response.id === '__ready__' && this.readyResolve) {
                this.readyResolve(response);
                this.readyResolve = null;
                return;
            }

            // Find the pending request
            const pending = this.pendingRequests.get(response.id);
            if (pending) {
                clearTimeout(pending.timeout);
                this.pendingRequests.delete(response.id);
                pending.resolve(response);
            }

            this.emit('response', response);
        } catch (e) {
            this.emit('parseError', line, e);
        }
    }

    /**
     * Send a request to the process and wait for a response
     */
    async send(method: string, body?: Record<string, unknown>, timeoutMs = 10000): Promise<IPCResponse> {
        if (!this.process || !this.process.stdin) {
            throw new Error('Process not spawned or stdin not available');
        }

        const id = `req_${++this.requestCounter}`;
        const request: IPCRequest = { id, method, body };

        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                this.pendingRequests.delete(id);
                reject(new Error(`Request ${id} timed out after ${timeoutMs}ms`));
            }, timeoutMs);

            this.pendingRequests.set(id, { resolve, reject, timeout });

            const json = JSON.stringify(request) + '\n';
            this.process!.stdin!.write(json, (err) => {
                if (err) {
                    clearTimeout(timeout);
                    this.pendingRequests.delete(id);
                    reject(err);
                }
            });
        });
    }

    /**
     * Send a ping request
     */
    async ping(): Promise<IPCResponse> {
        return this.send('ping');
    }

    /**
     * Get app information
     */
    async getAppInfo(): Promise<IPCResponse> {
        return this.send('getAppInfo');
    }

    /**
     * List all agents
     */
    async listAgents(): Promise<IPCResponse> {
        return this.send('listAgents');
    }

    /**
     * List all toolkits
     */
    async listToolkits(): Promise<IPCResponse> {
        return this.send('listToolkits');
    }

    /**
     * Invoke a tool
     */
    async invokeTool(toolkit: string, tool: string, args: Record<string, unknown> = {}): Promise<IPCResponse> {
        return this.send('invokeTool', { toolkit, tool, args });
    }

    /**
     * Send shutdown request
     */
    async shutdown(): Promise<IPCResponse> {
        return this.send('shutdown');
    }

    /**
     * Get captured stderr output
     */
    getStderr(): string {
        return this.stderrBuffer.join('');
    }

    /**
     * Kill the process
     */
    kill(signal: NodeJS.Signals = 'SIGTERM'): void {
        if (this.process) {
            this.process.kill(signal);
        }
    }

    /**
     * Wait for the process to exit
     */
    async waitForExit(timeoutMs = 5000): Promise<number | null> {
        if (!this.process) return null;

        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error(`Process did not exit within ${timeoutMs}ms`));
            }, timeoutMs);

            this.process!.on('exit', (code) => {
                clearTimeout(timeout);
                resolve(code);
            });
        });
    }

    /**
     * Cleanup resources
     */
    private cleanup(): void {
        if (this.readline) {
            this.readline.close();
            this.readline = null;
        }
        this.rejectAllPending(new Error('Process exited'));
    }

    /**
     * Reject all pending requests
     */
    private rejectAllPending(error: Error): void {
        for (const [_id, pending] of this.pendingRequests) {
            clearTimeout(pending.timeout);
            pending.reject(error);
        }
        this.pendingRequests.clear();
    }

    /**
     * Close the client and kill the process
     */
    async close(): Promise<void> {
        try {
            await this.shutdown();
        } catch {
            // Ignore shutdown errors
        }
        this.kill();
        this.cleanup();
    }
}

/**
 * Create and spawn an IPC client
 */
export async function createIPCClient(options: SpawnAppOptions): Promise<IPCClient> {
    const client = new IPCClient();
    await client.spawn(options);
    return client;
}

