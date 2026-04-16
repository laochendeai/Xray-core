<template>
  <n-space vertical :size="16" class="node-pool-page">
    <n-space justify="space-between" align="center" wrap>
      <div>
        <h2>{{ t("nodePool.title") }}</h2>
        <p class="node-pool-subtitle">{{ t("nodePool.subtitle") }}</p>
      </div>
      <n-button :loading="loading" @click="refreshAll">{{
        t("common.refresh")
      }}</n-button>
    </n-space>

    <n-alert :type="machineBannerType" :title="machineBannerTitle">
      {{ machineBannerBody }}
    </n-alert>

    <n-card size="small" class="machine-strip">
      <n-space justify="space-between" align="start" wrap>
        <n-space vertical :size="10">
          <n-space align="center" :size="12" wrap>
            <strong>{{ t("nodePool.machineState") }}</strong>
            <n-tag :type="machineStateTagType" size="small">
              {{ machineStateLabel }}
            </n-tag>
            <n-tag :type="summary.healthy ? 'success' : 'warning'" size="small">
              {{
                t("nodePool.activePoolCount", {
                  active: summary.activeNodes,
                  minimum: summary.minActiveNodes,
                })
              }}
            </n-tag>
          </n-space>

          <div class="node-pool-meta">
            {{ machineReasonLabel }}
            <span v-if="formattedMachineChangedAt"
              >· {{ formattedMachineChangedAt }}</span
            >
          </div>

          <div class="node-pool-meta" v-if="tunStatus.message">
            {{ tunStatus.message }}
          </div>
        </n-space>

        <n-space :size="12" wrap class="machine-actions">
          <n-button
            type="primary"
            :loading="tunUpdating"
            @click="handleEnableTransparent"
          >
            {{ t("nodePool.enableTransparent") }}
          </n-button>
          <n-button :loading="tunUpdating" @click="handleRestoreClean">
            {{ t("nodePool.restoreClean") }}
          </n-button>
          <n-button
            type="warning"
            secondary
            :loading="installingTunBootstrap"
            @click="handleInstallTunBootstrap"
          >
            {{ t("settings.installTunPrivilege") }}
          </n-button>
        </n-space>
      </n-space>
    </n-card>

    <n-alert
      v-if="tunRepairRecommended"
      type="warning"
      :title="t('settings.tunRepairTitle')"
    >
      {{ t("settings.tunRepairDesc") }}
    </n-alert>

    <n-alert
      v-if="tunBootstrapNeeded"
      type="warning"
      :title="t('settings.tunPrivilegeTitle')"
    >
      <n-space justify="space-between" align="center" wrap>
        <span>{{ t("settings.tunInstallDesc") }}</span>
        <n-button
          type="warning"
          secondary
          :loading="installingTunBootstrap"
          @click="handleInstallTunBootstrap"
        >
          {{ t("settings.installTunPrivilege") }}
        </n-button>
      </n-space>
    </n-alert>

    <n-collapse class="node-pool-sections">
      <n-collapse-item
        :title="t('nodePool.transparentRoutingConfig')"
        name="transparent-routing"
      >
        <n-card size="small" class="section-block tun-settings-card">
          <n-space vertical :size="12">
            <n-alert type="info">
              {{ t("nodePool.transparentRoutingDesc") }}
            </n-alert>
            <n-alert
              type="success"
              :title="t('nodePool.recommendedConfigTitle')"
            >
              {{ t("nodePool.recommendedConfigBody") }}
            </n-alert>
            <n-alert v-if="tunStatus.running" type="warning">
              {{ t("nodePool.transparentRoutingRunningHint") }}
            </n-alert>
            <div class="node-pool-meta">
              {{ t("nodePool.transparentRoutingWhitelistHint") }}
            </div>
            <n-alert type="info" :title="t('nodePool.remoteDnsGuideTitle')">
              {{ t("nodePool.remoteDnsGuideBody") }}
            </n-alert>
            <div
              v-if="tunSettingsForm.routeMode === 'auto_tested'"
              class="node-pool-meta"
            >
              {{ t("nodePool.routeModeAutoHint") }}
            </div>
            <n-alert
              :type="aggregationStatusAlertType"
              :title="t('nodePool.aggregationStatusTitle')"
            >
              {{ aggregationStatusSummary }}
            </n-alert>
            <n-card
              v-if="aggregationPrototype"
              size="small"
              class="aggregation-prototype-card"
            >
              <n-space vertical :size="12">
                <strong>{{ t("nodePool.aggregationPrototypeTitle") }}</strong>
                <div class="node-pool-meta">
                  {{
                    t("nodePool.aggregationPrototypeSummary", {
                      candidates: aggregationPrototype.candidatePathCount,
                      selected: aggregationPrototype.selectedPathCount,
                      sessions: aggregationPrototype.sessionCount,
                      ttl: aggregationPrototype.sessionTtlSeconds,
                      source: aggregationPrototypeMetricSourceLabel(
                        aggregationPrototype.metricSource,
                      ),
                    })
                  }}
                </div>
                <div v-if="aggregationPrototype.note" class="node-pool-meta">
                  {{ aggregationPrototype.note }}
                </div>

                <div
                  v-if="aggregationPrototype.paths.length"
                  class="aggregation-prototype-block"
                >
                  <strong>{{
                    t("nodePool.aggregationPrototypePathsTitle")
                  }}</strong>
                  <n-list bordered>
                    <n-list-item
                      v-for="path in aggregationPrototype.paths"
                      :key="path.nodeId"
                    >
                      <div class="event-row">
                        <div class="event-main">
                          <n-space align="center" :size="8" wrap>
                            <strong>{{ path.remark || path.nodeId }}</strong>
                            <n-tag
                              size="small"
                              :type="
                                aggregationPrototypePathStateTagType(path.state)
                              "
                            >
                              {{
                                aggregationPrototypePathStateLabel(path.state)
                              }}
                            </n-tag>
                            <n-tag size="small">{{ path.outboundTag }}</n-tag>
                          </n-space>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationPrototypePathMetrics", {
                                latency: aggregationLatencyLabel(
                                  path.latencyMs,
                                ),
                                loss: formatLossPct(path.lossPct),
                                score: formatAggregationScore(path.score),
                                checked:
                                  formatDateTime(path.lastCheckedAt) || "-",
                              })
                            }}
                          </div>
                          <div class="node-pool-meta">{{ path.reason }}</div>
                        </div>
                      </div>
                    </n-list-item>
                  </n-list>
                </div>

                <div
                  v-if="aggregationPrototype.sessions.length"
                  class="aggregation-prototype-block"
                >
                  <strong>{{
                    t("nodePool.aggregationPrototypeSessionsTitle")
                  }}</strong>
                  <n-list bordered>
                    <n-list-item
                      v-for="session in aggregationPrototype.sessions"
                      :key="session.sessionId"
                    >
                      <div class="event-row">
                        <div class="event-main">
                          <n-space align="center" :size="8" wrap>
                            <strong>{{ session.flow }}</strong>
                            <n-tag size="small">{{
                              aggregationSchedulerPolicyLabel(
                                session.schedulerPolicy,
                              )
                            }}</n-tag>
                          </n-space>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationPrototypeSessionSummary", {
                                selected: formatPathIds(
                                  session.selectedPathIds,
                                ),
                                candidates: formatPathIds(
                                  session.candidatePathIds,
                                ),
                                expires:
                                  formatDateTime(session.expiresAt) || "-",
                              })
                            }}
                          </div>
                          <div class="node-pool-meta">{{ session.reason }}</div>
                        </div>
                      </div>
                    </n-list-item>
                  </n-list>
                </div>
              </n-space>
            </n-card>
            <n-card
              v-if="aggregationRelay"
              size="small"
              class="aggregation-prototype-card"
            >
              <n-space vertical :size="12">
                <strong>{{ t("nodePool.aggregationRelayTitle") }}</strong>
                <div class="node-pool-meta">
                  {{
                    t("nodePool.aggregationRelaySummary", {
                      version: aggregationRelay.contractVersion,
                      endpoint: aggregationRelay.endpoint || "-",
                      sessions: aggregationRelay.sessionCount,
                      delivered: aggregationRelay.deliveredPacketCount,
                      packets: aggregationRelay.packetCount,
                      duplicates: aggregationRelay.duplicateDrops,
                      reordered: aggregationRelay.reorderedPackets,
                      buffer: aggregationRelay.maxReorderBufferDepth,
                    })
                  }}
                </div>
                <div v-if="aggregationRelay.note" class="node-pool-meta">
                  {{ aggregationRelay.note }}
                </div>

                <div
                  v-if="aggregationRelay.sessions.length"
                  class="aggregation-prototype-block"
                >
                  <strong>{{
                    t("nodePool.aggregationRelaySessionsTitle")
                  }}</strong>
                  <n-list bordered>
                    <n-list-item
                      v-for="session in aggregationRelay.sessions"
                      :key="session.sessionId"
                    >
                      <div class="event-row">
                        <div class="event-main">
                          <n-space align="center" :size="8" wrap>
                            <strong>{{ session.flow }}</strong>
                            <n-tag size="small">{{
                              aggregationSchedulerPolicyLabel(
                                session.schedulerPolicy,
                              )
                            }}</n-tag>
                          </n-space>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationRelaySessionSummary", {
                                paths: formatPathIds(session.pathIds),
                                delivered: session.deliveredPacketCount,
                                packets: session.packetCount,
                                startup: formatDurationMs(
                                  session.startupLatencyMs,
                                ),
                                stalls: session.stallCount,
                                goodput: formatAggregationGoodput(
                                  session.goodputKbps,
                                ),
                                duplicates: session.duplicateDrops,
                                reordered: session.reorderedPackets,
                                buffer: session.maxReorderBufferDepth,
                              })
                            }}
                          </div>
                          <div class="node-pool-meta">{{ session.reason }}</div>
                        </div>
                      </div>
                    </n-list-item>
                  </n-list>
                </div>
              </n-space>
            </n-card>
            <n-card
              v-if="aggregationBenchmark"
              size="small"
              class="aggregation-prototype-card"
            >
              <n-space vertical :size="12">
                <strong>{{ t("nodePool.aggregationBenchmarkTitle") }}</strong>
                <div class="node-pool-meta">
                  {{
                    t("nodePool.aggregationBenchmarkSummary", {
                      scenarios: aggregationBenchmark.scenarios.length,
                      packets: aggregationBenchmark.packetCount,
                      payload: aggregationBenchmark.payloadBytes,
                    })
                  }}
                </div>
                <div v-if="aggregationBenchmark.note" class="node-pool-meta">
                  {{ aggregationBenchmark.note }}
                </div>

                <div
                  v-if="aggregationBenchmark.scenarios.length"
                  class="aggregation-prototype-block"
                >
                  <strong>{{
                    t("nodePool.aggregationBenchmarkScenariosTitle")
                  }}</strong>
                  <n-list bordered>
                    <n-list-item
                      v-for="scenario in aggregationBenchmark.scenarios"
                      :key="scenario.name"
                    >
                      <div class="event-row">
                        <div class="event-main">
                          <n-space align="center" :size="8" wrap>
                            <strong>{{
                              aggregationBenchmarkScenarioLabel(scenario.name)
                            }}</strong>
                          </n-space>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationBenchmarkResultSummary", {
                                label: t(
                                  "nodePool.aggregationBenchmarkBaseline",
                                ),
                                startup: formatDurationMs(
                                  scenario.baseline.startupLatencyMs,
                                ),
                                stalls: scenario.baseline.stallCount,
                                goodput: formatAggregationGoodput(
                                  scenario.baseline.goodputKbps,
                                ),
                                loss: formatLossPct(scenario.baseline.lossPct),
                                stability: formatLossPct(
                                  scenario.baseline.stabilityPct,
                                ),
                              })
                            }}
                          </div>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationBenchmarkResultSummary", {
                                label: t(
                                  "nodePool.aggregationBenchmarkAggregated",
                                ),
                                startup: formatDurationMs(
                                  scenario.aggregated.startupLatencyMs,
                                ),
                                stalls: scenario.aggregated.stallCount,
                                goodput: formatAggregationGoodput(
                                  scenario.aggregated.goodputKbps,
                                ),
                                loss: formatLossPct(
                                  scenario.aggregated.lossPct,
                                ),
                                stability: formatLossPct(
                                  scenario.aggregated.stabilityPct,
                                ),
                              })
                            }}
                          </div>
                          <div class="node-pool-meta">
                            {{
                              t("nodePool.aggregationBenchmarkGainSummary", {
                                latency: formatSignedMetric(
                                  scenario.startupLatencyGainMs,
                                  "ms",
                                  0,
                                ),
                                stalls: formatSignedInt(
                                  scenario.stallReduction,
                                ),
                                goodput: formatSignedMetric(
                                  scenario.goodputGainKbps,
                                  " kbps",
                                ),
                                loss: formatSignedMetric(
                                  scenario.lossReductionPct,
                                  "pp",
                                ),
                                stability: formatSignedMetric(
                                  scenario.stabilityGainPct,
                                  "pp",
                                ),
                              })
                            }}
                          </div>
                        </div>
                      </div>
                    </n-list-item>
                  </n-list>
                </div>
              </n-space>
            </n-card>
            <n-form
              :model="tunSettingsForm"
              label-placement="left"
              label-width="220px"
            >
              <n-form-item :label="t('nodePool.aggregationEnabled')">
                <n-switch v-model:value="tunSettingsForm.aggregation.enabled" />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationMode')">
                <n-select
                  v-model:value="tunSettingsForm.aggregation.mode"
                  :options="aggregationModeOptions"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationSchedulerPolicy')">
                <n-select
                  v-model:value="tunSettingsForm.aggregation.schedulerPolicy"
                  :options="aggregationSchedulerPolicyOptions"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationMaxPathsPerSession')">
                <n-input-number
                  v-model:value="tunSettingsForm.aggregation.maxPathsPerSession"
                  :min="1"
                  :max="8"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationRelayEndpoint')">
                <n-input
                  v-model:value="tunSettingsForm.aggregation.relayEndpoint"
                  :placeholder="
                    t('nodePool.aggregationRelayEndpointPlaceholder')
                  "
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationMaxSessionLossPct')">
                <n-input-number
                  v-model:value="
                    tunSettingsForm.aggregation.health.maxSessionLossPct
                  "
                  :min="1"
                  :max="100"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.aggregationMaxPathJitterMs')">
                <n-input-number
                  v-model:value="
                    tunSettingsForm.aggregation.health.maxPathJitterMs
                  "
                  :min="1"
                  :max="5000"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item
                :label="t('nodePool.aggregationRollbackOnConsecutiveFailures')"
              >
                <n-input-number
                  v-model:value="
                    tunSettingsForm.aggregation.health
                      .rollbackOnConsecutiveFailures
                  "
                  :min="1"
                  :max="100"
                  :disabled="!tunSettingsForm.aggregation.enabled"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.routeMode')">
                <n-select
                  v-model:value="tunSettingsForm.routeMode"
                  :options="routeModeOptions"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.selectionPolicy')">
                <n-select
                  v-model:value="tunSettingsForm.selectionPolicy"
                  :options="selectionPolicyOptions"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.destinationBindings')">
                <div class="destination-binding-editor">
                  <n-alert
                    type="info"
                    :title="t('nodePool.destinationBindingsTitle')"
                  >
                    {{ t("nodePool.destinationBindingsDesc") }}
                  </n-alert>
                  <div
                    v-if="!destinationBindingDrafts.length"
                    class="node-pool-meta"
                  >
                    {{ t("nodePool.destinationBindingsEmpty") }}
                  </div>
                  <n-alert v-if="!activeNodes.length" type="warning">
                    {{ t("nodePool.destinationBindingsNoActiveNodes") }}
                  </n-alert>
                  <div
                    v-if="destinationBindingDrafts.length"
                    class="destination-binding-list"
                  >
                    <n-card
                      v-for="binding in destinationBindingDrafts"
                      :key="binding.key"
                      size="small"
                      class="destination-binding-card"
                    >
                      <n-space vertical :size="12">
                        <div class="destination-binding-row">
                          <div class="destination-binding-field">
                            <div class="destination-binding-label">
                              {{ t("nodePool.destinationBindingPreset") }}
                            </div>
                            <n-select
                              v-model:value="binding.preset"
                              :options="destinationBindingPresetOptions"
                            />
                          </div>
      <div class="destination-binding-field">
        <div class="destination-binding-label">
          {{ t("nodePool.destinationBindingSelectionMode") }}
        </div>
        <n-select
          v-model:value="binding.selectionMode"
          :options="destinationBindingSelectionModeOptions"
        />
      </div>
      <div class="destination-binding-field">
        <div class="destination-binding-label">
          {{ t("nodePool.destinationBindingNode") }}
        </div>
        <n-select
          v-model:value="binding.nodeId"
          :options="bindingNodeOptions(binding)"
        />
      </div>
      <div class="destination-binding-field">
        <div class="destination-binding-label">
          {{ t("nodePool.destinationBindingFallbackNodes") }}
        </div>
        <n-select
          v-model:value="binding.fallbackNodeIds"
          multiple
          :options="bindingFallbackNodeOptions(binding)"
          :placeholder="t('nodePool.destinationBindingFallbackNodesPlaceholder')"
        />
      </div>
                        </div>
                        <div
                          v-if="binding.preset === 'custom'"
                          class="destination-binding-field"
                        >
                          <div class="destination-binding-label">
                            {{ t("nodePool.destinationBindingDomains") }}
                          </div>
                          <n-input
                            v-model:value="binding.domainsText"
                            type="textarea"
                            :autosize="{ minRows: 3, maxRows: 8 }"
                            :placeholder="
                              t('nodePool.destinationBindingDomainsPlaceholder')
                            "
                          />
                        </div>
                        <div class="node-pool-meta">
                          {{
                            t("nodePool.destinationBindingResolvedDomains", {
                              domains:
                                bindingResolvedDomains(binding).join(", ") ||
                                "-",
                            })
                          }}
                        </div>
                        <div
                          v-if="bindingTargetMissing(binding)"
                          class="node-pool-meta destination-binding-warning"
                        >
                          {{
                            t("nodePool.destinationBindingInactiveTarget", {
                              nodeId: binding.nodeId,
                            })
                          }}
                        </div>
                        <n-space wrap>
                          <n-button
                            size="small"
                            secondary
                            :loading="
                              testingDestinationBindingKey === binding.key
                            "
                            @click="handleTestDestinationBinding(binding)"
                          >
                            {{ t("nodePool.destinationBindingTest") }}
                          </n-button>
                          <n-button
                            size="small"
                            tertiary
                            type="error"
                            @click="removeDestinationBindingDraft(binding.key)"
                          >
                            {{ t("common.delete") }}
                          </n-button>
                        </n-space>
                        <n-alert
                          v-if="destinationBindingTestResults[binding.key]"
                          :type="bindingTestAlertType(binding.key)"
                          class="destination-binding-test-alert"
                        >
                          {{ bindingTestSummary(binding.key) }}
                        </n-alert>
                      </n-space>
                    </n-card>
                  </div>
                  <n-button secondary @click="addDestinationBindingDraft">
                    {{ t("nodePool.destinationBindingAdd") }}
                  </n-button>
                </div>
              </n-form-item>
              <n-form-item :label="t('nodePool.remoteDns')">
                <n-input
                  v-model:value="tunRemoteDnsText"
                  type="textarea"
                  :autosize="{ minRows: 4, maxRows: 10 }"
                  :placeholder="t('nodePool.remoteDnsPlaceholder')"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.protectDomains')">
                <n-input
                  v-model:value="tunProtectDomainsText"
                  type="textarea"
                  :autosize="{ minRows: 4, maxRows: 10 }"
                  :placeholder="t('nodePool.protectDomainsPlaceholder')"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.protectCidrs')">
                <n-input
                  v-model:value="tunProtectCidrsText"
                  type="textarea"
                  :autosize="{ minRows: 4, maxRows: 10 }"
                  :placeholder="t('nodePool.protectCidrsPlaceholder')"
                />
              </n-form-item>
              <n-form-item>
                <n-button
                  type="primary"
                  :loading="savingTunSettings"
                  @click="handleSaveTunSettings"
                >
                  {{ t("nodePool.saveTunSettings") }}
                </n-button>
              </n-form-item>
            </n-form>
          </n-space>
        </n-card>
      </n-collapse-item>
    </n-collapse>

    <div class="summary-strip">
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.status.active") }}</div>
        <div class="summary-value">{{ summary.activeCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.status.staging") }}</div>
        <div class="summary-value">{{ summary.stagingCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.status.quarantine") }}</div>
        <div class="summary-value">{{ summary.quarantineCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.status.candidate") }}</div>
        <div class="summary-value">{{ summary.candidateCount }}</div>
      </div>
    </div>

    <div class="node-pool-meta">
      {{ t("nodePool.intelligenceSummaryTitle") }}
    </div>
    <div class="summary-strip">
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.cleanliness.trusted") }}</div>
        <div class="summary-value">{{ intelligenceSummary.trustedCount }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">
          {{ t("nodePool.cleanliness.suspicious") }}
        </div>
        <div class="summary-value">
          {{ intelligenceSummary.suspiciousCount }}
        </div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.cleanliness.unknown") }}</div>
        <div class="summary-value">
          {{ intelligenceSummary.unknownCleanCount }}
        </div>
      </div>
      <div class="summary-item">
        <div class="summary-label">
          {{ t("nodePool.networkType.residential_likely") }}
        </div>
        <div class="summary-value">
          {{ intelligenceSummary.residentialCount }}
        </div>
      </div>
      <div class="summary-item">
        <div class="summary-label">
          {{ t("nodePool.networkType.isp_likely") }}
        </div>
        <div class="summary-value">
          {{ intelligenceSummary.ispLikeCount }}
        </div>
      </div>
      <div class="summary-item">
        <div class="summary-label">
          {{ t("nodePool.networkType.datacenter_likely") }}
        </div>
        <div class="summary-value">
          {{ intelligenceSummary.datacenterCount }}
        </div>
      </div>
      <div class="summary-item">
        <div class="summary-label">{{ t("nodePool.networkType.unknown") }}</div>
        <div class="summary-value">
          {{ intelligenceSummary.unknownNetworkCount }}
        </div>
      </div>
    </div>

    <n-collapse class="node-pool-sections">
      <n-collapse-item
        :title="t('nodePool.lifecycleGuideTitle')"
        name="pool-guide"
      >
        <section class="section-block">
          <div class="pool-guide-grid">
            <div
              v-for="step in poolLifecycleSteps"
              :key="step.status"
              class="pool-guide-item"
            >
              <n-tag size="small" :type="statusTagType(step.status)">
                {{ statusLabel(step.status) }}
              </n-tag>
              <div class="pool-guide-desc">{{ step.description }}</div>
            </div>
          </div>
          <div class="node-pool-meta">
            {{ t("nodePool.lifecycleGuideHint") }}
          </div>
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="t('nodePool.fullFailureCleanupTitle')"
        name="full-failure-cleanup"
      >
        <section class="section-block">
          <n-space justify="space-between" align="center" wrap>
            <n-space align="center" :size="8" wrap>
              <span class="node-pool-meta">
                {{
                  t("nodePool.fullFailureCleanupCount", {
                    count: fullFailureRemovalCandidates.length,
                  })
                }}
              </span>
              <n-input-number
                v-model:value="fullFailureThreshold"
                :min="1"
                :max="100000"
              />
              <n-popconfirm @positive-click="handleBulkRemoveFullFailures">
                <template #trigger>
                  <n-button
                    type="warning"
                    :disabled="!fullFailureRemovalCandidates.length"
                    :loading="bulkRemovingFullFailures"
                  >
                    {{ t("nodePool.fullFailureCleanupAction") }}
                  </n-button>
                </template>
                {{
                  t("nodePool.fullFailureCleanupConfirm", {
                    count: fullFailureRemovalCandidates.length,
                  })
                }}
              </n-popconfirm>
            </n-space>
          </n-space>
          <div class="node-pool-meta">
            {{ t("nodePool.fullFailureCleanupHint") }}
          </div>
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="`${t('nodePool.recentEvents')} (${recentEvents.length})`"
        name="recent-events"
      >
        <section class="section-block">
          <div class="node-pool-meta" v-if="formattedSummaryEvaluatedAt">
            {{ t("nodePool.lastEvaluatedAt") }}:
            {{ formattedSummaryEvaluatedAt }}
          </div>
          <n-empty
            v-if="!recentEvents.length"
            :description="t('nodePool.emptyEvents')"
          />
          <n-list v-else bordered>
            <n-list-item
              v-for="event in recentEvents"
              :key="`${event.nodeId}-${event.at}-${event.reason}`"
            >
              <div class="event-row">
                <div class="event-main">
                  <n-space align="center" :size="8" wrap>
                    <strong>{{
                      event.remark || event.nodeAddress || event.nodeId
                    }}</strong>
                    <n-tag size="small" :type="statusTagType(event.status)">
                      {{ statusLabel(event.status) }}
                    </n-tag>
                    <n-tag size="small">
                      {{ reasonLabel(event.reason) }}
                    </n-tag>
                  </n-space>
                  <div class="node-pool-meta">
                    {{ event.details || event.nodeAddress || event.nodeId }}
                  </div>
                </div>
                <div class="event-time">{{ formatDateTime(event.at) }}</div>
              </div>
            </n-list-item>
          </n-list>
        </section>
      </n-collapse-item>
    </n-collapse>

    <n-collapse class="node-pool-sections">
      <n-collapse-item
        :title="`${t('nodePool.status.active')} (${activeNodes.length})`"
        name="active-pool"
      >
        <section class="section-block">
          <n-space justify="space-between" align="center" wrap>
            <n-space align="center" :size="8" wrap>
              <n-button
                v-if="activeNodes.length"
                size="small"
                secondary
                :disabled="!activeExportableNodes.length"
                @click="handleExportActiveNodes"
              >
                {{ t("nodePool.exportActivePool") }}
              </n-button>
              <n-select
                v-model:value="activeSortMode"
                :options="poolSortOptions"
                class="pool-sort-select"
              />
            </n-space>
          </n-space>
          <template v-if="activeNodes.length">
            <n-data-table
              v-if="!isCompact"
              :columns="activeColumns"
              :data="activeNodes"
              :loading="loading"
              :scroll-x="1680"
              :pagination="{ pageSize: 10 }"
            />
            <div v-else class="node-card-list">
              <div v-for="node in activeNodes" :key="node.id" class="node-card">
                <div class="node-card-header">
                  <strong>{{ node.remark || node.address }}</strong>
                  <n-tag size="small" :type="statusTagType(node.status)">
                    {{ statusLabel(node.status) }}
                  </n-tag>
                </div>
                <div class="node-card-meta">
                  {{ node.address }}:{{ node.port }}
                </div>
                <div class="node-card-meta">{{ nodeReasonLabel(node) }}</div>
                <n-space :size="8" wrap>
                  <n-tag
                    v-if="showSubscriptionMissingTag(node)"
                    size="small"
                    type="warning"
                  >
                    {{ t("nodePool.subscriptionMissingFlag") }}
                  </n-tag>
                  <n-popover trigger="hover" placement="top-start">
                    <template #trigger>
                      <n-space :size="8" wrap class="node-intelligence-trigger">
                        <n-tag
                          size="small"
                          :type="cleanlinessTagType(node.cleanliness)"
                        >
                          {{ cleanlinessLabel(node.cleanliness) }}
                        </n-tag>
                        <n-tag
                          size="small"
                          :type="networkTypeTagType(node.networkType)"
                        >
                          {{ networkTypeLabel(node.networkType) }}
                        </n-tag>
                      </n-space>
                    </template>
                    <div class="node-intelligence-popover">
                      <div class="node-intelligence-popover-title">
                        {{ t("nodePool.intelligenceLabel") }}
                      </div>
                      <div
                        v-for="(line, index) in nodeIntelligencePopoverLines(
                          node,
                        )"
                        :key="`${node.id}-active-${index}`"
                        class="node-intelligence-popover-line"
                      >
                        {{ line }}
                      </div>
                    </div>
                  </n-popover>
                  <n-tag size="small">{{ delayLabel(node) }}</n-tag>
                  <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
                </n-space>
                <n-space :size="8" class="node-card-actions">
                  <n-button
                    size="small"
                    type="warning"
                    @click="handleQuarantine(node.id)"
                  >
                    {{ t("nodePool.quarantine") }}
                  </n-button>
                  <n-button
                    size="small"
                    type="error"
                    @click="handleRemove(node.id)"
                  >
                    {{ t("nodePool.remove") }}
                  </n-button>
                </n-space>
              </div>
            </div>
          </template>
          <n-empty v-else :description="t('nodePool.emptyActive')" />
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="`${t('nodePool.status.staging')} (${stagingNodes.length})`"
        name="staging-pool"
      >
        <section class="section-block">
          <n-space justify="space-between" align="center" wrap>
            <n-space align="center" :size="8" wrap>
              <span v-if="selectedStagingIds.length" class="node-pool-meta">
                {{
                  t("nodePool.selectedCount", {
                    count: selectedStagingIds.length,
                  })
                }}
              </span>
              <n-button
                v-if="stagingNodes.length"
                size="small"
                type="success"
                :disabled="!selectedStagingIds.length"
                :loading="bulkPromotingGroup === 'staging'"
                @click="handleBulkPromote('staging')"
              >
                {{ t("nodePool.bulkPromote") }}
              </n-button>
              <n-button
                v-if="selectedStagingIds.length"
                size="small"
                secondary
                @click="clearSelection('staging')"
              >
                {{ t("common.reset") }}
              </n-button>
              <n-select
                v-model:value="stagingSortMode"
                :options="poolSortOptions"
                class="pool-sort-select"
              />
            </n-space>
          </n-space>
          <div v-if="validationPoolHint" class="node-pool-meta">
            {{ validationPoolHint }}
          </div>
          <template v-if="stagingNodes.length">
            <n-data-table
              v-if="!isCompact"
              :columns="stagingColumns"
              :data="stagingNodes"
              :row-key="rowKey"
              :checked-row-keys="selectedStagingIds"
              :loading="loading"
              :scroll-x="1680"
              :pagination="{ pageSize: 10 }"
              @update:checked-row-keys="
                (keys) => handleCheckedRowKeysUpdate('staging', keys)
              "
            />
            <div v-else class="node-card-list">
              <div
                v-for="node in stagingNodes"
                :key="node.id"
                class="node-card"
              >
                <div class="node-card-header">
                  <n-checkbox
                    :checked="isSelected('staging', node.id)"
                    @update:checked="
                      (checked) => setCardSelection('staging', node.id, checked)
                    "
                  >
                    {{ node.remark || node.address }}
                  </n-checkbox>
                  <n-tag size="small" :type="statusTagType(node.status)">
                    {{ statusLabel(node.status) }}
                  </n-tag>
                </div>
                <div class="node-card-meta">
                  {{ node.address }}:{{ node.port }}
                </div>
                <div class="node-card-meta">{{ nodeReasonLabel(node) }}</div>
                <n-space :size="8" wrap>
                  <n-tag
                    v-if="showSubscriptionMissingTag(node)"
                    size="small"
                    type="warning"
                  >
                    {{ t("nodePool.subscriptionMissingFlag") }}
                  </n-tag>
                  <n-popover trigger="hover" placement="top-start">
                    <template #trigger>
                      <n-space :size="8" wrap class="node-intelligence-trigger">
                        <n-tag
                          size="small"
                          :type="cleanlinessTagType(node.cleanliness)"
                        >
                          {{ cleanlinessLabel(node.cleanliness) }}
                        </n-tag>
                        <n-tag
                          size="small"
                          :type="networkTypeTagType(node.networkType)"
                        >
                          {{ networkTypeLabel(node.networkType) }}
                        </n-tag>
                      </n-space>
                    </template>
                    <div class="node-intelligence-popover">
                      <div class="node-intelligence-popover-title">
                        {{ t("nodePool.intelligenceLabel") }}
                      </div>
                      <div
                        v-for="(line, index) in nodeIntelligencePopoverLines(
                          node,
                        )"
                        :key="`${node.id}-candidate-${index}`"
                        class="node-intelligence-popover-line"
                      >
                        {{ line }}
                      </div>
                    </div>
                  </n-popover>
                  <n-tag size="small">{{ delayLabel(node) }}</n-tag>
                  <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
                </n-space>
                <n-space :size="8" class="node-card-actions">
                  <n-button
                    size="small"
                    type="success"
                    @click="handlePromote(node.id)"
                  >
                    {{ t("nodePool.promote") }}
                  </n-button>
                  <n-button
                    size="small"
                    type="error"
                    @click="handleRemove(node.id)"
                  >
                    {{ t("nodePool.remove") }}
                  </n-button>
                </n-space>
              </div>
            </div>
          </template>
          <n-empty v-else :description="t('nodePool.emptyStaging')" />
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="`${t('nodePool.status.quarantine')} (${quarantineNodes.length})`"
        name="quarantine-pool"
      >
        <section class="section-block">
          <n-space justify="space-between" align="center" wrap>
            <n-space align="center" :size="8" wrap>
              <span v-if="selectedQuarantineIds.length" class="node-pool-meta">
                {{
                  t("nodePool.selectedCount", {
                    count: selectedQuarantineIds.length,
                  })
                }}
              </span>
              <n-button
                v-if="quarantineNodes.length"
                size="small"
                type="success"
                :disabled="!selectedQuarantineIds.length"
                :loading="bulkPromotingGroup === 'quarantine'"
                @click="handleBulkPromote('quarantine')"
              >
                {{ t("nodePool.bulkPromote") }}
              </n-button>
              <n-button
                v-if="selectedQuarantineIds.length"
                size="small"
                secondary
                @click="clearSelection('quarantine')"
              >
                {{ t("common.reset") }}
              </n-button>
              <n-button
                v-if="quarantineNodes.length"
                size="small"
                type="warning"
                :loading="bulkRemoving"
                @click="handleBulkRemoveQuarantine"
              >
                {{ t("nodePool.bulkRemoveUnstable") }}
              </n-button>
              <n-select
                v-model:value="quarantineSortMode"
                :options="poolSortOptions"
                class="pool-sort-select"
              />
            </n-space>
          </n-space>
          <template v-if="quarantineNodes.length">
            <n-data-table
              v-if="!isCompact"
              :columns="quarantineColumns"
              :data="quarantineNodes"
              :row-key="rowKey"
              :checked-row-keys="selectedQuarantineIds"
              :loading="loading"
              :scroll-x="1680"
              :pagination="{ pageSize: 10 }"
              @update:checked-row-keys="
                (keys) => handleCheckedRowKeysUpdate('quarantine', keys)
              "
            />
            <div v-else class="node-card-list">
              <div
                v-for="node in quarantineNodes"
                :key="node.id"
                class="node-card"
              >
                <div class="node-card-header">
                  <n-checkbox
                    :checked="isSelected('quarantine', node.id)"
                    @update:checked="
                      (checked) =>
                        setCardSelection('quarantine', node.id, checked)
                    "
                  >
                    {{ node.remark || node.address }}
                  </n-checkbox>
                  <n-tag size="small" :type="statusTagType(node.status)">
                    {{ statusLabel(node.status) }}
                  </n-tag>
                </div>
                <div class="node-card-meta">
                  {{ node.address }}:{{ node.port }}
                </div>
                <div class="node-card-meta">{{ nodeReasonLabel(node) }}</div>
                <n-space :size="8" wrap>
                  <n-tag
                    v-if="showSubscriptionMissingTag(node)"
                    size="small"
                    type="warning"
                  >
                    {{ t("nodePool.subscriptionMissingFlag") }}
                  </n-tag>
                  <n-popover trigger="hover" placement="top-start">
                    <template #trigger>
                      <n-space :size="8" wrap class="node-intelligence-trigger">
                        <n-tag
                          size="small"
                          :type="cleanlinessTagType(node.cleanliness)"
                        >
                          {{ cleanlinessLabel(node.cleanliness) }}
                        </n-tag>
                        <n-tag
                          size="small"
                          :type="networkTypeTagType(node.networkType)"
                        >
                          {{ networkTypeLabel(node.networkType) }}
                        </n-tag>
                      </n-space>
                    </template>
                    <div class="node-intelligence-popover">
                      <div class="node-intelligence-popover-title">
                        {{ t("nodePool.intelligenceLabel") }}
                      </div>
                      <div
                        v-for="(line, index) in nodeIntelligencePopoverLines(
                          node,
                        )"
                        :key="`${node.id}-standby-${index}`"
                        class="node-intelligence-popover-line"
                      >
                        {{ line }}
                      </div>
                    </div>
                  </n-popover>
                  <n-tag size="small">{{ delayLabel(node) }}</n-tag>
                  <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
                </n-space>
                <n-space :size="8" class="node-card-actions">
                  <n-button
                    size="small"
                    type="success"
                    @click="handlePromote(node.id)"
                  >
                    {{ t("nodePool.promote") }}
                  </n-button>
                  <n-button
                    size="small"
                    type="error"
                    @click="handleRemove(node.id)"
                  >
                    {{ t("nodePool.remove") }}
                  </n-button>
                </n-space>
              </div>
            </div>
          </template>
          <n-empty v-else :description="t('nodePool.emptyQuarantine')" />
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="`${t('nodePool.status.candidate')} (${candidateNodes.length})`"
        name="candidate-pool"
      >
        <section class="section-block">
          <n-space justify="space-between" align="center" wrap>
            <n-space align="center" :size="8" wrap>
              <span v-if="selectedCandidateIds.length" class="node-pool-meta">
                {{
                  t("nodePool.selectedCount", {
                    count: selectedCandidateIds.length,
                  })
                }}
              </span>
              <n-button
                v-if="candidateNodes.length"
                size="small"
                type="primary"
                :disabled="!selectedCandidateIds.length"
                :loading="bulkValidating"
                @click="handleBulkValidateCandidate"
              >
                {{ t("nodePool.bulkValidate") }}
              </n-button>
              <n-popconfirm
                v-if="selectedCandidateIds.length"
                @positive-click="handleBulkRemoveCandidate"
              >
                <template #trigger>
                  <n-button
                    size="small"
                    type="error"
                    secondary
                    :loading="bulkRemovingCandidates"
                  >
                    {{ t("nodePool.bulkRemoveSelectedCandidates") }}
                  </n-button>
                </template>
                {{
                  t("nodePool.bulkRemoveSelectedCandidatesConfirm", {
                    count: selectedCandidateIds.length,
                  })
                }}
              </n-popconfirm>
              <n-button
                v-if="selectedCandidateIds.length"
                size="small"
                secondary
                @click="clearSelection('candidate')"
              >
                {{ t("common.reset") }}
              </n-button>
              <n-select
                v-model:value="candidateSortMode"
                :options="poolSortOptions"
                class="pool-sort-select"
              />
            </n-space>
          </n-space>
          <div class="node-pool-meta">
            {{ t("nodePool.candidateSectionHint") }}
          </div>
          <template v-if="candidateNodes.length">
            <n-data-table
              v-if="!isCompact"
              :columns="candidateColumns"
              :data="candidateNodes"
              :row-key="rowKey"
              :checked-row-keys="selectedCandidateIds"
              :loading="loading"
              :scroll-x="1680"
              :pagination="{ pageSize: 10 }"
              @update:checked-row-keys="
                (keys) => handleCheckedRowKeysUpdate('candidate', keys)
              "
            />
            <div v-else class="node-card-list">
              <div
                v-for="node in candidateNodes"
                :key="node.id"
                class="node-card"
              >
                <div class="node-card-header">
                  <n-checkbox
                    :checked="isSelected('candidate', node.id)"
                    @update:checked="
                      (checked) =>
                        setCardSelection('candidate', node.id, checked)
                    "
                  >
                    {{ node.remark || node.address }}
                  </n-checkbox>
                  <n-tag size="small" :type="statusTagType(node.status)">
                    {{ statusLabel(node.status) }}
                  </n-tag>
                </div>
                <div class="node-card-meta">
                  {{ node.address }}:{{ node.port }}
                </div>
                <div class="node-card-meta">{{ nodeReasonLabel(node) }}</div>
                <n-space :size="8" wrap>
                  <n-tag
                    v-if="showSubscriptionMissingTag(node)"
                    size="small"
                    type="warning"
                  >
                    {{ t("nodePool.subscriptionMissingFlag") }}
                  </n-tag>
                  <n-popover trigger="hover" placement="top-start">
                    <template #trigger>
                      <n-space :size="8" wrap class="node-intelligence-trigger">
                        <n-tag
                          size="small"
                          :type="cleanlinessTagType(node.cleanliness)"
                        >
                          {{ cleanlinessLabel(node.cleanliness) }}
                        </n-tag>
                        <n-tag
                          size="small"
                          :type="networkTypeTagType(node.networkType)"
                        >
                          {{ networkTypeLabel(node.networkType) }}
                        </n-tag>
                      </n-space>
                    </template>
                    <div class="node-intelligence-popover">
                      <div class="node-intelligence-popover-title">
                        {{ t("nodePool.intelligenceLabel") }}
                      </div>
                      <div
                        v-for="(line, index) in nodeIntelligencePopoverLines(
                          node,
                        )"
                        :key="`${node.id}-quarantine-${index}`"
                        class="node-intelligence-popover-line"
                      >
                        {{ line }}
                      </div>
                    </div>
                  </n-popover>
                  <n-tag size="small">{{ failRateLabel(node) }}</n-tag>
                </n-space>
                <n-space :size="8" class="node-card-actions">
                  <n-button
                    size="small"
                    type="primary"
                    @click="handleValidate(node.id)"
                  >
                    {{ t("nodePool.validate") }}
                  </n-button>
                  <n-button
                    size="small"
                    type="error"
                    @click="handleRemove(node.id)"
                  >
                    {{ t("nodePool.remove") }}
                  </n-button>
                </n-space>
              </div>
            </div>
          </template>
          <n-empty v-else :description="t('nodePool.emptyCandidate')" />
        </section>
      </n-collapse-item>

      <n-collapse-item
        :title="`${t('nodePool.status.removed')} (${removedNodes.length})`"
        name="removed-pool"
      >
        <section class="section-block">
          <n-space
            justify="space-between"
            align="center"
            wrap
            class="removed-toolbar"
          >
            <n-space align="center" :size="8" wrap>
              <div class="node-pool-meta">
                {{ t("nodePool.removedSectionHint") }}
              </div>
              <span v-if="selectedRemovedIds.length" class="node-pool-meta">
                {{
                  t("nodePool.selectedCount", {
                    count: selectedRemovedIds.length,
                  })
                }}
              </span>
              <span
                v-if="removedFullFailureNodes.length"
                class="node-pool-meta"
              >
                {{
                  t("nodePool.removedFullFailureCount", {
                    count: removedFullFailureNodes.length,
                  })
                }}
              </span>
              <n-button
                v-if="removedNodes.length"
                size="small"
                type="primary"
                :disabled="!selectedRemovedIds.length"
                :loading="bulkRestoringRemoved"
                @click="handleBulkRestoreRemoved"
              >
                {{ t("nodePool.bulkRestoreCandidate") }}
              </n-button>
              <n-button
                v-if="selectedRemovedIds.length"
                size="small"
                secondary
                @click="clearSelection('removed')"
              >
                {{ t("common.reset") }}
              </n-button>
              <n-popconfirm
                v-if="selectedRemovedIds.length"
                @positive-click="handleBulkPurgeSelectedRemoved"
              >
                <template #trigger>
                  <n-button
                    size="small"
                    type="error"
                    secondary
                    :loading="bulkPurgingSelectedRemoved"
                  >
                    {{ t("nodePool.bulkPurgeSelectedRemoved") }}
                  </n-button>
                </template>
                {{
                  t("nodePool.bulkPurgeSelectedRemovedConfirm", {
                    count: selectedRemovedIds.length,
                  })
                }}
              </n-popconfirm>
              <n-popconfirm
                v-if="removedFullFailureNodes.length"
                @positive-click="handleBulkPurgeRemovedFullFailures"
              >
                <template #trigger>
                  <n-button
                    size="small"
                    type="error"
                    secondary
                    :loading="bulkPurgingRemoved"
                  >
                    {{ t("nodePool.bulkPurgeRemovedFullFailures") }}
                  </n-button>
                </template>
                {{
                  t("nodePool.bulkPurgeRemovedFullFailuresConfirm", {
                    count: removedFullFailureNodes.length,
                  })
                }}
              </n-popconfirm>
            </n-space>
            <n-select
              v-model:value="removedSortMode"
              :options="removedSortOptions"
              class="pool-sort-select"
            />
          </n-space>
          <n-empty
            v-if="!removedNodes.length"
            :description="t('nodePool.emptyRemoved')"
          />
          <n-data-table
            v-else-if="!isCompact"
            :columns="removedColumns"
            :data="removedNodes"
            :row-key="rowKey"
            :checked-row-keys="selectedRemovedIds"
            :loading="loading"
            :scroll-x="1680"
            :pagination="{ pageSize: 10 }"
            @update:checked-row-keys="
              (keys) => handleCheckedRowKeysUpdate('removed', keys)
            "
          />
          <div v-else class="node-card-list">
            <div v-for="node in removedNodes" :key="node.id" class="node-card">
              <div class="node-card-header">
                <n-checkbox
                  :checked="isSelected('removed', node.id)"
                  @update:checked="
                    (checked) => setCardSelection('removed', node.id, checked)
                  "
                >
                  {{ node.remark || node.address }}
                </n-checkbox>
                <n-tag size="small" :type="statusTagType(node.status)">
                  {{ statusLabel(node.status) }}
                </n-tag>
              </div>
              <div class="node-card-meta">
                {{ node.address }}:{{ node.port }}
              </div>
              <div class="node-card-meta">{{ nodeReasonLabel(node) }}</div>
              <div class="node-card-meta">
                {{ t("nodePool.removedAt") }}:
                {{
                  formatDateTime(
                    node.statusUpdatedAt || node.lastEventAt || node.addedAt,
                  ) || "-"
                }}
              </div>
              <n-space :size="8" wrap>
                <n-tag
                  v-if="showSubscriptionMissingTag(node)"
                  size="small"
                  type="warning"
                >
                  {{ t("nodePool.subscriptionMissingFlag") }}
                </n-tag>
                <n-popover trigger="hover" placement="top-start">
                  <template #trigger>
                    <n-space :size="8" wrap class="node-intelligence-trigger">
                      <n-tag
                        size="small"
                        :type="cleanlinessTagType(node.cleanliness)"
                      >
                        {{ cleanlinessLabel(node.cleanliness) }}
                      </n-tag>
                      <n-tag
                        size="small"
                        :type="networkTypeTagType(node.networkType)"
                      >
                        {{ networkTypeLabel(node.networkType) }}
                      </n-tag>
                    </n-space>
                  </template>
                  <div class="node-intelligence-popover">
                    <div class="node-intelligence-popover-title">
                      {{ t("nodePool.intelligenceLabel") }}
                    </div>
                    <div
                      v-for="(line, index) in nodeIntelligencePopoverLines(
                        node,
                      )"
                      :key="`${node.id}-removed-${index}`"
                      class="node-intelligence-popover-line"
                    >
                      {{ line }}
                    </div>
                  </div>
                </n-popover>
              </n-space>
              <n-space :size="8" class="node-card-actions">
                <n-button
                  size="small"
                  type="primary"
                  @click="handleRestore(node.id)"
                >
                  {{ t("nodePool.restoreNode") }}
                </n-button>
                <n-popconfirm @positive-click="handlePurgeRemoved(node.id)">
                  <template #trigger>
                    <n-button
                      size="small"
                      type="error"
                      secondary
                      :loading="purgingRemovedNodeId === node.id"
                    >
                      {{ t("nodePool.purgeRemovedNode") }}
                    </n-button>
                  </template>
                  {{ t("nodePool.purgeRemovedConfirm") }}
                </n-popconfirm>
              </n-space>
            </div>
          </div>
        </section>
      </n-collapse-item>
    </n-collapse>

    <n-collapse class="node-pool-sections">
      <n-collapse-item
        :title="t('nodePool.validationConfig')"
        name="validation-config"
      >
        <section class="section-block">
          <div class="validation-config-layout">
            <n-form
              :model="configForm"
              label-placement="left"
              label-width="220px"
              class="validation-config-form"
            >
              <n-form-item :label="t('nodePool.minActiveNodes')">
                <n-input-number
                  v-model:value="configForm.minActiveNodes"
                  :min="1"
                  :max="20"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.minSamples')">
                <n-input-number
                  v-model:value="configForm.minSamples"
                  :min="1"
                  :max="100"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.maxFailRate')">
                <n-input-number
                  v-model:value="configForm.maxFailRate"
                  :min="0"
                  :max="1"
                  :step="0.05"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.maxAvgDelay')">
                <n-input-number
                  v-model:value="configForm.maxAvgDelayMs"
                  :min="100"
                  :max="10000"
                  :step="100"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.demoteAfterFails')">
                <n-input-number
                  v-model:value="configForm.demoteAfterFails"
                  :min="1"
                  :max="50"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.probeInterval')">
                <n-input-number
                  v-model:value="configForm.probeIntervalSec"
                  :min="10"
                  :max="3600"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.probeUrl')">
                <n-input v-model:value="configForm.probeUrl" />
              </n-form-item>
              <n-form-item :label="t('nodePool.minBandwidthKbps')">
                <n-input-number
                  v-model:value="configForm.minBandwidthKbps"
                  :min="0"
                  :max="1000000"
                  :step="1000"
                />
              </n-form-item>
              <n-form-item :label="t('nodePool.autoRemoveDemoted')">
                <n-switch v-model:value="configForm.autoRemoveDemoted" />
              </n-form-item>
              <n-form-item>
                <n-button
                  type="primary"
                  :loading="savingConfig"
                  @click="handleSaveConfig"
                >
                  {{ t("nodePool.saveConfig") }}
                </n-button>
              </n-form-item>
            </n-form>

            <div class="validation-config-guide">
              <n-alert type="info" :title="t('nodePool.validationGuideTitle')">
                {{ t("nodePool.validationGuideSummary") }}
              </n-alert>
              <div class="validation-guide-list">
                <div
                  v-for="item in validationConfigGuideItems"
                  :key="item.key"
                  class="validation-guide-item"
                >
                  <strong>{{ item.label }}</strong>
                  <div class="node-pool-meta">{{ item.description }}</div>
                </div>
              </div>
              <div class="validation-guide-footnote">
                {{ t("nodePool.validationGuideRuleOfThumb") }}
              </div>
            </div>
          </div>
        </section>
      </n-collapse-item>
    </n-collapse>
  </n-space>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from "vue";
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NCollapse,
  NCollapseItem,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NList,
  NListItem,
  NPopconfirm,
  NPopover,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns,
  type DataTableRowKey,
  type SelectOption,
} from "naive-ui";
import { useI18n } from "vue-i18n";
import { nodePoolAPI, routingAPI, tunAPI } from "@/api/client";
import type {
  CleanlinessStatus,
  MachineState,
  MachineStateReason,
  NodeEvent,
  NodeExitIPStatus,
  NodeIntelligenceConfidence,
  TunAggregationBenchmarkResult,
  TunAggregationBenchmarkScenarioName,
  TunAggregationBenchmarkStatus,
  NodePoolDashboardResponse,
  NodePoolSummary,
  NodeNetworkType,
  NodeRecord,
  NodeStatus,
  TunAggregationMode,
  TunAggregationPrototypePathState,
  TunAggregationPrototypeStatus,
  TunAggregationRelayStatus,
  TunAggregationSchedulerPolicy,
  TunAggregationSettings,
  TunAggregationStatus,
  TunAggregationStatusCode,
  TunAggregationRuntimePath,
  TunDestinationBinding,
  TunDestinationBindingPreset,
  TunDestinationBindingSelectionMode,
  TunEditableSettings,
  TunRouteMode,
  TunSelectionPolicy,
  TransitionReason,
  TunStatusResponse,
  ValidationConfig,
} from "@/api/types";
import {
  bindingPreferredNodeId,
  bindingPreviewDomains,
  bindingPrimaryTestDomain,
  firstNodeIntelligenceDetail,
  normalizeListInput,
  sortBindingNodes,
  sortPoolNodes,
  sortRemovedNodes,
  summarizeNodeIntelligence,
  type PoolSortMode,
  type RemovedSortMode,
} from "@/utils/nodePool";

const { t, te } = useI18n();
const message = useMessage();

function createDefaultAggregationSettings(): TunAggregationSettings {
  return {
    enabled: false,
    mode: "single_best",
    maxPathsPerSession: 2,
    schedulerPolicy: "weighted_split",
    relayEndpoint: "",
    health: {
      maxSessionLossPct: 5,
      maxPathJitterMs: 120,
      rollbackOnConsecutiveFailures: 3,
    },
  };
}

function normalizeAggregationSettings(
  value?: Partial<TunAggregationSettings>,
): TunAggregationSettings {
  const base = createDefaultAggregationSettings();
  const health = value?.health || {};
  const rawMaxPaths = Number(value?.maxPathsPerSession);
  return {
    ...base,
    ...value,
    maxPathsPerSession:
      Number.isFinite(rawMaxPaths) && rawMaxPaths > 0
        ? Math.min(8, Math.max(1, Math.trunc(rawMaxPaths)))
        : base.maxPathsPerSession,
    relayEndpoint: (value?.relayEndpoint || "").trim(),
    health: {
      ...base.health,
      ...health,
    },
  };
}

function createDefaultAggregationStatus(): TunAggregationStatus {
  const defaults = createDefaultAggregationSettings();
  return {
    enabled: false,
    status: "disabled",
    requestedPath: "stable_single_path",
    effectivePath: "stable_single_path",
    ready: false,
    relayConfigured: false,
    mode: defaults.mode,
    maxPathsPerSession: defaults.maxPathsPerSession,
    schedulerPolicy: defaults.schedulerPolicy,
    relayEndpoint: "",
    reason: "",
    prototype: undefined,
    relay: undefined,
    benchmark: undefined,
  };
}

function createDefaultAggregationPrototype(): TunAggregationPrototypeStatus {
  return {
    ready: false,
    metricSource: "node_pool_probe_history",
    sessionTtlSeconds: 45,
    candidatePathCount: 0,
    selectedPathCount: 0,
    sessionCount: 0,
    paths: [],
    sessions: [],
  };
}

function createDefaultAggregationRelay(): TunAggregationRelayStatus {
  return {
    ready: false,
    contractVersion: "relay_preview_v1",
    endpoint: "",
    sessionCount: 0,
    packetCount: 0,
    deliveredPacketCount: 0,
    duplicateDrops: 0,
    reorderedPackets: 0,
    maxReorderBufferDepth: 0,
    sessions: [],
  };
}

function createDefaultAggregationBenchmarkResult(): TunAggregationBenchmarkResult {
  return {
    startupLatencyMs: 0,
    stallCount: 0,
    goodputKbps: 0,
    lossPct: 0,
    stabilityPct: 0,
  };
}

function createDefaultAggregationBenchmark(): TunAggregationBenchmarkStatus {
  return {
    ready: false,
    packetCount: 0,
    payloadBytes: 0,
    scenarios: [],
  };
}

function normalizeAggregationPrototype(
  value?: Partial<TunAggregationPrototypeStatus>,
): TunAggregationPrototypeStatus {
  const base = createDefaultAggregationPrototype();
  return {
    ...base,
    ...value,
    paths: Array.isArray(value?.paths) ? value!.paths : [],
    sessions: Array.isArray(value?.sessions) ? value!.sessions : [],
  };
}

function normalizeAggregationRelay(
  value?: Partial<TunAggregationRelayStatus>,
): TunAggregationRelayStatus {
  const base = createDefaultAggregationRelay();
  return {
    ...base,
    ...value,
    endpoint: (value?.endpoint || "").trim(),
    sessions: Array.isArray(value?.sessions) ? value!.sessions : [],
  };
}

function normalizeAggregationBenchmark(
  value?: Partial<TunAggregationBenchmarkStatus>,
): TunAggregationBenchmarkStatus {
  const base = createDefaultAggregationBenchmark();
  return {
    ...base,
    ...value,
    scenarios: Array.isArray(value?.scenarios)
      ? value!.scenarios.map((scenario) => ({
          ...scenario,
          baseline: {
            ...createDefaultAggregationBenchmarkResult(),
            ...scenario.baseline,
          },
          aggregated: {
            ...createDefaultAggregationBenchmarkResult(),
            ...scenario.aggregated,
          },
        }))
      : [],
  };
}

function normalizeAggregationStatus(
  value?: Partial<TunAggregationStatus>,
): TunAggregationStatus {
  const base = createDefaultAggregationStatus();
  return {
    ...base,
    ...value,
    relayEndpoint: (value?.relayEndpoint || "").trim(),
    prototype: value?.prototype
      ? normalizeAggregationPrototype(value.prototype)
      : undefined,
    relay: value?.relay ? normalizeAggregationRelay(value.relay) : undefined,
    benchmark: value?.benchmark
      ? normalizeAggregationBenchmark(value.benchmark)
      : undefined,
  };
}

interface DestinationBindingDraft {
  key: string;
  preset: TunDestinationBindingPreset;
  domainsText: string;
  nodeId: string;
  fallbackNodeIds: string[];
  selectionMode: TunDestinationBindingSelectionMode;
}

interface DestinationBindingTestResult {
  domain: string;
  expectedTarget: string;
  actualTarget: string;
  matched: boolean;
}

const loading = ref(false);
const tunUpdating = ref(false);
const installingTunBootstrap = ref(false);
const savingConfig = ref(false);
const savingTunSettings = ref(false);
const bulkRemoving = ref(false);
const bulkRemovingCandidates = ref(false);
const bulkValidating = ref(false);
const bulkRemovingFullFailures = ref(false);
const bulkRestoringRemoved = ref(false);
const bulkPurgingSelectedRemoved = ref(false);
const bulkPurgingRemoved = ref(false);
const purgingRemovedNodeId = ref<string | null>(null);
const bulkPromotingGroup = ref<"staging" | "quarantine" | null>(null);
const activeSortMode = ref<PoolSortMode>("quality");
const stagingSortMode = ref<PoolSortMode>("quality");
const quarantineSortMode = ref<PoolSortMode>("quality");
const candidateSortMode = ref<PoolSortMode>("last_checked_desc");
const removedSortMode = ref<RemovedSortMode>("removed_desc");
const fullFailureThreshold = ref(20);
const isCompact = ref(
  typeof window !== "undefined" ? window.innerWidth < 768 : false,
);
const selectedCandidateIds = ref<string[]>([]);
const selectedStagingIds = ref<string[]>([]);
const selectedQuarantineIds = ref<string[]>([]);
const selectedRemovedIds = ref<string[]>([]);
const destinationBindingDrafts = ref<DestinationBindingDraft[]>([]);
const destinationBindingTestResults = ref<
  Record<string, DestinationBindingTestResult>
>({});
const testingDestinationBindingKey = ref<string | null>(null);
const tunRemoteDnsText = ref("");
const tunProtectDomainsText = ref("");
const tunProtectCidrsText = ref("");
let destinationBindingDraftSeed = 0;

const dashboard = ref<NodePoolDashboardResponse>({
  nodes: [],
  summary: {
    candidateCount: 0,
    stagingCount: 0,
    activeCount: 0,
    quarantineCount: 0,
    removedCount: 0,
    trustedCount: 0,
    suspiciousCount: 0,
    unknownCleanCount: 0,
    activeNodes: 0,
    minActiveNodes: 0,
    healthy: false,
    lastEvaluatedAt: "",
  },
  recentEvents: [],
});

const tunStatus = ref<TunStatusResponse>({
  status: "unknown",
  running: false,
  available: false,
  allowRemote: false,
  useSudo: true,
  helperExists: false,
  elevationReady: false,
  helperCurrent: true,
  binaryCurrent: true,
  privilegeInstallRecommended: false,
  binaryPath: "",
  helperPath: "",
  stateDir: "",
  runtimeConfigPath: "",
  interfaceName: "",
  mtu: 0,
  remoteDns: [],
  configPath: "",
  xrayBinary: "",
  message: "",
  aggregation: createDefaultAggregationStatus(),
});

const configForm = ref<ValidationConfig>({
  minSamples: 10,
  maxFailRate: 0.3,
  maxAvgDelayMs: 1000,
  probeIntervalSec: 60,
  probeUrl: "https://www.gstatic.com/generate_204",
  demoteAfterFails: 5,
  autoRemoveDemoted: false,
  minActiveNodes: 3,
  minBandwidthKbps: 0,
});

const tunSettingsForm = ref<TunEditableSettings>({
  selectionPolicy: "fastest",
  routeMode: "strict_proxy",
  remoteDns: [],
  protectDomains: [],
  protectCidrs: [],
  destinationBindings: [],
  aggregation: createDefaultAggregationSettings(),
});

const nodes = computed(() => dashboard.value.nodes || []);
const summary = computed<NodePoolSummary>(() => dashboard.value.summary);
const recentEvents = computed<NodeEvent[]>(
  () => dashboard.value.recentEvents || [],
);
const intelligenceSummary = computed(() =>
  summarizeNodeIntelligence(nodes.value),
);
const activeNodes = computed(() =>
  sortPoolNodes(
    nodes.value.filter((node) => node.status === "active"),
    activeSortMode.value,
  ),
);
const activeExportableNodes = computed(() =>
  activeNodes.value.filter(
    (node) => typeof node.uri === "string" && node.uri.trim().length > 0,
  ),
);
const stagingNodes = computed(() =>
  sortPoolNodes(
    nodes.value.filter((node) => node.status === "staging"),
    stagingSortMode.value,
  ),
);
const quarantineNodes = computed(() =>
  sortPoolNodes(
    nodes.value.filter((node) => node.status === "quarantine"),
    quarantineSortMode.value,
  ),
);
const candidateNodes = computed(() =>
  sortPoolNodes(
    nodes.value.filter((node) => node.status === "candidate"),
    candidateSortMode.value,
  ),
);
const removedNodes = computed(() =>
  sortRemovedNodes(
    nodes.value.filter((node) => node.status === "removed"),
    removedSortMode.value,
  ),
);
const removedFullFailureNodes = computed(() =>
  removedNodes.value.filter(
    (node) => node.totalPings > 0 && node.failedPings === node.totalPings,
  ),
);
const fullFailureRemovalCandidates = computed(() =>
  nodes.value.filter(
    (node) =>
      node.status !== "removed" &&
      node.totalPings >= fullFailureThreshold.value &&
      node.totalPings > 0 &&
      node.failedPings === node.totalPings,
  ),
);

const machineState = computed<MachineState>(
  () => tunStatus.value.machineState || "clean",
);
const tunBootstrapNeeded = computed(() =>
  Boolean(tunStatus.value.privilegeInstallRecommended),
);
const tunRepairRecommended = computed(
  () =>
    tunStatus.value.helperCurrent === false ||
    tunStatus.value.binaryCurrent === false,
);
const machineStateLabel = computed(() =>
  translateCode("nodePool.machineStateLabel", machineState.value),
);
const machineReasonLabel = computed(() =>
  translateCode(
    "nodePool.machineReason",
    tunStatus.value.lastStateReason || "startup_default_clean",
  ),
);
const formattedMachineChangedAt = computed(() =>
  formatDateTime(tunStatus.value.lastStateChangedAt),
);
const formattedSummaryEvaluatedAt = computed(() =>
  formatDateTime(summary.value.lastEvaluatedAt),
);

const machineStateTagType = computed(() => {
  switch (machineState.value) {
    case "proxied":
      return "success";
    case "degraded":
      return "error";
    case "recovering":
      return "warning";
    default:
      return "default";
  }
});

const machineBannerType = computed(() => {
  if (machineState.value === "degraded") return "error";
  if (!summary.value.healthy || tunStatus.value.status === "blocked")
    return "warning";
  if (machineState.value === "proxied") return "success";
  return "info";
});

const machineBannerTitle = computed(() => {
  if (machineState.value === "degraded")
    return t("nodePool.banner.degradedTitle");
  if (!summary.value.healthy || tunStatus.value.status === "blocked")
    return t("nodePool.banner.poolWarningTitle");
  if (machineState.value === "proxied")
    return t("nodePool.banner.proxiedTitle");
  return t("nodePool.banner.cleanTitle");
});

const machineBannerBody = computed(() => {
  if (machineState.value === "degraded") {
    return `${machineReasonLabel.value}. ${tunStatus.value.message || ""}`.trim();
  }
  if (!summary.value.healthy || tunStatus.value.status === "blocked") {
    return `${t("nodePool.activePoolCount", { active: summary.value.activeNodes, minimum: summary.value.minActiveNodes })}. ${machineReasonLabel.value}`;
  }
  return tunStatus.value.message || machineReasonLabel.value;
});

const routeModeOptions = computed(() => [
  {
    label: t("nodePool.routeModeOptions.strict_proxy"),
    value: "strict_proxy" as TunRouteMode,
  },
  {
    label: t("nodePool.routeModeOptions.auto_tested"),
    value: "auto_tested" as TunRouteMode,
  },
]);

const selectionPolicyOptions = computed(() => [
  {
    label: t("nodePool.selectionPolicyOptions.fastest"),
    value: "fastest" as TunSelectionPolicy,
  },
  {
    label: t("nodePool.selectionPolicyOptions.lowest_latency"),
    value: "lowest_latency" as TunSelectionPolicy,
  },
  {
    label: t("nodePool.selectionPolicyOptions.lowest_fail_rate"),
    value: "lowest_fail_rate" as TunSelectionPolicy,
  },
]);

const aggregationModeOptions = computed(() => [
  {
    label: t("nodePool.aggregationModeOptions.single_best"),
    value: "single_best" as TunAggregationMode,
  },
  {
    label: t("nodePool.aggregationModeOptions.redundant_2"),
    value: "redundant_2" as TunAggregationMode,
  },
  {
    label: t("nodePool.aggregationModeOptions.weighted_split"),
    value: "weighted_split" as TunAggregationMode,
  },
]);

const aggregationSchedulerPolicyOptions = computed(() => [
  {
    label: t("nodePool.aggregationSchedulerPolicyOptions.single_best"),
    value: "single_best" as TunAggregationSchedulerPolicy,
  },
  {
    label: t("nodePool.aggregationSchedulerPolicyOptions.redundant_2"),
    value: "redundant_2" as TunAggregationSchedulerPolicy,
  },
  {
    label: t("nodePool.aggregationSchedulerPolicyOptions.weighted_split"),
    value: "weighted_split" as TunAggregationSchedulerPolicy,
  },
]);

const destinationBindingPresetOptions = computed(() => [
  {
    label: t("nodePool.destinationBindingPresetOptions.openai"),
    value: "openai" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.chatgpt"),
    value: "chatgpt" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.claude"),
    value: "claude" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.gemini"),
    value: "gemini" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.github_copilot"),
    value: "github_copilot" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.openrouter"),
    value: "openrouter" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.cursor"),
    value: "cursor" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.qwen"),
    value: "qwen" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.perplexity"),
    value: "perplexity" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.deepseek"),
    value: "deepseek" as TunDestinationBindingPreset,
  },
  {
    label: t("nodePool.destinationBindingPresetOptions.custom"),
    value: "custom" as TunDestinationBindingPreset,
  },
]);

const destinationBindingSelectionModeOptions = computed(() => [
  {
    label: t("nodePool.destinationBindingSelectionModeOptions.primary_only"),
    value: "primary_only" as TunDestinationBindingSelectionMode,
  },
  {
    label: t("nodePool.destinationBindingSelectionModeOptions.failover_ordered"),
    value: "failover_ordered" as TunDestinationBindingSelectionMode,
  },
  {
    label: t("nodePool.destinationBindingSelectionModeOptions.failover_fastest"),
    value: "failover_fastest" as TunDestinationBindingSelectionMode,
  },
]);

const removedSortOptions = computed(() => [
  {
    label: t("nodePool.removedSortOptions.removed_desc"),
    value: "removed_desc" as const,
  },
  {
    label: t("nodePool.removedSortOptions.removed_asc"),
    value: "removed_asc" as const,
  },
  {
    label: t("nodePool.removedSortOptions.fail_rate_asc"),
    value: "fail_rate_asc" as const,
  },
  {
    label: t("nodePool.removedSortOptions.fail_rate_desc"),
    value: "fail_rate_desc" as const,
  },
  {
    label: t("nodePool.removedSortOptions.avg_delay_asc"),
    value: "avg_delay_asc" as const,
  },
  {
    label: t("nodePool.removedSortOptions.avg_delay_desc"),
    value: "avg_delay_desc" as const,
  },
]);

const poolSortOptions = computed(() => [
  { label: t("nodePool.poolSortOptions.quality"), value: "quality" as const },
  {
    label: t("nodePool.poolSortOptions.cleanliness_desc"),
    value: "cleanliness_desc" as const,
  },
  {
    label: t("nodePool.poolSortOptions.last_checked_asc"),
    value: "last_checked_asc" as const,
  },
  {
    label: t("nodePool.poolSortOptions.fail_rate_asc"),
    value: "fail_rate_asc" as const,
  },
  {
    label: t("nodePool.poolSortOptions.fail_rate_desc"),
    value: "fail_rate_desc" as const,
  },
  {
    label: t("nodePool.poolSortOptions.avg_delay_asc"),
    value: "avg_delay_asc" as const,
  },
  {
    label: t("nodePool.poolSortOptions.avg_delay_desc"),
    value: "avg_delay_desc" as const,
  },
]);

const poolLifecycleSteps = computed(() => [
  {
    status: "candidate" as NodeStatus,
    description: t("nodePool.lifecycleGuide.candidate"),
  },
  {
    status: "staging" as NodeStatus,
    description: t("nodePool.lifecycleGuide.staging"),
  },
  {
    status: "active" as NodeStatus,
    description: t("nodePool.lifecycleGuide.active"),
  },
  {
    status: "quarantine" as NodeStatus,
    description: t("nodePool.lifecycleGuide.quarantine"),
  },
  {
    status: "removed" as NodeStatus,
    description: t("nodePool.lifecycleGuide.removed"),
  },
]);

const validationConfigGuideItems = computed(() => [
  {
    key: "minActiveNodes",
    label: t("nodePool.minActiveNodes"),
    description: t("nodePool.validationGuideMinActiveNodes"),
  },
  {
    key: "minSamples",
    label: t("nodePool.minSamples"),
    description: t("nodePool.validationGuideMinSamples"),
  },
  {
    key: "maxFailRate",
    label: t("nodePool.maxFailRate"),
    description: t("nodePool.validationGuideMaxFailRate"),
  },
  {
    key: "maxAvgDelay",
    label: t("nodePool.maxAvgDelay"),
    description: t("nodePool.validationGuideMaxAvgDelay"),
  },
  {
    key: "demoteAfterFails",
    label: t("nodePool.demoteAfterFails"),
    description: t("nodePool.validationGuideDemoteAfterFails"),
  },
  {
    key: "probeInterval",
    label: t("nodePool.probeInterval"),
    description: t("nodePool.validationGuideProbeInterval"),
  },
  {
    key: "probeUrl",
    label: t("nodePool.probeUrl"),
    description: t("nodePool.validationGuideProbeUrl"),
  },
  {
    key: "minBandwidthKbps",
    label: t("nodePool.minBandwidthKbps"),
    description: t("nodePool.validationGuideMinBandwidthKbps"),
  },
  {
    key: "autoRemoveDemoted",
    label: t("nodePool.autoRemoveDemoted"),
    description: t("nodePool.validationGuideAutoRemoveDemoted"),
  },
]);

const validationPoolHint = computed(() => {
  const hints: string[] = [];

  if (!stagingNodes.value.length && configForm.value.minSamples <= 1) {
    hints.push(
      t("nodePool.validationPoolZeroHintFast", {
        minSamples: configForm.value.minSamples,
      }),
    );
  }
  if (candidateNodes.value.length) {
    hints.push(
      t("nodePool.validationPoolZeroHintCandidate", {
        count: candidateNodes.value.length,
      }),
    );
  }

  return hints.join(" ");
});

function translateCode(prefix: string, code: string): string {
  const key = `${prefix}.${code}`;
  return te(key) ? t(key) : code;
}

function statusLabel(status: NodeStatus) {
  return translateCode("nodePool.status", status);
}

function reasonLabel(reason: TransitionReason | MachineStateReason) {
  return translateCode("nodePool.reason", reason);
}

function showSubscriptionMissingTag(node: NodeRecord) {
  return (
    !!node.subscriptionMissing && node.statusReason !== "subscription_missing"
  );
}

function nodeReasonLabel(node: NodeRecord) {
  if (
    node.subscriptionMissing &&
    node.statusReason === "subscription_missing"
  ) {
    return reasonLabel("subscription_missing");
  }
  return reasonLabel(node.statusReason);
}

function cleanlinessLabel(cleanliness: CleanlinessStatus) {
  return translateCode("nodePool.cleanliness", cleanliness);
}

function networkTypeLabel(networkType: NodeNetworkType) {
  return translateCode("nodePool.networkType", networkType);
}

function confidenceLabel(confidence: NodeIntelligenceConfidence) {
  return translateCode("nodePool.confidence", confidence);
}

function exitIpStatusLabel(status: NodeExitIPStatus) {
  return translateCode("nodePool.exitIpStatus", status);
}

function intelligenceReasonLabel(reason?: string) {
  return translateCode(
    "nodePool.intelligenceReason",
    reason || "insufficient_signal",
  );
}

function aggregationStatusLabel(status?: TunAggregationStatusCode) {
  return translateCode("nodePool.aggregationStatus", status || "disabled");
}

function aggregationPathLabel(path?: TunAggregationRuntimePath) {
  return translateCode(
    "nodePool.aggregationPath",
    path || "stable_single_path",
  );
}

function aggregationSchedulerPolicyLabel(
  policy?: TunAggregationSchedulerPolicy,
) {
  return translateCode(
    "nodePool.aggregationSchedulerPolicyOptions",
    policy || "weighted_split",
  );
}

function aggregationPrototypeMetricSourceLabel(source?: string) {
  return translateCode(
    "nodePool.aggregationPrototypeMetricSource",
    source || "node_pool_probe_history",
  );
}

function aggregationPrototypePathStateLabel(
  state?: TunAggregationPrototypePathState,
) {
  return translateCode(
    "nodePool.aggregationPrototypePathState",
    state || "excluded",
  );
}

function aggregationBenchmarkScenarioLabel(
  name?: TunAggregationBenchmarkScenarioName,
) {
  return translateCode(
    "nodePool.aggregationBenchmarkScenario",
    name || "clean_paths",
  );
}

function aggregationPrototypePathStateTagType(
  state?: TunAggregationPrototypePathState,
) {
  switch (state) {
    case "selected":
      return "success";
    case "standby":
      return "info";
    default:
      return "default";
  }
}

function statusTagType(status: NodeStatus) {
  switch (status) {
    case "active":
      return "success";
    case "quarantine":
      return "error";
    case "candidate":
      return "warning";
    case "removed":
      return "default";
    default:
      return "info";
  }
}

function cleanlinessTagType(cleanliness: CleanlinessStatus) {
  switch (cleanliness) {
    case "trusted":
      return "success";
    case "suspicious":
      return "error";
    default:
      return "warning";
  }
}

function networkTypeTagType(networkType: NodeNetworkType) {
  switch (networkType) {
    case "residential_likely":
      return "success";
    case "isp_likely":
      return "warning";
    case "datacenter_likely":
      return "warning";
    default:
      return "default";
  }
}

function failRateLabel(node: NodeRecord) {
  if (!node.totalPings) return t("nodePool.failRateUnknown");
  return `${((node.failedPings / node.totalPings) * 100).toFixed(1)}%`;
}

function delayLabel(node: NodeRecord) {
  return node.avgDelayMs > 0
    ? `${node.avgDelayMs}ms`
    : t("nodePool.delayUnknown");
}

function aggregationLatencyLabel(latencyMs?: number) {
  return latencyMs && latencyMs > 0 ? `${latencyMs}ms` : "-";
}

function formatDurationMs(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value) || value < 0) return "-";
  return `${Math.round(value)}ms`;
}

