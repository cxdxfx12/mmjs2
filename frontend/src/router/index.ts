import { createRouter, createWebHashHistory } from 'vue-router'
import { ElMessage } from 'element-plus'

async function verifyToken(token: string): Promise<boolean> {
  try {
    const res = await fetch(`/api/auth/verify?token=${encodeURIComponent(token)}`)
    const data = await res.json()
    return data.ok === true
  } catch {
    return false
  }
}

async function checkLicense(): Promise<boolean> {
  try {
    const token = localStorage.getItem('yunfei_token')
    if (!token) return false
    const res = await fetch('/api/license/info', {
      headers: { Authorization: `Bearer ${token}` },
    })
    const data = await res.json()
    return data.is_valid === true
  } catch {
    return false
  }
}

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { title: '登录', noAuth: true },
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      redirect: '/home',
      children: [
        { path: 'home', name: 'Home', component: () => import('@/views/Home.vue'), meta: { title: '首页' } },
        { path: 'calc', name: 'Calc', component: () => import('@/views/Calc.vue'), meta: { title: '计费结算', requireLicense: true } },
        { path: 'rules', name: 'Rules', component: () => import('@/views/Rules.vue'), meta: { title: '规则管理' } },
        { path: 'test-rule', name: 'TestRule', component: () => import('@/views/TestRule.vue'), meta: { title: '规则测试' } },
        { path: 'history', name: 'History', component: () => import('@/views/History.vue'), meta: { title: '历史记录' } },
        { path: 'license', name: 'License', component: () => import('@/views/License.vue'), meta: { title: '授权管理' } },
        { path: 'settings', name: 'Settings', component: () => import('@/views/Settings.vue'), meta: { title: '系统设置' } },
      ],
    },
  ],
})

// 标记是否已检查过首次登录授权跳转
let firstAuthChecked = false

router.beforeEach(async (to, _from, next) => {
  // 不需要认证的页面直接放行
  if (to.meta.noAuth) {
    next()
    return
  }
  const token = localStorage.getItem('yunfei_token')
  if (!token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
    return
  }
  const ok = await verifyToken(token)
  if (!ok) {
    localStorage.removeItem('yunfei_token')
    next({ name: 'Login', query: { redirect: to.fullPath } })
    return
  }

  // 首次登录后如果未授权，跳转到授权页面
  if (!firstAuthChecked && to.name !== 'License') {
    firstAuthChecked = true
    const licensed = await checkLicense()
    if (!licensed) {
      next({ name: 'License' })
      return
    }
  }

  // 需要授权的功能页面（如计费结算）
  if (to.meta.requireLicense) {
    const licensed = await checkLicense()
    if (!licensed) {
      ElMessage.warning('请联系管理员完成授权后再使用计费功能')
      next({ name: 'License' })
      return
    }
  }

  next()
})

export default router

