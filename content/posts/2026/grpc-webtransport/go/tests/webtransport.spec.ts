import { expect, type Page, test } from "@playwright/test";

async function renderedLog(page: Page) {
	return page.locator("p").evaluateAll((paragraphs) =>
		paragraphs.map((paragraph) => paragraph.textContent ?? "").join("\n"),
	);
}

test("browser client establishes a WebTransport session", async ({ page }) => {
	const consoleMessages: string[] = [];
	page.on("console", (message) => {
		consoleMessages.push(`${message.type()}: ${message.text()}`);
	});
	page.on("pageerror", (error) => {
		consoleMessages.push(`pageerror: ${error.message}`);
	});

	await page.goto("/");

	await expect(page.getByText("Opening a WebTransport session")).toBeVisible();

	await page.waitForFunction(
		() =>
			Array.from(document.querySelectorAll("p")).some((paragraph) =>
				/Session ready|Session failed|Session closed with error/.test(
					paragraph.textContent ?? "",
				),
			),
		undefined,
		{ timeout: 15_000 },
	);

	const pageLog = await renderedLog(page);
	expect(
		pageLog,
		["page output:", pageLog, "console output:", ...consoleMessages].join("\n"),
	).toContain("Session ready");
	await expect(page.getByText("Received: Hello, WebTransport!")).toBeVisible();

	expect(consoleMessages, consoleMessages.join("\n")).not.toContain(
		"WebTransport connection rejected",
	);
});
