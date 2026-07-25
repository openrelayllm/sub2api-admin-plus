import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LocalAccountPurityModal from './LocalAccountPurityModal.vue'

const apiMocks = vi.hoisted(() => ({
  listLocalAccountTestModels: vi.fn(),
  localAccountPurityStreamURL: vi.fn(() => '/api/purity/account/7')
}))

vi.mock('@/api/admin/adminPlus', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/admin/adminPlus')>()),
  ...apiMocks
}))

vi.mock('@/utils/purityPdf', () => ({
  downloadPurityReportPDF: vi.fn()
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: (options: { messages?: Record<string, Record<string, unknown>> }) => ({
    t: (key: string, values?: Record<string, string | number>) => {
      const locale = localStorage.getItem('sub2api_locale') === 'en' ? 'en' : 'zh'
      const message = key.split('.').reduce<unknown>((current, part) => {
        if (!current || typeof current !== 'object') return undefined
        return (current as Record<string, unknown>)[part]
      }, options.messages?.[locale])
      const text = typeof message === 'string' ? message : key
      return Object.entries(values || {}).reduce((result, [name, value]) => result.replaceAll(`{${name}}`, String(value)), text)
    }
  })
}))

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  apiMocks.listLocalAccountTestModels.mockReset()
  document.body.innerHTML = ''
  localStorage.clear()
})

