<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">Admin Login</h1>
      <form class="login-form" @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username" class="form-label">Username</label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="form-input"
            placeholder="Enter username"
            required
          />
        </div>
        <div class="form-group">
          <label for="password" class="form-label">Password</label>
          <div class="password-wrapper">
            <input
              id="password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              class="form-input"
              placeholder="Enter password"
              required
            />
            <button type="button" class="password-toggle" @click="showPassword = !showPassword">
              {{ showPassword ? 'Hide' : 'Show' }}
            </button>
          </div>
        </div>
        <p v-if="error" class="form-error">{{ error }}</p>
        <button type="submit" class="login-button" :disabled="loading">
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })

const { apiFetch } = useApi()
const router = useRouter()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await apiFetch('/api/admin/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    await router.push('/admin')
  } catch (e: any) {
    error.value = e?.data?.error || 'Login failed'
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
  background: var(--color-bg-secondary);
}

.login-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.login-title {
  font-size: 1.25rem;
  font-weight: 700;
  margin-bottom: 24px;
  text-align: center;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.9rem;
  outline: none;
  transition: border-color 0.15s;
}

.form-input:focus {
  border-color: #3b82f6;
}

.password-wrapper {
  position: relative;
  display: flex;
}

.password-wrapper .form-input {
  flex: 1;
  padding-right: 60px;
}

.password-toggle {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--color-text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  padding: 4px 6px;
}

.password-toggle:hover {
  color: var(--color-text);
}

.form-error {
  color: var(--color-red);
  font-size: 0.85rem;
}

.login-button {
  padding: 10px 16px;
  background: #1f2937;
  color: #ffffff;
  border: none;
  border-radius: 6px;
  font-size: 0.9rem;
  font-weight: 500;
  transition: background 0.15s;
}

.login-button:hover:not(:disabled) {
  background: #374151;
}

.login-button:disabled {
  opacity: 0.6;
}
</style>
