import {foo} from "index";

describe("foo", () => {
    it("should return a +1", () => {
        expect(foo(1)).toBe(2);
    });
})