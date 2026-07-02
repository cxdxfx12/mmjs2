<template>
  <div class="home">
    <!-- 欢迎横幅 -->
    <div class="welcome-banner">
      <div class="welcome-text">
        <h1>欢迎使用喵喵云结算</h1>
        <p>快速、准确地完成快递运费结算</p>
      </div>
      <el-button type="primary" size="large" round @click="$router.push('/calc')" class="cta-btn">
        <el-icon><Upload /></el-icon>开始计算
      </el-button>
    </div>

    <!-- 授权提醒 -->
    <transition name="fade">
      <el-alert v-if="!store.isLicensed" title="软件未激活，请前往授权管理激活" type="warning" show-icon :closable="false" class="mb20 alert-rounded">
        <el-button type="warning" link @click="$router.push('/license')">前往激活 →</el-button>
      </el-alert>
    </transition>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card stat-blue" @click="$router.push('/calc')">
        <div class="stat-icon-box"><el-icon :size="28"><Upload /></el-icon></div>
        <div class="stat-content">
          <div class="stat-val">{{ fmtWan(stats.totalCount) }}<span class="stat-unit">万</span></div>
          <div class="stat-label">累计计算件数</div>
        </div>
      </div>
      <div class="stat-card stat-green">
        <div class="stat-icon-box"><el-icon :size="28"><Coin /></el-icon></div>
        <div class="stat-content">
          <div class="stat-val">¥{{ fmtWan(stats.totalFee) }}<span class="stat-unit">万</span></div>
          <div class="stat-label">累计运费总额</div>
        </div>
      </div>
      <div class="stat-card stat-orange">
        <div class="stat-icon-box"><el-icon :size="28"><TrendCharts /></el-icon></div>
        <div class="stat-content">
          <div class="stat-val">¥{{ fmtWan(stats.avgFee) }}<span class="stat-unit">万</span></div>
          <div class="stat-label">历史平均运费</div>
        </div>
      </div>
      <div class="stat-card stat-purple" @click="$router.push('/rules')">
        <div class="stat-icon-box"><el-icon :size="28"><Setting /></el-icon></div>
        <div class="stat-content">
          <div class="stat-val">{{ rulesCount }}</div>
          <div class="stat-label">计费规则数</div>
        </div>
      </div>
    </div>

    <!-- 快捷操作 + 授权 -->
    <div class="row-2col">
      <div class="quick-actions card-glass">
        <div class="card-title"><el-icon><Promotion /></el-icon> 快捷操作</div>
        <div class="qa-grid">
          <div class="qa-item" @click="$router.push('/calc')">
            <div class="qa-icon qa-blue"><el-icon :size="24"><DocumentAdd /></el-icon></div>
            <div class="qa-text">
              <span>上传文件计费</span>
              <small>上传 Excel 开始结算</small>
            </div>
          </div>
          <div class="qa-item" @click="$router.push('/rules')">
            <div class="qa-icon qa-green"><el-icon :size="24"><EditPen /></el-icon></div>
            <div class="qa-text">
              <span>管理计费规则</span>
              <small>编辑首重/续重费率</small>
            </div>
          </div>
          <div class="qa-item" @click="$router.push('/history')">
            <div class="qa-icon qa-orange"><el-icon :size="24"><Clock /></el-icon></div>
            <div class="qa-text">
              <span>查看历史记录</span>
              <small>回顾过往结算数据</small>
            </div>
          </div>
          <div class="qa-item" @click="$router.push('/license')">
            <div class="qa-icon qa-gray"><el-icon :size="24"><Key /></el-icon></div>
            <div class="qa-text">
              <span>授权管理</span>
              <small>查看/续期软件授权</small>
            </div>
          </div>
        </div>
      </div>

      <div class="license-card card-glass">
        <div class="card-title"><el-icon><CircleCheck /></el-icon> 授权状态</div>
        <div v-if="store.license" class="li-body">
          <div class="li-customer">{{ store.license.customer_name || '未设置' }}</div>
          <el-progress :percentage="lcPct" :color="lcColor" :stroke-width="14" :show-text="false" class="li-bar" />
          <div class="li-days">
            <span class="li-num" :style="{color:lcColor}">{{ store.daysLeft }}</span> 天剩余
          </div>
          <div class="li-expire">到期 {{ store.license.expires_at }}</div>
        </div>
        <div v-else class="li-empty">
          <el-icon :size="40" color="#c0c4cc"><WarningFilled /></el-icon>
          <p>软件未激活</p>
          <el-button type="primary" size="small" round @click="$router.push('/license')">前往激活</el-button>
        </div>
      </div>
    </div>

    <!-- 最近记录 -->
    <div v-if="history && history.length" class="card-glass mt20">
      <div class="card-title row-between">
        <span><el-icon><Clock /></el-icon> 最近计算记录</span>
        <el-button text type="primary" @click="$router.push('/history')">查看全部 →</el-button>
      </div>
      <el-table :data="history.slice(0,8)" stripe size="small" class="history-table">
        <el-table-column prop="created_at" label="时间" width="170" />
        <el-table-column label="源文件" min-width="200" show-overflow-tooltip>
          <template #default="{row}"><span style="color:#4472C4;cursor:pointer">{{ baseName(row.input_file) }}</span></template>
        </el-table-column>
        <el-table-column prop="total_count" label="件数" width="90" />
        <el-table-column label="运费" width="130">
          <template #default="{row}">¥{{ (row.total_fee||0).toLocaleString(undefined,{minimumFractionDigits:2}) }}</template>
        </el-table-column>
        <el-table-column label="均价" width="90">
          <template #default="{row}">¥{{ (row.avg_fee||0).toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="70">
          <template #default="{row}">{{ row.calc_duration }}s</template>
        </el-table-column>
        <el-table-column prop="rule_summary" label="规则" width="100" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAppStore } from '@/stores/app'
