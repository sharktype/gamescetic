import { describe, expect, it } from "vitest";

import { add } from "./index.ts";

describe("add", () => {
	it("adds two positive numbers", () => {
		expect(add(2, 3)).toBe(5);
	});

	it("adds two negative numbers", () => {
		expect(add(-2, -3)).toBe(-5);
	});

	it("adds zeros", () => {
		expect(add(0, 0)).toBe(0);
	});
});
