<template>
  <BaseDialog :show="show" :title="dialogTitle" width="extra-wide" @close="handleClose">
    <div class="space-y-4">
      <div
        v-if="account"
        class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700"
      >
        <div class="flex min-w-0 items-center gap-3">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-500 text-white">
            <Icon name="shield" size="md" :stroke-width="2" />
          </div>
          <div class="min-w-0">
            <div class="truncate font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
              <span class="rounded bg-primary-50 px-1.5 py-0.5 font-medium uppercase text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                {{ providerLabel }} / {{ account.type }}
              </span>
              <span class="font-mono">#{{ account.id }}</span>
            </div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Select
            v-model="selectedModelId"
            :options="modelOptions"
            :disabled="loadingModels || runStatus === 'running'"
            value-key="id"
            label-key="display_name"
            class="min-w-[220px]"
            :placeholder="loadingModels ? t('purity.ui.loadingModels') : t('purity.ui.selectModel')"
            :empty-text="t('purity.ui.noModels')"
          />
          <label class="inline-flex min-h-9 items-center gap-2 text-xs text-gray-600 dark:text-dark-300">
            <input
              v-model="checkTokenUsage"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :disabled="runStatus === 'running'"
            />
            <span>{{ t('purity.detail.tokenAuditOption') }}</span>
          </label>
          <button type="button" class="btn btn-primary btn-sm" :disabled="runStatus === 'running' || !selectedModelId || !isSupportedAccount" @click="startCheck">
            <Icon v-if="runStatus === 'running'" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <Icon v-else name="play" size="sm" :stroke-width="2" />
            <span>{{ runStatus === 'running' ? t('purity.ui.checking') : t('purity.ui.start') }}</span>
          </button>
        </div>
      </div>

      <div v-if="!isSupportedAccount" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200">
        {{ t('purity.ui.unsupportedAccount') }}
      </div>
      <div v-if="fatalReportError" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-500/40 dark:bg-red-900/20 dark:text-red-200">
        <div class="font-semibold">{{ t('purity.ui.checkFailed') }}</div>
        <div class="mt-1 break-words">{{ fatalReportError }}</div>
      </div>
      <div v-else-if="probeIssueMessage" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200">
        <div class="font-semibold">{{ t('purity.ui.partialProbeIssue') }}</div>
        <div class="mt-1 break-words">{{ probeIssueMessage }}</div>
      </div>

      <div class="grid gap-3 lg:grid-cols-[260px_1fr]">
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-500 dark:bg-dark-700">
          <div class="flex items-center justify-center">
            <div class="score-ring" :style="scoreRingStyle">
              <div class="score-ring-inner">
                <div class="text-3xl font-bold text-gray-950 dark:text-white">{{ displayScore }}</div>
                <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-dark-400">proxyai.best</div>
              </div>
            </div>
          </div>
          <div class="mt-4 text-center">
            <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ verdictLabel }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ assessmentDisplayTitle || runningSummary }}</div>
          </div>
          <div class="mt-4">
            <div class="mb-1 flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
              <span>{{ stepLabel }}</span>
              <span>{{ progressPercent }}%</span>
            </div>
            <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
              <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: `${progressPercent}%` }" />
            </div>
          </div>
          <div class="mt-4 grid grid-cols-2 gap-2 text-center text-xs">
            <div class="rounded-md bg-white p-2 dark:bg-dark-600">
              <div class="font-semibold text-gray-900 dark:text-gray-100">{{ report?.compatibility_score ?? '-' }}</div>
              <div class="text-gray-500 dark:text-dark-400">{{ t('purity.ui.compatibilityScore') }}</div>
            </div>
            <div class="rounded-md bg-white p-2 dark:bg-dark-600">
              <div class="font-semibold text-gray-900 dark:text-gray-100">{{ report?.official_score ?? '-' }}</div>
              <div class="text-gray-500 dark:text-dark-400">{{ t('purity.ui.officialScore') }}</div>
            </div>
          </div>
        </div>

        <div class="space-y-3">
          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="item in displayedValidations"
            :key="item.id"
            class="rounded-lg border bg-white p-3 dark:bg-dark-700"
            :class="validationCardClass(item.status)"
          >
            <div class="flex items-start gap-2">
              <span class="mt-0.5 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full" :class="validationIconClass(item.status)">
                <Icon :name="validationIcon(item.status)" size="sm" :class="{ 'animate-spin': item.status === 'running' }" :stroke-width="2" />
              </span>
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-1.5">
                  <span class="font-medium text-gray-900 dark:text-gray-100">{{ item.name }}</span>
                  <span class="rounded px-1.5 py-0.5 text-[10px] font-medium uppercase" :class="validationBadgeClass(item.status)">
                    {{ validationStatusLabel(item.status) }}
                  </span>
                </div>
                <div class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ item.message }}</div>
              </div>
            </div>
          </div>
          </div>
          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
            <div v-for="item in scoreBreakdownItems" :key="item.key" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ item.label }}</span>
                <span class="text-xs font-semibold text-gray-900 dark:text-gray-100">{{ item.display }}</span>
              </div>
              <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-600">
                <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: `${item.percent}%` }" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div v-for="item in evidenceCards" :key="item.label" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</div>
          <div class="mt-1 truncate text-sm font-semibold text-gray-900 dark:text-gray-100" :title="item.value">{{ item.value }}</div>
          <div class="mt-1 line-clamp-2 text-xs leading-5 text-gray-500 dark:text-dark-400" :title="item.description">{{ item.description }}</div>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div v-for="metric in metricCards" :key="metric.label" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ metric.label }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">{{ metric.value }}</div>
        </div>
      </div>

      <div v-if="assessment" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-500 dark:bg-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('purity.detail.assessment') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-gray-100">{{ assessmentDisplayTitle }}</div>
          </div>
          <span class="badge" :class="assessmentStatusClass">{{ assessmentStatusLabel }}</span>
        </div>
        <div class="mt-3 grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
          <div v-for="item in assessmentFacts" :key="item.label" class="min-w-0">
            <div class="text-gray-500 dark:text-dark-400">{{ item.label }}</div>
            <div class="mt-0.5 break-words font-medium text-gray-900 dark:text-gray-100">{{ item.value }}</div>
          </div>
        </div>
      </div>

      <div v-if="scorePolicy" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-500 dark:bg-dark-700">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('purity.detail.scorePolicy') }}</div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-gray-100">{{ scorePolicyDisplayName }}</div>
          </div>
          <span class="badge badge-secondary">{{ channelDisplayName(scorePolicy.channel) }}</span>
        </div>
        <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
          <div v-for="dimension in scorePolicy.dimensions" :key="dimension.id" class="min-w-0 rounded-md bg-gray-50 px-2.5 py-2 text-xs dark:bg-dark-600">
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-gray-600 dark:text-dark-300">{{ scoreDimensionLabel(dimension.id) }}</span>
              <span class="font-semibold text-gray-900 dark:text-gray-100">{{ dimension.max_score }}</span>
            </div>
            <div class="mt-1 text-[10px] leading-4 text-gray-500 dark:text-dark-400">{{ scoreDimensionRule(dimension) }}</div>
          </div>
        </div>
        <div v-if="scorePolicy.excluded_dimensions?.length" class="mt-3 text-xs leading-5 text-gray-500 dark:text-dark-400">
          <span class="font-semibold text-gray-700 dark:text-dark-300">{{ t('purity.detail.scoreExcluded') }}：</span>
          {{ scorePolicy.excluded_dimensions.map(scoreDimensionLabel).join(' · ') }}
        </div>
        <div v-if="scoreAdjustments.length" class="mt-4 border-t border-gray-200 pt-3 dark:border-dark-500">
          <div class="mb-2 text-xs font-semibold text-gray-800 dark:text-gray-100">{{ t('purity.detail.scoreAdjustments') }}</div>
          <div class="space-y-2">
            <div v-for="adjustment in scoreAdjustments" :key="adjustment.id" class="rounded-md border border-red-200 bg-red-50 p-3 text-xs dark:border-red-500/40 dark:bg-red-900/20">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <span class="font-semibold text-red-900 dark:text-red-100">{{ scoreAdjustmentTitle(adjustment.reason_code) }}</span>
                <span class="font-semibold text-red-700 dark:text-red-200">{{ adjustment.base_score }} {{ adjustment.points }} = {{ adjustment.result_score }}</span>
              </div>
              <div class="mt-1 leading-5 text-red-800 dark:text-red-100">{{ scoreAdjustmentDescription(adjustment.reason_code) }}</div>
              <div class="mt-1 leading-5 text-red-800 dark:text-red-100">{{ t(`purity.detail.clientImpact.${adjustment.client_impact}`) }}</div>
              <div class="mt-1 font-mono text-[10px] text-red-700 dark:text-red-200">{{ adjustment.case_id }} · {{ adjustment.reason_code }}</div>
            </div>
          </div>
        </div>
      </div>

      <details class="group rounded-lg border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-700" data-testid="purity-report-details">
        <summary class="flex cursor-pointer list-none items-center justify-between gap-3 px-4 py-3">
          <div>
            <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('purity.detail.title') }}</div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ t('purity.detail.summary', { count: dimensions.length || 12 }) }}</div>
          </div>
          <Icon name="chevronDown" size="sm" class="transition-transform group-open:rotate-180" :stroke-width="2" />
        </summary>

        <div class="space-y-4 border-t border-gray-200 p-4 dark:border-dark-500">
          <section>
            <div class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('purity.detail.dimensions') }}</div>
            <div class="divide-y divide-gray-200 rounded-md border border-gray-200 dark:divide-dark-500 dark:border-dark-500">
              <details
                v-for="dimension in dimensions"
                :key="dimension.id"
                class="group/dimension"
                :data-dimension-id="dimension.id"
              >
                <summary class="grid cursor-pointer list-none grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 px-3 py-2.5">
                  <div class="min-w-0">
                    <div class="truncate text-xs font-semibold text-gray-900 dark:text-gray-100">{{ dimensionDisplayName(dimension) }}</div>
                    <div class="mt-0.5 truncate font-mono text-[10px] text-gray-400 dark:text-dark-400">{{ dimension.id }}</div>
                  </div>
                  <span class="text-[10px] font-medium" :class="dimensionStatusClass(dimension.status)">{{ dimensionStatusLabel(dimension.status) }}</span>
                  <span class="min-w-[64px] text-right text-xs font-semibold text-gray-900 dark:text-gray-100">{{ dimensionScoreLabel(dimension) }}</span>
                </summary>
                <div class="space-y-3 border-t border-gray-100 px-3 py-3 text-xs dark:border-dark-600">
                  <p class="leading-5 text-gray-600 dark:text-dark-300">{{ dimensionDisplayMessage(dimension) }}</p>
                  <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                    <div>
                      <span class="text-gray-400">{{ t('purity.detail.category') }}：</span>
                      <span class="text-gray-700 dark:text-dark-300">{{ dimensionCategoryLabel(dimension.category) }}</span>
                    </div>
                    <div>
                      <span class="text-gray-400">{{ t('purity.detail.mode') }}：</span>
                      <span class="text-gray-700 dark:text-dark-300">{{ dimensionModeLabel(dimension.mode) }}</span>
                    </div>
                    <div>
                      <span class="text-gray-400">{{ t('purity.detail.sourceChecks') }}：</span>
                      <span class="break-words font-mono text-gray-700 dark:text-dark-300">{{ dimension.source_check_ids?.join(', ') || '-' }}</span>
                    </div>
                    <div>
                      <span class="text-gray-400">{{ t('purity.detail.limitations') }}：</span>
                      <span class="break-words text-gray-700 dark:text-dark-300">{{ dimensionLimitationsLabel(dimension.limitations) }}</span>
                    </div>
                  </div>
                  <dl v-if="safePurityDetailEntries(dimension.details).length" class="grid gap-2 rounded-md bg-gray-50 p-2 dark:bg-dark-600">
                    <div v-for="entry in safePurityDetailEntries(dimension.details)" :key="entry[0]" class="grid gap-1 sm:grid-cols-[150px_minmax(0,1fr)]">
                      <dt class="font-medium text-gray-500 dark:text-dark-400">{{ purityDetailKeyLabel(entry[0]) }}</dt>
                      <dd class="break-all whitespace-pre-wrap text-gray-800 dark:text-gray-100">{{ formatPurityDetailValue(entry[1]) }}</dd>
                    </div>
                  </dl>
                  <div v-if="sourceChecksForDimension(dimension).length" class="space-y-2">
                    <div class="font-semibold text-gray-700 dark:text-dark-300">{{ t('purity.detail.sourceResults') }}</div>
                    <details v-for="check in sourceChecksForDimension(dimension)" :key="check.id" class="rounded-md border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-700">
                      <summary class="flex cursor-pointer list-none items-center justify-between gap-2 px-2.5 py-2">
                        <span class="font-medium text-gray-800 dark:text-gray-100">{{ checkDisplayName(check) }}</span>
                        <span :class="checkStatusClass(check.status)">{{ check.score }}/{{ check.max_score }}</span>
                      </summary>
                      <div class="border-t border-gray-100 px-2.5 py-2 dark:border-dark-600">
                        <p class="text-gray-500 dark:text-dark-400">{{ checkDisplayMessage(check) }}</p>
                        <dl v-if="safePurityDetailEntries(check.details).length" class="mt-2 space-y-1">
                          <div v-for="entry in safePurityDetailEntries(check.details)" :key="entry[0]" class="grid gap-1 sm:grid-cols-[130px_minmax(0,1fr)]">
                            <dt class="font-medium text-gray-400">{{ purityDetailKeyLabel(entry[0]) }}</dt>
                            <dd class="break-all whitespace-pre-wrap text-gray-700 dark:text-dark-300">{{ formatPurityDetailValue(entry[1]) }}</dd>
                          </div>
                        </dl>
                      </div>
                    </details>
                  </div>
                </div>
              </details>
              <div v-if="dimensions.length === 0" class="p-4 text-center text-xs text-gray-400 dark:text-dark-400">{{ t('purity.detail.waitingDimensions') }}</div>
            </div>
          </section>

          <div class="grid gap-3" :class="tokenAuditRequested ? 'lg:grid-cols-[1fr_320px]' : 'grid-cols-1'">
        <div v-if="tokenAuditRequested" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-500 dark:bg-dark-700">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('purity.detail.tokenAudit') }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ tokenAuditSummary }}</div>
            </div>
            <span class="badge" :class="tokenAuditBadgeClass">{{ tokenAuditStatusLabel }}</span>
          </div>
          <div class="mb-3 rounded-md border px-3 py-2 text-xs font-medium leading-5" :class="tokenAuditNoticeClass">
            {{ tokenAuditNoticeText }}
          </div>
          <div class="mb-3 grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
            <div v-for="item in tokenAuditMetricCards" :key="item.label" class="rounded-md bg-gray-50 p-2 dark:bg-dark-600">
              <div class="text-[10px] text-gray-500 dark:text-dark-400">{{ item.label }}</div>
              <div class="mt-0.5 text-sm font-semibold" :class="auditValueTextClass(item.tone)">{{ item.value }}</div>
            </div>
          </div>
          <div class="h-36 overflow-x-auto">
            <div class="flex h-full min-w-[520px] items-end gap-2">
              <div v-for="sample in auditSamplesForChart" :key="sample.index" class="flex h-full flex-1 flex-col justify-end gap-1">
                <div class="flex flex-1 items-end justify-center gap-1 rounded bg-gray-100 px-1 dark:bg-dark-600">
                  <div
                    class="w-2 rounded-t bg-gray-400 transition-all dark:bg-dark-400"
                    :style="{ height: `${sampleBaselineBarHeight(sample)}%` }"
                  />
                  <div
                    class="w-2 rounded-t transition-all"
                    :class="auditBarClass(tokenAuditSampleRatioCell(sample).tone)"
                    :style="{ height: `${sampleActualBarHeight(sample)}%` }"
                  />
                </div>
                <div class="text-center text-[10px] text-gray-500 dark:text-dark-400">R{{ sample.index }}</div>
                <div class="text-center text-[10px] font-semibold" :class="auditToneTextClass(tokenAuditSampleRatioCell(sample).tone)" :title="tokenAuditSampleRatioCell(sample).title">
                  {{ tokenAuditSampleRatioCell(sample).display }}
                </div>
              </div>
            </div>
          </div>
          <div class="mt-2 flex justify-end gap-3 text-[10px] text-gray-500 dark:text-dark-400">
            <span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-gray-400 dark:bg-dark-400" />{{ t('purity.ui.audit.officialBaseline') }}</span>
            <span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-emerald-400" />{{ t('purity.ui.audit.usageEstimate') }}</span>
          </div>
          <div class="mt-3 overflow-x-auto rounded-md border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-800">
            <table class="min-w-[780px] table-fixed text-left text-xs text-gray-950 dark:text-gray-100">
              <thead class="text-gray-700 dark:text-dark-300">
                <tr>
                  <th class="w-12 py-1 pr-2 font-medium">{{ t('purity.ui.audit.round') }}</th>
                  <th class="w-24 py-1 pr-2 font-medium">{{ t('purity.ui.audit.mode') }}</th>
                  <th class="w-16 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.latency') }}</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.input') }}</th>
                  <th class="w-20 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.output') }}</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.cacheCreation') }}</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.cacheRead') }}</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">{{ t('purity.ui.audit.usageEstimate') }}</th>
                  <th class="w-20 py-1 pr-2 font-medium text-right">{{ tokenAuditSampleRatioHeader }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
                <tr v-for="sample in auditSamplesForTable" :key="sample.index">
                  <td class="py-1.5 pr-2 font-mono">R{{ sample.index }}</td>
                  <td class="py-1.5 pr-2">
                    <div class="text-gray-700 dark:text-dark-300">{{ auditRequestModeLabel(sample.request_mode) }}</div>
                    <div v-if="auditSampleFailureSummary(sample)" class="mt-0.5 max-w-[88px] truncate text-[10px] font-semibold text-red-600 dark:text-red-300" :title="auditSampleFailureTitle(sample)">
                      {{ auditSampleFailureSummary(sample) }}
                    </div>
                  </td>
                  <td class="py-1.5 pr-2 text-right">
                    <span class="inline-flex min-w-[54px] justify-center rounded-full border px-1.5 py-0.5 text-[10px] font-semibold" :class="auditToneBadgeClass(tokenAuditSampleRow(sample).latency.tone)">
                      {{ formatTokenAuditLatencyMS(tokenAuditSampleRow(sample).latency.value) }}
                    </span>
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRow(sample).input.tone)">
                    {{ formatInteger(tokenAuditSampleRow(sample).input.value) }}<span v-if="sample.input_delta_pct" class="ml-1 text-[10px]" :class="deltaTextClass(sample.input_delta_pct)">{{ deltaLabel(sample.input_delta_pct) }}</span>
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRow(sample).output.tone)">
                    {{ formatInteger(tokenAuditSampleRow(sample).output.value) }}<span v-if="sample.output_delta_pct" class="ml-1 text-[10px]" :class="deltaTextClass(sample.output_delta_pct)">{{ deltaLabel(sample.output_delta_pct) }}</span>
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRow(sample).cacheCreation.tone)" :title="tokenAuditSampleRow(sample).cacheCreation.title">
                    {{ tokenAuditSampleRow(sample).cacheCreation.display }}<span v-if="tokenAuditSampleRow(sample).cacheCreation.available && sample.cache_creation_delta_pct" class="ml-1 text-[10px]" :class="deltaTextClass(sample.cache_creation_delta_pct)">{{ deltaLabel(sample.cache_creation_delta_pct) }}</span>
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRow(sample).cacheRead.tone)" :title="tokenAuditSampleRow(sample).cacheRead.title">
                    {{ tokenAuditSampleRow(sample).cacheRead.display }}<span v-if="tokenAuditSampleRow(sample).cacheRead.available && sample.cache_read_delta_pct" class="ml-1 text-[10px]" :class="deltaTextClass(sample.cache_read_delta_pct)">{{ deltaLabel(sample.cache_read_delta_pct) }}</span>
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRow(sample).cost.tone)">
                    {{ formatUSD(tokenAuditSampleRow(sample).cost.value) }}
                  </td>
                  <td class="py-1.5 pr-2 text-right font-semibold" :class="auditValueTextClass(tokenAuditSampleRatioCell(sample).tone)" :title="tokenAuditSampleRatioCell(sample).title">
                    {{ tokenAuditSampleRatioCell(sample).display }}
                  </td>
                </tr>
                <tr v-if="auditSamplesForTable.length === 0">
                  <td colspan="9" class="py-4 text-center text-gray-700 dark:text-dark-300">{{ emptyAuditTableText }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('purity.detail.rawChecks') }}</div>
          <div class="mt-3 max-h-[300px] space-y-2 overflow-y-auto pr-1">
            <details v-for="check in reportChecks" :key="check.id" class="rounded-md bg-gray-50 dark:bg-dark-600">
              <summary class="flex cursor-pointer list-none items-center justify-between gap-2 p-2">
                <span class="text-xs font-medium text-gray-800 dark:text-gray-100">{{ checkDisplayName(check) }}</span>
                <span class="text-xs" :class="checkStatusClass(check.status)">{{ check.score }}/{{ check.max_score }}</span>
              </summary>
              <div class="border-t border-gray-200 px-2 pb-2 pt-1.5 text-xs dark:border-dark-500">
                <div class="leading-5 text-gray-500 dark:text-dark-400">{{ checkDisplayMessage(check) }}</div>
                <dl v-if="safePurityDetailEntries(check.details).length" class="mt-2 space-y-1">
                  <div v-for="entry in safePurityDetailEntries(check.details)" :key="entry[0]" class="grid gap-1 sm:grid-cols-[120px_minmax(0,1fr)]">
                    <dt class="font-medium text-gray-400">{{ purityDetailKeyLabel(entry[0]) }}</dt>
                    <dd class="break-all whitespace-pre-wrap text-gray-700 dark:text-dark-300">{{ formatPurityDetailValue(entry[1]) }}</dd>
                  </div>
                </dl>
              </div>
            </details>
            <div v-if="reportChecks.length === 0" class="rounded-md bg-gray-50 p-4 text-center text-xs text-gray-400 dark:bg-dark-600 dark:text-dark-400">
              {{ t('purity.detail.waitingChecks') }}
            </div>
          </div>
        </div>
          </div>
        </div>
      </details>

      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-500/40 dark:bg-red-900/20 dark:text-red-200">
        {{ errorMessage }}
      </div>
    </div>

    <template #footer>
      <div class="flex flex-wrap justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">{{ t('purity.ui.close') }}</button>
        <button type="button" class="btn btn-secondary" :disabled="!canDownloadPDF || runStatus === 'running'" @click="downloadPDF">
          <Icon name="download" size="sm" :stroke-width="2" />
          <span>{{ t('purity.ui.downloadPDF') }}</span>
        </button>
        <button type="button" class="btn btn-primary" :disabled="runStatus === 'running' || !selectedModelId || !isSupportedAccount" @click="startCheck">
          <Icon v-if="runStatus === 'running'" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
          <Icon v-else name="shield" size="sm" :stroke-width="2" />
          <span>{{ runStatus === 'running' ? t('purity.ui.checking') : report ? t('purity.ui.rerun') : t('purity.ui.start') }}</span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  listLocalAccountTestModels,
  localAccountPurityStreamURL,
  type LocalAccountPurityPayload,
  type LocalAccountTestModel,
  type LocalSub2APIAccount,
  type PurityCheckEvent,
  type PurityCheckMetrics,
  type PurityCheckResult,
  type PurityCheckStatus,
  type PurityDimensionResult,
  type PurityDimensionStatus,
  type PurityProvider,
  type PurityReport,
  type PurityScoreBreakdown,
  type PurityTokenAuditReport,
  type PurityTokenAuditSample,
  type PurityValidationResult
} from '@/api/admin/adminPlus'
import { formatPurityDetailValue, purityDetailKeyLabel, safePurityDetailEntries } from '@/utils/purityDetail'
import { downloadPurityReportPDF } from '@/utils/purityPdf'
import {
  formatTokenAuditLatencyMS,
  hasTokenAuditSampleData,
  isGeminiTokenAuditProvider,
  multiplierTone,
  normalizeTokenAuditProvider,
  tokenAuditCostTotals,
  tokenAuditDisplayRatio,
  tokenAuditSampleDisplayRow,
  tokenAuditSampleRatioDisplayCell,
  type TokenAuditTone
} from '@/utils/purityAuditDisplay'
import { formatInteger } from '@/views/admin/operations/SupplierAccountsUtils'

