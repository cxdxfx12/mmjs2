<template>
  <el-container class="layout">
    <!-- 侧边栏 -->
    <el-aside width="230px" class="sidebar">
      <div class="logo" @click="$router.push('/home')">
        <div class="logo-icon">
          <img src="/monkey-icon.png" alt="喵喵云结算" class="logo-img" />
        </div>
        <div class="logo-text">
          <span class="logo-name">喵喵云结算</span>
          <span class="logo-ver">v1.0</span>
        </div>
      </div>
      <el-menu :default-active="activeMenu" router background-color="transparent" text-color="rgba(255,255,255,0.7)" active-text-color="#fff" class="side-menu">
        <el-menu-item index="/home">
          <el-icon><HomeFilled /></el-icon>
          <span>首页</span>
        </el-menu-item>
        <el-menu-item index="/calc">
          <el-icon><Coin /></el-icon>
          <span>计费结算</span>
          <el-badge v-if="store.calculating" is-dot class="calc-badge" />
        </el-menu-item>
        <el-menu-item index="/rules">
          <el-icon><Setting /></el-icon>
          <span>规则管理</span>
        </el-menu-item>
        <el-menu-item index="/test-rule">
          <el-icon><Aim /></el-icon>
          <span>规则测试</span>
        </el-menu-item>
        <el-menu-item index="/history">
          <el-icon><Clock /></el-icon>
          <span>历史记录</span>
        </el-menu-item>
        <el-menu-item index="/license">
          <el-icon><Key /></el-icon>
          <span>授权管理</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Tools /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-bottom">
        <div class="license-status" @click="$router.push('/license')">
          <span class="status-dot" :class="dotClass" />
          <span class="status-text">{{ dotText }}</span>
        </div>
        <div class="user-info">
          <el-icon><UserFilled /></el-icon>
          <span>{{ userDisplay }}</span>
          <el-button link class="logout-btn" @click="handleLogout" title="退出登录">
            <el-icon><SwitchButton /></el-icon>
          </el-button>
        </div>
      </div>
    </el-aside>

    <!-- 主内容区 -->
    <el-container class="main-area">
      <!-- 全局计算中横幅 -->
      <div v-if="store.calculating" class="calc-banner" @click="$router.push('/calc')">
        <el-icon class="calc-banner-icon spinning"><Loading /></el-icon>
        <span>后台计算任务进行中，点击查看进度</span>
      </div>
      <el-header class="header">
        <div class="header-left">
          <el-icon class="header-icon" :size="20"><component :is="headerIcon" /></el-icon>
          <span>{{ $route.meta.title || '喵喵云结算' }}</span>
        </div>
        <div class="header-right">
          <el-tag v-if="store.license" :type="licenseTagType" size="small" effect="dark" round>{{ licenseText }}</el-tag>
          <span v-if="store.license?.customer_name" class="customer-name">{{ store.license.customer_name }}</span>
        </div>
      </el-header>
      <el-main class="main-content"><router-view v-slot="{ Component }"><keep-alive><component :is="Component" /></keep-alive></router-view></el-main>
      <div class="company-footer">
        <div class="footer-content">
          <span class="footer-company">杭州喵喵至家网络有限公司</span>
          <span class="footer-divider">|</span>
          <span class="footer-service">客服：<a href="tel:17771300068">17771300068</a></span>
        </div>
      </div>
    </el-container>
    <KnowledgeAI />
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { HomeFilled, Coin, Setting, Clock, Key, Tools, UserFilled, SwitchButton, Loading, Aim } from '@element-plus/icons-vue'
import KnowledgeAI from '@/components/KnowledgeAI.vue'

const route = useRoute()
const router = useRouter()
const store = useAppStore()
const activeMenu = computed(() => route.path)
const userDisplay = computed(() => localStorage.getItem('yunfei_user') || 'admin')
const headerIcon = computed(() => {
  const m: Record<string, any> = { '首页': HomeFilled, '计费结算': Coin, '规则管理': Setting, '规则测试': Aim, '历史记录': Clock, '授权管理': Key, '系统设置': Tools }
  return m[route.meta.title as string] || HomeFilled
})
const licenseTagType = computed(()=>{
  switch(store.licenseStatus){
    case 'active': return 'success'
    case 'expiring': return 'warning'
    case 'expired': return 'danger'
    default: return 'info'
  }
})
const licenseText = computed(()=>{
  if(!store.license) return '未授权'
  if(!store.license.is_valid) return '已过期'
  return '剩余 '+store.daysLeft+' 天'
})
const dotClass = computed(()=>{
  if(!store.license||!store.license.is_valid) return 'off'
  if(store.daysLeft <= 7) return 'warn'
  return 'on'
})
const dotText = computed(()=>{
  if(!store.license) return '未激活'
  if(!store.license.is_valid) return '已过期'
  return '有效期至 '+store.license.expires_at
})

