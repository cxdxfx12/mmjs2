<template>
  <div class="test-page">
    <!-- 测试输入区 -->
    <div class="test-toolbar">
      <div class="tt-title">
        <el-icon :size="20"><Aim /></el-icon>
        <span>规则快速测试</span>
      </div>
      <div class="tt-fields">
        <el-select v-model="testForm.customer" filterable placeholder="选择客户" size="default" style="width:180px">
          <el-option v-for="c in customers" :key="c.name" :label="c.name" :value="c.name" />
        </el-select>
        <el-select v-model="testForm.province" filterable placeholder="选择省份(all=全部)" size="default" style="width:180px">
          <el-option label="全部省份" value="all" />
          <el-option v-for="p in PROVINCES" :key="p" :label="p" :value="p" />
        </el-select>
        <el-input-number v-model="testForm.weight" :min="0.01" :step="0.5" :precision="2" size="default" controls-position="right" style="width:140px" />
        <span class="tt-unit">kg</span>
        <el-button type="primary" size="default" @click="runTest" :loading="testing">单次测试</el-button>
        <el-button type="success" size="default" @click="runBatchTest" :loading="testing">批量测试</el-button>
        <el-button text @click="clearResults">清空结果</el-button>
      </div>
    </div>

    <div v-if="testResult || batchTestResults.length > 0" class="test-summary">
      <div class="summary-item">
        <span class="summary-label">客户</span>
        <span class="summary-value">{{ testForm.customer || '未选择' }}</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">省份</span>
        <span class="summary-value">{{ provinceLabel }}</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">重量</span>
        <span class="summary-value">{{ formatWeight(testForm.weight) }} kg</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">状态</span>
        <span class="summary-value" :class="testResult?.rule_id ? 'good' : 'warn'">{{ testResult?.rule_id ? '已命中规则' : (testResult ? '已回退到兜底' : '已完成批量测试') }}</span>
      </div>
    </div>

    <!-- 结果区域：左右分栏 -->
    <div class="test-body" v-loading="testing">
      <!-- 左侧：单次测试结果 -->
      <div class="single-panel">
        <div class="panel-header">
          <el-icon><Document /></el-icon>
          <span>单次测试结果</span>
        </div>
        <div class="panel-content" v-if="testResult">
          <div class="tr-fee">
            <span class="tr-label">计算运费</span>
            <span class="tr-value">¥{{ formatMoney(testResult.fee) }}</span>
          </div>
          <div class="tr-note" :class="testResult.rule_id ? 'good' : 'warn'">
            {{ testResult.rule_id ? ('已命中' + ruleLevelText(testResult.rule_level)) : '当前未命中具体规则，已使用默认/兜底结果' }}
          </div>
          <div class="tr-grid">
            <div class="tr-item"><span class="tr-k">规则级别</span><span class="tr-v">{{ ruleLevelText(testResult.rule_level) }}</span></div>
            <div class="tr-item"><span class="tr-k">计费模式</span><span class="tr-v">{{ testResult.calc_mode === 'bracket' ? '区间计费' : '首重续重' }}</span></div>
            <div class="tr-item"><span class="tr-k">续重模式</span><span class="tr-v">{{ testResult.cont_mode === 'hundred_gram' ? '百克续重' : (testResult.cont_mode === 'actual_weight' ? '实际重量' : '整kg续重') }}</span></div>
            <div class="tr-item" v-if="testResult.zone_name"><span class="tr-k">所属区域</span><span class="tr-v">{{ testResult.zone_name }}</span></div>
            <div class="tr-item"><span class="tr-k">首重/单价</span><span class="tr-v">{{ formatWeight(testResult.first_weight) }}kg / ¥{{ formatMoney(testResult.first_price) }}</span></div>
            <div class="tr-item"><span class="tr-k">续重单价</span><span class="tr-v">¥{{ formatMoney(testResult.cont_price) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.surcharge)"><span class="tr-k">偏远附加费</span><span class="tr-v">¥{{ formatMoney(testResult.surcharge) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.province_surcharge)"><span class="tr-k">省份加价</span><span class="tr-v">¥{{ formatMoney(testResult.province_surcharge) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.global_markup_fixed)"><span class="tr-k">全局固定加价</span><span class="tr-v">¥{{ formatMoney(testResult.global_markup_fixed) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.global_markup_percent)"><span class="tr-k">全局百分比加价</span><span class="tr-v">{{ formatNumber(testResult.global_markup_percent) }}%</span></div>
            <div class="tr-item" v-if="hasValue(testResult.min_fee)"><span class="tr-k">保底价</span><span class="tr-v">¥{{ formatMoney(testResult.min_fee) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.max_fee)"><span class="tr-k">最高价</span><span class="tr-v">¥{{ formatMoney(testResult.max_fee) }}</span></div>
            <div class="tr-item"><span class="tr-k">原始运费</span><span class="tr-v">¥{{ formatMoney(testResult.raw_fee) }}</span></div>
            <div class="tr-item" v-if="hasValue(testResult.markup)"><span class="tr-k">加价合计</span><span class="tr-v">¥{{ formatMoney(testResult.markup) }}</span></div>
            <div v-if="testResult.avg_weight_rule" class="tr-item tr-aw">
              <span class="tr-k">拉均重规则</span>
              <span class="tr-v">
                <span :class="{ 'aw-disabled': !testResult.avg_weight_rule.is_enabled }">
                  {{ testResult.avg_weight_rule.is_enabled ? '已启用' : '已禁用' }}
                </span>
                <span v-if="testResult.avg_weight_rule.is_enabled">
                  · 基准{{ testResult.avg_weight_rule.base_weight }}kg
                  <span v-if="testResult.avg_weight_rule.weight_limit > 0">· 上限{{ testResult.avg_weight_rule.weight_limit }}kg</span>
                  · {{ testResult.avg_weight_rule.step_price }}元/kg
                </span>
              </span>
            </div>
          </div>
          <div v-if="testResult.brackets && testResult.brackets.length > 0" class="tr-brackets">
            <div class="tr-b-header">区间价格明细</div>
            <div v-for="(b, i) in testResult.brackets" :key="i" class="tr-b-item">
              <span class="tr-b-range">{{ formatBracketRange(b) }}</span>
              <span v-if="b.calc_type === 'fixed'" class="tr-b-price">¥{{ formatMoney(b.fixed_price) }}</span>
              <span v-else class="tr-b-price">首重{{ formatWeight(b.first_weight) }}kg ¥{{ formatMoney(b.first_price) }} + 续重¥{{ formatMoney(b.cont_price) }}/{{ b.cont_mode === 'hundred_gram' ? '百克' : 'kg' }}</span>
            </div>
          </div>
          <div v-else-if="testResult.calc_mode === 'bracket'" class="tr-brackets">
            <div class="tr-b-header">区间价格明细</div>
            <div class="tr-b-item">
              <span class="tr-b-range">未返回区间明细</span>
              <span class="tr-b-price">请检查规则配置</span>
            </div>
          </div>
        </div>
        <div v-else class="panel-empty">
          <el-icon :size="40" color="#dcdfe6"><Document /></el-icon>
          <p>选择客户和省份，点击「单次测试」查看结果</p>
        </div>
      </div>

      <!-- 右侧：批量测试结果 -->
      <div class="batch-panel">
        <div class="panel-header">
          <el-icon><Grid /></el-icon>
          <span>批量测试结果</span>
          <el-tag v-if="batchTestResults.length > 0" size="small" type="info" effect="plain" style="margin-left:8px">{{ batchTestResults.length }}条</el-tag>
        </div>
        <div class="panel-content" v-if="batchTestResults.length > 0">
          <div class="br-summary">
            <div class="br-stat">
              <span class="br-stat-label">平均运费</span>
              <span class="br-stat-value">¥{{ batchAvgFee }}</span>
            </div>
            <div class="br-stat">
              <span class="br-stat-label">最高运费</span>
              <span class="br-stat-value highlight">¥{{ batchMaxFee }}</span>
            </div>
            <div class="br-stat">
              <span class="br-stat-label">最低运费</span>
              <span class="br-stat-value low">¥{{ batchMinFee }}</span>
            </div>
          </div>
          <el-table :data="batchTestResults" border size="small" max-height="calc(100vh - 320px)" class="batch-table">
            <el-table-column prop="province" label="省份" width="90" fixed="left" />
            <el-table-column prop="weight" label="重量(kg)" width="90" align="center" />
            <el-table-column prop="zone_name" label="区域" width="70" align="center">
              <template #default="{row}">{{ row.zone_name || '—' }}</template>
            </el-table-column>
            <el-table-column prop="calc_mode" label="计费模式" width="90" align="center">
              <template #default="{row}">{{ row.calc_mode === 'bracket' ? '区间' : '标准' }}</template>
            </el-table-column>
            <el-table-column prop="cont_mode" label="续重" width="80" align="center">
              <template #default="{row}">{{ row.cont_mode === 'hundred_gram' ? '百克' : (row.cont_mode === 'actual_weight' ? '实际' : '整kg') }}</template>
            </el-table-column>
            <el-table-column prop="rule_level" label="规则级别" width="90" align="center">
              <template #default="{row}">
                <el-tag :type="ruleLevelTag(row.rule_level)" size="small" effect="plain">{{ ruleLevelText(row.rule_level) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="raw_fee" label="原始运费" width="90" align="center">
              <template #default="{row}">¥{{ formatMoney(row.raw_fee) }}</template>
            </el-table-column>
            <el-table-column prop="markup" label="加价" width="80" align="center">
              <template #default="{row}">{{ hasValue(row.markup) ? '¥' + formatMoney(row.markup) : '—' }}</template>
            </el-table-column>
            <el-table-column prop="fee" label="最终运费" width="100" align="center" fixed="right">
              <template #default="{row}">
                <span class="fee-final">¥{{ formatMoney(row.fee) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <div v-else class="panel-empty">
          <el-icon :size="40" color="#dcdfe6"><Grid /></el-icon>
          <p>点击「批量测试」查看所有省份×重量的运费</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useAppStore, type CustomerInfo } from '@/stores/app'
import { ElMessage } from 'element-plus'
import { Aim, Document, Grid } from '@element-plus/icons-vue'

const store = useAppStore()

const customers = ref<CustomerInfo[]>([])
const testing = ref(false)
const testForm = reactive({ customer: '', province: 'all', weight: 1 })
const testResult = ref<any>(null)
const batchTestResults = ref<any[]>([])

const provinceLabel = computed(() => {
  if (!testForm.province || testForm.province === 'all') return '全部省份'
  return testForm.province
})
const batchAvgFee = computed(() => {
  if (!batchTestResults.value.length) return '0.00'
  const sum = batchTestResults.value.reduce((s, r) => s + (Number(r.fee) || 0), 0)
  return (sum / batchTestResults.value.length).toFixed(2)
})
const batchMaxFee = computed(() => {
  if (!batchTestResults.value.length) return '0.00'
  return Math.max(...batchTestResults.value.map(r => Number(r.fee) || 0)).toFixed(2)
})
const batchMinFee = computed(() => {
  if (!batchTestResults.value.length) return '0.00'
  return Math.min(...batchTestResults.value.map(r => Number(r.fee) || 0)).toFixed(2)
})

async function loadCustomers() {
  try {
    const data = await store.fetchCustomers()
    customers.value = Array.isArray(data) ? data : []
    if (customers.value.length > 0 && !testForm.customer) {
      testForm.customer = customers.value[0].name
    }
  } catch {
    customers.value = []
  }
}

function hasValue(value: unknown) {
  return value !== null && value !== undefined && value !== '' && value !== 0
}

function formatMoney(value: number | string | null | undefined) {
  const num = Number(value)
  return Number.isFinite(num) ? num.toFixed(2) : '0.00'
}

function formatWeight(value: number | string | null | undefined) {
  const num = Number(value)
  return Number.isFinite(num) ? num.toFixed(2) : '0.00'
}

function formatNumber(value: number | string | null | undefined) {
  const num = Number(value)
  return Number.isFinite(num) ? num.toFixed(2) : '0.00'
}

function formatBracketRange(bracket: any) {
  const from = formatWeight(bracket?.weight_from)
  const to = bracket?.weight_to > 0 ? formatWeight(bracket.weight_to) : '∞'
  return `${from}-${to}kg`
}

function clearResults() {
  testResult.value = null
  batchTestResults.value = []
}

async function runTest() {
  if (!testForm.customer || !testForm.weight) {
    ElMessage.warning('请填写客户和重量')
    return
  }
  testing.value = true
  batchTestResults.value = []
  try {
    testResult.value = await store.testRule(testForm.customer, testForm.province, testForm.weight)
  } catch { ElMessage.error('测试失败') }
  finally { testing.value = false }
}

async function runBatchTest() {
  if (!testForm.customer) {
    ElMessage.warning('请选择客户')
    return
  }
  testing.value = true
  testResult.value = null
  batchTestResults.value = []
  try {
    const weights = [0.3, 0.8, 1.5, 2.5, 5, 10, 20, 35]
    const result = await store.testRuleBatch(testForm.customer, testForm.province || 'all', weights)
    if (result && result.results && Array.isArray(result.results)) {
      batchTestResults.value = result.results
      if (batchTestResults.value.length === 0) {
        ElMessage.warning('没有测试结果，请检查客户是否有规则')
      } else {
        ElMessage.success(`批量测试完成，共 ${batchTestResults.value.length} 条结果`)
      }
    } else {
      ElMessage.error('批量测试返回数据格式异常')
    }
  } catch (e) {
    console.error('批量测试异常:', e)
    ElMessage.error('批量测试失败')
  }
  finally { testing.value = false }
}

function ruleLevelText(level: string): string {
  const map: Record<string, string> = {
    campaign: '活动规则',
    customer: '客户规则',
    global: '全局规则',
    default: '默认规则',
    fallback: '全局保底',
  }
  return map[level] || level || '未匹配'
}

function ruleLevelTag(level: string): string {
  const map: Record<string, string> = {
    campaign: 'danger',
    customer: 'success',
    global: 'primary',
    default: 'info',
    fallback: 'warning',
  }
  return map[level] || 'info'
}

const PROVINCES = [
  '北京','天津','上海','重庆',
  '河北','山西','辽宁','吉林','黑龙江',
  '江苏','浙江','安徽','福建','江西','山东',
  '河南','湖北','湖南','广东',
  '四川','贵州','云南','陕西','甘肃','青海',
  '广西','内蒙古','宁夏','新疆','西藏','海南',
  '香港','澳门','台湾',
]

onMounted(() => {
  loadCustomers()
})
</script>

<style scoped>
.test-page { display: flex; flex-direction: column; gap: 16px; max-width: 1400px; height: 100%; }

/* 工具栏 */
.test-toolbar {
  background: #fff; border-radius: 10px; border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0,0,0,.04);
  padding: 16px 20px; display: flex; align-items: center; justify-content: space-between;
  flex-wrap: wrap; gap: 12px;
}
.tt-title { display: flex; align-items: center; gap: 8px; font-size: 16px; font-weight: 600; color: #303133; }
.tt-fields { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.tt-unit { font-size: 13px; color: #909399; }

/* 结果区域 */
.test-summary {
  display: flex; gap: 12px; flex-wrap: wrap; padding: 14px 16px; background: #fff;
  border-radius: 10px; border: 1px solid #e4e7ed; box-shadow: 0 1px 4px rgba(0,0,0,.04);
}
.summary-item { display: flex; flex-direction: column; gap: 4px; min-width: 120px; }
.summary-label { font-size: 12px; color: #909399; }
.summary-value { font-size: 14px; color: #303133; font-weight: 600; }
.summary-value.good { color: #67c23a; }
.summary-value.warn { color: #e6a23c; }
.test-body {
  display: flex; gap: 16px; flex: 1; min-height: 0;
}

/* 单次结果面板 */
.single-panel {
  width: 380px; min-width: 340px; background: #fff;
  border-radius: 10px; border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0,0,0,.04);
  display: flex; flex-direction: column; overflow: hidden;
}

/* 批量结果面板 */
.batch-panel {
  flex: 1; background: #fff; border-radius: 10px;
  border: 1px solid #e4e7ed; box-shadow: 0 1px 4px rgba(0,0,0,.04);
  display: flex; flex-direction: column; overflow: hidden; min-width: 0;
}

.panel-header {
  display: flex; align-items: center; gap: 8px;
  padding: 14px 18px; border-bottom: 1px solid #ebeef5;
  font-size: 15px; font-weight: 600; color: #303133;
  background: linear-gradient(135deg, #fafbff, #f5f7ff);
}
.panel-content { flex: 1; overflow-y: auto; padding: 18px; }
.panel-empty {
  flex: 1; display: flex; flex-direction: column; align-items: center;
  justify-content: center; gap: 10px; color: #c0c4cc; font-size: 13px;
}

/* 单次测试结果 */
.tr-fee {
  display: flex; align-items: center; gap: 12px; margin-bottom: 16px;
  padding-bottom: 14px; border-bottom: 1px dashed #ebeef5;
}
.tr-label { font-size: 14px; color: #606266; }
.tr-value { font-size: 32px; font-weight: 700; color: #f56c6c; }
.tr-note { margin-bottom: 12px; padding: 8px 10px; border-radius: 6px; background: #f5f7fa; font-size: 13px; }
.tr-note.good { color: #67c23a; background: #f0f9eb; }
.tr-note.warn { color: #e6a23c; background: #fdf6ec; }
.tr-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.tr-item { display: flex; flex-direction: column; gap: 3px; }
.tr-aw { grid-column: 1 / -1; }
.tr-k { font-size: 12px; color: #909399; }
.tr-v { font-size: 14px; color: #303133; font-weight: 500; }
.tr-v .aw-disabled { color: #c0c4cc; }
.tr-brackets { margin-top: 16px; padding-top: 14px; border-top: 1px dashed #ebeef5; }
.tr-b-header { font-size: 13px; font-weight: 600; color: #606266; margin-bottom: 8px; }
.tr-b-item { display: flex; justify-content: space-between; padding: 6px 10px; background: #fafafa; border-radius: 6px; margin-bottom: 4px; font-size: 13px; }
.tr-b-range { color: #606266; font-weight: 500; }
.tr-b-price { color: #409eff; }

/* 批量测试结果 */
.br-summary {
  display: flex; gap: 20px; margin-bottom: 16px;
  padding: 16px; background: linear-gradient(135deg, #fafbff, #f5f7ff);
  border-radius: 8px; border: 1px solid #ebeef5;
}
.br-stat { display: flex; flex-direction: column; gap: 4px; }
.br-stat-label { font-size: 12px; color: #909399; }
.br-stat-value { font-size: 22px; font-weight: 700; color: #303133; }
.br-stat-value.highlight { color: #f56c6c; }
.br-stat-value.low { color: #67c23a; }

.batch-table { flex: 1; }
.fee-final { font-weight: 700; color: #f56c6c; font-size: 15px; }
</style>
