import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import dayjs from 'dayjs'

export interface LicenseInfo {
  machine_code: string
  customer_name: string
  expires_at: string
  issued_at: string
  days_left: number
  is_valid: boolean
  cached?: boolean  // 在线授权缓存标记
  error?: string
}

export interface WeightBracket {
  id: number
  rule_id: number
  weight_from: number
  weight_to: number
  calc_type: string
  fixed_price: number
  first_weight: number
  first_price: number
  cont_price: number
  cont_mode: string
  sort_order: number
}

export interface FreightRule {
  id: number
  rule_type: string
  customer_name: string
  province: string
  cont_mode: string
  first_weight: number
  first_price: number
  cont_price: number
  min_fee: number
  max_fee: number
  surcharge: number
  campaign_name: string
  campaign_start: string
  campaign_end: string
  is_enabled: number
  remark: string
  calc_mode: string
  zone_id: number
  zone_name: string
  brackets?: WeightBracket[]
}

export interface AvgWeightRule {
  id: number
  scope_type: string
  customer_name: string
  base_weight: number
  weight_limit: number
  step_weight: number
  step_price: number
  max_markup: number
  round_mode: string
  is_enabled: number
  remark: string
}

export interface CustomerInfo {
  name: string
  rule_count: number
}

export interface GlobalRule {
  default_first_weight: number
  default_first_price: number
  default_cont_price: number
  default_min_fee: number
  no_weight_price: number
  markup_fixed: number
  markup_percent: number
}

export interface ProvinceSurcharge {
  id: number
  province_name: string
  surcharge: number
  remark: string
}

function authHeaders(): Record<string,string> {
  return { Authorization: `Bearer ${localStorage.getItem('yunfei_token') || ''}` }
}