function handleLogout() {
  localStorage.removeItem('yunfei_token')
  localStorage.removeItem('yunfei_user')
  router.replace('/login')
}

onMounted(()=>{
  store.fetchLicense()
  store.fetchMachineCode()
  store.fetchRules()
})
</script>

<style scoped>
.layout { height: 100vh; }

/* 侧边栏 */
.sidebar {
  background: linear-gradient(180deg, #1a1635 0%, #252050 50%, #1a1635 100%);
  overflow-y: auto; display: flex; flex-direction: column;
  box-shadow: 2px 0 24px rgba(0,0,0,0.15);
}
.logo {
  display: flex; align-items: center; gap: 10px;
  padding: 22px 20px; cursor: pointer;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.logo-icon {
  width: 40px; height: 40px; border-radius: 10px;
  background: linear-gradient(135deg, #4472C4, #2d2854);
  display: flex; align-items: center; justify-content: center;
  color: #fff; overflow: hidden;
}
.logo-img { width: 100%; height: 100%; object-fit: cover; border-radius: 10px; }
.logo-text { display: flex; flex-direction: column; }
.logo-name { font-size: 20px; font-weight: 700; color: #fff; letter-spacing: 1px; }
.logo-ver { font-size: 11px; color: rgba(255,255,255,0.4); }

.side-menu { border-right: none !important; margin-top: 8px; }
.side-menu :deep(.el-menu-item) {
  margin: 2px 12px; border-radius: 8px; height: 44px;
  transition: all 0.25s;
}
.side-menu :deep(.el-menu-item:hover) { background: rgba(255,255,255,0.06) !important; }
.side-menu :deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, rgba(68,114,196,0.3), rgba(45,40,84,0.2)) !important;
  border-left: 3px solid #4472C4;
}
.calc-badge { margin-left: 6px; }
.calc-badge :deep(.el-badge__content) { background: #e6a23c; }

.sidebar-bottom {
  margin-top: auto; padding: 12px 16px;
  border-top: 1px solid rgba(255,255,255,0.06);
}
.license-status { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; cursor: pointer; padding: 6px 8px; border-radius: 6px; transition: background 0.2s; }
.license-status:hover { background: rgba(255,255,255,0.04); }
.status-dot { width: 7px; height: 7px; border-radius: 50%; }
.status-dot.on { background: #67c23a; box-shadow: 0 0 6px rgba(103,194,58,0.5); }
.status-dot.warn { background: #e6a23c; animation: blink 1.2s infinite; }
.status-dot.off { background: #f56c6c; }
@keyframes blink { 50% { opacity: 0.3; } }
.status-text { font-size: 11px; color: rgba(255,255,255,0.5); }
.user-info {
  display: flex; align-items: center; gap: 6px;
  padding: 8px; border-radius: 8px;
  background: rgba(255,255,255,0.04);
  color: rgba(255,255,255,0.7); font-size: 13px;
}
.logout-btn {
  margin-left: auto; color: rgba(255,255,255,0.4) !important;
  padding: 4px !important;
}
.logout-btn:hover { color: #f56c6c !important; }

/* 主区域 */
.main-area { background: #f0f2f5; }
/* 全局计算中横幅 */
.calc-banner {
  display: flex; align-items: center; gap: 10px; justify-content: center;
  padding: 10px 0; background: linear-gradient(90deg, #e6a23c, #f5d44a);
  color: #fff; font-size: 14px; font-weight: 600;
  cursor: pointer; transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(230,162,60,0.3);
}
.calc-banner:hover { background: linear-gradient(90deg, #d8942a, #e8c742); }
.calc-banner-icon { font-size: 18px; }
.spinning { animation: spin 1s linear infinite; }
@keyframes spin { 100% { transform: rotate(360deg); } }
.header {
  display: flex; align-items: center; justify-content: space-between;
  background: #fff; border-bottom: 1px solid #e4e7ed;
  padding: 0 28px; height: 56px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.header-left { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; color: #303133; }
.header-icon { color: #4472C4; }
.header-right { display: flex; align-items: center; gap: 12px; }
.customer-name { color: #909399; font-size: 13px; }
.el-main { padding: 24px; background: #f0f2f5; min-height: 100%; }
.main-content { background: #f0f2f5; }

/* 底部公司信息 */
.company-footer {
  text-align: center; padding: 10px 0; margin-top: auto;
  background: #fff; border-top: 1px solid #e4e7ed;
}
.footer-content {
  display: flex; align-items: center; justify-content: center; gap: 12px;
  font-size: 12px; color: #909399;
}
.footer-divider { color: #dcdfe6; }
.footer-service a { color: #4472C4; text-decoration: none; }
.footer-service a:hover { color: #2d5aa7; }
</style>
