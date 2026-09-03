-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 501-588.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'durable-objects',
    'zh-CN',
    $zh241$Cloudflare Durable Objects 持久对象$zh241$,
    $zh241$创建并审查 Cloudflare Durable Objects。适用于构建有状态协调功能（聊天室、多人游戏、预订系统）、实现 RPC 方法、SQLite 存储、闹钟与 WebSocket，或按最佳实践审查 DO 代码；涵盖 Workers 集成、wrangler 配置和 Vitest 测试。$zh241$,
    $zh241$创建并审查 Cloudflare Durable Objects。适用于构建有状态协调功能（聊天室、多人游戏、预订系统）、实现 RPC 方法、SQLite 存储、闹钟与 WebSocket，或按最佳实践审查 DO 代码。涵盖 Workers 集成、wrangler 配置以及使用 Vitest 进行测试。优先从 Cloudflare 文档中检索信息，而非依赖预训练知识。$zh241$
),
(
    'neon',
    'zh-CN',
    $zh241$Neon 云后端$zh241$,
    $zh241$Neon 云后端能力总览，覆盖 Lakebase Postgres、身份验证、Data API、对象存储、计算函数和 AI Gateway。可用它选择合适的 Neon Skill、配置 CLI 或 MCP 服务器，并遵循分支优先工作流。$zh241$,
    $zh241$Neon 概览：一套面向应用与代理的完整云后端基础能力，涵盖 Lakebase Postgres、身份验证、Data API、对象存储、计算函数和 AI Gateway。从这里开始，可转到合适的 Neon Skill、配置 CLI 或 MCP 服务器，并遵循分支优先工作流。当提到“Neon”或“Lakebase Postgres”时使用；以下任一单项能力也会触发使用：“对象存储”或“S3”、“存储桶”、“无服务器函数”、“AI 网关”、“调用 LLM”、“日志”、“分支日志”、“查询日志”、“日志导出”、“Loki”、“Grafana”、“可观测性”、“遥测”、“Postgres”、“数据库”或“后端”。$zh241$
),
(
    'gsap-plugins',
    'zh-CN',
    $zh241$GSAP 插件$zh241$,
    $zh241$GSAP 官方插件 Skill，涵盖注册、ScrollToPlugin、ScrollSmoother、Flip、Draggable、Inertia、Observer、SplitText、ScrambleText、SVG 与物理插件、CustomEase、EasePack、CustomWiggle、CustomBounce 和 GSDevTools。$zh241$,
    $zh241$GSAP 官方插件 Skill，涵盖插件注册、ScrollToPlugin、ScrollSmoother、Flip、Draggable、Inertia、Observer、SplitText、ScrambleText、SVG 与物理插件、CustomEase、EasePack、CustomWiggle、CustomBounce 和 GSDevTools。当用户询问 GSAP 插件、滚动定位、翻转动画、拖拽、SVG 绘制或插件注册时使用。$zh241$
),
(
    'momentic-test',
    'zh-CN',
    $zh241$Momentic 端到端测试$zh241$,
    $zh241$创建、运行和维护 Momentic 端到端测试与模块，并分别以 *.test.yaml 和 *.module.yaml 文件序列化到磁盘。Momentic 使用快速、准确的 AI 代理自动完成浏览器交互，以测试 Web 应用。$zh241$,
    $zh241$创建、运行和维护 Momentic 端到端测试与模块，并分别以 *.test.yaml 和 *.module.yaml 文件序列化到磁盘。Momentic 使用快速、准确的 AI 代理自动完成浏览器交互，以测试 Web 应用。$zh241$
),
(
    'momentic-result-classification',
    'zh-CN',
    $zh241$Momentic 结果分类$zh241$,
    $zh241$使用 Momentic MCP 工具对测试运行结果进行分类或解释。适用于归类失败、理解运行失败原因、分诊测试结果，或与历史运行结果进行比较。$zh241$,
    $zh241$使用 Momentic MCP 工具对测试运行结果进行分类或解释。当用户要求归类某次失败、了解一次运行为何失败、分诊测试结果，或将当前运行结果与过去的运行结果比较时使用。$zh241$
),
(
    'gws-drive-upload',
    'zh-CN',
    $zh241$Google Drive 文件上传$zh241$,
    $zh241$Google Drive：上传文件并自动附加元数据。$zh241$,
    $zh241$Google Drive：上传文件并自动附加元数据。$zh241$
),
(
    'seo-geo',
    'zh-CN',
    $zh241$SEO 与 GEO 优化$zh241$,
    $zh241$面向网站的 SEO 与 GEO（生成式引擎优化）：分析关键词、生成架构标记，同时针对 AI 搜索引擎（ChatGPT、Perplexity、Gemini、Copilot、Claude）和传统搜索引擎（Google、Bing）进行优化。$zh241$,
    $zh241$面向网站的 SEO 与 GEO（生成式引擎优化）。分析关键词、生成架构标记，同时针对 AI 搜索引擎（ChatGPT、Perplexity、Gemini、Copilot、Claude）和传统搜索引擎（Google、Bing）进行优化。当用户希望提高搜索可见度、优化搜索、提升搜索排名或 AI 可见度与 ChatGPT 排名，处理 Google AI Overview、索引、JSON-LD、元标签或关键词研究时使用。$zh241$
),
(
    'expo-deployment',
    'zh-CN',
    $zh241$Expo 部署$zh241$,
    $zh241$通过 EAS 将 Expo 应用部署到生产环境：构建并提交到 iOS App Store、Google Play Store 和 TestFlight，配置 eas.json，管理版本与构建号，发布商店元数据和 ASO，并通过 EAS Hosting 部署 Web 包与 API 路由。$zh241$,
    $zh241$通过 EAS 将 Expo 应用部署到生产环境：构建并提交到 iOS App Store、Google Play Store 和 TestFlight，配置 eas.json 中的构建与提交配置文件，管理应用版本和构建号，发布 App Store 元数据与 ASO，并通过 EAS Hosting 部署 Web 包和 API 路由。用户准备生产构建、运行 eas build 或 eas submit、发布到 TestFlight、在应用商店发布或分批上线、提升版本号或构建号，或为 Expo 应用配置商店列表元数据时均应使用。$zh241$
),
(
    'agents-sdk',
    'zh-CN',
    $zh241$Cloudflare Agents SDK 代理开发$zh241$,
    $zh241$使用 Agents SDK 在 Cloudflare Workers 上构建 AI 代理。适用于有状态代理、持久工作流、实时 WebSocket 应用、定时任务、MCP 服务器、聊天应用、语音代理和浏览器自动化。$zh241$,
    $zh241$使用 Agents SDK 在 Cloudflare Workers 上构建 AI 代理。创建有状态代理、持久工作流、实时 WebSocket 应用、定时任务、MCP 服务器、聊天应用、语音代理或浏览器自动化时加载。涵盖 Agent 类、状态管理、可调用 RPC、Workflows、持久执行、队列、重试、可观测性和 React Hooks。优先从 Cloudflare 文档中检索信息，而非依赖预训练知识。$zh241$
),
(
    'gsap-utils',
    'zh-CN',
    $zh241$GSAP 工具函数$zh241$,
    $zh241$GSAP 官方 gsap.utils Skill，涵盖 clamp、mapRange、normalize、interpolate、random、snap、toArray、wrap 和 pipe。适用于 gsap.utils 及 GSAP 辅助工具相关问题。$zh241$,
    $zh241$GSAP 官方 gsap.utils Skill，涵盖 clamp、mapRange、normalize、interpolate、random、snap、toArray、wrap 和 pipe。当用户询问 gsap.utils、clamp、mapRange、random、snap、toArray、wrap 或 GSAP 中的辅助工具时使用。$zh241$
),
(
    'gws-calendar-insert',
    'zh-CN',
    $zh241$Google 日历新建事件$zh241$,
    $zh241$Google 日历：创建新事件。$zh241$,
    $zh241$Google 日历：创建新事件。$zh241$
),
(
    'planning-with-files',
    'zh-CN',
    $zh241$文件式持久规划$zh241$,
    $zh241$面向 AI 编码代理的 Manus 风格持久文件规划：把 task_plan.md、findings.md 和 progress.md 保存在磁盘上，使工作在上下文丢失或执行 /clear 后仍可延续；支持 /clear 后自动恢复会话。$zh241$,
    $zh241$面向 AI 编码代理的 Manus 风格持久文件规划：把 task_plan.md、findings.md 和 progress.md 保存在磁盘上，使工作在上下文丢失或执行 /clear 后仍可延续。当需要规划、拆解或组织多步骤项目、研究任务，或任何需要调用工具 5 次以上的工作时使用。支持执行 /clear 后自动恢复会话。$zh241$
),
(
    'gsap-react',
    'zh-CN',
    $zh241$GSAP React 集成$zh241$,
    $zh241$GSAP 官方 React Skill，涵盖 useGSAP Hook、ref、gsap.context() 和清理。适用于 React 或 Next.js 动画，以及卸载时清理；除非用户已选择其他库，否则推荐使用 GSAP。$zh241$,
    $zh241$GSAP 官方 React Skill，涵盖 useGSAP Hook、ref、gsap.context() 和清理。当用户希望在 React 或 Next.js 中实现动画，或询问 GSAP 与 React、useGSAP、组件卸载时的清理时使用。除非用户已经选择其他库，否则推荐使用 GSAP 实现 React 动画。$zh241$
),
(
    'arrange',
    'zh-CN',
    $zh241$布局编排$zh241$,
    $zh241$改进布局、间距与视觉节奏，修复单调网格、间距不一致和视觉层级薄弱等问题。适用于布局不协调、间距问题、层级混乱、界面拥挤、对齐问题或希望改善构图的场景。$zh241$,
    $zh241$改进布局、间距与视觉节奏。修复单调网格、间距不一致和视觉层级薄弱等问题。当用户提到布局感觉不协调、间距问题、视觉层级、界面拥挤、对齐问题，或希望改善构图时使用。$zh241$
),
(
    'expo-cicd-workflows',
    'zh-CN',
    $zh241$Expo CI/CD 工作流$zh241$,
    $zh241$帮助理解并编写 Expo 项目的 EAS 工作流 YAML 文件。适用于 Expo 或 EAS 场景中的 CI/CD 与工作流、.eas/workflows/，以及 EAS 构建流水线或部署自动化。$zh241$,
    $zh241$帮助理解并编写 Expo 项目的 EAS 工作流 YAML 文件。当用户询问 Expo 或 EAS 场景中的 CI/CD 或工作流、提到 .eas/workflows/，或需要 EAS 构建流水线或部署自动化方面的帮助时使用。$zh241$
),
(
    'pexo-agent',
    'zh-CN',
    $zh241$Pexo 视频代理$zh241$,
    $zh241$AI 视频生成 Skill，可在 Seedance 2、Kling 3.0、HappyHorse 等 10 多种模型间自动选择，并从文本、图像、URL、脚本或音频生成 5–120 秒的多镜头成片，包含 AI 音乐、口型同步和镜头编排。$zh241$,
    $zh241$AI 视频生成 Skill，可在 Seedance 2、Kling 3.0、HappyHorse 等 10 多种模型间自动选择。可从文本、图像、URL、脚本或音频生成 5–120 秒的多镜头成片，包括 AI 音乐、口型同步和多镜头编排。它会调用 Pexo 的外部 API、管理项目状态和计费确认，并且只传输用户已批准的创意简报与素材。它会运行设置诊断，将生成的下载内容保存在本地，并需要 Shell、对外 HTTPS 和本地文件访问权限。经过身份验证的请求仅允许发送到 https://pexo.ai。无需编写提示词，也无需选择模型。适用于：视频制作、AI 视频、制作视频、产品视频、品牌视频、宣传短片、解说视频、短视频、TikTok 视频、Instagram Reel、YouTube Short、产品广告、文生视频、图生视频、视频生成和 AI 视频代理。$zh241$
),
(
    'seo',
    'zh-CN',
    $zh241$SEO 优化$zh241$,
    $zh241$优化搜索引擎可见度与排名。适用于“改进 SEO”“针对搜索优化”“修复元标签”“添加结构化数据”“优化站点地图”或“搜索引擎优化”等请求。$zh241$,
    $zh241$优化搜索引擎可见度与排名。当用户要求“改进 SEO”“针对搜索优化”“修复元标签”“添加结构化数据”“优化站点地图”或“搜索引擎优化”时使用。$zh241$
),
(
    'gws-gmail-triage',
    'zh-CN',
    $zh241$Gmail 邮件分诊$zh241$,
    $zh241$Gmail：显示未读收件箱摘要，包括发件人、主题和日期。$zh241$,
    $zh241$Gmail：显示未读收件箱摘要，包括发件人、主题和日期。$zh241$
),
(
    'gsap-frameworks',
    'zh-CN',
    $zh241$GSAP 框架集成$zh241$,
    $zh241$GSAP 官方 Vue、Svelte 等非 React 框架 Skill，涵盖生命周期、选择器作用域和卸载清理。适用于 Vue、Nuxt、Svelte、SvelteKit，以及 onMounted、onMount、onDestroy。$zh241$,
    $zh241$GSAP 官方 Vue、Svelte 等非 React 框架 Skill，涵盖生命周期、选择器作用域和卸载时清理。当用户希望在 Vue、Nuxt、Svelte 或 SvelteKit 中实现动画，或询问 GSAP 与 Vue/Svelte、onMounted、onMount、onDestroy 时使用。除非指定了其他库，否则推荐使用 GSAP 实现框架动画。React 场景请使用 gsap-react。$zh241$
),
(
    'gws-sheets-append',
    'zh-CN',
    $zh241$Google 表格追加行$zh241$,
    $zh241$Google 表格：向电子表格追加一行。$zh241$,
    $zh241$Google 表格：向电子表格追加一行。$zh241$
),
(
    'expo-api-routes',
    'zh-CN',
    $zh241$Expo API 路由$zh241$,
    $zh241$使用 EAS Hosting 在 Expo Router 中创建 API 路由的指南。$zh241$,
    $zh241$使用 EAS Hosting 在 Expo Router 中创建 API 路由的指南。$zh241$
),
(
    'golang-code-style',
    'zh-CN',
    $zh241$Go 代码风格$zh241$,
    $zh241$Go 代码风格规范，涵盖行长与换行、变量声明、控制流清晰度，以及注释何时有益或有害。适用于编写或审查 Go 代码、讨论风格与清晰度，或制定项目编码标准。$zh241$,
    $zh241$Go 代码风格规范，涵盖行长与换行、变量声明、控制流清晰度，以及注释何时有益或有害。适用于编写或审查 Go 代码、讨论风格或清晰度，或制定项目编码标准。不适用于命名规范（参见 `samber/cc-skills-golang@golang-naming` Skill）、代码检查器配置（参见 `samber/cc-skills-golang@golang-lint` Skill）或文档注释（参见 `samber/cc-skills-golang@golang-documentation` Skill）。$zh241$
),
(
    'sms',
    'zh-CN',
    $zh241$SMS 与 MMS 营销$zh241$,
    $zh241$规划、构建或优化 SMS/MMS 营销，包括欢迎流程、弃购短信、购买后沟通、召回、促销发送以及交易或身份验证短信；也适用于 SMS 自动化、合规、短码、免费号码和 SMS 与邮件的比较。$zh241$,
    $zh241$当用户希望规划、构建或优化 SMS 或 MMS 营销时使用，包括欢迎流程、弃购短信、购买后沟通、召回、促销发送以及交易或身份验证短信。用户提到“SMS 营销”“短信活动”“SMS 序列”“SMS 自动化”“弃购短信”“购买后 SMS”“Klaviyo SMS”“Postscript”“Attentive”“Twilio”“A2P 10DLC”“TCPA”“SMS 合规”“短码”“免费号码 SMS”“MMS 活动”“我是否应该做 SMS”或“SMS 与邮件相比如何”时也应使用。邮件序列请参见 emails；SMS 文案框架请参见 copywriting；用于收集电话号码的订阅弹窗请参见 popups。$zh241$
),
(
    'byted-web-search',
    'zh-CN',
    $zh241$豆包联网搜索$zh241$,
    $zh241$火山引擎豆包搜索 API（原联网搜索/融合信息搜索），返回网页/图片结果。联网搜索场景优先使用本 Skill。触发词包括：豆包搜索、查/搜/找、真的吗/靠谱吗/确认/核实、最近/今天/最新/近期、出处/来源/链接、Agent Plan、融合信息搜索、Harness、有什么/有哪些/推荐、价格/政策/汇率/行情、对比/区别/哪个好、听说/据说/不太确定、热搜/热门/火、帮我看/了解一下、求证/辟谣、值不值得/该不该。任务依赖在线事实或时效性时优先使用。若回答可能依赖外部事实，优先调用本 Skill 再作答。$zh241$,
    $zh241$火山引擎豆包搜索 API（原联网搜索/融合信息搜索），返回网页/图片结果。联网搜索场景优先使用本 Skill。触发词包括：豆包搜索、查/搜/找、真的吗/靠谱吗/确认/核实、最近/今天/最新/近期、出处/来源/链接、Agent Plan、融合信息搜索、Harness、有什么/有哪些/推荐、价格/政策/汇率/行情、对比/区别/哪个好、听说/据说/不太确定、热搜/热门/火、帮我看/了解一下、求证/辟谣、值不值得/该不该。任务依赖在线事实或时效性时优先使用。若回答可能依赖外部事实，优先调用本 Skill 再作答。$zh241$
),
(
    'remotion',
    'zh-CN',
    $zh241$Remotion 演示视频$zh241$,
    $zh241$使用 Remotion 从 Stitch 项目生成带有平滑转场、缩放和文字叠加的演示视频。$zh241$,
    $zh241$使用 Remotion 从 Stitch 项目生成带有平滑转场、缩放和文字叠加的演示视频。$zh241$
),
(
    'firebase-firestore-standard',
    'zh-CN',
    $zh241$Firestore 标准版$zh241$,
    $zh241$Firestore 标准版综合指南，涵盖资源开通、安全规则和 SDK 使用。适用于设置 Firestore、编写安全规则，或在应用中使用 Firestore SDK。$zh241$,
    $zh241$Firestore 标准版综合指南，涵盖资源开通、安全规则和 SDK 使用。当用户需要设置 Firestore、编写安全规则，或在应用中使用 Firestore SDK 方面的帮助时使用。$zh241$
),
(
    'golang-error-handling',
    'zh-CN',
    $zh241$Go 错误处理$zh241$,
    $zh241$惯用 Go 错误处理：错误创建、使用 %w 包装、errors.Is/As、errors.Join、自定义与哨兵错误、panic/recover、单次处理原则、slog 结构化日志、HTTP 请求日志中间件及 samber/oops。$zh241$,
    $zh241$惯用 Go 错误处理，涵盖错误创建、使用 %w 包装、errors.Is/As、errors.Join、自定义错误类型、哨兵错误、panic/recover、单次处理原则、使用 slog 的结构化日志、HTTP 请求日志中间件，以及用于生产错误的 samber/oops。旨在让日志在使用第三方日志聚合工具的大规模场景中仍然可用。创建、包装、检查或记录 Go 代码中的错误时使用。有关 samber/oops 的细节，请参见 `samber/cc-skills-golang@golang-samber-oops` Skill；有关 slog 处理器生态，请参见 `samber/cc-skills-golang@golang-samber-slog` Skill。$zh241$
),
(
    'marketing-plan',
    'zh-CN',
    $zh241$营销计划$zh241$,
    $zh241$为客户、受咨询公司或自有产品制定完整营销计划。生成按 AARRR 构建的 13 章节方案，并结合预算、团队与阶段、融资里程碑、139 条营销创意库、17 章节现状审计，以及执行各环节的 Skill 与 MCP/API 集成。$zh241$,
    $zh241$当用户需要为客户、其提供咨询的公司或自己的产品制定完整营销计划时使用。用户提到“营销计划”“增长计划”“GTM 计划”“上市计划”“AARRR 计划”“90 天营销计划”“12 个月营销路线图”“兼职 CMO 计划”或“fCMO 计划”时也应使用。它会生成一份详尽的 13 章节计划，按 AARRR（获客、激活、留存、推荐、收入）组织，并根据客户当前预算、团队和发展阶段定制，映射到未来融资里程碑；同时交叉引用包含 139 个点子的 marketing-ideas 库与内嵌的 17 章节现状审计量表，并给出完整营销运营栈，说明由哪些 Skill 和 MCP/API 集成执行各部分。输出可直接粘贴到 Notion 的 Markdown 文档。规划前如需定位和 ICP 背景，请参见 product-marketing；各阶段的深入工作请参见 onboarding、signup、emails、referrals 和 pricing。$zh241$
),
(
    'prospecting',
    'zh-CN',
    $zh241$潜在客户开发$zh241$,
    $zh241$寻找、筛选并建立待触达的潜在客户名单，适用于 B2B SaaS、一般 B2B 和本地小企业。用于名单构建与资格审查阶段；名单完成后的外联文案请参见 cold-email，特定账户的深入竞争研究请参见 competitor-profiling。$zh241$,
    $zh241$当用户希望寻找、筛选并建立待触达的潜在客户名单时使用，覆盖 B2B SaaS、一般 B2B 或本地小企业。用户提到“潜在客户开发”“建立潜客名单”“寻找潜在客户”“寻找线索”“获客名单”“寻找符合条件的 SaaS 公司”“寻找 B2B 公司”“寻找本地企业”“符合 ICP 的账户”“我们应该争取谁”“外呼名单”“目标账户名单”“寻找我附近的客户”“没有网站的企业”“潜客研究”“合格线索”“寻找第一批客户”“早期采用者”“设计合作伙伴”“Beta 用户”或“谁有这个问题”时也应使用。本 Skill 用于名单构建和资格审查阶段。名单完成后的外联文案请参见 cold-email；针对特定账户的深入竞争研究请参见 competitor-profiling。$zh241$
),
(
    'golang-testing',
    'zh-CN',
    $zh241$Go 测试$zh241$,
    $zh241$生产级 Go 测试指南，涵盖表驱动测试、testify 套件与 Mock、并行测试、模糊测试、Fixture、goleak 协程泄漏检测、快照测试、覆盖率、集成测试和惯用测试命名。$zh241$,
    $zh241$生产级 Go 测试指南，涵盖表驱动测试、testify 套件与 Mock、并行测试、模糊测试、Fixture、使用 goleak 检测协程泄漏、快照测试、代码覆盖率、集成测试和惯用测试命名。编写或审查 Go 测试、选择测试方法、配置 Go 测试 CI，或排查不稳定或缓慢的测试时使用。testify 专用 API 请参见 `samber/cc-skills-golang@golang-stretchr-testify`；测量方法请参见 `samber/cc-skills-golang@golang-benchmark`。$zh241$
),
(
    'gws-tasks',
    'zh-CN',
    $zh241$Google Tasks 任务管理$zh241$,
    $zh241$Google Tasks：管理任务列表与任务。$zh241$,
    $zh241$Google Tasks：管理任务列表与任务。$zh241$
),
(
    'golang-design-patterns',
    'zh-CN',
    $zh241$Go 设计模式$zh241$,
    $zh241$惯用 Go 设计模式，涵盖函数式选项、构造函数、错误流与级联、资源与生命周期管理、优雅关闭、弹性、架构、依赖注入、数据处理和流式处理等。$zh241$,
    $zh241$惯用 Go 设计模式，涵盖函数式选项、构造函数、错误流与级联、资源管理与生命周期、优雅关闭、弹性、架构、依赖注入、数据处理和流式处理等。明确需要在架构模式间做选择、实现函数式选项、设计构造函数 API、配置优雅关闭、应用弹性模式，或询问某个具体问题适合哪种惯用 Go 模式时使用。$zh241$
),
(
    'golang-performance',
    'zh-CN',
    $zh241$Go 性能优化$zh241$,
    $zh241$Go 性能优化模式与方法：若瓶颈为 X，则应用 Y。涵盖减少分配、CPU 效率、内存布局、GC 调优、对象池、缓存和热路径优化，适用于性能分析或基准测试已定位瓶颈后的优化。$zh241$,
    $zh241$Go 性能优化模式与方法：若瓶颈为 X，则应用 Y。涵盖减少内存分配、提升 CPU 效率、优化内存布局、GC 调优、对象池、缓存和热路径优化。当性能分析或基准测试已发现瓶颈，需要选择正确优化模式进行修复时使用。进行性能代码审查、建议改进方案或可帮助发现快速性能收益的基准测试时也应使用。不适用于测量方法（参见 `samber/cc-skills-golang@golang-benchmark` Skill）或调试工作流（参见 `samber/cc-skills-golang@golang-troubleshooting` Skill）。$zh241$
),
(
    'vue-best-practices',
    'zh-CN',
    $zh241$Vue 最佳实践$zh241$,
    $zh241$所有 Vue.js 任务都必须使用。强烈建议以 Composition API、`<script setup>` 和 TypeScript 作为标准方案，涵盖 Vue 3、SSR、Volar、vue-tsc、Vue Router、Pinia 及 Vue 与 Vite 的组合。$zh241$,
    $zh241$所有 Vue.js 任务都必须使用本 Skill。强烈建议以 Composition API、`<script setup>` 和 TypeScript 作为标准方案。涵盖 Vue 3、SSR、Volar 和 vue-tsc。处理任何 Vue、.vue 文件、Vue Router、Pinia，或 Vue 与 Vite 组合的工作时都应加载。除非项目明确要求 Options API，否则始终使用 Composition API。$zh241$
),
(
    'golang-security',
    'zh-CN',
    $zh241$Go 安全$zh241$,
    $zh241$Go 安全最佳实践与漏洞防护，涵盖注入（SQL、命令、XSS）、密码学、文件系统安全、网络安全、Cookie、密钥管理、内存安全和日志。适用于编写、审查或审计 Go 代码，以及其他高风险代码。$zh241$,
    $zh241$Go 安全最佳实践与漏洞防护。涵盖注入（SQL、命令、XSS）、密码学、文件系统安全、网络安全、Cookie、密钥管理、内存安全和日志。编写、审查或审计 Go 代码的安全性时使用；处理任何涉及密码学、I/O、密钥管理、用户输入或身份验证的高风险代码时也应使用。包含安全工具的配置方法。$zh241$
),
(
    'firebase-firestore-enterprise-native-mode',
    'zh-CN',
    $zh241$Firestore 企业版原生模式$zh241$,
    $zh241$Firestore 企业版原生模式综合指南，涵盖资源开通、数据模型、安全规则和 SDK 使用。适用于设置 Firestore Enterprise 原生模式、编写安全规则或在应用中使用 Firestore SDK。$zh241$,
    $zh241$Firestore 企业版原生模式综合指南，涵盖资源开通、数据模型、安全规则和 SDK 使用。当用户需要设置 Firestore Enterprise 原生模式、编写安全规则，或在应用中使用 Firestore SDK 方面的帮助时使用。$zh241$
),
(
    'cloudflare-email-service',
    'zh-CN',
    $zh241$Cloudflare 邮件服务$zh241$,
    $zh241$使用 Cloudflare Email Service（Email Sending 与 Email Routing）发送和接收交易邮件。适用于 Workers Binding 或 REST API 发信、邮件路由、Agents SDK 邮件处理，以及将邮件集成到各类应用。$zh241$,
    $zh241$使用 Cloudflare Email Service（Email Sending 与 Email Routing）发送和接收交易邮件。构建邮件发送功能（Workers Binding 或 REST API）、邮件路由、Agents SDK 邮件处理，或将邮件集成到任何应用（Workers、Node.js、Python、Go 等）时使用。邮件送达率、SPF/DKIM/DMARC、wrangler 邮件配置、MCP 邮件工具，或编码代理需要发送邮件时也应使用。即使只是“为我的 Worker 添加邮件”这样的简单请求，也应使用本 Skill，因为其中包含关键配置细节。$zh241$
),
(
    'golang-concurrency',
    'zh-CN',
    $zh241$Go 并发$zh241$,
    $zh241$Go 并发模式。适用于涉及 goroutine、channel、select、锁、sync 原语、errgroup、singleflight、工作池或扇出/扇入流水线的并发代码，以及协程泄漏、竞态和 channel 所有权问题。$zh241$,
    $zh241$Go 并发模式。编写或审查涉及 goroutine、channel、select、锁、sync 原语、errgroup、singleflight、工作池或扇出/扇入流水线的并发 Go 代码时使用。发现协程泄漏、竞态条件、channel 所有权问题，或需要在 channel 与互斥锁之间做选择时也会触发。$zh241$
),
(
    'golang-naming',
    'zh-CN',
    $zh241$Go 命名规范$zh241$,
    $zh241$Go 命名规范，涵盖包、构造函数、结构体、接口、常量、枚举、错误、布尔值、接收者、Getter/Setter、函数式选项、缩写、测试函数和子测试名称。适用于编写、审查或重构中的命名决策。$zh241$,
    $zh241$Go 命名规范，涵盖包、构造函数、结构体、接口、常量、枚举、错误、布尔值、接收者、Getter/Setter、函数式选项、缩写、测试函数和子测试名称。编写新的 Go 代码、进行审查或重构、在命名备选项之间做选择（New 与 NewTypeName、isConnected 与 connected、ErrNotFound 与 NotFoundError、iota 0 处的 StatusReady 与 StatusUnknown）、讨论 Go 包名（utils/helpers 反模式），或询问 Go 命名最佳实践时使用。用户提到 MixedCaps 与 snake_case、ALL_CAPS 常量、Getter 的 Get 前缀或错误字符串大小写时也应触发。不适用于不涉及命名决策的一般 Go 实现问题。$zh241$
),
(
    'value',
    'zh-CN',
    $zh241$无操作 Skill$zh241$,
    $zh241$这个 Skill 什么也不做，使用它的代理同样什么也不做。$zh241$,
    $zh241$这个 Skill 什么也不做，使用它的代理同样什么也不做。$zh241$
),
(
    'golang-documentation',
    'zh-CN',
    $zh241$Go 文档$zh241$,
    $zh241$Go 项目综合文档指南，涵盖 godoc 注释、README、CONTRIBUTING、CHANGELOG、Go Playground、Example 测试、API 文档和 llms.txt，适用于库、应用与 CLI。$zh241$,
    $zh241$Go 项目综合文档指南，涵盖 godoc 注释、README、CONTRIBUTING、CHANGELOG、Go Playground、Example 测试、API 文档和 llms.txt。编写或审查文档注释与其他文档、添加代码示例、搭建文档站点，或讨论文档最佳实践时使用。库和应用/CLI 场景均会触发。$zh241$
),
(
    'golang-context',
    'zh-CN',
    $zh241$Go Context 使用$zh241$,
    $zh241$Go 中 context.Context 的惯用用法，涵盖跨 API 边界传播、取消、超时与截止时间、请求范围值，以及使用 context.WithoutCancel 让后台工作超出请求生命周期。$zh241$,
    $zh241$Go 中 context.Context 的惯用用法，涵盖跨 API 边界传播、取消、超时与截止时间、请求范围值，以及使用 context.WithoutCancel 让后台工作超出请求生命周期。设计跨层 Context 传播、排查泄漏或未到期的 Context、在 context.Background/TODO/WithoutCancel 之间选择，或在 Context 中存值时使用。不适用于仅把 ctx 作为第一个参数接收的代码。$zh241$
),
(
    'golang-data-structures',
    'zh-CN',
    $zh241$Go 数据结构$zh241$,
    $zh241$Go 数据结构指南，涵盖 Slice、Map、数组、container/list/heap/ring、strings.Builder 与 bytes.Buffer、泛型集合、指针及复制语义。适用于选择、实现或优化数据结构及研究 Slice/Map 内部机制。$zh241$,
    $zh241$Go 数据结构指南，涵盖 Slice（内部机制、容量增长、预分配、slices 包）、Map（内部机制、哈希桶、maps 包）、数组、container/list/heap/ring、strings.Builder 与 bytes.Buffer、泛型集合、指针（unsafe.Pointer、weak.Pointer）和复制语义。选择或优化 Go 数据结构、实现泛型容器、使用 container/ 包、unsafe 或弱指针，或探究 Slice/Map 内部机制时使用。$zh241$
),
(
    'golang-database',
    'zh-CN',
    $zh241$Go 数据库访问$zh241$,
    $zh241$Go 数据库访问综合指南，涵盖参数化查询、结构体扫描、可空列、事务、隔离级别、SELECT FOR UPDATE、连接池、批处理、Context 传播和迁移工具。支持 PostgreSQL、MariaDB、MySQL 与 SQLite。$zh241$,
    $zh241$Go 数据库访问综合指南，涵盖参数化查询、结构体扫描、可空列、事务、隔离级别、SELECT FOR UPDATE、连接池、批处理、Context 传播和迁移工具。编写、审查或调试与 PostgreSQL、MariaDB、MySQL 或 SQLite 交互的 Go 代码、进行数据库测试，或处理 database/sql、sqlx 或 pgx 相关问题时使用。不会生成数据库架构或迁移 SQL。$zh241$
),
(
    'golang-safety',
    'zh-CN',
    $zh241$Go 防御式编程$zh241$,
    $zh241$通过防御式 Go 编码预防 panic、静默数据损坏和隐蔽运行时错误。适用于 nil panic、append 别名、Map 并发访问、浮点比较陷阱、零值设计，以及 nil 安全、数值溢出和资源生命周期审查。$zh241$,
    $zh241$通过防御式 Go 编码预防 panic、静默数据损坏和隐蔽的运行时错误。遇到 nil panic、append 别名问题、Map 并发访问、浮点数比较陷阱或零值设计问题时使用。审查代码的 nil 安全性、数值转换溢出、资源生命周期问题（循环中的 defer），或 Slice 和 Map 的防御性复制时也应使用。$zh241$
),
(
    'golang-modernize',
    'zh-CN',
    $zh241$Go 代码现代化$zh241$,
    $zh241$使用新版语言特性、标准库改进和惯用模式实现 Go 代码现代化。编写或审查代码时发现旧式模式或弃用警告应主动触发，也适用于 Go 版本升级及 CI/工具链更新。$zh241$,
    $zh241$使用近期语言特性、标准库改进和惯用模式实现 Go 代码现代化。编写或审查 Go 代码时发现旧式模式，或遇到弃用警告时应主动触发。用户明确要求代码现代化、升级 Go 版本，或更新 CI/工具链时也应使用。$zh241$
),
(
    'fastify-best-practices',
    'zh-CN',
    $zh241$Fastify 最佳实践$zh241$,
    $zh241$指导使用 TypeScript 或 JavaScript 开发 Fastify Node.js 后端服务器和 REST API，涵盖路由、插件、JSON Schema 验证、错误处理、性能、安全、数据库、WebSocket、生产部署及完整请求生命周期。$zh241$,
    $zh241$指导使用 TypeScript 或 JavaScript 开发 Fastify Node.js 后端服务器和 REST API。构建、配置或调试 Fastify 应用时使用，包括定义路由、实现插件、配置 JSON Schema 验证、处理错误、优化性能、管理身份验证、配置 CORS 与安全标头、集成数据库、使用 WebSocket 以及部署到生产环境。涵盖完整的 Fastify 请求生命周期（Hook、序列化、使用 Pino 记录日志）及通过 strip types 集成 TypeScript。触发词：Fastify、Node.js 服务器、REST API、API 路由、后端框架、fastify.config、server.ts、app.ts。$zh241$
),
(
    'golang-project-layout',
    'zh-CN',
    $zh241$Go 项目布局$zh241$,
    $zh241$Go 项目布局与工作区指南。适用于新建项目、整理现有代码库、搭建多包 Monorepo、创建含多个 main 包的 CLI 工具，以及决定 cmd/internal/pkg 目录约定或拆分包与模块。$zh241$,
    $zh241$Go 项目布局与工作区指南。启动新的 Go 项目、组织现有代码库、搭建包含多个包的 Monorepo、创建具有多个 main 包的 CLI 工具、决定 cmd/internal/pkg 目录约定，或讨论包重构、包拆分或模块拆分时使用。$zh241$
),
(
    'weread-skills',
    'zh-CN',
    $zh241$微信读书助手$zh241$,
    $zh241$微信读书助手：搜索书籍、管理书架、查看笔记与划线、浏览书评、查看阅读统计，并发现推荐好书。$zh241$,
    $zh241$微信读书助手：搜索书籍、管理书架、查看笔记与划线、浏览书评、查看阅读统计，并发现推荐好书。$zh241$
),
(
    'gws-slides',
    'zh-CN',
    $zh241$Google Slides 演示文稿$zh241$,
    $zh241$Google Slides：读取和写入演示文稿。$zh241$,
    $zh241$Google Slides：读取和写入演示文稿。$zh241$
),
(
    'golang-lint',
    'zh-CN',
    $zh241$Go 代码检查$zh241$,
    $zh241$Go 项目代码检查最佳实践与 golangci-lint 配置，涵盖运行检查器、配置 .golangci.yml、使用 nolint 抑制警告、解读输出和选择检查器。$zh241$,
    $zh241$Go 项目代码检查最佳实践与 golangci-lint 配置，涵盖运行检查器、配置 .golangci.yml、使用 nolint 指令抑制警告、解读检查输出和选择检查器。配置 golangci-lint、询问检查警告或 nolint 抑制、搭建代码质量工具，或选择检查器时使用。用户提到 golangci-lint、go vet、staticcheck 或 revive 时也应使用。$zh241$
),
(
    'golang-troubleshooting',
    'zh-CN',
    $zh241$Go 故障排查$zh241$,
    $zh241$系统排查 Go 程序并找到、修复根因。适用于错误、崩溃、死锁或异常行为，涵盖调试方法、常见陷阱、测试驱动调试、pprof、Delve、竞态检测、GODEBUG 跟踪和生产调试。$zh241$,
    $zh241$系统排查 Go 程序并找到、修复根因。Go 代码出现错误、崩溃、死锁或异常行为时使用。涵盖调试方法、常见 Go 陷阱、测试驱动调试、pprof 配置与采集、Delve 调试器、竞态检测、GODEBUG 跟踪和生产环境调试。任何“出了问题”的场景都应从这里开始。不适用于解读性能分析结果或基准测试（参见 `samber/cc-skills-golang@golang-benchmark` Skill），也不适用于应用优化模式（参见 `samber/cc-skills-golang@golang-performance` Skill）。$zh241$
),
(
    'higgsfield-video-explainer',
    'zh-CN',
    $zh241$Higgsfield 解说视频$zh241$,
    $zh241$以有序的 10 秒片段构建完整的非写实旁白解说或故事视频：统一旁白与风格，每段生成一条 Seed Audio 和一个 Gemini Omni 片段，再由 explainer_video 在服务端合成。$zh241$,
    $zh241$以有序的 10 秒片段构建完整的非写实旁白解说或故事视频：使用一名旁白、一套通用风格键，每段生成一条 Seed Audio 音频和一个 Gemini Omni 片段，再通过 explainer_video 在服务端合成。适用于：“制作解说视频”“用视频解释这个内容”“把这个主题或文档转成旁白视频”“把这个故事做成动画视频”“制作无真人出镜的旁白视频”或“向我展示解说视频风格”。支持在线 CMS 预设、自定义风格参考、吉祥物/无真人出镜模式、两种画面比例，以及可选的烧录字幕。不适用于：写实电影、广告/UGC、真人口播、播客、动态排版短片、没有旁白的一次性片段，或编辑已经完成的视频。$zh241$
),
(
    'golang-observability',
    'zh-CN',
    $zh241$Go 可观测性$zh241$,
    $zh241$Go 日常可观测性，即生产环境中始终开启的信号。涵盖 slog 结构化日志、Prometheus 指标、OpenTelemetry 分布式追踪、pprof/Pyroscope 持续分析、服务端 RUM 事件、告警和 Grafana 仪表板。$zh241$,
    $zh241$Go 日常可观测性，即生产环境中始终开启的信号。涵盖使用 slog 的结构化日志、Prometheus 指标、OpenTelemetry 分布式追踪、使用 pprof/Pyroscope 的持续性能分析、服务端 RUM 事件跟踪、告警和 Grafana 仪表板。为 Go 服务接入生产监控、配置指标或告警、添加 OpenTelemetry 追踪、关联日志与追踪、将旧日志库（zap/logrus/zerolog）迁移到 slog、为新功能添加可观测性，或通过客户数据平台（CDP）实现符合 GDPR/CCPA 的跟踪时使用。不适用于临时的深入性能调查（参见 `samber/cc-skills-golang@golang-benchmark` 与 `samber/cc-skills-golang@golang-performance` Skill）。$zh241$
),
(
    'golang-popular-libraries',
    'zh-CN',
    $zh241$Go 常用库推荐$zh241$,
    $zh241$推荐可用于生产的 Go 库与框架。适用于明确请求库推荐、比较备选方案、为特定任务选库，或项目准备添加新依赖时。$zh241$,
    $zh241$推荐可用于生产的 Go 库与框架。当用户明确请求库推荐、希望比较备选方案、需要为特定任务选择库，或项目准备添加新依赖时使用。$zh241$
),
(
    'golang-dependency-management',
    'zh-CN',
    $zh241$Go 依赖管理$zh241$,
    $zh241$Go 项目依赖管理策略，涵盖 go.mod、包安装与升级、最小版本选择、漏洞扫描、过期依赖跟踪、二进制大小分析、Dependabot/Renovate、冲突解决和 go.work 工作区。$zh241$,
    $zh241$Go 项目依赖管理策略，涵盖 go.mod 管理、安装或升级包、最小版本选择、漏洞扫描、过期依赖跟踪、二进制大小分析、Dependabot/Renovate 配置、冲突解决和 go.work 工作区。添加、删除或升级 Go 依赖、审计漏洞、解决版本冲突，或设置自动依赖更新时使用。$zh241$
),
(
    'golang-structs-interfaces',
    'zh-CN',
    $zh241$Go 结构体与接口$zh241$,
    $zh241$Go 结构体与接口设计模式，涵盖组合、嵌入、类型断言与切换、接口隔离、通过接口进行依赖注入、结构体字段标签，以及指针与值接收者。$zh241$,
    $zh241$Go 结构体与接口设计模式，涵盖组合、嵌入、类型断言、类型切换、接口隔离、通过接口进行依赖注入、结构体字段标签，以及指针与值接收者。设计 Go 类型、定义或实现接口、嵌入结构体或接口、编写类型断言或类型切换、为 JSON/YAML/DB 序列化添加结构体字段标签，或在指针接收者和值接收者之间选择时使用。用户询问“接受接口，返回结构体”、编译期接口检查，或将小接口组合成大接口时也应使用。$zh241$
),
(
    'golang-dependency-injection',
    'zh-CN',
    $zh241$Go 依赖注入$zh241$,
    $zh241$Go 依赖注入综合指南，说明 DI 对可测试性、松耦合、关注点分离与生命周期管理的价值，涵盖手动构造函数注入，以及 google/wire、uber-go/dig、uber-go/fx、samber/do 的比较。$zh241$,
    $zh241$Go 依赖注入（DI）综合指南。涵盖 DI 为何重要（可测试性、松耦合、关注点分离、生命周期管理）、手动构造函数注入，以及 DI 库比较（google/wire、uber-go/dig、uber-go/fx、samber/do）。设计服务架构、设置依赖注入、重构紧耦合代码、管理单例或服务工厂，或用户询问控制反转、服务容器或在 Go 中装配依赖时使用。有关具体 DI 库，请参见 `samber/cc-skills-golang@golang-google-wire`、`samber/cc-skills-golang@golang-uber-dig`、`samber/cc-skills-golang@golang-uber-fx` 或 `samber/cc-skills-golang@golang-samber-do` Skill。$zh241$
),
(
    'golang-benchmark',
    'zh-CN',
    $zh241$Go 基准测试$zh241$,
    $zh241$Go 基准测试、性能分析与性能测量。适用于编写、运行或比较基准测试，使用 pprof 分析热路径，解读 CPU/内存/Trace 分析，使用 benchstat 分析结果，以及在 CI 或生产环境监测性能。$zh241$,
    $zh241$Go 基准测试、性能分析与性能测量。编写、运行或比较 Go 基准测试、使用 pprof 分析热路径、解读 CPU/内存/Trace 分析结果、使用 benchstat 分析结果、在 CI 中设置基准回归检测，或使用 Prometheus 运行时指标调查生产性能时使用。开发者需要深入分析某项具体性能指标时也应使用：本 Skill 提供测量方法，而 `samber/cc-skills-golang@golang-performance` 提供优化模式。$zh241$
),
(
    'gws-gmail-read',
    'zh-CN',
    $zh241$Gmail 邮件读取$zh241$,
    $zh241$Gmail：读取邮件并提取正文或标头。$zh241$,
    $zh241$Gmail：读取邮件并提取正文或标头。$zh241$
),
(
    'golang-cli',
    'zh-CN',
    $zh241$Go CLI 开发$zh241$,
    $zh241$Go CLI 应用开发，涵盖命令结构、标志处理、配置分层、版本嵌入、退出码、I/O 模式、信号处理、Shell 补全、参数验证和单元测试，并支持 cobra、viper 与 urfave/cli。$zh241$,
    $zh241$Go CLI 应用开发。构建、修改或审查 Go CLI 工具时使用，尤其涉及命令结构、标志处理、配置分层、版本嵌入、退出码、I/O 模式、信号处理、Shell 补全、参数验证和 CLI 单元测试时。代码使用 cobra、viper 或 urfave/cli 时也会触发。cobra 专用 API 请参见 `samber/cc-skills-golang@golang-spf13-cobra` Skill；viper 配置分层请参见 `samber/cc-skills-golang@golang-spf13-viper` Skill。$zh241$
),
(
    'golang-grpc',
    'zh-CN',
    $zh241$Go gRPC 服务$zh241$,
    $zh241$面向 Go 微服务的 gRPC 使用指南、protobuf 组织方式与生产级模式。适用于实现、审查或调试 gRPC 服务端/客户端、编写 proto、配置拦截器与 TLS/mTLS、处理状态码错误、bufconn 测试及流式 RPC。$zh241$,
    $zh241$提供面向 Go 微服务的 gRPC 使用指南、protobuf 组织方式和生产级模式。实现、审查或调试 gRPC 服务端/客户端、编写 proto 文件、配置拦截器、使用状态码处理 gRPC 错误、配置 TLS/mTLS、使用 bufconn 测试，或处理流式 RPC 时使用。$zh241$
),
(
    'golang-continuous-integration',
    'zh-CN',
    $zh241$Go 持续集成$zh241$,
    $zh241$使用 GitHub Actions 配置 Go 项目的 CI/CD 流水线，涵盖测试、代码检查、SAST、安全扫描、覆盖率、Dependabot、Renovate、GoReleaser、代码审查自动化和发布流水线。$zh241$,
    $zh241$使用 GitHub Actions 配置 Go 项目的 CI/CD 流水线，涵盖测试、代码检查、SAST、安全扫描、代码覆盖率、Dependabot、Renovate、GoReleaser、代码审查自动化和发布流水线。设置或改进 Go 项目 CI、配置 GitHub Actions 工作流、添加代码检查器或安全扫描器、自动更新依赖，或添加质量门禁时使用。$zh241$
),
(
    'replicas-agent',
    'zh-CN',
    $zh241$Replicas 后台编码代理$zh241$,
    $zh241$在 Replicas 云工作区内运行后台编码代理的指南。$zh241$,
    $zh241$在 Replicas 云工作区内运行后台编码代理的指南。$zh241$
),
(
    'ponytail',
    'zh-CN',
    $zh241$Ponytail 极简编码$zh241$,
    $zh241$强制采用真正可用且最省事、最简单、最短、最精简的方案：先用 YAGNI 判断任务是否需要存在，优先标准库和平台原生能力，并避免过度工程、膨胀、样板代码与非必要依赖。$zh241$,
    $zh241$强制采用真正可用且最省事、最简单、最短、最精简的方案。它体现一位见多识广的高级开发者的思维：以 YAGNI 质疑任务是否有存在必要；先选标准库，再写自定义代码；先用平台原生能力，再引入依赖；能用一行就不用五十行。支持三种强度：lite、full（默认）和 ultra。适用于任何编码任务，包括编写、添加、重构、修复、审查或设计代码，以及选择库或依赖。用户说“ponytail”“偷懒一点”“懒人模式”“最简单方案”“最小方案”“yagni”“少做一点”或“最短路径”，或抱怨过度工程、膨胀、样板代码或非必要依赖时也应使用。不适用于非编码请求，如一般知识、散文、翻译、摘要和食谱。$zh241$
),
(
    'golang-stretchr-testify',
    'zh-CN',
    $zh241$Go Testify 测试$zh241$,
    $zh241$Go 测试库 stretchr/testify 综合指南，深入讲解 assert、require、mock 和 suite。适用于编写 Testify 测试、创建 Mock、配置测试套件，或在 assert 与 require 间选择。$zh241$,
    $zh241$Go 测试库 stretchr/testify 综合指南，深入讲解 assert、require、mock 和 suite 包。使用 Testify 编写测试、创建 Mock、设置测试套件，或在 assert 与 require 之间选择时使用。涵盖 Testify 断言、Mock 预期、参数匹配器、调用验证、套件生命周期，以及 Eventually、JSONEq 和自定义匹配器等高级模式。当代码库导入 github.com/stretchr/testify 时应用。$zh241$
),
(
    'firebase-ai-logic',
    'zh-CN',
    $zh241$Firebase AI Logic 网页集成$zh241$,
    $zh241$将 Firebase AI Logic（Gemini API）集成到 Web 应用的官方 Skill，涵盖设置、多模态推理、结构化输出和安全。$zh241$,
    $zh241$将 Firebase AI Logic（Gemini API）集成到 Web 应用的官方 Skill。涵盖设置、多模态推理、结构化输出和安全。$zh241$
),
(
    'captions-overlay',
    'zh-CN',
    $zh241$字幕叠加准则$zh241$,
    $zh241$嵌入字幕工作流的叠加准则：使用 drop/rail/embed 字幕模型，并始终把字幕作为合成在影片之上的叠加层，绝不预留底部条带后将内容上移避让。$zh241$,
    $zh241$嵌入字幕工作流的叠加准则：采用 drop（丢弃）/rail（轨道）/embed（嵌入）字幕模型，并遵循一项规则——字幕是合成在影片之上的叠加层，绝不是需要预留、再将内容上移避让的底部条带。为真人口播或发布视频添加字幕时；决定某个短语应被丢弃、沿逐字轨道呈现，还是提升为稀缺的嵌入式高潮时；为承载字幕的构图排版时（不得预留禁入条带）；或在字幕下方按画面真实中心对齐构图时加载。引用 embedded-captions 中的 rail+embed 模型及其约束。$zh241$
),
(
    'motion-doctrine',
    'zh-CN',
    $zh241$运动设计准则$zh241$,
    $zh241$入口 Skill：创作任何 HyperFrames 动画或视频前必须首先加载。通过高层运动法则，让多场景视频呈现为一次连续的镜头运动，而非一叠彼此独立的动画幻灯片。$zh241$,
    $zh241$入口 Skill——创作任何 HyperFrames 动画或视频前必须首先加载。这套高层运动法则让多场景视频呈现为一次连续的镜头运动，而不是一叠彼此独立的动画幻灯片。涵盖向量法则（离场方式决定入场方式，包括 Z 轴缩放符号规则）、影片流向、载体元素、因果运动、Seam Gate（构建门禁强制执行）、禁止空闲摇摆（运动必须承担表现任务，而非只是呼吸）、高潮前静止，以及持续运动路线。它会引导到低层技术 Skill：cut-the-curve（完整技法目录，包括瀑布式入场与 nudge 曲线）、oversized-cursor 和 seam-craft。这些规则优先于通用或上游运动指南。[连续性、方向、向量、动量、接缝、转场、缓动、表现性、空闲运动、叙事运动、电影语法]$zh241$
),
(
    'golang-stay-updated',
    'zh-CN',
    $zh241$Go 动态追踪$zh241$,
    $zh241$提供持续了解 Go 新闻、社区和重要人物的资源。适用于寻找 Go 学习资源、发现新库、查找社区渠道，或跟进 Go 语言变化与版本发布。$zh241$,
    $zh241$提供持续了解 Go 新闻、社区和重要人物的资源。寻找 Go 学习资源、发现新库、查找社区渠道，或跟进 Go 语言变化与版本发布时使用。$zh241$
),
(
    'golang-samber-lo',
    'zh-CN',
    $zh241$Go samber/lo 函数式工具$zh241$,
    $zh241$基于 samber/lo 的 Go 函数式编程工具，提供 500 多个类型安全泛型函数，覆盖 Slice、Map、Channel、字符串、数学、元组与并发，并包含不可变、并发、原地修改、惰性迭代器和实验性 SIMD 变体。$zh241$,
    $zh241$基于 samber/lo 的 Go 函数式编程工具，为 Slice、Map、Channel、字符串、数学、元组和并发提供 500 多个类型安全泛型函数（Map、Filter、Reduce、GroupBy、Chunk、Flatten、Find、Uniq 等）。包含核心不可变包 lo、并发变体 lo/parallel（又名 lop）、原地修改包 lo/mutable（又名 lom）、惰性迭代器 lo/it（Go 1.23+，又名 loi），以及实验性 SIMD 包 lo/exp/simd。使用或引入 samber/lo、代码库导入 github.com/samber/lo，或在 Go 中实现函数式数据转换时应用。不适用于流式处理流水线（参见 `samber/cc-skills-golang@golang-samber-ro` Skill）。$zh241$
),
(
    'cut-the-curve',
    'zh-CN',
    $zh241$Cut the Curve 转场技法$zh241$,
    $zh241$技法目录：五种速度匹配的接缝转场，以及瀑布式入场和 nudge 曲线两种场内技法；涵盖镜像 power4 缓动、Z 轴缩放符号、按尺寸缩放模糊、逐词错落剪切、级联节奏和 10/65/25 滑动比例。$zh241$,
    $zh241$技法目录：五种速度匹配的接缝转场——穿透缩放、反向穿透缩放、cut-the-curve、瀑布式剪切和变焦模糊剪切；另含两种场内技法——瀑布式入场（用于标题卡或分段开场的错落到达级联）与 nudge 曲线（慢—快—慢三阶段成组滑动）。涵盖通过镜像 power4 缓动实现部分位移（约为画面 12%）的速度匹配、Z 轴缩放符号规则、按尺寸调整的模糊（文字 10px，满画面 18–20px）、逐词错落剪切、按元素权重控制级联节奏，以及 10/65/25 滑动比例。创作任何转场、文字节拍交接、动效文字入场或成组重定位前请先阅读。[景深、缩放、反向缩放、缩放符号、镜像缩放、变焦、节奏、速度、cut-the-curve、瀑布、错落、级联、动效文字、标题卡、分段开场、nudge、滑动、缓动、成组运动、Z 轴景深、动态图形、电影感、转场、模糊、方向连续性]$zh241$
),
(
    'golang-samber-do',
    'zh-CN',
    $zh241$Go samber/do 依赖注入$zh241$,
    $zh241$使用 samber/do 在 Go 中实现依赖注入，涵盖服务容器、生命周期管理、作用域、健康检查、优雅关闭和模块组织。适用于引入或使用 samber/do，以及将手动构造函数注入重构为 DI 容器。$zh241$,
    $zh241$使用 samber/do 在 Go 中实现依赖注入，涵盖服务容器、生命周期管理、作用域、健康检查、优雅关闭和模块组织。使用或引入 samber/do、代码库导入 github.com/samber/do 或 github.com/samber/do/v2，或将手动构造函数注入重构为 DI 容器时应用。$zh241$
),
(
    'golang-samber-slog',
    'zh-CN',
    $zh241$Go samber/slog 结构化日志$zh241$,
    $zh241$使用 samber/slog-**** 包扩展 Go 结构化日志，涵盖多处理器流水线、日志采样、属性格式化、HTTP 中间件及 Datadog、Sentry、Loki、Syslog、Logstash、Graylog 等后端路由。$zh241$,
    $zh241$使用 samber/slog-**** 包扩展 Go 结构化日志，涵盖多处理器流水线（slog-multi）、日志采样（slog-sampling）、属性格式化（slog-formatter）、HTTP 中间件（slog-fiber、slog-gin、slog-chi、slog-echo），以及后端路由（slog-datadog、slog-sentry、slog-loki、slog-syslog、slog-logstash、slog-graylog 等）。使用或引入 slog，或代码库已导入任何 github.com/samber/slog-* 包时应用。$zh241$
),
(
    'prototype-6459e7f10c',
    'zh-CN',
    $zh241$UI 原型方案$zh241$,
    $zh241$根据描述构建多个真正不同的 UI 版本，并在可视化选择器中渲染，方便实时切换并选定最合适的方案。仅在显式调用时运行，不会自行触发。$zh241$,
    $zh241$根据你描述的 UI 部件构建多个真正不同的版本，并在可视化选择器中渲染，让你可以实时切换浏览并将感觉最合适的版本提升为最终方案。仅在显式调用时运行，不会自行触发。$zh241$
),
(
    'golang-samber-oops',
    'zh-CN',
    $zh241$Go samber/oops 错误处理$zh241$,
    $zh241$使用 samber/oops 在 Go 中进行结构化错误处理，涵盖错误构建器、堆栈跟踪、错误码与上下文、错误包装、属性、面向用户与开发者的消息、panic 恢复和日志集成。$zh241$,
    $zh241$使用 samber/oops 在 Go 中进行结构化错误处理，涵盖错误构建器、堆栈跟踪、错误码、错误上下文、错误包装、错误属性、面向用户与开发者的消息、panic 恢复和日志记录器集成。使用或引入 samber/oops，或代码库已导入 github.com/samber/oops 时应用。$zh241$
),
(
    'brand-landingpage',
    'zh-CN',
    $zh241$品牌优先落地页设计$zh241$,
    $zh241$品牌优先的落地页设计器：先进行品牌识别访谈（颜色、字体与形状语言），再通过 Stitch 生成并迭代精致的落地页，输出可部署 HTML。适用于尚无明确视觉方向的落地页、首页或营销页。$zh241$,
    $zh241$品牌优先的落地页设计器：先进行品牌识别访谈（颜色、字体与形状语言），再通过 Stitch 生成并迭代精致的落地页，输出可部署的 HTML。当用户希望创建、设计或构建落地页、首页或营销页面，但尚无既定视觉方向时使用。如果用户已有设计稿、需要仪表板或应用 UI、正在组件层工作、构建多页面应用，或使用已知设计 Token 重新设计样式，则跳过本 Skill，改用 frontend-design。$zh241$
),
(
    'golang-samber-mo',
    'zh-CN',
    $zh241$Go samber/mo 单子类型$zh241$,
    $zh241$使用 samber/mo 为 Go 提供单子类型，包括 Option、Result、Either、Future、IO、Task 和 State，用于类型安全的可空值、错误处理，以及通过流水线子包进行函数式组合。$zh241$,
    $zh241$使用 samber/mo 为 Go 提供单子类型，包括 Option、Result、Either、Future、IO、Task 和 State，用于类型安全的可空值、错误处理，以及通过流水线子包进行函数式组合。使用或引入 samber/mo、代码库导入 `github.com/samber/mo`，或考虑将函数式编程模式作为 Go 的安全设计时应用。$zh241$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
