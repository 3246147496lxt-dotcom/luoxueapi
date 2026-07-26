// Static visual-review fixtures only. These values never represent live admin data.
export const previewNavGroups = [
  {
    label: '总览',
    items: [
      { label: '管理仪表盘', icon: 'chartNoAxesColumn', active: true },
      { label: '运营监控', icon: 'activity', active: false }
    ]
  },
  {
    label: '用户与权限',
    items: [
      { label: '用户管理', icon: 'users', active: false },
      { label: 'API 密钥', icon: 'key', active: false },
      { label: '账号管理', icon: 'server', active: false }
    ]
  },
  {
    label: '计费管理',
    items: [
      { label: '用量明细', icon: 'coins', active: false },
      { label: '订单与订阅', icon: 'creditCard', active: false }
    ]
  },
  {
    label: '系统',
    items: [{ label: '系统设置', icon: 'cog', active: false }]
  }
] as const

export const previewMetrics = [
  {
    label: 'API 密钥',
    value: '1,284',
    detail: '1,106 个已启用 · 248 个本周活跃',
    icon: 'key',
    trend: '+4.8%',
    trendLabel: '较上周同期',
    trendAriaLabel: '预览数据：本周活跃 API 密钥较上周同期增长 4.8%'
  },
  {
    label: '账号资源',
    value: '64',
    detail: '58 当前健康 · 2 异常',
    icon: 'server',
    trend: '90.6%',
    trendLabel: '健康率',
    trendAriaLabel: '预览数据：账号健康率按当前健康账号数除以全部未删除账号数计算，58/64 = 90.6%；健康账号排除有效限流、过载、临时冷却与到期自动暂停，账号配额窗口不纳入本指标；2 个异常、3 个限流、1 个过载，状态可能重叠且不作相加'
  },
  {
    label: '今日请求',
    value: '182,460',
    detail: '累计 28,930,412 次',
    icon: 'arrowLeftRight',
    trend: '+12.4%',
    trendLabel: '较昨日同期',
    trendAriaLabel: '预览数据：今日请求量较昨日同期增长 12.4%'
  },
  {
    label: '今日新增用户',
    value: '126',
    detail: '总用户：8,642人',
    icon: 'userPlus',
    trend: '+8.2%',
    trendLabel: '较昨日同期',
    trendAriaLabel: '预览数据：今日新增用户较昨日同期增长 8.2%'
  }
] as const

export const previewPerformance = [
  {
    label: '今日 Token',
    value: '38.42M',
    meta: '实际扣费 ¥1,286.40',
    icon: 'coins'
  },
  {
    label: '累计 Token',
    value: '6.81B',
    meta: '标准计费 ¥326,840',
    icon: 'database'
  },
  {
    label: '实时吞吐',
    value: '286 RPM',
    meta: '1.72M TPM',
    icon: 'gauge'
  },
  {
    label: '平均响应',
    value: '842 ms',
    meta: '活跃用户 1,936',
    icon: 'timer'
  }
] as const

export const previewModels = [
  { name: 'Claude 4 Sonnet', value: '15.8M', share: 41, tone: 'violet' },
  { name: 'Gemini 2.5 Pro', value: '9.6M', share: 25, tone: 'blue' },
  { name: 'GPT-5', value: '7.3M', share: 19, tone: 'green' },
  { name: '其他模型', value: '5.7M', share: 15, tone: 'muted' }
] as const

export const previewUsers = [
  {
    rank: 1,
    name: '北极星工作室',
    initials: '北',
    model: 'Claude 4 Sonnet',
    tokens: '5.82M',
    cost: '¥194.62',
    requests: '28,460',
    activity: [24, 35, 30, 48, 43, 64, 72]
  },
  {
    rank: 2,
    name: 'Aster Labs',
    initials: 'A',
    model: 'Gemini 2.5 Pro',
    tokens: '4.97M',
    cost: '¥168.34',
    requests: '21,905',
    activity: [18, 28, 38, 34, 52, 58, 66]
  },
  {
    rank: 3,
    name: '林间产品组',
    initials: '林',
    model: 'GPT-5',
    tokens: '4.16M',
    cost: '¥141.08',
    requests: '19,284',
    activity: [32, 27, 36, 49, 45, 56, 60]
  },
  {
    rank: 4,
    name: 'Snow AI',
    initials: 'S',
    model: 'Claude 4 Sonnet',
    tokens: '3.62M',
    cost: '¥122.70',
    requests: '16,872',
    activity: [20, 31, 26, 42, 40, 51, 54]
  }
] as const

