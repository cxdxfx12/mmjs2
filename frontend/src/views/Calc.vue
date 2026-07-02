<template>
  <div class="calc-page">
    <!-- 步骤指示器 -->
    <div class="steps-bar">
      <div class="step-item" :class="{ active: step >= 0, done: step > 0 }" @click="step > 0 && goStep(0)">
        <div class="step-num"><el-icon v-if="step>0"><Check /></el-icon><span v-else>1</span></div>
        <div class="step-text"><div class="step-title">上传文件</div><div class="step-desc">最多5个</div></div>
      </div>
      <div class="step-line" :class="{ done: step > 0 }" />
      <div class="step-item" :class="{ active: step >= 1, done: step > 1 }" @click="step > 1 && goStep(1)">
        <div class="step-num"><el-icon v-if="step>1"><Check /></el-icon><span v-else>2</span></div>
        <div class="step-text"><div class="step-title">预览确认</div><div class="step-desc">核对数据</div></div>
      </div>
      <div class="step-line" :class="{ done: step > 1 }" />
      <div class="step-item" :class="{ active: step >= 2, done: step > 2 }">
        <div class="step-num"><span>3</span></div>
        <div class="step-text"><div class="step-title">计算运费</div><div class="step-desc">并行处理</div></div>
      </div>
      <div class="step-line" />
      <div class="step-item" :class="{ active: step >= 3 }">
        <div class="step-num"><span>4</span></div>
        <div class="step-text"><div class="step-title">导出结果</div><div class="step-desc">分别下载</div></div>
      </div>
    </div>

    <!-- Step 0: 上传多文件 -->
    <transition name="fade-slide" mode="out-in">
      <div v-if="step === 0" key="upload" class="step-card card-glass">
        <div class="upload-zone" @click="selectFiles" @dragover.prevent @drop.prevent="onDrop"
             :class="{ dragover }">
          <div class="upload-icon-box"><el-icon :size="48"><UploadFilled /></el-icon></div>
          <h3>{{ dragover ? '释放文件以上传' : '点击或拖拽 Excel 文件（最多5个）' }}</h3>
          <p class="hint">支持 .xlsx / .xls，单文件最大 100MB，多文件并行计算</p>
          <input ref="fileInput" type="file" accept=".xlsx,.xls" multiple style="display:none" @change="onFilesSelected" />
          <el-button type="primary" size="large" round @click.stop="selectFiles" :loading="uploading" class="upload-btn">
            <el-icon><FolderOpened /></el-icon>{{ uploading ? '上传中...' : '浏览选择文件' }}
          </el-button>
          <!-- 已选文件列表 -->
          <div v-if="fileList.length" class="file-chips">
            <div v-for="(f,i) in fileList" :key="i" class="file-chip">
              <el-icon class="chip-icon"><Document /></el-icon>
              <span class="chip-name">{{ f.name }}</span>
              <span class="chip-progress" v-if="uploading">&nbsp;{{ uploadPct }}%</span>
              <el-icon class="chip-close" @click.stop="removeFile(i)"><Close /></el-icon>
            </div>
          </div>
          <!-- 上传进度条 -->
          <div v-if="uploading" class="upload-progress">
            <el-progress :percentage="uploadPct" :stroke-width="6" :color="progressColors" />
            <span class="progress-text">{{ uploadProgressText }}</span>
          </div>
        </div>
        <div v-if="recentFiles.length" class="recent-section">
          <el-divider><span style="color:#909399;font-size:13px">最近使用</span></el-divider>
          <div class="recent-list">
            <div v-for="(f,i) in recentFiles" :key="i" class="recent-item" @click="quickOpen(f)">
              <el-icon><Document /></el-icon><span>{{ f.name }}</span>
              <el-icon class="recent-arrow"><ArrowRight /></el-icon>
            </div>
          </div>
        </div>
        <div v-if="fileList.length >= 1" class="action-bar">
          <el-button @click="fileList=[]" round>清空列表</el-button>
          <el-button type="primary" size="large" @click="goPreview" :loading="loadingPreview" :disabled="!fileList.length" round>
            <el-icon><DataAnalysis /></el-icon>预览确认 ({{ fileList.length }} 个文件)
          </el-button>
        </div>
      </div>

      <!-- Step 1: 多文件预览 -->
      <div v-else-if="step === 1" key="preview" class="step-card card-glass">
        <div class="preview-container">
          <div class="preview-header-bar">
            <h3><el-icon :size="22" color="#4472C4"><DataAnalysis /></el-icon>文件预览确认</h3>
            <el-tag type="info" size="small" round>{{ fileList.length }} 个文件</el-tag>
          </div>
          <!-- 整体读取进度 -->
          <div v-if="previewLoading" class="preview-loading">
            <el-icon class="loading-icon-spin" :size="32" color="#4472C4"><Loading /></el-icon>
            <el-progress :percentage="previewProgress" :stroke-width="8" :color="progressColors" />
            <p>正在读取文件信息... {{ previewProgress }}%</p>
          </div>
          <div v-for="(p,i) in previews" :key="i" class="preview-card" :class="{ 'has-error': p.error }">
            <div class="preview-card-header">
              <el-icon :size="20" :color="p.error?'#f56c6c':'#4472C4'"><component :is="p.error?WarningFilled:DocumentChecked" /></el-icon>
              <span class="preview-fname">{{ p.name }}</span>
              <el-button text size="small" type="danger" @click="removeFile(i)">移除</el-button>
            </div>
            <div v-if="p.error" class="preview-error">{{ p.error }}</div>
            <div v-else class="preview-info">
              <el-tag effect="dark" round type="success" size="small">{{ p.customers?.length || 0 }}客户</el-tag>
              <el-tag effect="dark" round type="warning" size="small">{{ p.provinces?.length || 0 }}省份</el-tag>
              <el-tag effect="dark" round type="info" size="small">{{ (p.total_rows||0).toLocaleString() }}行</el-tag>
              <el-tag effect="plain" size="small" v-for="c in (p.columns||[]).slice(0,5)" :key="c" class="col-chip">{{ c }}</el-tag>
              <el-tag v-if="(p.columns||[]).length>5" effect="plain" size="small">+{{ p.columns.length-5 }}</el-tag>
            </div>
            <!-- 数据采样 -->
            <el-table v-if="!p.error && p.samples?.length" :data="p.samples.slice(0,5)" border size="small" class="preview-table">
              <el-table-column v-for="(c,ci) in (p.columns||[]).slice(0,8)" :key="ci" :label="c" min-width="100" show-overflow-tooltip>
                <template #default="{row}">{{ row[ci] }}</template>
              </el-table-column>
            </el-table>
          </div>
          <div class="action-bar">
            <el-button @click="goStep(0)" :icon="ArrowLeft" round>重新选择</el-button>
            <el-button type="primary" size="large" @click="startBatchCalc" :loading="calculating"
                       :disabled="!validFileCount" :icon="Coin" round>
              {{ calculating ? '计算中...' : `开始计算 ${validFileCount} 个文件运费` }}
            </el-button>
          </div>
        </div>
      </div>

      <!-- Step 2: 并行计算进度 -->
      <div v-else-if="step === 2" key="calc" class="step-card card-glass">
        <div class="calc-header">
          <h3><el-icon :size="22"><Loading :class="{spinning:calculating}" color="#e6a23c" /></el-icon>并行计算中...</h3>
          <el-tag v-if="calculating" type="warning" size="small" round>{{ doneCount }}/{{ fileList.length }} 完成</el-tag>
          <el-tag v-else type="success" size="small" round>全部完成</el-tag>
        </div>
        <!-- 每个文件的独立进度条 -->
        <div v-for="(t,i) in calcTasks" :key="t.task_id||i" class="task-progress-item">
          <div class="task-progress-header">
            <el-icon :size="18" :color="t.phase==='done'?'#67c23a':t.phase==='error'?'#f56c6c':t.phase==='calculating'?'#e6a23c':'#4472C4'">
              <component :is="t.phase==='done'?CircleCheck:t.phase==='error'?CircleClose:t.phase==='waiting'?Clock:Loading" :class="{spinning:t.phase==='calculating'||t.phase==='reading'}" />
            </el-icon>
            <span class="task-name">{{ fileList[i]?.name }}</span>
            <span class="task-status">{{ t.message }}</span>
          </div>
          <el-progress :percentage="t.pct||0" :stroke-width="8"
                       :color="t.phase==='done'?'#67c23a':t.phase==='error'?'#f56c6c':t.phase==='calculating'?'#e6a23c':'#4472C4'"
                       :status="t.phase==='error'?'exception':undefined" />
          <div v-if="t.current&&t.total" class="task-detail">{{ t.current?.toLocaleString() }} / {{ t.total?.toLocaleString() }}</div>
        </div>
      </div>

      <!-- Step 3: 结果展示 + 分别导出 -->
      <div v-else-if="step === 3" key="result" class="step-card card-glass">
        <div class="result-header">
          <h3><el-icon :size="22" color="#67c23a"><CircleCheck /></el-icon>计算完成</h3>
          <el-button type="primary" size="small" round @click="exportAll" :loading="exportingAll">
            <el-icon><Download /></el-icon>一键导出全部
          </el-button>
        </div>

        <!-- 每个文件的结果卡片 -->
        <div v-for="(ri,i) in batchResults" :key="i" class="result-file-card">
          <div class="result-file-header" @click="expandedFile = expandedFile===i ? -1 : i">
            <el-icon :size="20" color="#4472C4"><Document /></el-icon>
            <span class="rf-name">{{ ri.file_name }}</span>
            <span class="rf-total" v-if="ri.summary">¥{{ (ri.summary.total_fee||0).toLocaleString(undefined,{minimumFractionDigits:2}) }}</span>
            <el-icon class="rf-expand" :class="{rotated:expandedFile===i}"><ArrowDown /></el-icon>
          </div>
          <div v-if="expandedFile===i && ri.summary" class="result-file-body">
            <div class="summary-cards">
              <div class="sum-card sum-blue">
                <div class="sum-val">{{ ri.summary.total_count?.toLocaleString() }}</div>
                <div class="sum-label">总件数</div>
              </div>
              <div class="sum-card sum-green">
                <div class="sum-val">¥{{ (ri.summary.total_fee||0).toLocaleString(undefined,{minimumFractionDigits:2}) }}</div>
                <div class="sum-label">总运费</div>
              </div>
              <div class="sum-card sum-orange">
                <div class="sum-val">¥{{ (ri.summary.avg_fee||0).toFixed(2) }}</div>
                <div class="sum-label">平均运费</div>
              </div>
              <div class="sum-card sum-gray">
                <div class="sum-val">{{ ri.summary.duration_sec }}s</div>
                <div class="sum-label">耗时</div>
              </div>
            </div>
            <!-- 省份/客户汇总 -->
            <el-tabs type="border-card" class="result-tabs">
              <el-tab-pane label="按省份">
                <el-table :data="topProvinces(ri.summary)" stripe size="small" max-height="240">
                  <el-table-column prop="name" label="省份" />
                  <el-table-column prop="count" label="件数" width="90" />
                  <el-table-column label="运费(元)" width="130">
                    <template #default="{row}">¥{{ row.fee?.toLocaleString(undefined,{minimumFractionDigits:2}) }}</template>
                  </el-table-column>
                </el-table>
              </el-tab-pane>
              <el-tab-pane label="按客户">
                <el-table :data="topCustomers(ri.summary)" stripe size="small" max-height="240">
                  <el-table-column prop="name" label="客户" />
                  <el-table-column prop="count" label="件数" width="90" />
                  <el-table-column label="运费(元)" width="130">
                    <template #default="{row}">¥{{ row.fee?.toLocaleString(undefined,{minimumFractionDigits:2}) }}</template>
                  </el-table-column>
                </el-table>
              </el-tab-pane>
            </el-tabs>
            <el-button type="success" size="default" round @click="exportSingle(i)" :loading="exportingIdx===i" class="export-single-btn">
              <el-icon><Download /></el-icon>导出 {{ ri.file_name }}
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onActivated } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UploadFilled, FolderOpened, Coin, DataAnalysis, Download, Document, Loading, ArrowLeft, ArrowRight, ArrowDown, Check, CircleCheck, CircleClose, Close, Clock, WarningFilled, DocumentChecked } from '@element-plus/icons-vue'
import { useAppStore } from '@/stores/app'

