#!/usr/bin/env node
import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const outputDir = process.env.IPPURE_OUTPUT_DIR || path.join("runtime", "ippure-verification");
const allPages = [
  ["home", "https://ippure.com/"],
  ["ip", "https://ippure.com/IP-leak-Detect"],
  ["webrtc", "https://ippure.com/Browser-WebRTC-Leak-Detect"],
  ["dns", "https://ippure.com/DNS-Leak-Detect"],
  ["fingerprint", "https://ippure.com/fingerprint"],
];
const requestedPages = new Set(
  (process.env.IPPURE_PAGES || "")
    .split(",")
    .map((pageName) => pageName.trim())
    .filter(Boolean),
);
const pages = requestedPages.size > 0 ? allPages.filter(([name]) => requestedPages.has(name)) : allPages;
if (pages.length === 0) {
  throw new Error(`No IPPure pages matched IPPURE_PAGES=${process.env.IPPURE_PAGES}`);
}

await mkdir(outputDir, { recursive: true });

const hardenBrowser = process.env.IPPURE_HARDEN_BROWSER !== "0";
const launchArgs = ["--no-sandbox", "--disable-setuid-sandbox"];
if (hardenBrowser) {
  launchArgs.push(
    "--disable-quic",
    "--disable-features=EncryptedClientHello,UseDnsHttpsSvcbAlpn",
    "--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
    "--webrtc-ip-handling-policy=disable_non_proxied_udp",
  );
}
const navigationTimeoutMs = Number.parseInt(process.env.IPPURE_NAV_TIMEOUT_MS || "60000", 10);
const networkIdleTimeoutMs = Number.parseInt(process.env.IPPURE_NETWORK_IDLE_TIMEOUT_MS || "10000", 10);
const settleTimeoutMs = Number.parseInt(process.env.IPPURE_SETTLE_TIMEOUT_MS || "3000", 10);
const proxyServer = process.env.IPPURE_PROXY_SERVER || "";

const browser = await chromium.launch({
  headless: true,
  args: launchArgs,
  proxy: proxyServer ? { server: proxyServer } : undefined,
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
        timeout: navigationTimeoutMs,
      });
      await page.waitForLoadState("networkidle", { timeout: networkIdleTimeoutMs }).catch(() => undefined);
      await page.waitForTimeout(settleTimeoutMs);
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