type RunStatus = 'idle' | 'running' | 'success' | 'error'
type DisplayStatus = 'idle' | 'running' | PurityCheckStatus
type IconName = 'checkCircle' | 'exclamationTriangle' | 'xCircle' | 'refresh' | 'clock'
type ScoreBreakdownKey = 'tag_check' | 'structure' | 'behavior' | 'signature_proto' | 'multimodal' | 'token_audit'

interface ValidationDefinition {
  id: string
  name: string
  message: string
}

interface DisplayValidation {
  id: string
  name: string
  status: DisplayStatus
  message: string
}

const props = defineProps<{
  show: boolean
  account: LocalSub2APIAccount | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n({
  useScope: 'local',
  inheritLocale: true,
  messages: {
    zh: {
      purity: {
        ui: {
          dialogTitle: '{provider} API 纯度检测',
          loadingModels: '加载中...',
          selectModel: '选择模型',
          noModels: '暂无模型',
          start: '开始检测',
          rerun: '重新检测',
          checking: '检测中',
          close: '关闭',
          downloadPDF: '下载 PDF',
          unsupportedAccount: '仅支持 OpenAI、Claude 或 Gemini API Key 账号执行纯度检测。',
          checkFailed: '检测失败',
          partialProbeIssue: '部分探针异常',
          compatibilityScore: '兼容分',
          officialScore: '官方分',
          preparing: '准备检测',
          waitingStart: '等待开始',
          runningSummary: '后端探针正在执行：{step}',
          notStarted: '尚未开始检测',
          loadingModelsFailed: '加载模型失败',
          emptyResponse: '响应体为空',
          eventParseFailed: '无法解析检测事件：{event}',
          validationNames: {
            llm_fingerprint: 'LLM 指纹验证',
            schema_integrity: '结构完整性',
            behavior: '行为验证',
            signature: '签名校验',
            multimodal: '多模态能力',
            token_audit: 'Token 用量审计',
            model_identity: '模型身份验证',
            wrapper_fingerprint: '包装指纹验证'
          },
          validationWaiting: {
            llm_fingerprint: '等待模型列表和 Base 域名探测',
            schema_integrity: '等待协议 schema 探测',
            behavior: '等待工具调用和流式事件探测',
            signature: '等待 usage 与协议签名探测',
            multimodal: '等待图像输入探测',
            token_audit: '等待 R1-R11 用量审计',
            model_identity: '等待请求模型与响应模型比对',
            wrapper_fingerprint: '等待中转、反代和兼容网关指纹聚合'
          },
          steps: {
            tag: 'LLM 指纹验证',
            structure: '结构完整性',
            behavior: '行为验证',
            signature: '签名校验',
            multimodal: '多模态能力',
            token_audit: 'Token 用量审计',
            evaluate: '最终评估'
          },
          audit: {
            officialBaseline: '官方基线',
            usageEstimate: 'Usage 估算',
            billingMultiplier: '平台计费倍率',
            usageRatio: 'Usage 比值',
            cacheHitRate: '缓存命中率',
            platformRatio: '平台倍率',
            round: '轮次',
            mode: '模式',
            latency: '耗时',
            input: '输入',
            output: '输出',
            cacheCreation: '缓存创建',
            cacheRead: '缓存读取',
            collecting: '采集中 · {progress}',
            waitingSamples: '等待样本',
            notStarted: '尚未开始',
            waitingFailedSamples: '等待失败诊断样本',
            waitingAuditSamples: '等待审计样本',
            diagnosticRounds: '{count} 轮仅返回诊断',
            geminiCacheNotice: 'Gemini 未返回或未命中的缓存字段以 0 展示；缓存创建字段不可确认，缓存读取 0 表示本轮未观察到命中。',
            multiplierNotice: '平台计费倍率 {billing}；Usage 比值 {ratio}。两者口径不同，前者来自账号配置或 /v1/usage 扣费增量。',
            geminiRatioNotice: 'Usage 比值 {ratio}；Gemini usageMetadata 只能确认本轮 token 统计，平台计费倍率需结合账号配置或 /v1/usage 扣费增量。',
            unknownRatioNotice: 'Usage 比值暂无法确认，需结合每轮 usage 字段和平台账单复核。',
            badRatioNotice: 'Usage 比值 {ratio}，明显高于常见范围，可能存在异常扣费或 Token 统计混淆。',
            warnRatioNotice: 'Usage 比值 {ratio}，高于常见范围，建议结合平台单价或倍率复核。',
            goodRatioNotice: 'Usage 比值 {ratio}，当前未发现明显超额消耗；平台计费倍率需结合账号配置或账单复核。',
            modelList: '模型列表',
            firstToken: '首 Token',
            totalLatency: '总耗时'
          }
        },
        verdict: {
          official_openai: 'OpenAI 官方',
          openai_compatible: 'OpenAI 兼容',
          official_claude: 'Claude 官方',
          claude_compatible: 'Claude 兼容',
          official_gemini: 'Gemini 官方',
          gemini_compatible: 'Gemini 兼容',
          partial_compatible: '兼容受限',
          invalid_or_unavailable: '不可用',
          waiting: '等待检测',
          running: '检测中'
        },
        status: {
          pass: '通过',
          warn: '警告',
          fail: '失败',
          running: '检测中',
          idle: '等待'
        },
        evidence: {
          requestModel: '请求模型',
          requestModelDesc: '检测请求使用的目标模型',
          responseModel: '响应模型',
          responseVendor: '响应厂商',
          responseSource: '来源',
          responseModelPending: '等待上游返回 model 字段',
          modelIdentity: '模型身份',
          modelIdentityPending: '等待模型身份一致性检查',
          wrapperSignals: '包装信号',
          wrapperSignalCount: '{count} 个',
          wrapperSignalsNone: '未发现',
          wrapperSignalsNoneDesc: '未检测到中转、反代或兼容网关指纹',
          suspectedUpstreamVendor: '疑似上游厂商'
        },
        modelIdentity: {
          exactMatch: '请求模型与响应模型一致',
          compatibleAlias: '同厂商别名或预览模型，需结合模型列表确认',
          responseModelMissing: '响应缺少 model 字段，无法完整确认',
          probeModelFallback: '请求模型不可用，已使用同协议可用模型完成探针',
          crossVendorAlias: '请求模型与响应模型属于不同厂商',
          familyMismatch: '请求模型与响应模型属于不同模型家族',
          versionDowngrade: '响应模型版本低于请求模型',
          tierDowngrade: '响应模型档位低于请求模型',
          protocolVendorMismatch: '请求模型与当前协议厂商不一致',
          wrapperVendorMismatch: '包装层暴露的上游厂商与请求模型不一致',
          reasoningTokensMismatch: '非 reasoning 模型响应暴露了 reasoning_tokens',
          completed: '模型身份检查已完成'
        },
        detail: {
          tokenAuditOption: '检测 Token 用量是否异常（额外 11 轮请求）',
          assessment: '结构化判定',
          title: '检测 Detail 与逐项评分',
          summary: '{count} 个检测维度，每项 Detail 独立展开',
          dimensions: '12 维检测矩阵',
          category: '分类',
          mode: '探针模式',
          sourceChecks: '来源探针',
          sourceResults: '来源探针结果',
          limitations: '限制',
          waitingDimensions: '等待后端返回检测维度',
          waitingChecks: '等待后端探针结果',
          rawChecks: '全部原始 Check 与得分',
          tokenAudit: 'Token 用量审计',
          tokenAuditDisabled: '默认未启用，不发送额外 11 轮请求，也不参与本次判定。重新检测前勾选即可执行。',
          scorePolicy: '渠道评分基线',
          scoreExcluded: '本渠道基线不计分',
          scoreAdjustments: '总分调整判例',
          scoreAdjustmentBedrockMaskTitle: 'AWS Bedrock 来源伪装扣分',
          scoreAdjustmentBedrockMaskDescription: '签名同时命中 Bedrock 来源字段与 Anthropic 原生元数据字段。上游仍识别为 AWS Bedrock，但网关未如实保持来源边界，因此按来源透明度判例扣 5 分。',
          scoreAdjustmentModelIdentityTitle: '模型身份冲突满扣',
          scoreAdjustmentModelIdentityDescription: '请求模型与上游返回的模型身份存在会影响客户端使用的冲突，因此该渠道基线中的 LLM 指纹维度执行满扣。',
          scoreDimensionFullDeduction: '失败时该项满扣 {points} 分',
          clientImpact: {
            none: '不影响本轮客户端使用，仅影响来源透明度',
            limited: '失败会影响部分客户端能力',
            breaking: '失败会影响客户端核心使用'
          },
          scorePolicyNames: {
            compatible_protocol: '兼容协议基线',
            anthropic_native_messages: 'Anthropic 原生 Messages 基线',
            aws_bedrock_messages: 'AWS Bedrock Messages 基线',
            google_vertex_claude: 'Google Vertex Claude 基线',
            openai_responses_native: 'OpenAI 原生 Responses 基线',
            google_ai_studio_native: 'Google AI Studio 原生基线'
          },
          scoreDimensions: {
            tag_check: 'LLM 指纹验证',
            structure: '结构完整性',
            behavior: '行为验证',
            signature_proto: '签名校验',
            multimodal: '多模态能力',
            websearch: 'WebSearch',
            fingerprint: '协议与网关指纹',
            token_audit: 'Token 用量审计'
          },
          unscored: '未评分',
          scored: '项已评分',
          channel: '上游渠道',
          identity: '模型身份',
          protocol: '协议兼容',
          wrapper: '网关包装',
          metering: 'Token 审计',
          coverage: '维度覆盖',
          limitationsFact: '能力限制',
          noLimitations: '本轮已执行项未发现明确限制',
          unknown: '未知',
          notTested: '未检测',
          dimensionNames: {
            tag_check: 'LLM 指纹验证',
            stream_structure: '流结构校验',
            non_stream: '非流结构校验',
            websearch: 'WebSearch',
            signature_proto: '签名校验',
            output_config: '结构化输出',
            server_tool: '工具调用',
            token_inject: 'Token 注入',
            knowledge: '知识库检测',
            doc_recognition: '文档识别',
            image_recognition: '图片识别',
            fingerprint: '协议与网关指纹'
          },
          dimensionStatus: {
            pass: '通过',
            warn: '警告',
            fail: '失败',
            not_run: '未执行',
            not_applicable: '不适用',
            unsupported_by_upstream: '上游不支持'
          },
          dimensionMessage: {
            pass: '该维度已执行，关联探针均通过。',
            warn: '该维度已执行，但存在需要复核的证据。',
            fail: '该维度关联探针未通过。',
            not_run: '本轮未执行该主动探针，不计为通过或失败。',
            not_applicable: '该维度不适用于当前协议或渠道。',
            unsupported_by_upstream: '该能力不在当前上游渠道的官方支持范围内。'
          },
          checkMessage: {
            pass: '探针已通过，并返回预期的协议行为。',
            warn: '探针已完成，但证据仍需复核。',
            fail: '探针未返回预期的协议行为。'
          },
          checkNames: {
            base_url: 'Base URL 探测',
            models_schema: '模型列表结构',
            responses_schema: 'Responses 非流式结构',
            responses_structured_output: '结构化输出',
            responses_store_include: '推理状态与 include',
            tool_call: '强制工具调用',
            usage: 'Usage 字段',
            streaming: '流式事件',
            multimodal: '多模态输入',
            chat_completions: 'Chat Completions 兼容',
            chat_completions_n: '多候选输出',
            model_identity: '模型身份一致性',
            channel_attribution: '上游渠道归因',
            wrapper_fingerprint: '网关包装指纹',
            token_audit: 'Token 用量审计',
            claude_messages_schema: 'Messages 非流式结构',
            claude_tool_use: '强制工具调用',
            claude_usage: 'Usage 字段',
            claude_streaming: 'Messages 流式事件',
            claude_multimodal: '多模态输入',
            claude_signature_provenance: 'Thinking 签名来源',
            claude_thinking_signature: 'Thinking 签名结构',
            claude_thinking_budget: 'Thinking 预算约束',
            claude_cache_control_overflow: '缓存控制边界'
          },
          categories: {
            identity: '模型身份',
            model_identity: '模型身份',
            protocol: '协议结构',
            capability: '能力',
            channel_attribution: '渠道归因',
            official_behavior: '官方行为',
            behavior: '行为',
            request_integrity: '请求完整性',
            multimodal: '多模态',
            gateway: '网关证据'
          },
          modes: {
            identity: '模型身份校验',
            stream: '流式协议',
            non_stream: '非流式协议',
            provider_native: '厂商原生行为',
            provider_constraint: '厂商约束',
            channel_evidence: '渠道证据',
            behavior_and_provenance: '行为与来源证据',
            official_behavior: '官方行为',
            encrypted_reasoning_behavior: '加密推理行为',
            json_schema: 'JSON Schema',
            client_tool: '客户端工具调用',
            synthetic_image: '合成图片',
            evidence_only: '仅证据，不计分'
          },
          limitationCodes: {
            managed_websearch_unsupported: '上游渠道不提供 Anthropic 托管 WebSearch。',
            managed_websearch_unsupported_by_bedrock: 'AWS Bedrock 不提供 Anthropic 托管 WebSearch。',
            anthropic_managed_websearch_not_applicable: 'Anthropic 托管 WebSearch 不适用于当前协议。',
            gateway_fingerprint_not_protocol_score: '网关指纹仅作为证据展示，不降低协议兼容分。',
            active_probe_not_implemented: '当前版本未执行该主动完整性探针。',
            versioned_knowledge_probe_not_run: '本轮未执行版本化知识探针。',
            synthetic_document_probe_not_run: '本轮未执行合成文档探针。',
            signature_probe_not_applicable: '当前协议没有适用的同类签名来源探针。',
            structured_output_probe_not_run: '本轮未执行当前协议的结构化输出探针。',
            managed_websearch_not_probed: '托管 WebSearch 可能产生外部搜索费用，本轮未主动探测。'
          },
          limitationsSummary: '{count} 项能力限制，详见检测维度 Detail',
          assessmentKind: {
            official_native: '原生官方渠道',
            official_cloud_channel: '官方云渠道',
            transparent_relay: '透明中转',
            compatible_channel: '兼容渠道',
            channel_conflicted: '渠道证据冲突',
            identity_conflict: '模型身份冲突',
            compatibility_risk: '兼容链路风险',
            invalid_or_unavailable: '接口不可用或协议不完整'
          },
          assessmentStatus: {
            ready: '可用',
            limited: '兼容受限',
            risky: '高风险',
            invalid: '不可用'
          },
          channelStatus: {
            identified: '已识别',
            likely: '较可能',
            unknown: '证据不足',
            conflicted: '证据冲突'
          },
          protocolStatus: {
            high: '高',
            medium: '中',
            low: '低',
            unavailable: '不可用'
          },
          wrapperMode: {
            none: '未发现明显包装信号',
            transparent: '透明中转或兼容网关',
            obfuscating: '存在混淆模型或协议的信号'
          }
        }
      }
    },
    en: {
      purity: {
        ui: {
          dialogTitle: '{provider} API purity check',
          loadingModels: 'Loading...',
          selectModel: 'Select a model',
          noModels: 'No models available',
          start: 'Start check',
          rerun: 'Run again',
          checking: 'Checking',
          close: 'Close',
          downloadPDF: 'Download PDF',
          unsupportedAccount: 'Only OpenAI, Claude, or Gemini API key accounts are supported.',
          checkFailed: 'Check failed',
          partialProbeIssue: 'Some probes reported issues',
          compatibilityScore: 'Compatibility',
          officialScore: 'Official score',
          preparing: 'Preparing',
          waitingStart: 'Waiting to start',
          runningSummary: 'Backend probes are running: {step}',
          notStarted: 'No check has started',
          loadingModelsFailed: 'Failed to load models',
          emptyResponse: 'The response body is empty',
          eventParseFailed: 'Unable to parse detection event: {event}',
          validationNames: {
            llm_fingerprint: 'LLM fingerprint',
            schema_integrity: 'Structure integrity',
            behavior: 'Behavior validation',
            signature: 'Signature validation',
            multimodal: 'Multimodal capability',
            token_audit: 'Token usage audit',
            model_identity: 'Model identity validation',
            wrapper_fingerprint: 'Wrapper fingerprint validation'
          },
          validationWaiting: {
            llm_fingerprint: 'Waiting for model-list and Base URL probes',
            schema_integrity: 'Waiting for protocol schema probes',
            behavior: 'Waiting for tool-call and streaming probes',
            signature: 'Waiting for usage and signature probes',
            multimodal: 'Waiting for image-input probes',
            token_audit: 'Waiting for the R1-R11 usage audit',
            model_identity: 'Waiting to compare requested and response models',
            wrapper_fingerprint: 'Waiting for relay, proxy, and compatible-gateway fingerprint aggregation'
          },
          steps: {
            tag: 'LLM fingerprint',
            structure: 'Structure integrity',
            behavior: 'Behavior validation',
            signature: 'Signature validation',
            multimodal: 'Multimodal capability',
            token_audit: 'Token usage audit',
            evaluate: 'Final assessment'
          },
          audit: {
            officialBaseline: 'Official baseline',
            usageEstimate: 'Usage estimate',
            billingMultiplier: 'Platform billing multiplier',
            usageRatio: 'Usage ratio',
            cacheHitRate: 'Cache hit rate',
            platformRatio: 'Platform ratio',
            round: 'Round',
            mode: 'Mode',
            latency: 'Latency',
            input: 'Input',
            output: 'Output',
            cacheCreation: 'Cache creation',
            cacheRead: 'Cache read',
            collecting: 'Collecting · {progress}',
            waitingSamples: 'Waiting for samples',
            notStarted: 'Not started',
            waitingFailedSamples: 'Waiting for failed diagnostic samples',
            waitingAuditSamples: 'Waiting for audit samples',
            diagnosticRounds: '{count} diagnostic-only round(s)',
            geminiCacheNotice: 'Missing or unobserved Gemini cache fields are shown as zero. Cache creation cannot be confirmed; a cache-read value of zero means no hit was observed in this round.',
            multiplierNotice: 'Platform billing multiplier {billing}; usage ratio {ratio}. These use different measurement bases. The former comes from account configuration or the /v1/usage billing delta.',
            geminiRatioNotice: 'Usage ratio {ratio}. Gemini usageMetadata confirms only this request\'s token counts; verify the platform billing multiplier through account configuration or the /v1/usage billing delta.',
            unknownRatioNotice: 'The usage ratio is unavailable; verify the per-round usage fields and platform bill.',
            badRatioNotice: 'Usage ratio {ratio} is well above the common range and may indicate abnormal billing or token-count confusion.',
            warnRatioNotice: 'Usage ratio {ratio} is above the common range; verify the platform price or multiplier.',
            goodRatioNotice: 'Usage ratio {ratio} shows no obvious excess consumption. Verify the platform billing multiplier through account configuration or billing data.',
            modelList: 'Model list',
            firstToken: 'First token',
            totalLatency: 'Total latency'
          }
        },
        verdict: {
          official_openai: 'Official OpenAI',
          openai_compatible: 'OpenAI compatible',
          official_claude: 'Official Claude',
          claude_compatible: 'Claude compatible',
          official_gemini: 'Official Gemini',
          gemini_compatible: 'Gemini compatible',
          partial_compatible: 'Compatibility limited',
          invalid_or_unavailable: 'Invalid or unavailable',
          waiting: 'Waiting',
          running: 'Checking'
        },
        status: {
          pass: 'Pass',
          warn: 'Warning',
          fail: 'Fail',
          running: 'Checking',
          idle: 'Waiting'
        },
        evidence: {
          requestModel: 'Requested model',
          requestModelDesc: 'Target model used by this check',
          responseModel: 'Response model',
          responseVendor: 'Response vendor',
          responseSource: 'Source',
          responseModelPending: 'Waiting for upstream model field',
          modelIdentity: 'Model identity',
          modelIdentityPending: 'Waiting for model identity check',
          wrapperSignals: 'Wrapper signals',
          wrapperSignalCount: '{count} signal(s)',
          wrapperSignalsNone: 'None',
          wrapperSignalsNoneDesc: 'No relay, proxy, or compatible gateway fingerprint detected',
          suspectedUpstreamVendor: 'Suspected upstream vendor'
        },
        modelIdentity: {
          exactMatch: 'Requested and response models match',
          compatibleAlias: 'Same-vendor alias or preview model; confirm with the model list',
          responseModelMissing: 'The response does not include a model field',
          probeModelFallback: 'Requested model was unavailable; probes used an available model on the same protocol',
          crossVendorAlias: 'Requested and response models belong to different vendors',
          familyMismatch: 'Requested and response models belong to different model families',
          versionDowngrade: 'The response model version is lower than requested',
          tierDowngrade: 'The response model tier is lower than requested',
          protocolVendorMismatch: 'The requested model does not match the current protocol vendor',
          wrapperVendorMismatch: 'Wrapper-exposed upstream vendor does not match the requested model',
          reasoningTokensMismatch: 'A non-reasoning model response exposed reasoning_tokens',
          completed: 'Model identity check completed'
        },
        detail: {
          tokenAuditOption: 'Check abnormal token usage (11 extra requests)',
          assessment: 'Structured assessment',
          title: 'Detection detail and per-item scores',
          summary: '{count} dimensions, each detail expands independently',
          dimensions: '12-dimension detection matrix',
          category: 'Category',
          mode: 'Probe mode',
          sourceChecks: 'Source probes',
          sourceResults: 'Source probe results',
          limitations: 'Limitations',
          waitingDimensions: 'Waiting for detection dimensions',
          waitingChecks: 'Waiting for backend probe results',
          rawChecks: 'All raw checks and scores',
          tokenAudit: 'Token usage audit',
          tokenAuditDisabled: 'Disabled by default. No additional 11-round audit is sent or used in this assessment. Enable it before rerunning to audit usage.',
          scorePolicy: 'Channel scoring baseline',
          scoreExcluded: 'Excluded from this channel baseline',
          scoreAdjustments: 'Overall score adjustments',
          scoreAdjustmentBedrockMaskTitle: 'AWS Bedrock provenance masking penalty',
          scoreAdjustmentBedrockMaskDescription: 'The signature contains both Bedrock provenance fields and Anthropic-native metadata. The upstream remains AWS Bedrock, but the gateway obscures that boundary, so the provenance-transparency case deducts 5 points.',
          scoreAdjustmentModelIdentityTitle: 'Full model-identity deduction',
          scoreAdjustmentModelIdentityDescription: 'The requested model conflicts with the upstream response identity in a way that affects client usage, so the full LLM fingerprint dimension is deducted from this channel baseline.',
          scoreDimensionFullDeduction: 'Failure deducts the full {points}-point dimension',
          clientImpact: {
            none: 'No observed client impact; provenance transparency only',
            limited: 'Failure affects some client capabilities',
            breaking: 'Failure affects core client usage'
          },
          scorePolicyNames: {
            compatible_protocol: 'Compatible protocol baseline',
            anthropic_native_messages: 'Anthropic native Messages baseline',
            aws_bedrock_messages: 'AWS Bedrock Messages baseline',
            google_vertex_claude: 'Google Vertex Claude baseline',
            openai_responses_native: 'OpenAI native Responses baseline',
            google_ai_studio_native: 'Google AI Studio native baseline'
          },
          scoreDimensions: {
            tag_check: 'LLM fingerprint',
            structure: 'Structure integrity',
            behavior: 'Behavior validation',
            signature_proto: 'Signature validation',
            multimodal: 'Multimodal capability',
            websearch: 'WebSearch',
            fingerprint: 'Protocol and gateway fingerprint',
            token_audit: 'Token usage audit'
          },
          unscored: 'Unscored',
          scored: 'scored',
          channel: 'Upstream channel',
          identity: 'Model identity',
          protocol: 'Protocol compatibility',
          wrapper: 'Gateway wrapper',
          metering: 'Token audit',
          coverage: 'Dimension coverage',
          limitationsFact: 'Capability limits',
          noLimitations: 'No explicit limitation was found in the probes executed',
          unknown: 'Unknown',
          notTested: 'Not tested',
          dimensionNames: {
            tag_check: 'LLM fingerprint',
            stream_structure: 'Streaming structure',
            non_stream: 'Non-streaming structure',
            websearch: 'WebSearch',
            signature_proto: 'Signature validation',
            output_config: 'Structured output',
            server_tool: 'Tool call',
            token_inject: 'Token injection',
            knowledge: 'Knowledge check',
            doc_recognition: 'Document recognition',
            image_recognition: 'Image recognition',
            fingerprint: 'Protocol and gateway fingerprint'
          },
          dimensionStatus: {
            pass: 'Pass',
            warn: 'Warning',
            fail: 'Fail',
            not_run: 'Not run',
            not_applicable: 'Not applicable',
            unsupported_by_upstream: 'Unsupported upstream'
          },
          dimensionMessage: {
            pass: 'This dimension ran and all linked probes passed.',
            warn: 'This dimension ran, but some evidence needs review.',
            fail: 'One or more probes linked to this dimension failed.',
            not_run: 'This active probe was not run and is neither a pass nor a failure.',
            not_applicable: 'This dimension does not apply to the current protocol or channel.',
            unsupported_by_upstream: 'This capability is outside the upstream channel\'s official support boundary.'
          },
          checkMessage: {
            pass: 'The probe passed and returned the expected protocol behavior.',
            warn: 'The probe completed, but its evidence needs review.',
            fail: 'The probe did not return the expected protocol behavior.'
          },
          checkNames: {
            base_url: 'Base URL probe',
            models_schema: 'Model list schema',
            responses_schema: 'Responses non-stream schema',
            responses_structured_output: 'Structured output',
            responses_store_include: 'Reasoning state and include',
            tool_call: 'Forced tool call',
            usage: 'Usage fields',
            streaming: 'Streaming events',
            multimodal: 'Multimodal input',
            chat_completions: 'Chat Completions compatibility',
            chat_completions_n: 'Multiple-choice output',
            model_identity: 'Model identity consistency',
            channel_attribution: 'Upstream channel attribution',
            wrapper_fingerprint: 'Gateway wrapper fingerprint',
            token_audit: 'Token usage audit',
            claude_messages_schema: 'Messages non-stream schema',
            claude_tool_use: 'Forced tool call',
            claude_usage: 'Usage fields',
            claude_streaming: 'Messages streaming events',
            claude_multimodal: 'Multimodal input',
            claude_signature_provenance: 'Thinking signature provenance',
            claude_thinking_signature: 'Thinking signature structure',
            claude_thinking_budget: 'Thinking budget constraint',
            claude_cache_control_overflow: 'Cache-control boundary'
          },
          categories: {
            identity: 'Model identity',
            model_identity: 'Model identity',
            protocol: 'Protocol structure',
            capability: 'Capability',
            channel_attribution: 'Channel attribution',
            official_behavior: 'Official behavior',
            behavior: 'Behavior',
            request_integrity: 'Request integrity',
            multimodal: 'Multimodal',
            gateway: 'Gateway evidence'
          },
          modes: {
            identity: 'Model identity validation',
            stream: 'Streaming protocol',
            non_stream: 'Non-streaming protocol',
            provider_native: 'Provider-native behavior',
            provider_constraint: 'Provider constraint',
            channel_evidence: 'Channel evidence',
            behavior_and_provenance: 'Behavior and provenance evidence',
            official_behavior: 'Official behavior',
            encrypted_reasoning_behavior: 'Encrypted reasoning behavior',
            json_schema: 'JSON Schema',
            client_tool: 'Client tool calling',
            synthetic_image: 'Synthetic image',
            evidence_only: 'Evidence only, not scored'
          },
          limitationCodes: {
            managed_websearch_unsupported: 'The upstream channel does not provide Anthropic managed WebSearch.',
            managed_websearch_unsupported_by_bedrock: 'AWS Bedrock does not provide Anthropic managed WebSearch.',
            anthropic_managed_websearch_not_applicable: 'Anthropic managed WebSearch does not apply to this protocol.',
            gateway_fingerprint_not_protocol_score: 'Gateway fingerprints are evidence only and do not reduce the protocol score.',
            active_probe_not_implemented: 'This active integrity probe was not run in the current release.',
            versioned_knowledge_probe_not_run: 'The versioned knowledge probe was not run.',
            synthetic_document_probe_not_run: 'The synthetic document probe was not run.',
            signature_probe_not_applicable: 'No equivalent signature provenance probe applies to this protocol.',
            structured_output_probe_not_run: 'The protocol-specific structured output probe was not run.',
            managed_websearch_not_probed: 'Managed WebSearch was not actively probed because it may incur external search cost.'
          },
          limitationsSummary: '{count} capability limitation(s); see the dimension details',
          assessmentKind: {
            official_native: 'Official native channel',
            official_cloud_channel: 'Official cloud channel',
            transparent_relay: 'Transparent relay',
            compatible_channel: 'Compatible channel',
            channel_conflicted: 'Channel evidence conflict',
            identity_conflict: 'Model identity conflict',
            compatibility_risk: 'Compatibility risk',
            invalid_or_unavailable: 'Unavailable or incomplete protocol'
          },
          assessmentStatus: {
            ready: 'Ready',
            limited: 'Limited',
            risky: 'Risky',
            invalid: 'Invalid'
          },
          channelStatus: {
            identified: 'Identified',
            likely: 'Likely',
            unknown: 'Insufficient evidence',
            conflicted: 'Conflicted evidence'
          },
          protocolStatus: {
            high: 'High',
            medium: 'Medium',
            low: 'Low',
            unavailable: 'Unavailable'
          },
          wrapperMode: {
            none: 'No obvious wrapper signal',
            transparent: 'Transparent relay or compatible gateway',
            obfuscating: 'Model or protocol obfuscation signal detected'
          }
        }
      }
    }
  }
})

const validationDefinitions: ValidationDefinition[] = [
  { id: 'llm_fingerprint', name: 'LLM 指纹验证', message: '等待模型列表和 Base 域名探测' },
  { id: 'schema_integrity', name: '结构完整性', message: '等待协议 schema 探测' },
  { id: 'behavior', name: '行为验证', message: '等待工具调用和流式事件探测' },
  { id: 'signature', name: '签名校验', message: '等待 usage 与协议签名探测' },
  { id: 'multimodal', name: '多模态能力', message: '等待图像输入探测' },
  { id: 'token_audit', name: 'Token 用量审计', message: '等待 R1-R11 用量审计' },
  { id: 'model_identity', name: '模型身份验证', message: '等待请求模型与响应模型比对' },
  { id: 'wrapper_fingerprint', name: '包装指纹验证', message: '等待中转、反代和兼容网关指纹聚合' }
]

const activeValidationByStep: Record<string, string> = {
  tag: 'llm_fingerprint',
  structure: 'schema_integrity',
  behavior: 'behavior',
  signature: 'signature',
  multimodal: 'multimodal',
  token_audit: 'token_audit',
  evaluate: 'model_identity'
}

const defaultScoreDefinitions: Array<{ key: ScoreBreakdownKey; label: string; max: number }> = [
  { key: 'tag_check', label: '指纹', max: 10 },
  { key: 'structure', label: '结构', max: 20 },
  { key: 'behavior', label: '行为', max: 30 },
  { key: 'signature_proto', label: '签名', max: 30 },
  { key: 'multimodal', label: '多模态', max: 10 },
  { key: 'token_audit', label: 'Token', max: 10 }
]

const runStatus = ref<RunStatus>('idle')
const loadingModels = ref(false)
const availableModels = ref<LocalAccountTestModel[]>([])
const selectedModelId = ref('')
const checkTokenUsage = ref(false)
const report = ref<PurityReport | null>(null)
const metrics = ref<PurityCheckMetrics>({})
const scores = ref<PurityScoreBreakdown>({})
const tokenAudit = ref<PurityTokenAuditReport | null>(null)
const auditSamples = ref<PurityTokenAuditSample[]>([])
const checks = ref<PurityCheckResult[]>([])
const validations = ref<Record<string, PurityValidationResult>>({})
const stepName = ref('')
const progress = ref(0)
const tokenAuditProgress = ref('')
const errorMessage = ref('')
const started = ref(false)

let abortController: AbortController | null = null

const modelOptions = computed(() => availableModels.value as unknown as Array<Record<string, unknown>>)
const currentProvider = computed<PurityProvider | null>(() => normalizeAccountProvider(props.account?.platform))
const isGeminiProvider = computed(() => isGeminiTokenAuditProvider(currentProvider.value))
const isSupportedAccount = computed(() => {
  const account = props.account
  return Boolean(account && currentProvider.value && account.type.toLowerCase() === 'apikey')
})
const providerLabel = computed(() => {
  if (currentProvider.value === 'anthropic') return 'Claude'
  if (isGeminiProvider.value) return 'Gemini'
  return 'OpenAI'
})
const dialogTitle = computed(() => t('purity.ui.dialogTitle', { provider: providerLabel.value }))
const displayScore = computed(() => report.value?.score ?? (started.value ? 0 : '-'))
const scoreRingStyle = computed(() => {
  const score = typeof displayScore.value === 'number' ? displayScore.value : 0
  return {
    '--score-angle': `${Math.max(0, Math.min(100, score))}%`,
    '--score-color': scoreRingColor(score)
  }
})
const verdictLabel = computed(() => {
  const verdict = report.value?.verdict || ''
  if (
    verdict === 'official_openai' ||
    verdict === 'openai_compatible' ||
    verdict === 'official_claude' ||
    verdict === 'claude_compatible' ||
    verdict === 'official_gemini' ||
    verdict === 'gemini_compatible' ||
    verdict === 'partial_compatible' ||
    verdict === 'invalid_or_unavailable'
  ) {
    return t(`purity.verdict.${verdict}`)
  }
  return started.value ? t('purity.verdict.running') : t('purity.verdict.waiting')
})
const currentStepName = computed(() => report.value?.step_name || stepName.value)
const stepLabel = computed(() => currentStepName.value
  ? t(`purity.ui.steps.${currentStepName.value}`)
  : started.value
    ? t('purity.ui.preparing')
    : t('purity.ui.waitingStart'))
const progressPercent = computed(() => {
  const value = normalizeProgress(report.value?.progress ?? progress.value)
  return Math.round(value * 100)
})
const runningSummary = computed(() => runStatus.value === 'running'
  ? t('purity.ui.runningSummary', { step: stepLabel.value })
  : t('purity.ui.notStarted'))
const currentRunningValidation = computed(() => activeValidationByStep[currentStepName.value] || '')
const fatalReportError = computed(() => {
  if (report.value?.status === 'error' || report.value?.error) {
    return report.value?.error || metrics.value.error_message || t('purity.ui.checkFailed')
  }
  return ''
})
const probeIssueMessage = computed(() => {
  if (fatalReportError.value || !metrics.value.error_message) return ''
  return metrics.value.error_message
})
const displayedValidations = computed<DisplayValidation[]>(() => validationDefinitions
  .filter((definition) => definition.id !== 'token_audit' || tokenAuditRequested.value)
  .map((definition) => {
    const result = validations.value[definition.id]
    if (result) {
      return {
        id: definition.id,
        name: validationDisplayName(definition),
        status: result.status as DisplayStatus,
        message: result.status === 'pass' || result.status === 'warn' || result.status === 'fail'
          ? t(`purity.detail.checkMessage.${result.status}`)
          : validationWaitingMessage(definition)
      }
    }
    return {
      ...definition,
      name: validationDisplayName(definition),
      message: validationWaitingMessage(definition),
      status: started.value && runStatus.value === 'running' && currentRunningValidation.value === definition.id ? 'running' : 'idle'
    }
  }))
const scoreBreakdownItems = computed(() => {
  const source = report.value?.scores || scores.value
  const policyDefinitions = scorePolicy.value?.dimensions?.length
    ? scorePolicy.value.dimensions.map((dimension) => ({
        key: dimension.id as ScoreBreakdownKey,
        label: scoreDimensionLabel(dimension.id),
        max: dimension.max_score
      }))
    : defaultScoreDefinitions
        .filter((definition) => definition.key !== 'token_audit')
        .map((definition) => ({ ...definition, label: scoreDimensionLabel(definition.key) }))
  const tokenDefinition = defaultScoreDefinitions.find((definition) => definition.key === 'token_audit')!
  const definitions = tokenAuditRequested.value
    ? [...policyDefinitions, { ...tokenDefinition, label: scoreDimensionLabel(tokenDefinition.key) }]
    : policyDefinitions
  return definitions.map((definition) => {
    const rawValue = source[definition.key] ?? 0
    const value = Math.max(0, Math.min(definition.max, rawValue))
    return {
      ...definition,
      value,
      display: `${value}/${definition.max}`,
      percent: Math.round((value / definition.max) * 100)
    }
  })
})
const validAuditSamples = computed(() => normalizedAuditSamples().filter(hasAuditSampleData))
const failedAuditSampleCount = computed(() => normalizedAuditSamples().filter((sample) => !hasAuditSampleData(sample)).length)
const auditSamplesForChart = computed(() => validAuditSamples.value)
const auditSamplesForTable = computed(() => normalizedAuditSamples())
const reportChecks = computed<PurityCheckResult[]>(() => {
  const values = report.value?.checks?.length ? report.value.checks : checks.value
  return values.filter((check) => check.id !== 'token_audit' || tokenAuditRequested.value)
})
const dimensions = computed<PurityDimensionResult[]>(() => {
  const snake = report.value?.dimension_matrix || []
  const values = snake.length ? snake : report.value?.dimensionMatrix || []
  return values.filter((dimension) => dimension.status !== 'not_run')
})
const assessment = computed(() => report.value?.assessment || report.value?.assessmentResult || null)
const scorePolicy = computed(() => report.value?.score_policy || report.value?.scorePolicy || null)
const scoreAdjustments = computed(() => report.value?.score_adjustments || report.value?.scoreAdjustments || [])
const tokenAuditRequested = computed(() => {
  if (report.value) {
    const requested = report.value.check_token_usage ?? report.value.checkTokenUsage
    if (typeof requested === 'boolean') return requested
  }
  return Boolean(checkTokenUsage.value || tokenAudit.value)
})
const assessmentDisplayTitle = computed(() => {
  const value = assessment.value
  if (!value) return ''
  const model = report.value?.response_model || report.value?.responseModel || report.value?.expected_model || report.value?.expectedModel || report.value?.model_id || '-'
  return `${assessmentKindLabel(value.kind)} · ${channelDisplayName(value.channel)} · ${model}`
})
const assessmentStatusLabel = computed(() => assessment.value ? t(`purity.detail.assessmentStatus.${assessment.value.status}`) : t('purity.detail.unknown'))
const assessmentStatusClass = computed(() => {
  if (assessment.value?.status === 'ready') return 'badge-success'
  if (assessment.value?.status === 'limited') return 'badge-warning'
  if (assessment.value?.status === 'risky' || assessment.value?.status === 'invalid') return 'badge-danger'
  return 'badge-secondary'
})
const assessmentFacts = computed(() => {
  const value = assessment.value
  if (!value) return []
  const protocolScore = report.value?.protocol_score ?? report.value?.protocolScore
  const limitations = value.limitations || []
  return [
    {
      label: t('purity.detail.channel'),
      value: `${channelDisplayName(value.channel)} · ${t(`purity.detail.channelStatus.${value.channel_status}`)} · ${formatPercent(value.channel_confidence)}`
    },
    {
      label: t('purity.detail.identity'),
      value: identityStatusDisplay(value.identity_status)
    },
    {
      label: t('purity.detail.protocol'),
      value: `${typeof protocolScore === 'number' ? `${protocolScore}/100 · ` : ''}${t(`purity.detail.protocolStatus.${value.protocol_status}`)}`
    },
    {
      label: t('purity.detail.wrapper'),
      value: t(`purity.detail.wrapperMode.${value.wrapper_mode}`)
    },
    ...(tokenAuditRequested.value ? [{
      label: t('purity.detail.metering'),
      value: meteringStatusDisplay(value.metering_status)
    }] : []),
    {
      label: t('purity.detail.coverage'),
      value: `${value.dimension_executed}/${value.dimension_total} · ${value.dimension_scored} ${t('purity.detail.scored')}`
    },
    {
      label: t('purity.detail.limitationsFact'),
      value: limitations.length ? t('purity.detail.limitationsSummary', { count: limitations.length }) : t('purity.detail.noLimitations')
    }
  ]
})
const scorePolicyDisplayName = computed(() => {
  const id = scorePolicy.value?.id
  if (!id) return ''
  const known = new Set([
    'compatible_protocol',
    'anthropic_native_messages',
    'aws_bedrock_messages',
    'google_vertex_claude',
    'openai_responses_native',
    'google_ai_studio_native'
  ])
  return known.has(id) ? t(`purity.detail.scorePolicyNames.${id}`) : id
})
function scoreAdjustmentTitle(reasonCode: string): string {
  if (reasonCode === 'bedrock_anthropic_signature_mask') return t('purity.detail.scoreAdjustmentBedrockMaskTitle')
  if (reasonCode === 'model_identity_conflict') return t('purity.detail.scoreAdjustmentModelIdentityTitle')
  return reasonCode
}

function scoreAdjustmentDescription(reasonCode: string): string {
  if (reasonCode === 'bedrock_anthropic_signature_mask') return t('purity.detail.scoreAdjustmentBedrockMaskDescription')
  if (reasonCode === 'model_identity_conflict') return t('purity.detail.scoreAdjustmentModelIdentityDescription')
  return reasonCode
}
function scoreDimensionRule(dimension: { max_score: number; client_impact: 'none' | 'limited' | 'breaking' }): string {
  return `${t(`purity.detail.clientImpact.${dimension.client_impact}`)} · ${t('purity.detail.scoreDimensionFullDeduction', { points: dimension.max_score })}`
}
const tokenAuditSummary = computed(() => {
  if (tokenAudit.value) {
    const diagnostic = failedAuditSampleCount.value > 0
      ? ` · ${t('purity.ui.audit.diagnosticRounds', { count: failedAuditSampleCount.value })}`
      : ''
    return `${auditSamplesForTable.value.length}/${tokenAudit.value.sample_count || 11}${diagnostic}`
  }
  if (auditSamples.value.length > 0) {
    return t('purity.ui.audit.collecting', { progress: tokenAuditProgress.value || `${auditSamples.value.length}/11` })
  }
  return started.value ? t('purity.ui.audit.waitingSamples') : t('purity.ui.audit.notStarted')
})
const emptyAuditTableText = computed(() => failedAuditSampleCount.value > 0 ? t('purity.ui.audit.waitingFailedSamples') : t('purity.ui.audit.waitingAuditSamples'))
const tokenAuditMetricCards = computed(() => {
  const audit = tokenAudit.value
  const totals = tokenAuditCostTotals(audit)
  const ratio = tokenAuditDisplayRatio(audit)
  const billingMultiplier = audit?.billing_multiplier ?? audit?.billingMultiplier
  const hasBillingMultiplier = typeof billingMultiplier === 'number' && Number.isFinite(billingMultiplier)
  const cards = [
    { label: t('purity.ui.audit.officialBaseline'), value: formatUSD(totals.officialBaselineUSD), tone: 'neutral' as TokenAuditTone },
    { label: t('purity.ui.audit.usageEstimate'), value: formatUSD(totals.actualCostUSD), tone: ratio > 0 ? multiplierTone(ratio) : 'neutral' as TokenAuditTone },
    hasBillingMultiplier
      ? { label: t('purity.ui.audit.billingMultiplier'), value: formatMultiplier(billingMultiplier), tone: 'good' as TokenAuditTone }
      : { label: t('purity.ui.audit.billingMultiplier'), value: '-', tone: 'neutral' as TokenAuditTone },
    { label: t('purity.ui.audit.usageRatio'), value: formatMultiplier(ratio), tone: multiplierTone(ratio) },
    { label: t('purity.ui.audit.cacheHitRate'), value: formatPercent(audit?.cacheHitRate ?? audit?.cache_hit_rate), tone: audit?.cacheHitRate || audit?.cache_hit_rate ? 'good' as TokenAuditTone : 'neutral' as TokenAuditTone }
  ]
  return cards
})
const tokenAuditRatio = computed(() => tokenAuditDisplayRatio(tokenAudit.value))
const tokenAuditRatioTone = computed(() => multiplierTone(tokenAuditRatio.value))
const tokenAuditBillingMultiplier = computed(() => tokenAudit.value?.billing_multiplier ?? tokenAudit.value?.billingMultiplier)
const hasTokenAuditBillingMultiplier = computed(() => typeof tokenAuditBillingMultiplier.value === 'number' && Number.isFinite(tokenAuditBillingMultiplier.value))
const tokenAuditSampleRatioHeader = computed(() => hasTokenAuditBillingMultiplier.value ? t('purity.ui.audit.platformRatio') : t('purity.ui.audit.usageRatio'))
const geminiCacheFieldNotice = computed(() => {
  if (!isGeminiProvider.value) return ''
  const samples = auditSamplesForTable.value
  if (!samples.length) return ''
  const missingCacheCreate = samples.some((sample) => tokenAuditSampleRow(sample).cacheCreation.available === false)
  const missingCacheRead = samples.some((sample) => tokenAuditSampleRow(sample).cacheRead.available === false)
  if (!missingCacheCreate && !missingCacheRead) return ''
  return t('purity.ui.audit.geminiCacheNotice')
})
const tokenAuditNoticeText = computed(() => {
  const ratio = tokenAuditRatio.value
  const billingMultiplier = tokenAuditBillingMultiplier.value
  const cacheNotice = geminiCacheFieldNotice.value ? ` ${geminiCacheFieldNotice.value}` : ''
  if (typeof billingMultiplier === 'number' && Number.isFinite(billingMultiplier)) {
    return `${t('purity.ui.audit.multiplierNotice', { billing: formatMultiplier(billingMultiplier), ratio: formatMultiplier(ratio) })}${cacheNotice}`
  }
  if (isGeminiProvider.value) return `${t('purity.ui.audit.geminiRatioNotice', { ratio: formatMultiplier(ratio) })}${cacheNotice}`
  if (!ratio) return t('purity.ui.audit.unknownRatioNotice')
  if (tokenAuditRatioTone.value === 'bad') return t('purity.ui.audit.badRatioNotice', { ratio: formatMultiplier(ratio) })
  if (tokenAuditRatioTone.value === 'warn') return t('purity.ui.audit.warnRatioNotice', { ratio: formatMultiplier(ratio) })
  return `${t('purity.ui.audit.goodRatioNotice', { ratio: formatMultiplier(ratio) })}${cacheNotice}`
})
const tokenAuditNoticeClass = computed(() => {
  if (tokenAudit.value?.status === 'fail') return auditToneNoticeClass('bad')
  if (tokenAudit.value?.status === 'warn') return auditToneNoticeClass(tokenAuditRatioTone.value === 'neutral' ? 'warn' : tokenAuditRatioTone.value)
  return auditToneNoticeClass(tokenAuditRatioTone.value)
})
const tokenAuditStatusLabel = computed(() => validationStatusLabel((tokenAudit.value?.status || (auditSamples.value.length > 0 ? 'running' : 'idle')) as DisplayStatus))
const tokenAuditBadgeClass = computed(() => validationBadgeClass((tokenAudit.value?.status || (auditSamples.value.length > 0 ? 'running' : 'idle')) as DisplayStatus))
const metricCards = computed(() => [
  { label: t('purity.ui.audit.modelList'), value: latencyLabel(metrics.value.models_latency_ms) },
  {
    label: currentProvider.value === 'anthropic' ? 'Messages' : isGeminiProvider.value ? 'GenerateContent' : 'Responses',
    value: latencyLabel(currentProvider.value === 'anthropic' ? metrics.value.messages_latency_ms : isGeminiProvider.value ? metrics.value.generate_content_latency_ms || metrics.value.responses_latency_ms : metrics.value.responses_latency_ms)
  },
  { label: t('purity.ui.audit.firstToken'), value: latencyLabel(metrics.value.stream_first_token_ms) },
  { label: t('purity.ui.audit.totalLatency'), value: latencyLabel(metrics.value.latency_ms) }
])
const modelIdentity = computed(() => report.value?.model_identity || report.value?.modelIdentity || null)
const wrapperSignals = computed(() => {
  const snake = report.value?.wrapper_signals || []
  const camel = report.value?.wrapperSignals || []
  return snake.length ? snake : camel
})
const evidenceCards = computed(() => {
	const identity = modelIdentity.value
	const signals = wrapperSignals.value
	return [
		{
			label: t('purity.evidence.requestModel'),
			value: report.value?.expected_model || report.value?.expectedModel || selectedModelId.value || '-',
			description: t('purity.evidence.requestModelDesc')
		},
		{
			label: t('purity.evidence.responseModel'),
			value: report.value?.response_model || report.value?.responseModel || '-',
			description: responseModelDescription(identity)
		},
		{
			label: t('purity.evidence.modelIdentity'),
			value: identity ? validationStatusLabel(identity.status) : t('purity.status.idle'),
			description: identity ? modelIdentityEvidenceDescription(identity) : t('purity.evidence.modelIdentityPending')
		},
		{
			label: t('purity.evidence.wrapperSignals'),
			value: signals.length ? t('purity.evidence.wrapperSignalCount', { count: signals.length }) : t('purity.evidence.wrapperSignalsNone'),
			description: signals.length ? signals.join('、') : t('purity.evidence.wrapperSignalsNoneDesc')
		}
	]
})
const canDownloadPDF = computed(() => Boolean(report.value || started.value))

watch(
  () => props.show,
  async (show) => {
    if (show && props.account) {
      resetAll()
      await loadModels()
      return
    }
    abortStream()
  }
)

async function loadModels() {
  if (!props.account) return
  loadingModels.value = true
  selectedModelId.value = ''
  try {
    const models = await listLocalAccountTestModels(props.account.id)
    availableModels.value = models
    selectedModelId.value = preferredModel(models)
  } catch (error) {
    availableModels.value = []
    errorMessage.value = (error as { message?: string }).message || t('purity.ui.loadingModelsFailed')
    runStatus.value = 'error'
  } finally {
    loadingModels.value = false
  }
}

function preferredModel(models: LocalAccountTestModel[]): string {
  if (currentProvider.value === 'anthropic') {
    return findPreferredModel(models, ['claude-opus-4-8', 'claude-opus-4-7', 'claude-opus', 'opus', 'claude-sonnet-4-6', 'claude-sonnet-4-5', 'claude-sonnet', 'sonnet', 'claude'])
  }
  if (isGeminiProvider.value) {
    return findPreferredModel(models, ['gemini-3.5-flash', 'gemini-3.1-pro', 'gemini-3.1-pro-thinking', 'gemini-3.5-flash-thinking', 'gemini-3-pro-preview', 'gemini-2.5-flash-image', 'gemini-3-flash-preview', 'gemini-3.1-flash-image'])
  }
  return findPreferredModel(models, ['gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5', 'gpt'])
}

function resetAll() {
  abortStream()
  resetRun()
  checkTokenUsage.value = false
  runStatus.value = 'idle'
}

function resetRun() {
  report.value = null
  metrics.value = {}
  scores.value = {}
  tokenAudit.value = null
  auditSamples.value = []
  checks.value = []
  validations.value = {}
  stepName.value = ''
  progress.value = 0
  tokenAuditProgress.value = ''
  errorMessage.value = ''
  started.value = false
}

function handleClose() {
  abortStream()
  emit('close')
}

async function downloadPDF() {
  const snapshot = buildPDFReportSnapshot()
  if (!snapshot) return
  await downloadPurityReportPDF(snapshot, { language: 'zh-CN' })
}

function abortStream() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

async function startCheck() {
  if (!props.account || !selectedModelId.value || !isSupportedAccount.value) return
  resetRun()
  runStatus.value = 'running'
  started.value = true
  abortController = new AbortController()

  try {
    const payload: LocalAccountPurityPayload = {
      provider: currentProvider.value || 'openai',
      model_id: selectedModelId.value,
      check_token_usage: checkTokenUsage.value
    }
    const response = await fetch(localAccountPurityStreamURL(props.account.id), {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
        'Content-Type': 'application/json'
      },
      credentials: 'include',
      body: JSON.stringify(payload),
      signal: abortController.signal
    })
    if (!response.ok) {
      throw new Error(await responseErrorMessage(response))
    }
    if (!response.body) {
      throw new Error(t('purity.ui.emptyResponse'))
    }
    await readNDJSON(response.body)
    if (runStatus.value === 'running') runStatus.value = 'success'
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      runStatus.value = 'idle'
      return
    }
    runStatus.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : t('purity.ui.checkFailed')
  } finally {
    abortController = null
  }
}

