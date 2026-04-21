import type {
  PrivacyDiagnosticsContextResponse,
  PrivacyWebRTCCandidate,
} from "@/api/types";

export type PrivacyRiskLevel = "unknown" | "low" | "warning" | "high";

export interface PrivacyWebRTCRisk {
  leakRisk: PrivacyRiskLevel;
  exposedPrivateAddress: boolean;
  exposedPublicAddress: boolean;
}

export interface PrivacyIpExposureResult {
  leakRisk: PrivacyRiskLevel;
  browserIp: string;
  directIp: string;
  proxyIp: string;
  browserMatchesDirect: boolean;
  browserMatchesProxy: boolean;
  tunRunning: boolean;
}

export interface PrivacyRuntimeDnsRisk {
  leakRisk: PrivacyRiskLevel;
  hasRemoteDnsRoute: boolean;
  hasDirectDnsRoute: boolean;
  hasRemoteResolvers: boolean;
  tunRunning: boolean;
}

export interface FingerprintSnapshot {
  userAgent: string;
  languages: string[];
  timezone: string;
  screen: {
    width: number;
    height: number;
    colorDepth: number;
    devicePixelRatio: number;
  };
  hardwareConcurrency: number | null;
  deviceMemory: number | null;
  cookieEnabled: boolean;
  doNotTrack: string | null;
  canvasHash: string;
  webglVendor: string;
  webglRenderer: string;
  audioSampleRate: number | null;
}

export interface FingerprintRisk {
  leakRisk: PrivacyRiskLevel;
  highEntropySurfaceCount: number;
}

function normalizeIp(value: string | undefined): string {
  return (value || "").trim().toLowerCase();
}

export function isPrivateAddress(address: string): boolean {
  const value = normalizeIp(address);
  if (!value || value.endsWith(".local")) return false;
  return (
    /^(10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.|127\.|169\.254\.)/.test(value) ||
    /^(fc|fd|fe80:|::1)/.test(value)
  );
}

export function parseIceCandidate(candidate: string): PrivacyWebRTCCandidate {
  const parts = candidate.trim().split(/\s+/);
  const typIndex = parts.findIndex((part) => part === "typ");
  const protocol = (parts[2] || "unknown").toLowerCase();
  const address = parts[4] || "";
  const rawPort = Number(parts[5]);
  return {
    candidate,
    type: typIndex >= 0 ? parts[typIndex + 1] || "unknown" : "unknown",
    protocol: protocol === "udp" || protocol === "tcp" ? protocol : "unknown",
    address,
    port: Number.isFinite(rawPort) ? rawPort : null,
    isPrivateAddress: isPrivateAddress(address),
  };
}

export function classifyWebRTCRisk(candidates: PrivacyWebRTCCandidate[]): PrivacyWebRTCRisk {
  const exposedPrivateAddress = candidates.some((candidate) => candidate.isPrivateAddress);
  const exposedPublicAddress = candidates.some(
    (candidate) => candidate.type === "srflx" && !candidate.isPrivateAddress,
  );

  let leakRisk: PrivacyRiskLevel = "unknown";
  if (exposedPrivateAddress && exposedPublicAddress) {
    leakRisk = "high";
  } else if (exposedPrivateAddress || exposedPublicAddress) {
    leakRisk = "warning";
  } else if (candidates.length > 0) {
    leakRisk = "low";
  }

  return {
    leakRisk,
    exposedPrivateAddress,
    exposedPublicAddress,
  };
}

export function classifyIpExposure(
  browserIp: string,
  context: PrivacyDiagnosticsContextResponse | null,
): PrivacyIpExposureResult {
  const normalizedBrowserIp = normalizeIp(browserIp);
  const directIp = normalizeIp(context?.tunStatus?.directEgress?.ip);
  const proxyIp = normalizeIp(context?.tunStatus?.proxyEgress?.ip);
  const tunRunning = !!context?.tunStatus?.running;
  const browserMatchesDirect = !!normalizedBrowserIp && !!directIp && normalizedBrowserIp === directIp;
  const browserMatchesProxy = !!normalizedBrowserIp && !!proxyIp && normalizedBrowserIp === proxyIp;

  let leakRisk: PrivacyRiskLevel = "unknown";
  if (browserMatchesDirect && tunRunning) {
    leakRisk = "high";
  } else if (browserMatchesProxy) {
    leakRisk = "low";
  } else if (normalizedBrowserIp && tunRunning && proxyIp) {
    leakRisk = "warning";
  } else if (normalizedBrowserIp && !tunRunning) {
    leakRisk = "warning";
  }

  return {
    leakRisk,
    browserIp: normalizedBrowserIp,
    directIp,
    proxyIp,
    browserMatchesDirect,
    browserMatchesProxy,
    tunRunning,
  };
}