export const previewTimelineLabels = ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '现在']

export const previewModelStats: ModelStat[] = [
  { model: 'Claude 4 Sonnet', requests: 28_460, input_tokens: 10_200_000, output_tokens: 4_200_000, cache_creation_tokens: 600_000, cache_read_tokens: 800_000, total_tokens: 15_800_000, cost: 660, actual_cost: 528, account_cost: 396 },
  { model: 'Gemini 2.5 Pro', requests: 21_905, input_tokens: 6_300_000, output_tokens: 2_500_000, cache_creation_tokens: 300_000, cache_read_tokens: 500_000, total_tokens: 9_600_000, cost: 400, actual_cost: 321.6, account_cost: 240 },
  { model: 'GPT-5', requests: 19_284, input_tokens: 4_700_000, output_tokens: 1_900_000, cache_creation_tokens: 300_000, cache_read_tokens: 400_000, total_tokens: 7_300_000, cost: 300, actual_cost: 244.32, account_cost: 183.24 },
  { model: '其他模型', requests: 16_872, input_tokens: 3_800_000, output_tokens: 1_300_000, cache_creation_tokens: 200_000, cache_read_tokens: 400_000, total_tokens: 5_700_000, cost: 240, actual_cost: 190.48, account_cost: 142.86 }
]

const previewTrendInput = [1_100_000, 1_500_000, 2_000_000, 2_800_000, 3_600_000, 4_700_000, 6_300_000]
const previewTrendOutput = [500_000, 800_000, 1_100_000, 1_600_000, 2_000_000, 2_700_000, 3_600_000]
const previewTrendCache = [100_000, 200_000, 300_000, 500_000, 700_000, 1_000_000, 1_320_000]
const previewTrendActualCost = [80, 110, 140, 175, 210, 260, 311.4]
const previewTrendStandardCost = [100, 140, 180, 220, 260, 320, 380]

export const previewTrendData: TrendDataPoint[] = previewTimelineLabels.map((date, index) => {
  const cacheCreationTokens = Math.round(previewTrendCache[index] * 0.4)
  const cacheReadTokens = previewTrendCache[index] - cacheCreationTokens
  return {
    date,
    requests: [5_420, 7_160, 9_830, 13_250, 16_840, 21_300, 28_721][index],
    input_tokens: previewTrendInput[index],
    output_tokens: previewTrendOutput[index],
    cache_creation_tokens: cacheCreationTokens,
    cache_read_tokens: cacheReadTokens,
    total_tokens: previewTrendInput[index] + previewTrendOutput[index] + previewTrendCache[index],
    cost: previewTrendStandardCost[index],
    actual_cost: previewTrendActualCost[index]
  }
})

export const previewRankingItems: UserSpendingRankingItem[] = previewUsers.map((user, index) => ({
  user_id: index + 1,
  email: index === 0 ? 'northstar@example.com' : index === 1 ? 'aster@example.com' : index === 2 ? 'forest@example.com' : 'snow@example.com',
  username: user.name,
  main_model: user.model,
  actual_cost: Number(user.cost.replace(/[^0-9.]/g, '')),
  requests: Number(user.requests.replace(/,/g, '')),
  tokens: Number(user.tokens.replace('M', '')) * 1_000_000
}))

export const previewUserTrend: UserUsageTrendPoint[] = previewUsers.flatMap((user, userIndex) => (
  previewTimelineLabels.map((date, periodIndex) => ({
    date,
    user_id: userIndex + 1,
    email: previewRankingItems[userIndex].email,
    username: user.name,
    requests: Math.round(Number(user.requests.replace(/,/g, '')) * user.activity[periodIndex] / 300),
    tokens: user.activity[periodIndex] * 10_000,
    cost: user.activity[periodIndex] * 0.55,
    actual_cost: user.activity[periodIndex] * 0.42
  }))
))

export const previewStates = [
  { value: 'default', label: '完整数据' },
  { value: 'loading', label: '加载中' },
  { value: 'empty', label: '空数据' },
  { value: 'error', label: '异常状态' }
] as const

export type PreviewState = (typeof previewStates)[number]['value']
import type {
  ModelStat,
  TrendDataPoint,
  UserSpendingRankingItem,
  UserUsageTrendPoint
} from '@/types'
