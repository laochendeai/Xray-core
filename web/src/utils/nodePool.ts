import type {
  NodeRecord,
  TunDestinationBinding,
  TunDestinationBindingPreset,
} from "@/api/types";

export type PoolSortMode =
  | "quality"
  | "cleanliness_desc"
  | "last_checked_desc"
  | "last_checked_asc"
  | "fail_rate_asc"
  | "fail_rate_desc"
  | "avg_delay_asc"
  | "avg_delay_desc";

export type RemovedSortMode =
  | "removed_desc"
  | "removed_asc"
  | "fail_rate_asc"
  | "fail_rate_desc"
  | "avg_delay_asc"
  | "avg_delay_desc";

export interface NodeIntelligenceSummary {
  trustedCount: number;
  suspiciousCount: number;
  unknownCleanCount: number;
  residentialCount: number;
  ispLikeCount: number;
  datacenterCount: number;
  unknownNetworkCount: number;
}

export const tunDestinationBindingPresetDomains: Record<
  Exclude<TunDestinationBindingPreset, "custom">,
  string[]
> = {
  openai: [
    "domain:openai.com",
    "domain:api.openai.com",
    "domain:auth.openai.com",
    "domain:chatgpt.com",
    "domain:chat.openai.com",
    "domain:oaistatic.com",
    "domain:oaiusercontent.com",
  ],
  chatgpt: [
    "domain:chatgpt.com",
    "domain:chat.openai.com",
    "domain:oaistatic.com",
    "domain:oaiusercontent.com",
  ],
  claude: ["domain:claude.ai", "domain:anthropic.com"],
  gemini: [
    "domain:gemini.google.com",
    "domain:ai.google.dev",
    "domain:aistudio.google.com",
    "full:generativelanguage.googleapis.com",
  ],
  github_copilot: [
    "full:github.com",
    "full:api.github.com",
    "full:copilot.github.com",
  ],
  openrouter: ["domain:openrouter.ai"],
  cursor: ["domain:cursor.com"],
  qwen: ["domain:qwen.ai", "full:dashscope.aliyuncs.com"],
  perplexity: ["domain:perplexity.ai"],
  deepseek: ["domain:deepseek.com"],
};

function bindingCleanlinessRank(value: NodeRecord["cleanliness"]): number {
  switch (value) {
    case "trusted":
      return 0;
    case "unknown":
      return 1;
    case "suspicious":
      return 2;
    default:
      return 3;
  }
}

function bindingNetworkRank(value: NodeRecord["networkType"]): number {
  switch (value) {
    case "residential_likely":
      return 0;
    case "unknown":
      return 1;
    case "isp_likely":
      return 2;
    case "datacenter_likely":
      return 3;
    default:
      return 4;
  }
}

function bindingExitIpRank(value: NodeRecord["exitIpStatus"]): number {
  switch (value) {
    case "available":
      return 0;
    case "unknown":
      return 1;
    case "error":
      return 2;
    default:
      return 3;
  }
}

export function sortBindingNodes(entries: NodeRecord[]): NodeRecord[] {
  return [...entries].sort((a, b) => {
    const cleanlinessDiff =
      bindingCleanlinessRank(a.cleanliness) -
      bindingCleanlinessRank(b.cleanliness);
    if (cleanlinessDiff !== 0) return cleanlinessDiff;

    const networkDiff =
      bindingNetworkRank(a.networkType) - bindingNetworkRank(b.networkType);
    if (networkDiff !== 0) return networkDiff;

    const exitIpDiff =
      bindingExitIpRank(a.exitIpStatus) - bindingExitIpRank(b.exitIpStatus);
    if (exitIpDiff !== 0) return exitIpDiff;

    const failRateDiff = compareNullableMetric(
      a.totalPings ? failRateValue(a) : null,
      b.totalPings ? failRateValue(b) : null,
      "asc",
    );
    if (failRateDiff !== 0) return failRateDiff;

    const delayDiff = compareNullableMetric(
      a.avgDelayMs > 0 ? a.avgDelayMs : null,
      b.avgDelayMs > 0 ? b.avgDelayMs : null,
      "asc",
    );
    if (delayDiff !== 0) return delayDiff;

    return checkedAtValue(b) - checkedAtValue(a);
  });
}

