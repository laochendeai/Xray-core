#!/usr/bin/env node
import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const outputDir = process.env.IPPURE_OUTPUT_DIR || path.join("runtime", "ippure-verification");
const pages = [
  ["home", "https://ippure.com/"],
  ["ip", "https://ippure.com/IP-leak-Detect"],
  ["webrtc", "https://ippure.com/Browser-WebRTC-Leak-Detect"],
  ["dns", "https://ippure.com/DNS-Leak-Detect"],
  ["fingerprint", "https://ippure.com/fingerprint"],
];

await mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({
  headless: true,
  args: ["--no-sandbox", "--disable-setuid-sandbox"],
});

const results = [];
try {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
    ignoreHTTPSErrors: true,
  });
  const page = await context.newPage();

  for (const [name, url] of pages) {
    const startedAt = new Date().toISOString();
    const result = { name, url, startedAt, status: "unknown", title: "", error: "" };
    try {
      const response = await page.goto(url, {
        waitUntil: "domcontentloaded",
        timeout: 60000,
      });
      await page.waitForLoadState("networkidle", { timeout: 10000 }).catch(() => undefined);
      await page.waitForTimeout(3000);
      result.status = response ? String(response.status()) : "no-response";
      result.title = await page.title();
      const text = await page.locator("body").innerText({ timeout: 10000 }).catch(() => "");
      await writeFile(path.join(outputDir, `${name}.txt`), text);
      await page.screenshot({
        path: path.join(outputDir, `${name}.png`),
        fullPage: true,
      });
    } catch (error) {
      result.status = "error";
      result.error = error instanceof Error ? error.message : String(error);
      result.title = await page.title().catch(() => "");
      const text = await page.locator("body").innerText({ timeout: 3000 }).catch(() => "");
      if (text) {
        await writeFile(path.join(outputDir, `${name}.txt`), text);
      }
      await page
        .screenshot({
          path: path.join(outputDir, `${name}.png`),
          fullPage: true,
        })
        .catch(() => undefined);
    }
    results.push({ ...result, finishedAt: new Date().toISOString() });
  }
} finally {
  await browser.close();
}

await writeFile(path.join(outputDir, "summary.json"), JSON.stringify(results, null, 2));

const failed = results.filter((result) => result.status === "error");
for (const result of results) {
  console.log(`${result.name}: ${result.status} ${result.title || ""}`.trim());
  if (result.error) console.log(`  ${result.error}`);
}
if (failed.length > 0) {
  process.exitCode = 1;
}
