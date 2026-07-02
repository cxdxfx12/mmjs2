<template>
  <div class="license-page">
    <!-- 未授权激活流程 -->
    <template v-if="!store.isLicensed">
      <div class="page-header">
        <div class="header-left">
          <div class="header-icon-box warn"><el-icon :size="24"><Key /></el-icon></div>
          <div><h2>软件激活</h2><p>3步完成授权，开始使用全部功能</p></div>
        </div>
      </div>

      <!-- 步骤条 -->
      <div class="activation-steps">
        <div class="act-step" :class="{active:actStep===0,done:actStep>0}">
          <div class="act-num">1</div>
          <div class="act-info"><div class="act-title">获取机器码</div><div class="act-desc">复制唯一标识</div></div>
        </div>
        <div class="act-line" :class="{done:actStep>0}" />
        <div class="act-step" :class="{active:actStep===1,done:actStep>1}">
          <div class="act-num">2</div>
          <div class="act-info"><div class="act-title">联系客服</div><div class="act-desc">发送机器码获取授权</div></div>
        </div>
        <div class="act-line" :class="{done:actStep>1}" />
        <div class="act-step" :class="{active:actStep===2}">
          <div class="act-num">3</div>
          <div class="act-info"><div class="act-title">导入激活</div><div class="act-desc">上传 license.dat</div></div>
        </div>
      </div>

      <!-- Step 0 -->
      <div v-if="actStep === 0" class="act-card card-glass">
        <div class="act-card-icon blue"><el-icon :size="32"><Monitor /></el-icon></div>
        <h3>步骤一：获取机器码</h3>
        <p>以下为该设备的唯一标识码，请复制后发送给管理员</p>
        <div class="mc-box">
          <code>{{ store.machineCode || '正在获取...' }}</code>
          <el-button type="primary" size="small" round @click="copy" :disabled="!store.machineCode">
            <el-icon><CopyDocument /></el-icon>复制
          </el-button>
        </div>
        <el-button type="primary" size="large" round :disabled="!store.machineCode" @click="actStep=1" class="act-next">
          已复制，下一步 →
        </el-button>
      </div>

      <!-- Step 1 -->
      <div v-else-if="actStep === 1" class="act-card card-glass">
        <div class="act-card-icon orange"><el-icon :size="32"><ChatDotRound /></el-icon></div>
        <h3>步骤二：联系管理员获取授权码</h3>
        <div class="contact-list">
          <div class="contact-item">
            <el-icon :size="20" color="#67c23a"><CircleCheck /></el-icon>
            <span>将机器码发给管理员</span>
          </div>
          <div class="contact-item">
            <el-icon :size="20" color="#67c23a"><CircleCheck /></el-icon>
            <span>机器码：<code>{{ store.machineCode }}</code></span>
          </div>
          <div class="contact-item">
            <el-icon :size="20" color="#67c23a"><CircleCheck /></el-icon>
            <span>管理员生成后给你 <code>授权码</code>（YF- 开头）</span>
          </div>
          <div class="contact-item">
            <el-icon :size="20" color="#67c23a"><CircleCheck /></el-icon>
            <span>在下一步中输入授权码完成激活</span>
          </div>
        </div>
        <el-button type="primary" size="large" round @click="actStep=2" class="act-next">
          已获取授权码，下一步 →
        </el-button>
      </div>

      <!-- Step 2：在线激活码输入（推荐） -->
      <div v-else class="act-card card-glass">
        <div class="act-card-icon green"><el-icon :size="32"><Key /></el-icon></div>
        <h3>步骤三：输入授权码激活（推荐）</h3>
        <p>输入管理员发给你的授权码，联网激活</p>
        <div class="key-input-box">
          <el-input v-model="licenseKey" placeholder="例如：YF-XXXX-XXXX-XXXX" size="large" clearable
            @input="licenseKey = licenseKey.toUpperCase()" maxlength="19" class="key-input" />
        </div>
        <el-button type="primary" size="large" round :loading="activating" :disabled="!licenseKey || licenseKey.length < 4" @click="doActivate" class="act-next">
          <el-icon><CircleCheck /></el-icon>{{ activating ? '激活中...' : '联网激活' }}
        </el-button>
        <div v-if="activateMsg" class="act-msg" :class="{error:!activateOk}">{{ activateMsg }}</div>

        <!-- 分隔线 -->
        <div class="divider"><span>或使用离线文件激活</span></div>

        <p style="margin-top:12px">如果你有 license.dat 文件，也可在此导入</p>
        <div class="upload-zone-small" :class="{hasfile: lc}">
          <el-upload drag :auto-upload="false" :limit="1" accept=".dat" :on-change="onFile" :show-file-list="false">
            <el-icon :size="40"><UploadFilled /></el-icon>
            <div class="el-upload__text">拖拽或<em>点击选择</em> license.dat</div>
          </el-upload>
          <div v-if="fn" class="file-chosen">
            <el-icon color="#67c23a"><DocumentChecked /></el-icon>
            <span>{{ fn }}</span>
          </div>
        </div>
        <el-button type="warning" round :loading="importing" :disabled="!lc" @click="doImport">
          <el-icon><Upload /></el-icon>{{ importing ? '导入中...' : '导入文件激活' }}
        </el-button>
      </div>
    </template>

    <!-- 已授权信息 -->
    <template v-else>
      <div class="page-header">
        <div class="header-left">
          <div class="header-icon-box active"><el-icon :size="24"><CircleCheck /></el-icon></div>
          <div><h2>授权管理</h2><p>软件已激活，可正常使用全部功能</p></div>
        </div>
        <el-tag type="success" size="large" effect="dark" round>已授权</el-tag>
      </div>

      <div class="license-dashboard">
        <!-- 左侧信息 -->
        <div class="ld-left card-glass">
          <div class="ld-customer">
            <div class="ldc-avatar">{{ (store.license?.customer_name||'?')[0] }}</div>
            <div>
              <div class="ldc-name">{{ store.license?.customer_name || '未设置' }}</div>
              <div class="ldc-sub">授权客户</div>
            </div>
          </div>
          <div class="ld-details">
            <div class="ld-row">
              <span class="ldr-label">机器码</span>
              <el-tooltip :content="store.license?.machine_code"><span class="ldr-val mono">{{ (store.license?.machine_code||'').substring(0,16) }}...</span></el-tooltip>
            </div>
            <div class="ld-row">
              <span class="ldr-label">签发日期</span>
              <span class="ldr-val">{{ store.license?.issued_at }}</span>
            </div>
            <div class="ld-row">
              <span class="ldr-label">到期日期</span>
              <el-tag :type="store.licenseStatus==='active'?'success':'warning'" size="small" round>{{ store.license?.expires_at }}</el-tag>
            </div>
            <div class="ld-row">
              <span class="ldr-label">剩余天数</span>
              <span class="ldr-val days" :class="{expiring:store.daysLeft<=7,warning:store.daysLeft<=30}">{{ store.daysLeft }} 天</span>
            </div>
            <div class="ld-row">
              <span class="ldr-label">状态</span>
              <el-tag :type="statusTagType" size="small" round>{{ statusText }}</el-tag>
            </div>
          </div>
        </div>

        <!-- 右侧仪表盘 -->
        <div class="ld-right card-glass">
          <div class="gauge-wrap">
            <svg viewBox="0 0 180 110" class="gauge-svg">
              <defs>
                <linearGradient id="gaugeGrad" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" :stop-color="gaugeColor" />
                  <stop offset="100%" :stop-color="gaugeColor+'88'" />
                </linearGradient>
              </defs>
              <circle cx="90" cy="90" r="55" fill="none" stroke="#f0f0f0" stroke-width="18" :stroke-dasharray="`${280} 350`" stroke-dashoffset="0" />
              <circle cx="90" cy="90" r="55" fill="none" :stroke="gaugeColor" stroke-width="18"
                       :stroke-dasharray="`${280*(gaugePct/100)} 350`" stroke-dashoffset="0"
                       stroke-linecap="round" transform="rotate(-127 90 90)" style="transition: all 0.8s ease;" />
            </svg>
            <div class="gauge-center">
              <span class="gv-num">{{ store.daysLeft }}</span>
              <span class="gv-unit">天</span>
            </div>
          </div>
          <div class="gauge-label" :style="{color:gaugeColor}">{{ gaugeMessage }}</div>
          <!-- 进度条 -->
          <el-progress :percentage="gaugePct" :color="gaugeColor" :stroke-width="8" :show-text="false" />
          <div class="gauge-sub">已使用 {{ usedDays }} / {{ totalDays }} 天</div>
        </div>
      </div>

      <!-- 重新激活 -->
      <div class="reactivate card-glass mt20">
        <div class="reactive-header">
          <el-icon :size="20" color="#e6a23c"><RefreshRight /></el-icon>
          <span>续期或更换授权</span>
        </div>
        <p>如果您已续期或更换设备，可在线同步最新授权状态，或重新导入授权文件</p>
        <div class="reactive-row">
          <el-button type="success" round :loading="syncing" @click="doSync">
            <el-icon><Refresh /></el-icon>在线同步授权
          </el-button>
          <span style="color:#909399;font-size:13px;margin:0 8px;">或</span>
          <el-upload :auto-upload="false" :limit="1" accept=".dat" :on-change="onFile" :show-file-list="false">
            <el-button type="warning" round><el-icon><Upload /></el-icon>上传新授权文件</el-button>
          </el-upload>
          <span v-if="fn" class="reactive-file"><el-icon color="#67c23a"><DocumentChecked /></el-icon>{{ fn }}</span>
          <el-button type="primary" round :loading="importing" :disabled="!lc" @click="doImport">
            <el-icon><Key /></el-icon>更新激活
          </el-button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAppStore } from '@/stores/app'
