-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 671-751.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'baoyu-url-to-markdown',
    'zh-CN',
    $zh243$宝玉 URL 转 Markdown$zh243$,
    $zh243$使用 baoyu-fetch CLI 抓取任意 URL 并转换为 Markdown，支持多类站点适配器以及登录、CAPTCHA 交互等待。$zh243$,
    $zh243$使用 baoyu-fetch CLI（通过 Chrome CDP 并配备站点专用适配器）抓取任意 URL 并转换为 Markdown。内置适用于 X/Twitter、YouTube 字幕、Hacker News 讨论串的适配器，并通过 Defuddle 处理通用网页。可借助交互等待模式处理登录和 CAPTCHA。适用于用户希望将网页保存为 Markdown 的场景。$zh243$
),
(
    'tavily-search',
    'zh-CN',
    $zh243$Tavily 网页搜索$zh243$,
    $zh243$通过 Tavily CLI 获取面向 LLM 优化的网页搜索结果，支持来源发现、最新资讯、域名过滤、时间范围和多种搜索深度。$zh243$,
    $zh243$通过 Tavily CLI 搜索网页并返回面向 LLM 优化的结果。适用于用户希望搜索网页、查找文章、查询信息、获取最新新闻、发现来源，或提出“搜索”“帮我找”“查询”“某主题最新情况”“查找相关文章”等需要互联网当前信息的请求。结果包含相关内容片段、相关性评分和元数据，并针对 LLM 使用进行了优化。支持域名过滤、时间范围和多种搜索深度。$zh243$
),
(
    'angular-developer',
    'zh-CN',
    $zh243$Angular 开发指南$zh243$,
    $zh243$生成 Angular 代码并提供架构指导，涵盖项目、组件、服务、HTTP 通信、响应式、表单、路由、SSR、无障碍、测试与 CLI 工具。$zh243$,
    $zh243$生成 Angular 代码并提供架构指导。适用于创建项目、组件、服务或 HTTP 通信，也适用于获取响应式机制（signals、linkedSignal、resource、httpResource）、表单、依赖注入、路由、SSR、无障碍（ARIA）、动画、样式（组件样式、Tailwind CSS）、测试或 CLI 工具的最佳实践。$zh243$
),
(
    'gws-gmail-forward',
    'zh-CN',
    $zh243$Gmail 邮件转发$zh243$,
    $zh243$将一封 Gmail 邮件转发给新的收件人。$zh243$,
    $zh243$将一封 Gmail 邮件转发给新的收件人。$zh243$
),
(
    'expo-module',
    'zh-CN',
    $zh243$Expo 原生模块开发$zh243$,
    $zh243$指导使用 Expo Modules API 与 Swift、Kotlin、TypeScript 创建 Expo 原生模块和视图。$zh243$,
    $zh243$开源框架指南，介绍如何使用 Expo Modules API（Swift、Kotlin、TypeScript）创建和编写 Expo 原生模块与视图。涵盖模块定义 DSL、原生视图、共享对象、配置插件、生命周期钩子、自动链接和类型系统。适用于构建或修改 Expo 原生模块。不适用于将现有 Swift 模块从定义 DSL 迁移到 Expo Modules API 2.0 宏；该场景请使用 expo-experiments 插件中的 expo-migrate-module。$zh243$
),
(
    'gws-workflow',
    'zh-CN',
    $zh243$Google 跨服务工作流$zh243$,
    $zh243$使用 Google Workflow 构建跨服务的效率工作流。$zh243$,
    $zh243$使用 Google Workflow 构建跨服务的效率工作流。$zh243$
),
(
    'clerk-testing',
    'zh-CN',
    $zh243$Clerk 端到端测试$zh243$,
    $zh243$使用 Playwright 或 Cypress 为 Clerk 应用编写端到端认证流程测试。$zh243$,
    $zh243$为 Clerk 应用进行端到端测试。配合 Playwright 或 Cypress 测试认证流程。$zh243$
),
(
    'baoyu-comic',
    'zh-CN',
    $zh243$宝玉知识漫画创作$zh243$,
    $zh243$支持多种画风和语气的知识漫画创作工具，可设计详细分镜并批量生成原创教育漫画图像。$zh243$,
    $zh243$支持多种画风和语气的知识漫画创作工具。通过细致的分镜布局和支持批处理的图像生成，创作原创教育漫画。适用于用户要求创作“知识漫画”“教育漫画”“人物传记漫画”“教程漫画”或 Logicomix 风格漫画时。$zh243$
),
(
    'public-relations',
    'zh-CN',
    $zh243$公关与媒体传播$zh243$,
    $zh243$规划公关、赢得媒体报道、联络记者和制定媒体策略，包括寻找记者、故事提案、热点借势及回应媒体需求。$zh243$,
    $zh243$当用户需要公共关系、赢得媒体报道、新闻曝光、记者联络或媒体策略方面的帮助时使用（此处 PR 指公关，不是拉取请求）。用户提到公关、新闻稿、新闻报道、媒体联络、向记者提案、争取媒体收录、媒体名单、媒体资料包、新闻资料包、热点借势、HARO、Qwoted、Featured、Help A Reporter、记者需求、科技媒体、TechCrunch、赢得媒体、思想领导力曝光、署名评论、客座文章、媒体联系人或“如何获得媒体报道”时也使用。用于赢得媒体相关工作，包括寻找记者、提案故事、借势新闻热点和回应媒体请求。若要向创业公司、SaaS 或 AI 目录提交，请参阅 `directory-submissions`；产品发布请参阅 `launch`；社交媒体互动请参阅 `social`；向潜在客户发送陌生开发邮件请参阅 `cold-email`。$zh243$
),
(
    'organization-best-practices',
    'zh-CN',
    $zh243$组织管理最佳实践$zh243$,
    $zh243$使用 Better Auth 的 organization 插件配置多租户组织、成员、邀请、角色、权限、团队与 RBAC。$zh243$,
    $zh243$使用 Better Auth 的 organization 插件配置多租户组织，管理成员和邀请，定义自定义角色与权限，设置团队并实现 RBAC。适用于组织配置、团队管理、成员角色、访问控制或 Better Auth organization 插件相关需求。$zh243$
),
(
    'clerk-orgs',
    'zh-CN',
    $zh243$Clerk B2B 组织管理$zh243$,
    $zh243$使用 Clerk Organizations 构建支持组织切换、RBAC、验证域名和企业 SSO 的多租户 B2B SaaS 应用。$zh243$,
    $zh243$使用 Clerk Organizations 为 B2B SaaS 创建多租户应用，支持组织切换、基于角色的访问控制、验证域名和企业 SSO。适用于团队工作区、RBAC、基于组织的路由和成员管理。$zh243$
),
(
    'gws-chat',
    'zh-CN',
    $zh243$Google Chat 空间与消息管理$zh243$,
    $zh243$管理 Google Chat 空间和消息。$zh243$,
    $zh243$管理 Google Chat 空间和消息。$zh243$
),
(
    'excalidraw-diagram-generator',
    'zh-CN',
    $zh243$Excalidraw 图表生成器$zh243$,
    $zh243$根据自然语言生成 Excalidraw 流程图、关系图、思维导图和系统架构图，并输出可直接打开的 `.excalidraw` JSON 文件。$zh243$,
    $zh243$根据自然语言描述生成 Excalidraw 图表。适用于用户要求“创建图表”“制作流程图”“可视化流程”“绘制系统架构”“创建思维导图”或“生成 Excalidraw 文件”时。支持流程图、关系图、思维导图和系统架构图。输出可在 Excalidraw 中直接打开的 `.excalidraw` JSON 文件。$zh243$
),
(
    'gws-people',
    'zh-CN',
    $zh243$Google 联系人与个人资料管理$zh243$,
    $zh243$管理 Google 联系人和个人资料。$zh243$,
    $zh243$管理 Google 联系人和个人资料。$zh243$
),
(
    'gws-gmail-reply-all',
    'zh-CN',
    $zh243$Gmail 全员回复$zh243$,
    $zh243$对 Gmail 邮件执行全员回复，并自动处理会话串。$zh243$,
    $zh243$对 Gmail 邮件执行全员回复，并自动处理会话串。$zh243$
),
(
    'flutter-fix-layout-issues',
    'zh-CN',
    $zh243$Flutter 布局问题修复$zh243$,
    $zh243$使用 Dart 和 Flutter MCP 工具修复溢出、无界约束等 Flutter 布局错误。$zh243$,
    $zh243$使用 Dart 和 Flutter MCP 工具修复 Flutter 布局错误，例如溢出和无界约束。适用于处理 `RenderFlex overflowed`、`Vertical viewport was given unbounded height` 或类似布局问题。$zh243$
),
(
    'gws-workflow-email-to-task',
    'zh-CN',
    $zh243$Gmail 邮件转 Google Tasks$zh243$,
    $zh243$通过 Google Workflow 将 Gmail 邮件转换为 Google Tasks 任务。$zh243$,
    $zh243$通过 Google Workflow 将 Gmail 邮件转换为 Google Tasks 任务。$zh243$
),
(
    'recipe-create-presentation',
    'zh-CN',
    $zh243$Google Slides 演示文稿创建$zh243$,
    $zh243$新建 Google Slides 演示文稿并添加初始幻灯片。$zh243$,
    $zh243$新建 Google Slides 演示文稿并添加初始幻灯片。$zh243$
),
(
    'baoyu-post-to-x',
    'zh-CN',
    $zh243$宝玉 X 内容发布$zh243$,
    $zh243$向 X（Twitter）发布普通图文、视频帖子及长篇 Markdown 文章，并按可用环境选择安全的 Chrome 操作方式。$zh243$,
    $zh243$向 X（Twitter）发布内容和文章。支持带图片或视频的普通帖子，以及使用长篇 Markdown 的 X Articles。在 Codex 中，如果用户明确要求使用 Codex Chrome 插件或 @chrome，应采用 Chrome Extension 工作流；否则在可用时使用 Chrome Computer Use，仅在获准时才回退到真实 Chrome CDP 脚本。适用于用户要求“发布到 X”“发推文”“发布到 Twitter”或“分享到 X”时。$zh243$
),
(
    'mmx-cli',
    'zh-CN',
    $zh243$MiniMax mmx 命令行工具$zh243$,
    $zh243$通过 mmx 和 MiniMax AI 平台生成文本、图像、视频、语音与音乐，并支持模型对话、网页搜索和 API 资源管理。$zh243$,
    $zh243$使用 mmx 通过 MiniMax AI 平台生成文本、图像、视频、语音和音乐。适用于用户希望创作媒体内容、与 MiniMax 模型对话、执行网页搜索，或从终端管理 MiniMax API 资源时。$zh243$
),
(
    'nextjs-app-router-patterns',
    'zh-CN',
    $zh243$Next.js App Router 模式$zh243$,
    $zh243$掌握 Next.js 14+ App Router、Server Components、流式渲染、并行路由和高级数据获取模式。$zh243$,
    $zh243$掌握 Next.js 14+ App Router，包括 Server Components、流式渲染、并行路由和高级数据获取。适用于构建 Next.js 应用、实现 SSR/SSG 或优化 React Server Components。$zh243$
),
(
    'layout',
    'zh-CN',
    $zh243$布局与视觉节奏优化$zh243$,
    $zh243$改进布局、间距和视觉节奏，修复单调网格、间距不一致、层级薄弱、拥挤及对齐问题。$zh243$,
    $zh243$改进布局、间距和视觉节奏。修复单调的网格、间距不一致和视觉层级薄弱等问题。适用于用户觉得布局不协调，提到间距、视觉层级、界面拥挤、对齐问题，或希望改善整体构图时。$zh243$
),
(
    'flutter-add-widget-test',
    'zh-CN',
    $zh243$Flutter Widget 测试$zh243$,
    $zh243$使用 `WidgetTester` 编写组件级测试，验证界面渲染以及点击、滚动和文本输入等交互。$zh243$,
    $zh243$使用 `WidgetTester` 实现组件级测试，以验证界面渲染和用户交互，例如点击、滚动和输入文本。适用于验证特定 Widget 是否显示正确数据并按预期响应事件。$zh243$
),
(
    'momentic-mobile-test',
    'zh-CN',
    $zh243$Momentic 移动端测试$zh243$,
    $zh243$创建、运行和维护 Android、iOS 的 Momentic 移动端端到端测试与模块，并支持真机验证和高置信度 v2 YAML 修改。$zh243$,
    $zh243$为 Android 和 iOS 创建、运行并维护 Momentic 移动端端到端测试与模块。使用 Momentic MCP 工具进行实时设备验证；只有在对本地移动端 v2 变更有较高把握时，才直接编辑 v2 YAML。$zh243$
),
(
    'gws-workflow-meeting-prep',
    'zh-CN',
    $zh243$Google 会议准备工作流$zh243$,
    $zh243$通过 Google Workflow 准备下一场会议的议程、参会者和关联文档。$zh243$,
    $zh243$通过 Google Workflow 准备下一场会议的议程、参会者和关联文档。$zh243$
),
(
    'recipe-draft-email-from-doc',
    'zh-CN',
    $zh243$Google Docs 转 Gmail 草稿$zh243$,
    $zh243$读取 Google Docs 内容并用作 Gmail 邮件正文。$zh243$,
    $zh243$读取 Google Docs 内容并用作 Gmail 邮件正文。$zh243$
),
(
    'gws-chat-send',
    'zh-CN',
    $zh243$Google Chat 消息发送$zh243$,
    $zh243$向 Google Chat 空间发送消息。$zh243$,
    $zh243$向 Google Chat 空间发送消息。$zh243$
),
(
    'higgsfield-game-generation',
    'zh-CN',
    $zh243$Higgsfield 游戏生成$zh243$,
    $zh243$使用 Higgsfield CLI 构建浏览器游戏，并生成精灵图、纹理、动画 3D 资产、音乐、音效和语音等游戏资源。$zh243$,
    $zh243$使用 Higgsfield CLI 构建并迭代可玩的浏览器游戏，或创建游戏专用精灵图、纹理、带动画的 3D 资产、音乐、音效和语音。适用于“制作游戏”“构建浏览器游戏”“创建游戏资源”“制作精灵图集”“生成可平铺纹理”“为 3D 游戏角色制作动画”或“部署、发布游戏”等请求。支持单人、本地多人和在线多人游戏。不适用于普通图像或视频生成、游戏预告片、原生移动端或桌面端构建，也不适用于在没有源文件的情况下编辑游戏。$zh243$
),
(
    'code-review-excellence',
    'zh-CN',
    $zh243$卓越代码审查$zh243$,
    $zh243$通过建设性反馈、早期缺陷发现和知识共享提升代码审查质量，同时维护团队士气。$zh243$,
    $zh243$掌握高效的代码审查实践，在保持团队士气的同时提供建设性反馈、尽早发现缺陷并促进知识共享。适用于审查拉取请求、建立审查标准或指导开发者。$zh243$
),
(
    'recipe-email-drive-link',
    'zh-CN',
    $zh243$Google Drive 链接邮件分享$zh243$,
    $zh243$共享 Google Drive 文件，并通过邮件向收件人发送链接和附言。$zh243$,
    $zh243$共享 Google Drive 文件，并通过邮件向收件人发送链接和附言。$zh243$
),
(
    'insforge-debug',
    'zh-CN',
    $zh243$InsForge 问题诊断$zh243$,
    $zh243$诊断 InsForge 项目的 SDK、HTTP、网关、边缘函数、认证、RLS、实时通道、查询和部署故障，并支持安全、性能与上线准备审计。$zh243$,
    $zh243$用于诊断 InsForge 项目中的问题，包括响应式故障（SDK 错误对象、HTTP 4xx/5xx、网关 502/503/504 超时、边缘函数失败或超时、登录/OAuth/认证错误、RLS 拒绝、实时通道问题、单个端点查询缓慢、边缘函数或 Vercel 部署失败），以及主动审计（安全与 RLS 审查、性能与索引审查、系统健康检查、上线前准备度评估）。当用户遇到错误却不知道从何查起时也适用。$zh243$
),
(
    'persona-project-manager',
    'zh-CN',
    $zh243$项目经理工作助手$zh243$,
    $zh243$协调项目，包括跟踪任务、安排会议和共享文档。$zh243$,
    $zh243$协调项目，包括跟踪任务、安排会议和共享文档。$zh243$
),
(
    'gws-workflow-weekly-digest',
    'zh-CN',
    $zh243$Google 每周摘要工作流$zh243$,
    $zh243$通过 Google Workflow 汇总本周会议和未读邮件数量。$zh243$,
    $zh243$通过 Google Workflow 生成每周摘要，汇总本周会议和未读邮件数量。$zh243$
),
(
    'baoyu-compress-image',
    'zh-CN',
    $zh243$宝玉图片压缩$zh243$,
    $zh243$自动选择工具将图片压缩为 WebP（默认）或 PNG，用于图片优化、格式转换和减小文件体积。$zh243$,
    $zh243$自动选择合适工具，将图片压缩为 WebP（默认）或 PNG。适用于用户要求“压缩图片”“优化图片”“转换为 WebP”或减小图片文件体积时。$zh243$
),
(
    'gws-workflow-standup-report',
    'zh-CN',
    $zh243$Google 站会摘要工作流$zh243$,
    $zh243$通过 Google Workflow 将今日会议和未完成任务整理为站会摘要。$zh243$,
    $zh243$通过 Google Workflow 将今日会议和未完成任务整理为站会摘要。$zh243$
),
(
    'baoyu-danger-x-to-markdown',
    'zh-CN',
    $zh243$宝玉 X 转 Markdown（非官方 API）$zh243$,
    $zh243$将 X（Twitter）推文和文章转换为带 YAML front matter 的 Markdown；使用需经用户同意的逆向 API。$zh243$,
    $zh243$将 X（Twitter）推文和文章转换为带 YAML front matter 的 Markdown。该能力使用逆向分析所得的 API，必须先获得用户同意。适用于用户提到“X 转 Markdown”“推文转 Markdown”“保存推文”，或提供 x.com、twitter.com URL 要求转换时。$zh243$
),
(
    'flutter-add-integration-test',
    'zh-CN',
    $zh243$Flutter 集成测试$zh243$,
    $zh243$配置 Flutter Driver，将 MCP 操作转化为使用 `integration_test` 包的持久化集成测试。$zh243$,
    $zh243$配置 Flutter Driver 以操作应用，并将 MCP 操作转换为永久集成测试。适用于为项目添加集成测试、通过 MCP 探索界面组件，或使用 `integration_test` 包自动化用户流程。$zh243$
),
(
    'gws-events',
    'zh-CN',
    $zh243$Google Workspace 事件订阅$zh243$,
    $zh243$订阅 Google Workspace 事件。$zh243$,
    $zh243$订阅 Google Workspace 事件。$zh243$
),
(
    'docker-expert',
    'zh-CN',
    $zh243$Docker 专家$zh243$,
    $zh243$提供容器优化、安全加固、多阶段构建、编排模式和生产部署策略方面的高级 Docker 专业指导。$zh243$,
    $zh243$高级 Docker 容器化专家，依据当前行业最佳实践，提供全面而实用的容器优化、安全加固、多阶段构建、编排模式和生产部署策略知识。$zh243$
),
(
    'recipe-create-gmail-filter',
    'zh-CN',
    $zh243$Gmail 自动过滤器创建$zh243$,
    $zh243$创建 Gmail 过滤器，自动为收到的邮件添加标签、星标或分类。$zh243$,
    $zh243$创建 Gmail 过滤器，自动为收到的邮件添加标签、星标或分类。$zh243$
),
(
    'baoyu-danger-gemini-web',
    'zh-CN',
    $zh243$宝玉 Gemini Web 生成（非官方 API）$zh243$,
    $zh243$通过逆向分析的 Gemini Web API 生成文本和图像，支持参考图像、视觉输入与多轮对话。$zh243$,
    $zh243$通过逆向分析所得的 Gemini Web API 生成图像和文本。支持文本生成、根据提示词生成图像、将参考图像作为视觉输入，以及多轮对话。适用于其他技能需要图像生成后端，或用户要求“使用 Gemini 生成图像”“进行 Gemini 文本生成”，或需要具备视觉能力的 AI 生成时。$zh243$
),
(
    'baoyu-format-markdown',
    'zh-CN',
    $zh243$宝玉 Markdown 格式化$zh243$,
    $zh243$为纯文本或 Markdown 文件添加 front matter、标题、摘要、层级标题、粗体、列表和代码块，并优化文章版式。$zh243$,
    $zh243$为纯文本或 Markdown 文件添加 front matter、标题、摘要、层级标题、粗体、列表和代码块等格式。适用于用户要求“格式化 Markdown”“美化文章”“添加格式”或改进文章版式时。输出文件名为 `{filename}-formatted.md`。$zh243$
),
(
    'api-design-principles',
    'zh-CN',
    $zh243$API 设计原则$zh243$,
    $zh243$掌握 REST 与 GraphQL API 设计原则，构建直观、可扩展且易维护的开发者友好型 API。$zh243$,
    $zh243$掌握 REST 和 GraphQL API 设计原则，构建直观、可扩展、易维护且令开发者满意的 API。适用于设计新 API、审查 API 规范或建立 API 设计标准。$zh243$
),
(
    'code-review-and-quality',
    'zh-CN',
    $zh243$多维代码审查与质量评估$zh243$,
    $zh243$从多个维度开展代码审查，在任何变更合并到主分支前系统评估其代码质量。$zh243$,
    $zh243$开展多维度代码审查。任何变更合并前都应使用。适用于审查自己、其他 Agent 或人工编写的代码，也适用于代码进入主分支前需要从多个维度评估质量时。$zh243$
),
(
    'cloudflare-one',
    'zh-CN',
    $zh243$Cloudflare One 零信任指南$zh243$,
    $zh243$指导 Cloudflare One 的 Zero Trust 与 SASE 设计、配置、排障和审查，覆盖 Access、Gateway、WARP、Tunnel、DLP、CASB 等能力。$zh243$,
    $zh243$指导 Cloudflare One 在 Access、Gateway、WARP、Tunnel、Cloudflare WAN、DLP、CASB、设备状态和身份等方面的 Zero Trust 与 SASE 工作。适用于设计、配置、排查或审查 Cloudflare One 部署。遵循检索优先原则：使用当前 Cloudflare 文档和 API schema，不依赖内嵌产品文档。$zh243$
),
(
    'flutter-setup-declarative-routing',
    'zh-CN',
    $zh243$Flutter 声明式路由配置$zh243$,
    $zh243$使用 `go_router` 等包配置 `MaterialApp.router`，实现基于 URL 的高级导航、深层链接和浏览器历史支持。$zh243$,
    $zh243$使用 `go_router` 等包配置 `MaterialApp.router`，实现基于 URL 的高级导航。适用于开发需要特定深层链接和浏览器历史支持的 Web 应用或移动应用。$zh243$
),
(
    'offers',
    'zh-CN',
    $zh243$高转化销售方案设计$zh243$,
    $zh243$设计和优化真正售卖的销售方案，涵盖价值呈现、赠品组合、保障、稀缺与紧迫感、命名和付款结构。$zh243$,
    $zh243$当用户希望设计、构建或改进真正对外销售的方案时使用，涵盖价值呈现、赠品组合、保障机制设计、稀缺性与紧迫感、命名和付款结构。用户提到销售方案、方案设计、构建方案、重磅方案、不可抗拒的方案、价值组合、赠品组合、保障、风险逆转、退款保障、稀缺性、紧迫感、高客单价方案、服务产品化、方案命名、付款计划、降级销售、追加销售方案，或“为什么我的方案没有转化”时也适用。最适合服务、代理机构、课程、教练、信息产品、高客单价 B2B 和直接响应营销。若经营纯自助式 SaaS，请先阅读 `pricing`，因为层级和套餐设计更关键。若要确定价格水平本身，包括层级、免费增值和价值指标，请参阅 `pricing`；用于展示方案的页面请参阅 `copywriting`；发布时刻请参阅 `launch`；销售资料请参阅 `sales-enablement`。$zh243$
),
(
    'persona-researcher',
    'zh-CN',
    $zh243$研究资料管理助手$zh243$,
    $zh243$组织研究工作，管理参考资料、笔记和协作。$zh243$,
    $zh243$组织研究工作，管理参考资料、笔记和协作。$zh243$
),
(
    'recipe-save-email-attachments',
    'zh-CN',
    $zh243$Gmail 附件保存到 Google Drive$zh243$,
    $zh243$查找含附件的 Gmail 邮件，并将附件保存到 Google Drive 文件夹。$zh243$,
    $zh243$查找含附件的 Gmail 邮件，并将附件保存到 Google Drive 文件夹。$zh243$
),
(
    'backlink-analyzer',
    'zh-CN',
    $zh243$反向链接分析器$zh243$,
    $zh243$已导入反向链接分析技能 backlink-analyzer。$zh243$,
    $zh243$已导入反向链接分析技能 backlink-analyzer。$zh243$
),
(
    'two-factor-authentication-best-practices',
    'zh-CN',
    $zh243$双因素认证最佳实践$zh243$,
    $zh243$使用 Better Auth 的 twoFactor 插件配置 TOTP、邮件或短信 OTP、备用代码、受信任设备和 2FA 登录流程。$zh243$,
    $zh243$使用 Better Auth 的 twoFactor 插件配置 TOTP 身份验证器应用，通过邮件或 SMS 发送 OTP 验证码，管理备用代码，处理受信任设备，并实现 2FA 登录流程。适用于用户需要 MFA、多因素认证、身份验证器设置，或通过 Better Auth 加强登录安全时。$zh243$
),
(
    'recipe-backup-sheet-as-csv',
    'zh-CN',
    $zh243$Google Sheets CSV 备份$zh243$,
    $zh243$将 Google Sheets 电子表格导出为 CSV 文件，以便本地备份或处理。$zh243$,
    $zh243$将 Google Sheets 电子表格导出为 CSV 文件，以便本地备份或处理。$zh243$
),
(
    'recipe-organize-drive-folder',
    'zh-CN',
    $zh243$Google Drive 文件夹整理$zh243$,
    $zh243$创建 Google Drive 文件夹结构，并将文件移动到正确位置。$zh243$,
    $zh243$创建 Google Drive 文件夹结构，并将文件移动到正确位置。$zh243$
),
(
    'persona-exec-assistant',
    'zh-CN',
    $zh243$高管行政助理$zh243$,
    $zh243$管理高管的日程、收件箱和沟通事务。$zh243$,
    $zh243$管理高管的日程、收件箱和沟通事务。$zh243$
),
(
    'recipe-find-free-time',
    'zh-CN',
    $zh243$Google Calendar 空闲时间查找$zh243$,
    $zh243$查询多位用户的 Google Calendar 忙闲状态，以寻找合适的会议时段。$zh243$,
    $zh243$查询多位用户的 Google Calendar 忙闲状态，以寻找合适的会议时段。$zh243$
),
(
    'cloudflare-one-migrations',
    'zh-CN',
    $zh243$Cloudflare One 迁移规划$zh243$,
    $zh243$规划从 Zscaler、Palo Alto、传统 VPN、SWG 或 SASE 技术栈迁移到 Cloudflare One。$zh243$,
    $zh243$规划从 Zscaler ZIA/ZPA、Palo Alto、传统 VPN、SWG 或 SASE 技术栈迁移到 Cloudflare One。适用于迁移评估、策略映射、推广计划以及功能对等性与差距分析。$zh243$
),
(
    'recipe-compare-sheet-tabs',
    'zh-CN',
    $zh243$Google Sheets 标签页对比$zh243$,
    $zh243$读取同一 Google Sheet 中两个标签页的数据，进行比较并识别差异。$zh243$,
    $zh243$读取同一 Google Sheet 中两个标签页的数据，进行比较并识别差异。$zh243$
),
(
    'insforge-integrations',
    'zh-CN',
    $zh243$InsForge 外部认证与支付集成$zh243$,
    $zh243$将外部认证提供商接入 InsForge 以实现基于 JWT 的 RLS，或集成 OKX x402 链上按次付费。$zh243$,
    $zh243$适用于将外部认证提供商（Clerk、Auth0、WorkOS、Kinde、Stytch、Better Auth）接入 InsForge，以实现基于 JWT 的 RLS；也适用于添加 OKX x402 支付服务，为链上按次使用计费提供支持。$zh243$
),
(
    'recipe-sync-contacts-to-sheet',
    'zh-CN',
    $zh243$Google 联系人同步到 Sheets$zh243$,
    $zh243$将 Google Contacts 通讯录导出到 Google Sheets 电子表格。$zh243$,
    $zh243$将 Google Contacts 通讯录导出到 Google Sheets 电子表格。$zh243$
),
(
    'recipe-create-doc-from-template',
    'zh-CN',
    $zh243$Google Docs 模板文档创建$zh243$,
    $zh243$复制 Google Docs 模板，填充内容并与协作者共享。$zh243$,
    $zh243$复制 Google Docs 模板，填充内容并与协作者共享。$zh243$
),
(
    'recipe-plan-weekly-schedule',
    'zh-CN',
    $zh243$Google 每周日程规划$zh243$,
    $zh243$查看一周的 Google Calendar 日程，识别空档并添加事件加以安排。$zh243$,
    $zh243$查看一周的 Google Calendar 日程，识别空档并添加事件加以安排。$zh243$
),
(
    'gws-workflow-file-announce',
    'zh-CN',
    $zh243$Google Drive 文件公告工作流$zh243$,
    $zh243$通过 Google Workflow 在 Chat 空间中公告 Drive 文件。$zh243$,
    $zh243$通过 Google Workflow 在 Chat 空间中公告 Drive 文件。$zh243$
),
(
    'recipe-bulk-download-folder',
    'zh-CN',
    $zh243$Google Drive 文件夹批量下载$zh243$,
    $zh243$列出并下载 Google Drive 文件夹中的全部文件。$zh243$,
    $zh243$列出并下载 Google Drive 文件夹中的全部文件。$zh243$
),
(
    'gws-events-renew',
    'zh-CN',
    $zh243$Google Workspace 事件订阅续期$zh243$,
    $zh243$续订或重新激活 Google Workspace Events 订阅。$zh243$,
    $zh243$续订或重新激活 Google Workspace Events 订阅。$zh243$
),
(
    'gws-events-subscribe',
    'zh-CN',
    $zh243$Google Workspace 事件流订阅$zh243$,
    $zh243$订阅 Google Workspace 事件，并以 NDJSON 流式输出。$zh243$,
    $zh243$订阅 Google Workspace 事件，并以 NDJSON 流式输出。$zh243$
),
(
    'persona-content-creator',
    'zh-CN',
    $zh243$Workspace 内容创作助手$zh243$,
    $zh243$在 Google Workspace 中创建、组织和分发内容。$zh243$,
    $zh243$在 Google Workspace 中创建、组织和分发内容。$zh243$
),
(
    'flutter-implement-json-serialization',
    'zh-CN',
    $zh243$Flutter JSON 序列化$zh243$,
    $zh243$使用 `dart:convert` 创建含 `fromJson` 和 `toJson` 方法的模型类，手动映射简单数据结构。$zh243$,
    $zh243$使用 `dart:convert` 创建带有 `fromJson` 和 `toJson` 方法的模型类。适用于为简单数据结构手动将 JSON 键映射到类属性。$zh243$
),
(
    'gws-admin-reports',
    'zh-CN',
    $zh243$Google Workspace 审计与用量报告$zh243$,
    $zh243$通过 Google Workspace Admin SDK 获取审计日志和用量报告。$zh243$,
    $zh243$通过 Google Workspace Admin SDK 获取审计日志和用量报告。$zh243$
),
(
    'recipe-generate-report-from-sheet',
    'zh-CN',
    $zh243$Google Sheets 数据报告生成$zh243$,
    $zh243$读取 Google Sheet 数据并创建格式化的 Google Docs 报告。$zh243$,
    $zh243$读取 Google Sheet 数据并创建格式化的 Google Docs 报告。$zh243$
),
(
    'flutter-setup-localization',
    'zh-CN',
    $zh243$Flutter 本地化配置$zh243$,
    $zh243$添加 `flutter_localizations`、`intl` 依赖并配置 `pubspec.yaml` 与 `l10n.yaml`，为 Flutter 项目启用本地化。$zh243$,
    $zh243$添加 `flutter_localizations` 和 `intl` 依赖，在 `pubspec.yaml` 中启用 `generate: true`，并创建 `l10n.yaml` 配置文件。适用于为新的 Flutter 项目初始化本地化支持。$zh243$
),
(
    'recipe-reschedule-meeting',
    'zh-CN',
    $zh243$Google Calendar 会议改期$zh243$,
    $zh243$将 Google Calendar 事件移到新时间，并自动通知所有参会者。$zh243$,
    $zh243$将 Google Calendar 事件移到新时间，并自动通知所有参会者。$zh243$
),
(
    'recipe-label-and-archive-emails',
    'zh-CN',
    $zh243$Gmail 邮件加标签并归档$zh243$,
    $zh243$为匹配的 Gmail 邮件添加标签并归档，以保持收件箱整洁。$zh243$,
    $zh243$为匹配的 Gmail 邮件添加标签并归档，以保持收件箱整洁。$zh243$
),
(
    'flutter-add-widget-preview',
    'zh-CN',
    $zh243$Flutter Widget 交互式预览$zh243$,
    $zh243$使用 previews.dart 系统为项目添加交互式 Widget 预览，辅助界面设计一致性和交互测试。$zh243$,
    $zh243$使用 previews.dart 系统为项目添加交互式 Widget 预览。适用于创建新界面组件或更新现有页面，以确保设计一致并支持交互测试。$zh243$
),
(
    'recipe-save-email-to-doc',
    'zh-CN',
    $zh243$Gmail 邮件保存到 Google Docs$zh243$,
    $zh243$将 Gmail 邮件正文保存到 Google Docs 文档，以便归档或参考。$zh243$,
    $zh243$将 Gmail 邮件正文保存到 Google Docs 文档，以便归档或参考。$zh243$
),
(
    'recipe-watch-drive-changes',
    'zh-CN',
    $zh243$Google Drive 变更订阅$zh243$,
    $zh243$订阅 Google Drive 文件或文件夹的变更通知。$zh243$,
    $zh243$订阅 Google Drive 文件或文件夹的变更通知。$zh243$
),
(
    'recipe-block-focus-time',
    'zh-CN',
    $zh243$Google Calendar 专注时间安排$zh243$,
    $zh243$在 Google Calendar 中创建周期性专注时间段，保护深度工作时间。$zh243$,
    $zh243$在 Google Calendar 中创建周期性专注时间段，保护深度工作时间。$zh243$
),
(
    'persona-team-lead',
    'zh-CN',
    $zh243$团队负责人工作助手$zh243$,
    $zh243$带领团队开展站会、协调任务并保持沟通。$zh243$,
    $zh243$带领团队开展站会、协调任务并保持沟通。$zh243$
),
(
    'recipe-create-shared-drive',
    'zh-CN',
    $zh243$Google 共享云端硬盘创建$zh243$,
    $zh243$创建 Google 共享云端硬盘，并为成员分配适当角色。$zh243$,
    $zh243$创建 Google 共享云端硬盘，并为成员分配适当角色。$zh243$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
