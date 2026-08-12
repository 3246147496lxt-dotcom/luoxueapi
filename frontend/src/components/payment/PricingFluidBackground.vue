<template>
  <canvas
    ref="canvasRef"
    class="pricing-fluid-background"
    data-renderer="spectra-webgl"
    :data-tier="tier"
    :data-transitioning="transitioning ? 'true' : 'false'"
    aria-hidden="true"
  ></canvas>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

type PricingTier = 'low' | 'mid' | 'high'

interface FluidConfig {
  speed: number
  intensity: number
  fluidScale: number
  colors: number[][]
}

interface FluidTransition {
  from: FluidConfig
  to: FluidConfig
  startedAt: number
  duration: number
}

const props = defineProps<{
  tier: PricingTier
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const transitioning = ref(false)

const FLUID_CONFIGS: Record<PricingTier, FluidConfig> = {
  low: {
    speed: 0.6,
    intensity: 1.15,
    fluidScale: 1.4,
    colors: [
      [255, 255, 255],
      [234, 245, 255],
      [141, 204, 255],
      [11, 139, 237],
      [229, 231, 235],
      [243, 244, 246],
      [249, 250, 251],
      [255, 255, 255],
    ],
  },
  mid: {
    speed: 1.25,
    intensity: 1.5,
    fluidScale: 1.28,
    colors: [
      [255, 255, 255],
      [234, 245, 255],
      [124, 58, 237],
      [139, 92, 246],
      [109, 40, 217],
      [167, 139, 250],
      [91, 33, 182],
      [255, 255, 255],
    ],
  },
  high: {
    speed: 1.8,
    intensity: 2.1,
    fluidScale: 1.1,
    colors: [
      [255, 255, 255],
      [219, 39, 119],
      [124, 58, 237],
      [245, 158, 11],
      [190, 24, 93],
      [139, 92, 246],
      [236, 72, 153],
      [255, 255, 255],
    ],
  },
}

const VERTEX_SHADER = `
  attribute vec2 position;
  varying vec2 vUv;
  void main() {
    vUv = position * 0.5 + 0.5;
    gl_Position = vec4(position, 0.0, 1.0);
  }
`

const FRAGMENT_SHADER = `
  precision highp float;
  varying vec2 vUv;
  uniform float uTime;
  uniform vec2 uResolution;
  uniform float uSpeed;
  uniform float uIntensity;
  uniform float uFluidScale;
  uniform vec3 uColors[8];

  float hash21(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
  }

  float valueNoise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = hash21(i);
    float b = hash21(i + vec2(1.0, 0.0));
    float c = hash21(i + vec2(0.0, 1.0));
    float d = hash21(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
  }

  float fbm(vec2 p) {
    float v = 0.0;
    float a = 0.5;
    mat2 rot = mat2(0.83, -0.56, 0.56, 0.83);
    for (int i = 0; i < 4; i++) {
      v += a * valueNoise(p);
      p = rot * p * 2.01 + vec2(3.11, 1.73);
      a *= 0.5;
    }
    return v;
  }

  mat2 rotate2d(float a) {
    float s = sin(a), c = cos(a);
    return mat2(c, -s, s, c);
  }

  vec2 applyVortex(vec2 p, vec2 c, float s, float r, float ph) {
    vec2 d = p - c;
    float dist = length(d);
    float inf = exp(-(dist * dist) / max(r * r, 0.0001));
    return c + rotate2d(s * inf * ph) * d;
  }

  float organicField(vec2 q, float f) {
    return exp(-dot(q, q) * f);
  }

  void main() {
    vec2 uv = vUv;
    float aspect = uResolution.x / max(uResolution.y, 1.0);
    vec2 p = uv - 0.5;
    p.x *= aspect;
    float scale = max(uFluidScale, 0.35);
    float t = uTime;
    float rightAct = smoothstep(0.1, 0.5, uv.x);
    float surge = (0.5 + 0.5 * sin(t * 1.3)) * 0.4
      + (0.5 + 0.5 * sin(t * 2.1 + 1.7)) * 0.35
      + (0.5 + 0.5 * cos(t * 0.8 - 0.9)) * 0.25;
    float surgeStr = mix(0.78, 1.42, surge);
    vec2 dom = p / scale;
    vec2 flowA = vec2(
      fbm(dom * 0.7 + vec2(t * 0.16, -t * 0.1)),
      fbm(dom * 0.82 + vec2(-t * 0.14, t * 0.12) + 5.2)
    ) * 2.0 - 1.0;
    vec2 warpedP = p + flowA * (0.34 * surgeStr) * scale * rightAct;
    vec2 flowB = vec2(
      fbm(warpedP * 1.65 / scale + vec2(-t * 0.33, t * 0.2) + 3.4),
      fbm(warpedP * 1.85 / scale + vec2(t * 0.28, -t * 0.24) - 2.7)
    ) * 2.0 - 1.0;
    warpedP += flowB * (0.2 * surgeStr) * scale * rightAct;
    vec2 flowC = vec2(
      fbm(warpedP * 3.4 / scale + vec2(t * 0.52, -t * 0.38) + 7.0),
      fbm(warpedP * 3.9 / scale + vec2(-t * 0.44, t * 0.48) - 4.0)
    ) * 2.0 - 1.0;
    warpedP += flowC * (0.085 * surgeStr) * scale * rightAct;

    vec2 vA = vec2((0.53 - 0.5) * aspect, 0.22);
    vec2 vB = vec2((0.65 - 0.5) * aspect, -0.02);
    vec2 vC = vec2((0.49 - 0.5) * aspect, -0.24);
    float vBoost = mix(0.8, 1.25, surge);
    vec2 vW = applyVortex(
      warpedP,
      vA,
      1.25 * vBoost,
      0.34 * scale,
      0.56 + 0.44 * sin(t * 1.07 + 0.3)
    );
    vW = applyVortex(
      vW,
      vB,
      -1.1 * vBoost,
      0.29 * scale,
      0.52 + 0.48 * cos(t * 1.31 + 1.8)
    );
    vW = applyVortex(
      vW,
      vC,
      0.95 * vBoost,
      0.32 * scale,
      0.58 + 0.42 * sin(t * 0.91 + 3.1)
    );
    warpedP = mix(warpedP, vW, rightAct);

    vec2 cPink = vec2(
      (0.55 + sin(t * 1.05) * 0.14 - 0.5) * aspect,
      0.1 + cos(t * 0.84) * 0.18
    );
    vec2 qPink = (warpedP - cPink) / scale;
    float pinkField = clamp(
      organicField(qPink + flowB * 0.08, 3.3)
        * (uIntensity / 1.65)
        * mix(0.86, 1.22, surge),
      0.0,
      1.0
    );

    vec2 cOrng = vec2(
      (0.51 + cos(t * 0.92 + 1.4) * 0.16 - 0.5) * aspect,
      -0.2 + sin(t * 1.16 + 0.8) * 0.17
    );
    vec2 qOrng = (warpedP - cOrng) / scale;
    float orngField = clamp(
      organicField(qOrng - flowA * 0.09, 3.0)
        * (uIntensity / 1.65)
        * mix(1.2, 0.82, surge),
      0.0,
      1.0
    );

    vec2 cCoral = vec2(
      (0.47 + sin(t * 0.77 + 2.2) * 0.12 - 0.5) * aspect,
      0.02 + cos(t * 1.02) * 0.15
    );
    vec2 qCoral = (warpedP - cCoral) / scale;
    float coralField = clamp(
      organicField(qCoral + flowC * 0.12, 3.65)
        * (uIntensity / 1.65)
        * (0.9 + surge * 0.22),
      0.0,
      1.0
    );

    float fluidMask = smoothstep(-0.03, 0.50, uv.x);
    vec3 color = uColors[0];
    color = mix(color, uColors[1], organicField(warpedP / scale, 2.15) * 0.58);
    color = mix(color, uColors[2], smoothstep(0.08, 0.7, coralField) * 0.72);
    color = mix(color, uColors[3], smoothstep(0.06, 0.72, orngField) * 0.88);
    color = mix(color, uColors[4], smoothstep(0.08, 0.78, pinkField) * 0.88);

    color = mix(uColors[0], color, fluidMask);
    color = mix(color, uColors[0], 0.22 * smoothstep(0.58, 1.0, uv.x));
    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`

const TRANSITION_MS = 600
const cloneConfig = (value: FluidConfig): FluidConfig => ({
  ...value,
  colors: value.colors.map((color) => [...color]),
})
const mix = (from: number, to: number, amount: number) => from + (to - from) * amount
const clamp01 = (value: number) => Math.min(1, Math.max(0, value))
const easeOutCubic = (value: number) => 1 - Math.pow(1 - value, 3)
const mixConfigs = (from: FluidConfig, to: FluidConfig, amount: number): FluidConfig => ({
  ...to,
  speed: mix(from.speed, to.speed, amount),
  intensity: mix(from.intensity, to.intensity, amount),
  fluidScale: mix(from.fluidScale, to.fluidScale, amount),
  colors: from.colors.map((color, colorIndex) => (
    color.map((channel, channelIndex) => (
      mix(channel, to.colors[colorIndex][channelIndex], amount)
    ))
  )),
})

let selectTier: ((tier: PricingTier) => void) | null = null
let disposeRenderer: (() => void) | null = null

watch(() => props.tier, (tier) => {
  selectTier?.(tier)
})

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return

  const gl = canvas.getContext('webgl', { antialias: true, alpha: false })
  if (!gl) return

  const compileShader = (source: string, type: number) => {
    const shader = gl.createShader(type)
    if (!shader) return null
    gl.shaderSource(shader, source)
    gl.compileShader(shader)
    if (gl.getShaderParameter(shader, gl.COMPILE_STATUS)) return shader
    gl.deleteShader(shader)
    return null
  }

  const vertexShader = compileShader(VERTEX_SHADER, gl.VERTEX_SHADER)
  const fragmentShader = compileShader(FRAGMENT_SHADER, gl.FRAGMENT_SHADER)
  const program = gl.createProgram()
  if (!vertexShader || !fragmentShader || !program) return

  gl.attachShader(program, vertexShader)
  gl.attachShader(program, fragmentShader)
  gl.linkProgram(program)
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    gl.deleteProgram(program)
    gl.deleteShader(vertexShader)
    gl.deleteShader(fragmentShader)
    return
  }

  const positionBuffer = gl.createBuffer()
  if (!positionBuffer) return
  gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer)
  gl.bufferData(
    gl.ARRAY_BUFFER,
    new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]),
    gl.STATIC_DRAW,
  )

  const positionAttribute = gl.getAttribLocation(program, 'position')
  if (positionAttribute < 0) return
  gl.enableVertexAttribArray(positionAttribute)
  gl.vertexAttribPointer(positionAttribute, 2, gl.FLOAT, false, 0, 0)

  const uniforms = {
    time: gl.getUniformLocation(program, 'uTime'),
    resolution: gl.getUniformLocation(program, 'uResolution'),
    speed: gl.getUniformLocation(program, 'uSpeed'),
    intensity: gl.getUniformLocation(program, 'uIntensity'),
    fluidScale: gl.getUniformLocation(program, 'uFluidScale'),
    colors: gl.getUniformLocation(program, 'uColors'),
  }

  const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  let currentTier = props.tier
  let currentConfig = cloneConfig(FLUID_CONFIGS[currentTier])
  let transition: FluidTransition | null = null
  let fluidPhase = 0
  let lastIntegratedSpeed = currentConfig.speed
  let lastFrameAt: number | null = null
  let animationFrameId: number | null = null
  let running = false
  let isIntersecting = true

  const sampleTransition = (now: number) => {
    if (!transition) return currentConfig
    const progress = clamp01((now - transition.startedAt) / transition.duration)
    currentConfig = mixConfigs(
      transition.from,
      transition.to,
      easeOutCubic(progress),
    )
    if (progress >= 1) {
      currentConfig = cloneConfig(transition.to)
      transition = null
      transitioning.value = false
    }
    return currentConfig
  }

  const syncCanvasSize = () => {
    const rect = canvas.getBoundingClientRect()
    const dpr = Math.min(window.devicePixelRatio || 1, 1.5)
    const width = Math.max(1, Math.round(rect.width * dpr))
    const height = Math.max(1, Math.round(rect.height * dpr))
    if (canvas.width === width && canvas.height === height) return
    canvas.width = width
    canvas.height = height
    gl.viewport(0, 0, width, height)
  }

  const drawFrame = (now: number, sampledConfig?: FluidConfig) => {
    syncCanvasSize()
    const config = sampledConfig ?? sampleTransition(now)
    gl.useProgram(program)
    gl.uniform1f(uniforms.time, fluidPhase)
    gl.uniform2f(uniforms.resolution, canvas.width, canvas.height)
    gl.uniform1f(uniforms.speed, config.speed)
    gl.uniform1f(uniforms.intensity, config.intensity)
    gl.uniform1f(uniforms.fluidScale, config.fluidScale)
    gl.uniform3fv(
      uniforms.colors,
      new Float32Array(config.colors.flatMap((color) => color.map((value) => value / 255))),
    )
    gl.drawArrays(gl.TRIANGLES, 0, 6)
  }

  const stopLoop = () => {
    if (animationFrameId !== null) cancelAnimationFrame(animationFrameId)
    animationFrameId = null
    running = false
    lastFrameAt = null
  }

  const frame = (now: number) => {
    if (!running) return
    const deltaSeconds = lastFrameAt === null
      ? 0
      : Math.min(0.05, Math.max(0, (now - lastFrameAt) / 1000))
    lastFrameAt = now
    const config = sampleTransition(now)
    fluidPhase += deltaSeconds * ((lastIntegratedSpeed + config.speed) * 0.5)
    lastIntegratedSpeed = config.speed
    drawFrame(now, config)
    animationFrameId = requestAnimationFrame(frame)
  }

  const startLoop = () => {
    if (running || document.hidden || !isIntersecting || motionQuery.matches) return
    running = true
    const now = performance.now()
    lastFrameAt = now
    lastIntegratedSpeed = sampleTransition(now).speed
    animationFrameId = requestAnimationFrame(frame)
  }

  selectTier = (tier) => {
    const now = performance.now()
    if (tier === currentTier && !transition) return
    currentConfig = cloneConfig(sampleTransition(now))
    currentTier = tier
    const nextConfig = cloneConfig(FLUID_CONFIGS[tier])
    if (motionQuery.matches) {
      currentConfig = nextConfig
      transition = null
      transitioning.value = false
    } else {
      transition = {
        from: cloneConfig(currentConfig),
        to: nextConfig,
        startedAt: now,
        duration: TRANSITION_MS,
      }
      transitioning.value = true
    }
    if (motionQuery.matches) {
      drawFrame(now)
    } else {
      startLoop()
      if (!running) {
        currentConfig = cloneConfig(nextConfig)
        transition = null
        transitioning.value = false
        drawFrame(now, currentConfig)
      }
    }
  }

  const handleVisibilityChange = () => {
    if (document.hidden) stopLoop()
    else startLoop()
  }
  const handleMotionChange = () => {
    if (motionQuery.matches) {
      if (transition) currentConfig = cloneConfig(transition.to)
      transition = null
      transitioning.value = false
      stopLoop()
      drawFrame(performance.now())
    } else {
      startLoop()
    }
  }
  const handleContextLost = (event: Event) => {
    event.preventDefault()
    stopLoop()
  }
  const handleResize = () => {
    if (!running) drawFrame(performance.now())
  }

  document.addEventListener('visibilitychange', handleVisibilityChange)
  motionQuery.addEventListener('change', handleMotionChange)
  canvas.addEventListener('webglcontextlost', handleContextLost, false)

  const intersectionObserver = typeof IntersectionObserver === 'undefined'
    ? null
    : new IntersectionObserver(([entry]) => {
      isIntersecting = entry.isIntersecting
      if (isIntersecting) startLoop()
      else stopLoop()
    }, { threshold: 0.01 })
  intersectionObserver?.observe(canvas)

  const resizeObserver = typeof ResizeObserver === 'undefined'
    ? null
    : new ResizeObserver(handleResize)
  if (resizeObserver) resizeObserver.observe(canvas)
  else window.addEventListener('resize', handleResize)

  drawFrame(performance.now())
  startLoop()

  disposeRenderer = () => {
    stopLoop()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    motionQuery.removeEventListener('change', handleMotionChange)
    canvas.removeEventListener('webglcontextlost', handleContextLost, false)
    intersectionObserver?.disconnect()
    resizeObserver?.disconnect()
    if (!resizeObserver) window.removeEventListener('resize', handleResize)
    gl.deleteBuffer(positionBuffer)
    gl.deleteProgram(program)
    gl.deleteShader(vertexShader)
    gl.deleteShader(fragmentShader)
  }
})

onBeforeUnmount(() => {
  selectTier = null
  disposeRenderer?.()
  disposeRenderer = null
})
</script>

<style scoped>
.pricing-fluid-background {
  display: block;
  width: 100%;
  height: 100%;
  background: #fff;
}
</style>
