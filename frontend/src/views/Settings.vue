<template>
  <div class="settings-page">
    <div class="page-header">
      <div class="header-left">
        <div class="header-icon-box"><el-icon :size="22"><Tools /></el-icon></div>
        <div><h2>系统设置</h2><p>管理账号、数据和系统配置</p></div>
      </div>
    </div>

    <!-- 账号安全 -->
    <div class="section-card">
      <div class="section-header">
        <div class="section-icon sa"><el-icon :size="18"><UserFilled /></el-icon></div>
        <span>账号安全</span>
      </div>
      <el-form label-width="100px" class="settings-form">
        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="管理员用户名">
              <el-input v-model="s.admin_user" placeholder="admin" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="管理员密码">
              <el-input v-model="s.admin_pass" type="password" placeholder="至少6位" show-password />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="密钥种子">
          <el-input v-model="s.auth_secret" placeholder="用于生成 Token 的加密种子" />
          <div class="field-tip">修改后所有用户需重新登录</div>
        </el-form-item>
      </el-form>
    </div>

    <!-- 授权服务器地址 -->
    <div class="section-card mt20">
      <div class="section-header">
        <div class="section-icon se"><el-icon :size="18"><Link /></el-icon></div>
        <span>授权服务器地址</span>
      </div>
      <el-form label-width="100px" class="settings-form">
        <el-form-item label="主服务器">
          <el-input v-model="s.server_url" placeholder="http://www.hbdxm.com/yunfei_api" />
          <div class="field-tip">授权验证优先使用的服务器地址</div>
        </el-form-item>
        <el-form-item label="备用服务器">
          <el-input v-model="s.backup_server_url" placeholder="主服务器失效时自动切换，可不填" />
          <div class="field-tip">主服务器连接失败时自动使用此地址</div>
        </el-form-item>
        <el-form-item label="API 密钥">
          <el-input v-model="s.api_secret" placeholder="与服务器 yunfei_settings 表 api_secret 一致" show-password />
          <div class="field-tip">签名密钥，必须与服务器数据库中的 api_secret 一致，否则验证失败</div>
        </el-form-item>
      </el-form>
    </div>

    <!-- 计费默认值 -->
    <div class="section-card mt20">
      <div class="section-header">
        <div class="section-icon sb"><el-icon :size="18"><Coin /></el-icon></div>
        <span>新建规则默认值</span>
      </div>
      <el-form label-width="100px" class="settings-form">
        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="默认首重(kg)">
              <el-input-number v-model="s.first_weight" :min="0.1" :step="0.1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="默认首重单价(元)">
              <el-input-number v-model="s.first_price" :min="0" :precision="2" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="默认续重单价(元)">
              <el-input-number v-model="s.cont_price" :min="0" :precision="2" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="默认续重模式">
          <el-radio-group v-model="s.mode">
            <el-radio-button value="full_kg">全续（元/kg）</el-radio-button>
            <el-radio-button value="hundred_gram">百克续（元/100g）</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
    </div>

    <!-- 数据管理 -->
    <div class="section-card mt20">
      <div class="section-header">
        <div class="section-icon sc"><el-icon :size="18"><Delete /></el-icon></div>
        <span>数据管理</span>
      </div>
      <div class="data-actions">
        <div class="data-action-item">
          <div class="da-info">
            <div class="da-title">清除计算历史</div>
            <div class="da-desc">移除所有历史计算记录，释放存储空间</div>
          </div>
          <el-popconfirm title="确定清除所有计算历史？此操作不可撤销" @confirm="clearHistory">
            <template #reference><el-button type="danger" plain round>清除历史</el-button></template>
          </el-popconfirm>
        </div>
        <div class="data-action-item">
          <div class="da-info">
            <div class="da-title">重置计费规则</div>
            <div class="da-desc">恢复到默认的 Zone 分区费率（含蜜丝婷协议价）</div>
          </div>
          <el-popconfirm title="确定重置所有规则？自定义规则将丢失" @confirm="resetRules">
            <template #reference><el-button type="warning" plain round>重置规则</el-button></template>
          </el-popconfirm>
        </div>
        <div class="data-action-item">
          <div class="da-info">
            <div class="da-title">清除临时文件</div>
            <div class="da-desc">清理上传缓存与导出残留文件</div>
          </div>
          <el-button plain round @click="clearTemp">清除临时</el-button>
        </div>
        <div class="data-action-item">
          <div class="da-info">
            <div class="da-title">重置授权状态</div>
            <div class="da-desc">清除激活缓存，恢复到未激活状态（发给别人前重置）</div>
          </div>
          <el-popconfirm title="确定重置授权？需重新激活才能继续使用" @confirm="resetLicense">
            <template #reference><el-button type="danger" plain round>重置授权</el-button></template>
          </el-popconfirm>
        </div>
      </div>
    </div>

    <!-- 关于 -->
    <div class="section-card mt20 about-section">
      <div class="section-header">
        <div class="section-icon sd"><el-icon :size="18"><InfoFilled /></el-icon></div>
        <span>关于喵喵云结算</span>
      </div>
      <div class="about-grid">
        <div class="about-item">
          <div class="about-icon"><el-icon><Box /></el-icon></div>
          <div class="about-info"><span class="about-label">版本</span><span class="about-val">v1.3.0</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><Aim /></el-icon></div>
          <div class="about-info"><span class="about-label">定位</span><span class="about-val">快递运费结算软件</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><Coin /></el-icon></div>
          <div class="about-info"><span class="about-label">计费</span><span class="about-val">百克续 / 全续双模式</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><TrendCharts /></el-icon></div>
          <div class="about-info"><span class="about-label">规则</span><span class="about-val">活动>客户>分省默认4级</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><FolderOpened /></el-icon></div>
          <div class="about-info"><span class="about-label">存储</span><span class="about-val">本地 SQLite，数据不上传</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><Lock /></el-icon></div>
          <div class="about-info"><span class="about-label">授权</span><span class="about-val">RSA+AES 离线 + 联网防篡改</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><Connection /></el-icon></div>
          <div class="about-info"><span class="about-label">引擎</span><span class="about-val">Go 并行计算，5文件同时</span></div>
        </div>
        <div class="about-item">
          <div class="about-icon"><el-icon><Monitor /></el-icon></div>
          <div class="about-info"><span class="about-label">前端</span><span class="about-val">Vue3 + Element Plus</span></div>
        </div>
      </div>
    </div>

    <!-- 保存 -->
    <div class="save-bar">
      <el-button type="primary" size="large" round @click="save" :loading="saving" :icon="CircleCheck">
        {{ saving ? '保存中...' : '保存设置' }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Tools, UserFilled, Coin, Delete, InfoFilled, Box, Aim, TrendCharts, FolderOpened, Lock, Connection, Monitor, CircleCheck, Link } from '@element-plus/icons-vue'

