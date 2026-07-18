export default {
  batchImageGuide: {
    title: 'Batch Image Generation',
    description: 'Submit multiple prompts in one job and download the generated images when complete'
  },
  // Home Page
  home: {
    // KeyUsageView also reuses these base keys.
    viewDocs: 'View documentation',
    docs: 'Docs',
    switchToLight: 'Switch to light mode',
    switchToDark: 'Switch to dark mode',
    dashboard: 'Dashboard',
    login: 'Sign in',
    nav: {
      ariaLabel: 'Home navigation',
      capabilities: 'Capabilities',
      steps: 'Get started',
      providers: 'Model status',
      faq: 'FAQ',
      tutorial: 'Tutorial',
      openMenu: 'Open menu',
      closeMenu: 'Close menu'
    },
    hero: {
      status: 'GPT supported · Other models not supported yet',
      title: 'Reliable GPT API access with usage-based billing',
      description:
        'Create an API key and choose a group to connect OpenAI-compatible clients. View request logs, token usage, and costs in the dashboard.',
      register: 'Sign up to start',
      login: 'Sign in to dashboard',
      createKey: 'Create an API key',
      tutorial: 'View the beginner tutorial'
    },
    codeExample: {
      title: 'Send a request through an OpenAI-compatible API',
      description:
        'The endpoint comes from the current site configuration. Use the API key and model ID shown in the dashboard.',
      tabs: {
        curl: 'cURL',
        python: 'Python'
      },
      copy: 'Copy',
      copied: 'Copied',
      copyFailed: 'Copy failed. Please copy the code manually.',
      copyAria: 'Copy the {language} example',
      copiedAria: '{language} example copied'
    },
    facts: {
      gpt: 'GPT supported',
      usage: 'Detailed usage records',
      quota: 'Controllable key limits'
    },
    capabilities: {
      title: 'Manage requests and API keys in one place',
      description: 'Review usage records, key limits, and channel status from the dashboard.',
      imageAlt:
        'Luoxue API dashboard showing balance, API keys, request volume, and cost overview',
      items: {
        usage: {
          title: 'Review usage details',
          description: 'View requests by time and model, including token usage and costs.'
        },
        keys: {
          title: 'Control API key usage',
          description: 'Disable keys and configure limits such as spending quotas when needed.'
        },
        diagnostics: {
          title: 'Troubleshoot faster',
          description:
            'Use request records and channel status to narrow down the cause of failed calls.'
        }
      }
    },
    steps: {
      title: 'Connect in three steps',
      description: 'Create a key, import the configuration, and enable it in order.',
      items: {
        account: {
          title: 'Sign up or sign in',
          description: 'Open the dashboard and confirm that your account and main controls are available.'
        },
        key: {
          title: 'Create an API key and choose a valid group',
          description: 'Enter a name and select an available group. Do not choose default or leave it blank.'
        },
        client: {
          title: 'Import into CC Switch and enable it',
          description:
            'Follow the beginner tutorial to import the configuration, enable it, and restart Codex.'
        }
      },
      tutorial: 'View the complete tutorial'
    },
    providers: {
      title: 'Model availability',
      description: 'GPT is supported. Other models are not supported yet.',
      supported: 'Supported',
      unsupported: 'Not supported yet',
      note: 'Check the dashboard for the latest available model IDs and multipliers.',
      claude: 'Claude',
      gpt: 'GPT',
      gemini: 'Gemini',
      antigravity: 'Antigravity'
    },
    faq: {
      title: 'Frequently asked questions',
      description: 'Key details to check before connecting a client.',
      items: {
        models: {
          question: 'Which models are currently supported?',
          answer:
            'GPT is currently the only model family marked as supported. Claude, Gemini, and Antigravity are not supported yet. Check the dashboard for the latest model IDs and multipliers.'
        },
        group: {
          question: 'Why do I need to choose a group?',
          answer:
            'The group determines which platforms, models, and billing scope the key can use. Select a valid group when creating the key; do not choose default or leave it blank.'
        },
        endpoint: {
          question: 'Where can I find the endpoint and model ID?',
          answer:
            'The code example uses the API endpoint configured for this site. Use a model ID shown in the dashboard.'
        },
        billing: {
          question: 'Where can I review usage and costs?',
          answer:
            'After signing in, open Usage Records to review requests, token usage, and cost details.'
        },
        client: {
          question: 'How do I import into CC Switch, and what if a call fails?',
          answer:
            'Follow the beginner tutorial to import and enable the configuration, then restart Codex. If calls still fail, check the API key, group, and model, then review Usage Records and Channel Status.'
        }
      }
    },
    cta: {
      title: 'Ready to connect GPT?',
      description: 'Create an API key and follow the beginner tutorial to configure your client.',
      button: 'Start using GPT',
      tutorial: 'View the beginner tutorial'
    },
    footer: {
      tutorial: 'Tutorial',
      apiDocs: 'API docs',
      channelStatus: 'Channel status',
      allRightsReserved: 'All rights reserved.'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key Usage',
    subtitle: 'Enter your API Key to view real-time spending and usage status',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: 'Query',
    querying: 'Querying...',
    privacyNote: 'Your Key is processed locally in the browser and will not be stored',
    dateRange: 'Date Range:',
    dateRangeToday: 'Today',
    dateRange7d: '7 Days',
    dateRange30d: '30 Days',
    dateRange90d: '90 Days',
    dateRangeCustom: 'Custom',
    apply: 'Apply',
    used: 'Used',
    detailInfo: 'Detail Information',
    tokenStats: 'Token Statistics',
    dailyDetail: 'Daily Detail',
    modelStats: 'Model Usage Statistics',
    // Table headers
    date: 'Date',
    model: 'Model',
    requests: 'Requests',
    inputTokens: 'Input Tokens',
    outputTokens: 'Output Tokens',
    cacheCreationTokens: 'Cache Creation',
    cacheReadTokens: 'Cache Read',
    cacheWriteTokens: 'Cache Write',
    totalTokens: 'Total Tokens',
    cost: 'Cost',
    // Status
    quotaMode: 'Key Quota Mode',
    walletBalance: 'Wallet Balance',
    // Ring card titles
    totalQuota: 'Total Quota',
    limit5h: '5-Hour Limit',
    limitDaily: 'Daily Limit',
    limit7d: '7-Day Limit',
    limitWeekly: 'Weekly Limit',
    limitMonthly: 'Monthly Limit',
    // Detail rows
    remainingQuota: 'Remaining Quota',
    expiresAt: 'Expires At',
    todayExpires: '(expires today)',
    daysLeft: '({days} days)',
    usedQuota: 'Used Quota',
    resetNow: 'Resetting soon',
    subscriptionType: 'Subscription Type',
    subscriptionExpires: 'Subscription Expires',
    // Usage stat cells
    todayRequests: 'Today Requests',
    todayInputTokens: 'Today Input',
    todayOutputTokens: 'Today Output',
    todayTokens: 'Today Tokens',
    todayCacheCreation: 'Today Cache Creation',
    todayCacheRead: 'Today Cache Read',
    todayCost: 'Today Cost',
    rpmTpm: 'RPM / TPM',
    totalRequests: 'Total Requests',
    totalInputTokens: 'Total Input',
    totalOutputTokens: 'Total Output',
    totalTokensLabel: 'Total Tokens',
    totalCacheCreation: 'Total Cache Creation',
    totalCacheRead: 'Total Cache Read',
    totalCost: 'Total Cost',
    avgDuration: 'Avg Duration',
    // Messages
    enterApiKey: 'Please enter an API Key',
    querySuccess: 'Query successful',
    queryFailed: 'Query failed',
    queryFailedRetry: 'Query failed, please try again later',
    noDailyUsage: 'No daily usage data',
  },

  // Setup Wizard
  setup: {
    title: '落雪API Setup',
    description: 'Configure your 落雪API instance',
    database: {
      title: 'Database Configuration',
      description: 'Connect to your PostgreSQL database',
      host: 'Host',
      port: 'Port',
      username: 'Username',
      password: 'Password',
      databaseName: 'Database Name',
      sslMode: 'SSL Mode',
      passwordPlaceholder: 'Password',
      ssl: {
        disable: 'Disable',
        require: 'Require',
        verifyCa: 'Verify CA',
        verifyFull: 'Verify Full'
      }
    },
    redis: {
      title: 'Redis Configuration',
      description: 'Connect to your Redis server',
      host: 'Host',
      port: 'Port',
      password: 'Password (optional)',
      database: 'Database',
      passwordPlaceholder: 'Password',
      enableTls: 'Enable TLS',
      enableTlsHint: 'Use TLS when connecting to Redis (public CA certs)'
    },
    admin: {
      title: 'Admin Account',
      description: 'Create your administrator account',
      email: 'Email',
      password: 'Password',
      confirmPassword: 'Confirm Password',
      passwordPlaceholder: 'Min 8 characters',
      confirmPasswordPlaceholder: 'Confirm password',
      passwordMismatch: 'Passwords do not match'
    },
    ready: {
      title: 'Ready to Install',
      description: 'Review your configuration and complete setup',
      database: 'Database',
      redis: 'Redis',
      adminEmail: 'Admin Email'
    },
    status: {
      testing: 'Testing...',
      success: 'Connection Successful',
      testConnection: 'Test Connection',
      installing: 'Installing...',
      completeInstallation: 'Complete Installation',
      completed: 'Installation completed!',
      redirecting: 'Redirecting to login page...',
      restarting: 'Service is restarting, please wait...',
      timeout: 'Service restart is taking longer than expected. Please refresh the page manually.'
    }
  },

  // Common
}
