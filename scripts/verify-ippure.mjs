#!/usr/bin/env node
import { chromium } from "playwright";
import net from "node:net";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
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

function parseBooleanEnv(value, defaultValue) {
  if (value === undefined || value === "") return defaultValue;
  return !["0", "false", "no", "off"].includes(value.toLowerCase());
}

async function tcpReachable(host, port, timeoutMs = 500) {
  return await new Promise((resolve) => {
    const socket = net.connect({ host, port });
    const timer = setTimeout(() => {
      socket.destroy();
      resolve(false);
    }, timeoutMs);
    socket.once("connect", () => {
      clearTimeout(timer);
      socket.end();
      resolve(true);
    });
    socket.once("error", () => {
      clearTimeout(timer);
      resolve(false);
    });
  });
}

function normalizeHostForConnect(host) {
  return host.replace(/^\[|\]$/g, "");
}

function formatProxyHost(host) {
  const normalized = normalizeHostForConnect(host);
  return normalized.includes(":") ? `[${normalized}]` : normalized;
}

async function proxyFromConfig() {
  const candidateConfigPaths = [];
  if (process.env.IPPURE_CONFIG) {
    candidateConfigPaths.push({ path: process.env.IPPURE_CONFIG, source: "env-config" });
  }
  candidateConfigPaths.push({ path: path.join(process.cwd(), "dev-config.current-nodes.json"), source: "cwd-config" });
  for (const configPath of await discoverRunningConfigPaths()) {
    candidateConfigPaths.push({ path: configPath, source: "running-process" });
  }

  const seen = new Set();
  for (const candidate of candidateConfigPaths) {
    if (!candidate.path || seen.has(candidate.path)) continue;
    seen.add(candidate.path);
    const resolved = await proxyFromOneConfig(candidate.path, candidate.source);
    if (resolved.server) return resolved;
  }

  return { server: "", source: "none" };
}

async function discoverRunningConfigPaths() {
  let entries = [];
  try {
    entries = await readdir("/proc", { withFileTypes: true });
  } catch {
    return [];
  }

  const configPaths = [];
  for (const entry of entries) {
    if (!entry.isDirectory() || !/^\d+$/.test(entry.name)) continue;
    try {
      const cmdline = await readFile(path.join("/proc", entry.name, "cmdline"), "utf8");
      if (!cmdline) continue;
      const args = cmdline.split("\u0000").filter(Boolean);
      const runIndex = args.indexOf("run");
      const configFlagIndex = args.indexOf("-c");
      if (runIndex < 0 || configFlagIndex < 0 || configFlagIndex + 1 >= args.length) continue;
      if (!/xray/i.test(path.basename(args[0] || ""))) continue;
      configPaths.push(args[configFlagIndex + 1]);
    } catch {
      continue;
    }
  }
  return configPaths;
}

async function proxyFromOneConfig(configPath, sourceLabel) {
  let raw = "";
  try {
    raw = await readFile(configPath, "utf8");
  } catch {
    return { server: "", source: `${sourceLabel}-missing` };
  }

  let config;
  try {
    config = JSON.parse(raw);
  } catch {
    return { server: "", source: `${sourceLabel}-invalid` };
  }

  const inbound = (config.inbounds || []).find((item) => {
    const protocol = String(item?.protocol || "").toLowerCase();
    const listen = String(item?.listen || "127.0.0.1");
    const port = Number(item?.port);
    return protocol === "socks" && Number.isInteger(port) && port > 0 && /^(127\.0\.0\.1|localhost|\[::1\]|::1)$/.test(listen);
  });
  if (!inbound) return { server: "", source: `${sourceLabel}-no-socks` };

  const host = inbound.listen || "127.0.0.1";
  const port = Number(inbound.port);
  if (!(await tcpReachable(normalizeHostForConnect(host), port))) {
    return { server: "", source: `${sourceLabel}-unreachable` };
  }

  return { server: `socks5://${formatProxyHost(host)}:${port}`, source: `${sourceLabel}:${path.basename(configPath)}` };
}

const hardenBrowser = process.env.IPPURE_HARDEN_BROWSER !== "0";
const launchArgs = ["--no-sandbox", "--disable-setuid-sandbox"];
if (hardenBrowser) {
  launchArgs.push(
    "--disable-background-networking",
    "--disable-quic",
    "--disable-sync",
    "--disable-features=AsyncDns,DnsOverHttps,EncryptedClientHello,UseDnsHttpsSvcbAlpn",
    "--disable-ipv6",
    "--no-default-browser-check",
    "--no-first-run",
    "--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
    "--webrtc-ip-handling-policy=disable_non_proxied_udp",
  );
}
const navigationTimeoutMs = Number.parseInt(process.env.IPPURE_NAV_TIMEOUT_MS || "60000", 10);
const networkIdleTimeoutMs = Number.parseInt(process.env.IPPURE_NETWORK_IDLE_TIMEOUT_MS || "10000", 10);
const settleTimeoutMs = Number.parseInt(process.env.IPPURE_SETTLE_TIMEOUT_MS || "3000", 10);
const headless = parseBooleanEnv(process.env.IPPURE_HEADLESS, true);
const explicitProxyServer = process.env.IPPURE_PROXY_SERVER || "";
const detectedProxy = explicitProxyServer ? { server: explicitProxyServer, source: "env" } : await proxyFromConfig();
const proxyServer = detectedProxy.server;

const browser = await chromium.launch({
  headless,
  args: launchArgs,
  proxy: proxyServer ? { server: proxyServer } : undefined,
});

const results = [];
try {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
    ignoreHTTPSErrors: true,
    serviceWorkers: "block",
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

const summary = {
  generatedAt: new Date().toISOString(),
  hardenBrowser,
  headless,
  proxyServer: proxyServer ? proxyServer.replace(/\/\/.*@/, "//***@") : "",
  proxySource: detectedProxy.source,
  pages: results,
};

await writeFile(path.join(outputDir, "summary.json"), JSON.stringify(summary, null, 2));

const failed = results.filter((result) => result.status === "error");
console.log(`proxy: ${proxyServer || "none"} (${detectedProxy.source})`);
console.log(`hardened: ${hardenBrowser ? "yes" : "no"}, headless: ${headless ? "yes" : "no"}`);
for (const result of results) {
  console.log(`${result.name}: ${result.status} ${result.title || ""}`.trim());
  if (result.error) console.log(`  ${result.error}`);
}
if (failed.length > 0) {
  process.exitCode = 1;
}
