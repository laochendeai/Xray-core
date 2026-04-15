import { describe, expect, it } from "vitest";

import type { NodeRecord } from "@/api/types";
import {
  bindingPreferredNodeId,
  bindingPreviewDomains,
  bindingPrimaryTestDomain,
  failRateValue,
  firstNodeIntelligenceDetail,
  normalizeListInput,
  sortBindingNodes,
  sortPoolNodes,
  sortRemovedNodes,
  summarizeNodeIntelligence,
} from "@/utils/nodePool";

function makeNode(overrides: Partial<NodeRecord>): NodeRecord {
  return {
    id: "node",
    uri: "vless://node",
    remark: "node",
    protocol: "vless",
    address: "example.com",
    port: 443,
    outboundTag: "",
    status: "candidate",
    statusReason: "subscription_node_discovered",
    subscriptionId: "sub-1",
    addedAt: "2026-01-01T00:00:00Z",
    promotedAt: undefined,
    statusUpdatedAt: "2026-01-01T00:00:00Z",
    lastEventAt: "2026-01-01T00:00:00Z",
    totalPings: 0,
    failedPings: 0,
    avgDelayMs: 0,
    consecutiveFails: 0,
    lastCheckedAt: "2026-01-01T00:00:00Z",
    cleanliness: "unknown",
    cleanlinessConfidence: "unknown",
    bandwidthTier: "",
    exitIpStatus: "unknown",
    networkType: "unknown",
    networkTypeConfidence: "unknown",
    ...overrides,
  };
}

