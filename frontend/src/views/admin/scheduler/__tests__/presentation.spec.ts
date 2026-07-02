import { describe, expect, it } from 'vitest'
import type { SchedulerStepRecord } from '@/api/admin/adminPlus'
import { stepDiagnosticSummary, stepHasDiagnostics, stepRawDiagnostics } from '../presentation'

describe('scheduler presentation diagnostics', () => {
  it('uses attempt logs when failed step reason is empty', () => {
    const step: SchedulerStepRecord = {
      id: 33192,
      run_id: 'plan-supplier.costs.reconcile-test',
      supplier_id: 12,
      supplier_name: '登录 - 何意味',
      task_type: 'reconcile_supplier_costs',
      action: 'sync_costs',
      status: 'retryable_failed',
      schedule_key: 'scheduler:supplier.costs.reconcile:supplier:12:test',
      result_count: 0,
      attempts: 1,
      max_attempts: 3,
      operation_logs: [
        {
          id: 1,
          step_id: 33192,
          run_id: 'plan-supplier.costs.reconcile-test',
          supplier_id: 12,
          task_type: 'reconcile_supplier_costs',
          status: 'retryable_failed',
          attempt_no: 1,
          finished_at: '2026-07-02T00:44:05Z',
          duration_ms: 502,
          error_code: 'SUPPLIER_SESSION_BAD_STATUS',
          error_message: 'supplier session endpoint returned non-success status'
        }
      ]
    }

    expect(stepHasDiagnostics(step)).toBe(true)
    expect(stepDiagnosticSummary(step)).toBe('SUPPLIER_SESSION_BAD_STATUS')
    expect(stepRawDiagnostics(step)).toBe('supplier session endpoint returned non-success status')
  })
})
