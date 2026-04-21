import { describe, expect, it } from "vitest";

import type { PrivacyDiagnosticsContextResponse } from "@/api/types";
import {
  classifyFingerprintRisk,
  classifyIpExposure,
  classifyRuntimeDnsRisk,
  classifyWebRTCRisk,
  parseIceCandidate,
} from "@/utils/privacyDiagnostics";

describe("privacy diagnostics utilities", () => {
  it("parses WebRTC ICE candidates and detects exposed addresses", () => {
    const host = parseIceCandidate(
      "candidate:842163049 1 udp 1677729535 192.168.1.12 53123 typ host",
    );
    const srflx = parseIceCandidate(
      "candidate:842163049 1 udp 1677729535 203.0.113.44 53124 typ srflx raddr 192.168.1.12 rport 53123",
    );

    expect(host).toMatchObject({
      type: "host",
      protocol: "udp",
      address: "192.168.1.12",
      port: 53123,
      isPrivateAddress: true,
    });
    expect(srflx).toMatchObject({
      type: "srflx",
      protocol: "udp",
      address: "203.0.113.44",
      port: 53124,
      isPrivateAddress: false,
    });

    expect(classifyWebRTCRisk([host, srflx])).toMatchObject({
      leakRisk: "high",
      exposedPrivateAddress: true,
      exposedPublicAddress: true,
    });
  });

  it("marks relay-only WebRTC candidates as low risk", () => {
    const relay = parseIceCandidate(
      "candidate:1 1 udp 2122260223 198.51.100.10 3478 typ relay",
    );

    expect(classifyWebRTCRisk([relay])).toMatchObject({
      leakRisk: "low",
      exposedPrivateAddress: false,
      exposedPublicAddress: false,
    });
  });

  it("classifies IP exposure by comparing browser and runtime egress", () => {
    const context: PrivacyDiagnosticsContextResponse = {
      supported: true,
      tunStatus: {
        status: "running",
        running: true,
        available: true,
        allowRemote: false,
        useSudo: false,
        helperExists: true,
        elevationReady: true,
        helperCurrent: true,
        binaryCurrent: true,
        privilegeInstallRecommended: false,
        binaryPath: "/tmp/xray",
        helperPath: "/tmp/helper",
        stateDir: "/tmp/state",
        runtimeConfigPath: "/tmp/state/config.json",
        interfaceName: "xray0",
        mtu: 1500,
        remoteDns: ["1.1.1.1"],
        configPath: "/tmp/config.json",
        xrayBinary: "/tmp/xray",
        message: "running",
        directEgress: { status: "available", route: "direct", ip: "198.51.100.10" },
        proxyEgress: { status: "available", route: "proxy", ip: "203.0.113.20" },
      },
    };

    expect(classifyIpExposure("203.0.113.20", context)).toMatchObject({
      leakRisk: "low",
      browserMatchesProxy: true,
      browserMatchesDirect: false,
    });
    expect(classifyIpExposure("198.51.100.10", context)).toMatchObject({
      leakRisk: "high",
      browserMatchesDirect: true,
      browserMatchesProxy: false,
    });
  });

  it("uses runtime DNS evidence instead of route mode alone", () => {
    const context: PrivacyDiagnosticsContextResponse = {
      supported: true,
      tunStatus: {
        status: "running",
        running: true,
        available: true,
        allowRemote: false,
        useSudo: false,
        helperExists: true,
        elevationReady: true,
        helperCurrent: true,
        binaryCurrent: true,
        privilegeInstallRecommended: false,
        binaryPath: "/tmp/xray",
        helperPath: "/tmp/helper",
        stateDir: "/tmp/state",
        runtimeConfigPath: "/tmp/state/config.json",
        interfaceName: "xray0",
        mtu: 1500,
        remoteDns: ["1.1.1.1"],
        configPath: "/tmp/config.json",
        xrayBinary: "/tmp/xray",
        message: "running",
        routingDiagnostics: [
          {
            category: "default_proxy_domains",
            dnsPath: "dns-remote",
            resolver: "1.1.1.1",
            route: "proxy(node-pool-active)",
            reason: "remote DNS",
          },
        ],
      },
      tunSettings: {
        selectionPolicy: "fastest",
        routeMode: "strict_proxy",
        remoteDns: ["1.1.1.1"],
        protectDomains: [],
        protectCidrs: [],
        destinationBindings: [],
        aggregation: {
          enabled: false,
          mode: "single_best",
          maxPathsPerSession: 1,
          schedulerPolicy: "single_best",
          relayEndpoint: "",
          health: {
            maxSessionLossPct: 5,
            maxPathJitterMs: 250,
            rollbackOnConsecutiveFailures: 2,
          },
        },
      },
    };

    expect(classifyRuntimeDnsRisk(context)).toMatchObject({
      leakRisk: "low",
      hasRemoteDnsRoute: true,
      hasDirectDnsRoute: false,
    });
  });

  it("treats legacy China DNS direct routing as a leak risk", () => {
    const context: PrivacyDiagnosticsContextResponse = {
      supported: true,
      tunStatus: {
        status: "running",
        running: true,
        available: true,
        allowRemote: false,
        useSudo: false,
        helperExists: true,
        elevationReady: true,
        helperCurrent: true,
        binaryCurrent: true,
        privilegeInstallRecommended: false,
        binaryPath: "/tmp/xray",
        helperPath: "/tmp/helper",
        stateDir: "/tmp/state",
        runtimeConfigPath: "/tmp/state/config.json",
        interfaceName: "xray0",
        mtu: 1500,
        remoteDns: ["1.1.1.1"],
        configPath: "/tmp/config.json",
        xrayBinary: "/tmp/xray",
        message: "running",
        routingDiagnostics: [
          {
            category: "cn_direct_domains",
            dnsPath: "dns-cn",
            resolver: "https://dns.alidns.com/dns-query",
            route: "direct",
            reason: "legacy China DNS direct route",
          },
          {
            category: "default_proxy_domains",
            dnsPath: "dns-remote",
            resolver: "1.1.1.1",
            route: "proxy(node-pool-active)",
            reason: "remote DNS",
          },
        ],
      },
      tunSettings: {
        selectionPolicy: "fastest",
        routeMode: "strict_proxy",
        remoteDns: ["1.1.1.1"],
        protectDomains: [],
        protectCidrs: [],
        destinationBindings: [],
        aggregation: {
          enabled: false,
          mode: "single_best",
          maxPathsPerSession: 1,
          schedulerPolicy: "single_best",
          relayEndpoint: "",
          health: {
            maxSessionLossPct: 5,
            maxPathJitterMs: 250,
            rollbackOnConsecutiveFailures: 2,
          },
        },
      },
    };

    expect(classifyRuntimeDnsRisk(context)).toMatchObject({
      leakRisk: "warning",
      hasRemoteDnsRoute: true,
      hasDirectDnsRoute: true,
    });
  });

  it("raises fingerprint risk when high entropy surfaces are present", () => {
    expect(
      classifyFingerprintRisk({
        userAgent: "Mozilla/5.0",
        languages: ["zh-CN", "en-US"],
        timezone: "Asia/Shanghai",
        screen: { width: 2560, height: 1440, colorDepth: 30, devicePixelRatio: 2 },
        hardwareConcurrency: 16,
        deviceMemory: 16,
        cookieEnabled: true,
        doNotTrack: "1",
        canvasHash: "canvas",
        webglVendor: "NVIDIA Corporation",
        webglRenderer: "ANGLE NVIDIA GeForce",
        audioSampleRate: 48000,
      }),
    ).toMatchObject({
      leakRisk: "high",
      highEntropySurfaceCount: 8,
    });
  });
});