defineOptions({ name: 'CalcView' })

interface FileItem { name: string; path: string }
interface TaskProgressItem {
  task_id?: string; phase: string; pct: number; current?: number; total?: number; message: string; error?: string
}

const store = useAppStore()
const step = ref(0)
const uploading = ref(false); const loadingPreview = ref(false); const previewLoading = ref(false)
const calculating = ref(false); const exportingAll = ref(false); const exportingIdx = ref(-1)
const dragover = ref(false)
const uploadPct = ref(0); const previewProgress = ref(0)
const fileList = ref<FileItem[]>([]); const recentFiles = ref<FileItem[]>([]);
const previews = ref<any[]>([]); const calcTasks = ref<TaskProgressItem[]>([])
const batchResults = ref<any[]>([]); const expandedFile = ref(-1)
const fileInput = ref<HTMLInputElement>()
let pollTimer: any = null
let activeBatchId = ''  // 当前活跃的批次ID

const progressColors = [{color:'#4472C4',percentage:30},{color:'#67c23a',percentage:70},{color:'#e6a23c',percentage:100}]
const uploadProgressText = computed(() => fileList.value.length ? `${fileList.value.length} 个文件` : '')
const validFileCount = computed(() => previews.value.filter(p => !p.error).length)
const doneCount = computed(() => calcTasks.value.filter(t => t.phase==='done').length)

