<template>
  <div class="preview-shell" :class="{ 'is-dark': isDark, 'nav-is-open': isMobileNavOpen }">
    <a class="skip-link" href="#preview-main">跳到主要内容</a>

    <header class="topbar">
      <div class="topbar-brand-zone">
        <button
          class="icon-button mobile-menu-button"
          type="button"
          :aria-expanded="isMobileNavOpen"
          aria-controls="preview-sidebar"
          aria-label="打开导航"
          @click="isMobileNavOpen = !isMobileNavOpen"
        >
          <Icon name="menu" />
        </button>

        <button class="brand-lockup" type="button" @click="notifyPreview('首页入口')">
          <span class="brand-mark" aria-hidden="true">
            <img src="/brand/luoxue-snowpuff-extracted.svg" alt="" />
          </span>
          <span class="brand-copy">
            <strong>洛雪 API</strong>
          </span>
        </button>
      </div>

      <div class="topbar-context">
        <div class="page-context">
          <span class="context-kicker">控制台</span>
          <span class="context-divider" aria-hidden="true">/</span>
          <strong>管理概览</strong>
          <span class="preview-badge">首页风格预览</span>
        </div>

        <div class="topbar-actions">
          <span class="live-status"><i aria-hidden="true"></i> 演示数据</span>
          <button
            class="icon-button"
            type="button"
            :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
            :title="isDark ? '切换到浅色模式' : '切换到深色模式'"
            @click="isDark = !isDark"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" />
          </button>
          <button
            class="icon-button notification-button"
            type="button"
            aria-label="查看通知"
            title="查看通知"
            @click="notifyPreview('通知中心')"
          >
            <Icon name="bell" />
            <span aria-hidden="true"></span>
          </button>
          <button class="profile-button" type="button" @click="notifyPreview('账号菜单')">
            <span class="avatar">管</span>
            <span class="profile-copy">
              <strong>管理员</strong>
              <small>系统所有者</small>
            </span>
            <Icon name="chevronDown" size="sm" />
          </button>
        </div>
      </div>
    </header>

    <button
      v-if="isMobileNavOpen"
      class="nav-backdrop"
      type="button"
      aria-label="关闭导航"
      @click="isMobileNavOpen = false"
    ></button>

    <aside id="preview-sidebar" class="sidebar" aria-label="管理导航">
      <nav class="sidebar-nav">
        <div v-for="group in previewNavGroups" :key="group.label" class="nav-group">
          <p>{{ group.label }}</p>
          <button
            v-for="item in group.items"
            :key="item.label"
            class="nav-item"
            :class="{ active: item.active }"
            type="button"
            :aria-current="item.active ? 'page' : undefined"
            @click="handleNavigation(item.label, item.active)"
          >
            <span class="nav-icon"><Icon :name="item.icon" /></span>
            <span>{{ item.label }}</span>
            <span v-if="item.active" class="active-dot" aria-hidden="true"></span>
          </button>
        </div>
      </nav>

      <div class="sidebar-health">
        <div class="health-heading">
          <span class="health-icon"><Icon name="activity" /></span>
          <div>
            <strong>系统运行正常</strong>
            <small>所有服务在线</small>
          </div>
        </div>
        <div class="health-meter" aria-label="系统健康度 99.98%">
          <span></span>
        </div>
        <div class="health-meta">
          <span>健康度</span>
          <strong>99.98%</strong>
        </div>
      </div>
    </aside>

    <main id="preview-main" class="main-content">
      <div class="content-boundary">
        <section class="page-heading" aria-labelledby="dashboard-title">
          <div>
            <div class="eyebrow"><span></span> ADMIN OVERVIEW</div>
            <h1 id="dashboard-title">管理概览</h1>
            <p>查看平台关键指标、实时性能与用户用量。</p>
          </div>
          <div class="heading-actions">
            <div class="update-copy">
              <span>最近更新</span>
              <strong>今天 15:42</strong>
            </div>
            <button
              class="primary-button"
              type="button"
              :disabled="isRefreshing"
              @click="refreshPreview"
            >
              <Icon name="refresh" :class="{ spinning: isRefreshing }" />
              {{ isRefreshing ? '更新中' : '刷新数据' }}
            </button>
          </div>
        </section>

        <section class="preview-controller" aria-label="预览状态切换">
          <div class="controller-copy">
            <span class="controller-icon"><Icon name="sparkles" /></span>
            <div>
              <strong>视觉方案预览</strong>
              <span>本页不连接后台接口，也不会改动正式管理页</span>
            </div>
          </div>
          <div class="state-switcher" role="group" aria-label="页面数据状态">
            <button
              v-for="state in previewStates"
              :key="state.value"
              type="button"
              :class="{ active: activeState === state.value }"
              @click="activeState = state.value"
            >
              {{ state.label }}
            </button>
          </div>
        </section>

        <template v-if="activeState === 'loading'">
          <div class="loading-overview" aria-busy="true" aria-live="polite">
            <span class="sr-only">仪表盘正在加载</span>
            <div class="skeleton skeleton-title"></div>
            <div class="skeleton-metrics">
              <div v-for="index in 4" :key="index" class="skeleton-metric">
                <div class="skeleton skeleton-icon"></div>
                <div class="skeleton skeleton-line short"></div>
                <div class="skeleton skeleton-number"></div>
                <div class="skeleton skeleton-line"></div>
              </div>
            </div>
          </div>
          <div class="loading-grid">
            <div class="loading-panel"><div class="skeleton skeleton-chart"></div></div>
            <div class="loading-panel"><div class="skeleton skeleton-chart"></div></div>
          </div>
        </template>

        <section v-else-if="activeState === 'empty'" class="state-panel" aria-live="polite">
          <span class="state-illustration"><Icon name="chartNoAxesColumn" size="xl" /></span>
          <span class="state-kicker">等待第一条记录</span>
          <h2>还没有可展示的用量数据</h2>
          <p>当用户开始调用 API 后，请求、Token 与费用趋势会在这里出现。</p>
          <button class="secondary-button" type="button" @click="activeState = 'default'">
            <Icon name="play" />
            查看演示数据
          </button>
        </section>

        <section v-else-if="activeState === 'error'" class="state-panel error-state" aria-live="assertive">
          <span class="state-illustration"><Icon name="exclamationTriangle" size="xl" /></span>
          <span class="state-kicker">数据暂时不可用</span>
          <h2>概览加载失败</h2>
          <p>服务连接出现波动，现有数据不会被覆盖。你可以稍后再试。</p>
          <button class="primary-button" type="button" @click="recoverPreview">
            <Icon name="refresh" />
            重新加载
          </button>
        </section>

        <template v-else>
          <section class="surface overview-surface" aria-labelledby="overview-title">
            <div class="section-heading">
              <div>
                <span class="section-kicker">实时总览</span>
                <h2 id="overview-title">平台关键指标</h2>
              </div>
              <span class="healthy-pill"><i></i> 运行稳定</span>
            </div>

            <div class="metric-grid">
              <article v-for="metric in previewMetrics" :key="metric.label" class="metric-item">
                <div class="metric-topline">
                  <span class="metric-icon"><Icon :name="metric.icon" /></span>
                  <span
                    class="metric-trend"
                    :title="metric.trendAriaLabel"
                    :aria-label="metric.trendAriaLabel"
                    role="group"
                  >{{ metric.trend }} <small>{{ metric.trendLabel }}</small></span>
                </div>
                <p>{{ metric.label }}</p>
                <strong>{{ metric.value }}</strong>
                <span>{{ metric.detail }}</span>
              </article>
            </div>
          </section>

          <div class="dashboard-grid operational-grid">
            <section class="surface performance-panel" aria-labelledby="performance-title">
              <div class="section-heading compact">
                <div>
                  <span class="section-kicker">PERFORMANCE</span>
                  <h2 id="performance-title">实时性能</h2>
                </div>
                <span class="subtle-status"><i></i> 30 秒前同步</span>
              </div>
              <div class="performance-grid">
                <article v-for="item in previewPerformance" :key="item.label">
                  <span class="performance-icon"><Icon :name="item.icon" /></span>
                  <div>
                    <p>{{ item.label }}</p>
                    <strong>{{ item.value }}</strong>
                    <span>{{ item.meta }}</span>
                  </div>
                </article>
              </div>
            </section>

            <section class="surface quick-panel" aria-labelledby="quick-title">
              <div class="section-heading compact">
                <div>
                  <span class="section-kicker">SHORTCUTS</span>
                  <h2 id="quick-title">快捷操作</h2>
                </div>
              </div>
              <div class="quick-actions">
                <button type="button" @click="notifyPreview('批量生图')">
                  <span class="quick-icon violet"><Icon name="photo" /></span>
                  <span><strong>批量生图</strong><small>创建图像生成任务</small></span>
                  <Icon name="chevronRight" size="sm" />
                </button>
                <button type="button" @click="notifyPreview('分组定价')">
                  <span class="quick-icon blue"><Icon name="calculator" /></span>
                  <span><strong>分组定价</strong><small>调整费率与权限策略</small></span>
                  <Icon name="chevronRight" size="sm" />
                </button>
              </div>
              <button class="text-button" type="button" @click="notifyPreview('全部管理工具')">
                查看全部管理工具 <Icon name="arrowRight" size="sm" />
              </button>
            </section>
          </div>

          <section class="filter-bar" aria-label="趋势筛选">
            <div class="range-control" role="group" aria-label="时间范围">
              <button
                v-for="range in dateRanges"
                :key="range"
                type="button"
                :class="{ active: activeRange === range }"
                @click="activeRange = range"
              >
                {{ range }}
              </button>
            </div>
            <button class="date-button" type="button" @click="notifyPreview('自定义日期范围')">
              <Icon name="calendar" />
              2026/07/20 － 2026/07/20
              <Icon name="chevronDown" size="sm" />
            </button>
            <div class="granularity-control" role="group" aria-label="统计粒度">
              <button
                v-for="option in ['小时', '日']"
                :key="option"
                type="button"
                :class="{ active: granularity === option }"
                @click="granularity = option"
              >
                {{ option }}
              </button>
            </div>
          </section>

          <div class="dashboard-grid charts-grid preview-real-charts">
            <div class="preview-real-chart preview-model-chart">
              <ModelDistributionChart
                variant="home-clay"
                :model-stats="previewModelStats"
                :loading="false"
                :enable-breakdown="false"
                :show-account-cost="false"
              />
            </div>
            <div class="preview-real-chart preview-trend-chart">
              <TokenUsageTrend variant="home-clay" :trend-data="previewTrendData" :loading="false" />
            </div>
          </div>

          <DashboardTopUsers
            :items="previewRankingItems"
            :trend="previewUserTrend"
            :visible-limit="4"
            @select="notifyPreview('用户用量明细')"
            @retry="notifyPreview('重新加载用户排行')"
            @view-all="notifyPreview('完整用量明细')"
          />
        </template>
      </div>
    </main>

    <transition name="toast">
      <div v-if="toastMessage" class="preview-toast" role="status">
        <Icon name="infoCircle" />
        <span><strong>预览模式</strong>{{ toastMessage }}仅展示视觉反馈，未执行真实操作。</span>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import DashboardTopUsers from '@/components/admin/dashboard/DashboardTopUsers.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import {
  previewMetrics,
  previewModelStats,
  previewNavGroups,
  previewPerformance,
  previewRankingItems,
  previewStates,
  previewTrendData,
  previewUserTrend,
  type PreviewState
} from './adminDashboardPreview.fixture'