async function responseErrorMessage(response: Response): Promise<string> {
  const text = await response.text()
  if (!text) return `HTTP ${response.status}`
  try {
    const payload = JSON.parse(text) as { message?: string; error?: string }
    return payload.message || payload.error || `HTTP ${response.status}`
  } catch {
    return text.slice(0, 160)
  }
}

async function readNDJSON(body: ReadableStream<Uint8Array>) {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''
    for (const line of lines) handleEventLine(line)
  }
  if (buffer.trim()) handleEventLine(buffer)
}

function handleEventLine(line: string) {
  const trimmed = line.trim()
  if (!trimmed) return
  try {
    handleEvent(JSON.parse(trimmed) as PurityCheckEvent)
  } catch {
    errorMessage.value = t('purity.ui.eventParseFailed', { event: trimmed.slice(0, 120) })
  }
}

function handleEvent(event: PurityCheckEvent) {
  applyEventState(event)
  switch (event.type) {
    case 'started':
      if (event.report) {
        applyReportSnapshot(event.report)
      }
      break
    case 'progress':
      if (event.report) applyReportSnapshot(event.report)
      break
    case 'check':
      if (event.check) upsertCheck(event.check)
      break
    case 'validation':
      if (event.validation) validations.value = { ...validations.value, [event.validation.id]: event.validation }
      break
    case 'metrics':
      if (event.metrics) metrics.value = event.metrics
      break
    case 'token_audit_sample':
      if (event.sample) upsertAuditSample(event.sample)
      break
    case 'token_audit':
      if (event.token_audit) tokenAudit.value = event.token_audit
      break
    case 'report':
      if (event.report) {
        applyReportSnapshot(event.report)
        runStatus.value = event.report.status === 'error' ? 'error' : 'success'
      }
      break
    case 'error':
      errorMessage.value = event.error_message || t('purity.ui.checkFailed')
      runStatus.value = 'error'
      break
  }
}