export function classifyRuntimeDnsRisk(
  context: PrivacyDiagnosticsContextResponse | null,
): PrivacyRuntimeDnsRisk {
  const diagnostics = context?.tunStatus?.routingDiagnostics || [];
  const remoteDns = context?.tunSettings?.remoteDns || context?.tunStatus?.remoteDns || [];
  const tunRunning = !!context?.tunStatus?.running;
  const hasRemoteResolvers = remoteDns.length > 0;
  const hasRemoteDnsRoute = diagnostics.some(
    (diagnostic) =>
      diagnostic.dnsPath === "dns-remote" &&
      diagnostic.route.toLowerCase().includes("proxy"),
  );
  const hasDirectDnsRoute = diagnostics.some(
    (diagnostic) =>
      diagnostic.dnsPath !== "dns-cn" &&
      diagnostic.dnsPath !== "dns-direct-local" &&
      diagnostic.route.toLowerCase() === "direct",
  );

  let leakRisk: PrivacyRiskLevel = "unknown";
  if (!tunRunning) {
    leakRisk = "unknown";
  } else if (hasRemoteDnsRoute && hasRemoteResolvers && !hasDirectDnsRoute) {
    leakRisk = "low";
  } else {
    leakRisk = "warning";
  }

  return {
    leakRisk,
    hasRemoteDnsRoute,
    hasDirectDnsRoute,
    hasRemoteResolvers,
    tunRunning,
  };
}

export function classifyFingerprintRisk(snapshot: FingerprintSnapshot | null): FingerprintRisk {
  if (!snapshot) {
    return { leakRisk: "unknown", highEntropySurfaceCount: 0 };
  }

  let highEntropySurfaceCount = 0;
  if (snapshot.userAgent) highEntropySurfaceCount += 1;
  if (snapshot.languages.length > 0) highEntropySurfaceCount += 1;
  if (snapshot.timezone) highEntropySurfaceCount += 1;
  if (snapshot.canvasHash) highEntropySurfaceCount += 1;
  if (snapshot.webglVendor || snapshot.webglRenderer) highEntropySurfaceCount += 1;
  if (snapshot.audioSampleRate) highEntropySurfaceCount += 1;
  if ((snapshot.hardwareConcurrency || 0) >= 8 || (snapshot.deviceMemory || 0) >= 8) {
    highEntropySurfaceCount += 1;
  }
  if (
    snapshot.screen.width >= 1920 ||
    snapshot.screen.height >= 1080 ||
    snapshot.screen.colorDepth > 24 ||
    snapshot.screen.devicePixelRatio > 1
  ) {
    highEntropySurfaceCount += 1;
  }

  let leakRisk: PrivacyRiskLevel = "low";
  if (highEntropySurfaceCount >= 7) {
    leakRisk = "high";
  } else if (highEntropySurfaceCount >= 4) {
    leakRisk = "warning";
  }

  return {
    leakRisk,
    highEntropySurfaceCount,
  };
}

function simpleHash(value: string): string {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash << 5) - hash + value.charCodeAt(index);
    hash |= 0;
  }
  return Math.abs(hash).toString(16);
}

function canvasFingerprint(): string {
  const canvas = document.createElement("canvas");
  canvas.width = 240;
  canvas.height = 60;
  const ctx = canvas.getContext("2d");
  if (!ctx) return "";
  ctx.textBaseline = "top";
  ctx.font = "16px serif";
  ctx.fillStyle = "#f60";
  ctx.fillRect(10, 10, 120, 30);
  ctx.fillStyle = "#069";
  ctx.fillText("xray privacy probe", 12, 14);
  return simpleHash(canvas.toDataURL());
}

function webglFingerprint(): Pick<FingerprintSnapshot, "webglVendor" | "webglRenderer"> {
  const canvas = document.createElement("canvas");
  const gl = (canvas.getContext("webgl") ||
    canvas.getContext("experimental-webgl")) as WebGLRenderingContext | null;
  if (!gl) {
    return { webglVendor: "", webglRenderer: "" };
  }
  const debugInfo = gl.getExtension("WEBGL_debug_renderer_info");
  if (!debugInfo) {
    return {
      webglVendor: gl.getParameter(gl.VENDOR) || "",
      webglRenderer: gl.getParameter(gl.RENDERER) || "",
    };
  }
  return {
    webglVendor: gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || "",
    webglRenderer: gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || "",
  };
}

export async function collectFingerprintSnapshot(): Promise<FingerprintSnapshot> {
  const nav = navigator as Navigator & { deviceMemory?: number };
  const webgl = webglFingerprint();
  let audioSampleRate: number | null = null;
  try {
    const AudioContextCtor = window.AudioContext || (window as any).webkitAudioContext;
    if (AudioContextCtor) {
      const audio = new AudioContextCtor();
      audioSampleRate = audio.sampleRate || null;
      await audio.close?.();
    }
  } catch {
    audioSampleRate = null;
  }

  return {
    userAgent: nav.userAgent,
    languages: Array.from(nav.languages || []),
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "",
    screen: {
      width: window.screen.width,
      height: window.screen.height,
      colorDepth: window.screen.colorDepth,
      devicePixelRatio: window.devicePixelRatio || 1,
    },
    hardwareConcurrency: nav.hardwareConcurrency || null,
    deviceMemory: nav.deviceMemory || null,
    cookieEnabled: nav.cookieEnabled,
    doNotTrack: nav.doNotTrack || null,
    canvasHash: canvasFingerprint(),
    audioSampleRate,
    ...webgl,
  };
}