function topProvinces(s:any) {
  if(!s?.by_province) return []
  return Object.entries(s.by_province).map(([n,v]:any)=>({name:n,count:v.count,fee:v.fee})).sort((a:any,b:any)=>b.fee-a.fee).slice(0,10)
}
function topCustomers(s:any) {
  if(!s?.by_customer) return []
  return Object.entries(s.by_customer).map(([n,v]:any)=>({name:n,count:v.count,fee:v.fee})).sort((a:any,b:any)=>b.fee-a.fee).slice(0,10)
}

// 保存/清除活跃批次到 localStorage
function saveActiveBatch(batchId: string, files: FileItem[]) {
  activeBatchId = batchId
  localStorage.setItem('yunfei_calc_batch', JSON.stringify({ batch_id: batchId, files, ts: Date.now() }))
  store.calculating = true
}
function clearActiveBatch() {
  activeBatchId = ''
  localStorage.removeItem('yunfei_calc_batch')
  store.calculating = false
}

// 离开页面确认
onBeforeRouteLeave((_to, _from, next) => {
  if (calculating.value) {
    ElMessageBox.confirm('计算正在进行中，后台将继续执行不会中断。确定离开吗？', '提示', {
      confirmButtonText: '确定离开', cancelButtonText: '留在当前页', type: 'warning'
    }).then(() => next()).catch(() => next(false))
  } else {
    next()
  }
})

