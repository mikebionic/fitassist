<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { healthApi, type HeartRateData } from '../api/health'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, DataZoomComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, DataZoomComponent])

const hrData = ref<HeartRateData[]>([])
const days = ref(1)
const loading = ref(true)

function daysAgo(n: number) {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

async function loadData() {
  loading.value = true
  try {
    const { data } = await healthApi.heartrate(daysAgo(days.value), daysAgo(0))
    hrData.value = data || []
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

const stats = computed(() => {
  if (!hrData.value.length) return { avg: null, min: null, max: null }
  const bpms = hrData.value.map(h => h.bpm)
  return {
    avg: Math.round(bpms.reduce((a, b) => a + b, 0) / bpms.length),
    min: Math.min(...bpms),
    max: Math.max(...bpms),
  }
})

const chartOption = computed(() => {
  const isDark = document.documentElement.classList.contains('p-dark')
  const textColor = isDark ? '#94a3b8' : '#64748b'

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (p: any) => {
        const t = new Date(p[0].name).toLocaleTimeString()
        return `${t}<br/>HR: <b>${p[0].value} bpm</b>`
      },
    },
    grid: { left: 50, right: 20, top: 20, bottom: 60 },
    dataZoom: [{ type: 'slider', bottom: 10 }],
    xAxis: {
      type: 'category',
      data: hrData.value.map(h => h.measured_at),
      axisLabel: {
        color: textColor,
        formatter: (v: string) => new Date(v).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      },
    },
    yAxis: {
      type: 'value',
      min: 40,
      axisLabel: { color: textColor },
    },
    series: [{
      type: 'line',
      data: hrData.value.map(h => h.bpm),
      smooth: true,
      symbol: 'none',
      lineStyle: { color: '#ef4444', width: 2 },
      areaStyle: {
        color: {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(239, 68, 68, 0.3)' },
            { offset: 1, color: 'rgba(239, 68, 68, 0.02)' },
          ],
        },
      },
    }],
  }
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Heart Rate</h1>
      <div style="display: flex; gap: 0.5rem">
        <button v-for="d in [1, 3, 7]" :key="d" class="period-btn" :class="{ active: days === d }" @click="days = d; loadData()">
          {{ d === 1 ? 'Today' : d + 'd' }}
        </button>
      </div>
    </div>

    <div class="grid-3" style="margin-bottom: 1.5rem">
      <div class="stat-card">
        <div class="label">Average</div>
        <div class="value">{{ stats.avg ?? '—' }} <span class="unit">bpm</span></div>
      </div>
      <div class="stat-card">
        <div class="label">Min</div>
        <div class="value" style="color: var(--success)">{{ stats.min ?? '—' }} <span class="unit">bpm</span></div>
      </div>
      <div class="stat-card">
        <div class="label">Max</div>
        <div class="value" style="color: var(--danger)">{{ stats.max ?? '—' }} <span class="unit">bpm</span></div>
      </div>
    </div>

    <div class="chart-container">
      <h3>Heart Rate</h3>
      <v-chart v-if="hrData.length > 0" :option="chartOption" style="height: 350px" autoresize />
      <div v-else class="empty-state">
        <i class="pi pi-heart"></i>
        <p>No heart rate data available</p>
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
