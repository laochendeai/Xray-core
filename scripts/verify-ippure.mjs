#!/usr/bin/env node
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
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

function seededRandom(seedText) {
  let state = createHash("sha256").update(seedText).digest().readUInt32LE(0);
  return () => {
    state |= 0;
    state = (state + 0x6d2b79f5) | 0;
    let next = Math.imul(state ^ (state >>> 15), 1 | state);
    next = (next + Math.imul(next ^ (next >>> 7), 61 | next)) ^ next;
    return ((next ^ (next >>> 14)) >>> 0) / 4294967296;
  };
}

function pick(rng, values) {
  return values[Math.floor(rng() * values.length) % values.length];
}

function detectChromiumVersion() {
  const explicit = process.env.IPPURE_CHROME_VERSION?.trim();
  if (explicit) return explicit;

  try {
    const output = execFileSync(chromium.executablePath(), ["--version"], {
      encoding: "utf8",
      timeout: 3000,
    });
    const match = output.match(/(\d+\.\d+\.\d+\.\d+)/);
    if (match) return match[1];
  } catch {
    // Keep verification usable even if the local browser cannot report itself.
  }

  return "145.0.0.0";
}

function buildFingerprintProfile() {
  const seed = process.env.IPPURE_FINGERPRINT_SEED || randomBytes(16).toString("hex");
  const rng = seededRandom(seed);
  const chromeVersion = detectChromiumVersion();
  const chromeMajorVersion = chromeVersion.split(".")[0] || "145";
  const localeProfile = pick(rng, [
    { locale: "en-US", timezoneId: "America/New_York" },
    { locale: "en-US", timezoneId: "America/Los_Angeles" },
    { locale: "en-GB", timezoneId: "Europe/London" },
    { locale: "de-DE", timezoneId: "Europe/Berlin" },
    { locale: "fr-FR", timezoneId: "Europe/Paris" },
    { locale: "ja-JP", timezoneId: "Asia/Tokyo" },
  ]);
  const platformProfile = pick(rng, [
    {
      platform: "Win32",
      userAgentPlatform: "Windows",
      platformVersion: "10.0.0",
      userAgent: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${chromeVersion} Safari/537.36`,
      webglChoices: [
        { vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Intel UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0)" },
        { vendor: "Google Inc. (NVIDIA)", renderer: "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 Direct3D11 vs_5_0 ps_5_0)" },
        { vendor: "Google Inc. (AMD)", renderer: "ANGLE (AMD, AMD Radeon Pro 560X Direct3D11 vs_5_0 ps_5_0)" },
      ],
    },
    {
      platform: "MacIntel",
      userAgentPlatform: "macOS",
      platformVersion: "14.0.0",
      userAgent: `Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${chromeVersion} Safari/537.36`,
      webglChoices: [
        { vendor: "Intel Inc.", renderer: "Intel Iris OpenGL Engine" },
        { vendor: "Apple Inc.", renderer: "Apple M2" },
        { vendor: "Apple Inc.", renderer: "Apple M3" },
      ],
    },
    {
      platform: "Linux x86_64",
      userAgentPlatform: "Linux",
      platformVersion: "6.6.0",
      userAgent: `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${chromeVersion} Safari/537.36`,
      webglChoices: [
        { vendor: "Google Inc. (Intel)", renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620)" },
        { vendor: "Google Inc. (AMD)", renderer: "ANGLE (AMD, AMD Radeon Graphics (radeonsi))" },
      ],
    },
  ]);
  const viewport = pick(rng, [
    { width: 1366, height: 768 },
    { width: 1440, height: 900 },
    { width: 1536, height: 864 },
    { width: 1600, height: 900 },
    { width: 1920, height: 1080 },
  ]);
  const hardwareConcurrency = pick(rng, [4, 6, 8, 10, 12]);
  const deviceMemory = pick(rng, [4, 8, 16]);
  const deviceScaleFactor = pick(rng, [1, 1.25, 1.5, 2]);
  const webgl = pick(rng, platformProfile.webglChoices);
  const brands = [
    { brand: "Chromium", version: chromeMajorVersion },
    { brand: "Google Chrome", version: chromeMajorVersion },
    { brand: "Not.A/Brand", version: "99" },
  ];

  return {
    seed,
    viewport,
    locale: localeProfile.locale,
    timezoneId: localeProfile.timezoneId,
    platform: platformProfile.platform,
    userAgentPlatform: platformProfile.userAgentPlatform,
    platformVersion: platformProfile.platformVersion,
    hardwareConcurrency,
    deviceMemory,
    deviceScaleFactor,
    userAgent: platformProfile.userAgent,
    chromeVersion,
    chromeMajorVersion,
    brands,
    webgl,
    canvasNoise: Math.floor(rng() * 255),
    audioNoise: Number((rng() / 100000).toFixed(8)),
  };
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
const randomFingerprint = process.env.IPPURE_RANDOM_FINGERPRINT !== "0";
const disableWebRTC = process.env.IPPURE_DISABLE_WEBRTC !== "0";
const fingerprintProfile = randomFingerprint ? buildFingerprintProfile() : null;
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
const keepOpen = parseBooleanEnv(process.env.IPPURE_KEEP_OPEN, false);
const explicitProxyServer = process.env.IPPURE_PROXY_SERVER || "";
const detectedProxy = explicitProxyServer ? { server: explicitProxyServer, source: "env" } : await proxyFromConfig();
const proxyServer = detectedProxy.server;

const browser = await chromium.launch({
  headless,
  args: launchArgs,
  proxy: proxyServer ? { server: proxyServer } : undefined,
});

const results = [];
let browserClosed = false;
async function closeBrowser() {
  if (browserClosed) return;
  browserClosed = true;
  await browser.close();
}

try {
  const context = await browser.newContext({
    viewport: fingerprintProfile?.viewport || { width: 1440, height: 1100 },
    userAgent: fingerprintProfile?.userAgent,
    locale: fingerprintProfile?.locale,
    timezoneId: fingerprintProfile?.timezoneId,
    deviceScaleFactor: fingerprintProfile?.deviceScaleFactor,
    ignoreHTTPSErrors: true,
    serviceWorkers: "block",
  });
  if (fingerprintProfile || disableWebRTC) {
    await context.addInitScript(({ profile, disableWebRTCApi }) => {
      const defineGetter = (target, property, value) => {
        try {
          Object.defineProperty(target, property, {
            configurable: true,
            get: () => value,
          });
        } catch {
          // Ignore non-configurable browser properties.
        }
      };

      if (disableWebRTCApi) {
        defineGetter(window, "RTCPeerConnection", undefined);
        defineGetter(window, "webkitRTCPeerConnection", undefined);
        defineGetter(window, "RTCDataChannel", undefined);
      }

      if (!profile) return;

      defineGetter(Navigator.prototype, "userAgent", profile.userAgent);
      defineGetter(Navigator.prototype, "platform", profile.platform);
      defineGetter(Navigator.prototype, "languages", [profile.locale, profile.locale.split("-")[0]]);
      defineGetter(Navigator.prototype, "language", profile.locale);
      defineGetter(Navigator.prototype, "hardwareConcurrency", profile.hardwareConcurrency);
      defineGetter(Navigator.prototype, "deviceMemory", profile.deviceMemory);
      defineGetter(Navigator.prototype, "webdriver", undefined);
      defineGetter(Navigator.prototype, "userAgentData", {
        brands: profile.brands,
        mobile: false,
        platform: profile.userAgentPlatform,
        getHighEntropyValues: async (hints = []) => {
          const values = {
            brands: profile.brands,
            mobile: false,
            platform: profile.userAgentPlatform,
            platformVersion: profile.platformVersion,
            architecture: "x86",
            bitness: "64",
            model: "",
            uaFullVersion: profile.chromeVersion,
            fullVersionList: profile.brands.map((brand) => ({
              brand: brand.brand,
              version: brand.brand === "Not.A/Brand" ? "99.0.0.0" : profile.chromeVersion,
            })),
          };
          return Object.fromEntries(hints.filter((hint) => hint in values).map((hint) => [hint, values[hint]]));
        },
      });
      defineGetter(screen, "width", profile.viewport.width);
      defineGetter(screen, "height", profile.viewport.height);
      defineGetter(screen, "availWidth", profile.viewport.width);
      defineGetter(screen, "availHeight", profile.viewport.height - 40);
      defineGetter(screen, "colorDepth", 24);
      defineGetter(screen, "pixelDepth", 24);

      const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
      HTMLCanvasElement.prototype.toDataURL = function (...args) {
        const ctx = this.getContext("2d");
        if (ctx) {
          ctx.save();
          ctx.globalAlpha = 0.01;
          ctx.fillStyle = `rgb(${profile.canvasNoise}, ${255 - profile.canvasNoise}, 127)`;
          ctx.fillRect(0, 0, 1, 1);
          ctx.restore();
        }
        return originalToDataURL.apply(this, args);
      };

      const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;
      CanvasRenderingContext2D.prototype.getImageData = function (...args) {
        const imageData = originalGetImageData.apply(this, args);
        for (let index = 0; index < imageData.data.length; index += 64) {
          imageData.data[index] = (imageData.data[index] + profile.canvasNoise) % 255;
        }
        return imageData;
      };

      const patchWebGL = (prototype) => {
        if (!prototype || !prototype.getParameter) return;
        const originalGetParameter = prototype.getParameter;
        prototype.getParameter = function (parameter) {
          const debugInfo = this.getExtension("WEBGL_debug_renderer_info");
          if (debugInfo && parameter === debugInfo.UNMASKED_VENDOR_WEBGL) return profile.webgl.vendor;
          if (debugInfo && parameter === debugInfo.UNMASKED_RENDERER_WEBGL) return profile.webgl.renderer;
          return originalGetParameter.call(this, parameter);
        };
      };
      patchWebGL(window.WebGLRenderingContext?.prototype);
      patchWebGL(window.WebGL2RenderingContext?.prototype);

      const OriginalAudioContext = window.AudioContext || window.webkitAudioContext;
      if (OriginalAudioContext?.prototype?.createAnalyser) {
        const originalCreateAnalyser = OriginalAudioContext.prototype.createAnalyser;
        OriginalAudioContext.prototype.createAnalyser = function (...args) {
          const analyser = originalCreateAnalyser.apply(this, args);
          const originalGetFloatFrequencyData = analyser.getFloatFrequencyData;
          analyser.getFloatFrequencyData = function (array) {
            originalGetFloatFrequencyData.call(this, array);
            for (let index = 0; index < array.length; index += 16) {
              array[index] += profile.audioNoise;
            }
          };
          return analyser;
        };
      }
    }, { profile: fingerprintProfile, disableWebRTCApi: disableWebRTC });
  }
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
} catch (error) {
  await closeBrowser();
  throw error;
}

const summary = {
  generatedAt: new Date().toISOString(),
  hardenBrowser,
  randomFingerprint,
  disableWebRTC,
  headless,
  keepOpen: keepOpen && !headless,
  proxyServer: proxyServer ? proxyServer.replace(/\/\/.*@/, "//***@") : "",
  proxySource: detectedProxy.source,
  fingerprintProfile: fingerprintProfile
    ? {
        seed: fingerprintProfile.seed,
        viewport: fingerprintProfile.viewport,
        locale: fingerprintProfile.locale,
        timezoneId: fingerprintProfile.timezoneId,
        platform: fingerprintProfile.platform,
        userAgentPlatform: fingerprintProfile.userAgentPlatform,
        hardwareConcurrency: fingerprintProfile.hardwareConcurrency,
        deviceMemory: fingerprintProfile.deviceMemory,
        deviceScaleFactor: fingerprintProfile.deviceScaleFactor,
        chromeVersion: fingerprintProfile.chromeVersion,
        webgl: fingerprintProfile.webgl,
      }
    : null,
  pages: results,
};

await writeFile(path.join(outputDir, "summary.json"), JSON.stringify(summary, null, 2));

const failed = results.filter((result) => result.status === "error");
console.log(`proxy: ${proxyServer || "none"} (${detectedProxy.source})`);
console.log(`hardened: ${hardenBrowser ? "yes" : "no"}, randomFingerprint: ${randomFingerprint ? "yes" : "no"}, webRTCDisabled: ${disableWebRTC ? "yes" : "no"}, headless: ${headless ? "yes" : "no"}, keepOpen: ${keepOpen && !headless ? "yes" : "no"}`);
for (const result of results) {
  console.log(`${result.name}: ${result.status} ${result.title || ""}`.trim());
  if (result.error) console.log(`  ${result.error}`);
}
if (failed.length > 0) {
  process.exitCode = 1;
}

if (keepOpen && !headless) {
  console.log("IPPURE_KEEP_OPEN=1: the hardened browser remains open. Press Ctrl+C to close it.");
  await new Promise((resolve) => {
    const finish = () => resolve();
    browser.once("disconnected", finish);
    process.once("SIGINT", finish);
  });
  await closeBrowser();
} else {
  await closeBrowser();
}