describe("node pool utilities", () => {
  it("normalizes newline/comma separated lists and removes duplicates", () => {
    expect(
      normalizeListInput(
        " 1.1.1.1 \n8.8.8.8,1.1.1.1\n\nhttps://dns.google/dns-query ",
      ),
    ).toEqual(["1.1.1.1", "8.8.8.8", "https://dns.google/dns-query"]);
  });

  it("sorts pool nodes by quality and explicit metric modes", () => {
    const trusted = makeNode({
      id: "trusted",
      cleanliness: "trusted",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 120,
      lastCheckedAt: "2026-01-03T00:00:00Z",
    });
    const unknownClean = makeNode({
      id: "unknown",
      cleanliness: "unknown",
      totalPings: 20,
      failedPings: 2,
      avgDelayMs: 200,
      lastCheckedAt: "2026-01-02T00:00:00Z",
    });
    const suspicious = makeNode({
      id: "suspicious",
      cleanliness: "suspicious",
      totalPings: 0,
      failedPings: 0,
      avgDelayMs: 0,
      lastCheckedAt: "2026-01-04T00:00:00Z",
    });
    const highQuality = makeNode({
      id: "high",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 120,
      lastCheckedAt: "2026-01-03T00:00:00Z",
    });
    const mediumQuality = makeNode({
      id: "medium",
      totalPings: 20,
      failedPings: 2,
      avgDelayMs: 200,
      lastCheckedAt: "2026-01-02T00:00:00Z",
    });
    const unknownQuality = makeNode({
      id: "unknown-quality",
      totalPings: 0,
      failedPings: 0,
      avgDelayMs: 0,
      lastCheckedAt: "2026-01-04T00:00:00Z",
    });

    expect(
      sortPoolNodes([unknownClean, suspicious, trusted], "cleanliness_desc").map(
        (node) => node.id,
      ),
    ).toEqual(["trusted", "unknown", "suspicious"]);
    expect(
      sortPoolNodes(
        [mediumQuality, unknownQuality, highQuality],
        "quality",
      ).map((node) => node.id),
    ).toEqual(["high", "medium", "unknown-quality"]);
    expect(
      sortPoolNodes([mediumQuality, highQuality], "last_checked_desc").map(
        (node) => node.id,
      ),
    ).toEqual(["high", "medium"]);
    expect(
      sortPoolNodes([mediumQuality, highQuality], "avg_delay_desc").map(
        (node) => node.id,
      ),
    ).toEqual(["medium", "high"]);
  });

  it("sorts removed nodes by removal time and metrics", () => {
    const newest = makeNode({
      id: "newest",
      status: "removed",
      statusUpdatedAt: "2026-01-04T00:00:00Z",
      totalPings: 10,
      failedPings: 10,
      avgDelayMs: 900,
    });
    const oldest = makeNode({
      id: "oldest",
      status: "removed",
      statusUpdatedAt: "2026-01-01T00:00:00Z",
      totalPings: 10,
      failedPings: 1,
      avgDelayMs: 100,
    });

    expect(
      sortRemovedNodes([oldest, newest], "removed_desc").map((node) => node.id),
    ).toEqual(["newest", "oldest"]);
    expect(
      sortRemovedNodes([oldest, newest], "fail_rate_asc").map(
        (node) => node.id,
      ),
    ).toEqual(["oldest", "newest"]);
  });

  it("returns infinity for nodes without probe samples", () => {
    expect(failRateValue(makeNode({ totalPings: 0, failedPings: 0 }))).toBe(
      Number.POSITIVE_INFINITY,
    );
  });

  it("summarizes cleanliness and network-type verdicts", () => {
    const trustedResidential = makeNode({
      id: "trusted",
      cleanliness: "trusted",
      networkType: "residential_likely",
    });
    const suspiciousDatacenter = makeNode({
      id: "suspicious",
      cleanliness: "suspicious",
      networkType: "datacenter_likely",
    });
    const unknown = makeNode({
      id: "unknown",
      cleanliness: "unknown",
      networkType: "unknown",
    });

    const ispLike = makeNode({
      id: "isp-like",
      cleanliness: "unknown",
      networkType: "isp_likely",
    });

    expect(
      summarizeNodeIntelligence([
        trustedResidential,
        suspiciousDatacenter,
        unknown,
        ispLike,
      ]),
    ).toEqual({
      trustedCount: 1,
      suspiciousCount: 1,
      unknownCleanCount: 2,
      residentialCount: 1,
      ispLikeCount: 1,
      datacenterCount: 1,
      unknownNetworkCount: 1,
    });
  });

  it("keeps isp-like nodes below residential for destination bindings", () => {
    const ispLike = makeNode({
      id: "isp-like",
      cleanliness: "trusted",
      networkType: "isp_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 40,
    });
    const residentialStable = makeNode({
      id: "residential-stable",
      cleanliness: "trusted",
      networkType: "residential_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 1,
      avgDelayMs: 80,
    });

    expect(sortBindingNodes([ispLike, residentialStable]).map((node) => node.id)).toEqual([
      "residential-stable",
      "isp-like",
    ]);
  });

  it("keeps isp-like nodes above datacenter nodes for destination bindings", () => {
    const datacenterFast = makeNode({
      id: "datacenter-fast",
      cleanliness: "trusted",
      networkType: "datacenter_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 30,
    });
    const ispLike = makeNode({
      id: "isp-like",
      cleanliness: "trusted",
      networkType: "isp_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 40,
    });

    expect(sortBindingNodes([datacenterFast, ispLike]).map((node) => node.id)).toEqual([
      "isp-like",
      "datacenter-fast",
    ]);
  });

  it("prefers unknown nodes above isp-like nodes for destination bindings", () => {
    const unknownStable = makeNode({
      id: "unknown-stable",
      cleanliness: "trusted",
      networkType: "unknown",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 50,
    });
    const ispLike = makeNode({
      id: "isp-like",
      cleanliness: "trusted",
      networkType: "isp_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 40,
    });

    expect(sortBindingNodes([ispLike, unknownStable]).map((node) => node.id)).toEqual([
      "unknown-stable",
      "isp-like",
    ]);
  });

  it("picks the first usable intelligence detail", () => {
    expect(
      firstNodeIntelligenceDetail(
        makeNode({
          cleanlinessDetail: "",
          networkTypeDetail: "network detail",
          intelligenceError: "lookup error",
          exitIpError: "exit-ip error",
        }),
      ),
    ).toBe("network detail");

    expect(
      firstNodeIntelligenceDetail(
        makeNode({
          cleanlinessDetail: "",
          networkTypeDetail: "",
          intelligenceError: "",
          exitIpError: "",
        }),
      ),
    ).toBe("");
  });

  it("expands preset destination bindings and picks a test domain", () => {
    expect(bindingPreviewDomains({ preset: "openai", domains: [] })).toContain(
      "domain:api.openai.com",
    );
    expect(bindingPrimaryTestDomain({ preset: "chatgpt", domains: [] })).toBe(
      "chatgpt.com",
    );
    expect(bindingPreviewDomains({ preset: "claude", domains: [] })).toContain(
      "domain:claude.ai",
    );
    expect(bindingPreviewDomains({ preset: "gemini", domains: [] })).toContain(
      "full:generativelanguage.googleapis.com",
    );
    expect(
      bindingPreviewDomains({ preset: "github_copilot", domains: [] }),
    ).toContain("full:copilot.github.com");
    expect(
      bindingPreviewDomains({ preset: "openrouter", domains: [] }),
    ).toContain("domain:openrouter.ai");
    expect(bindingPreviewDomains({ preset: "cursor", domains: [] })).toContain(
      "domain:cursor.com",
    );
    expect(bindingPreviewDomains({ preset: "qwen", domains: [] })).toContain(
      "full:dashscope.aliyuncs.com",
    );
    expect(
      bindingPreviewDomains({ preset: "perplexity", domains: [] }),
    ).toContain("domain:perplexity.ai");
    expect(
      bindingPreviewDomains({ preset: "deepseek", domains: [] }),
    ).toContain("domain:deepseek.com");
    expect(
      bindingPreviewDomains({
        preset: "custom",
        domains: ["*.ignored.example", "domain:custom.example"],
      }),
    ).toEqual(["domain:ignored.example", "domain:custom.example"]);
  });

  it("prefers clean residential nodes for destination bindings", () => {
    const datacenterFast = makeNode({
      id: "datacenter-fast",
      cleanliness: "trusted",
      networkType: "datacenter_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 30,
    });
    const residentialStable = makeNode({
      id: "residential-stable",
      cleanliness: "trusted",
      networkType: "residential_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 1,
      avgDelayMs: 80,
    });
    const suspiciousResidential = makeNode({
      id: "suspicious-residential",
      cleanliness: "suspicious",
      networkType: "residential_likely",
      exitIpStatus: "available",
      totalPings: 20,
      failedPings: 0,
      avgDelayMs: 20,
    });

    expect(
      sortBindingNodes([
        datacenterFast,
        suspiciousResidential,
        residentialStable,
      ]).map((node) => node.id),
    ).toEqual([
      "residential-stable",
      "datacenter-fast",
      "suspicious-residential",
    ]);
    expect(bindingPreferredNodeId([datacenterFast, residentialStable])).toBe(
      "residential-stable",
    );
  });
});
