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
            :placeholder="loadingModels ? '加载中...' : '选择模型'"
            empty-text="暂无模型"
          />
          <button type="button" class="btn btn-primary btn-sm" :disabled="runStatus === 'running' || !selectedModelId || !isSupportedAccount" @click="startCheck">
            <Icon v-if="runStatus === 'running'" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <Icon v-else name="play" size="sm" :stroke-width="2" />
            <span>{{ runStatus === 'running' ? '检测中' : '开始检测' }}</span>
          </button>
        </div>
      </div>

      <div v-if="!isSupportedAccount" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200">
        仅支持 OpenAI 或 Claude API Key 账号执行纯度检测。
      </div>
      <div v-if="fatalReportError" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-500/40 dark:bg-red-900/20 dark:text-red-200">
        <div class="font-semibold">检测失败</div>
        <div class="mt-1 break-words">{{ fatalReportError }}</div>
      </div>
      <div v-else-if="probeIssueMessage" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200">
        <div class="font-semibold">部分探针异常</div>
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
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ report?.summary || runningSummary }}</div>
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
              <div class="text-gray-500 dark:text-dark-400">兼容分</div>
            </div>
            <div class="rounded-md bg-white p-2 dark:bg-dark-600">
              <div class="font-semibold text-gray-900 dark:text-gray-100">{{ report?.official_score ?? '-' }}</div>
              <div class="text-gray-500 dark:text-dark-400">官方分</div>
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
                <span class="text-xs font-semibold text-gray-900 dark:text-gray-100">{{ item.value }}/{{ item.max }}</span>
              </div>
              <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-600">
                <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: `${Math.round((item.value / item.max) * 100)}%` }" />
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

      <div class="grid gap-3 lg:grid-cols-[1fr_320px]">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-500 dark:bg-dark-700">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">Token 用量审计</div>
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
            <span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-gray-400 dark:bg-dark-400" />官方基线</span>
            <span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-emerald-400" />Usage 估算</span>
          </div>
          <div class="mt-3 overflow-x-auto rounded-md border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-800">
            <table class="min-w-[780px] table-fixed text-left text-xs text-gray-950 dark:text-gray-100">
              <thead class="text-gray-700 dark:text-dark-300">
                <tr>
                  <th class="w-12 py-1 pr-2 font-medium">轮次</th>
                  <th class="w-24 py-1 pr-2 font-medium">模式</th>
                  <th class="w-16 py-1 pr-2 font-medium text-right">耗时</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">输入</th>
                  <th class="w-20 py-1 pr-2 font-medium text-right">输出</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">缓存创建</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">缓存读取</th>
                  <th class="w-24 py-1 pr-2 font-medium text-right">Usage 估算</th>
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
          <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">检测明细</div>
          <div class="mt-3 max-h-[300px] space-y-2 overflow-y-auto pr-1">
            <div v-for="check in reportChecks" :key="check.id" class="rounded-md bg-gray-50 p-2 dark:bg-dark-600">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-medium text-gray-800 dark:text-gray-100">{{ check.name }}</span>
                <span class="text-xs" :class="checkStatusClass(check.status)">{{ check.score }}/{{ check.max_score }}</span>
              </div>
              <div class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ check.message }}</div>
            </div>
            <div v-if="reportChecks.length === 0" class="rounded-md bg-gray-50 p-4 text-center text-xs text-gray-400 dark:bg-dark-600 dark:text-dark-400">
              等待后端探针结果
            </div>
          </div>
        </div>
      </div>

      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-500/40 dark:bg-red-900/20 dark:text-red-200">
        {{ errorMessage }}
      </div>
    </div>

    <template #footer>
      <div class="flex flex-wrap justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">关闭</button>
        <button type="button" class="btn btn-secondary" :disabled="!canDownloadPDF || runStatus === 'running'" @click="downloadPDF">
          <Icon name="download" size="sm" :stroke-width="2" />
          <span>下载 PDF</span>
        </button>
        <button type="button" class="btn btn-primary" :disabled="runStatus === 'running' || !selectedModelId || !isSupportedAccount" @click="startCheck">
          <Icon v-if="runStatus === 'running'" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
          <Icon v-else name="shield" size="sm" :stroke-width="2" />
          <span>{{ runStatus === 'running' ? '检测中' : report ? '重新检测' : '开始检测' }}</span>
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
  type PurityProvider,
  type PurityReport,
  type PurityScoreBreakdown,
  type PurityTokenAuditReport,
  type PurityTokenAuditSample,
  type PurityValidationResult
} from '@/api/admin/adminPlus'
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
        }
      }
    },
    en: {
      purity: {
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

const stepLabels: Record<string, string> = {
  tag: 'LLM 指纹验证',
  structure: '结构完整性',
  behavior: '行为验证',
  signature: '签名校验',
  multimodal: '多模态能力',
  token_audit: 'Token 用量审计',
  evaluate: '最终评估'
}

const activeValidationByStep: Record<string, string> = {
  tag: 'llm_fingerprint',
  structure: 'schema_integrity',
  behavior: 'behavior',
  signature: 'signature',
  multimodal: 'multimodal',
  token_audit: 'token_audit',
  evaluate: 'model_identity'
}

const scoreDefinitions: Array<{ key: ScoreBreakdownKey; label: string; max: number }> = [
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
const dialogTitle = computed(() => `${providerLabel.value} API 纯度检测`)
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
const stepLabel = computed(() => stepLabels[currentStepName.value] || (started.value ? '准备检测' : '等待开始'))
const progressPercent = computed(() => {
  const value = normalizeProgress(report.value?.progress ?? progress.value)
  return Math.round(value * 100)
})
const runningSummary = computed(() => runStatus.value === 'running' ? `后端探针正在执行：${stepLabel.value}` : '尚未开始检测')
const currentRunningValidation = computed(() => activeValidationByStep[currentStepName.value] || '')
const fatalReportError = computed(() => {
  if (report.value?.status === 'error' || report.value?.error) {
    return report.value?.error || metrics.value.error_message || '检测失败'
  }
  return ''
})
const probeIssueMessage = computed(() => {
  if (fatalReportError.value || !metrics.value.error_message) return ''
  return metrics.value.error_message
})
const displayedValidations = computed<DisplayValidation[]>(() => validationDefinitions.map((definition) => {
  const result = validations.value[definition.id]
  if (result) {
    return {
      id: definition.id,
      name: result.name || definition.name,
      status: result.status as DisplayStatus,
      message: result.message || definition.message
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
  return scoreDefinitions.map((definition) => {
    const rawValue = source[definition.key] ?? 0
    const value = Math.max(0, Math.min(definition.max, rawValue))
    return { ...definition, value }
  })
})
const validAuditSamples = computed(() => normalizedAuditSamples().filter(hasAuditSampleData))
const failedAuditSampleCount = computed(() => normalizedAuditSamples().filter((sample) => !hasAuditSampleData(sample)).length)
const auditSamplesForChart = computed(() => validAuditSamples.value)
const auditSamplesForTable = computed(() => normalizedAuditSamples())
const reportChecks = computed<PurityCheckResult[]>(() => (report.value?.checks?.length ? report.value.checks : checks.value))
const tokenAuditSummary = computed(() => {
  if (tokenAudit.value) return `${tokenAudit.value.summary} · ${auditSamplesForTable.value.length}/${tokenAudit.value.sample_count || 11}${failedAuditSampleCount.value > 0 ? ` · ${failedAuditSampleCount.value} 轮仅返回诊断` : ''}`
  if (auditSamples.value.length > 0) return `采集中 · ${tokenAuditProgress.value || `${auditSamples.value.length}/11`}`
  return started.value ? '等待样本' : '尚未开始'
})
const emptyAuditTableText = computed(() => failedAuditSampleCount.value > 0 ? '等待失败诊断样本' : '等待审计样本')
const tokenAuditMetricCards = computed(() => {
  const audit = tokenAudit.value
  const totals = tokenAuditCostTotals(audit)
  const ratio = tokenAuditDisplayRatio(audit)
  const billingMultiplier = audit?.billing_multiplier ?? audit?.billingMultiplier
  const hasBillingMultiplier = typeof billingMultiplier === 'number' && Number.isFinite(billingMultiplier)
  const cards = [
    { label: '官方基线', value: formatUSD(totals.officialBaselineUSD), tone: 'neutral' as TokenAuditTone },
    { label: 'Usage 估算', value: formatUSD(totals.actualCostUSD), tone: ratio > 0 ? multiplierTone(ratio) : 'neutral' as TokenAuditTone },
    hasBillingMultiplier
      ? { label: '平台计费倍率', value: formatMultiplier(billingMultiplier), tone: 'good' as TokenAuditTone }
      : { label: '平台计费倍率', value: '-', tone: 'neutral' as TokenAuditTone },
    { label: 'Usage 比值', value: formatMultiplier(ratio), tone: multiplierTone(ratio) },
    { label: '缓存命中率', value: formatPercent(audit?.cacheHitRate ?? audit?.cache_hit_rate), tone: audit?.cacheHitRate || audit?.cache_hit_rate ? 'good' as TokenAuditTone : 'neutral' as TokenAuditTone }
  ]
  return cards
})
const tokenAuditRatio = computed(() => tokenAuditDisplayRatio(tokenAudit.value))
const tokenAuditRatioTone = computed(() => multiplierTone(tokenAuditRatio.value))
const tokenAuditBillingMultiplier = computed(() => tokenAudit.value?.billing_multiplier ?? tokenAudit.value?.billingMultiplier)
const hasTokenAuditBillingMultiplier = computed(() => typeof tokenAuditBillingMultiplier.value === 'number' && Number.isFinite(tokenAuditBillingMultiplier.value))
const tokenAuditSampleRatioHeader = computed(() => hasTokenAuditBillingMultiplier.value ? '平台倍率' : 'Usage 比值')
const geminiCacheFieldNotice = computed(() => {
  if (!isGeminiProvider.value) return ''
  const samples = auditSamplesForTable.value
  if (!samples.length) return ''
  const missingCacheCreate = samples.some((sample) => tokenAuditSampleRow(sample).cacheCreation.available === false)
  const missingCacheRead = samples.some((sample) => tokenAuditSampleRow(sample).cacheRead.available === false)
  if (!missingCacheCreate && !missingCacheRead) return ''
  return 'Gemini 未返回或未命中的缓存字段以 0 展示；缓存创建字段不可确认，缓存读取 0 表示本轮未观察到命中。'
})
const tokenAuditNoticeText = computed(() => {
  const ratio = tokenAuditRatio.value
  const billingMultiplier = tokenAuditBillingMultiplier.value
  const cacheNotice = geminiCacheFieldNotice.value ? ` ${geminiCacheFieldNotice.value}` : ''
  if (typeof billingMultiplier === 'number' && Number.isFinite(billingMultiplier)) {
    return `平台计费倍率 ${formatMultiplier(billingMultiplier)}；Usage 比值 ${formatMultiplier(ratio)}。两者口径不同，前者来自账号配置或 /v1/usage 扣费增量。${cacheNotice}`
  }
  if (isGeminiProvider.value) return `Usage 比值 ${formatMultiplier(ratio)}；Gemini usageMetadata 只能确认本轮 token 统计，平台计费倍率需结合账号配置或 /v1/usage 扣费增量。${cacheNotice}`
  if (!ratio) return 'Usage 比值暂无法确认，需结合每轮 usage 字段和平台账单复核。'
  if (tokenAuditRatioTone.value === 'bad') return `Usage 比值 ${formatMultiplier(ratio)}，明显高于常见范围，可能存在异常扣费或 Token 统计混淆。`
  if (tokenAuditRatioTone.value === 'warn') return `Usage 比值 ${formatMultiplier(ratio)}，高于常见范围，建议结合平台单价/倍率复核。`
  return `Usage 比值 ${formatMultiplier(ratio)}，当前未发现明显超额消耗；平台计费倍率需结合账号配置或账单复核。${cacheNotice}`
})
const tokenAuditNoticeClass = computed(() => {
  if (tokenAudit.value?.status === 'fail') return auditToneNoticeClass('bad')
  if (tokenAudit.value?.status === 'warn') return auditToneNoticeClass(tokenAuditRatioTone.value === 'neutral' ? 'warn' : tokenAuditRatioTone.value)
  return auditToneNoticeClass(tokenAuditRatioTone.value)
})
const tokenAuditStatusLabel = computed(() => validationStatusLabel((tokenAudit.value?.status || (auditSamples.value.length > 0 ? 'running' : 'idle')) as DisplayStatus))
const tokenAuditBadgeClass = computed(() => validationBadgeClass((tokenAudit.value?.status || (auditSamples.value.length > 0 ? 'running' : 'idle')) as DisplayStatus))
const metricCards = computed(() => [
  { label: '模型列表', value: latencyLabel(metrics.value.models_latency_ms) },
  {
    label: currentProvider.value === 'anthropic' ? 'Messages' : isGeminiProvider.value ? 'GenerateContent' : 'Responses',
    value: latencyLabel(currentProvider.value === 'anthropic' ? metrics.value.messages_latency_ms : isGeminiProvider.value ? metrics.value.generate_content_latency_ms || metrics.value.responses_latency_ms : metrics.value.responses_latency_ms)
  },
  { label: '首 Token', value: latencyLabel(metrics.value.stream_first_token_ms) },
  { label: '总耗时', value: latencyLabel(metrics.value.latency_ms) }
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
    errorMessage.value = (error as { message?: string }).message || '加载模型失败'
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
      model_id: selectedModelId.value
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
      throw new Error('响应体为空')
    }
    await readNDJSON(response.body)
    if (runStatus.value === 'running') runStatus.value = 'success'
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      runStatus.value = 'idle'
      return
    }
    runStatus.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : '检测失败'
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
    errorMessage.value = `无法解析检测事件: ${trimmed.slice(0, 120)}`
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
      errorMessage.value = event.error_message || '检测失败'
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
    if (definition.id === 'schema_integrity') return 'GenerateContent 结构完整性'
    if (definition.id === 'multimodal') return 'InlineData 多模态'
    return definition.name
  }
  if (currentProvider.value !== 'anthropic') return definition.name
  if (definition.id === 'schema_integrity') return 'Messages 结构完整性'
  if (definition.id === 'multimodal') return 'Image Block 多模态'
  return definition.name
}

function validationWaitingMessage(definition: ValidationDefinition): string {
  if (isGeminiProvider.value) {
    if (definition.id === 'schema_integrity') return '等待 GenerateContent schema 探测'
    if (definition.id === 'multimodal') return '等待 inlineData 探测'
    return definition.message
  }
  if (currentProvider.value !== 'anthropic') return definition.message
  if (definition.id === 'schema_integrity') return '等待 Messages schema 探测'
  if (definition.id === 'multimodal') return '等待 image block 探测'
  return definition.message
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
