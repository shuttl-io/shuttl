import { IOutcome } from "./IOutcomes";
import { IModelResponseStream } from "../models/types";

export class StreamingOutcome implements IOutcome {
    public send(messageStream: IModelResponseStream): Promise<void> {
        console.log("Sending message to Slack:", messageStream);
        return Promise.resolve();
    }

    public bindToRequest(request: any): Promise<void> {
        console.log("Binding request to Slack:", request);
        return Promise.resolve();
    }
}