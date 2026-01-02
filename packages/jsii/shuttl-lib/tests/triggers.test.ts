import { Rate, ITrigger } from "../src/trigger";

describe.skip("Rate", () => {
    describe("milliseconds()", () => {
        it("should create a Rate with the exact millisecond value", () => {
            const rate = Rate.milliseconds(500);

            expect(rate.triggerConfig.ms_rate).toBe(500);
        });

        it("should handle zero milliseconds", () => {
            const rate = Rate.milliseconds(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle large millisecond values", () => {
            const rate = Rate.milliseconds(1000000);

            expect(rate.triggerConfig.ms_rate).toBe(1000000);
        });
    });

    describe("seconds()", () => {
        it("should convert seconds to milliseconds", () => {
            const rate = Rate.seconds(1);

            expect(rate.triggerConfig.ms_rate).toBe(1000);
        });

        it("should handle multiple seconds", () => {
            const rate = Rate.seconds(30);

            expect(rate.triggerConfig.ms_rate).toBe(30000);
        });

        it("should handle zero seconds", () => {
            const rate = Rate.seconds(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional seconds", () => {
            const rate = Rate.seconds(1.5);

            expect(rate.triggerConfig.ms_rate).toBe(1500);
        });
    });

    describe("minutes()", () => {
        it("should convert minutes to milliseconds", () => {
            const rate = Rate.minutes(1);

            expect(rate.triggerConfig.ms_rate).toBe(60000);
        });

        it("should handle multiple minutes", () => {
            const rate = Rate.minutes(5);

            expect(rate.triggerConfig.ms_rate).toBe(300000);
        });

        it("should handle zero minutes", () => {
            const rate = Rate.minutes(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional minutes", () => {
            const rate = Rate.minutes(0.5);

            expect(rate.triggerConfig.ms_rate).toBe(30000);
        });
    });

    describe("hours()", () => {
        it("should convert hours to milliseconds", () => {
            const rate = Rate.hours(1);

            expect(rate.triggerConfig.ms_rate).toBe(3600000);
        });

        it("should handle multiple hours", () => {
            const rate = Rate.hours(24);

            expect(rate.triggerConfig.ms_rate).toBe(86400000);
        });

        it("should handle zero hours", () => {
            const rate = Rate.hours(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional hours", () => {
            const rate = Rate.hours(0.5);

            expect(rate.triggerConfig.ms_rate).toBe(1800000);
        });
    });

    describe("days()", () => {
        it("should convert days to milliseconds", () => {
            const rate = Rate.days(1);

            expect(rate.triggerConfig.ms_rate).toBe(86400000);
        });

        it("should handle multiple days", () => {
            const rate = Rate.days(7);

            expect(rate.triggerConfig.ms_rate).toBe(604800000);
        });

        it("should handle zero days", () => {
            const rate = Rate.days(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional days", () => {
            const rate = Rate.days(0.5);

            expect(rate.triggerConfig.ms_rate).toBe(43200000);
        });
    });

    describe("weeks()", () => {
        it("should convert weeks to milliseconds", () => {
            const rate = Rate.weeks(1);

            expect(rate.triggerConfig.ms_rate).toBe(604800000);
        });

        it("should handle multiple weeks", () => {
            const rate = Rate.weeks(4);

            expect(rate.triggerConfig.ms_rate).toBe(2419200000);
        });

        it("should handle zero weeks", () => {
            const rate = Rate.weeks(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional weeks", () => {
            const rate = Rate.weeks(0.5);

            expect(rate.triggerConfig.ms_rate).toBe(302400000);
        });
    });

    describe("months()", () => {
        it("should convert months to milliseconds (30 days per month)", () => {
            const rate = Rate.months(1);

            expect(rate.triggerConfig.ms_rate).toBe(2592000000);
        });

        it("should handle multiple months", () => {
            const rate = Rate.months(12);

            expect(rate.triggerConfig.ms_rate).toBe(31104000000);
        });

        it("should handle zero months", () => {
            const rate = Rate.months(0);

            expect(rate.triggerConfig.ms_rate).toBe(0);
        });

        it("should handle fractional months", () => {
            const rate = Rate.months(0.5);

            expect(rate.triggerConfig.ms_rate).toBe(1296000000);
        });
    });

    describe("conversion equivalences", () => {
        it("should have 1 minute equal to 60 seconds", () => {
            expect(Rate.minutes(1).triggerConfig.ms_rate).toBe(Rate.seconds(60).triggerConfig.ms_rate);
        });

        it("should have 1 hour equal to 60 minutes", () => {
            expect(Rate.hours(1).triggerConfig.ms_rate).toBe(Rate.minutes(60).triggerConfig.ms_rate);
        });

        it("should have 1 day equal to 24 hours", () => {
            expect(Rate.days(1).triggerConfig.ms_rate).toBe(Rate.hours(24).triggerConfig.ms_rate);
        });

        it("should have 1 week equal to 7 days", () => {
            expect(Rate.weeks(1).triggerConfig.ms_rate).toBe(Rate.days(7).triggerConfig.ms_rate);
        });

        it("should have 1 month equal to 30 days", () => {
            expect(Rate.months(1).triggerConfig.ms_rate).toBe(Rate.days(30).triggerConfig.ms_rate);
        });
    });
});

describe.skip("Trigger", () => {
    describe("onCron()", () => {
        it("should create a cron trigger with expression and default name", () => {
            const trigger = Rate.cron("0 0 * * *");

            expect(trigger.triggerType).toBe("cron");
            expect(trigger.triggerConfig.cronExpression).toEqual("0 0 * * *");
        });

        it("should create a cron trigger with custom name", () => {
            const trigger = Rate.cron("0 9 * * 1-5", "Weekday Morning Trigger");

            expect(trigger.triggerConfig.cronExpression).toBe("0 9 * * 1-5");
            expect(trigger.triggerConfig.timezone).toBe("Weekday Morning Trigger");
        });

        it("should handle various cron expressions", () => {
            const everyMinute = Rate.cron("* * * * *");
            const everyHour = Rate.cron("0 * * * *");
            const midnight = Rate.cron("0 0 * * *");
            const weeklyMonday = Rate.cron("0 0 * * 1");

            expect(everyMinute.triggerConfig.cronExpression).toBe("* * * * *");
            expect(everyHour.triggerConfig.cronExpression).toBe("0 * * * *");
            expect(midnight.triggerConfig.cronExpression).toBe("0 0 * * *");
            expect(weeklyMonday.triggerConfig.cronExpression).toBe("0 0 * * 1");
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Rate.cron("0 0 * * *");

            expect(trigger.triggerType).toBeDefined();
            expect(trigger.triggerConfig.cronExpression).toBeDefined();
        });
    });

    describe("onRate()", () => {
        it("should create a rate trigger with Rate and default name", () => {
            const rate = Rate.minutes(5);
            const trigger = rate.withOnTrigger({ onTrigger: () => Promise.resolve([]) });

            expect(trigger.triggerType).toBe("rate");
            expect(trigger.triggerConfig.ms_rate).toBe(300000);
        });

        it("should create a rate trigger with custom name", () => {
            const rate = Rate.hours(1);
            const trigger = rate.withOnTrigger({ onTrigger: () => Promise.resolve([]) });

            expect(trigger.triggerType).toBe("rate");
            expect(trigger.triggerConfig.ms_rate).toBe(3600000);
        });

        it("should work with various rate units", () => {
            const secondsRate = Rate.seconds(30).withOnTrigger({ onTrigger: () => Promise.resolve([]) });
            const minutesRate = Rate.minutes(15).withOnTrigger({ onTrigger: () => Promise.resolve([]) });
            const hoursRate = Rate.hours(2).withOnTrigger({ onTrigger: () => Promise.resolve([]) });
            const daysRate = Rate.days(1).withOnTrigger({ onTrigger: () => Promise.resolve([]) });

            expect(secondsRate.triggerConfig.ms_rate).toBe(30000);
            expect(minutesRate.triggerConfig.ms_rate).toBe(900000);
            expect(hoursRate.triggerConfig.ms_rate).toBe(7200000);
            expect(daysRate.triggerConfig.ms_rate).toBe(86400000);
        });

        it("should implement ITrigger interface", () => {
            const trigger: ITrigger = Rate.minutes(10).withOnTrigger({ onTrigger: () => Promise.resolve([]) });

            expect(trigger.triggerType).toBeDefined();
            expect(trigger.triggerConfig.ms_rate).toBeDefined();
        });
    });

});