function applyEventState(event: PurityCheckEvent) {
  if (event.step_name) stepName.value = event.step_name
  if (typeof event.progress === 'number') progress.value = normalizeProgress(event.progress)
  if (event.scores) scores.value = { ...scores.value, ...event.scores }
  if (event.metrics) metrics.value = event.metrics
  if (event.token_audit_progress) tokenAuditProgress.value = event.token_audit_progress
  if (event.token_audit_partial?.length) auditSamples.value = sortAuditSamples(event.token_audit_partial)
  if (event.token_audit) tokenAudit.value = event.token_audit
}

function applyReportSnapshot(snapshot: PurityReport) {
  report.value = snapshot
  metrics.value = snapshot.metrics || metrics.value
  if (snapshot.scores) scores.value = { ...scores.value, ...snapshot.scores }
  if (snapshot.token_audit_progress) tokenAuditProgress.value = snapshot.token_audit_progress
  if (snapshot.token_audit_partial?.length) auditSamples.value = sortAuditSamples(snapshot.token_audit_partial)
  tokenAudit.value = snapshot.token_audit || tokenAudit.value
  checks.value = snapshot.checks?.length ? snapshot.checks : checks.value
  if (snapshot.validations?.length) {
    validations.value = Object.fromEntries(snapshot.validations.map((item) => [item.id, item]))
  }
  if (snapshot.step_name) stepName.value = snapshot.step_name
  if (typeof snapshot.progress === 'number') progress.value = normalizeProgress(snapshot.progress)
}