import { ElMessage } from 'element-plus'
import { Key, CopyDocument, Monitor, UploadFilled, CircleCheck, DocumentChecked, Upload, RefreshRight, Refresh, ChatDotRound } from '@element-plus/icons-vue'

const store = useAppStore()
const actStep = ref(0); const fn = ref(''); const lc = ref(''); const importing = ref(false); const syncing = ref(false)
const licenseKey = ref(''); const activating = ref(false); const activateMsg = ref(''); const activateOk = ref(false)

const statusTagType = computed(()=>{
  if(!store.license||!store.license.is_valid) return 'danger'
  if(store.daysLeft<=7) return 'warning'
  return 'success'
})
const statusText = computed(()=>{
  if(!store.license) return '未授权'
  if(!store.license.is_valid) return '已过期'
  if(store.daysLeft<=7) return '即将到期'
  return '使用中'
})
const gaugePct = computed(()=>{
  if(!store.license||!store.daysLeft) return 0
  return Math.min(100, Math.round(store.daysLeft/365*100))
})
const totalDays = computed(()=>{
  if(!store.license) return 365
  const start = new Date(store.license.issued_at).getTime()
  const end = new Date(store.license.expires_at).getTime()
  return Math.max(1, Math.round((end-start)/(86400000)))
})
const usedDays = computed(()=> Math.max(0, totalDays.value - store.daysLeft))
const gaugeColor = computed(()=>{
  if(store.daysLeft<=7) return '#f56c6c'
  if(store.daysLeft<=30) return '#e6a23c'
  return '#67c23a'
})
const gaugeMessage = computed(()=>{
  if(!store.license) return ''
  if(store.daysLeft<=7) return '即将到期，建议尽快续期'
  if(store.daysLeft<=30) return '授权即将到期'
  return '授权状态良好'
})