function aHeaders(){ return { Authorization: `Bearer ${localStorage.getItem('yunfei_token')||''}` } }
const s = reactive<any>({first_weight:1, first_price:5, cont_price:2, mode:'full_kg', admin_user:'', admin_pass:'', auth_secret:'', server_url:'', backup_server_url:'', api_secret:''})
const saving = ref(false)

onMounted(async()=>{
  try{
    const r = await fetch('/api/settings', {headers:aHeaders()})
    const d = await r.json()
    if(d.first_weight!=null) {
      Object.assign(s, d)
      if(s.admin_pass) s.admin_pass = ''
    }
  }catch{}
  // 加载授权服务器地址
  try {
    const r2 = await fetch('/api/license/server-info')
    const d2 = await r2.json()
    if(d2.server_url) s.server_url = d2.server_url
    if(d2.backup_server_url) s.backup_server_url = d2.backup_server_url
  }catch{}
  // 加载 API 密钥
  try {
    const r3 = await fetch('/api/license/api-secret')
    const d3 = await r3.json()
    if(d3.api_secret) s.api_secret = d3.api_secret
  }catch{}
})

async function save(){
  saving.value = true
  try {
    const payload = { ...s }
    if(!payload.admin_pass) delete payload.admin_pass
    await fetch('/api/settings', {
      method:'POST',
      headers:{...aHeaders(),'Content-Type':'application/json'},
      body:JSON.stringify(payload)
    })
    // 保存主服务器地址
    if(s.server_url) {
      await fetch('/api/license/set-server', {
        method:'POST',
        headers:{...aHeaders(),'Content-Type':'application/json'},
        body:JSON.stringify({url:s.server_url})
      })
    }
    // 保存备用服务器地址
    await fetch('/api/license/set-backup-server', {
      method:'POST',
      headers:{...aHeaders(),'Content-Type':'application/json'},
      body:JSON.stringify({url:s.backup_server_url||''})
    })
    // 保存 API 密钥
    await fetch('/api/license/api-secret', {
      method:'POST',
      headers:{...aHeaders(),'Content-Type':'application/json'},
      body:JSON.stringify({api_secret:s.api_secret||''})
    })
    ElMessage.success('设置已保存')
  } catch { ElMessage.error('保存失败') }
  finally { saving.value = false }
}

