import { ISecret } from "./secrets";
import {  OpenAIFactory } from "./models/openAi";
import type { IModelFactory } from "./models/types";

export class Model {
    protected constructor(){}

    public static openAI(identifier: string, apiKey: ISecret, configuration?: Record<string, unknown>): IModelFactory {
        return new OpenAIFactory(identifier, apiKey, configuration ?? {});
    }
}