function upsertAuditSample(sample: PurityTokenAuditSample) {
  const next = auditSamples.value.filter((item) => item.index !== sample.index)
  next.push(sample)
  auditSamples.value = sortAuditSamples(next)
}

function upsertCheck(check: PurityCheckResult) {
  const next = checks.value.filter((item) => item.id !== check.id)
  next.push(check)
  checks.value = next
}

function normalizedAuditSamples(): PurityTokenAuditSample[] {
  const source = tokenAudit.value?.samples?.length ? tokenAudit.value.samples : tokenAudit.value?.rows?.length ? tokenAudit.value.rows : auditSamples.value
  return sortAuditSamples(source)
}

function sampleBaselineBarHeight(sample: PurityTokenAuditSample): number {
  const maxCost = maxAuditCost()
  const cost = sample.official_baseline_usd || sample.baseline_cost || 0
  return Math.max(8, Math.round((cost / maxCost) * 100))
}

function sampleActualBarHeight(sample: PurityTokenAuditSample): number {
  const maxCost = maxAuditCost()
  const cost = sample.actual_cost_usd || sample.cost || 0
  return Math.max(8, Math.round((cost / maxCost) * 100))
}

function maxAuditCost(): number {
  return Math.max(0.000001, ...validAuditSamples.value.map((item) => Math.max(item.official_baseline_usd || item.baseline_cost || 0, item.actual_cost_usd || item.cost || 0)))
}

