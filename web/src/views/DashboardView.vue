<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { healthApi, type DashboardSummary, type StepsData, type SleepData } from '../api/health'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent])

const summary = ref<DashboardSummary | null>(null)
const weekSteps = ref<StepsData[]>([])
const weekSleep = ref<SleepData[]>([])
const loading = ref(true)

function daysAgo(n: number) {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

onMounted(async () => {
  try {
    const [dashRes, stepsRes, sleepRes] = await Promise.all([
      healthApi.dashboard(),
      healthApi.steps(daysAgo(7), daysAgo(0)),
      healthApi.sleep(daysAgo(7), daysAgo(0)),
    ])
    summary.value = dashRes.data
    weekSteps.value = stepsRes.data || []
    weekSleep.value = sleepRes.data || []
  } catch (e) {
    console.error('Dashboard load error:', e)
  } finally {
    loading.value = false
  }
})

function formatNum(n: number | null | undefined) {
  if (n == null) return '—'
  return n.toLocaleString()
}

function formatDistance(m: number | null | undefined) {
  if (m == null) return '—'
  return (m / 1000).toFixed(1)
}

function formatSleep(min: number | null | undefined) {
  if (min == null) return '—'
  const h = Math.floor(min / 60)
  const m = min % 60
  return `${h}h ${m}m`
}

const stepsChartOption = ref({})
const sleepChartOption = ref({})

function buildCharts() {
  const isDark = document.documentElement.classList.contains('p-dark')
  const textColor = isDark ? '#94a3b8' : '#64748b'

  if (weekSteps.value.length > 0) {
    stepsChartOption.value = {
      tooltip: { trigger: 'axis' },
      grid: { left: 50, right: 20, top: 20, bottom: 30 },
      xAxis: {
        type: 'category',
        data: weekSteps.value.map(s => s.date.slice(5)),
        axisLabel: { color: textColor },
      },
      yAxis: { type: 'value', axisLabel: { color: textColor } },
      series: [{
        type: 'bar',
        data: weekSteps.value.map(s => s.total_steps || 0),
        itemStyle: { color: '#6366f1', borderRadius: [4, 4, 0, 0] },
      }],
    }
  }

  if (weekSleep.value.length > 0) {
    sleepChartOption.value = {
      tooltip: { trigger: 'axis', formatter: (p: any) => `${p[0].name}<br/>Sleep: ${(p[0].value / 60).toFixed(1)}h` },
      grid: { left: 50, right: 20, top: 20, bottom: 30 },
      xAxis: {
        type: 'category',
        data: weekSleep.value.map(s => s.date.slice(5)),
        axisLabel: { color: textColor },
      },
      yAxis: { type: 'value', name: 'hours', axisLabel: { color: textColor, formatter: (v: number) => (v / 60).toFixed(0) + 'h' } },
      series: [
        { name: 'Deep', type: 'bar', stack: 'sleep', data: weekSleep.value.map(s => s.deep_min || 0), itemStyle: { color: '#3b82f6' } },
        { name: 'Light', type: 'bar', stack: 'sleep', data: weekSleep.value.map(s => s.light_min || 0), itemStyle: { color: '#93c5fd' } },
        { name: 'REM', type: 'bar', stack: 'sleep', data: weekSleep.value.map(s => s.rem_min || 0), itemStyle: { color: '#a78bfa', borderRadius: [4, 4, 0, 0] } },
      ],
    }
  }
}

onMounted(() => {
  setTimeout(buildCharts, 100)
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Dashboard</h1>
    </div>

    <div v-if="loading" class="empty-state">
      <i class="pi pi-spin pi-spinner"></i>
      <p>Loading...</p>
    </div>

    <template v-else>
      <div class="grid-4" style="margin-bottom: 1.5rem">
        <div class="stat-card">
          <div class="label">Steps Today</div>
          <div class="value">{{ formatNum(summary?.steps_today) }}</div>
          <div class="sub">{{ formatDistance(summary?.distance_today) }} km</div>
        </div>
        <div class="stat-card">
          <div class="label">Calories</div>
          <div class="value">{{ formatNum(summary?.calories_today) }}</div>
          <div class="sub">kcal burned</div>
        </div>
        <div class="stat-card">
          <div class="label">Sleep</div>
          <div class="value" style="font-size: 1.75rem">{{ formatSleep(summary?.sleep_last_night_min) }}</div>
          <div class="sub">last night</div>
        </div>
        <div class="stat-card">
          <div class="label">Heart Rate</div>
          <div class="value">
            {{ summary?.last_hr ?? '—' }}
            <span class="unit">bpm</span>
          </div>
          <div class="sub">avg {{ summary?.avg_hr_today ? Math.round(summary.avg_hr_today) + ' bpm' : '—' }}</div>
        </div>
      </div>

      <div class="grid-2">
        <div class="chart-container">
          <h3>Steps (7 days)</h3>
          <v-chart
            v-if="weekSteps.length > 0"
            :option="stepsChartOption"
            style="height: 280px"
            autoresize
          />
          <div v-else class="empty-state">
            <i class="pi pi-chart-bar"></i>
            <p>No step data yet. Link your Mi Fitness account in Settings.</p>
          </div>
        </div>

        <div class="chart-container">
          <h3>Sleep (7 days)</h3>
          <v-chart
            v-if="weekSleep.length > 0"
            :option="sleepChartOption"
            style="height: 280px"
            autoresize
          />
          <div v-else class="empty-state">
            <i class="pi pi-moon"></i>
            <p>No sleep data yet. Link your Mi Fitness account in Settings.</p>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
