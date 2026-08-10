<template>
  <canvas
    ref="canvasRef"
    class="reasoning-max-canvas"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const FALLBACK_WIDTH = 198
const FALLBACK_HEIGHT = 26
const MAX_DEVICE_PIXEL_RATIO = 2
const MOTION_CYCLE_MS = 7800

const canvasRef = ref<HTMLCanvasElement | null>(null)
let context: CanvasRenderingContext2D | null = null
let animationFrame: number | null = null
let mediaQuery: MediaQueryList | null = null
let resizeObserver: ResizeObserver | null = null
let logicalWidth = FALLBACK_WIDTH
let logicalHeight = FALLBACK_HEIGHT
let elapsedTime = 0
let lastFrameTimestamp: number | null = null

function syncCanvasSize() {
  const canvas = canvasRef.value
  if (!canvas) return

  const rect = canvas.getBoundingClientRect()
  logicalWidth = Math.max(1, rect.width || FALLBACK_WIDTH)
  logicalHeight = Math.max(1, rect.height || FALLBACK_HEIGHT)
  const pixelRatio = Math.min(
    MAX_DEVICE_PIXEL_RATIO,
    Math.max(1, window.devicePixelRatio || 1),
  )
  const width = Math.round(logicalWidth * pixelRatio)
  const height = Math.round(logicalHeight * pixelRatio)
  if (canvas.width !== width) canvas.width = width
  if (canvas.height !== height) canvas.height = height
  context?.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0)
}

function draw(timestamp: number) {
  const ctx = context
  if (!ctx) return

  const width = logicalWidth
  const height = logicalHeight
  const phase = ((timestamp % MOTION_CYCLE_MS) / MOTION_CYCLE_MS) * Math.PI * 2

  ctx.clearRect(0, 0, width, height)
  ctx.globalCompositeOperation = 'source-over'

  const deepCenter = width * (0.5 + (Math.sin(phase + 2.15) * 0.54))
  const deepWave = ctx.createRadialGradient(
    deepCenter,
    height * 0.54,
    0,
    deepCenter,
    height * 0.54,
    width * 0.4,
  )
  deepWave.addColorStop(0, 'rgba(27, 5, 82, 0.46)')
  deepWave.addColorStop(0.46, 'rgba(52, 13, 122, 0.24)')
  deepWave.addColorStop(1, 'rgba(42, 9, 104, 0)')
  ctx.fillStyle = deepWave
  ctx.fillRect(0, 0, width, height)

  const magentaCenter = width * (0.5 + (Math.sin(phase) * 0.5))
  const magentaWave = ctx.createRadialGradient(
    magentaCenter,
    height * 0.42,
    0,
    magentaCenter,
    height * 0.42,
    width * 0.32,
  )
  magentaWave.addColorStop(0, 'rgba(197, 91, 232, 0.34)')
  magentaWave.addColorStop(0.42, 'rgba(146, 59, 218, 0.2)')
  magentaWave.addColorStop(1, 'rgba(91, 31, 177, 0)')
  ctx.fillStyle = magentaWave
  ctx.fillRect(0, 0, width, height)

  const violetCenter = width * (0.5 + (Math.sin(phase + 4.3) * 0.48))
  const violetWave = ctx.createRadialGradient(
    violetCenter,
    height * 0.68,
    0,
    violetCenter,
    height * 0.68,
    width * 0.27,
  )
  violetWave.addColorStop(0, 'rgba(111, 44, 205, 0.28)')
  violetWave.addColorStop(0.5, 'rgba(76, 21, 158, 0.14)')
  violetWave.addColorStop(1, 'rgba(58, 14, 126, 0)')
  ctx.fillStyle = violetWave
  ctx.fillRect(0, 0, width, height)
}

function stopAnimation() {
  if (animationFrame !== null) cancelAnimationFrame(animationFrame)
  animationFrame = null
  lastFrameTimestamp = null
}

function renderFrame(timestamp: number) {
  if (lastFrameTimestamp !== null) {
    elapsedTime += Math.max(0, timestamp - lastFrameTimestamp)
  }
  lastFrameTimestamp = timestamp
  draw(elapsedTime)
  animationFrame = requestAnimationFrame(renderFrame)
}

function startAnimation() {
  stopAnimation()
  syncCanvasSize()
  draw(elapsedTime)
  if (mediaQuery?.matches || document.hidden || !context) return
  animationFrame = requestAnimationFrame(renderFrame)
}

function onVisibilityChange() {
  if (document.hidden) {
    stopAnimation()
    return
  }
  startAnimation()
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  context = canvas.getContext('2d')
  mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  mediaQuery.addEventListener('change', startAnimation)
  document.addEventListener('visibilitychange', onVisibilityChange)
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => startAnimation())
    resizeObserver.observe(canvas)
  }
  startAnimation()
})

onBeforeUnmount(() => {
  stopAnimation()
  resizeObserver?.disconnect()
  resizeObserver = null
  mediaQuery?.removeEventListener('change', startAnimation)
  mediaQuery = null
  document.removeEventListener('visibilitychange', onVisibilityChange)
  context = null
})
</script>

<style scoped>
.reasoning-max-canvas {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
