import { IModelResponseStream } from "../models/types";

export interface IOutcome {
   send(messageStream: IModelResponseStream) : Promise<void>;
   bindToRequest(request: any) : Promise<void>;
}

export class SlackOutcome implements IOutcome {
    public send(messageStream: IModelResponseStream): Promise<void> {
        console.log("Sending message to Slack:", messageStream);
        return Promise.resolve();
    }

    public bindToRequest(request: any): Promise<void> {
        console.log("Binding request to Slack:", request);
        return Promise.resolve();
    }
}