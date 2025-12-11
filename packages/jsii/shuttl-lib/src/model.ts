export interface ModelProps {
    readonly identifier: string;
    readonly key: string;
}

export class Model {
    public readonly identifier: string;
    public readonly key: string;

    public constructor(props: ModelProps) {
        this.identifier = props.identifier;
        this.key = props.key;
    }
}