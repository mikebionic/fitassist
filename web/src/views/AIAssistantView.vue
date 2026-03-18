<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import { aiApi, type AISession, type AIMessage } from '../api/health'

const auth = useAuthStore()

const sessions = ref<AISession[]>([])
const currentSession = ref<AISession | null>(null)
const messages = ref<AIMessage[]>([])
const newMessage = ref('')
const loading = ref(false)
const streaming = ref(false)
const streamingText = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const summaryText = ref('')
const summaryLoading = ref(false)

let ws: WebSocket | null = null

const sortedSessions = computed(() =>
  [...sessions.value].sort((a, b) =>
    new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
  )
)

onMounted(async () => {
  await loadSessions()
  connectWS()
})

onUnmounted(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})

async function loadSessions() {
  try {
    const { data } = await aiApi.listSessions()
    sessions.value = data
  } catch { /* empty */ }
}

async function createSession() {
  try {
    const { data } = await aiApi.createSession('New Chat')
    sessions.value.unshift(data)
    await selectSession(data)
  } catch { /* empty */ }
}

async function selectSession(session: AISession) {
  currentSession.value = session
  try {
    const { data } = await aiApi.getSession(session.id)
    messages.value = data.messages
    await scrollToBottom()
  } catch { /* empty */ }
}

async function deleteSession(session: AISession) {
  try {
    await aiApi.deleteSession(session.id)
    sessions.value = sessions.value.filter(s => s.id !== session.id)
    if (currentSession.value?.id === session.id) {
      currentSession.value = null
      messages.value = []
    }
  } catch { /* empty */ }
}

function connectWS() {
  if (ws) ws.close()

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  ws = new WebSocket(`${protocol}//${host}/ws/ai/chat?token=${auth.accessToken}`)

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data)
    switch (msg.type) {
      case 'chunk':
        streamingText.value += msg.content
        scrollToBottom()
        break
      case 'done':
        if (streamingText.value) {
          messages.value.push({
            id: msg.message_id,
            session_id: msg.session_id,
            role: 'assistant',
            content: streamingText.value,
            tokens_used: msg.tokens,
            created_at: new Date().toISOString(),
          })
          streamingText.value = ''
        }
        streaming.value = false
        loading.value = false
        scrollToBottom()
        break
      case 'error':
        streamingText.value = ''
        streaming.value = false
        loading.value = false
        break
    }
  }

  ws.onclose = () => {
    // Reconnect after 3 seconds
    setTimeout(() => {
      if (!ws || ws.readyState === WebSocket.CLOSED) {
        connectWS()
      }
    }, 3000)
  }
}

async function sendMessage() {
  if (!newMessage.value.trim() || !currentSession.value || loading.value) return

  const text = newMessage.value.trim()
  newMessage.value = ''

  // Add user message to UI
  messages.value.push({
    id: 0,
    session_id: currentSession.value.id,
    role: 'user',
    content: text,
    created_at: new Date().toISOString(),
  })
  await scrollToBottom()

  loading.value = true
  streaming.value = true
  streamingText.value = ''

  // Try WebSocket first, fall back to REST
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'message',
      session_id: currentSession.value.id,
      content: text,
    }))
  } else {
    // Fallback to REST
    try {
      const { data } = await aiApi.sendMessage(currentSession.value.id, text)
      messages.value.push(data)
      streaming.value = false
      loading.value = false
      await scrollToBottom()
    } catch {
      streaming.value = false
      loading.value = false
    }
  }
}

async function getSummary() {
  summaryLoading.value = true
  try {
    const { data } = await aiApi.summary()
    summaryText.value = data.summary
  } catch {
    summaryText.value = 'Failed to generate summary.'
  }
  summaryLoading.value = false
}