function assessmentKindLabel(kind: string): string {
  const known = new Set([
    'official_native',
    'official_cloud_channel',
    'transparent_relay',
    'compatible_channel',
    'channel_conflicted',
    'identity_conflict',
    'compatibility_risk',
    'invalid_or_unavailable'
  ])
  return known.has(kind) ? t(`purity.detail.assessmentKind.${kind}`) : t('purity.detail.unknown')
}

function channelDisplayName(channel?: string): string {
  const labels: Record<string, string> = {
    anthropic_native: 'Anthropic API',
    aws_bedrock: 'AWS Bedrock',
    google_vertex: 'Google Vertex AI',
    google_ai_studio: 'Google AI Studio',
    openai_native: 'OpenAI API',
    azure_openai: 'Azure OpenAI',
    alibaba_bailian: 'Alibaba Cloud Model Studio',
    baidu_wenxin: 'Baidu Wenxin',
    baidu_qianfan: 'Baidu Qianfan',
    ai360: '360 AI',
    zhipu_bigmodel: 'Zhipu BigModel',
    tencent_hunyuan: 'Tencent Hunyuan',
    moonshot: 'Moonshot AI',
    perplexity: 'Perplexity AI',
    yi: '01.AI',
    cohere: 'Cohere',
    minimax: 'MiniMax',
    siliconflow: 'SiliconFlow',
    mistral: 'Mistral AI',
    deepseek: 'DeepSeek',
    volcengine_ark: 'Volcengine Ark',
    xai: 'xAI',
    zai_coding: 'Z.AI Coding',
    kimi_coding: 'Kimi Coding',
    openai_codex_subscription: 'OpenAI Codex Subscription',
    openrouter: 'OpenRouter',
    cloudflare_workers_ai: 'Cloudflare Workers AI',
    dify: 'Dify',
    coze: 'Coze',
    fastgpt: 'FastGPT',
    submodel: 'Submodel',
    openai_sb: 'OpenAI-SB',
    openaimax: 'OpenAIMax',
    ohmygpt: 'OhMyGPT',
    caipacity: 'CaiPac',
    aiproxy: 'AIProxy',
    api2gpt: 'API2GPT',
    aigc2d: 'AIGC2D',
    anthropic_compatible: 'Claude-compatible',
    openai_compatible: 'OpenAI-compatible',
    gemini_compatible: 'Gemini-compatible',
    kiro: 'Kiro',
    antigravity: 'Antigravity'
  }
  return channel ? labels[channel] || t('purity.detail.unknown') : t('purity.detail.unknown')
}

