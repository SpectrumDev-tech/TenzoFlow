<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref} from 'vue'
import {ApiError} from '@/api/client'
import {fetchStatus, type State, type StatusReport} from '@/api/status'

const REFRESH_MS = 5000
// sessionStorage (not localStorage): the token disappears when the tab is closed.
const TOKEN_KEY = 'tenzoflow.statusToken'

type ViewError = { kind: 'unreachable' | 'unauthorized' | 'http'; message: string }

const report = ref<StatusReport | null>(null)
const error = ref<ViewError | null>(null)
const loading = ref(true)
const lastUpdated = ref<Date | null>(null)
const token = ref(sessionStorage.getItem(TOKEN_KEY) ?? '')
const tokenInput = ref('')

let timer: number | undefined
let controller: AbortController | undefined
let stopped = false

function toViewError(e: unknown): ViewError {
  if (e instanceof ApiError) {
    if (e.status === undefined) return { kind: 'unreachable', message: e.message }
    if (e.status === 401) return { kind: 'unauthorized', message: e.message }
    return { kind: 'http', message: e.message }
  }
  return { kind: 'http', message: 'Unexpected error' }
}

async function load(): Promise<void> {
  controller?.abort() // never let two requests race: the older answer could overwrite the newer one
  controller = new AbortController()
  try {
    // The previous report is kept on errors, so the page does not go blank on a short blip.
    report.value = await fetchStatus(token.value || undefined, controller.signal)
    error.value = null
    lastUpdated.value = new Date()
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    error.value = toViewError(e)
  } finally {
    loading.value = false
  }
}

// A chain of setTimeout instead of setInterval: the next refresh is scheduled only after
// the previous one finished, so a slow backend never piles up requests.
function schedule(): void {
  timer = window.setTimeout(async () => {
    if (!document.hidden) await load() // do not poll while the tab is in the background
    if (!stopped) schedule()
  }, REFRESH_MS)
}

onMounted(async () => {
  await load()
  if (!stopped) schedule()
})

onUnmounted(() => {
  stopped = true
  window.clearTimeout(timer)
  controller?.abort()
})

function saveToken(): void {
  token.value = tokenInput.value.trim()
  sessionStorage.setItem(TOKEN_KEY, token.value)
  tokenInput.value = ''
  loading.value = true
  void load()
}

function clearToken(): void {
  token.value = ''
  sessionStorage.removeItem(TOKEN_KEY)
  void load()
}

const stateLabel: Record<State, string> = {
  ok: 'Operational',
  degraded: 'Degraded',
  down: 'Down',
}

const serviceRows = computed(() => {
  const s = report.value?.service
  if (!s) return []
  return [
    ['Service', s.name],
    ['API version', s.api_version],
    ['App version', s.version],
    ['Commit', s.commit],
    ['Built', s.build_time],
    ['Go', s.go_version],
    ['Environment', s.env],
    ['Started', new Date(s.started_at).toLocaleString()],
    ['Uptime', formatUptime(s.uptime_seconds)],
  ] as [string, string][]
})

function formatUptime(total: number): string {
  const d = Math.floor(total / 86400)
  const h = Math.floor((total % 86400) / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1>Service status</h1>
      <span v-if="report" class="badge" :class="report.status">{{ stateLabel[report.status] }}</span>
    </header>

    <p v-if="loading && !report" class="muted">Loading…</p>

    <!-- 1. The backend itself cannot be reached: a different problem from "database is down". -->
    <section v-if="error?.kind === 'unreachable'" class="banner down">
      <strong>Backend is unreachable.</strong>
      Is the API running? In development start it with <code>task run</code>.
      <span v-if="report"> Showing the last known data.</span>
    </section>

    <!-- 2. The backend asks for the status token. -->
    <section v-else-if="error?.kind === 'unauthorized'" class="banner warn">
      <strong>A token is required.</strong>
      <p class="muted">
        The status endpoint is protected. Enter the <code>STATUS_TOKEN</code> value of the backend.
        It is kept only for this browser tab.
      </p>
      <form class="token-form" @submit.prevent="saveToken">
        <input v-model="tokenInput" type="password" autocomplete="off" placeholder="Status token" />
        <button type="submit" :disabled="!tokenInput.trim()">Save</button>
      </form>
    </section>

    <section v-else-if="error" class="banner down">
      <strong>Could not load the status.</strong> {{ error.message }}
    </section>

    <template v-if="report">
      <p class="muted meta">
        Updated {{ lastUpdated?.toLocaleTimeString() }} · refreshes every {{ REFRESH_MS / 1000 }} s
        <button v-if="token" class="link" @click="clearToken">Forget token</button>
      </p>

      <div class="grid">
        <article class="card">
          <h2>API</h2>
          <dl>
            <template v-for="[label, value] in serviceRows" :key="label">
              <dt>{{ label }}</dt>
              <dd>{{ value }}</dd>
            </template>
          </dl>
        </article>

        <article v-for="c in report.components" :key="c.name" class="card">
          <h2>
            {{ c.name }}
            <span class="badge small" :class="c.status">{{ stateLabel[c.status] }}</span>
          </h2>
          <dl>
            <dt>Response time</dt>
            <dd>{{ c.latency_ms.toFixed(2) }} ms</dd>
            <template v-for="[key, value] in Object.entries(c.details ?? {})" :key="key">
              <dt>{{ key.replaceAll('_', ' ') }}</dt>
              <dd>{{ value }}</dd>
            </template>
          </dl>
          <p v-if="c.error" class="error-text">{{ c.error }}</p>
        </article>
      </div>
    </template>
  </main>
</template>

<style scoped>
.page {
  max-width: 960px;
  margin: 0 auto;
  padding: 2rem 1rem 4rem;
}

.header {
  display: flex;
  align-items: center;
  gap: 1rem;
}

h1 {
  margin: 0;
  font-size: 1.6rem;
}

h2 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0 0 0.75rem;
  font-size: 1.05rem;
  text-transform: capitalize;
}

.muted {
  color: var(--muted);
}

.meta {
  margin: 1rem 0;
  font-size: 0.9rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
}

.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
}

dl {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.35rem 1rem;
  margin: 0;
}

dt {
  color: var(--muted);
  text-transform: capitalize;
}

dd {
  margin: 0;
  overflow-wrap: anywhere;
  font-variant-numeric: tabular-nums;
}

.badge {
  padding: 0.2rem 0.7rem;
  border-radius: 999px;
  font-size: 0.85rem;
  font-weight: 600;
}

.badge.small {
  font-size: 0.75rem;
  text-transform: none;
}

.badge.ok {
  color: var(--ok);
  background: var(--ok-bg);
}

.badge.degraded {
  color: var(--warn);
  background: var(--warn-bg);
}

.badge.down {
  color: var(--down);
  background: var(--down-bg);
}

.banner {
  margin: 1rem 0;
  padding: 1rem 1.25rem;
  border-radius: 12px;
}

.banner.down {
  color: var(--down);
  background: var(--down-bg);
}

.banner.warn {
  color: var(--warn);
  background: var(--warn-bg);
}

.banner p {
  margin: 0.5rem 0;
}

.error-text {
  margin: 0.75rem 0 0;
  color: var(--down);
}

.token-form {
  display: flex;
  gap: 0.5rem;
}

.token-form input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--text);
}

button {
  padding: 0.5rem 1rem;
  border: 0;
  border-radius: 8px;
  background: var(--text);
  color: var(--bg);
  cursor: pointer;
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

button.link {
  margin-left: 0.5rem;
  padding: 0;
  background: none;
  color: var(--muted);
  text-decoration: underline;
}
</style>