async function scrollToBottom() {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

function formatTime(dateStr: string) {
  const d = new Date(dateStr)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}
</script>

<template>
  <div class="ai-container">
    <!-- Sidebar: Sessions list -->
    <div class="ai-sidebar">
      <div class="ai-sidebar-header">
        <h3>AI Assistant</h3>
        <button class="btn-new" @click="createSession" title="New Chat">+</button>
      </div>

      <button class="btn-summary" @click="getSummary" :disabled="summaryLoading">
        {{ summaryLoading ? 'Generating...' : 'Quick Summary' }}
      </button>

      <div class="session-list">
        <div
          v-for="session in sortedSessions"
          :key="session.id"
          class="session-item"
          :class="{ active: currentSession?.id === session.id }"
          @click="selectSession(session)"
        >
          <span class="session-title">{{ session.title || 'New Chat' }}</span>
          <button class="btn-delete" @click.stop="deleteSession(session)" title="Delete">&times;</button>
        </div>
        <p v-if="sessions.length === 0" class="empty-text">No conversations yet</p>
      </div>
    </div>

    <!-- Main chat area -->
    <div class="ai-main">
      <!-- Summary popup -->
      <div v-if="summaryText" class="summary-card">
        <div class="summary-header">
          <strong>Health Summary</strong>
          <button @click="summaryText = ''" class="btn-close">&times;</button>
        </div>
        <div class="summary-body" v-html="renderMarkdown(summaryText)"></div>
      </div>

      <template v-if="currentSession">
        <!-- Messages -->
        <div class="messages" ref="messagesContainer">
          <div v-if="messages.length === 0 && !streaming" class="empty-chat">
            <p>Start a conversation about your health data.</p>
            <p class="hint">Try: "How was my sleep this week?" or "Give me a workout plan"</p>
          </div>

          <div
            v-for="msg in messages"
            :key="msg.id"
            class="message"
            :class="msg.role"
          >
            <div class="message-avatar">
              {{ msg.role === 'user' ? 'You' : 'AI' }}
            </div>
            <div class="message-content">
              <div v-html="renderMarkdown(msg.content)"></div>
              <span class="message-time">{{ formatTime(msg.created_at) }}</span>
            </div>
          </div>

          <!-- Streaming message -->
          <div v-if="streaming && streamingText" class="message assistant">
            <div class="message-avatar">AI</div>
            <div class="message-content">
              <div v-html="renderMarkdown(streamingText)"></div>
              <span class="typing-indicator">...</span>
            </div>
          </div>

          <div v-if="loading && !streamingText" class="message assistant">
            <div class="message-avatar">AI</div>
            <div class="message-content">
              <span class="typing-indicator">Thinking...</span>
            </div>
          </div>
        </div>

        <!-- Input -->
        <div class="input-area">
          <textarea
            v-model="newMessage"
            @keydown="handleKeydown"
            placeholder="Ask about your health data..."
            :disabled="loading"
            rows="1"
          ></textarea>
          <button class="btn-send" @click="sendMessage" :disabled="loading || !newMessage.trim()">
            Send
          </button>
        </div>
      </template>

      <div v-else class="no-session">
        <h2>FitAssist AI</h2>
        <p>Your personal health assistant powered by AI.</p>
        <p>Select a conversation or start a new one.</p>
        <button class="btn-start" @click="createSession">Start New Chat</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// Simple markdown-like rendering (bold, italic, lists, line breaks)
function renderMarkdown(text: string): string {
  if (!text) return ''
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/`(.+?)`/g, '<code>$1</code>')
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/^# (.+)$/gm, '<h2>$1</h2>')
    .replace(/^- (.+)$/gm, '<li>$1</li>')
    .replace(/(<li>.*<\/li>)/gs, '<ul>$1</ul>')
    .replace(/<\/ul>\s*<ul>/g, '')
    .replace(/\n\n/g, '<br><br>')
    .replace(/\n/g, '<br>')
}
</script>

<style scoped>
.ai-container {
  display: flex;
  height: calc(100vh - 4rem);
  margin: -2rem;
  overflow: hidden;
}

.ai-sidebar {
  width: 260px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
}

.ai-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-bottom: 1px solid var(--border);
}

.ai-sidebar-header h3 {
  margin: 0;
  font-size: 1rem;
}

