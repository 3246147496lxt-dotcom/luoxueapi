export const messages = {
  'zh-CN': {
    nav: { overview: '总览', codex: 'Codex', requests: '请求', settings: '设置' },
    common: {
      running: '运行中', stopped: '已停止', connected: '已连接', unavailable: '不可用',
      restart: '重启网关', openCodex: '打开 Codex', copyCommand: '复制启动命令',
      save: '保存', cancel: '取消', retry: '重试', loading: '正在加载', all: '全部',
      primaryNavigation: '主导航', todaySummary: '今日汇总', actions: '操作'
    },
    overview: {
      title: '本地网关', todayRequests: '今日请求', todayTokens: '今日 Token',
      todayCost: '今日消费', balance: '可用余额', latency: '平均首字',
      route: '当前线路', model: '默认模型', recent: '最近请求',
      takeoverOn: 'Codex 已接管', takeoverOff: 'Codex 未接管',
      restartHint: '线路已更新，重启 Codex 后生效',
      sessionPinned: '已有会话保持原线路', noRoutes: '没有可用的 OpenAI 分组',
      noRoutesHint: 'Codex 保持未接管，请先在网页完成分组配置。', manageRoutes: '打开网页处理',
      noBalance: '可用余额不足', noBalanceHint: 'Codex 保持当前状态，请充值后再发起请求。', recharge: '前往充值'
    },
    codex: {
      title: 'Codex 接管', cli: 'Codex CLI', desktop: 'Codex Desktop',
      installed: '已安装', notFound: '未检测到', config: '配置状态',
      clean: '可安全接管', managed: '由落雪API管理', conflict: '检测到 CCSwitch 接管',
      invalid: 'Codex 配置无法解析', permission_denied: '没有 Codex 配置权限', applications: '应用',
      enable: '启用接管', restore: '恢复原配置', localAddress: '本地地址',
      localToken: '本机令牌', hidden: '仅保存在本机配置中', routeRequired: '请先选择可用线路和默认模型',
      balanceRequired: '余额不足，暂时不能启用接管'
    },
    requests: {
      title: '请求', privacy: '仅保存运行元数据，7 天后自动清除', time: '时间', model: '模型',
      status: '状态', tokens: 'Token', duration: '耗时', requestId: '请求 ID', empty: '还没有请求记录',
      searchPlaceholder: '搜索模型或请求 ID', statusFilter: '状态筛选', loading: '正在加载请求', copyRequestId: '复制请求 ID'
    },
    settings: {
      title: '设置', account: '账号与设备', startup: '登录时启动',
      startupWarning: '关闭前将先恢复 Codex 配置，避免下次登录指向离线端口',
      theme: '外观', system: '跟随系统', light: '浅色', dark: '深色',
      language: '语言', retention: '本地记录保留', days: '{count} 天',
      notifications: '系统通知', update: '软件更新', checkUpdate: '检查更新',
      uninstall: '恢复并准备卸载', logout: '退出登录', device: '当前设备', systemSection: '系统',
      notificationsHint: '仅通知网关、设备、线路和更新异常', appearanceHint: '选择界面外观',
      retentionHint: '不保存提示词或响应正文', uninstallSection: '卸载',
      uninstallHint: '恢复 Codex、停止网关并关闭开机启动', upToDate: '已是最新版本',
      updateAvailable: '版本 {version} 可以安装', updateError: '检查更新失败',
      installUpdate: '安装并重启', installingUpdate: '正在安装', diagnostics: '诊断与支持',
      diagnosticPackage: '生成诊断摘要', diagnosticHint: '先检查脱敏内容，再由你确认上传',
      previewDiagnostic: '预览诊断', diagnosticPreview: '诊断摘要预览',
      diagnosticPrivacy: '仅包含下方运行元数据，不包含账号、路径、Key、请求头、提示词或响应正文。',
      confirmUpload: '确认上传', uploadingDiagnostic: '正在上传', diagnosticUploaded: '诊断已上传',
      diagnosticId: '诊断编号'
    },
    pairing: {
      brand: '落雪API Desktop', title: '让 Codex 连接落雪API',
      body: '在浏览器批准这台 Mac 后，App 会自动配置本地网关。',
      start: '连接落雪API账号', waiting: '等待浏览器确认', code: '配对码',
      openAgain: '重新打开浏览器', limit: '每个账号最多绑定 5 台设备'
    }
  },
  en: {
    nav: { overview: 'Overview', codex: 'Codex', requests: 'Requests', settings: 'Settings' },
    common: {
      running: 'Running', stopped: 'Stopped', connected: 'Connected', unavailable: 'Unavailable',
      restart: 'Restart gateway', openCodex: 'Open Codex', copyCommand: 'Copy launch command',
      save: 'Save', cancel: 'Cancel', retry: 'Retry', loading: 'Loading', all: 'All',
      primaryNavigation: 'Primary navigation', todaySummary: 'Today summary', actions: 'Actions'
    },
    overview: {
      title: 'Local gateway', todayRequests: 'Requests today', todayTokens: 'Tokens today',
      todayCost: 'Spend today', balance: 'Balance', latency: 'Avg. first token',
      route: 'Route', model: 'Default model', recent: 'Recent requests',
      takeoverOn: 'Codex managed', takeoverOff: 'Codex not managed',
      restartHint: 'Route updated. Restart Codex to apply.',
      sessionPinned: 'Existing sessions stay on their original route', noRoutes: 'No OpenAI route available',
      noRoutesHint: 'Codex remains unmanaged until route access is configured on the web.', manageRoutes: 'Open web settings',
      noBalance: 'Insufficient balance', noBalanceHint: 'Codex keeps its current state until the account is funded.', recharge: 'Add balance'
    },
    codex: {
      title: 'Codex takeover', cli: 'Codex CLI', desktop: 'Codex Desktop',
      installed: 'Installed', notFound: 'Not found', config: 'Configuration',
      clean: 'Ready to manage', managed: 'Managed by LuoxueAPI', conflict: 'CCSwitch takeover detected',
      invalid: 'Codex configuration cannot be parsed', permission_denied: 'Codex configuration permission denied', applications: 'Applications',
      enable: 'Enable takeover', restore: 'Restore configuration', localAddress: 'Local address',
      localToken: 'Local token', hidden: 'Stored only in the local Codex config', routeRequired: 'Select an available route and default model first',
      balanceRequired: 'Add balance before enabling takeover'
    },
    requests: {
      title: 'Requests', privacy: 'Only operational metadata is stored and removed after 7 days',
      time: 'Time', model: 'Model', status: 'Status', tokens: 'Tokens', duration: 'Duration',
      requestId: 'Request ID', empty: 'No requests yet', searchPlaceholder: 'Search model or request ID',
      statusFilter: 'Status filter', loading: 'Loading requests', copyRequestId: 'Copy request ID'
    },
    settings: {
      title: 'Settings', account: 'Account and device', startup: 'Launch at login',
      startupWarning: 'Turning this off restores Codex first so it never points at an offline gateway',
      theme: 'Appearance', system: 'System', light: 'Light', dark: 'Dark', language: 'Language',
      retention: 'Local history', days: '{count} days', notifications: 'System notifications',
      update: 'Software updates', checkUpdate: 'Check for updates', uninstall: 'Restore and prepare to uninstall',
      logout: 'Sign out', device: 'Current device', systemSection: 'System',
      notificationsHint: 'Gateway, device, route and update alerts only', appearanceHint: 'Choose the app appearance',
      retentionHint: 'No prompt or response content is stored', uninstallSection: 'Uninstall',
      uninstallHint: 'Restore Codex, stop the gateway and disable launch at login', upToDate: 'Up to date',
      updateAvailable: 'Version {version} is ready', updateError: 'Update check failed',
      installUpdate: 'Install and restart', installingUpdate: 'Installing', diagnostics: 'Diagnostics and support',
      diagnosticPackage: 'Generate diagnostic summary', diagnosticHint: 'Review the redacted data before confirming upload',
      previewDiagnostic: 'Preview diagnostics', diagnosticPreview: 'Diagnostic summary preview',
      diagnosticPrivacy: 'Contains only the runtime metadata below. Account data, paths, keys, headers, prompts and responses are excluded.',
      confirmUpload: 'Confirm upload', uploadingDiagnostic: 'Uploading', diagnosticUploaded: 'Diagnostic uploaded',
      diagnosticId: 'Diagnostic ID'
    },
    pairing: {
      brand: 'LuoxueAPI Desktop', title: 'Connect Codex to LuoxueAPI',
      body: 'Approve this Mac in your browser, then the app will configure the local gateway.',
      start: 'Connect LuoxueAPI account', waiting: 'Waiting for browser approval', code: 'Pairing code',
      openAgain: 'Open browser again', limit: 'Up to 5 devices per account'
    }
  }
} as const
