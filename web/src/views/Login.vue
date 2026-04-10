<template>
  <div class="login-shell">
    <div class="login-orb login-orb-blue"></div>
    <div class="login-orb login-orb-green"></div>

    <div class="login-layout">
      <section class="login-brand">
        <span class="login-brand-mark">X</span>
        <div class="login-brand-copy">
          <p class="login-brand-name">Xray Panel</p>
          <h1 class="login-brand-title">{{ t('auth.loginTitle') }}</h1>
        </div>
      </section>

      <n-card class="login-card" :bordered="false">
        <template #header>
          <div class="login-card-title">{{ t('auth.loginTitle') }}</div>
        </template>

        <n-form ref="formRef" :model="form" :rules="rules">
          <n-form-item :label="t('auth.username')" path="username">
            <n-input v-model:value="form.username" :placeholder="t('auth.username')" @keyup.enter="handleLogin" />
          </n-form-item>
          <n-form-item :label="t('auth.password')" path="password">
            <n-input
              v-model:value="form.password"
              type="password"
              :placeholder="t('auth.password')"
              show-password-on="click"
              @keyup.enter="handleLogin"
            />
          </n-form-item>
          <n-button type="primary" block :loading="loading" @click="handleLogin">
            {{ t('auth.loginButton') }}
          </n-button>
        </n-form>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, useMessage, type FormRules } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const authStore = useAuthStore()
const message = useMessage()
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules: FormRules = {
  username: { required: true, message: t('auth.username'), trigger: 'blur' },
  password: { required: true, message: t('auth.password'), trigger: 'blur' }
}

async function handleLogin() {
  if (!form.username || !form.password) return
  loading.value = true
  try {
    await authStore.login(form.username, form.password)
    message.success(t('common.success'))
  } catch (err: any) {
    message.error(err?.error || t('auth.loginError'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-shell {
  position: relative;
  display: grid;
  min-height: 100vh;
  place-items: center;
  overflow: hidden;
  padding: 24px;
  background: var(--panel-page-bg);
}

.login-orb {
  display: none;
}

.login-orb-blue {
  top: -90px;
  left: -80px;
  background: rgba(37, 99, 235, 0.28);
}

.login-orb-green {
  right: -120px;
  bottom: -120px;
  background: rgba(20, 184, 166, 0.24);
}

.login-layout {
  position: relative;
  z-index: 1;
  display: grid;
  width: min(980px, 100%);
  align-items: center;
  gap: 28px;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 420px);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: clamp(18px, 3vw, 34px);
}

.login-brand-mark {
  display: grid;
  width: 58px;
  height: 58px;
  place-items: center;
  border-radius: 20px;
  background: var(--panel-primary);
  color: #fff;
  font-size: 24px;
  font-weight: 700;
}

.login-brand-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.login-brand-name {
  margin: 0;
  color: var(--panel-text-3);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
}

.login-brand-title {
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 1.05;
  letter-spacing: -0.04em;
}

.login-card {
  border: 1px solid var(--panel-border);
  border-radius: 30px;
  background: var(--panel-surface);
  box-shadow: var(--panel-shadow);
}

.login-card-title {
  font-size: 18px;
  font-weight: 700;
}

:deep(.login-card > .n-card-header) {
  padding: 24px 24px 0;
}

:deep(.login-card > .n-card__content) {
  padding: 20px 24px 24px;
}

@media (max-width: 860px) {
  .login-layout {
    grid-template-columns: 1fr;
  }

  .login-brand {
    justify-content: center;
    text-align: center;
  }
}

@media (max-width: 520px) {
  .login-shell {
    padding: 16px;
  }

  .login-layout {
    gap: 18px;
  }
}
</style>
