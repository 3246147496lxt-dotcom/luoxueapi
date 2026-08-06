export default {
  batchImageGuide: {
    title: '图片批量生成',
    description: '一次提交多条提示词，任务完成后可统一下载图片结果'
  },
  // Home Page
  home: {
    // KeyUsageView 也在复用这些基础键。
    viewDocs: '查看文档',
    docs: '文档',
    switchToLight: '切换到浅色模式',
    switchToDark: '切换到深色模式',
    dashboard: '控制台',
    login: '登录',
    nav: {
      ariaLabel: '首页导航',
      quickStart: '快速开始',
      tutorial: '使用教程',
      openMenu: '打开菜单',
      closeMenu: '关闭菜单'
    },
    hero: {
      status: 'GPT 已支持 · 其他模型暂不支持',
      title: '稳定接入 GPT API，按量计费',
      description:
        '创建 API 密钥并选择分组，即可接入兼容 OpenAI 的客户端；调用记录、Token 和费用可在控制台查询。',
      register: '注册并开始',
      login: '登录控制台',
      createKey: '创建 API 密钥',
      tutorial: '查看新手教程'
    },
    codeExample: {
      title: '使用兼容 OpenAI 的接口发起请求',
      description: '接口地址来自当前站点配置；API 密钥和模型名以控制台显示为准。',
      tabs: {
        curl: 'cURL',
        python: 'Python'
      },
      copy: '复制',
      copied: '已复制',
      copyFailed: '复制失败，请手动复制',
      copyAria: '复制 {language} 示例代码',
      copiedAria: '{language} 示例代码已复制'
    },
    facts: {
      gpt: 'OpenAI 兼容接口',
      usage: '用量明细可查',
      quota: '密钥额度可控'
    },
    capabilities: {
      title: '调用与密钥管理，一处完成',
      description: '调用记录、密钥限制和渠道状态都能在控制台中清晰查看。',
      imageAlt: '落雪API 仪表盘，展示余额、API 密钥、请求量和费用概览',
      items: {
        usage: {
          title: '核对调用明细',
          description: '按时间与模型查看调用记录，核对 Token 和费用。'
        },
        keys: {
          title: '控制密钥使用',
          description: '可停用密钥，并按需要设置额度等使用限制。'
        },
        diagnostics: {
          title: '快速排查异常',
          description: '发生调用异常时，结合使用记录与渠道状态缩小排查范围。'
        },
        quotaViewer: {
          title: '额度留在桌面',
          description: '查看周剩余、重置倒计时和月到期日，不必反复打开网页。',
          link: '查看桌面版'
        }
      }
    },
    steps: {
      title: '三步完成接入',
      description: '从创建密钥到启用配置，按顺序完成即可。',
      items: {
        account: {
          title: '注册或登录控制台',
          description: '进入仪表盘，确认账户和常用入口可正常显示。'
        },
        key: {
          title: '创建 API 密钥并选择有效分组',
          description: '填写名称并选择可用分组；不要选择 default，也不要留空。'
        },
        client: {
          title: '导入 CC Switch 并启用',
          description: '按新手教程导入配置，启用后重启 Codex 使新配置生效。'
        }
      },
      tutorial: '查看完整图文教程'
    },
    providers: {
      title: '模型支持情况',
      description: 'GPT 已支持，其他模型暂不支持。',
      supported: '已支持',
      unsupported: '暂不支持',
      note: '具体可用型号和倍率以控制台实时列表为准。',
      claude: 'Claude',
      gpt: 'GPT',
      gemini: 'Gemini',
      antigravity: 'Antigravity'
    },
    faq: {
      title: '常见问题',
      description: '开始接入前，你可能想先确认这些信息。',
      items: {
        models: {
          question: '目前支持哪些模型？',
          answer:
            '目前首页仅标记 GPT 为已支持。Claude、Gemini 和 Antigravity 暂不支持；具体型号与倍率以控制台实时列表为准。'
        },
        group: {
          question: '为什么必须选择分组？',
          answer:
            '分组决定密钥可用的平台、模型和计费范围。创建密钥时请选择有效分组，不要选择 default，也不要留空。'
        },
        endpoint: {
          question: '接口地址和模型名在哪里查看？',
          answer: '首页代码示例会使用当前站点的 API 地址；可用模型名请以控制台显示为准。'
        },
        billing: {
          question: '在哪里查看用量和费用？',
          answer: '登录控制台后，可在使用记录中查看请求、Token 和费用明细。'
        },
        client: {
          question: '如何导入 CC Switch？调用失败怎么办？',
          answer:
            '按照新手教程导入并启用配置，然后重启 Codex。若仍调用失败，请核对密钥、分组和模型，并查看使用记录与渠道状态。'
        }
      }
    },
    cta: {
      title: '准备开始接入 GPT？',
      description: '创建 API 密钥，按新手教程完成客户端配置。',
      button: '开始接入 GPT',
      tutorial: '查看新手教程'
    },
    footer: {
      ariaLabel: '页脚导航',
      tutorial: '使用教程',
      apiDocs: 'API 文档',
      channelStatus: '渠道状态',
      allRightsReserved: '保留所有权利。'
    }
  },

  modelCatalog: {
    navLabel: '模型广场',
    title: '模型广场',
    description: '无需登录即可查看已上架模型的模型 ID、能力范围和公开标准价格。',
    publicPriceNote: '这里展示所有注册用户可使用的公开标准价，价格单位为雪花额度；不包含专属分组、订阅优惠或用户个性倍率。最终费用以实际调用记录为准。',
    modelCount: '{count} 个模型',
    pricingUpdatedAt: '价格更新于 {time}',
    searchLabel: '搜索模型',
    searchPlaceholder: '搜索模型名称、模型 ID 或能力',
    filters: {
      ariaLabel: '模型筛选',
      provider: '厂商',
      category: '类型',
      all: '全部'
    },
    resultsTitle: '公开模型',
    resultCount: '显示 {count} 个结果',
    loading: '正在加载模型',
    featured: '推荐',
    copy: '复制',
    copied: '已复制',
    copySuccess: '模型 ID 已复制',
    copyModelAria: '复制模型 ID：{model}',
    contextWindow: '上下文窗口',
    maxOutput: '最大输出',
    capabilities: '模型能力',
    unavailable: {
      title: '模型广场暂未开放',
      description: '当前站点尚未启用公开模型广场，或暂时没有可公开展示的目录。'
    },
    error: {
      title: '模型目录加载失败',
      description: '暂时无法获取模型与价格信息，请稍后重试。',
      retry: '重新加载'
    },
    empty: {
      title: '暂未上架公开模型',
      description: '管理员完成模型审核并发布后，模型会显示在这里。'
    },
    noResults: {
      title: '没有匹配的模型',
      description: '换一个关键词，或清除厂商和类型筛选后再试。',
      clear: '清除筛选'
    },
    providers: {
      openai: 'OpenAI',
      anthropic: 'Anthropic',
      google: 'Google',
      gemini: 'Google',
      deepseek: 'DeepSeek',
      moonshot: 'Kimi',
      kimi: 'Kimi',
      zhipu: '智谱 AI',
      alibaba: '阿里云',
      qwen: '通义千问',
      xai: 'xAI',
      mistral: 'Mistral AI',
      meta: 'Meta'
    },
    categories: {
      chat: '文本对话',
      text: '文本对话',
      text_chat: '文本对话',
      reasoning: '推理',
      code: '代码',
      embedding: '嵌入',
      image: '图像',
      audio: '语音',
      video: '视频',
      multimodal: '多模态',
      other: '其他'
    },
    capabilityLabels: {
      vision: '视觉理解',
      image_input: '图像输入',
      reasoning: '深度推理',
      tools: '工具调用',
      tool_calling: '工具调用',
      function_calling: '函数调用',
      prompt_caching: '提示词缓存',
      caching: '缓存',
      cache: '缓存',
      pdf: 'PDF',
      web_search: '联网搜索',
      structured_output: '结构化输出',
      audio_input: '语音输入',
      audio_output: '语音输出',
      video: '视频'
    },
    pricing: {
      publicLabel: '公开标准价',
      details: '价格详情',
      dialogTitle: '公开价格详情',
      dialogDescription: '所有价格均以雪花额度展示。Token 模型按每百万 Token 展示；0 表示免费，“—”表示暂未提供该项价格。',
      billingMode: '计费方式',
      billingModes: {
        token: '按 Token 计费',
        perRequest: '按次计费',
        image: '按图计费'
      },
      input: '输入',
      output: '输出',
      cacheWrite: '缓存写入',
      cacheWrite1h: '缓存写入（1 小时）',
      cacheRead: '缓存读取',
      priorityInput: 'Priority 输入',
      priorityOutput: 'Priority 输出',
      priorityCacheWrite: 'Priority 缓存写入',
      priorityCacheRead: 'Priority 缓存读取',
      imageInput: '图像输入',
      imageOutput: '图像输出',
      request: '单次请求',
      image: '单张图片',
      perMillionTokens: '/ 百万 Token',
      perRequest: '/ 次',
      perImage: '/ 张',
      intervalTitle: '区间价格',
      longContext: '长上下文价格',
      peakRate: '高峰时段 {start}–{end}（{timezone}）按 {multiplier} 倍计费。',
      unknown: '暂未提供价格'
    },
    cta: {
      title: '选好模型后，进入控制台完成接入',
      description: '创建 API 密钥并选择有效分组，再将模型 ID 填入你的客户端配置。',
      register: '注册并使用',
      login: '登录控制台',
      dashboard: '进入控制台',
      availableChannels: '查看我的可用渠道',
      tutorial: '查看使用教程'
    },
    meta: {
      title: '模型广场',
      description: '查看落雪API已公开上架的模型、能力、上下文窗口和公开标准价格。'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key 用量查询',
    subtitle: '输入您的 API Key 以查看实时消费金额与使用状态',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: '查询',
    querying: '查询中...',
    privacyNote: '您的 Key 仅在浏览器本地处理，不会被存储',
    dateRange: '统计范围:',
    dateRangeToday: '今日',
    dateRange7d: '7 天',
    dateRange30d: '30 天',
    dateRange90d: '90 天',
    dateRangeCustom: '自定义',
    apply: '应用',
    used: '已使用',
    detailInfo: '详细信息',
    tokenStats: 'Token 统计',
    dailyDetail: '按日明细',
    modelStats: '模型用量统计',
    // Table headers
    date: '日期',
    model: '模型',
    requests: '请求数',
    inputTokens: '输入 Tokens',
    outputTokens: '输出 Tokens',
    cacheCreationTokens: '缓存创建',
    cacheReadTokens: '缓存读取',
    cacheWriteTokens: '缓存写入',
    totalTokens: '总 Tokens',
    cost: '费用',
    // Status
    quotaMode: 'Key 限额模式',
    walletBalance: '钱包余额',
    // Ring card titles
    totalQuota: '总额度',
    limit5h: '5 小时限额',
    limitDaily: '日限额',
    limit7d: '7 天限额',
    limitWeekly: '周限额',
    limitMonthly: '月限额',
    // Detail rows
    remainingQuota: '剩余额度',
    expiresAt: '过期时间',
    todayExpires: '(今日到期)',
    daysLeft: '({days} 天)',
    usedQuota: '已用额度',
    resetNow: '即将重置',
    subscriptionType: '订阅类型',
    subscriptionExpires: '订阅到期',
    // Usage stat cells
    todayRequests: '今日请求',
    todayInputTokens: '今日输入',
    todayOutputTokens: '今日输出',
    todayTokens: '今日 Tokens',
    todayCacheCreation: '今日缓存创建',
    todayCacheRead: '今日缓存读取',
    todayCost: '今日费用',
    rpmTpm: 'RPM / TPM',
    totalRequests: '累计请求',
    totalInputTokens: '累计输入',
    totalOutputTokens: '累计输出',
    totalTokensLabel: '累计 Tokens',
    totalCacheCreation: '累计缓存创建',
    totalCacheRead: '累计缓存读取',
    totalCost: '累计费用',
    avgDuration: '平均耗时',
    // Messages
    enterApiKey: '请输入 API Key',
    querySuccess: '查询成功',
    queryFailed: '查询失败',
    queryFailedRetry: '查询失败，请稍后重试',
    noDailyUsage: '暂无按日用量数据',
  },

  // Setup Wizard
  setup: {
    title: '落雪API 安装向导',
    description: '配置您的落雪API实例',
    database: {
      title: '数据库配置',
      description: '连接到您的 PostgreSQL 数据库',
      host: '主机',
      port: '端口',
      username: '用户名',
      password: '密码',
      databaseName: '数据库名称',
      sslMode: 'SSL 模式',
      passwordPlaceholder: '密码',
      ssl: {
        disable: '禁用',
        require: '要求',
        verifyCa: '验证 CA',
        verifyFull: '完全验证'
      }
    },
    redis: {
      title: 'Redis 配置',
      description: '连接到您的 Redis 服务器',
      host: '主机',
      port: '端口',
      password: '密码（可选）',
      database: '数据库',
      passwordPlaceholder: '密码',
      enableTls: '启用 TLS',
      enableTlsHint: '连接 Redis 时使用 TLS（公共 CA 证书）'
    },
    admin: {
      title: '管理员账户',
      description: '创建您的管理员账户',
      email: '邮箱',
      password: '密码',
      confirmPassword: '确认密码',
      passwordPlaceholder: '至少 8 个字符',
      confirmPasswordPlaceholder: '确认密码',
      passwordMismatch: '密码不匹配'
    },
    ready: {
      title: '准备安装',
      description: '检查您的配置并完成安装',
      database: '数据库',
      redis: 'Redis',
      adminEmail: '管理员邮箱'
    },
    status: {
      testing: '测试中...',
      success: '连接成功',
      testConnection: '测试连接',
      installing: '安装中...',
      completeInstallation: '完成安装',
      completed: '安装完成！',
      redirecting: '正在跳转到登录页面...',
      restarting: '服务正在重启，请稍候...',
      timeout: '服务重启时间超出预期，请手动刷新页面。'
    }
  },

  // Common
}
