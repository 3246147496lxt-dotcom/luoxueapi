export const LUOXUE_CLAY_CHART_CATEGORICAL = [
  '#7c3aed',
  '#0b8bed',
  '#10b981',
  '#a78bfa',
  '#38bdf8',
  '#34d399',
  '#6d28d9',
  '#0284c7',
  '#059669',
  '#c4b5fd',
  '#7dd3fc',
  '#6ee7b7',
] as const

export const getLuoxueClayChartTheme = (isDark: boolean) => ({
  text: isDark ? '#c5bccf' : '#635f69',
  grid: isDark ? 'rgba(255,255,255,0.11)' : 'rgba(91,80,112,0.14)',
  primary: isDark ? '#a78bfa' : '#7c3aed',
  info: isDark ? '#38bdf8' : '#0b8bed',
  infoSoft: isDark ? 'rgba(56,189,248,0.12)' : 'rgba(11,139,237,0.1)',
  primarySoft: isDark ? '#c4b5fd' : '#8b5cf6',
  success: isDark ? '#34d399' : '#10b981',
  neutral: isDark ? '#94a3b8' : '#64748b',
  categorical: LUOXUE_CLAY_CHART_CATEGORICAL,
})