.btn-new {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 1.2rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.btn-new:hover { background: var(--accent); color: #fff; }

.btn-summary {
  margin: 0.75rem;
  padding: 0.5rem;
  border-radius: 8px;
  border: 1px solid var(--accent);
  background: transparent;
  color: var(--accent);
  cursor: pointer;
  font-size: 0.85rem;
  font-family: inherit;
}
.btn-summary:hover { background: var(--accent); color: #fff; }
.btn-summary:disabled { opacity: 0.5; cursor: not-allowed; }

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.85rem;
  color: var(--text-secondary);
  transition: background 0.15s;
}
.session-item:hover { background: var(--border); color: var(--text-primary); }
.session-item.active { background: var(--accent); color: #fff; }

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.btn-delete {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 1.1rem;
  opacity: 0;
  padding: 0 0.25rem;
}
.session-item:hover .btn-delete { opacity: 0.6; }
.btn-delete:hover { opacity: 1 !important; }

.empty-text {
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.85rem;
  padding: 1rem;
}

/* Main chat */
.ai-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
}

.summary-card {
  position: absolute;
  top: 1rem;
  left: 50%;
  transform: translateX(-50%);
  width: min(90%, 600px);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1rem;
  z-index: 10;
  box-shadow: 0 4px 20px rgba(0,0,0,0.15);
}

.summary-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  color: var(--text-secondary);
}

.summary-body {
  font-size: 0.9rem;
  line-height: 1.6;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.empty-chat {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary);
  text-align: center;
}
.hint { font-size: 0.85rem; opacity: 0.7; margin-top: 0.5rem; }

.message {
  display: flex;
  gap: 0.75rem;
  max-width: 80%;
}
.message.user { align-self: flex-end; flex-direction: row-reverse; }
.message.assistant { align-self: flex-start; }

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: 700;
  flex-shrink: 0;
}
.message.user .message-avatar {
  background: var(--accent);
  color: #fff;
}
.message.assistant .message-avatar {
  background: var(--border);
  color: var(--text-primary);
}

.message-content {
  padding: 0.75rem 1rem;
  border-radius: 12px;
  font-size: 0.9rem;
  line-height: 1.6;
}
.message.user .message-content {
  background: var(--accent);
  color: #fff;
  border-bottom-right-radius: 4px;
}
.message.assistant .message-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-bottom-left-radius: 4px;
}

.message-time {
  display: block;
  font-size: 0.7rem;
  opacity: 0.6;
  margin-top: 0.25rem;
}

.typing-indicator {
  display: inline-block;
  animation: blink 1s infinite;
}
@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0.3; }
}

.message-content :deep(code) {
  background: rgba(0,0,0,0.1);
  padding: 0.1em 0.3em;
  border-radius: 3px;
  font-size: 0.85em;
}

.message-content :deep(h2),
.message-content :deep(h3),
.message-content :deep(h4) {
  margin: 0.5em 0 0.25em;
}

.message-content :deep(ul) {
  margin: 0.25em 0;
  padding-left: 1.25em;
}

/* Input */
.input-area {
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border);
  background: var(--bg-primary);
}

.input-area textarea {
  flex: 1;
  resize: none;
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 0.75rem 1rem;
  font-size: 0.9rem;
  font-family: inherit;
  background: var(--bg-secondary);
  color: var(--text-primary);
  max-height: 120px;
  line-height: 1.4;
}
.input-area textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.btn-send {
  padding: 0.75rem 1.5rem;
  border-radius: 12px;
  border: none;
  background: var(--accent);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.9rem;
}
.btn-send:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-send:not(:disabled):hover { filter: brightness(1.1); }

/* No session selected */
.no-session {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  color: var(--text-secondary);
}
.no-session h2 {
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}
.no-session p { margin: 0.25rem 0; }

.btn-start {
  margin-top: 1.5rem;
  padding: 0.75rem 2rem;
  border-radius: 12px;
  border: none;
  background: var(--accent);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  font-size: 1rem;
}
.btn-start:hover { filter: brightness(1.1); }
</style>
