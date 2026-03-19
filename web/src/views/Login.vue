<template>
  <div style="display: flex; justify-content: center; align-items: center; height: 100vh; background: var(--n-body-color)">
    <n-card style="width: 400px" :title="t('auth.loginTitle')">
      <n-form ref="formRef" :model="form" :rules="rules">
        <n-form-item :label="t('auth.username')" path="username">
          <n-input v-model:value="form.username" :placeholder="t('auth.username')" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-form-item :label="t('auth.password')" path="password">
          <n-input v-model:value="form.password" type="password" :placeholder="t('auth.password')" show-password-on="click" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-button type="primary" block :loading="loading" @click="handleLogin">
          {{ t('auth.loginButton') }}
        </n-button>
      </n-form>
    </n-card>
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