export function bindingPreferredNodeId(entries: NodeRecord[]): string {
  return sortBindingNodes(entries)[0]?.id || "";
}

export function failRateValue(
  node: Pick<NodeRecord, "totalPings" | "failedPings">,
): number {
  if (!node.totalPings) return Number.POSITIVE_INFINITY;
  return node.failedPings / node.totalPings;
}

function delayValue(node: Pick<NodeRecord, "avgDelayMs">): number {
  return node.avgDelayMs > 0 ? node.avgDelayMs : Number.POSITIVE_INFINITY;
}

function checkedAtValue(
  node: Pick<NodeRecord, "lastCheckedAt" | "statusUpdatedAt" | "addedAt">,
): number {
  const value = node.lastCheckedAt || node.statusUpdatedAt || node.addedAt;
  const timestamp = value ? new Date(value).getTime() : 0;
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function removedAtValue(
  node: Pick<NodeRecord, "statusUpdatedAt" | "lastEventAt" | "addedAt">,
): number {
  const value = node.statusUpdatedAt || node.lastEventAt || node.addedAt;
  const timestamp = value ? new Date(value).getTime() : 0;
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function compareNodesByQuality(a: NodeRecord, b: NodeRecord): number {
  const failRateDiff = failRateValue(a) - failRateValue(b);
  if (failRateDiff !== 0) return failRateDiff;

  const delayDiff = delayValue(a) - delayValue(b);
  if (delayDiff !== 0) return delayDiff;

  if (a.consecutiveFails !== b.consecutiveFails) {
    return a.consecutiveFails - b.consecutiveFails;
  }

  if (a.totalPings !== b.totalPings) {
    return b.totalPings - a.totalPings;
  }

  return checkedAtValue(b) - checkedAtValue(a);
}

function comparePoolCleanliness(a: NodeRecord, b: NodeRecord): number {
  const diff = bindingCleanlinessRank(a.cleanliness) - bindingCleanlinessRank(b.cleanliness);
  if (diff !== 0) return diff;
  return compareNodesByQuality(a, b);
}

function compareNullableMetric(
  aValue: number | null,
  bValue: number | null,
  direction: "asc" | "desc",
): number {
  if (aValue == null && bValue == null) return 0;
  if (aValue == null) return 1;
  if (bValue == null) return -1;
  return direction === "asc" ? aValue - bValue : bValue - aValue;
}

export function sortPoolNodes(
  entries: NodeRecord[],
  mode: PoolSortMode,
): NodeRecord[] {
  return [...entries].sort((a, b) => {
    switch (mode) {
      case "cleanliness_desc":
        return comparePoolCleanliness(a, b);
      case "last_checked_desc":
        return checkedAtValue(b) - checkedAtValue(a);
      case "last_checked_asc":
        return checkedAtValue(a) - checkedAtValue(b);
      case "fail_rate_asc": {
        const diff = compareNullableMetric(
          a.totalPings ? failRateValue(a) : null,
          b.totalPings ? failRateValue(b) : null,
          "asc",
        );
        return diff !== 0 ? diff : compareNodesByQuality(a, b);
      }
      case "fail_rate_desc": {
        const diff = compareNullableMetric(
          a.totalPings ? failRateValue(a) : null,
          b.totalPings ? failRateValue(b) : null,
          "desc",
        );
        return diff !== 0 ? diff : compareNodesByQuality(a, b);
      }
      case "avg_delay_asc": {
        const diff = compareNullableMetric(
          a.avgDelayMs > 0 ? a.avgDelayMs : null,
          b.avgDelayMs > 0 ? b.avgDelayMs : null,
          "asc",
        );
        return diff !== 0 ? diff : compareNodesByQuality(a, b);
      }
      case "avg_delay_desc": {
        const diff = compareNullableMetric(
          a.avgDelayMs > 0 ? a.avgDelayMs : null,
          b.avgDelayMs > 0 ? b.avgDelayMs : null,
          "desc",
        );
        return diff !== 0 ? diff : compareNodesByQuality(a, b);
      }
      default:
        return compareNodesByQuality(a, b);
    }
  });
}

export function sortRemovedNodes(
  entries: NodeRecord[],
  mode: RemovedSortMode,
): NodeRecord[] {
  return [...entries].sort((a, b) => {
    switch (mode) {
      case "removed_asc":
        return removedAtValue(a) - removedAtValue(b);
      case "fail_rate_asc": {
        const diff = compareNullableMetric(
          a.totalPings ? failRateValue(a) : null,
          b.totalPings ? failRateValue(b) : null,
          "asc",
        );
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a);
      }
      case "fail_rate_desc": {
        const diff = compareNullableMetric(
          a.totalPings ? failRateValue(a) : null,
          b.totalPings ? failRateValue(b) : null,
          "desc",
        );
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a);
      }
      case "avg_delay_asc": {
        const diff = compareNullableMetric(
          a.avgDelayMs > 0 ? a.avgDelayMs : null,
          b.avgDelayMs > 0 ? b.avgDelayMs : null,
          "asc",
        );
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a);
      }
      case "avg_delay_desc": {
        const diff = compareNullableMetric(
          a.avgDelayMs > 0 ? a.avgDelayMs : null,
          b.avgDelayMs > 0 ? b.avgDelayMs : null,
          "desc",
        );
        return diff !== 0 ? diff : removedAtValue(b) - removedAtValue(a);
      }
      default:
        return removedAtValue(b) - removedAtValue(a);
    }
  });
}

