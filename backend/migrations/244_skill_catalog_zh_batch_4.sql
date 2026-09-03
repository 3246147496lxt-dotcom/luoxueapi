-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 752-836.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'stitch-design',
    'zh-CN',
    $zh244$Stitch 设计工作流$zh244$,
    $zh244$Stitch 设计工作的统一入口，支持提示词增强、设计系统整合以及通过 Stitch MCP 生成和编辑高保真界面。$zh244$,
    $zh244$Stitch 设计工作的统一入口。负责提示词增强（UI/UX 关键词、氛围）、设计系统整合（`.stitch/DESIGN.md`），以及通过 Stitch MCP 生成和编辑高保真界面。$zh244$
),
(
    'recipe-create-task-list',
    'zh-CN',
    $zh244$Google Tasks 清单创建$zh244$,
    $zh244$新建 Google Tasks 清单并添加初始任务。$zh244$,
    $zh244$新建 Google Tasks 清单并添加初始任务。$zh244$
),
(
    'recipe-share-doc-and-notify',
    'zh-CN',
    $zh244$Google Docs 共享与通知$zh244$,
    $zh244$以编辑权限共享 Google Docs 文档，并通过邮件向协作者发送链接。$zh244$,
    $zh244$以编辑权限共享 Google Docs 文档，并通过邮件向协作者发送链接。$zh244$
),
(
    'recipe-schedule-recurring-event',
    'zh-CN',
    $zh244$Google Calendar 周期事件安排$zh244$,
    $zh244$创建包含参会者的周期性 Google Calendar 事件。$zh244$,
    $zh244$创建包含参会者的周期性 Google Calendar 事件。$zh244$
),
(
    'recipe-find-large-files',
    'zh-CN',
    $zh244$Google Drive 大文件查找$zh244$,
    $zh244$识别占用存储配额较多的 Google Drive 大文件。$zh244$,
    $zh244$识别占用存储配额较多的 Google Drive 大文件。$zh244$
),
(
    'herdr',
    'zh-CN',
    $zh244$Herdr 终端多路复用控制$zh244$,
    $zh244$控制面向编码 Agent 的 Herdr 终端多路复用器，仅在用户明确指定 Herdr 时用于检查或控制窗格、标签页、工作区、命令或其他 Agent。$zh244$,
    $zh244$控制面向编码 Agent 的终端多路复用器 Herdr。仅当用户明确提到 Herdr，或要求使用 Herdr 检查或控制窗格、标签页、工作区、命令或其他 Agent 时使用。不能仅因为任务可能受益于后台终端、委派或并行工作就使用。要求设置 `HERDR_ENV=1`。$zh244$
),
(
    'recipe-copy-sheet-for-new-month',
    'zh-CN',
    $zh244$Google Sheets 月度模板复制$zh244$,
    $zh244$复制 Google Sheets 模板标签页，用于新一个月的数据跟踪。$zh244$,
    $zh244$复制 Google Sheets 模板标签页，用于新一个月的数据跟踪。$zh244$
),
(
    'recipe-review-overdue-tasks',
    'zh-CN',
    $zh244$Google Tasks 逾期任务检查$zh244$,
    $zh244$查找已经逾期且需要处理的 Google Tasks 任务。$zh244$,
    $zh244$查找已经逾期且需要处理的 Google Tasks 任务。$zh244$
),
(
    'recipe-batch-invite-to-event',
    'zh-CN',
    $zh244$Google Calendar 批量邀请$zh244$,
    $zh244$将一组参会者添加到现有 Google Calendar 事件并发送通知。$zh244$,
    $zh244$将一组参会者添加到现有 Google Calendar 事件并发送通知。$zh244$
),
(
    'recipe-collect-form-responses',
    'zh-CN',
    $zh244$Google Forms 回应收集$zh244$,
    $zh244$获取并审查 Google Forms 表单回应。$zh244$,
    $zh244$获取并审查 Google Forms 表单回应。$zh244$
),
(
    'recipe-create-events-from-sheet',
    'zh-CN',
    $zh244$从 Google Sheets 创建日历事件$zh244$,
    $zh244$从 Google Sheets 读取事件数据，并为每一行创建对应的 Google Calendar 事件。$zh244$,
    $zh244$从 Google Sheets 电子表格读取事件数据，并为每一行创建对应的 Google Calendar 事件。$zh244$
),
(
    'flutter-use-http-package',
    'zh-CN',
    $zh244$Flutter HTTP 请求$zh244$,
    $zh244$使用 `http` 包执行 GET、POST、PUT 或 DELETE 请求，从 REST API 获取数据或向其发送数据。$zh244$,
    $zh244$使用 `http` 包执行 GET、POST、PUT 或 DELETE 请求。适用于需要从 REST API 获取数据或向其发送数据时。$zh244$
),
(
    'persona-it-admin',
    'zh-CN',
    $zh244$IT 管理助手$zh244$,
    $zh244$执行 IT 管理，包括安全监控和 Google Workspace 配置。$zh244$,
    $zh244$执行 IT 管理，包括安全监控和 Google Workspace 配置。$zh244$
),
(
    'recipe-share-folder-with-team',
    'zh-CN',
    $zh244$Google Drive 团队文件夹共享$zh244$,
    $zh244$与指定协作者共享 Google Drive 文件夹及其中全部内容。$zh244$,
    $zh244$与指定协作者共享 Google Drive 文件夹及其中全部内容。$zh244$
),
(
    'recipe-forward-labeled-emails',
    'zh-CN',
    $zh244$Gmail 标签邮件转发$zh244$,
    $zh244$查找带有指定标签的 Gmail 邮件，并将其转发到另一个地址。$zh244$,
    $zh244$查找带有指定标签的 Gmail 邮件，并将其转发到另一个地址。$zh244$
),
(
    'persona-event-coordinator',
    'zh-CN',
    $zh244$活动协调助手$zh244$,
    $zh244$规划和管理活动，包括日程安排、邀请和后勤协调。$zh244$,
    $zh244$规划和管理活动，包括日程安排、邀请和后勤协调。$zh244$
),
(
    'persona-customer-support',
    'zh-CN',
    $zh244$客户支持助手$zh244$,
    $zh244$管理客户支持，包括跟踪工单、回复客户和升级问题。$zh244$,
    $zh244$管理客户支持，包括跟踪工单、回复客户和升级问题。$zh244$
),
(
    'persona-sales-ops',
    'zh-CN',
    $zh244$销售运营助手$zh244$,
    $zh244$管理销售工作流，包括跟踪商机、安排通话和客户沟通。$zh244$,
    $zh244$管理销售工作流，包括跟踪商机、安排通话和客户沟通。$zh244$
),
(
    'recipe-create-meet-space',
    'zh-CN',
    $zh244$Google Meet 会议空间创建$zh244$,
    $zh244$创建 Google Meet 会议空间并共享加入链接。$zh244$,
    $zh244$创建 Google Meet 会议空间并共享加入链接。$zh244$
),
(
    'recipe-create-vacation-responder',
    'zh-CN',
    $zh244$Gmail 休假自动回复$zh244$,
    $zh244$启用 Gmail 外出自动回复，并设置自定义消息和日期范围。$zh244$,
    $zh244$启用 Gmail 外出自动回复，并设置自定义消息和日期范围。$zh244$
),
(
    'recipe-create-expense-tracker',
    'zh-CN',
    $zh244$Google Sheets 费用追踪表$zh244$,
    $zh244$创建用于跟踪费用的 Google Sheets 电子表格，包含表头和初始记录。$zh244$,
    $zh244$创建用于跟踪费用的 Google Sheets 电子表格，包含表头和初始记录。$zh244$
),
(
    'recipe-create-feedback-form',
    'zh-CN',
    $zh244$Google Forms 反馈表创建$zh244$,
    $zh244$创建 Google Forms 反馈表并通过 Gmail 分享。$zh244$,
    $zh244$创建 Google Forms 反馈表并通过 Gmail 分享。$zh244$
),
(
    'mcp-apps-builder',
    'zh-CN',
    $zh244$MCP Apps 构建器$zh244$,
    $zh244$使用 mcp-use v2 构建、修改、调试、迁移或审查 TypeScript MCP 服务器和交互式 MCP Apps。$zh244$,
    $zh244$使用 mcp-use v2 构建、修改、调试、迁移或审查 TypeScript MCP 服务器和交互式 MCP Apps。适用于 mcp-use 项目中的工具、资源、提示词、Views、React 宿主交互、OAuth、中间件、Inspector 工作流、包边界迁移和发布就绪验证。$zh244$
),
(
    'recipe-review-meet-participants',
    'zh-CN',
    $zh244$Google Meet 参会记录审查$zh244$,
    $zh244$查看 Google Meet 会议的参会人员及各自参会时长。$zh244$,
    $zh244$查看 Google Meet 会议的参会人员及各自参会时长。$zh244$
),
(
    'gws-modelarmor',
    'zh-CN',
    $zh244$Google Model Armor 内容安全过滤$zh244$,
    $zh244$使用 Google Model Armor 对用户生成内容进行安全过滤。$zh244$,
    $zh244$使用 Google Model Armor 对用户生成内容进行安全过滤。$zh244$
),
(
    'recipe-post-mortem-setup',
    'zh-CN',
    $zh244$Google 事故复盘安排$zh244$,
    $zh244$创建 Google Docs 事故复盘文档，安排 Google Calendar 评审，并通过 Chat 通知。$zh244$,
    $zh244$创建 Google Docs 事故复盘文档，安排 Google Calendar 评审，并通过 Chat 通知。$zh244$
),
(
    'persona-hr-coordinator',
    'zh-CN',
    $zh244$人力资源协调助手$zh244$,
    $zh244$处理人力资源工作流，包括入职、公告和员工沟通。$zh244$,
    $zh244$处理人力资源工作流，包括入职、公告和员工沟通。$zh244$
),
(
    'gws-modelarmor-create-template',
    'zh-CN',
    $zh244$Google Model Armor 模板创建$zh244$,
    $zh244$创建新的 Google Model Armor 模板。$zh244$,
    $zh244$创建新的 Google Model Armor 模板。$zh244$
),
(
    'recipe-share-event-materials',
    'zh-CN',
    $zh244$Google Calendar 活动资料共享$zh244$,
    $zh244$与 Google Calendar 事件的所有参会者共享 Google Drive 文件。$zh244$,
    $zh244$与 Google Calendar 事件的所有参会者共享 Google Drive 文件。$zh244$
),
(
    'recipe-send-team-announcement',
    'zh-CN',
    $zh244$Google 团队公告发送$zh244$,
    $zh244$同时通过 Gmail 和 Google Chat 空间发送团队公告。$zh244$,
    $zh244$同时通过 Gmail 和 Google Chat 空间发送团队公告。$zh244$
),
(
    'nestjs-best-practices',
    'zh-CN',
    $zh244$NestJS 最佳实践$zh244$,
    $zh244$面向生产级应用的 NestJS 最佳实践和架构模式，涵盖模块、依赖注入、安全与性能。$zh244$,
    $zh244$用于构建生产级应用的 NestJS 最佳实践和架构模式。在编写、审查或重构 NestJS 代码时使用，以确保模块、依赖注入、安全和性能采用正确模式。$zh244$
),
(
    'documentation-writer',
    'zh-CN',
    $zh244$Diátaxis 文档写作专家$zh244$,
    $zh244$依据 Diátaxis 技术文档创作框架的原则和结构，编写高质量软件文档。$zh244$,
    $zh244$Diátaxis 文档专家。专业技术写作者，依据 Diátaxis 技术文档创作框架的原则和结构，专注于编写高质量软件文档。$zh244$
),
(
    'better-auth-security-best-practices',
    'zh-CN',
    $zh244$Better Auth 安全最佳实践$zh244$,
    $zh244$为 Better Auth 配置限流、密钥、CSRF 防护、可信来源、安全会话与 Cookie、OAuth 令牌加密、IP 跟踪和审计日志。$zh244$,
    $zh244$为 Better Auth 配置速率限制、管理认证密钥、设置 CSRF 防护、定义可信来源、保护会话和 Cookie、加密 OAuth 令牌、跟踪 IP 地址并实现审计日志。适用于保护认证配置、防止暴力破解攻击或加固 Better Auth 部署。$zh244$
),
(
    'recipe-log-deal-update',
    'zh-CN',
    $zh244$Google Sheets 商机状态记录$zh244$,
    $zh244$将商机状态更新追加到 Google Sheets 销售跟踪表。$zh244$,
    $zh244$将商机状态更新追加到 Google Sheets 销售跟踪表。$zh244$
),
(
    'gws-modelarmor-sanitize-prompt',
    'zh-CN',
    $zh244$Google Model Armor 提示词净化$zh244$,
    $zh244$通过 Google Model Armor 模板净化用户提示词。$zh244$,
    $zh244$通过 Google Model Armor 模板净化用户提示词。$zh244$
),
(
    'gws-modelarmor-sanitize-response',
    'zh-CN',
    $zh244$Google Model Armor 响应净化$zh244$,
    $zh244$通过 Google Model Armor 模板净化模型响应。$zh244$,
    $zh244$通过 Google Model Armor 模板净化模型响应。$zh244$
),
(
    'gws-classroom',
    'zh-CN',
    $zh244$Google Classroom 管理$zh244$,
    $zh244$管理 Google Classroom 课程、花名册和课程作业。$zh244$,
    $zh244$管理 Google Classroom 课程、花名册和课程作业。$zh244$
),
(
    'recipe-create-classroom-course',
    'zh-CN',
    $zh244$Google Classroom 课程创建$zh244$,
    $zh244$创建 Google Classroom 课程并邀请学生。$zh244$,
    $zh244$创建 Google Classroom 课程并邀请学生。$zh244$
),
(
    'wind-find-finance-skill',
    'zh-CN',
    $zh244$万得金融技能发现$zh244$,
    $zh244$万得金融能力发现与安装入口，面向金融数据、行情、市场分析、选股、复盘、仓位、交易计划和回测等任务。$zh244$,
    $zh244$万得金融能力发现与安装入口。适用于金融数据、行情查询、市场分析、今日大盘、市场主线、板块轮动、资金流向、估值、选股、复盘、仓位、交易计划、回测等任务。必须先读取目录，判断必需的数据技能或工作流技能；缺失时必须展示安装选项，用户确认后由 AI 直接安装，不得只提供命令或以通用分析替代。$zh244$
),
(
    'release-skills',
    'zh-CN',
    $zh244$宝玉通用发布工作流$zh244$,
    $zh244$自动检测版本文件和变更日志，支持 Node.js、Python、Rust、Claude Plugin、GitHub Releases、标签及通用项目的发布。$zh244$,
    $zh244$通用发布工作流。自动检测版本文件和变更日志。支持 Node.js、Python、Rust、Claude Plugin、GitHub Releases、注解标签、历史发布回填和通用项目。适用于用户提出发布、新版本、升级版本、推送、发布说明、GitHub Release 或回填 Release 时。$zh244$
),
(
    'frontend-ui-engineering',
    'zh-CN',
    $zh244$生产级前端 UI 工程$zh244$,
    $zh244$构建生产级、无障碍且响应式的用户界面，涵盖页面、组件、布局、WCAG、状态管理和高质量视觉呈现。$zh244$,
    $zh244$构建生产级、无障碍且响应式的用户界面。适用于构建或修改界面与页面、创建组件、实现布局、满足 WCAG 无障碍要求、管理状态，或输出需要呈现生产级品质而非 AI 生成感时。$zh244$
),
(
    'baoyu-translate',
    'zh-CN',
    $zh244$宝玉翻译工具$zh244$,
    $zh244$提供快速、常规和精细三种翻译模式，支持自定义术语表、文章和文件翻译、双语转换、本地化与译文校对。$zh244$,
    $zh244$适用于用户要求翻译、精细翻译、翻译文章、译为中文、译为英文、改成中文或英文、中文转换、本地化、精修译文、校对译文、快速翻译，或提供 URL、文件并表达翻译意图时。支持快速、常规、精细三种模式，并可使用自定义术语表。$zh244$
),
(
    'interface-design',
    'zh-CN',
    $zh244$产品界面设计$zh244$,
    $zh244$以工艺品质为先，为仪表盘、管理面板、SaaS、工具、设置页、数据界面和交互式产品设计一致的产品 UI。$zh244$,
    $zh244$以工艺品质为先的界面设计，面向仪表盘、管理面板、SaaS 应用、工具、设置页、数据界面和交互式产品。适用于设计、构建、审查、审计或优化产品 UI，尤其关注视觉品质、布局层级、设计令牌、状态、视觉方向或设计系统一致性时。不适用于营销页面、落地页、营销活动或纯品牌工作。$zh244$
),
(
    'postgresql-table-design',
    'zh-CN',
    $zh244$PostgreSQL 表结构设计$zh244$,
    $zh244$设计或审查 PostgreSQL 专用 schema，涵盖最佳实践、数据类型、索引、约束、性能模式和高级功能。$zh244$,
    $zh244$适用于设计或审查 PostgreSQL 专用 schema。涵盖最佳实践、数据类型、索引、约束、性能模式和高级功能。$zh244$
),
(
    'planning-and-task-breakdown',
    'zh-CN',
    $zh244$规划与任务拆解$zh244$,
    $zh244$将工作拆分为有序且可实施的任务，适用于明确需求后的执行规划、范围估算和可并行工作识别。$zh244$,
    $zh244$将工作拆分为有序任务。适用于已经有规范或明确需求，需要把工作拆成可实施任务时；也适用于任务大到难以开始、需要估算范围，或存在并行工作机会时。$zh244$
),
(
    'spec-driven-development',
    'zh-CN',
    $zh244$规范驱动开发$zh244$,
    $zh244$在编码前创建规范，帮助澄清新项目、功能或重大变更中模糊、不明确或仅停留在想法层面的需求。$zh244$,
    $zh244$在编码前创建规范。适用于启动尚无规范的新项目、新功能或重大变更，也适用于需求不清晰、有歧义或只有模糊想法时。$zh244$
),
(
    'eas-update-insights',
    'zh-CN',
    $zh244$EAS Update 运行状况分析$zh244$,
    $zh244$检查已发布 EAS Update 的崩溃率、安装与启动次数、独立用户、负载大小，以及各渠道内嵌版本与 OTA 用户分布。$zh244$,
    $zh244$EAS 付费服务。检查已发布 EAS Update 的运行状况，包括崩溃率、安装与启动次数、独立用户数、负载大小，以及每个渠道中内嵌版本用户与 OTA 用户的分布。适用于用户询问更新表现、推广是否健康、内嵌构建与 OTA 各有多少用户，或希望根据更新健康状况设置 CI 门禁时。$zh244$
),
(
    'fastapi-templates',
    'zh-CN',
    $zh244$FastAPI 项目模板$zh244$,
    $zh244$使用异步模式、依赖注入和完善的错误处理创建生产级 FastAPI 项目。$zh244$,
    $zh244$使用异步模式、依赖注入和完善的错误处理创建生产级 FastAPI 项目。适用于构建新的 FastAPI 应用或搭建后端 API 项目。$zh244$
),
(
    'summarize',
    'zh-CN',
    $zh244$内容摘要与转写$zh244$,
    $zh244$对 URL、YouTube 或其他视频、播客、文章、文字稿、PDF 和本地文件进行摘要或转写。$zh244$,
    $zh244$对 URL、YouTube 或其他视频、播客、文章、文字稿、PDF 和本地文件进行摘要或转写。$zh244$
),
(
    'review-loop',
    'zh-CN',
    $zh244$迭代评审闭环$zh244$,
    $zh244$通过工作者与评审者循环，让批评子 Agent 按 1–10 分评分并给出可执行反馈，持续修改直至达到质量门槛。$zh244$,
    $zh244$迭代式工作者与评审者循环：启动批评子 Agent，对成果按 1–10 分评分并提供可执行反馈，然后持续修改直到达到质量门槛。适用于功能实现、规范编写、现有代码审查，以及任何质量重于速度的任务。触发方式包括“使用 review-loop”“润色这个”“继续迭代”“`/review-loop`”或“用反馈闭环评审”。$zh244$
),
(
    'security-and-hardening',
    'zh-CN',
    $zh244$安全与加固$zh244$,
    $zh244$针对漏洞加固代码，适用于用户输入、认证、数据存储、外部集成、不可信数据、用户会话和第三方服务交互。$zh244$,
    $zh244$针对漏洞加固代码。适用于处理用户输入、认证、数据存储或外部集成，也适用于构建任何接收不可信数据、管理用户会话或与第三方服务交互的功能。$zh244$
),
(
    'performance-optimization',
    'zh-CN',
    $zh244$性能优化$zh244$,
    $zh244$优化前端、后端、查询和数据库性能，处理性能回退、Core Web Vitals、加载时间、N+1 查询和性能分析发现的瓶颈。$zh244$,
    $zh244$优化应用在前端、后端、查询和数据库层面的性能。适用于存在性能要求、怀疑发生性能回退、需要改善 Core Web Vitals 或加载时间、需要修复 N+1 查询模式，或性能分析发现瓶颈时。$zh244$
),
(
    'prd',
    'zh-CN',
    $zh244$产品需求文档生成$zh244$,
    $zh244$为软件系统和 AI 功能生成高质量产品需求文档（PRD），包括执行摘要、用户故事、技术规格和风险分析。$zh244$,
    $zh244$为软件系统和 AI 驱动功能生成高质量产品需求文档（PRD）。内容包括执行摘要、用户故事、技术规格和风险分析。$zh244$
),
(
    'documentation-and-adrs',
    'zh-CN',
    $zh244$文档与架构决策记录$zh244$,
    $zh244$记录文档和决策，适用于架构决策、公共 API 变更、功能发布，以及为未来工程师和 Agent 留存代码库背景。$zh244$,
    $zh244$记录决策和文档。适用于作出架构决策、修改公共 API、发布功能，或需要记录未来工程师和 Agent 理解代码库所需背景时。$zh244$
),
(
    'clerk',
    'zh-CN',
    $zh244$Clerk 认证路由$zh244$,
    $zh244$根据 Clerk 相关任务自动路由到具体技能，涵盖 CLI、认证接入、自定义登录、跨框架模式、组织、计费、用户同步与测试。$zh244$,
    $zh244$Clerk 认证路由。适用于用户询问 Clerk CLI 操作、添加认证、设置 Clerk、自定义登录流程、Swift 或原生 iOS 认证、原生 Android 认证、Next.js、React、Vue、Nuxt、Astro、TanStack Start、Expo、React Router 或 Chrome Extension 模式，也适用于组织、计费、订阅、支付、定价、套餐、按席位定价、功能权益、用户同步、测试、模拟用户以及本地测试 Webhook。根据用户任务自动路由到相应的具体技能。$zh244$
),
(
    'gh-cli',
    'zh-CN',
    $zh244$GitHub CLI 综合参考$zh244$,
    $zh244$GitHub CLI（gh）综合参考，覆盖仓库、议题、拉取请求、Actions、项目、发布、Gist、Codespaces、组织和扩展等命令行操作。$zh244$,
    $zh244$GitHub CLI（gh）综合参考，涵盖从命令行执行的仓库、议题、拉取请求、Actions、项目、发布、Gist、Codespaces、组织、扩展及其他 GitHub 操作。$zh244$
),
(
    'niche-signal-discovery',
    'zh-CN',
    $zh244$细分市场信号发现$zh244$,
    $zh244$发现区分 Closed Won 与 Closed Lost 客户的细分第一方信号，用于 ICP 分析、账户评分和潜客筛选。$zh244$,
    $zh244$发现能够区分 Closed Won（已赢单）与 Closed Lost（已丢单）账户的细分第一方信号，用于 ICP 分析。适用于用户提供赢单和丢单客户域名列表，希望从网站内容、招聘信息、技术栈、成熟度标志等方面找到差异信号，以构建账户评分模型和潜客筛选标准。触发场景包括 ICP 分析、细分信号、赢单与丢单分析、差异信号、信号发现、ICP 信号报告、账户评分信号、线索评分、第一方信号和买家信号。阅读本文件前，必须先阅读 `deepline-gtm`，了解 Deepline CLI 工具及其用法；随后再阅读本文件获取任务指导。$zh244$
),
(
    'vue-best-practices-52cd7883ed',
    'zh-CN',
    $zh244$Vue.js 最佳实践$zh244$,
    $zh244$Vue.js 任务必须使用的最佳实践，推荐以 Composition API、`<script setup>` 和 TypeScript 为标准方案。$zh244$,
    $zh244$处理 Vue.js 任务时必须使用。强烈建议将 Composition API、`<script setup>` 和 TypeScript 作为标准方案。涵盖 Vue 3、SSR、Volar 和 vue-tsc。任何涉及 Vue、`.vue` 文件、Vue Router、Pinia 或配合 Vue 使用 Vite 的工作都应加载。除非项目明确要求 Options API，否则始终使用 Composition API。$zh244$
),
(
    'incremental-implementation',
    'zh-CN',
    $zh244$增量式实现$zh244$,
    $zh244$以增量方式交付跨多个文件的功能或变更，避免一次编写过多代码，并将大型任务拆成可逐步落地的修改。$zh244$,
    $zh244$以增量方式交付变更。适用于实现任何涉及多个文件的功能或修改，也适用于即将一次编写大量代码，或任务过大而难以一步落地时。$zh244$
),
(
    'debugging-and-error-recovery',
    'zh-CN',
    $zh244$系统化调试与错误恢复$zh244$,
    $zh244$指导系统化根因调试，在测试失败、构建中断、行为不符预期或出现意外错误时定位并修复真正原因。$zh244$,
    $zh244$指导系统化的根因调试。适用于测试失败、构建中断、行为与预期不符或遇到任何意外错误时，也适用于需要系统化查找并修复根因而不是猜测时。$zh244$
),
(
    'code-simplification',
    'zh-CN',
    $zh244$代码简化$zh244$,
    $zh244$在不改变行为的前提下简化代码，提高可读性、可维护性和可扩展性，消除不必要的复杂度。$zh244$,
    $zh244$简化代码以提升清晰度。适用于在不改变行为的情况下重构代码，也适用于代码虽能工作但比应有状态更难阅读、维护或扩展，或审查已经积累不必要复杂度的代码时。$zh244$
),
(
    'build-tam',
    'zh-CN',
    $zh244$TAM 名单构建$zh244$,
    $zh244$从 Crustdata、Dropleads、PDL 等提供商获取账户和联系人，构建总体可服务市场（TAM）名单。$zh244$,
    $zh244$从 Crustdata、Dropleads、PDL 等提供商获取账户和联系人，构建总体可服务市场（TAM）名单。$zh244$
),
(
    'portfolio-prospecting',
    'zh-CN',
    $zh244$投资组合企业拓客$zh244$,
    $zh244$查找特定投资者或加速器支持的公司，再寻找联系人并开展个性化外联。$zh244$,
    $zh244$查找由特定投资者或加速器支持的公司，再寻找联系人并开展个性化外联。$zh244$
),
(
    'deepline-feedback',
    'zh-CN',
    $zh244$Deepline 反馈与缺陷报告$zh244$,
    $zh244$向 Deepline 团队发送产品反馈或缺陷报告，并附带会话记录和环境信息。$zh244$,
    $zh244$向 Deepline 团队发送反馈或缺陷报告，并附带会话记录和环境信息。适用于用户要求报告缺陷、发送产品反馈，或与 Deepline 支持团队共享当前 Claude/Cowork 会话时。$zh244$
),
(
    'clay-to-deepline',
    'zh-CN',
    $zh244$Clay 到 Deepline 转换$zh244$,
    $zh244$将 Clay 表配置转换为本地 Deepline 脚本，涵盖提取、文档、操作映射、脚本生成和对等性验证。$zh244$,
    $zh244$将 Clay 表配置转换为本地 Deepline 脚本。处理配置提取（MCP 或脚本）、文档编写、操作映射、脚本生成，以及相对于 Clay 基准事实的对等性验证。$zh244$
),
(
    'linkedin-url-lookup',
    'zh-CN',
    $zh244$LinkedIn 资料 URL 查找$zh244$,
    $zh244$根据姓名和公司解析 LinkedIn 个人资料 URL，并通过严格身份验证避免误匹配。$zh244$,
    $zh244$根据姓名和公司解析 LinkedIn 个人资料 URL，并通过严格身份验证避免误匹配。$zh244$
),
(
    'test-driven-development-807c312a98',
    'zh-CN',
    $zh244$测试驱动开发$zh244$,
    $zh244$以测试驱动逻辑实现、缺陷修复和行为变更，用可重复验证证明代码正确工作。$zh244$,
    $zh244$通过测试驱动开发。适用于实现任何逻辑、修复任何缺陷或改变任何行为，也适用于需要证明代码能够工作、收到缺陷报告，或即将修改现有功能时。$zh244$
),
(
    'web-quality-audit',
    'zh-CN',
    $zh244$Web 质量审计$zh244$,
    $zh244$全面审计 Web 性能、无障碍、SEO 和最佳实践，适用于站点审计、Lighthouse 检查、页面质量评估与网站优化。$zh244$,
    $zh244$全面的 Web 质量审计，涵盖性能、无障碍、SEO 和最佳实践。适用于用户要求“审计我的网站”“审查 Web 质量”“运行 Lighthouse 审计”“检查页面质量”或“优化我的网站”时。$zh244$
),
(
    'deepline-quickstart',
    'zh-CN',
    $zh244$Deepline 快速上手$zh244$,
    $zh244$运行 Deepline 快速演示流程，向用户展示 Deepline 的工作方式。$zh244$,
    $zh244$运行 Deepline 快速演示流程，向用户展示 Deepline 的工作方式。$zh244$
),
(
    'nuxt',
    'zh-CN',
    $zh244$Nuxt 全栈开发$zh244$,
    $zh244$使用具备 SSR、自动导入和文件路由的 Nuxt 全栈 Vue 框架开发应用、服务端路由、中间件和混合渲染。$zh244$,
    $zh244$Nuxt 是支持 SSR、自动导入和基于文件路由的全栈 Vue 框架。适用于处理 Nuxt 应用、服务端路由、useFetch、中间件或混合渲染。$zh244$
),
(
    'context-engineering',
    'zh-CN',
    $zh244$Agent 上下文工程$zh244$,
    $zh244$优化 Agent 上下文配置，适用于新会话、输出质量下降、任务切换，以及项目规则文件和上下文配置。$zh244$,
    $zh244$优化 Agent 上下文配置。适用于开始新会话、Agent 输出质量下降、在任务间切换，或需要为项目配置规则文件和上下文时。$zh244$
),
(
    'gws-script',
    'zh-CN',
    $zh244$Google Apps Script 项目管理$zh244$,
    $zh244$管理 Google Apps Script 项目。$zh244$,
    $zh244$管理 Google Apps Script 项目。$zh244$
),
(
    'mediabunny',
    'zh-CN',
    $zh244$Mediabunny 多媒体处理$zh244$,
    $zh244$使用 Mediabunny 库处理多媒体。$zh244$,
    $zh244$使用 Mediabunny 库处理多媒体。$zh244$
),
(
    'e2e-testing-patterns',
    'zh-CN',
    $zh244$端到端测试模式$zh244$,
    $zh244$掌握 Playwright 和 Cypress 端到端测试，构建可靠测试套件、发现缺陷并支持快速部署。$zh244$,
    $zh244$掌握使用 Playwright 和 Cypress 进行端到端测试，构建能够发现缺陷、提升信心并支持快速部署的可靠测试套件。适用于实现端到端测试、排查不稳定测试或建立测试标准。$zh244$
),
(
    'refactor',
    'zh-CN',
    $zh244$渐进式代码重构$zh244$,
    $zh244$在不改变行为的前提下进行精细重构，涵盖函数提取、变量重命名、拆分巨型函数、类型安全、代码异味和设计模式。$zh244$,
    $zh244$通过精细的代码重构，在不改变行为的前提下提升可维护性。涵盖提取函数、重命名变量、拆分巨型函数、改善类型安全、消除代码异味和应用设计模式。相比 repo-rebuilder 改动更温和，适合渐进式改进。$zh244$
),
(
    'multi-stage-dockerfile',
    'zh-CN',
    $zh244$多阶段 Dockerfile$zh244$,
    $zh244$为任意语言或框架创建经过优化的多阶段 Dockerfile。$zh244$,
    $zh244$为任意语言或框架创建经过优化的多阶段 Dockerfile。$zh244$
),
(
    'skill-vetter',
    'zh-CN',
    $zh244$OpenClaw 技能安全审查$zh244$,
    $zh244$安装 ClawHub、GitHub 或其他来源的 OpenClaw 技能前进行安全优先审查，检查风险信号、权限范围和可疑模式。$zh244$,
    $zh244$以安全为先审查 OpenClaw 技能。在安装来自 ClawHub、GitHub 或其他来源的任何技能前使用。检查风险信号、权限范围和可疑模式。$zh244$
),
(
    'gws-script-push',
    'zh-CN',
    $zh244$Google Apps Script 本地文件上传$zh244$,
    $zh244$将本地文件上传到 Google Apps Script 项目。$zh244$,
    $zh244$将本地文件上传到 Google Apps Script 项目。$zh244$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