function copy(){
  navigator.clipboard.writeText(store.machineCode).then(()=>ElMessage.success('机器码已复制')).catch(()=>ElMessage.info('请手动复制'))
}
function onFile(file:any){
  fn.value = file.name
  const r = new FileReader()
  r.onload = e => { lc.value = (e.target?.result as string)||'' }
  r.readAsText(file.raw)
}
async function doSync(){
  syncing.value = true
  try {
    const online = await store.checkOnlineLicense()
    if (online?.valid) {
      // 重新获取本地授权信息（syncLicenseInfo 已更新 license_info 表）
      await store.fetchLicense()
      ElMessage.success(`同步成功！到期日: ${online.expires_at || '未知'}，剩余 ${online.days_left ?? '?'} 天`)
    } else {
      ElMessage.error(online?.error || '在线同步失败，请检查网络或联系管理员')
    }
  } catch { ElMessage.error('同步失败') }
  finally { syncing.value = false }
}
async function doImport(){
  importing.value = true
  try {
    const r = await store.importLicense(lc.value.trim())
    if(r.ok) ElMessage.success('激活成功！')
    else ElMessage.error(r.msg||'激活失败')
  } catch { ElMessage.error('激活失败') }
  finally { importing.value = false }
}

async function doActivate(){
  if(!licenseKey.value) return
  activating.value = true; activateMsg.value = ''
  try {
    const r = await store.activateOnline(licenseKey.value.trim())
    activateOk.value = r.ok
    activateMsg.value = r.msg || (r.ok ? '激活成功！' : '激活失败')
    if(r.ok) {
      ElMessage.success('在线激活成功！有效期至 ' + (r.expires_at || ''))
    } else {
      ElMessage.error(r.msg || '激活失败')
    }
  } catch { activateMsg.value = '无法连接授权服务器'; activateOk.value = false }
  finally { activating.value = false }
}

