<template>
  <span
    v-if="hasData"
    class="dashboard-sparkline"
    aria-hidden="true"
  >
    <Line :data="chartData" :options="chartOptions" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  CategoryScale,
  Chart as ChartJS,
  LinearScale,
  LineElement,
  PointElement,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'vue-chartjs'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement)

const props = defineProps<{
  values: number[]
  color: string
}>()

const sanitizedValues = computed(() => (
  props.values.map(value => Number.isFinite(value) ? value : 0)
))

const hasData = computed(() => sanitizedValues.value.some(value => value !== 0))

const chartData = computed<ChartData<'line', number[], string>>(() => ({
  labels: sanitizedValues.value.map((_, index) => String(index)),
  datasets: [
    {
      data: sanitizedValues.value,
      borderColor: props.color,
      borderWidth: 2,
      borderCapStyle: 'round',
      borderJoinStyle: 'round',
      pointRadius: 0,
      pointHoverRadius: 0,
      fill: false,
      tension: 0,
    },
  ],
}))

const chartOptions: ChartOptions<'line'> = {
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  events: [],
  layout: {
    padding: 2,
  },
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      enabled: false,
    },
  },
  scales: {
    x: {
      display: false,
      grid: {
        display: false,
      },
      border: {
        display: false,
      },
      ticks: {
        display: false,
      },
    },
    y: {
      display: false,
      beginAtZero: true,
      grid: {
        display: false,
      },
      border: {
        display: false,
      },
      ticks: {
        display: false,
      },
    },
  },
}
</script>

<style scoped>
.dashboard-sparkline {
  display: block;
  flex: 0 0 100px;
  width: 100px;
  height: 40px;
  pointer-events: none;
}
</style>
