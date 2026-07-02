<template>
  <div class="login-page">
    <div class="login-bg">
      <div class="circles">
        <div v-for="i in 6" :key="i" class="circle" :style="circleStyle(i)" />
      </div>
    </div>
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <img src="/monkey-icon.png" alt="喵喵云结算" class="login-logo-img" />
        </div>
        <h1>喵喵云结算</h1>
        <p>快递运费结算系统</p>
      </div>
      <el-form ref="formRef" :model="form" :rules="rules" @keyup.enter="handleLogin">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" :prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" :loading="loading" @click="handleLogin" class="login-btn" round>
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">
        <span>默认账号 admin / admin123</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = reactive({ username: 'admin', password: 'admin123' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

function circleStyle(i: number) {
  const size = 80 + Math.random() * 200
  const x = Math.random() * 100
  const y = Math.random() * 100
  const delay = Math.random() * 5
  return {
    width: `${size}px`, height: `${size}px`,
    left: `${x}%`, top: `${y}%`,
    animationDelay: `${delay}s`,
  }
}

async function handleLogin() {
  loading.value = true
  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    })
    const data = await res.json()
    if (data.ok) {
      localStorage.setItem('yunfei_token', data.token)
      localStorage.setItem('yunfei_user', data.username)
      const redirect = (route.query.redirect as string) || '/home'
      router.replace(redirect)
      ElMessage.success('登录成功')
    } else {
      ElMessage.error(data.error || '登录失败')
    }
  } catch {
    ElMessage.error('网络错误，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1635 0%, #2d2854 50%, #1e3a5f 100%);
  overflow: hidden;
  position: relative;
}
.login-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.circles {
  position: absolute;
  inset: -50%;
}
.circle {
  position: absolute;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255,255,255,0.06) 0%, transparent 70%);
  animation: float 8s ease-in-out infinite alternate;
}
@keyframes float {
  0% { transform: translate(0, 0) scale(1); }
  100% { transform: translate(30px, -30px) scale(1.1); }
}
.login-card {
  position: relative;
  width: 420px;
  padding: 48px 40px 36px;
  background: rgba(255,255,255,0.95);
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
  backdrop-filter: blur(10px);
  animation: slideUp 0.6s ease;
}
@keyframes slideUp {
  from { opacity: 0; transform: translateY(30px); }
  to { opacity: 1; transform: translateY(0); }
}
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-logo {
  width: 72px; height: 72px;
  border-radius: 18px;
  background: linear-gradient(135deg, #4472C4, #2d2854);
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 16px;
  box-shadow: 0 8px 24px rgba(68,114,196,0.4);
  overflow: hidden;
}
.login-logo-img { width: 100%; height: 100%; object-fit: cover; border-radius: 18px; }
.login-header h1 {
  font-size: 28px; font-weight: 700; color: #1a1635; margin: 0;
}
.login-header p {
  font-size: 14px; color: #909399; margin: 4px 0 0;
}
.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  letter-spacing: 4px;
  background: linear-gradient(135deg, #4472C4, #2d2854);
  border: none;
}
.login-btn:hover {
  opacity: 0.92;
}
.login-footer {
  text-align: center;
  color: #c0c4cc;
  font-size: 12px;
  margin-top: 8px;
}
</style>
