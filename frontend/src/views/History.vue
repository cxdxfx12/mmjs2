<template>
  <div class="hp">
    <div class="ph">
      <h3>计算历史记录</h3>
      <el-space>
        <el-input v-model="search" placeholder="搜索文件名..." clearable size="default" style="width:240px" :prefix-icon="Search"/>
        <el-popconfirm title="确定清空所有记录？" @confirm="clearAll"><template #reference><el-button type="danger" plain size="default">清空全部</el-button></template></el-popconfirm>
      </el-space>
    </div>
    <el-card>
      <el-table :data="filtered" stripe border size="small" v-loading="loading" @row-click="showDetail" highlight-current-row>
        <el-table-column prop="id" label="#" width="60"/>
        <el-table-column prop="created_at" label="计算时间" width="170" sortable/>
        <el-table-column label="源文件" min-width="200" show-overflow-tooltip><template #default="{row}"><span style="color:#4472C4;cursor:pointer">{{ baseName(row.input_file) }}</span></template></el-table-column>
        <el-table-column label="输出文件" min-width="180" show-overflow-tooltip><template #default="{row}"><span v-if="row.output_file" style="color:#67c23a">{{ baseName(row.output_file) }}</span><span v-else style="color:#c0c4cc">未导出</span></template></el-table-column>
        <el-table-column prop="total_count" label="件数" width="100"/>
        <el-table-column label="总运费" width="110"><template #default="{row}">¥{{ (row.total_fee||0).toFixed(2) }}</template></el-table-column>
        <el-table-column label="均价" width="90"><template #default="{row}">¥{{ (row.avg_fee||0).toFixed(2) }}</template></el-table-column>
        <el-table-column label="最高" width="90"><template #default="{row}">¥{{ (row.max_fee||0).toFixed(2) }}</template></el-table-column>
        <el-table-column label="最低" width="90"><template #default="{row}">¥{{ (row.min_fee||0).toFixed(2) }}</template></el-table-column>
        <el-table-column prop="calc_duration" label="耗时" width="80"><template #default="{row}">{{ row.calc_duration }}s</template></el-table-column>
        <el-table-column prop="rule_summary" label="规则" width="100"/>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{row}"><el-popconfirm title="确定删除？" @click.stop @confirm="del(row.id)"><template #reference><el-button link type="danger" size="small" @click.stop>删除</el-button></template></el-popconfirm></template>
        </el-table-column>
      </el-table>
      <div v-if="filtered.length===0" style="text-align:center;padding:60px 0;color:#909399">
        <el-icon :size="48"><FolderOpened /></el-icon>
        <p>暂无计算记录</p>
        <el-button type="primary" @click="$router.push('/calc')">开始第一次计费</el-button>
      </div>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog v-model="dd" title="历史记录详情" width="550px">
      <el-descriptions :column="2" border size="small" v-if="detail">
        <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
        <el-descriptions-item label="时间">{{ detail.created_at }}</el-descriptions-item>
        <el-descriptions-item label="源文件" :span="2">{{ baseName(detail.input_file) }}</el-descriptions-item>
        <el-descriptions-item label="输出文件" :span="2">{{ detail.output_file ? baseName(detail.output_file) : '未导出' }}</el-descriptions-item>
        <el-descriptions-item label="件数">{{ detail.total_count?.toLocaleString() }}</el-descriptions-item>
        <el-descriptions-item label="总运费">¥{{ (detail.total_fee||0).toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="均价">¥{{ (detail.avg_fee||0).toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="最高运费">¥{{ (detail.max_fee||0).toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="最低运费">¥{{ (detail.min_fee||0).toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ detail.calc_duration }}秒</el-descriptions-item>
        <el-descriptions-item label="规则">{{ detail.rule_summary }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, FolderOpened } from '@element-plus/icons-vue'
const history = ref<any[]>([])
const loading = ref(false); const search = ref('')
const dd = ref(false); const detail = ref<any>(null)
const filtered = computed(()=>{ if(!search.value)return history.value; const s=search.value.toLowerCase(); return history.value.filter(h=>(h.input_file||'').toLowerCase().includes(s)||(h.output_file||'').toLowerCase().includes(s)) })
function baseName(p:string){ if(!p)return'-'; const a=p.replace(/\\/g,'/').split('/'); return a[a.length-1]||p }
function aHeaders(){ return { Authorization: `Bearer ${localStorage.getItem('yunfei_token')||''}` } }
async function fetchData(){ loading.value=true; try{ const r=await fetch('/api/history',{headers:aHeaders()}); history.value=await r.json() }catch{}finally{ loading.value=false } }
async function del(id:number){ await fetch('/api/history',{method:'DELETE',headers:{...aHeaders(),'Content-Type':'application/json'},body:JSON.stringify({id})}); ElMessage.success('已删除'); fetchData() }
async function clearAll(){ await fetch('/api/history/clear',{method:'POST',headers:aHeaders()}); ElMessage.success('已清空'); fetchData() }
async function showDetail(row:any){ detail.value=row; dd.value=true }
onMounted(fetchData)
</script>

<style scoped>
.hp{display:flex;flex-direction:column;gap:16px}
.ph{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px}
</style>