export function summarizeNodeIntelligence(
  entries: Array<Pick<NodeRecord, "cleanliness" | "networkType">>,
): NodeIntelligenceSummary {
  return entries.reduce<NodeIntelligenceSummary>(
    (summary, node) => {
      switch (node.cleanliness) {
        case "trusted":
          summary.trustedCount += 1;
          break;
        case "suspicious":
          summary.suspiciousCount += 1;
          break;
        default:
          summary.unknownCleanCount += 1;
          break;
      }

      switch (node.networkType) {
        case "residential_likely":
          summary.residentialCount += 1;
          break;
        case "isp_likely":
          summary.ispLikeCount += 1;
          break;
        case "datacenter_likely":
          summary.datacenterCount += 1;
          break;
        default:
          summary.unknownNetworkCount += 1;
          break;
      }

      return summary;
    },
    {
      trustedCount: 0,
      suspiciousCount: 0,
      unknownCleanCount: 0,
      residentialCount: 0,
      ispLikeCount: 0,
      datacenterCount: 0,
      unknownNetworkCount: 0,
    },
  );
}

export function firstNodeIntelligenceDetail(
  node: Pick<
    NodeRecord,
    | "cleanlinessDetail"
    | "networkTypeDetail"
    | "intelligenceError"
    | "exitIpError"
  >,
): string {
  const candidates = [
    node.cleanlinessDetail,
    node.networkTypeDetail,
    node.intelligenceError,
    node.exitIpError,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }
  return "";
}

export function normalizeListInput(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  );
}

export function bindingPreviewDomains(
  binding: Pick<TunDestinationBinding, "preset" | "domains">,
): string[] {
  if (binding.preset === "custom") {
    return normalizeBindingDomainRules(binding.domains);
  }
  return tunDestinationBindingPresetDomains[binding.preset] || [];
}

export function bindingPrimaryTestDomain(
  binding: Pick<TunDestinationBinding, "preset" | "domains">,
): string {
  const first = bindingPreviewDomains(binding)[0] || "";
  return first.replace(/^(full:|domain:)/, "");
}

function normalizeBindingDomainRules(values: string[]): string[] {
  return Array.from(
    new Set(
      values.map((value) => normalizeBindingDomainRule(value)).filter(Boolean),
    ),
  );
}

function normalizeBindingDomainRule(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("*.")) {
    const host = trimmed.slice(2).replace(/^\.+|\.+$/g, "");
    return host ? `domain:${host}` : "";
  }
  if (trimmed.startsWith(".")) {
    const host = trimmed.slice(1).replace(/^\.+|\.+$/g, "");
    return host ? `domain:${host}` : "";
  }
  return trimmed;
}