function formatLossPct(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value)) return "-";
  return `${value.toFixed(1)}%`;
}

function formatAggregationGoodput(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value)) return "-";
  return `${value.toFixed(1)} kbps`;
}

function formatSignedMetric(
  value: number | undefined,
  suffix: string,
  digits = 1,
) {
  if (typeof value !== "number" || Number.isNaN(value)) return "-";
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(digits)}${suffix}`;
}

function formatSignedInt(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value)) return "-";
  const rounded = Math.round(value);
  const sign = rounded >= 0 ? "+" : "";
  return `${sign}${rounded}`;
}

function formatAggregationScore(value?: number) {
  if (typeof value !== "number" || Number.isNaN(value)) return "-";
  return value.toFixed(1);
}

function formatPathIds(values?: string[]) {
  return Array.isArray(values) && values.length ? values.join(", ") : "-";
}

function nodeExitIpHeadline(node: NodeRecord) {
  if (node.exitIpStatus === "available" && node.exitIp) {
    return `${t("nodePool.exitIpLabel")}: ${node.exitIp}`;
  }
  return `${t("nodePool.exitIpLabel")}: ${exitIpStatusLabel(node.exitIpStatus || "unknown")}`;
}

function nodeExitIpMeta(node: NodeRecord) {
  if (node.exitIpStatus === "available") {
    const parts = [
      node.exitIpSource,
      formatDateTime(node.exitIpCheckedAt),
    ].filter(Boolean);
    return parts.join(" · ");
  }
  return node.exitIpError || "";
}

function nodeIntelligenceDetail(node: NodeRecord) {
  return firstNodeIntelligenceDetail(node);
}

function nodeIntelligenceDetailLine(node: NodeRecord) {
  const detail = nodeIntelligenceDetail(node);
  return detail ? `${t("nodePool.intelligenceDetailLabel")}: ${detail}` : "";
}

type VerdictKind = "cleanliness" | "network";

function nodeVerdictLine(
  kind: VerdictKind,
  reason: string | undefined,
  confidence: NodeIntelligenceConfidence,
) {
  const labelKey =
    kind === "cleanliness" ? "cleanlinessReasonLabel" : "networkReasonLabel";
  return `${t(`nodePool.${labelKey}`)}: ${intelligenceReasonLabel(reason)} (${confidenceLabel(confidence || "unknown")})`;
}

function nodeVerdictTitle(kind: VerdictKind, node: NodeRecord) {
  const detail =
    kind === "cleanliness" ? node.cleanlinessDetail : node.networkTypeDetail;
  const extra = kind === "cleanliness" ? node.intelligenceError : "";
  return [detail, extra].filter(Boolean).join("\n");
}

function nodeIntelligencePopoverLines(node: NodeRecord) {
  return [
    nodeExitIpHeadline(node),
    nodeExitIpMeta(node),
    nodeVerdictLine(
      "cleanliness",
      node.cleanlinessReason,
      node.cleanlinessConfidence,
    ),
    nodeVerdictLine(
      "network",
      node.networkTypeReason,
      node.networkTypeConfidence,
    ),
    nodeIntelligenceDetailLine(node),
  ].filter(Boolean);
}

function renderNodeIntelligenceTags(row: NodeRecord) {
  return h(
    NSpace,
    { size: 6, wrap: true, class: "node-intelligence-tag-group" },
    {
      default: () => [
        h(
          NTag,
          { size: "small", type: cleanlinessTagType(row.cleanliness) },
          { default: () => cleanlinessLabel(row.cleanliness) },
        ),
        h(
          NTag,
          { size: "small", type: networkTypeTagType(row.networkType) },
          { default: () => networkTypeLabel(row.networkType) },
        ),
      ],
    },
  );
}

function renderNodeIntelligence(row: NodeRecord) {
  const lines = nodeIntelligencePopoverLines(row);
  return h(
    NPopover,
    { trigger: "hover", placement: "top-start" },
    {
      trigger: () =>
        h(
          "div",
          { class: "node-intelligence-cell node-intelligence-trigger" },
          [renderNodeIntelligenceTags(row)],
        ),
      default: () =>
        h("div", { class: "node-intelligence-popover" }, [
          h(
            "div",
            { class: "node-intelligence-popover-title" },
            t("nodePool.intelligenceLabel"),
          ),
          ...lines.map((line, index) =>
            h(
              "div",
              {
                key: `${row.id}-${index}`,
                class: "node-intelligence-popover-line",
              },
              line,
            ),
          ),
        ]),
    },
  );
}

function formatDateTime(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString();
}

function syncViewport() {
  isCompact.value = window.innerWidth < 768;
}

function rowKey(row: NodeRecord) {
  return row.id;
}

function syncSelections() {
  const candidateSet = new Set(candidateNodes.value.map((node) => node.id));
  const stagingSet = new Set(stagingNodes.value.map((node) => node.id));
  const quarantineSet = new Set(quarantineNodes.value.map((node) => node.id));
  const removedSet = new Set(removedNodes.value.map((node) => node.id));
  selectedCandidateIds.value = selectedCandidateIds.value.filter((id) =>
    candidateSet.has(id),
  );
  selectedStagingIds.value = selectedStagingIds.value.filter((id) =>
    stagingSet.has(id),
  );
  selectedQuarantineIds.value = selectedQuarantineIds.value.filter((id) =>
    quarantineSet.has(id),
  );
  selectedRemovedIds.value = selectedRemovedIds.value.filter((id) =>
    removedSet.has(id),
  );
}

function clearSelection(
  group: "candidate" | "staging" | "quarantine" | "removed",
) {
  if (group === "candidate") {
    selectedCandidateIds.value = [];
    return;
  }
  if (group === "staging") {
    selectedStagingIds.value = [];
    return;
  }
  if (group === "quarantine") {
    selectedQuarantineIds.value = [];
    return;
  }
  selectedRemovedIds.value = [];
}

function handleCheckedRowKeysUpdate(
  group: "candidate" | "staging" | "quarantine" | "removed",
  keys: DataTableRowKey[],
) {
  const ids = keys.map((key) => String(key));
  if (group === "candidate") {
    selectedCandidateIds.value = ids;
    return;
  }
  if (group === "staging") {
    selectedStagingIds.value = ids;
    return;
  }
  if (group === "quarantine") {
    selectedQuarantineIds.value = ids;
    return;
  }
  selectedRemovedIds.value = ids;
}

function isSelected(
  group: "candidate" | "staging" | "quarantine" | "removed",
  id: string,
) {
  if (group === "candidate") {
    return selectedCandidateIds.value.includes(id);
  }
  if (group === "staging") {
    return selectedStagingIds.value.includes(id);
  }
  if (group === "quarantine") {
    return selectedQuarantineIds.value.includes(id);
  }
  return selectedRemovedIds.value.includes(id);
}

function setCardSelection(
  group: "candidate" | "staging" | "quarantine" | "removed",
  id: string,
  checked: boolean,
) {
  const source =
    group === "candidate"
      ? selectedCandidateIds.value
      : group === "staging"
        ? selectedStagingIds.value
        : group === "quarantine"
          ? selectedQuarantineIds.value
          : selectedRemovedIds.value;
  const next = new Set(source);
  if (checked) {
    next.add(id);
  } else {
    next.delete(id);
  }
  if (group === "candidate") {
    selectedCandidateIds.value = Array.from(next);
    return;
  }
  if (group === "staging") {
    selectedStagingIds.value = Array.from(next);
    return;
  }
  if (group === "quarantine") {
    selectedQuarantineIds.value = Array.from(next);
    return;
  }
  selectedRemovedIds.value = Array.from(next);
}

function createColumns(
  group: "candidate" | "staging" | "active" | "quarantine" | "removed",
): DataTableColumns<NodeRecord> {
  const columns: DataTableColumns<NodeRecord> = [
    ...(group === "candidate" ||
    group === "staging" ||
    group === "quarantine" ||
    group === "removed"
      ? [
          {
            type: "selection",
          } as const,
        ]
      : []),
    {
      title: () => t("subscriptions.remark"),
      key: "remark",
      width: 160,
      ellipsis: { tooltip: true },
    },
    {
      title: () => t("common.status"),
      key: "status",
      width: 120,
      render: (row) =>
        h(
          NTag,
          { size: "small", type: statusTagType(row.status) },
          { default: () => statusLabel(row.status) },
        ),
    },
    {
      title: () => t("nodePool.intelligenceLabel"),
      key: "intelligence",
      width: 340,
      render: (row) => renderNodeIntelligence(row),
    },
    {
      title: () => t("nodePool.address"),
      key: "address",
      width: 240,
      render: (row) => {
        if (!showSubscriptionMissingTag(row)) {
          return `${row.address}:${row.port}`;
        }
        return h(
          NSpace,
          { size: 6, align: "center" },
          {
            default: () => [
              h("span", `${row.address}:${row.port}`),
              h(
                NTag,
                { size: "small", type: "warning" },
                { default: () => t("nodePool.subscriptionMissingFlag") },
              ),
            ],
          },
        );
      },
    },
    {
      title: () => t("nodePool.reasonLabel"),
      key: "statusReason",
      width: 200,
      ellipsis: { tooltip: true },
      render: (row) => nodeReasonLabel(row),
    },
    {
      title: () => t("nodePool.avgDelay"),
      key: "avgDelayMs",
      width: 110,
      render: (row) => delayLabel(row),
    },
    {
      title: () => t("nodePool.failRate"),
      key: "failRate",
      width: 110,
      render: (row) => failRateLabel(row),
    },
    {
      title: () =>
        t(
          group === "removed" ? "nodePool.removedAt" : "nodePool.lastCheckedAt",
        ),
      key: "lastCheckedAt",
      width: 180,
      render: (row) =>
        formatDateTime(
          group === "removed"
            ? row.statusUpdatedAt || row.lastEventAt || row.addedAt
            : row.lastCheckedAt || row.statusUpdatedAt || row.addedAt,
        ) || "-",
    },
  ];

  columns.push({
    title: () => t("common.actions"),
    key: "actions",
    width: group === "removed" ? 260 : 220,
    render: (row) => {
      const actions: any[] = [];

      if (group === "removed") {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handleRestore(row.id) },
            {
              trigger: () =>
                h(
                  NButton,
                  { size: "small", type: "primary" },
                  { default: () => t("nodePool.restoreNode") },
                ),
              default: () => t("nodePool.restoreConfirm"),
            },
          ),
        );

        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handlePurgeRemoved(row.id) },
            {
              trigger: () =>
                h(
                  NButton,
                  {
                    size: "small",
                    type: "error",
                    secondary: true,
                    loading: purgingRemovedNodeId.value === row.id,
                  },
                  { default: () => t("nodePool.purgeRemovedNode") },
                ),
              default: () => t("nodePool.purgeRemovedConfirm"),
            },
          ),
        );

        return h(NSpace, { size: "small" }, { default: () => actions });
      }

      if (group === "candidate") {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handleValidate(row.id) },
            {
              trigger: () =>
                h(
                  NButton,
                  { size: "small", type: "primary" },
                  { default: () => t("nodePool.validate") },
                ),
              default: () => t("nodePool.validateConfirm"),
            },
          ),
        );
      }

      if (group === "staging" || group === "quarantine") {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handlePromote(row.id) },
            {
              trigger: () =>
                h(
                  NButton,
                  { size: "small", type: "success" },
                  { default: () => t("nodePool.promote") },
                ),
              default: () => t("nodePool.promoteConfirm"),
            },
          ),
        );
      }

      if (group === "active") {
        actions.push(
          h(
            NPopconfirm,
            { onPositiveClick: () => handleQuarantine(row.id) },
            {
              trigger: () =>
                h(
                  NButton,
                  { size: "small", type: "warning" },
                  { default: () => t("nodePool.quarantine") },
                ),
              default: () => t("nodePool.quarantineConfirm"),
            },
          ),
        );
      }

      actions.push(
        h(
          NPopconfirm,
          { onPositiveClick: () => handleRemove(row.id) },
          {
            trigger: () =>
              h(
                NButton,
                { size: "small", type: "error" },
                { default: () => t("nodePool.remove") },
              ),
            default: () => t("nodePool.removeConfirm"),
          },
        ),
      );

      return h(NSpace, { size: "small" }, { default: () => actions });
    },
  });

  return columns;
}

const activeColumns = computed(() => createColumns("active"));
const stagingColumns = computed(() => createColumns("staging"));
const quarantineColumns = computed(() => createColumns("quarantine"));
const candidateColumns = computed(() => createColumns("candidate"));
const removedColumns = computed(() => createColumns("removed"));

async function fetchDashboard() {
  const data = await nodePoolAPI.list();
  dashboard.value = data;
  syncSelections();
}

async function fetchConfig() {
  const data = await nodePoolAPI.getConfig();
  configForm.value = { ...configForm.value, ...data };
}

function syncTunSettingsTextAreas() {
  tunRemoteDnsText.value = (tunSettingsForm.value.remoteDns || []).join("\n");
  tunProtectDomainsText.value = (
    tunSettingsForm.value.protectDomains || []
  ).join("\n");
  tunProtectCidrsText.value = (tunSettingsForm.value.protectCidrs || []).join(
    "\n",
  );
}

function nextDestinationBindingDraftKey() {
  destinationBindingDraftSeed += 1;
  return `binding-${destinationBindingDraftSeed}`;
}

function bindingToDraft(
  binding: TunDestinationBinding,
): DestinationBindingDraft {
  return {
    key: nextDestinationBindingDraftKey(),
    preset: binding.preset || "openai",
    domainsText: Array.isArray(binding.domains)
      ? binding.domains.join("\n")
      : "",
    nodeId: binding.nodeId || "",
    selectionMode: binding.selectionMode || "primary_only",
    fallbackNodeIds: Array.isArray(binding.fallbackNodeIds)
      ? binding.fallbackNodeIds
      : [],
  };
}

function draftToBinding(
  binding: DestinationBindingDraft,
): TunDestinationBinding {
  return {
    preset: binding.preset,
    domains:
      binding.preset === "custom"
        ? normalizeListInput(binding.domainsText)
        : [],
    nodeId: binding.nodeId,
    selectionMode: binding.selectionMode,
    fallbackNodeIds: binding.fallbackNodeIds,
  };
}

function syncDestinationBindingDrafts(bindings: TunDestinationBinding[]) {
  destinationBindingDrafts.value = Array.isArray(bindings)
    ? bindings.map(bindingToDraft)
    : [];
  destinationBindingTestResults.value = {};
}

function addDestinationBindingDraft() {
  destinationBindingDrafts.value.push({
    key: nextDestinationBindingDraftKey(),
    preset: "openai",
    domainsText: "",
    nodeId: bindingPreferredNodeId(activeNodes.value),
    selectionMode: "primary_only",
    fallbackNodeIds: [],
  });
}

function removeDestinationBindingDraft(key: string) {
  destinationBindingDrafts.value = destinationBindingDrafts.value.filter(
    (binding) => binding.key !== key,
  );
  const nextResults = { ...destinationBindingTestResults.value };
  delete nextResults[key];
  destinationBindingTestResults.value = nextResults;
}

function bindingResolvedDomains(binding: DestinationBindingDraft) {
  return bindingPreviewDomains(draftToBinding(binding));
}

function bindingNodeOptionLabel(node: NodeRecord) {
  const title = node.remark || `${node.address}:${node.port}`;
  return [
    title,
    cleanlinessLabel(node.cleanliness),
    networkTypeLabel(node.networkType),
  ]
    .filter(Boolean)
    .join(" · ");
}

function bindingNodeOptions(binding: DestinationBindingDraft) {
  const options: SelectOption[] = sortBindingNodes(activeNodes.value).map(
    (node) => ({
      label: bindingNodeOptionLabel(node),
      value: node.id,
    }),
  );
  if (
    binding.nodeId &&
    !activeNodes.value.some((node) => node.id === binding.nodeId)
  ) {
    options.unshift({
      label: t("nodePool.destinationBindingInactiveTarget", {
        nodeId: binding.nodeId,
      }),
      value: binding.nodeId,
      disabled: true,
    });
  }
  return options;
}

function bindingFallbackNodeOptions(binding: DestinationBindingDraft) {
  const selected = new Set([binding.nodeId, ...binding.fallbackNodeIds]);
  const options: SelectOption[] = sortBindingNodes(activeNodes.value)
    .filter((node) => node.id !== binding.nodeId)
    .map((node) => ({
      label: bindingNodeOptionLabel(node),
      value: node.id,
    }))
    .filter((option) => !selected.has(String(option.value)));
  return options;
}

function bindingTargetMissing(binding: DestinationBindingDraft) {
  return (
    !!binding.nodeId &&
    !activeNodes.value.some((node) => node.id === binding.nodeId)
  );
}

function bindingTestAlertType(key: string) {
  const result = destinationBindingTestResults.value[key];
  if (!result) return "info";
  return result.matched ? "success" : "warning";
}

function bindingTestSummary(key: string) {
  const result = destinationBindingTestResults.value[key];
  if (!result) return "";
  if (result.matched) {
    return t("nodePool.destinationBindingTestMatched", {
      domain: result.domain,
      target: result.actualTarget,
    });
  }
  return t("nodePool.destinationBindingTestMismatch", {
    domain: result.domain,
    expected: result.expectedTarget,
    actual: result.actualTarget || "-",
  });
}

async function fetchTunSettings() {
  const data = await tunAPI.getSettings();
  tunSettingsForm.value = {
    selectionPolicy: data.selectionPolicy || "fastest",
    routeMode: data.routeMode || "strict_proxy",
    remoteDns: Array.isArray(data.remoteDns) ? data.remoteDns : [],
    protectDomains: Array.isArray(data.protectDomains)
      ? data.protectDomains
      : [],
    protectCidrs: Array.isArray(data.protectCidrs) ? data.protectCidrs : [],
    destinationBindings: Array.isArray(data.destinationBindings)
      ? data.destinationBindings
      : [],
    aggregation: normalizeAggregationSettings(data.aggregation),
  };
  syncTunSettingsTextAreas();
  syncDestinationBindingDrafts(tunSettingsForm.value.destinationBindings);
}

async function fetchTunStatus() {
  applyTunStatus(await tunAPI.status());
}

function applyTunStatus(status: TunStatusResponse) {
  tunStatus.value = {
    ...tunStatus.value,
    ...status,
    diagnostics: Array.isArray(status?.diagnostics) ? status.diagnostics : [],
    aggregation: normalizeAggregationStatus(status?.aggregation),
  };
}

const aggregationStatusAlertType = computed(() => {
  const status = tunStatus.value.aggregation?.status || "disabled";
  if (status === "fallback_stable") return "warning";
  if (status === "requested") return "info";
  return "info";
});

const aggregationStatusSummary = computed(() => {
  const aggregation = normalizeAggregationStatus(tunStatus.value.aggregation);
  return t("nodePool.aggregationStatusSummary", {
    status: aggregationStatusLabel(aggregation.status),
    requested: aggregationPathLabel(aggregation.requestedPath),
    effective: aggregationPathLabel(aggregation.effectivePath),
    reason: aggregation.reason || t("nodePool.aggregationDisabledSummary"),
  });
});

const aggregationPrototype = computed(() => {
  const prototype = tunStatus.value.aggregation?.prototype;
  return prototype ? normalizeAggregationPrototype(prototype) : null;
});

const aggregationRelay = computed(() => {
  const relay = tunStatus.value.aggregation?.relay;
  return relay ? normalizeAggregationRelay(relay) : null;
});

const aggregationBenchmark = computed(() => {
  const benchmark = tunStatus.value.aggregation?.benchmark;
  return benchmark ? normalizeAggregationBenchmark(benchmark) : null;
});

async function refreshAll() {
  loading.value = true;
  try {
    await Promise.all([
      fetchDashboard(),
      fetchConfig(),
      fetchTunSettings(),
      fetchTunStatus(),
    ]);
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    loading.value = false;
  }
}

function createExportTimestamp() {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, "0");
  return [
    now.getFullYear(),
    pad(now.getMonth() + 1),
    pad(now.getDate()),
    "-",
    pad(now.getHours()),
    pad(now.getMinutes()),
    pad(now.getSeconds()),
  ].join("");
}

function downloadTextFile(content: string, filename: string) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  window.URL.revokeObjectURL(url);
}

function handleExportActiveNodes() {
  const links = activeExportableNodes.value
    .map((node) => node.uri.trim())
    .filter((uri) => uri.length > 0);

  if (!links.length) {
    message.warning(t("nodePool.exportActivePoolEmpty"));
    return;
  }

  downloadTextFile(
    `${links.join("\n")}\n`,
    `xray-active-pool-${createExportTimestamp()}.txt`,
  );
  message.success(
    t("nodePool.exportActivePoolSuccess", { count: links.length }),
  );
}

async function handleEnableTransparent() {
  tunUpdating.value = true;
  try {
    applyTunStatus(await tunAPI.start());
    message.success(tunStatus.value.message || t("common.success"));
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err);
    }
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    tunUpdating.value = false;
    await refreshAll();
  }
}

async function handleRestoreClean() {
  tunUpdating.value = true;
  try {
    applyTunStatus(await tunAPI.restoreClean());
    message.success(tunStatus.value.message || t("common.success"));
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err);
    }
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    tunUpdating.value = false;
    await refreshAll();
  }
}

async function handleInstallTunBootstrap() {
  installingTunBootstrap.value = true;
  try {
    applyTunStatus(await tunAPI.installPrivilege());
    message.success(tunStatus.value.message || t("common.success"));
  } catch (err: any) {
    if (err?.status) {
      applyTunStatus(err);
    }
    message.error(err?.message || err?.error || t("common.error"));
    await fetchTunStatus();
  } finally {
    installingTunBootstrap.value = false;
  }
}

async function handlePromote(id: string) {
  try {
    await nodePoolAPI.promote(id);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  }
}

async function handleValidate(id: string) {
  try {
    await nodePoolAPI.validate(id);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  }
}

async function handleQuarantine(id: string) {
  try {
    await nodePoolAPI.quarantine(id);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  }
}

async function handleRemove(id: string) {
  try {
    await nodePoolAPI.remove(id);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  }
}

async function handleRestore(id: string) {
  try {
    await nodePoolAPI.restore(id);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  }
}

async function handleBulkValidateCandidate() {
  if (!selectedCandidateIds.value.length) return;

  bulkValidating.value = true;
  try {
    await nodePoolAPI.bulkValidate({ ids: selectedCandidateIds.value });
    clearSelection("candidate");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkValidating.value = false;
  }
}

async function handleBulkRemoveCandidate() {
  const ids = [...selectedCandidateIds.value];
  if (!ids.length) return;

  bulkRemovingCandidates.value = true;
  try {
    await nodePoolAPI.bulkRemove({ ids });
    clearSelection("candidate");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkRemovingCandidates.value = false;
  }
}

async function handleBulkRestoreRemoved() {
  if (!selectedRemovedIds.value.length) return;

  bulkRestoringRemoved.value = true;
  try {
    await nodePoolAPI.bulkRestore({ ids: selectedRemovedIds.value });
    clearSelection("removed");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkRestoringRemoved.value = false;
  }
}

async function handleBulkPurgeSelectedRemoved() {
  const ids = [...selectedRemovedIds.value];
  if (!ids.length) return;

  bulkPurgingSelectedRemoved.value = true;
  try {
    await nodePoolAPI.bulkPurgeRemoved({ ids });
    clearSelection("removed");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkPurgingSelectedRemoved.value = false;
  }
}

async function handlePurgeRemoved(id: string) {
  purgingRemovedNodeId.value = id;
  try {
    await nodePoolAPI.bulkPurgeRemoved({ ids: [id] });
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    if (purgingRemovedNodeId.value === id) {
      purgingRemovedNodeId.value = null;
    }
  }
}

async function handleBulkPurgeRemovedFullFailures() {
  const ids = removedFullFailureNodes.value.map((node) => node.id);
  if (!ids.length) return;

  bulkPurgingRemoved.value = true;
  try {
    await nodePoolAPI.bulkPurgeRemoved({ ids });
    clearSelection("removed");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkPurgingRemoved.value = false;
  }
}

async function handleBulkPromote(group: "staging" | "quarantine") {
  const ids =
    group === "staging"
      ? selectedStagingIds.value
      : selectedQuarantineIds.value;
  if (!ids.length) return;

  bulkPromotingGroup.value = group;
  try {
    await nodePoolAPI.bulkPromote({ ids });
    clearSelection(group);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkPromotingGroup.value = null;
  }
}

async function handleBulkRemoveQuarantine() {
  bulkRemoving.value = true;
  try {
    await nodePoolAPI.bulkRemove({
      statuses: ["quarantine"],
      onlyUnstable: true,
    });
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkRemoving.value = false;
  }
}

async function handleBulkRemoveFullFailures() {
  const ids = fullFailureRemovalCandidates.value.map((node) => node.id);
  if (!ids.length) return;

  bulkRemovingFullFailures.value = true;
  try {
    await nodePoolAPI.bulkRemove({ ids });
    clearSelection("candidate");
    clearSelection("staging");
    clearSelection("quarantine");
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    bulkRemovingFullFailures.value = false;
  }
}

async function handleSaveConfig() {
  savingConfig.value = true;
  try {
    await nodePoolAPI.updateConfig(configForm.value);
    message.success(t("common.success"));
    await refreshAll();
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    savingConfig.value = false;
  }
}

async function handleTestDestinationBinding(binding: DestinationBindingDraft) {
  const payload = draftToBinding(binding);
  const domain = bindingPrimaryTestDomain(payload);
  if (!payload.nodeId) {
    message.error(t("nodePool.destinationBindingNodeRequired"));
    return;
  }
  if (!domain) {
    message.error(t("nodePool.destinationBindingDomainRequired"));
    return;
  }

  testingDestinationBindingKey.value = binding.key;
  try {
    const data = await routingAPI.testRoute({
      scope: "tun",
      domain,
      port: 443,
      network: "tcp",
      inboundTag: "tun-in",
    });
    const result = data?.result || {};
    const expectedTarget = `pool-active-${payload.nodeId}`;
    const actualTarget = result.outboundTag || result.balancerTag || "";
    destinationBindingTestResults.value = {
      ...destinationBindingTestResults.value,
      [binding.key]: {
        domain,
        expectedTarget,
        actualTarget,
        matched: actualTarget === expectedTarget,
      },
    };
    if (actualTarget === expectedTarget) {
      message.success(
        t("nodePool.destinationBindingTestMatched", {
          domain,
          target: actualTarget,
        }),
      );
    } else {
      message.warning(
        t("nodePool.destinationBindingTestMismatch", {
          domain,
          expected: expectedTarget,
          actual: actualTarget || "-",
        }),
      );
    }
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    testingDestinationBindingKey.value = null;
  }
}

async function handleSaveTunSettings() {
  savingTunSettings.value = true;
  try {
    const saved = await tunAPI.updateSettings({
      selectionPolicy: tunSettingsForm.value.selectionPolicy,
      routeMode: tunSettingsForm.value.routeMode,
      remoteDns: normalizeListInput(tunRemoteDnsText.value),
      protectDomains: normalizeListInput(tunProtectDomainsText.value),
      protectCidrs: normalizeListInput(tunProtectCidrsText.value),
      destinationBindings: destinationBindingDrafts.value.map(draftToBinding),
      aggregation: normalizeAggregationSettings(
        tunSettingsForm.value.aggregation,
      ),
    });
    tunSettingsForm.value = {
      selectionPolicy: saved.selectionPolicy || "fastest",
      routeMode: saved.routeMode || "strict_proxy",
      remoteDns: Array.isArray(saved.remoteDns) ? saved.remoteDns : [],
      protectDomains: Array.isArray(saved.protectDomains)
        ? saved.protectDomains
        : [],
      protectCidrs: Array.isArray(saved.protectCidrs) ? saved.protectCidrs : [],
      destinationBindings: Array.isArray(saved.destinationBindings)
        ? saved.destinationBindings
        : [],
      aggregation: normalizeAggregationSettings(saved.aggregation),
    };
    syncTunSettingsTextAreas();
    syncDestinationBindingDrafts(tunSettingsForm.value.destinationBindings);
    message.success(
      tunStatus.value.running
        ? t("nodePool.tunSettingsSavedRunning")
        : t("nodePool.tunSettingsSaved"),
    );
  } catch (err: any) {
    message.error(err?.message || err?.error || t("common.error"));
  } finally {
    savingTunSettings.value = false;
  }
}

onMounted(() => {
  syncViewport();
  window.addEventListener("resize", syncViewport);
  refreshAll();
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", syncViewport);
});
</script>

<style scoped>
.node-pool-page h2,
.node-pool-page h3 {
  margin: 0;
}

.node-pool-subtitle {
  margin: 6px 0 0;
  color: var(--n-text-color-3);
}

.machine-strip {
  border-left: 4px solid var(--n-color-target, #18a058);
}

.tun-settings-card {
  border-left: 4px solid var(--n-color-target, #2080f0);
}

.machine-actions {
  justify-content: flex-end;
}

.removed-toolbar {
  margin-bottom: 12px;
}

.node-pool-sections :deep(.n-collapse-item__header) {
  font-weight: 600;
}

.node-pool-sections :deep(.n-collapse-item__content-inner) {
  padding-top: 16px;
}

.pool-sort-select {
  min-width: 220px;
}

.validation-config-layout {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(0, 1.5fr) minmax(320px, 1fr);
  align-items: start;
}

.validation-config-form {
  min-width: 0;
}

.validation-config-guide {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  background: var(--n-color);
}

.validation-guide-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.validation-guide-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-bottom: 12px;
  border-bottom: 1px dashed var(--n-border-color);
}

.validation-guide-item:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

.validation-guide-footnote {
  padding: 12px 14px;
  border-radius: 10px;
  background: color-mix(
    in srgb,
    var(--n-color-target, #2080f0) 8%,
    transparent
  );
  color: var(--n-text-color-2);
  font-size: 13px;
  line-height: 1.6;
}

.node-pool-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}

.summary-strip {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.pool-guide-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.pool-guide-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  background: var(--n-color);
}

.pool-guide-desc {
  color: var(--n-text-color-2);
  font-size: 13px;
  line-height: 1.6;
}

.summary-item {
  padding: 14px 16px;
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  background: var(--n-color);
}

.summary-label {
  color: var(--n-text-color-3);
  font-size: 12px;
}

.summary-value {
  margin-top: 6px;
  font-size: 24px;
  font-weight: 600;
}

.node-intelligence-cell {
  display: inline-flex;
  align-items: center;
}

.node-intelligence-trigger {
  cursor: help;
}

.node-intelligence-tag-group {
  max-width: 100%;
}

.node-intelligence-popover {
  display: flex;
  max-width: min(360px, 70vw);
  flex-direction: column;
  gap: 8px;
}

.node-intelligence-popover-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--n-text-color-3);
}

.node-intelligence-popover-line {
  line-height: 1.5;
  word-break: break-word;
}

.node-intelligence-secondary,
.node-card-meta-detail {
  line-height: 1.5;
}

.section-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.event-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.event-time {
  color: var(--n-text-color-3);
  font-size: 12px;
  white-space: nowrap;
}

.node-card-list {
  display: grid;
  gap: 12px;
}

.node-card {
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-card-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.node-card-meta {
  color: var(--n-text-color-3);
  font-size: 13px;
}

.node-card-actions {
  margin-top: 4px;
}

.aggregation-prototype-card {
  width: 100%;
}

.aggregation-prototype-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.destination-binding-editor {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.destination-binding-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.destination-binding-card {
  border-left: 3px solid
    color-mix(in srgb, var(--n-color-target, #2080f0) 45%, transparent);
}

.destination-binding-row {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.destination-binding-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.destination-binding-label {
  font-size: 12px;
  color: var(--n-text-color-3);
}

.destination-binding-warning {
  color: var(--n-warning-color, #f0a020);
}

.destination-binding-test-alert {
  margin-top: 4px;
}

@media (max-width: 1199px) {
  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pool-guide-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .validation-config-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .summary-strip {
    grid-template-columns: 1fr;
  }

  .pool-guide-grid {
    grid-template-columns: 1fr;
  }

  .machine-actions {
    width: 100%;
  }

  .machine-actions :deep(button) {
    width: 100%;
  }

  .pool-sort-select {
    width: 100%;
  }

  .destination-binding-row {
    grid-template-columns: 1fr;
  }

  .event-row {
    flex-direction: column;
  }
}
</style>
