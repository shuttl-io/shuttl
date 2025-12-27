import { Rate, Trigger, ITrigger } from "../src/Triggers";

describe("Rate", () => {
    describe("milliseconds()", () => {
        it("should create a Rate with the exact millisecond value", () => {
            const rate = Rate.milliseconds(500);

            expect(rate.value).toBe(500);
        });

        it("should handle zero milliseconds", () => {
            const rate = Rate.milliseconds(0);

            expect(rate.value).toBe(0);
        });

        it("should handle large millisecond values", () => {
            const rate = Rate.milliseconds(1000000);

            expect(rate.value).toBe(1000000);
        });
    });

    describe("seconds()", () => {
        it("should convert seconds to milliseconds", () => {
            const rate = Rate.seconds(1);

            expect(rate.value).toBe(1000);
        });

        it("should handle multiple seconds", () => {
            const rate = Rate.seconds(30);

            expect(rate.value).toBe(30000);
        });

        it("should handle zero seconds", () => {
            const rate = Rate.seconds(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional seconds", () => {
            const rate = Rate.seconds(1.5);

            expect(rate.value).toBe(1500);
        });
    });

    describe("minutes()", () => {
        it("should convert minutes to milliseconds", () => {
            const rate = Rate.minutes(1);

            expect(rate.value).toBe(60000);
        });

        it("should handle multiple minutes", () => {
            const rate = Rate.minutes(5);

            expect(rate.value).toBe(300000);
        });

        it("should handle zero minutes", () => {
            const rate = Rate.minutes(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional minutes", () => {
            const rate = Rate.minutes(0.5);

            expect(rate.value).toBe(30000);
        });
    });

    describe("hours()", () => {
        it("should convert hours to milliseconds", () => {
            const rate = Rate.hours(1);

            expect(rate.value).toBe(3600000);
        });

        it("should handle multiple hours", () => {
            const rate = Rate.hours(24);

            expect(rate.value).toBe(86400000);
        });

        it("should handle zero hours", () => {
            const rate = Rate.hours(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional hours", () => {
            const rate = Rate.hours(0.5);

            expect(rate.value).toBe(1800000);
        });
    });

    describe("days()", () => {
        it("should convert days to milliseconds", () => {
            const rate = Rate.days(1);

            expect(rate.value).toBe(86400000);
        });

        it("should handle multiple days", () => {
            const rate = Rate.days(7);

            expect(rate.value).toBe(604800000);
        });

        it("should handle zero days", () => {
            const rate = Rate.days(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional days", () => {
            const rate = Rate.days(0.5);

            expect(rate.value).toBe(43200000);
        });
    });

    describe("weeks()", () => {
        it("should convert weeks to milliseconds", () => {
            const rate = Rate.weeks(1);

            expect(rate.value).toBe(604800000);
        });

        it("should handle multiple weeks", () => {
            const rate = Rate.weeks(4);

            expect(rate.value).toBe(2419200000);
        });

        it("should handle zero weeks", () => {
            const rate = Rate.weeks(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional weeks", () => {
            const rate = Rate.weeks(0.5);

            expect(rate.value).toBe(302400000);
        });
    });

    describe("months()", () => {
        it("should convert months to milliseconds (30 days per month)", () => {
            const rate = Rate.months(1);

            expect(rate.value).toBe(2592000000);
        });

        it("should handle multiple months", () => {
            const rate = Rate.months(12);

            expect(rate.value).toBe(31104000000);
        });

        it("should handle zero months", () => {
            const rate = Rate.months(0);

            expect(rate.value).toBe(0);
        });

        it("should handle fractional months", () => {
            const rate = Rate.months(0.5);

            expect(rate.value).toBe(1296000000);
        });
    });

    describe("conversion equivalences", () => {
        it("should have 1 minute equal to 60 seconds", () => {
            expect(Rate.minutes(1).value).toBe(Rate.seconds(60).value);
        });

        it("should have 1 hour equal to 60 minutes", () => {
            expect(Rate.hours(1).value).toBe(Rate.minutes(60).value);
        });

        it("should have 1 day equal to 24 hours", () => {
            expect(Rate.days(1).value).toBe(Rate.hours(24).value);
        });

        it("should have 1 week equal to 7 days", () => {
            expect(Rate.weeks(1).value).toBe(Rate.days(7).value);
        });

        it("should have 1 month equal to 30 days", () => {
            expect(Rate.months(1).value).toBe(Rate.days(30).value);
        });
    });
});

describe("Trigger", () => {
    describe("onCron()", () => {
        it("should create a cron trigger with expression and default name", () => {
            const trigger = Trigger.onCron("0 0 * * *");

            expect(trigger.name).toBe("Cron Trigger");
            expect(trigger.triggerType).toBe("cron");
            expect(trigger.description).toBe("Trigger that activates on a cron schedule");
            expect(trigger.args).toEqual({ cronExpression: "0 0 * * *" });
        });

        it("should create a cron trigger with custom name", () => {
            const trigger = Trigger.onCron("0 9 * * 1-5", "Weekday Morning Trigger");

            expect(trigger.name).toBe("Weekday Morning Trigger");
            expect(trigger.triggerType).toBe("cron");
            expect(trigger.args.cronExpression).toBe("0 9 * * 1-5");
        });

        it("should handle various cron expressions", () => {
            const everyMinute = Trigger.onCron("* * * * *");
            const everyHour = Trigger.onCron("0 * * * *");
            const midnight = Trigger.onCron("0 0 * * *");
            const weeklyMonday = Trigger.onCron("0 0 * * 1");

            expect(everyMinute.args.cronExpression).toBe("* * * * *");
            expect(everyHour.args.cronExpression).toBe("0 * * * *");
            expect(midnight.args.cronExpression).toBe("0 0 * * *");
            expect(weeklyMonday.args.cronExpression).toBe("0 0 * * 1");
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Trigger.onCron("0 0 * * *");

            expect(trigger.name).toBeDefined();
            expect(trigger.triggerType).toBeDefined();
            expect(trigger.description).toBeDefined();
        });
    });

    describe("onRate()", () => {
        it("should create a rate trigger with Rate and default name", () => {
            const rate = Rate.minutes(5);
            const trigger = Trigger.onRate(rate);

            expect(trigger.name).toBe("Rate Trigger");
            expect(trigger.triggerType).toBe("rate");
            expect(trigger.description).toBe("Trigger that activates on a rate schedule");
            expect(trigger.args).toEqual({ rate: rate });
        });

        it("should create a rate trigger with custom name", () => {
            const rate = Rate.hours(1);
            const trigger = Trigger.onRate(rate, "Hourly Check");

            expect(trigger.name).toBe("Hourly Check");
            expect(trigger.triggerType).toBe("rate");
        });

        it("should work with various rate units", () => {
            const secondsRate = Trigger.onRate(Rate.seconds(30));
            const minutesRate = Trigger.onRate(Rate.minutes(15));
            const hoursRate = Trigger.onRate(Rate.hours(2));
            const daysRate = Trigger.onRate(Rate.days(1));

            expect(secondsRate.args.rate.value).toBe(30000);
            expect(minutesRate.args.rate.value).toBe(900000);
            expect(hoursRate.args.rate.value).toBe(7200000);
            expect(daysRate.args.rate.value).toBe(86400000);
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Trigger.onRate(Rate.minutes(10));

            expect(trigger.name).toBeDefined();
            expect(trigger.triggerType).toBeDefined();
            expect(trigger.description).toBeDefined();
        });
    });

    describe("onEvent()", () => {
        it("should create an event trigger with name", () => {
            const trigger = Trigger.onEvent("user.signup");

            expect(trigger.name).toBe("user.signup");
            expect(trigger.triggerType).toBe("event");
            expect(trigger.description).toBe("Trigger that activates on an event");
            expect(trigger.args).toBeUndefined();
        });

        it("should handle various event names", () => {
            const signup = Trigger.onEvent("user.signup");
            const purchase = Trigger.onEvent("order.completed");
            const custom = Trigger.onEvent("custom-event-name");

            expect(signup.name).toBe("user.signup");
            expect(purchase.name).toBe("order.completed");
            expect(custom.name).toBe("custom-event-name");
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Trigger.onEvent("test.event");

            expect(trigger.name).toBeDefined();
            expect(trigger.triggerType).toBe("event");
            expect(trigger.description).toBeDefined();
        });
    });

    describe("onFileUpload()", () => {
        it("should create a file upload trigger with name", () => {
            const trigger = Trigger.onFileUpload("document_upload");

            expect(trigger.name).toBe("document_upload");
            expect(trigger.triggerType).toBe("file_upload");
            expect(trigger.description).toBe("Trigger that activates on a file upload");
            expect(trigger.args).toBeUndefined();
        });

        it("should handle various upload names", () => {
            const image = Trigger.onFileUpload("image_upload");
            const document = Trigger.onFileUpload("document_upload");
            const data = Trigger.onFileUpload("data_import");

            expect(image.name).toBe("image_upload");
            expect(document.name).toBe("document_upload");
            expect(data.name).toBe("data_import");
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Trigger.onFileUpload("file_trigger");

            expect(trigger.name).toBeDefined();
            expect(trigger.triggerType).toBe("file_upload");
            expect(trigger.description).toBeDefined();
        });
    });

    describe("onSlack()", () => {
        it("should create a slack trigger with name and channel", () => {
            const trigger = Trigger.onSlack("slack_notification", "#general");

            expect(trigger.name).toBe("slack_notification");
            expect(trigger.triggerType).toBe("slack");
            expect(trigger.description).toBe("Trigger that activates on a slack message");
            expect(trigger.args).toEqual({ channel: "#general" });
        });

        it("should handle various channel formats", () => {
            const hashChannel = Trigger.onSlack("trigger1", "#engineering");
            const directChannel = Trigger.onSlack("trigger2", "general");
            const idChannel = Trigger.onSlack("trigger3", "C1234567890");

            expect(hashChannel.args.channel).toBe("#engineering");
            expect(directChannel.args.channel).toBe("general");
            expect(idChannel.args.channel).toBe("C1234567890");
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Trigger.onSlack("slack_trigger", "#support");

            expect(trigger.name).toBeDefined();
            expect(trigger.triggerType).toBe("slack");
            expect(trigger.description).toBeDefined();
            expect(trigger.args).toBeDefined();
        });
    });

    describe("ITrigger interface compliance", () => {
        it("should have all required ITrigger properties", () => {
            const triggers = [
                Trigger.onCron("0 0 * * *"),
                Trigger.onRate(Rate.hours(1)),
                Trigger.onEvent("test.event"),
                Trigger.onFileUpload("upload_trigger"),
                Trigger.onSlack("slack_trigger", "#channel"),
            ];

            triggers.forEach((trigger) => {
                expect(typeof trigger.name).toBe("string");
                expect(typeof trigger.triggerType).toBe("string");
                expect(typeof trigger.description).toBe("string");
            });
        });

        it("should have distinct trigger types", () => {
            const cron = Trigger.onCron("0 0 * * *");
            const rate = Trigger.onRate(Rate.hours(1));
            const event = Trigger.onEvent("test");
            const file = Trigger.onFileUpload("upload");
            const slack = Trigger.onSlack("slack", "#ch");

            const types = new Set([
                cron.triggerType,
                rate.triggerType,
                event.triggerType,
                file.triggerType,
                slack.triggerType,
            ]);

            expect(types.size).toBe(5);
        });
    });

    describe("edge cases", () => {
        it("should handle empty string names", () => {
            const event = Trigger.onEvent("");
            const file = Trigger.onFileUpload("");
            const slack = Trigger.onSlack("", "");

            expect(event.name).toBe("");
            expect(file.name).toBe("");
            expect(slack.name).toBe("");
            expect(slack.args.channel).toBe("");
        });

        it("should handle complex cron expressions", () => {
            const complex = Trigger.onCron("0 30 9 1,15 * *");

            expect(complex.args.cronExpression).toBe("0 30 9 1,15 * *");
        });

        it("should handle very small rates", () => {
            const smallRate = Trigger.onRate(Rate.milliseconds(1));

            expect(smallRate.args.rate.value).toBe(1);
        });

        it("should handle very large rates", () => {
            const largeRate = Trigger.onRate(Rate.months(12));

            expect(largeRate.args.rate.value).toBe(31104000000);
        });
    });
});