function cleanup() { if (pollTimer) { clearInterval(pollTimer); pollTimer = null } }
onUnmounted(() => cleanup())

// keep-alive 重新激活时确保轮询恢复
onActivated(() => {
  if (step.value === 2 && activeBatchId && !pollTimer) {
    resumePolling(activeBatchId)
  }
})

// 恢复活跃批次
onMounted(async () => {
  try {
    const saved = JSON.parse(localStorage.getItem('yunfei_calc_batch') || '')
    if (!saved?.batch_id) return
    // 超过30分钟的批次视为过期
    if (Date.now() - saved.ts > 30 * 60 * 1000) {
      clearActiveBatch()
      return
    }
    activeBatchId = saved.batch_id
    fileList.value = saved.files || []
    step.value = 2
    calculating.value = true
    store.calculating = true
    calcTasks.value = (saved.files || []).map(() => ({ phase: 'waiting', pct: 0, message: '恢复中...' }))
    resumePolling(activeBatchId)
  } catch { clearActiveBatch() }
})

// 恢复轮询
async function resumePolling(batchId: string) {
  const token = localStorage.getItem('yunfei_token') || ''
  pollTimer = setInterval(async () => {
    try {
      const pr = await fetch(`/api/calculate/batch-progress?batch_id=${batchId}`, {
        headers: { Authorization: `Bearer ${token}` }
      })
      const prog = await pr.json()
      if (!prog.tasks) return
      calcTasks.value = prog.tasks.map((t: any) => ({
        task_id: t.task_id, phase: t.phase, pct: t.pct || 0,
        current: t.current, total: t.total, message: t.message, error: t.error
      }))
      if (prog.all_done) {
        cleanup()
        const rr = await fetch(`/api/calculate/batch-result?batch_id=${batchId}`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        batchResults.value = await rr.json()
        calculating.value = false
        clearActiveBatch()
        step.value = 3
        ElMessage.success('全部计算完成！')
      }
    } catch { /* 网络错误静默继续 */ }
  }, 400)
}

try { recentFiles.value = JSON.parse(localStorage.getItem('yunfei_recent')||'[]') } catch {}
function saveRecent(n:string,p:string){ const l=recentFiles.value.filter(f=>f.path!==p); l.unshift({name:n,path:p}); if(l.length>5)l.pop(); recentFiles.value=l; localStorage.setItem('yunfei_recent',JSON.stringify(l)) }
function goStep(s:number) { step.value = s }
function selectFiles(){ fileInput.value?.click() }
function removeFile(i:number){ fileList.value.splice(i,1) }

function onDrop(e:DragEvent){
  dragover.value = false
  const files = e.dataTransfer?.files
  if(files) addFiles(Array.from(files))
}
async function onFilesSelected(e:Event){
  const files = (e.target as HTMLInputElement).files
  if(!files?.length) return
  await addFiles(Array.from(files))
}

async function addFiles(files: File[]) {
  if(files.length + fileList.value.length > 5){
    ElMessage.warning('一次最多选择5个文件')
    return
  }
  uploading.value = true; uploadPct.value = 0
  const token = localStorage.getItem('yunfei_token')||''
  try {
    const fd = new FormData()
    files.forEach(f => fd.append('files', f))
    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/excel/upload')
    xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.upload.onprogress = (e) => {
      if(e.lengthComputable) uploadPct.value = Math.round(e.loaded/e.total*100)
    }
    const resp: any = await new Promise((resolve, reject) => {
      xhr.onload = () => { try{resolve(JSON.parse(xhr.responseText))}catch{reject(new Error('解析失败'))} }
      xhr.onerror = () => reject(new Error('上传失败'))
      xhr.send(fd)
    })
    if(resp.error){ ElMessage.error(resp.error); return }
    if(!resp.files?.length){ ElMessage.error('上传失败'); return }
    // 合并到文件列表（去重）
    for(const f of resp.files){
      if(!fileList.value.some(x=>x.path===f.path)){
        fileList.value.push({name:f.name, path:f.path})
        saveRecent(f.name, f.path)
      }
    }
    ElMessage.success(`已添加 ${resp.files.length} 个文件`)
  } catch(e:any){ ElMessage.error('上传失败: '+(e.message||e)) }
  finally { uploading.value = false }
}

async function quickOpen(f:FileItem){
  if(!fileList.value.some(x=>x.path===f.path)){
    fileList.value.push(f)
  }
}

async function goPreview(){
  loadingPreview.value = true; previewLoading.value = true; previewProgress.value = 0
  step.value = 1
  try {
    const token = localStorage.getItem('yunfei_token')||''
    // 逐个读取预览（带进度）
    const results: any[] = []
    for(let i=0; i<fileList.value.length; i++){
      previewProgress.value = Math.round((i/fileList.value.length)*100)
      const res = await fetch('/api/excel/preview', {
        method:'POST',
        headers:{'Content-Type':'application/json', Authorization:`Bearer ${token}`},
        body: JSON.stringify({path:fileList.value[i].path})
      })
      const d = await res.json()
      results.push({ ...d, name: fileList.value[i].name, path: fileList.value[i].path })
    }
    previewProgress.value = 100
    previews.value = results
  } catch(e){ ElMessage.error('预览失败') }
  finally { loadingPreview.value = false; previewLoading.value = false }
}

async function startBatchCalc(){
  step.value = 2; calculating.value = true; store.calculating = true
  calcTasks.value = fileList.value.map(()=>({phase:'waiting',pct:0,message:'等待中...'}))
  const token = localStorage.getItem('yunfei_token')||''
  try {
    const res = await fetch('/api/calculate/batch', {
      method:'POST',
      headers:{'Content-Type':'application/json', Authorization:`Bearer ${token}`},
      body: JSON.stringify({files: fileList.value})
    })
    const { batch_id, tasks } = await res.json()
    if(!batch_id){ ElMessage.error('启动计算失败'); calculating.value=false; store.calculating=false; return }
    // 持久化批次信息
    saveActiveBatch(batch_id, fileList.value)
    // 初始化任务列表
    calcTasks.value = (tasks||[]).map((t:any)=>({task_id:t.task_id,phase:'reading',pct:0,message:'启动中...'}))
    // 开始轮询
    resumePolling(batch_id)
  } catch(e:any){
    ElMessage.error('启动失败: '+(e.message||e))
    calculating.value = false
    store.calculating = false
  }
}

async function exportSingle(idx: number) {
  exportingIdx.value = idx
  const ri = batchResults.value[idx]
  try {
    const token = localStorage.getItem('yunfei_token')||''
    const res = await fetch(`/api/export?task_id=${encodeURIComponent(ri.task_id)}`, {
      headers:{Authorization:`Bearer ${token}`}
    })
    if(!res.ok){ ElMessage.error('导出失败'); return }
    const blob = await res.blob()
    const name = (ri.file_name||'结果').replace(/\.xlsx?$/,'')+'_结算结果.xlsx'
    downloadBlob(blob, name)
    ElMessage.success('导出成功')
  } catch{ ElMessage.error('导出失败') }
  finally { exportingIdx.value = -1 }
}

async function exportAll(){
  exportingAll.value = true
  try {
    for(let i=0; i<batchResults.value.length; i++){
      await exportSingle(i)
    }
    ElMessage.success(`已导出 ${batchResults.value.length} 个文件`)
  } finally { exportingAll.value = false }
}

function downloadBlob(blob: Blob, name: string){
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = name
  document.body.appendChild(a); a.click()
  document.body.removeChild(a); URL.revokeObjectURL(url)
}
</script>

<style scoped>
.calc-page { max-width: 1100px; margin: 0 auto; }

.steps-bar {
  display: flex; align-items: center; justify-content: center;
  gap: 0; margin-bottom: 28px; padding: 0 20px;
}
.step-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; border-radius: 10px; cursor: default; transition: all 0.3s; }
.step-item.done { cursor: pointer; }
.step-num {
  width: 36px; height: 36px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 700; background: #e4e7ed; color: #909399; transition: all 0.4s;
}
.step-item.active .step-num { background: linear-gradient(135deg,#4472C4,#2d2854); color: #fff; box-shadow: 0 4px 12px rgba(68,114,196,0.35); }
.step-item.done .step-num { background: #67c23a; color: #fff; }
.step-title { font-size: 14px; font-weight: 600; color: #303133; white-space: nowrap; }
.step-desc { font-size: 12px; color: #c0c4cc; }
.step-item.active .step-title { color: #4472C4; }
.step-line { flex: 1; max-width: 60px; height: 2px; background: #e4e7ed; margin: 0 8px; transition: background 0.4s; }
.step-line.done { background: #67c23a; }

.card-glass { background: #fff; border-radius: 16px; box-shadow: 0 2px 20px rgba(0,0,0,0.06); padding: 36px 32px; transition: all 0.3s; }
.card-glass:hover { box-shadow: 0 4px 28px rgba(0,0,0,0.1); }

.upload-zone {
  display: flex; flex-direction: column; align-items: center; gap: 14px;
  padding: 40px 20px; border: 2px dashed #dcdfe6; border-radius: 14px;
  cursor: pointer; transition: all 0.35s;
}
.upload-zone:hover, .upload-zone.dragover { border-color: #4472C4; background: linear-gradient(135deg,#f0f4ff,#e8eeff); }
.upload-icon-box {
  width: 72px; height: 72px; border-radius: 50%;
  background: linear-gradient(135deg,#4472C4,#2d2854);
  display: flex; align-items: center; justify-content: center;
  color: #fff; box-shadow: 0 8px 24px rgba(68,114,196,0.3); margin-bottom: 4px;
}
.upload-zone h3 { font-size: 17px; color: #303133; font-weight: 600; margin: 0; }
.hint { color: #c0c4cc; font-size: 13px; margin: 0; }
.upload-btn { margin-top: 4px; }
.upload-progress { width: 100%; max-width: 400px; display: flex; align-items: center; gap: 12px; margin-top: 4px; }
.progress-text { font-size: 13px; font-weight: 600; color: #4472C4; min-width: 80px; text-align:right; }

/* 文件标签 */
.file-chips { display: flex; flex-wrap: wrap; gap: 8px; justify-content: center; max-width: 600px; }
.file-chip {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 12px; border-radius: 8px; background: #f0f4ff;
  border: 1px solid #d4dfff; font-size: 13px; color: #4472C4;
}
.chip-icon { font-size: 16px; }
.chip-name { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.chip-progress { font-weight: 600; }
.chip-close { font-size: 14px; cursor: pointer; color: #c0c4cc; }
.chip-close:hover { color: #f56c6c; }

.recent-section { margin-top: 20px; }
.recent-list { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.recent-item {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; border-radius: 8px; background: #f5f7fa;
  cursor: pointer; transition: all 0.25s; font-size: 13px; color: #606266;
}
.recent-item:hover { background: #e8eeff; color: #4472C4; }
.recent-arrow { font-size: 12px; color: #c0c4cc; }

/* 预览 */
.preview-header-bar { display: flex; align-items: center; gap: 8px; margin-bottom: 20px; }
.preview-header-bar h3 { margin: 0; font-size: 18px; display: flex; align-items: center; gap: 8px; }
.preview-loading {
  display: flex; flex-direction: column; align-items: center; gap: 16px;
  padding: 48px 0 32px;
}
.preview-loading .el-progress { width: 100%; max-width: 420px; }
.preview-loading p { color: #606266; font-size: 14px; margin: 0; }
.loading-icon-spin { animation: spin 1.2s linear infinite; }
.preview-card {
  border: 1px solid #ebeef5; border-radius: 12px; padding: 16px; margin-bottom: 14px; transition: all 0.2s;
}
.preview-card:hover { border-color: #4472C4; box-shadow: 0 2px 12px rgba(68,114,196,0.08); }
.preview-card.has-error { border-color: #fde2e2; background: #fef0f0; }
.preview-card-header { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.preview-fname { font-size: 15px; font-weight: 600; color: #303133; flex: 1; }
.preview-info { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; margin-bottom: 12px; }
.preview-error { color: #f56c6c; font-size: 13px; margin-bottom: 8px; }
.col-chip { max-width: 100px; overflow: hidden; text-overflow: ellipsis; }
.preview-table { border-radius: 8px; margin-top: 8px; }

/* 计算进度 */
.calc-header { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; }
.calc-header h3 { margin: 0; font-size: 18px; display: flex; align-items: center; gap: 8px; }
.task-progress-item {
  padding: 16px 20px; border-radius: 12px; background: #fafafa;
  border: 1px solid #eee; margin-bottom: 12px;
}
.task-progress-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.task-name { font-size: 14px; font-weight: 600; color: #303133; flex: 1; }
.task-status { font-size: 12px; color: #909399; }
.task-detail { font-size: 12px; color: #c0c4cc; margin-top: 4px; padding-left: 26px; }

/* 结果 */
.result-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 20px; }
.result-header h3 { margin: 0; font-size: 18px; display: flex; align-items: center; gap: 8px; }
.result-file-card {
  border: 1px solid #ebeef5; border-radius: 12px;
  margin-bottom: 12px; overflow: hidden;
}
.result-file-header {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 20px; cursor: pointer; background: #fafafa;
  transition: background 0.2s;
}
.result-file-header:hover { background: #f0f4ff; }
.rf-name { font-size: 15px; font-weight: 600; color: #303133; flex: 1; }
.rf-total { font-size: 18px; font-weight: 700; color: #67c23a; }
.rf-expand { transition: transform 0.3s; color: #c0c4cc; }
.rf-expand.rotated { transform: rotate(180deg); }
.result-file-body { padding: 16px 20px; border-top: 1px solid #ebeef5; }
.export-single-btn { margin-top: 16px; }

.summary-cards { display: grid; grid-template-columns: repeat(4,1fr); gap: 10px; margin-bottom: 14px; }
.sum-card { text-align: center; padding: 14px 8px; border-radius: 10px; color: #fff; }
.sum-val { font-size: 17px; font-weight: 700; }
.sum-label { font-size: 11px; opacity: 0.8; margin-top: 2px; }
.sum-blue { background: linear-gradient(135deg,#4472C4,#5b8def); }
.sum-green { background: linear-gradient(135deg,#67c23a,#85ce61); }
.sum-orange { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.sum-gray { background: linear-gradient(135deg,#909399,#b4b7bd); }

.result-tabs { border-radius: 10px; overflow: hidden; margin-bottom: 0; }

.action-bar {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 24px; padding-top: 20px; border-top: 1px solid #ebeef5;
}

.fade-slide-enter-active { transition: all 0.4s ease; }
.fade-slide-leave-active { transition: all 0.25s ease; }
.fade-slide-enter-from { opacity: 0; transform: translateY(20px); }
.fade-slide-leave-to { opacity: 0; transform: translateY(-10px); }

.spinning { animation: spin 1s linear infinite; }
@keyframes spin { 100% { transform: rotate(360deg); } }
</style>
