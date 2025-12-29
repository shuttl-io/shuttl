import { IOutcome } from "./IOutcomes";
import { IModelResponseStream } from "../models/types";

export class Outcomes implements IOutcome {
    private outcomes: IOutcome[];

    public constructor(outcomes: IOutcome[]) {
        this.outcomes = outcomes;
    }
    public static combine(...outcomes: IOutcome[]): Outcomes { 
        return new Outcomes(outcomes);
    }

    public async send(messageStream: IModelResponseStream): Promise<void> {
        let shouldContinue = true;
        while (shouldContinue) {
            const value = await messageStream.next();
            if (value.done) {
                shouldContinue = false;
            }
            const createNext = {
                next: async () => {
                    return value;
                },
            }
            const allOutcomeResults = []
            for (const outcome of this.outcomes) {
                const result = outcome.send(createNext);
                allOutcomeResults.push(result);
            }
            await Promise.all(allOutcomeResults);
        }
        return Promise.resolve();
    }

    public async bindToRequest(request: any): Promise<void> {
        await Promise.all(this.outcomes.map(outcome => outcome.bindToRequest(request)));
    }
}