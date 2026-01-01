import { IOutcome } from "./IOutcomes";
import { IModelResponseStream } from "../models/types";
import { stdout } from "process";

export class StreamingOutcome implements IOutcome {
    public async send(messageStream: IModelResponseStream): Promise<void> {
        console.log("StreamingOutcome.send");
        let shouldContinue = true;
        while (shouldContinue) {
            const value = await messageStream.next();
            if (value.done) {
                shouldContinue = false;
            } else {
                stdout.write(JSON.stringify(value.value) + "\n");
            }
        }
        return;
    }

    public bindToRequest(request: any): Promise<void> {
        console.log("Binding request to Slack:", request);
        return Promise.resolve();
    }
}