function identityStatusDisplay(status?: string): string {
  if (status === 'pass' || status === 'warn' || status === 'fail') return validationStatusLabel(status)
  return t('purity.detail.unknown')
}

function meteringStatusDisplay(status?: string): string {
  if (status === 'not_tested') return t('purity.detail.notTested')
  if (status === 'supported' || status === 'pass') return t('purity.status.pass')
  if (status === 'limited' || status === 'warn') return t('purity.status.warn')
  if (status === 'unsupported' || status === 'fail') return t('purity.status.fail')
  return t('purity.detail.unknown')
}

function dimensionDisplayName(dimension: PurityDimensionResult): string {
  const known = new Set([
    'tag_check',
    'stream_structure',
    'non_stream',
    'websearch',
    'signature_proto',
    'output_config',
    'server_tool',
    'token_inject',
    'knowledge',
    'doc_recognition',
    'image_recognition',
    'fingerprint'
  ])
  return known.has(dimension.id) ? t(`purity.detail.dimensionNames.${dimension.id}`) : dimension.name
}

function dimensionDisplayMessage(dimension: PurityDimensionResult): string {
  return t(`purity.detail.dimensionMessage.${dimension.status}`)
}

const knownDimensionCategories = new Set([
  'identity',
  'model_identity',
  'protocol',
  'capability',
  'channel_attribution',
  'official_behavior',
  'behavior',
  'request_integrity',
  'multimodal',
  'gateway'
])

function dimensionCategoryLabel(category: string): string {
  return knownDimensionCategories.has(category) ? t(`purity.detail.categories.${category}`) : t('purity.detail.unknown')
}

const knownDimensionModes = new Set([
  'identity',
  'stream',
  'non_stream',
  'provider_native',
  'provider_constraint',
  'channel_evidence',
  'behavior_and_provenance',
  'official_behavior',
  'encrypted_reasoning_behavior',
  'json_schema',
  'client_tool',
  'synthetic_image',
  'evidence_only'
])

function dimensionModeLabel(mode?: string): string {
  if (!mode) return '-'
  return knownDimensionModes.has(mode) ? t(`purity.detail.modes.${mode}`) : t('purity.detail.unknown')
}

const knownLimitationCodes = new Set([
  'managed_websearch_unsupported',
  'managed_websearch_unsupported_by_bedrock',
  'anthropic_managed_websearch_not_applicable',
  'gateway_fingerprint_not_protocol_score',
  'active_probe_not_implemented',
  'versioned_knowledge_probe_not_run',
  'synthetic_document_probe_not_run',
  'signature_probe_not_applicable',
  'structured_output_probe_not_run',
  'managed_websearch_not_probed'
])

function limitationCodeLabel(code: string): string {
  return knownLimitationCodes.has(code) ? t(`purity.detail.limitationCodes.${code}`) : t('purity.detail.unknown')
}

function dimensionLimitationsLabel(limitations?: string[]): string {
  return limitations?.length ? limitations.map(limitationCodeLabel).join(' ') : '-'
}

const knownCheckNames = new Set([
  'base_url',
  'models_schema',
  'responses_schema',
  'responses_structured_output',
  'responses_store_include',
  'tool_call',
  'usage',
  'streaming',
  'multimodal',
  'chat_completions',
  'chat_completions_n',
  'model_identity',
  'channel_attribution',
  'wrapper_fingerprint',
  'token_audit',
  'claude_messages_schema',
  'claude_tool_use',
  'claude_usage',
  'claude_streaming',
  'claude_multimodal',
  'claude_signature_provenance',
  'claude_thinking_signature',
  'claude_thinking_budget',
  'claude_cache_control_overflow'
])

function checkDisplayName(check: PurityCheckResult): string {
  return knownCheckNames.has(check.id) ? t(`purity.detail.checkNames.${check.id}`) : check.id
}

function checkDisplayMessage(check: PurityCheckResult): string {
  return t(`purity.detail.checkMessage.${check.status}`)
}

function scoreDimensionLabel(id: string): string {
  const known = new Set(['tag_check', 'structure', 'behavior', 'signature_proto', 'multimodal', 'websearch', 'fingerprint', 'token_audit'])
  return known.has(id) ? t(`purity.detail.scoreDimensions.${id}`) : id
}

function dimensionStatusLabel(status: PurityDimensionStatus): string {
  return t(`purity.detail.dimensionStatus.${status}`)
}

function dimensionStatusClass(status: PurityDimensionStatus): string {
  if (status === 'pass') return 'text-emerald-600 dark:text-emerald-300'
  if (status === 'warn' || status === 'unsupported_by_upstream') return 'text-amber-600 dark:text-amber-300'
  if (status === 'fail') return 'text-red-600 dark:text-red-300'
  return 'text-gray-500 dark:text-dark-400'
}

function dimensionScoreLabel(dimension: PurityDimensionResult): string {
  return dimension.scored ? `${dimension.score}/${dimension.max_score}` : t('purity.detail.unscored')
}

function sourceChecksForDimension(dimension: PurityDimensionResult): PurityCheckResult[] {
  const sourceIDs = new Set(dimension.source_check_ids || [])
  return reportChecks.value.filter((check) => sourceIDs.has(check.id))
}

function validationStatusLabel(status: DisplayStatus): string {
	if (status === 'pass') return t('purity.status.pass')
	if (status === 'warn') return t('purity.status.warn')
	if (status === 'fail') return t('purity.status.fail')
	if (status === 'running') return t('purity.status.running')
	return t('purity.status.idle')
}

function scoreRingColor(score: number): string {
  if (score >= 85) return '#10b981'
  if (score >= 60) return '#f59e0b'
  return '#ef4444'
}

function validationIcon(status: DisplayStatus): IconName {
  if (status === 'pass') return 'checkCircle'
  if (status === 'warn') return 'exclamationTriangle'
  if (status === 'fail') return 'xCircle'
  if (status === 'running') return 'refresh'
  return 'clock'
}

function validationCardClass(status: DisplayStatus): string {
  if (status === 'pass') return 'border-emerald-200 dark:border-emerald-500/40'
  if (status === 'warn') return 'border-amber-200 dark:border-amber-500/40'
  if (status === 'fail') return 'border-red-200 dark:border-red-500/40'
  if (status === 'running') return 'border-primary-200 dark:border-primary-500/40'
  return 'border-gray-200 dark:border-dark-500'
}