describe('LocalAccountPurityModal', () => {
  it('keeps token audit opt-in and detailed dimensions collapsed by default', async () => {
    apiMocks.listLocalAccountTestModels.mockResolvedValue([{ id: 'gpt-5.5', display_name: 'GPT-5.5' }])
    const report = {
      provider: 'openai',
      report_id: 'report-admin-ui',
      access_mode: 'account',
      billing_mode: 'account_internal',
      api_base_host: 'api.openai.com',
      model_id: 'gpt-5.5',
      expected_model: 'gpt-5.5',
      response_model: 'gpt-5.5',
      check_token_usage: false,
      status: 'done',
      progress: 1,
      score: 95,
      official_score: 95,
      compatibility_score: 98,
      protocol_score: 98,
      scores: {
        tag_check: 10,
        structure: 25,
        behavior: 35,
        signature_proto: 20,
        multimodal: 10
      },
      score_policy: {
        id: 'aws_bedrock_messages',
        channel: 'aws_bedrock',
        baseline: 'aws_bedrock_messages',
        dimensions: [
          { id: 'tag_check', validation_id: 'llm_fingerprint', max_score: 10, client_impact: 'breaking', failure_policy: 'full_dimension_deduction' },
          { id: 'structure', validation_id: 'schema_integrity', max_score: 25, client_impact: 'breaking', failure_policy: 'full_dimension_deduction' },
          { id: 'behavior', validation_id: 'behavior', max_score: 35, client_impact: 'breaking', failure_policy: 'full_dimension_deduction' },
          { id: 'signature_proto', validation_id: 'signature', max_score: 20, client_impact: 'breaking', failure_policy: 'full_dimension_deduction' },
          { id: 'multimodal', validation_id: 'multimodal', max_score: 10, client_impact: 'limited', failure_policy: 'full_dimension_deduction' }
        ],
        excluded_dimensions: ['websearch', 'fingerprint']
      },
      score_adjustments: [{
        id: 'bedrock_anthropic_signature_mask_penalty',
        category: 'provenance_transparency',
        reason_code: 'bedrock_anthropic_signature_mask',
        case_id: 'PURITY-BEDROCK-MASK-001',
        client_impact: 'none',
        impact_scope: 'channel_attribution_only',
        base_score: 100,
        points: -5,
        result_score: 95,
        evidence: ['bedrock_metadata_family_present', 'anthropic_native_metadata_present']
      }],
      verdict: 'official_openai',
      summary: 'backend summary',
      assessment: {
        kind: 'official_native',
        status: 'ready',
        title: 'backend title',
        summary: 'backend assessment summary',
        channel: 'openai_native',
        channel_status: 'identified',
        channel_confidence: 0.98,
        identity_status: 'pass',
        protocol_status: 'high',
        wrapper_mode: 'none',
        metering_status: 'not_tested',
        dimension_total: 12,
        dimension_executed: 1,
        dimension_scored: 1,
        limitations: [],
        reason_codes: ['token_audit_not_requested']
      },
      dimension_matrix: [
        {
          id: 'tag_check',
          name: 'backend dimension name',
          category: 'identity',
          status: 'pass',
          score: 10,
          max_score: 10,
          scored: true,
          message: 'identity matched',
          source_check_ids: ['model_identity'],
          limitations: [],
          details: { status_code: 200, api_key: 'sk-dimension-secret' }
        }
      ],
      validations: [],
      checks: [
        {
          id: 'model_identity',
          name: 'Model identity',
          status: 'pass',
          score: 10,
          max_score: 10,
          message: 'matched',
          details: { response_model: 'gpt-5.5', authorization: 'Bearer secret-value' }
        }
      ],
      metrics: {},
      checked_at: '2026-07-25T00:00:00Z'
    }
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect(JSON.parse(String(init?.body))).toMatchObject({
        provider: 'openai',
        model_id: 'gpt-5.5',
        check_token_usage: false
      })
      return new Response(`${JSON.stringify({ type: 'report', report_id: report.report_id, report })}\n`, {
        headers: { 'Content-Type': 'application/x-ndjson' }
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mount(LocalAccountPurityModal, {
      attachTo: document.body,
      props: {
        show: false,
        account: {
          id: 7,
          name: 'OpenAI test account',
          platform: 'openai',
          type: 'apikey',
          status: 'active',
          schedulable: true,
          concurrency: 1,
          priority: 1,
          rate_multiplier: 1,
          auto_pause_on_expired: false,
          created_at: '2026-07-25T00:00:00Z',
          updated_at: '2026-07-25T00:00:00Z'
        }
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="item in options" :key="item.id" :value="item.id">{{ item.display_name }}</option></select>'
          },
          Icon: { template: '<span />' }
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(apiMocks.listLocalAccountTestModels).toHaveBeenCalledWith(7)
    expect(wrapper.get('input[type="checkbox"]').element).toMatchObject({ checked: false })
    expect(wrapper.findAll('select option')).toHaveLength(1)
    await wrapper.get('select').setValue('gpt-5.5')
    expect((wrapper.get('select').element as HTMLSelectElement).value).toBe('gpt-5.5')
    expect(wrapper.text()).not.toContain('仅支持 OpenAI 或 Claude API Key 账号执行纯度检测')
    const startButton = wrapper.findAll('button').find((button) => button.text().includes('开始检测'))
    expect(startButton).toBeDefined()
    expect(startButton!.attributes('disabled')).toBeUndefined()
    await startButton!.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledOnce()
    const reportDetails = wrapper.get('[data-testid="purity-report-details"]')
    expect(reportDetails.attributes('open')).toBeUndefined()
    const dimensionDetails = wrapper.get('[data-dimension-id="tag_check"]')
    expect(dimensionDetails.attributes('open')).toBeUndefined()
    expect(wrapper.text()).toContain('原生官方渠道 · OpenAI API · gpt-5.5')
    expect(wrapper.text()).toContain('LLM 指纹验证')
    expect(wrapper.text()).toContain('10/10')
    expect(wrapper.text()).toContain('AWS Bedrock Messages 基线')
    expect(wrapper.text()).toContain('25/25')
    expect(wrapper.text()).toContain('35/35')
    expect(wrapper.text()).toContain('总分调整判例')
    expect(wrapper.text()).toContain('AWS Bedrock 来源伪装扣分')
    expect(wrapper.text()).toContain('100 -5 = 95')
    expect(wrapper.text()).toContain('不影响本轮客户端使用，仅影响来源透明度')
    expect(wrapper.text()).toContain('PURITY-BEDROCK-MASK-001')
    expect(wrapper.text()).not.toContain('默认未启用')
    expect(wrapper.text()).not.toContain('Token 用量审计未评分')
    expect(wrapper.text()).not.toContain('sk-dimension-secret')
    expect(wrapper.text()).not.toContain('Bearer secret-value')

    wrapper.unmount()
  })

  it('localizes backend dimension and check text in English', async () => {
    localStorage.setItem('sub2api_locale', 'en')
    apiMocks.listLocalAccountTestModels.mockResolvedValue([{ id: 'claude-opus-4-8', display_name: 'Claude Opus 4.8' }])
    const report = {
      provider: 'anthropic',
      report_id: 'report-admin-en',
      api_base_host: 'relay.example.com',
      model_id: 'claude-opus-4-8',
      expected_model: 'claude-opus-4-8',
      response_model: 'claude-opus-4-8',
      check_token_usage: false,
      status: 'done',
      progress: 1,
      score: 100,
      official_score: 100,
      compatibility_score: 100,
      protocol_score: 100,
      verdict: 'claude_compatible',
      summary: '后端中文摘要不应显示',
      assessment: {
        kind: 'official_cloud_channel',
        status: 'limited',
        channel: 'aws_bedrock',
        channel_status: 'identified',
        channel_confidence: 0.96,
        identity_status: 'pass',
        protocol_status: 'high',
        wrapper_mode: 'none',
        metering_status: 'not_tested',
        dimension_total: 12,
        dimension_executed: 1,
        dimension_scored: 0,
        limitations: ['WebSearch：上游不支持'],
        reason_codes: []
      },
      dimension_matrix: [{
        id: 'websearch',
        name: '后端 WebSearch',
        category: 'capability',
        status: 'unsupported_by_upstream',
        score: 0,
        max_score: 10,
        scored: false,
        message: 'AWS Bedrock 官方能力边界不支持 Anthropic 托管 WebSearch。',
        mode: 'provider_native',
        source_check_ids: ['claude_messages_schema'],
        limitations: ['managed_websearch_unsupported_by_bedrock'],
        details: {}
      }],
      validations: [],
      checks: [{
        id: 'claude_messages_schema',
        name: 'Messages 非流式结构',
        status: 'pass',
        score: 20,
        max_score: 20,
        message: '结构通过。',
        details: {}
      }],
      metrics: {},
      checked_at: '2026-07-25T00:00:00Z'
    }
    vi.stubGlobal('fetch', vi.fn(async () => new Response(`${JSON.stringify({ type: 'report', report_id: report.report_id, report })}\n`, {
      headers: { 'Content-Type': 'application/x-ndjson' }
    })))

    const wrapper = mount(LocalAccountPurityModal, {
      attachTo: document.body,
      props: {
        show: false,
        account: {
          id: 7,
          name: 'Claude test account',
          platform: 'anthropic',
          type: 'apikey',
          status: 'active',
          schedulable: true,
          concurrency: 1,
          priority: 1,
          rate_multiplier: 1,
          auto_pause_on_expired: false,
          created_at: '2026-07-25T00:00:00Z',
          updated_at: '2026-07-25T00:00:00Z'
        }
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="item in options" :key="item.id" :value="item.id">{{ item.display_name }}</option></select>'
          },
          Icon: { template: '<span />' }
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.get('select').setValue('claude-opus-4-8')
    const startButton = wrapper.findAll('button').find((button) => button.text().includes('Start check'))
    await startButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Official cloud channel · AWS Bedrock · claude-opus-4-8')
    expect(wrapper.text()).toContain('This capability is outside the upstream channel\'s official support boundary.')
    expect(wrapper.text()).toContain('AWS Bedrock does not provide Anthropic managed WebSearch.')
    expect(wrapper.text()).toContain('Messages non-stream schema')
    expect(wrapper.text()).not.toMatch(/后端中文|官方能力边界|结构通过|managed_websearch_unsupported_by_bedrock/)

    wrapper.unmount()
  })
})