const validStates = new Set<PreviewState>(previewStates.map(state => state.value))
const requestedState = typeof window !== 'undefined'
  ? new URLSearchParams(window.location.search).get('state')
  : null

const activeState = ref<PreviewState>(
  requestedState && validStates.has(requestedState as PreviewState)
    ? requestedState as PreviewState
    : 'default'
)
const isDark = ref(false)
const isMobileNavOpen = ref(false)
const isRefreshing = ref(false)
const toastMessage = ref('')
const activeRange = ref('最近 24 小时')
const granularity = ref('小时')
const dateRanges = ['最近 24 小时', '7 天', '30 天'] as const

let toastTimer: ReturnType<typeof setTimeout> | undefined

const notifyPreview = (target: string) => {
  toastMessage.value = target
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastMessage.value = ''
  }, 2600)
}

const handleNavigation = (label: string, isActive?: boolean) => {
  isMobileNavOpen.value = false
  if (!isActive) notifyPreview(label)
}

const refreshPreview = () => {
  if (isRefreshing.value) return
  isRefreshing.value = true
  setTimeout(() => {
    isRefreshing.value = false
    notifyPreview('刷新数据')
  }, 720)
}

const recoverPreview = () => {
  activeState.value = 'loading'
  setTimeout(() => {
    activeState.value = 'default'
  }, 720)
}
</script>

<style scoped src="./AdminDashboardHomePreviewView.css"></style>