function validationIconClass(status: DisplayStatus): string {
  if (status === 'pass') return 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/25 dark:text-emerald-300'
  if (status === 'warn') return 'bg-amber-50 text-amber-600 dark:bg-amber-900/25 dark:text-amber-300'
  if (status === 'fail') return 'bg-red-50 text-red-600 dark:bg-red-900/25 dark:text-red-300'
  if (status === 'running') return 'bg-primary-50 text-primary-600 dark:bg-primary-900/25 dark:text-primary-300'
  return 'bg-gray-100 text-gray-400 dark:bg-dark-600 dark:text-dark-400'
}

function validationBadgeClass(status: DisplayStatus): string {
  if (status === 'pass') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'warn') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (status === 'fail') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (status === 'running') return 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
  return 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-dark-300'
}

function checkStatusClass(status: PurityCheckStatus): string {
  if (status === 'pass') return 'text-emerald-600 dark:text-emerald-300'
  if (status === 'warn') return 'text-amber-600 dark:text-amber-300'
  return 'text-red-600 dark:text-red-300'
}

function auditToneTextClass(tone: TokenAuditTone): string {
  if (tone === 'good') return 'text-emerald-600 dark:text-emerald-300'
  if (tone === 'warn') return 'text-amber-600 dark:text-amber-300'
  if (tone === 'bad') return 'text-red-600 dark:text-red-300'
  return 'text-gray-900 dark:text-gray-100'
}

function auditValueTextClass(tone: TokenAuditTone): string {
  if (tone === 'good') return 'text-emerald-600 dark:text-emerald-300'
  if (tone === 'warn') return 'text-amber-600 dark:text-amber-300'
  if (tone === 'bad') return 'text-red-600 dark:text-red-300'
  return 'text-gray-950 dark:text-gray-100'
}

function auditToneBadgeClass(tone: TokenAuditTone): string {
  if (tone === 'good') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/40 dark:bg-emerald-900/25 dark:text-emerald-300'
  if (tone === 'warn') return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/40 dark:bg-amber-900/25 dark:text-amber-300'
  if (tone === 'bad') return 'border-red-200 bg-red-50 text-red-700 dark:border-red-500/40 dark:bg-red-900/25 dark:text-red-300'
  return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-500 dark:bg-dark-600 dark:text-dark-300'
}

function auditToneNoticeClass(tone: TokenAuditTone): string {
  if (tone === 'good') return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/40 dark:bg-emerald-900/20 dark:text-emerald-200'
  if (tone === 'warn') return 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200'
  if (tone === 'bad') return 'border-red-200 bg-red-50 text-red-800 dark:border-red-500/40 dark:bg-red-900/20 dark:text-red-200'
  return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-500 dark:bg-dark-600 dark:text-dark-300'
}

function auditBarClass(tone: TokenAuditTone): string {
  if (tone === 'good') return 'bg-emerald-400'
  if (tone === 'warn') return 'bg-amber-400'
  if (tone === 'bad') return 'bg-red-400'
  return 'bg-primary-500'
}

function latencyLabel(value?: number): string {
  if (!value || value < 0) return '-'
  return `${Math.round(value)} ms`
}

function formatMultiplier(value?: number): string {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) return '-'
  return `${value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')}x`
}

function formatUSD(value?: number): string {
  if (!value) return '$0'
  return `$${value.toFixed(6).replace(/0+$/, '').replace(/\.$/, '.0')}`
}

function formatPercent(value?: number): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `${Math.round(value > 1 ? value : value * 100)}%`
}

function deltaLabel(value?: number): string {
  if (!value) return ''
  const abs = Math.abs(Math.round(value))
  return `${value > 0 ? '↑' : '↓'}${abs > 999 ? '>999' : abs}%`
}

function deltaTextClass(value?: number): string {
  if (!value) return ''
  return value > 0 ? 'text-red-600 dark:text-red-300' : 'text-emerald-600 dark:text-emerald-300'
}

function normalizeAccountProvider(platform?: string): PurityProvider | null {
  const value = normalizeTokenAuditProvider(platform)
  if (value === 'openai' || value === 'anthropic' || value === 'gemini') return value
  return null
}

function normalizeProgress(value?: number): number {
  if (!value || value < 0) return 0
  if (value > 1) return Math.min(1, value / 100)
  return value
}

function findPreferredModel(models: LocalAccountTestModel[], candidates: string[]): string {
  for (const candidate of candidates) {
    const exact = models.find((model) => model.id === candidate)
    if (exact) return exact.id
    const fuzzy = models.find((model) => model.id.toLowerCase().includes(candidate))
    if (fuzzy) return fuzzy.id
  }
  return models[0]?.id || ''
}

function validationDisplayName(definition: ValidationDefinition): string {
  if (isGeminiProvider.value) {
    if (definition.id === 'schema_integrity') return `GenerateContent ${t('purity.ui.validationNames.schema_integrity')}`
    if (definition.id === 'multimodal') return `InlineData ${t('purity.ui.validationNames.multimodal')}`
    return t(`purity.ui.validationNames.${definition.id}`)
  }
  if (currentProvider.value === 'anthropic' && definition.id === 'schema_integrity') return `Messages ${t('purity.ui.validationNames.schema_integrity')}`
  if (currentProvider.value === 'anthropic' && definition.id === 'multimodal') return `Image Block ${t('purity.ui.validationNames.multimodal')}`
  return t(`purity.ui.validationNames.${definition.id}`)
}

function validationWaitingMessage(definition: ValidationDefinition): string {
  if (isGeminiProvider.value) {
    if (definition.id === 'schema_integrity') return `GenerateContent: ${t('purity.ui.validationWaiting.schema_integrity')}`
    if (definition.id === 'multimodal') return `inlineData: ${t('purity.ui.validationWaiting.multimodal')}`
    return t(`purity.ui.validationWaiting.${definition.id}`)
  }
  if (currentProvider.value === 'anthropic' && definition.id === 'schema_integrity') return `Messages: ${t('purity.ui.validationWaiting.schema_integrity')}`
  if (currentProvider.value === 'anthropic' && definition.id === 'multimodal') return `image block: ${t('purity.ui.validationWaiting.multimodal')}`
  return t(`purity.ui.validationWaiting.${definition.id}`)
}

function modelIdentityReasonLabel(reason?: string): string {
	const reasonKeys: Record<string, string> = {
		exact_match: 'purity.modelIdentity.exactMatch',
		compatible_alias: 'purity.modelIdentity.compatibleAlias',
		response_model_missing: 'purity.modelIdentity.responseModelMissing',
		probe_model_fallback: 'purity.modelIdentity.probeModelFallback',
		cross_vendor_alias: 'purity.modelIdentity.crossVendorAlias',
		family_mismatch: 'purity.modelIdentity.familyMismatch',
		version_downgrade: 'purity.modelIdentity.versionDowngrade',
		tier_downgrade: 'purity.modelIdentity.tierDowngrade',
		protocol_model_vendor_mismatch: 'purity.modelIdentity.protocolVendorMismatch',
		wrapper_vendor_signal_mismatch: 'purity.modelIdentity.wrapperVendorMismatch',
		reasoning_tokens_mismatch: 'purity.modelIdentity.reasoningTokensMismatch'
	}
	if (reason && reasonKeys[reason]) return t(reasonKeys[reason])
	return reason || t('purity.modelIdentity.completed')
}

function responseModelDescription(identity?: PurityReport['model_identity'] | PurityReport['modelIdentity'] | null): string {
	const parts: string[] = []
	if (identity?.response_vendor) parts.push(`${t('purity.evidence.responseVendor')}：${identity.response_vendor}`)
	const source = report.value?.response_model_source || report.value?.responseModelSource
	if (source) parts.push(`${t('purity.evidence.responseSource')}：${source}`)
	return parts.length ? parts.join('；') : t('purity.evidence.responseModelPending')
}

function modelIdentityEvidenceDescription(identity: NonNullable<PurityReport['model_identity'] | PurityReport['modelIdentity']>): string {
	const suspectedVendor = suspectedUpstreamVendor(identity)
	const reason = modelIdentityReasonLabel(identity.reason)
	return suspectedVendor ? `${reason}；${t('purity.evidence.suspectedUpstreamVendor')}：${suspectedVendor}` : reason
}

function suspectedUpstreamVendor(identity?: PurityReport['model_identity'] | PurityReport['modelIdentity'] | null): string {
  const value = identity?.evidence?.suspected_upstream_vendor
  return typeof value === 'string' ? value : ''
}

function sortAuditSamples(samples: PurityTokenAuditSample[]): PurityTokenAuditSample[] {
  return [...samples].sort((a, b) => a.index - b.index)
}

function hasAuditSampleData(sample: PurityTokenAuditSample): boolean {
  return hasTokenAuditSampleData(sample)
}

function tokenAuditSampleRow(sample: PurityTokenAuditSample) {
  return tokenAuditSampleDisplayRow(sample, currentProvider.value || 'openai')
}

function tokenAuditSampleRatioCell(sample: PurityTokenAuditSample): { display: string; tone: TokenAuditTone; title: string } {
  return tokenAuditSampleRatioDisplayCell(sample, currentProvider.value || 'openai', tokenAuditBillingMultiplier.value)
}

function auditRequestModeLabel(mode?: string): string {
  if (mode === 'cache_probe') return '缓存'
  if (mode === 'stateful') return '状态'
  if (mode === 'context_replay') return '上下文'
  if (mode === 'minimal_retry') return '重试'
  if (mode === 'history_replay') return '历史'
  if (mode === 'gemini_history_replay') return 'Gemini 历史'
  if (mode === 'chat_completions') return 'Chat'
  return '-'
}

function auditSampleFailureReason(sample: PurityTokenAuditSample): string {
  if (sample.status === 'pass' && !sample.error_class && !sample.error_message) return ''
  const parts: string[] = []
  if (sample.status_code && (sample.status_code < 200 || sample.status_code >= 300)) parts.push(`HTTP ${sample.status_code}`)
  if (sample.error_class) parts.push(sample.error_class)
  if (sample.error_message) parts.push(shortAuditFailureText(sample.error_message))
  if (!parts.length && sample.status && sample.status !== 'pass') parts.push(sample.status)
  return parts.join(' · ')
}

function auditSampleFailureSummary(sample: PurityTokenAuditSample): string {
  const reason = auditSampleFailureReason(sample)
  return reason ? shortAuditFailureText(reason) : ''
}

function auditSampleFailureTitle(sample: PurityTokenAuditSample): string {
  const parts: string[] = []
  if (sample.status_code && (sample.status_code < 200 || sample.status_code >= 300)) parts.push(`HTTP ${sample.status_code}`)
  if (sample.error_class) parts.push(sample.error_class)
  if (sample.error_message) parts.push(sample.error_message.trim())
  if (!parts.length && sample.status && sample.status !== 'pass') parts.push(sample.status)
  return parts.join(' · ')
}

function shortAuditFailureText(value: string): string {
  const text = value.trim()
  return text.length <= 120 ? text : `${text.slice(0, 120)}...`
}

function buildPDFReportSnapshot(): PurityReport | null {
  if (!props.account) return report.value
  if (report.value) {
    return {
      ...report.value,
      metrics: report.value.metrics || metrics.value,
      scores: report.value.scores || scores.value,
      checks: report.value.checks?.length ? report.value.checks : checks.value,
      validations: report.value.validations?.length ? report.value.validations : Object.values(validations.value),
      token_audit: report.value.token_audit || tokenAudit.value || undefined,
      token_audit_partial: report.value.token_audit_partial || auditSamples.value,
      api_base_host: report.value.api_base_host || props.account.name
    }
  }
  if (!started.value) return null
  const provider = currentProvider.value || 'openai'
  return {
    provider,
    report_id: `account-${props.account.id}-${Date.now()}`,
    access_mode: 'account',
    billing_mode: 'account_internal',
    api_base_host: props.account.name,
    model_id: selectedModelId.value || '-',
    check_token_usage: checkTokenUsage.value,
    expected_model: selectedModelId.value || undefined,
    status: runStatus.value,
    step_name: stepName.value,
    progress: progress.value,
    scores: scores.value,
    score: typeof displayScore.value === 'number' ? displayScore.value : 0,
    official_score: 0,
    compatibility_score: 0,
    verdict: 'unknown',
    summary: runningSummary.value,
    error: errorMessage.value || undefined,
    validations: Object.values(validations.value),
    checks: checks.value,
    metrics: metrics.value,
    token_audit: tokenAudit.value || undefined,
    token_audit_progress: tokenAuditProgress.value || undefined,
    token_audit_partial: auditSamples.value,
    checked_at: new Date().toISOString()
  }
}
</script>

<style scoped>
.score-ring {
  display: grid;
  width: 128px;
  height: 128px;
  place-items: center;
  border-radius: 9999px;
  background: conic-gradient(var(--score-color, #14b8a6) var(--score-angle), #e5e7eb 0);
}

.score-ring-inner {
  display: grid;
  width: 96px;
  height: 96px;
  place-items: center;
  border-radius: 9999px;
  background: #fff;
}

:global(.dark) .score-ring {
  background: conic-gradient(var(--score-color, #2dd4bf) var(--score-angle), #374151 0);
}

:global(.dark) .score-ring-inner {
  background: #1f2937;
}
</style>
