export interface IOutcome {
   send(message: any) : Promise<void>;
}


export class SlackOutcome implements IOutcome {
    public send(message: any): Promise<void> {
        console.log("Sending message to Slack:", message);
        return Promise.resolve();
    }
}