onMounted(()=>{ store.fetchMachineCode(); store.fetchLicense() })
</script>

<style scoped>
.license-page { max-width: 860px; margin: 0 auto; }
.mt20 { margin-top: 20px; }

.page-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 28px;
}
.header-left { display: flex; align-items: center; gap: 14px; }
.header-icon-box {
  width: 48px; height: 48px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center; color: #fff;
}
.header-icon-box.warn { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.header-icon-box.active { background: linear-gradient(135deg,#67c23a,#85ce61); }
.header-left h2 { margin: 0; font-size: 22px; font-weight: 700; color: #303133; }
.header-left p { margin: 2px 0 0; font-size: 13px; color: #909399; }

/* 激活步骤 */
.activation-steps { display: flex; align-items: center; justify-content: center; margin-bottom: 28px; gap: 0; }
.act-step { display: flex; align-items: center; gap: 10px; padding: 0 8px; }
.act-num {
  width: 38px; height: 38px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; font-weight: 700;
  background: #e4e7ed; color: #909399; transition: all 0.4s;
}
.act-step.active .act-num { background: linear-gradient(135deg,#4472C4,#2d2854); color: #fff; box-shadow: 0 4px 12px rgba(68,114,196,0.35); }
.act-step.done .act-num { background: #67c23a; color: #fff; }
.act-title { font-size: 14px; font-weight: 600; color: #303133; }
.act-desc { font-size: 11px; color: #c0c4cc; }
.act-step.active .act-title { color: #4472C4; }
.act-line { flex: 1; max-width: 80px; height: 2px; background: #e4e7ed; margin: 0 4px; transition: background 0.4s; }
.act-line.done { background: #67c23a; }

.card-glass { background: #fff; border-radius: 16px; box-shadow: 0 2px 20px rgba(0,0,0,0.06); padding: 36px 32px; }

.act-card { text-align: center; }
.act-card-icon {
  width: 64px; height: 64px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 16px; color: #fff;
}
.act-card-icon.blue { background: linear-gradient(135deg,#4472C4,#5b8def); }
.act-card-icon.orange { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.act-card-icon.green { background: linear-gradient(135deg,#67c23a,#85ce61); }
.act-card h3 { font-size: 18px; font-weight: 600; margin: 0 0 8px; color: #303133; }
.act-card p { color: #909399; font-size: 14px; margin: 0 0 20px; }

.mc-box {
  display: flex; align-items: center; justify-content: center; gap: 12px;
  background: #f5f7fa; padding: 16px 24px; border-radius: 10px;
  margin: 0 auto 20px; max-width: 500px;
}
.mc-box code { font-size: 18px; font-weight: 700; letter-spacing: 1px; color: #4472C4; font-family: 'Consolas',monospace; }

.act-next { margin-top: 8px; }

.contact-list { display: flex; flex-direction: column; gap: 12px; max-width: 400px; margin: 0 auto 20px; text-align: left; }
.contact-item { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #606266; }
.contact-item code { background: #f5f7fa; padding: 2px 8px; border-radius: 4px; font-size: 13px; color: #4472C4; }

.upload-zone-small { max-width: 400px; margin: 0 auto 20px; }
.upload-zone-small.hasfile { border-color: #67c23a; }
.file-chosen {
  display: flex; align-items: center; gap: 8px; justify-content: center;
  margin-top: 12px; color: #67c23a; font-weight: 500;
}

/* 在线激活 */
.key-input-box { max-width: 380px; margin: 0 auto 20px; }
.key-input :deep(.el-input__inner) { text-align: center; font-size: 18px; letter-spacing: 2px; font-family: 'Consolas',monospace; font-weight: 700; }
.act-msg { margin-top: 12px; font-size: 14px; font-weight: 500; color: #67c23a; }
.act-msg.error { color: #f56c6c; }

.divider {
  display: flex; align-items: center; margin: 24px 0 0;
  color: #c0c4cc; font-size: 12px;
}
.divider::before, .divider::after {
  content: ''; flex: 1; height: 1px; background: #e4e7ed;
}
.divider span { padding: 0 16px; }

/* 已授权仪表盘 */
.license-dashboard { display: grid; grid-template-columns: 5fr 4fr; gap: 20px; }

.ld-customer {
  display: flex; align-items: center; gap: 14px;
  padding-bottom: 20px; margin-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
}
.ldc-avatar {
  width: 52px; height: 52px; border-radius: 14px;
  background: linear-gradient(135deg,#4472C4,#2d2854);
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 22px; font-weight: 700;
}
.ldc-name { font-size: 18px; font-weight: 700; color: #303133; }
.ldc-sub { font-size: 12px; color: #909399; margin-top: 2px; }

.ld-details { display: flex; flex-direction: column; gap: 12px; }
.ld-row { display: flex; justify-content: space-between; align-items: center; }
.ldr-label { font-size: 13px; color: #909399; }
.ldr-val { font-size: 14px; color: #303133; font-weight: 500; }
.ldr-val.mono { font-family: 'Consolas',monospace; font-size: 13px; }
.ldr-val.days { font-size: 20px; font-weight: 700; color: #67c23a; }
.ldr-val.days.expiring { color: #f56c6c; }
.ldr-val.days.warning { color: #e6a23c; }

/* 仪表盘 */
.ld-right { text-align: center; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.gauge-wrap { position: relative; width: 180px; height: 120px; }
.gauge-svg { width: 180px; height: 110px; margin-top: -5px; }
.gauge-center {
  position: absolute; bottom: 5px; left: 50%; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: center;
}
.gv-num { font-size: 32px; font-weight: 700; color: #303133; line-height: 1; }
.gv-unit { font-size: 13px; color: #909399; }
.gauge-label { font-size: 13px; margin: 4px 0 12px; }
.gauge-sub { font-size: 12px; color: #c0c4cc; margin-top: 6px; }

/* 重新激活 */
.reactivate { text-align: center; }
.reactive-header { display: flex; align-items: center; justify-content: center; gap: 8px; font-size: 16px; font-weight: 600; color: #303133; margin-bottom: 8px; }
.reactivate p { color: #909399; font-size: 13px; margin: 0 0 16px; }
.reactive-row { display: flex; align-items: center; justify-content: center; gap: 12px; }
.reactive-file { font-size: 13px; color: #67c23a; display: flex; align-items: center; gap: 4px; }
</style>