async function clearHistory(){
  await fetch('/api/history/clear', {method:'POST',headers:aHeaders()})
  ElMessage.success('历史记录已清除')
}
async function clearTemp(){ ElMessage.success('临时文件已清除') }
async function resetLicense(){
  try {
    const r = await fetch('/api/license/reset', {method:'POST',headers:aHeaders()})
    const d = await r.json()
    if(d.ok) ElMessage.success('授权已重置，恢复到未激活状态')
    else ElMessage.error('重置失败')
  } catch { ElMessage.error('重置失败') }
}
async function resetRules(){
  try {
    const r = await fetch('/api/rules/seed', {method:'POST',headers:aHeaders()})
    const d = await r.json()
    ElMessage.success(`已重置 ${d.count} 条规则`)
  } catch { ElMessage.error('重置失败') }
}
</script>

<style scoped>
.settings-page { max-width: 860px; margin: 0 auto; }
.mt20 { margin-top: 20px; }

.page-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 24px;
}
.header-left { display: flex; align-items: center; gap: 14px; }
.header-icon-box {
  width: 44px; height: 44px; border-radius: 12px;
  background: linear-gradient(135deg,#1a1635,#2d2854);
  display: flex; align-items: center; justify-content: center; color: #fff;
}
.header-left h2 { margin: 0; font-size: 22px; font-weight: 700; color: #303133; }
.header-left p { margin: 2px 0 0; font-size: 13px; color: #909399; }

/* 分区卡片 */
.section-card {
  background: #fff; border-radius: 14px; padding: 24px;
  box-shadow: 0 1px 12px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
}
.section-header {
  display: flex; align-items: center; gap: 10px;
  font-size: 16px; font-weight: 600; color: #303133;
  padding-bottom: 16px; margin-bottom: 16px;
  border-bottom: 1px solid #f5f5f5;
}
.section-icon {
  width: 34px; height: 34px; border-radius: 8px;
  display: flex; align-items: center; justify-content: center; color: #fff;
}
.section-icon.sa { background: linear-gradient(135deg,#4472C4,#5b8def); }
.section-icon.sb { background: linear-gradient(135deg,#67c23a,#85ce61); }
.section-icon.sc { background: linear-gradient(135deg,#e6a23c,#ebb563); }
.section-icon.sd { background: linear-gradient(135deg,#909399,#b4b7bd); }
.section-icon.se { background: linear-gradient(135deg,#4472C4,#67c2f3); }

.settings-form :deep(.el-form-item) { margin-bottom: 18px; }
.field-tip { font-size: 12px; color: #c0c4cc; margin-top: 4px; }

/* 数据操作 */
.data-actions { display: flex; flex-direction: column; gap: 0; }
.data-action-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 0; border-bottom: 1px solid #fafafa;
}
.data-action-item:last-child { border-bottom: none; }
.da-title { font-size: 14px; font-weight: 500; color: #303133; }
.da-desc { font-size: 12px; color: #c0c4cc; margin-top: 2px; }

/* 关于 */
.about-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.about-item {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 16px; border-radius: 10px;
  background: linear-gradient(135deg,#f8f9fc,#f0f2f5);
  transition: all 0.2s;
}
.about-item:hover { background: linear-gradient(135deg,#f0f4ff,#e8eeff); }
.about-icon {
  width: 38px; height: 38px; border-radius: 8px;
  background: linear-gradient(135deg,#4472C4,#2d2854);
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 18px;
}
.about-info { display: flex; flex-direction: column; gap: 2px; }
.about-label { font-size: 12px; color: #909399; }
.about-val { font-size: 14px; color: #303133; font-weight: 500; }

.save-bar {
  display: flex; justify-content: center; margin-top: 24px; padding: 20px 0;
}
</style>
