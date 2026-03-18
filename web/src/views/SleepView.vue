<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { healthApi, type SleepData } from '../api/health'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, LegendComponent])

const sleepData = ref<SleepData[]>([])
const days = ref(30)
const loading = ref(true)

function daysAgo(n: number) {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

async function loadData() {
  loading.value = true
  try {
    const { data } = await healthApi.sleep(daysAgo(days.value), daysAgo(0))
    sleepData.value = data || []
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

const avgSleep = computed(() => {
  if (!sleepData.value.length) return null
  const total = sleepData.value.reduce((s, d) => s + (d.duration_min || 0), 0)
  return Math.round(total / sleepData.value.length)
})

const avgDeep = computed(() => {
  if (!sleepData.value.length) return null
  const total = sleepData.value.reduce((s, d) => s + (d.deep_min || 0), 0)
  return Math.round(total / sleepData.value.length)
})

const chartOption = computed(() => {
  const isDark = document.documentElement.classList.contains('p-dark')
  const textColor = isDark ? '#94a3b8' : '#64748b'

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const date = params[0].name
        let total = 0
        let html = `<b>${date}</b><br/>`
        for (const p of params) {
          html += `${p.marker} ${p.seriesName}: ${(p.value / 60).toFixed(1)}h<br/>`
          total += p.value
        }
        html += `<b>Total: ${(total / 60).toFixed(1)}h</b>`
        return html
      },
    },
    legend: { data: ['Deep', 'Light', 'REM', 'Awake'], textStyle: { color: textColor } },
    grid: { left: 50, right: 20, top: 40, bottom: 30 },
    xAxis: {
      type: 'category',
      data: sleepData.value.map(s => s.date.slice(5)),
      axisLabel: { color: textColor, rotate: sleepData.value.length > 14 ? 45 : 0 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: textColor, formatter: (v: number) => (v / 60).toFixed(0) + 'h' },
    },
    series: [
      { name: 'Deep', type: 'bar', stack: 'sleep', data: sleepData.value.map(s => s.deep_min || 0), itemStyle: { color: '#1e40af' } },
      { name: 'Light', type: 'bar', stack: 'sleep', data: sleepData.value.map(s => s.light_min || 0), itemStyle: { color: '#60a5fa' } },
      { name: 'REM', type: 'bar', stack: 'sleep', data: sleepData.value.map(s => s.rem_min || 0), itemStyle: { color: '#a78bfa' } },
      { name: 'Awake', type: 'bar', stack: 'sleep', data: sleepData.value.map(s => s.awake_min || 0), itemStyle: { color: '#f97316', borderRadius: [4, 4, 0, 0] } },
    ],
  }
})

function fmtMin(m: number | null) {
  if (m == null) return '—'
  return `${Math.floor(m / 60)}h ${m % 60}m`
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Sleep</h1>
      <div style="display: flex; gap: 0.5rem">
        <button v-for="d in [7, 14, 30]" :key="d" class="period-btn" :class="{ active: days === d }" @click="days = d; loadData()">
          {{ d }}d
        </button>
      </div>
    </div>

    <div class="grid-3" style="margin-bottom: 1.5rem">
      <div class="stat-card">
        <div class="label">Avg Duration</div>
        <div class="value" style="font-size: 1.75rem">{{ fmtMin(avgSleep) }}</div>
      </div>
      <div class="stat-card">
        <div class="label">Avg Deep Sleep</div>
        <div class="value" style="font-size: 1.75rem">{{ fmtMin(avgDeep) }}</div>
      </div>
      <div class="stat-card">
        <div class="label">Records</div>
        <div class="value">{{ sleepData.length }}</div>
        <div class="sub">nights tracked</div>
      </div>
    </div>

    <div class="chart-container">
      <h3>Sleep Stages</h3>
      <v-chart v-if="sleepData.length > 0" :option="chartOption" style="height: 350px" autoresize />
      <div v-else class="empty-state">
        <i class="pi pi-moon"></i>
        <p>No sleep data available</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.period-btn {
  padding: 0.375rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 0.8rem;
  font-family: inherit;
  font-weight: 600;
}
.period-btn.active {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
</style>