import { Upload, Coin, TrendCharts, Setting, Promotion, DocumentAdd, EditPen, Clock, Key, CircleCheck, WarningFilled } from '@element-plus/icons-vue'

const store = useAppStore()
const rulesCount = computed(() => store.rules.length)
const stats = ref({totalCount:0,totalFee:0,avgFee:'0'})
const history = ref<any[]>([])
const lcPct = computed(()=>{
  if(!store.license||!store.daysLeft) return 0
  return Math.min(100, Math.round((store.daysLeft/365)*100))
})
const lcColor = computed(()=>{
  if(store.daysLeft<=7) return '#f56c6c'
  if(store.daysLeft<=30) return '#e6a23c'
  return '#67c23a'
})
function baseName(p: string) {
  if(!p) return '-'
  const a = p.replace(/\\/g,'/').split('/')
  return a[a.length-1] || p
}
function fmtWan(v: number | string) {
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (!n || n === 0) return '0'
  const w = n / 10000
  if (w >= 1000) return w.toFixed(0)
  if (w >= 10) return w.toFixed(1)
  return w.toFixed(2)
}
function fmtYuan(v: number | string) {
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (!n || n === 0) return '0.00'
  return parseFloat(n.toFixed(2)).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
onMounted(async()=>{
  try {
    const token = localStorage.getItem('yunfei_token')
    const r = await fetch('/api/history', { headers: { Authorization: `Bearer ${token || ''}` } })
    history.value = await r.json()
    if(Array.isArray(history.value) && history.value.length) {
      let tc = 0, tf = 0
      for(const h of history.value) { tc += h.total_count||0; tf += h.total_fee||0 }
      stats.value = {
        totalCount: tc,
        totalFee: Math.round(tf*100)/100,
        avgFee: tc>0 ? (tf/history.value.length).toFixed(2) : '0'
      }
    }
  } catch {}
})
</script>

<style scoped>
.home { max-width: 1200px; margin: 0 auto; }
.mb20 { margin-bottom: 20px; }
.mt20 { margin-top: 20px; }

/* 欢迎横幅 */
.welcome-banner {
  display: flex; align-items: center; justify-content: space-between;
  background: linear-gradient(135deg, #1a1635, #2d2854);
  border-radius: 16px; padding: 32px 36px; margin-bottom: 24px;
  color: #fff; box-shadow: 0 4px 24px rgba(26,22,53,0.2);
}
.welcome-text h1 { font-size: 26px; font-weight: 700; margin: 0 0 6px; }
.welcome-text p { font-size: 14px; opacity: 0.7; margin: 0; }
.cta-btn { background: linear-gradient(135deg,#4472C4,#6ba0ff) !important; border: none !important; }

.alert-rounded { border-radius: 12px !important; }

/* 统计卡片网格 */
.stats-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px;
}
.stat-card {
  display: flex; align-items: center; gap: 16px;
  padding: 24px 20px; border-radius: 14px; color: #fff;
  cursor: pointer; transition: all 0.3s;
}
.stat-card:hover { transform: translateY(-3px); box-shadow: 0 8px 28px rgba(0,0,0,0.15); }
.stat-icon-box { opacity: 0.9; }
.stat-content { flex: 1; }
.stat-val { font-size: 22px; font-weight: 700; margin-bottom: 2px; }
.stat-unit { font-size: 14px; font-weight: 400; opacity: 0.8; margin-left: 2px; }
.stat-label { font-size: 12px; opacity: 0.8; }
.stat-blue { background: linear-gradient(135deg,#4472C4,#5b8def); }
.stat-green { background: linear-gradient(135deg,#67c23a,#85ce61); }
.stat-orange { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.stat-purple { background: linear-gradient(135deg,#9b59b6,#b07cc6); }

/* 两列 */
.row-2col { display: grid; grid-template-columns: 1.6fr 1fr; gap: 20px; }

/* 卡片 */
.card-glass {
  background: #fff; border-radius: 14px; padding: 24px;
  box-shadow: 0 2px 16px rgba(0,0,0,0.05);
}
.card-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 16px; font-weight: 600; color: #303133;
  margin-bottom: 20px;
}
.row-between { justify-content: space-between; margin-bottom: 8px; }

/* 快捷操作 */
.qa-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.qa-item {
  display: flex; align-items: center; gap: 12px;
  padding: 16px; border-radius: 12px; cursor: pointer;
  border: 1px solid #ebeef5; transition: all 0.25s;
}
.qa-item:hover { background: #f0f4ff; border-color: #4472C4; transform: translateY(-2px); }
.qa-icon { width: 44px; height: 44px; border-radius: 12px; display: flex; align-items: center; justify-content: center; color: #fff; }
.qa-blue { background: linear-gradient(135deg,#4472C4,#5b8def); }
.qa-green { background: linear-gradient(135deg,#67c23a,#85ce61); }
.qa-orange { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.qa-gray { background: linear-gradient(135deg,#909399,#b4b7bd); }
.qa-text { display: flex; flex-direction: column; gap: 2px; }
.qa-text span { font-size: 14px; font-weight: 500; color: #303133; }
.qa-text small { font-size: 12px; color: #909399; }

/* 授权卡片 */
.li-body { text-align: center; padding: 8px 0; }
.li-customer { font-size: 22px; font-weight: 700; color: #303133; margin-bottom: 16px; }
.li-bar { margin-bottom: 10px; }
.li-days { color: #909399; font-size: 14px; margin-bottom: 4px; }
.li-num { font-weight: 700; font-size: 20px; margin-right: 4px; }
.li-expire { font-size: 12px; color: #c0c4cc; }
.li-empty {
  display: flex; flex-direction: column; align-items: center; gap: 12px;
  padding: 20px 0; color: #909399;
}
.li-empty p { margin: 0; }

.history-table { border-radius: 10px; overflow: hidden; }

.fade-enter-active, .fade-leave-active { transition: opacity 0.3s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