export const useAppStore = defineStore('app', () => {
  const license = ref<LicenseInfo | null>(null)
  const rules = ref<FreightRule[]>([])
  const machineCode = ref('')
  const calculating = ref(false)

  const isLicensed = computed(() => license.value?.is_valid ?? false)
  const daysLeft = computed(() => license.value?.days_left ?? 0)
  const licenseStatus = computed(() => {
    if (!license.value) return 'unknown'
    if (!license.value.is_valid) return 'expired'
    if (daysLeft.value <= 7) return 'expiring'
    return 'active'
  })

  async function fetchLicense() {
    try {
      // 优先检查在线授权（不需要登录）
      const online = await checkOnlineLicense()
      if (online?.valid) {
        license.value = { ...online, is_valid: true } as LicenseInfo
        return
      }
      // 回退到本地离线授权
      const res = await fetch('/api/license/info', { headers: authHeaders() })
      license.value = await res.json()
    } catch (e) {
      console.error('获取授权信息失败', e)
    }
  }

  // 检查在线授权（无需登录token）
  async function checkOnlineLicense(): Promise<any> {
    try {
      const res = await fetch('/api/license/check-online')
      return await res.json()
    } catch { return null }
  }

  // 在线激活（输入授权码）
  async function activateOnline(licenseKey: string) {
    const res = await fetch('/api/license/activate-online', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ license_key: licenseKey }),
    })
    const data = await res.json()
    if (data.ok) {
      await fetchLicense()
    }
    return data
  }

  async function fetchMachineCode() {
    try {
      const res = await fetch('/api/machine-code', { headers: authHeaders() })
      const data = await res.json()
      machineCode.value = data.code
    } catch (e) {
      console.error('获取机器码失败', e)
    }
  }

  async function importLicense(b64: string) {
    const res = await fetch('/api/license/import', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ license: b64 }),
    })
    const data = await res.json()
    if (data.ok) {
      await fetchLicense()
    }
    return data
  }

  async function fetchRules() {
    try {
      const res = await fetch('/api/rules', { headers: authHeaders() })
      const data = await res.json()
      rules.value = Array.isArray(data) ? data : []
    } catch (e) {
      console.error('获取规则失败', e)
      rules.value = []
    }
  }

  async function fetchRulesByCustomer(customer: string) {
    try {
      const res = await fetch('/api/rules?customer=' + encodeURIComponent(customer), { headers: authHeaders() })
      const data = await res.json()
      return Array.isArray(data) ? data as FreightRule[] : []
    } catch (e) {
      console.error('获取客户规则失败', e)
      return []
    }
  }

  async function saveRule(rule: Partial<FreightRule>) {
    const res = await fetch('/api/rules/save', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(rule),
    })
    const data = await res.json()
    // 如果后端返回的 id 为 0 或未包含 id，则视为失败（后端可能返回 0 表示保存被拒绝）
    if (!data || !data.id || Number(data.id) === 0) {
      // 刷新规则以保持界面一致
      await fetchRules()
      return { ok: false, error: data && data.error ? data.error : '保存失败' }
    }
    await fetchRules()
    return { ok: true, id: data.id }
  }

  async function deleteRule(id: number) {
    const res = await fetch('/api/rules/delete', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ id }),
    })
    await fetchRules()
    return await res.json()
  }

  async function deleteRulesBatch(ids: number[]) {
    const res = await fetch('/api/rules/delete-batch', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids }),
    })
    await fetchRules()
    return await res.json()
  }

  // ===== 客户管理 =====
  async function fetchCustomers() {
    try {
      const res = await fetch('/api/customers', { headers: authHeaders() })
      const data = await res.json()
      return Array.isArray(data) ? data as CustomerInfo[] : []
    } catch (e) {
      return []
    }
  }

  async function deleteCustomer(name: string) {
    const res = await fetch('/api/customers/delete', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    await fetchRules()
    return await res.json()
  }

  async function copyCustomerRules(from: string, to: string) {
    const res = await fetch('/api/customers/copy-rules', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ from, to }),
    })
    await fetchRules()
    return await res.json()
  }

  // ===== 批量导入 =====
  async function importCustomerRules(file: File) {
    const fd = new FormData()
    fd.append('file', file)
    const res = await fetch('/api/customers/import', {
      method: 'POST',
      headers: authHeaders(),
      body: fd,
    })
    await fetchRules()
    return await res.json()
  }

  // ===== 下载模板 =====
  async function downloadTemplate() {
    const res = await fetch('/api/customers/template', { headers: authHeaders() })
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = '客户规则导入模板.xlsx'; a.click()
    URL.revokeObjectURL(url)
  }

  // ===== 全局规则 =====
  async function fetchGlobalRules() {
    try {
      const res = await fetch('/api/global-rules', { headers: authHeaders() })
      return await res.json() as GlobalRule
    } catch (e) {
      return null
    }
  }

  async function saveGlobalRules(gr: GlobalRule) {
    const res = await fetch('/api/global-rules', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(gr),
    })
    return await res.json()
  }

  // ===== 省份加价 =====
  async function fetchProvinceSurcharges() {
    try {
      const res = await fetch('/api/province-surcharges', { headers: authHeaders() })
      const data = await res.json()
      return Array.isArray(data) ? data as ProvinceSurcharge[] : []
    } catch { return [] }
  }

  async function saveProvinceSurcharge(p: ProvinceSurcharge) {
    const res = await fetch('/api/province-surcharges', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(p),
    })
    return await res.json()
  }

  async function deleteProvinceSurcharge(id: number) {
    const res = await fetch('/api/province-surcharges/delete', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ id }),
    })
    return await res.json()
  }

  // ===== 规则快速测试 =====
  async function testRule(customer: string, province: string, weight: number) {
    const res = await fetch('/api/rules/test', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer, province, weight, batch: false }),
    })
    return await res.json()
  }

  async function testRuleBatch(customer: string, province: string, weights: number[]) {
    const res = await fetch('/api/rules/test', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer, province, weights, batch: true }),
    })
    return await res.json()
  }

  // ===== 导出客户规则 =====
  async function exportCustomerRules(customer: string) {
    const url = '/api/customers/export' + (customer ? '?customer=' + encodeURIComponent(customer) : '')
    const res = await fetch(url, { headers: authHeaders() })
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = customer ? (customer + '_规则.xlsx') : '全部客户规则.xlsx'
    a.click()
    URL.revokeObjectURL(blobUrl)
  }

  // ===== 区域模板 =====
  async function fetchZoneTemplates() {
    try {
      const res = await fetch('/api/zones/templates', { headers: authHeaders() })
      const data = await res.json()
      return Array.isArray(data) ? data : []
    } catch { return [] }
  }

  async function fetchSamplePriceTable() {
    try {
      const res = await fetch('/api/zones/sample-price', { headers: authHeaders() })
      return await res.json()
    } catch { return null }
  }

  async function importSamplePriceTable(file: File) {
    const fd = new FormData()
    fd.append('file', file)
    const res = await fetch('/api/zones/import-price', {
      method: 'POST',
      headers: authHeaders(),
      body: fd,
    })
    return await res.json()
  }

  async function downloadSamplePriceTemplate() {
    const res = await fetch('/api/zones/price-template', { headers: authHeaders() })
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = '区域参考价模板.xlsx'; a.click()
    URL.revokeObjectURL(url)
  }

  async function generateZoneRules(customerName: string, contMode: string, calcMode: string, priceTable: Record<string, any>) {
    const res = await fetch('/api/zones/generate-rules', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer_name: customerName, cont_mode: contMode, calc_mode: calcMode, price_table: priceTable }),
    })
    return await res.json()
  }

  async function fetchAvgWeightRule(customerName: string): Promise<AvgWeightRule | null> {
    try {
      const res = await fetch('/api/avg-weight?customer=' + encodeURIComponent(customerName), { headers: authHeaders() })
      if (!res.ok) return null
      const data = await res.json()
      return data as AvgWeightRule
    } catch {
      return null
    }
  }

  async function saveAvgWeightRule(rule: AvgWeightRule) {
    const res = await fetch('/api/avg-weight-rules', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(rule),
    })
    return await res.json()
  }

  async function toggleAvgWeight(id: number, enabled: number) {
    const res = await fetch('/api/avg-weight/toggle', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, is_enabled: enabled }),
    })
    return await res.json()
  }

  return {
    license, rules, machineCode, calculating, isLicensed, daysLeft, licenseStatus,
    fetchLicense, fetchMachineCode, importLicense, fetchRules, fetchRulesByCustomer, saveRule, deleteRule, deleteRulesBatch,
    fetchCustomers, deleteCustomer, copyCustomerRules, importCustomerRules, downloadTemplate,
    fetchGlobalRules, saveGlobalRules,
    fetchProvinceSurcharges, saveProvinceSurcharge, deleteProvinceSurcharge,
    testRule, testRuleBatch, exportCustomerRules,
    checkOnlineLicense, activateOnline,
    fetchZoneTemplates, fetchSamplePriceTable, importSamplePriceTable, downloadSamplePriceTemplate, generateZoneRules,
    fetchAvgWeightRule, saveAvgWeightRule, toggleAvgWeight,
  }
})
