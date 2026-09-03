-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 589-670.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'golang-samber-hot',
    'zh-CN',
    $zh242$Golang samber/hot 内存缓存$zh242$,
    $zh242$使用 samber/hot 在 Golang 中实现内存缓存，涵盖淘汰算法（LRU、LFU、TinyLFU、W-TinyLFU、S3FIFO、ARC、TwoQueue、SIEVE、FIFO）、TTL、缓存加载器、分片、陈旧数据后台重验证、缺失键缓存和 Prometheus 指标。适用于使用或引入 samber/hot 的场景。$zh242$,
    $zh242$使用 samber/hot 在 Golang 中实现内存缓存，涵盖淘汰算法（LRU、LFU、TinyLFU、W-TinyLFU、S3FIFO、ARC、TwoQueue、SIEVE、FIFO）、TTL、缓存加载器、分片、陈旧数据后台重验证、缺失键缓存和 Prometheus 指标。适用于正在使用或准备引入 samber/hot、代码库导入 github.com/samber/hot，或项目需要高频重复加载相同的中低基数资源并降低延迟或后端压力的场景。$zh242$
),
(
    'golang-samber-ro',
    'zh-CN',
    $zh242$Golang samber/ro 响应式流$zh242$,
    $zh242$使用 samber/ro 在 Golang 中进行响应式流与事件驱动编程：提供含 150 多个类型安全操作符的 ReactiveX 实现、冷/热可观察对象、5 种 Subject（Publish、Behavior、Replay、Async、Unicast）、通过 Pipe 构建的声明式管道，以及 40 多个插件（HTTP、cron、fsnotify 等）。$zh242$,
    $zh242$使用 samber/ro 在 Golang 中进行响应式流与事件驱动编程：提供含 150 多个类型安全操作符的 ReactiveX 实现、冷/热可观察对象、5 种 Subject（Publish、Behavior、Replay、Async、Unicast）、通过 Pipe 构建的声明式管道、40 多个插件（HTTP、cron、fsnotify、JSON、日志）、自动背压、错误传播和 Go context 集成。适用于正在使用或准备引入 samber/ro、代码库导入 github.com/samber/ro，或在 Go 中构建异步事件驱动管道、实时数据处理、数据流或响应式架构的场景。不适用于有限切片转换（→ 请参阅 `samber/cc-skills-golang@golang-samber-lo` 技能）。$zh242$
),
(
    'seam-craft',
    'zh-CN',
    $zh242$HyperFrames 场景接缝工艺$zh242$,
    $zh242$HyperFrames 发布视频中场景接缝的渲染正确性准则，说明转场在主时间线上正确合成所需的前提。适用于组装主时间线或 index.html、剪切或交叉淡化接缝处出现白闪，或分析转场不透明度下降为何会透出底层的场景。$zh242$,
    $zh242$HyperFrames 发布视频中场景接缝的渲染正确性准则，说明转场在主时间线上正确合成所需的前提。适用于组装主时间线或 index.html、剪切或交叉淡化接缝处出现白闪（尤其在暗色影片中）、分析转场不透明度下降为何会透出底层，或验证重叠场景包装器如何混合的渲染端机制。涵盖不透明舞台底层（#root 背景）的白闪防护，以及注入器如何重叠包装器、保持最终帧、让轨道往返播放，并将通过 lint 的模板代码写入主时间线。不包含逐项转场目录；各个转场条目请参阅转场注册表。$zh242$
),
(
    'oversized-cursor',
    'zh-CN',
    $zh242$HyperFrames 超大光标$zh242$,
    $zh242$HyperFrames 发布视频的标准超大 macOS 光标技法。凡场景涉及光标或由指针引导的操作、需要开启 UI 场景、用点击引出变形/转场/输入片段，或场景显得静止、沉闷、陈旧而需要动态元素时使用。$zh242$,
    $zh242$HyperFrames 发布视频的标准超大 macOS 光标技法。凡场景涉及光标或由指针引导的操作、需要开启 UI 场景、用点击引出变形/转场/输入片段，或场景显得静止、沉闷、陈旧而需要一个低成本高收益的动态来源来引导观众视线并使其摆脱停滞感时使用。涵盖光标的尺寸与外观（包括采用品牌图案的光标）、从屏幕外进入的规则、尖端对准与点击动作、由点击引出下一节拍，以及退出和跨场景交接。$zh242$
),
(
    'developing-genkit-python',
    'zh-CN',
    $zh242$使用 Python 开发 Genkit$zh242$,
    $zh242$使用 Python 开发由 AI 驱动的 Genkit 应用。适用于用户询问 Python 中的 Genkit、AI 智能体、流程或工具，以及遇到 Genkit 错误、导入问题或 API 问题的场景。$zh242$,
    $zh242$使用 Python 开发由 AI 驱动的 Genkit 应用。适用于用户询问 Python 中的 Genkit、AI 智能体、流程或工具，以及遇到 Genkit 错误、导入问题或 API 问题的场景。$zh242$
),
(
    'design-doc-mermaid',
    'zh-CN',
    $zh242$Mermaid 设计文档图表$zh242$,
    $zh242$根据文字说明或源代码创建 Mermaid 图表（活动、部署、时序、架构）。适用于“创建图表”“生成 Mermaid”“记录架构”“代码转图表”“创建设计文档”或“将代码转换为图表”等请求，并支持分层按需加载指南。$zh242$,
    $zh242$根据文字说明或源代码创建 Mermaid 图表（活动、部署、时序、架构）。适用于“创建图表”“生成 Mermaid”“记录架构”“代码转图表”“创建设计文档”或“将代码转换为图表”等请求。支持分层按需加载指南、Unicode 语义符号，以及用于提取图表和转换图像的 Python 实用工具。$zh242$
),
(
    'gws-forms',
    'zh-CN',
    $zh242$Google 表单读写$zh242$,
    $zh242$读取和写入 Google 表单。$zh242$,
    $zh242$读取和写入 Google 表单。$zh242$
),
(
    'sandbox-sdk',
    'zh-CN',
    $zh242$Sandbox SDK 安全沙箱$zh242$,
    $zh242$构建用于安全执行代码的沙箱应用。适用于构建 AI 代码执行、代码解释器、CI/CD 系统、交互式开发环境，或执行不受信任代码的场景。涵盖 Sandbox SDK 生命周期、命令、文件、代码解释器和预览 URL。$zh242$,
    $zh242$构建用于安全执行代码的沙箱应用。适用于构建 AI 代码执行、代码解释器、CI/CD 系统、交互式开发环境，或执行不受信任代码的场景。涵盖 Sandbox SDK 生命周期、命令、文件、代码解释器和预览 URL。相比预训练知识，本技能更倾向于从 Cloudflare 文档检索信息。$zh242$
),
(
    'self-improving-agent',
    'zh-CN',
    $zh242$自我改进智能体$zh242$,
    $zh242$一个能从所有技能经历中学习的通用自我改进智能体。采用多记忆架构（语义记忆、情景记忆和工作记忆）持续演进代码库，并在技能完成或出错时通过钩子自动触发自我纠正。$zh242$,
    $zh242$一个能从所有技能经历中学习的通用自我改进智能体。采用多记忆架构（语义记忆、情景记忆和工作记忆）持续演进代码库，并在技能完成或出错时通过钩子自动触发自我纠正。$zh242$
),
(
    'gws-meet',
    'zh-CN',
    $zh242$Google Meet 会议管理$zh242$,
    $zh242$管理 Google Meet 会议。$zh242$,
    $zh242$管理 Google Meet 会议。$zh242$
),
(
    'golang-swagger',
    'zh-CN',
    $zh242$Golang Swagger 文档$zh242$,
    $zh242$使用 swaggo/swag 编写 Golang OpenAPI/Swagger 文档，涵盖注解注释（@Summary、@Param、@Success、@Router、@Security）、swag init 代码生成、框架集成（gin、echo、fiber、chi、net/http）、安全定义（Bearer/JWT、OAuth2、API key）和结构体标签。$zh242$,
    $zh242$使用 swaggo/swag 编写 Golang OpenAPI/Swagger 文档，涵盖注解注释（@Summary、@Param、@Success、@Router、@Security）、swag init 代码生成、框架集成（gin、echo、fiber、chi、net/http）、安全定义（Bearer/JWT、OAuth2、API key）和结构体标签（swaggertype、enums、example、swaggerignore）。适用于在 Go 项目中新增或维护 Swagger/OpenAPI 文档，或代码库导入 github.com/swaggo/swag、github.com/swaggo/gin-swagger、github.com/swaggo/echo-swagger、github.com/swaggo/http-swagger 或 github.com/swaggo/files 的场景。$zh242$
),
(
    'ckm-design-system',
    'zh-CN',
    $zh242$CKM 设计系统$zh242$,
    $zh242$令牌架构、组件规范和幻灯片生成。涵盖三层令牌（原始→语义→组件）、CSS 变量、间距与排版尺度、组件规范和战略性幻灯片创作。适用于设计令牌、系统化设计和符合品牌规范的演示文稿。$zh242$,
    $zh242$令牌架构、组件规范和幻灯片生成。涵盖三层令牌（原始→语义→组件）、CSS 变量、间距与排版尺度、组件规范和战略性幻灯片创作。适用于设计令牌、系统化设计和符合品牌规范的演示文稿。$zh242$
),
(
    'golang-graphql',
    'zh-CN',
    $zh242$Golang GraphQL API 开发$zh242$,
    $zh242$使用 gqlgen 或 graphql-go 在 Golang 中实现 GraphQL API。适用于构建 GraphQL 服务器、设计 schema、编写 resolver、处理订阅，或将 GraphQL 与现有 Go HTTP 服务集成的场景，也适用于代码库导入相关包时。$zh242$,
    $zh242$使用 gqlgen 或 graphql-go 在 Golang 中实现 GraphQL API。适用于构建 GraphQL 服务器、设计 schema、编写 resolver、处理订阅，或将 GraphQL 与现有 Go HTTP 服务集成的场景。代码库导入 `github.com/99designs/gqlgen` 或 `github.com/graph-gophers/graphql-go` 时也应使用。$zh242$
),
(
    'golang-spf13-cobra',
    'zh-CN',
    $zh242$Golang Cobra 命令树$zh242$,
    $zh242$使用 spf13/cobra 构建 Golang CLI 命令树，涵盖 cobra.Command、RunE 与 Run、PersistentPreRunE 钩子链、Args 验证器（NoArgs、ExactArgs、MatchAll、自定义）、持久与本地 flag、命令分组、ValidArgsFunction、RegisterFlagCompletionFunc、ShellCompDirective 和用法配置。$zh242$,
    $zh242$使用 spf13/cobra 构建 Golang CLI 命令树，涵盖 cobra.Command、RunE 与 Run、PersistentPreRunE 钩子链、Args 验证器（NoArgs、ExactArgs、MatchAll、自定义）、持久与本地 flag、命令分组、ValidArgsFunction、RegisterFlagCompletionFunc、ShellCompDirective、用法/帮助模板自定义、man 页面和 Markdown 文档生成，以及使用 SetArgs/SetOut/SetErr 进行测试。适用于正在使用或准备引入 spf13/cobra，或代码库导入 `github.com/spf13/cobra` 的场景。需要与 cobra 配套的分层配置时，请参阅 `samber/cc-skills-golang@golang-spf13-viper` 技能。通用 CLI 架构（项目布局、退出码、信号处理、I/O 模式）请参阅 `samber/cc-skills-golang@golang-cli`。$zh242$
),
(
    'vite',
    'zh-CN',
    $zh242$Vite 构建与迁移$zh242$,
    $zh242$Vite 构建工具配置、插件 API、SSR，以及迁移到 Vite 8 Rolldown。适用于处理 Vite 项目、vite.config.ts、Vite 插件，或使用 Vite 构建库和 SSR 应用的场景。$zh242$,
    $zh242$Vite 构建工具配置、插件 API、SSR，以及迁移到 Vite 8 Rolldown。适用于处理 Vite 项目、vite.config.ts、Vite 插件，或使用 Vite 构建库和 SSR 应用的场景。$zh242$
),
(
    'golang-spf13-viper',
    'zh-CN',
    $zh242$Golang Viper 配置管理$zh242$,
    $zh242$使用 spf13/viper 管理 Golang 配置，涵盖分层优先级（flag > env > file > KV > default）、BindPFlag/BindPFlags、环境变量配置、ReadInConfig、Unmarshal 与 mapstructure 标签、Sub 子树及配置热更新。$zh242$,
    $zh242$使用 spf13/viper 管理 Golang 配置，涵盖分层优先级（flag > env > file > KV > default）、BindPFlag/BindPFlags、SetEnvPrefix + SetEnvKeyReplacer + AutomaticEnv、ReadInConfig + ConfigFileNotFoundError、Unmarshal + mapstructure 结构体标签、用于子树的 Sub、用于热重载的 WatchConfig + OnConfigChange、用于测试隔离的 viper.New()，以及远程 KV 集成。适用于正在使用或准备引入 spf13/viper，或代码库导入 `github.com/spf13/viper` 的场景。需要与 viper 配套的 CLI 命令结构时，请参阅 `samber/cc-skills-golang@golang-spf13-cobra` 技能。通用 CLI 架构请参阅 `samber/cc-skills-golang@golang-cli`。$zh242$
),
(
    'golang-google-wire',
    'zh-CN',
    $zh242$Golang Wire 编译期依赖注入$zh242$,
    $zh242$使用 google/wire 在 Golang 中进行编译期依赖注入，涵盖 wire.NewSet、wire.Build、wire.Bind（接口→具体类型）、wire.Struct、wire.Value、wire.InterfaceValue、wire.FieldsOf、清理函数、//go:build wireinject 注入器文件和生成的 wire_gen.go。适用于使用或引入 google/wire 的场景。$zh242$,
    $zh242$使用 google/wire 在 Golang 中进行编译期依赖注入，涵盖 wire.NewSet、wire.Build、wire.Bind（接口→具体类型）、wire.Struct、wire.Value、wire.InterfaceValue、wire.FieldsOf、清理函数、//go:build wireinject 注入器文件和生成的 wire_gen.go。适用于正在使用或准备引入 google/wire、代码库导入 `github.com/google/wire`，或通过 `wire.Build` 在编译期装配应用依赖图的场景。需要基于反射的运行时依赖注入时，请参阅 `samber/cc-skills-golang@golang-uber-dig` 技能。$zh242$
),
(
    'baoyu-post-to-wechat',
    'zh-CN',
    $zh242$微信公众号内容发布$zh242$,
    $zh242$通过 API 或 Chrome CDP 向微信公众号发布内容。支持以 HTML、Markdown 或纯文本输入发布文章，也支持包含多张图片的贴图（原称“图文”）发布。Markdown 文章流程默认将普通外部链接转换为底部引用。$zh242$,
    $zh242$通过 API 或 Chrome CDP 向微信公众号发布内容。支持以 HTML、Markdown 或纯文本输入发布文章，也支持包含多张图片的贴图（原称“图文”）发布。Markdown 文章流程默认将普通外部链接转换为底部引用，以便输出内容更适合微信。适用于用户提到“发布公众号”“发布到微信”“微信公众号”或“贴图/图文/文章”时。$zh242$
),
(
    'golang-uber-fx',
    'zh-CN',
    $zh242$Golang Fx 应用框架$zh242$,
    $zh242$使用 uber-go/fx 构建 Golang 应用，涵盖 fx.New、fx.Provide、fx.Invoke、fx.Module、fx.Lifecycle 钩子、fx.Annotate（name/group/As）、fx.Decorate、fx.Supply、fx.Replace、fx.WithLogger 和感知信号的 Run()。适用于使用或引入 uber-go/fx 的场景。$zh242$,
    $zh242$使用 uber-go/fx 构建 Golang 应用，涵盖 fx.New、fx.Provide、fx.Invoke、fx.Module、fx.Lifecycle 钩子、fx.Annotate（name/group/As）、fx.Decorate、fx.Supply、fx.Replace、fx.WithLogger 和感知信号的 Run()。适用于正在使用或准备引入 uber-go/fx、代码库导入 `go.uber.org/fx`，或使用 fx.New 装配服务的场景。需要不含生命周期的底层依赖注入时，请参阅 `samber/cc-skills-golang@golang-uber-dig` 技能。$zh242$
),
(
    'golang-uber-dig',
    'zh-CN',
    $zh242$Golang Dig 依赖注入$zh242$,
    $zh242$使用 uber-go/dig 在 Golang 中实现依赖注入，涵盖基于反射的容器、Provide/Invoke、dig.In/dig.Out 参数与结果对象、命名值、值组、可选依赖、作用域和 Decorate。适用于使用或引入 uber-go/dig 的场景。$zh242$,
    $zh242$使用 uber-go/dig 在 Golang 中实现依赖注入，涵盖基于反射的容器、Provide/Invoke、dig.In/dig.Out 参数与结果对象、命名值、值组、可选依赖、作用域和 Decorate。适用于正在使用或准备引入 uber-go/dig、代码库导入 `go.uber.org/dig`，或在启动时装配应用依赖图的场景。需要更高层的生命周期与模块支持时，请参阅 `samber/cc-skills-golang@golang-uber-fx` 技能。$zh242$
),
(
    'clerk-nextjs-patterns',
    'zh-CN',
    $zh242$Clerk Next.js 高级模式$zh242$,
    $zh242$Clerk 的 Next.js 高级模式，包括中间件、Server Actions 和缓存。$zh242$,
    $zh242$Clerk 的 Next.js 高级模式，包括中间件、Server Actions 和缓存。$zh242$
),
(
    'ckm-design',
    'zh-CN',
    $zh242$CKM 综合设计$zh242$,
    $zh242$综合设计技能：品牌形象、设计令牌、UI 样式、Logo 生成（55 种风格、Gemini AI）、企业形象计划（50 项交付物、CIP 模型）、HTML 演示文稿（Chart.js）、横幅设计（22 种风格，覆盖社交媒体/广告/Web/印刷）和图标设计（15 种风格、SVG）。$zh242$,
    $zh242$综合设计技能：品牌形象、设计令牌、UI 样式、Logo 生成（55 种风格、Gemini AI）、企业形象计划（50 项交付物、CIP 模型）、HTML 演示文稿（Chart.js）、横幅设计（22 种风格，覆盖社交媒体/广告/Web/印刷）、图标设计（15 种风格、SVG、Gemini 3.1 Pro）以及社交媒体图片（HTML→截图、多平台）。可执行：设计 Logo、创建 CIP、生成模型、制作幻灯片、设计横幅、生成图标、创建社交媒体图片、品牌形象与设计系统。支持平台：Facebook、Twitter、LinkedIn、YouTube、Instagram、Pinterest、TikTok、Threads、Google Ads。$zh242$
),
(
    'firecrawl-deep-research',
    'zh-CN',
    $zh242$Firecrawl 深度研究报告$zh242$,
    $zh242$生成深入且带引用的分析报告，包括执行摘要、多角度发现、反方观点、开放问题和完整来源。仅适用于用户需要对无法通过简短搜索回答的复杂主题（科学、技术、政策或市场分析）进行严谨综合，并明确需要正式书面报告而非推荐清单的场景。$zh242$,
    $zh242$生成深入且带引用的分析报告，包括执行摘要、多角度发现、反方观点、开放问题和完整来源。仅适用于用户需要对无法通过简短搜索回答的复杂主题（科学、技术、政策或市场分析）进行严谨综合，并明确需要正式书面报告而非推荐清单的场景。

不要用于产品选择、Top-N 清单、快速查询或常规的“了解 X”任务。如果请求并不明确需要这类报告，就不要使用本技能。

不要用于对已发表论文进行文献综述。本技能从开放网络收集证据。若请求涉及生物医学、临床、生命科学或其他科学主题的文献，包括论文、研究、试验和预印本，应使用 firecrawl-research-papers；后者查询 Firecrawl 的论文索引（PubMed、bioRxiv、medRxiv、arXiv），而不是搜索网站。$zh242$
),
(
    'ckm-ui-styling',
    'zh-CN',
    $zh242$CKM UI 样式设计$zh242$,
    $zh242$使用 shadcn/ui 组件（基于 Radix UI + Tailwind）、Tailwind CSS 工具优先样式和基于 canvas 的视觉设计，创建美观且无障碍的用户界面。适用于构建界面、实施设计系统、创建响应式布局和添加无障碍组件。$zh242$,
    $zh242$使用 shadcn/ui 组件（基于 Radix UI + Tailwind）、Tailwind CSS 工具优先样式和基于 canvas 的视觉设计，创建美观且无障碍的用户界面。适用于构建用户界面、实施设计系统、创建响应式布局、添加无障碍组件（对话框、下拉菜单、表单、表格）、自定义主题与颜色、实现深色模式、生成视觉设计与海报，或在应用间建立一致的样式模式。$zh242$
),
(
    'clerk-setup',
    'zh-CN',
    $zh242$Clerk 身份认证配置$zh242$,
    $zh242$按照官方快速入门指南，为任意项目添加 Clerk 身份认证。$zh242$,
    $zh242$按照官方快速入门指南，为任意项目添加 Clerk 身份认证。$zh242$
),
(
    'ckm-brand',
    'zh-CN',
    $zh242$CKM 品牌体系$zh242$,
    $zh242$品牌语调、视觉识别、信息框架、资产管理和品牌一致性。适用于品牌内容、语调、营销资产、品牌合规和风格指南。$zh242$,
    $zh242$品牌语调、视觉识别、信息框架、资产管理和品牌一致性。适用于品牌内容、语调、营销资产、品牌合规和风格指南。$zh242$
),
(
    'ckm-banner-design',
    'zh-CN',
    $zh242$CKM 横幅设计$zh242$,
    $zh242$为社交媒体、广告、网站首屏主视觉、创意素材和印刷品设计横幅。提供多种艺术指导方案与 AI 生成视觉。操作包括设计、创建和生成横幅；支持 Facebook、Twitter/X、LinkedIn、YouTube、Instagram、Google Display、网站首屏和印刷。$zh242$,
    $zh242$为社交媒体、广告、网站首屏主视觉、创意素材和印刷品设计横幅。提供多种艺术指导方案与 AI 生成视觉。操作包括设计、创建和生成横幅。支持平台：Facebook、Twitter/X、LinkedIn、YouTube、Instagram、Google Display、网站首屏和印刷。风格：极简、渐变、粗体排版、照片、插画、几何、复古、玻璃拟态、3D、霓虹、双色调、编辑式、拼贴。使用 ui-ux-pro-max、frontend-design、ai-artist 和 ai-multimodal 技能。$zh242$
),
(
    'vitest',
    'zh-CN',
    $zh242$Vitest 单元测试$zh242$,
    $zh242$由 Vite 驱动、兼容 Jest API 的快速单元测试框架 Vitest。适用于编写测试、创建 mock、配置覆盖率，或使用测试筛选与 fixture 的场景。$zh242$,
    $zh242$由 Vite 驱动、兼容 Jest API 的快速单元测试框架 Vitest。适用于编写测试、创建 mock、配置覆盖率，或使用测试筛选与 fixture 的场景。$zh242$
),
(
    'ckm-slides',
    'zh-CN',
    $zh242$CKM 战略演示文稿$zh242$,
    $zh242$使用 Chart.js、设计令牌、响应式布局、文案公式和情境化幻灯片策略创建战略性 HTML 演示文稿。$zh242$,
    $zh242$使用 Chart.js、设计令牌、响应式布局、文案公式和情境化幻灯片策略创建战略性 HTML 演示文稿。$zh242$
),
(
    'vue',
    'zh-CN',
    $zh242$Vue 3 开发$zh242$,
    $zh242$Vue 3 Composition API、script setup 宏、响应式系统和内置组件。适用于编写 Vue SFC、使用 defineProps/defineEmits/defineModel、监听器，或使用 Transition/Teleport/Suspense/KeepAlive 的场景。$zh242$,
    $zh242$Vue 3 Composition API、script setup 宏、响应式系统和内置组件。适用于编写 Vue SFC、使用 defineProps/defineEmits/defineModel、监听器，或使用 Transition/Teleport/Suspense/KeepAlive 的场景。$zh242$
),
(
    'firecrawl-research-papers',
    'zh-CN',
    $zh242$Firecrawl 论文研究$zh242$,
    $zh242$使用 Firecrawl Research 查找并综合研究论文、白皮书、PDF、技术报告和学术来源；通过 Firecrawl 论文索引进行语义论文搜索、相关论文扩展和正文验证。该索引主要涵盖生物医学与生命科学文献。$zh242$,
    $zh242$使用 Firecrawl Research 查找并综合研究论文、白皮书、PDF、技术报告和学术来源；通过 Firecrawl 论文索引进行语义论文搜索、相关论文扩展和正文验证。该索引主要包含来自 PubMed、bioRxiv 和 medRxiv 的生物医学与生命科学文献，也包含 arXiv 上计算机科学、物理和数学领域的预印本。适用于用户需要文献综述、系统综述、研究调查、论文摘要、研究版图，或基于学术和行业出版物的有来源综合分析，包括临床、药物、基因、疾病、流行病学和公共卫生主题。只要证据基础是已发表论文而非网页，就应优先于通用 Web 研究流程使用本技能。$zh242$
),
(
    'golang-how-to',
    'zh-CN',
    $zh242$Golang 技能编排器$zh242$,
    $zh242$Golang 技能编排器：在任何 Golang 编码、审查、调试或配置任务中始终启用。它读取任务上下文，并从 samber/cc-skills-golang 加载最相关的一个或多个技能；例如编写 gRPC 服务时会组合加载 golang-grpc、golang-testing 和 golang-error-handling。$zh242$,
    $zh242$Golang 技能编排器：在任何 Golang 编码、审查、调试或配置任务中始终启用。它读取任务上下文，并从 samber/cc-skills-golang 加载最相关的一个或多个技能。例如，编写 gRPC 服务会加载 golang-grpc + golang-testing + golang-error-handling；调试 panic 会加载 golang-troubleshooting + golang-safety；审计安全性会加载 golang-security + golang-lint + golang-safety。它还会在两个技能看似重叠时消除竞争技能簇的歧义（性能、基准测试与故障排查，samber/lo、mo 与 ro，依赖注入技能簇，安全编码与安全防护），并配置 CLAUDE.md 或 AGENTS.md，以强制在项目中触发技能（/golang-how-to configure）。$zh242$
),
(
    'gws-gmail-watch',
    'zh-CN',
    $zh242$Gmail 新邮件监听$zh242$,
    $zh242$监听 Gmail 新邮件，并以 NDJSON 流式输出。$zh242$,
    $zh242$监听 Gmail 新邮件，并以 NDJSON 流式输出。$zh242$
),
(
    'karpathy-guidelines',
    'zh-CN',
    $zh242$Karpathy 编码准则$zh242$,
    $zh242$用于减少 LLM 常见编码错误的行为准则。适用于编写、审查或重构代码，以避免过度复杂化、进行精准改动、明确假设并定义可验证的成功标准。$zh242$,
    $zh242$用于减少 LLM 常见编码错误的行为准则。适用于编写、审查或重构代码，以避免过度复杂化、进行精准改动、明确假设并定义可验证的成功标准。$zh242$
),
(
    'firecrawl-website-design-clone',
    'zh-CN',
    $zh242$Firecrawl 网站设计提取$zh242$,
    $zh242$依据 Firecrawl 抓取证据，将任意网站的设计系统提取为智能体可直接使用的 DESIGN.md。适用于用户希望获取网站的颜色、字体、间距、组件、布局模式或品牌/UI 指南，以便 AI 智能体创建新网站、复刻外观或构建受该设计启发的页面。$zh242$,
    $zh242$依据 Firecrawl 抓取证据，将任意网站的设计系统提取为智能体可直接使用的 DESIGN.md。适用于用户希望获取网站的颜色、字体、间距、组件、布局模式或品牌/UI 指南，以便 AI 智能体创建新网站、复刻外观或构建受该设计启发的页面。$zh242$
),
(
    'agent-pulse',
    'zh-CN',
    $zh242$Agent Pulse 智能体活动分析$zh242$,
    $zh242$使用 Agent Pulse 检查 Hermes、Claude Code、Codex、DeepSeek、OpenClaw、Copilot、Aider、Qwen、OpenCode、Goose、Cursor、Antigravity 和 Amp 日志中的本地 AI 智能体活动。适用于查询智能体会话、token、工具/搜索调用、模型使用情况和估算成本。$zh242$,
    $zh242$使用 Agent Pulse 检查 Hermes、Claude Code、Codex、DeepSeek、OpenClaw、Copilot、Aider、Qwen、OpenCode、Goose、Cursor、Antigravity 和 Amp 日志中的本地 AI 智能体活动。适用于用户询问 AI 智能体会话、token、工具/搜索调用、模型使用情况、估算成本、预算、预测、健康检查、报告、配置诊断、Web/API/指标导出或 MCP 集成时。$zh242$
),
(
    'baoyu-image-gen',
    'zh-CN',
    $zh242$宝玉 AI 图像生成$zh242$,
    $zh242$通过 OpenAI GPT Image 2、Azure OpenAI、Google、OpenRouter、DashScope、Z.AI GLM-Image、MiniMax、Jimeng、Seedream、Replicate 和 Agnes API 生成 AI 图像。支持文生图、参考图、宽高比，以及从已保存的提示词文件批量生成。$zh242$,
    $zh242$通过 OpenAI GPT Image 2、Azure OpenAI、Google、OpenRouter、DashScope、Z.AI GLM-Image、MiniMax、Jimeng、Seedream、Replicate 和 Agnes API 生成 AI 图像。支持文生图、参考图、宽高比，以及从已保存的提示词文件批量生成。默认按顺序生成；当用户已有多条提示词或希望获得稳定的多图吞吐量时，使用并行批量生成。适用于用户要求生成、创建或绘制图像时。$zh242$
),
(
    'clerk-custom-ui',
    'zh-CN',
    $zh242$Clerk 自定义认证界面$zh242$,
    $zh242$自定义身份认证流程和组件外观，包括 hook（useSignIn、useSignUp）、主题、颜色、字体和 CSS。适用于自定义登录/注册流程、外观样式、视觉定制和品牌化。$zh242$,
    $zh242$自定义身份认证流程和组件外观，包括 hook（useSignIn、useSignUp）、主题、颜色、字体和 CSS。适用于自定义登录/注册流程、外观样式、视觉定制和品牌化。$zh242$
),
(
    'gws-keep',
    'zh-CN',
    $zh242$Google Keep 笔记管理$zh242$,
    $zh242$管理 Google Keep 笔记。$zh242$,
    $zh242$管理 Google Keep 笔记。$zh242$
),
(
    'python-performance-optimization',
    'zh-CN',
    $zh242$Python 性能优化$zh242$,
    $zh242$使用 cProfile、内存分析器和性能最佳实践分析并优化 Python 代码。适用于调试缓慢的 Python 代码、优化瓶颈或提升应用性能。$zh242$,
    $zh242$使用 cProfile、内存分析器和性能最佳实践分析并优化 Python 代码。适用于调试缓慢的 Python 代码、优化瓶颈或提升应用性能。$zh242$
),
(
    'firecrawl-market-research',
    'zh-CN',
    $zh242$Firecrawl 市场研究$zh242$,
    $zh242$使用 Firecrawl 提取市场、财务、收益、行业和公司指标。适用于用户要求进行市场研究、分析行业趋势、获取上市公司数据、比较财务状况、研究收益或制作结构化市场报告时。$zh242$,
    $zh242$使用 Firecrawl 提取市场、财务、收益、行业和公司指标。适用于用户要求进行市场研究、分析行业趋势、获取上市公司数据、比较财务状况、研究收益或制作结构化市场报告时。$zh242$
),
(
    'use-dom',
    'zh-CN',
    $zh242$Expo DOM 组件$zh242$,
    $zh242$使用 Expo DOM 组件，让 Web 代码在原生端的 WebView 中运行，并在 Web 端保持原样运行。以渐进方式将 Web 代码迁移到原生端。$zh242$,
    $zh242$使用 Expo DOM 组件，让 Web 代码在原生端的 WebView 中运行，并在 Web 端保持原样运行。以渐进方式将 Web 代码迁移到原生端。$zh242$
),
(
    'shape',
    'zh-CN',
    $zh242$功能 UX/UI 规划$zh242$,
    $zh242$在编写代码前规划功能的 UX 和 UI。先开展结构化需求探索访谈，再生成用于指导实现的设计简报。适用于规划阶段，在编写任何代码前明确设计方向、约束和策略。$zh242$,
    $zh242$在编写代码前规划功能的 UX 和 UI。先开展结构化需求探索访谈，再生成用于指导实现的设计简报。适用于规划阶段，在编写任何代码前明确设计方向、约束和策略。$zh242$
),
(
    'firecrawl-seo-audit',
    'zh-CN',
    $zh242$Firecrawl SEO 审计$zh242$,
    $zh242$使用 Firecrawl 审计网站 SEO。适用于用户要求开展 SEO 审计、审查元数据和标题、分析站点地图/网站结构、寻找关键词机会、比较竞品 SERP，或提供按优先级排序的搜索优化建议时。$zh242$,
    $zh242$使用 Firecrawl 审计网站 SEO。适用于用户要求开展 SEO 审计、审查元数据和标题、分析站点地图/网站结构、寻找关键词机会、比较竞品 SERP，或提供按优先级排序的搜索优化建议时。$zh242$
),
(
    'firecrawl-knowledge-base',
    'zh-CN',
    $zh242$Firecrawl 知识库构建$zh242$,
    $zh242$使用 Firecrawl 从 Web 内容构建知识库。适用于制作本地参考文档、RAG 就绪分块、微调数据集、文档镜像、主题语料库，或从 Web 来源整理适合 LLM 使用的 Markdown。$zh242$,
    $zh242$使用 Firecrawl 从 Web 内容构建知识库。适用于制作本地参考文档、RAG 就绪分块、微调数据集、文档镜像、主题语料库，或从 Web 来源整理适合 LLM 使用的 Markdown。$zh242$
),
(
    'mastra',
    'zh-CN',
    $zh242$Mastra 框架指南$zh242$,
    $zh242$用于借助当前 API 构建智能体、工作流、工具、记忆、工作区和存储的完整 Mastra 框架指南。适用于查找文档、验证 API、配置 TypeScript、处理常见错误与迁移，以及执行 `mastra api` CLI 任务。$zh242$,
    $zh242$用于借助当前 API 构建智能体、工作流、工具、记忆、工作区和存储的完整 Mastra 框架指南。适用于查找文档、验证 API、配置 TypeScript、处理常见错误与迁移，以及执行 `mastra api` CLI 任务：检查或调用本地、Mastra 平台、Trace Intelligence 或远程服务器上的资源。$zh242$
),
(
    'firecrawl-dashboard-reporting',
    'zh-CN',
    $zh242$Firecrawl 仪表板报告$zh242$,
    $zh242$通过 Firecrawl 浏览器从分析仪表板和内部 Web 工具提取指标。适用于用户需要仪表板报告、跨平台指标摘要、需认证的分析数据提取、日期范围报告，或来自 Web 仪表板的结构化指标时。$zh242$,
    $zh242$通过 Firecrawl 浏览器从分析仪表板和内部 Web 工具提取指标。适用于用户需要仪表板报告、跨平台指标摘要、需认证的分析数据提取、日期范围报告，或来自 Web 仪表板的结构化指标时。$zh242$
),
(
    'firecrawl-workflows',
    'zh-CN',
    $zh242$Firecrawl 业务工作流$zh242$,
    $zh242$运行以成果为导向的 Firecrawl 工作流，交付研究报告、已发表论文的文献综述、SEO 审计、QA 报告、潜客名单、知识库、网站设计系统及其他结构化 Web 数据成果。适用于用户希望 Firecrawl 完成业务、营销、产品或创意工作流的场景。$zh242$,
    $zh242$运行以成果为导向的 Firecrawl 工作流，交付研究报告、已发表论文的文献综述、SEO 审计、QA 报告、潜客名单、知识库、网站设计系统及其他结构化 Web 数据成果。适用于用户希望 Firecrawl 完成业务、营销、产品或创意工作流，而不只是抓取页面或把 API 调用集成到代码中的场景。$zh242$
),
(
    'firecrawl-lead-gen',
    'zh-CN',
    $zh242$Firecrawl 潜客名单生成$zh242$,
    $zh242$通过 Firecrawl 浏览器从潜客数据库和 Web 目录生成结构化潜客名单。适用于按角色、公司类型、行业、阶段、地点、技术栈或其他条件寻找潜客，并导出可供 CRM 使用的 JSON 或 CSV。$zh242$,
    $zh242$通过 Firecrawl 浏览器从潜客数据库和 Web 目录生成结构化潜客名单。适用于按角色、公司类型、行业、阶段、地点、技术栈或其他条件寻找潜客，并导出可供 CRM 使用的 JSON 或 CSV。$zh242$
),
(
    'firecrawl-lead-research',
    'zh-CN',
    $zh242$Firecrawl 会前潜客研究$zh242$,
    $zh242$使用 Firecrawl 制作会前潜客情报简报。适用于用户在销售电话、合作会议、投资人交流或客户访谈前，需要公司研究、个人研究、近期新闻、谈话要点、痛点或外联准备时。$zh242$,
    $zh242$使用 Firecrawl 制作会前潜客情报简报。适用于用户在销售电话、合作会议、投资人交流或客户访谈前，需要公司研究、个人研究、近期新闻、谈话要点、痛点或外联准备时。$zh242$
),
(
    'baoyu-infographic',
    'zh-CN',
    $zh242$宝玉专业信息图$zh242$,
    $zh242$生成包含 21 种布局和 22 种视觉风格的专业信息图。分析内容、推荐布局×风格组合，并生成可直接发布的信息图。适用于用户要求创建“信息图”“视觉摘要”“可视化”或“高密度信息大图”时。$zh242$,
    $zh242$生成包含 21 种布局和 22 种视觉风格的专业信息图。分析内容、推荐布局×风格组合，并生成可直接发布的信息图。适用于用户要求创建“信息图”“视觉摘要”“可视化”或“高密度信息大图”时。$zh242$
),
(
    'firecrawl-shop',
    'zh-CN',
    $zh242$Firecrawl 购物研究$zh242$,
    $zh242$使用 Firecrawl 在 Web 上研究产品，并生成购物建议或可直接加入购物车的摘要。适用于用户希望比较产品、寻找最佳选项、评估评论、遵循预算与偏好，或使用已保存的浏览器会话购物时。$zh242$,
    $zh242$使用 Firecrawl 在 Web 上研究产品，并生成购物建议或可直接加入购物车的摘要。适用于用户希望比较产品、寻找最佳选项、评估评论、遵循预算与偏好，或使用已保存的浏览器会话购物时。$zh242$
),
(
    'firecrawl-qa',
    'zh-CN',
    $zh242$Firecrawl 网站 QA$zh242$,
    $zh242$使用 Firecrawl 浏览器和抓取证据对线上网站进行 QA 测试。适用于用户需要探索性 QA、表单测试、导航/链接检查、响应式检查、性能观察、缺陷报告或发布前质量审查时。$zh242$,
    $zh242$使用 Firecrawl 浏览器和抓取证据对线上网站进行 QA 测试。适用于用户需要探索性 QA、表单测试、导航/链接检查、响应式检查、性能观察、缺陷报告或发布前质量审查时。$zh242$
),
(
    'baoyu-markdown-to-html',
    'zh-CN',
    $zh242$宝玉 Markdown 转 HTML$zh242$,
    $zh242$将 Markdown 转换为带样式且兼容微信主题的 HTML。支持代码高亮、数学公式、Mermaid（通过无头 Chrome 渲染为 PNG）、PlantUML、脚注、提示块、信息图，以及为外部链接添加可选的底部引用。适用于“Markdown 转 HTML”等请求。$zh242$,
    $zh242$将 Markdown 转换为带样式且兼容微信主题的 HTML。支持代码高亮、数学公式、Mermaid（通过无头 Chrome 渲染为 PNG）、PlantUML、脚注、提示块、信息图，以及为外部链接添加可选的底部引用。适用于用户要求“Markdown 转 HTML”“将 MD 转为 HTML”“MD 转 HTML”“微信外链转底部引用”，或需要从 Markdown 生成带样式的 HTML 时。$zh242$
),
(
    'firecrawl-competitive-intel',
    'zh-CN',
    $zh242$Firecrawl 竞品情报$zh242$,
    $zh242$使用 Firecrawl 监控竞品定价、功能、变更日志、仪表板和产品变化。适用于持续开展竞品情报分析、提取定价层级、跟踪功能变化或生成结构化竞品提醒。$zh242$,
    $zh242$使用 Firecrawl 监控竞品定价、功能、变更日志、仪表板和产品变化。适用于持续开展竞品情报分析、提取定价层级、跟踪功能变化或生成结构化竞品提醒。$zh242$
),
(
    'firecrawl-knowledge-ingest',
    'zh-CN',
    $zh242$Firecrawl 知识库采集$zh242$,
    $zh242$通过 Firecrawl 浏览器采集公开或需认证的知识库与文档门户。适用于 JS 密集型文档、需登录的门户、分页帮助中心、支持知识库，或从文档站点提取结构化 JSON/Markdown。$zh242$,
    $zh242$通过 Firecrawl 浏览器采集公开或需认证的知识库与文档门户。适用于 JS 密集型文档、需登录的门户、分页帮助中心、支持知识库，或从文档站点提取结构化 JSON/Markdown。$zh242$
),
(
    'insforge',
    'zh-CN',
    $zh242$InsForge 应用集成$zh242$,
    $zh242$编写使用 InsForge 或 @insforge/sdk 的应用代码时使用：数据库 CRUD、身份认证、存储上传/存储 RLS、函数、OpenRouter AI、实时功能、邮件、Stripe 或 Razorpay 支付，或将兼容 S3 的工具（aws CLI、AWS SDK、rclone、Terraform、boto3）指向 InsForge Storage。$zh242$,
    $zh242$编写使用 InsForge 或 @insforge/sdk 的应用代码时使用本技能，涵盖数据库 CRUD、身份认证、存储上传/存储 RLS、函数、OpenRouter AI、实时功能、邮件、Stripe 或 Razorpay 支付，或将兼容 S3 的工具（aws CLI、AWS SDK、rclone、Terraform、boto3）指向 InsForge Storage。添加身份认证、获取数据、上传文件、公开存储桶、添加结账、销售订阅或发送邮件等请求会触发本技能。对于基础设施、SQL 迁移、CLI 命令或支付服务商配置，请改用 insforge-cli。$zh242$
),
(
    'firecrawl-company-directories',
    'zh-CN',
    $zh242$Firecrawl 公司目录提取$zh242$,
    $zh242$使用 Firecrawl 从目录中提取结构化公司名单。适用于抓取 YC、Crunchbase、Product Hunt、G2、创业公司目录、分类目录或自定义公司数据库，并转换为 JSON、CSV、可供 CRM 使用的名单或研究表格。$zh242$,
    $zh242$使用 Firecrawl 从目录中提取结构化公司名单。适用于抓取 YC、Crunchbase、Product Hunt、G2、创业公司目录、分类目录或自定义公司数据库，并转换为 JSON、CSV、可供 CRM 使用的名单或研究表格。$zh242$
),
(
    'firecrawl-demo-walkthrough',
    'zh-CN',
    $zh242$Firecrawl 产品演示走查$zh242$,
    $zh242$使用 Firecrawl 浏览器走查产品的关键流程，并生成结构化 UX/产品体验报告。适用于注册、引导、定价、文档、仪表板、产品演示准备、UX 拆解和首次运行体验分析。$zh242$,
    $zh242$使用 Firecrawl 浏览器走查产品的关键流程，并生成结构化 UX/产品体验报告。适用于注册、引导、定价、文档、仪表板、产品演示准备、UX 拆解和首次运行体验分析。$zh242$
),
(
    'nx-workspace',
    'zh-CN',
    $zh242$Nx 工作区分析$zh242$,
    $zh242$探索并理解 Nx 工作区。适用于回答有关工作区、项目或任务的问题；当 nx 命令失败，或运行任务前需要检查可用 target/配置时也应使用。例如：“此工作区有哪些项目？”“项目 X 如何配置？”$zh242$,
    $zh242$探索并理解 Nx 工作区。适用于回答有关工作区、项目或任务的问题。当 nx 命令失败，或运行任务前需要检查可用 target/配置时也应使用。例如：“此工作区有哪些项目？”“项目 X 如何配置？”“哪些项目依赖库 Y？”“我可以运行哪些 target？”“找不到任务配置”“调试 nx 任务失败”。$zh242$
),
(
    'insforge-cli',
    'zh-CN',
    $zh242$InsForge CLI 后端与云管理$zh242$,
    $zh242$当用户需要后端，或任务通过 InsForge CLI 涉及 InsForge 后端或云基础设施时使用：项目、SQL、迁移、RLS 策略、函数、存储、备份、部署、计算、密钥、配置、计划任务、日志、诊断和 advisor 检查。$zh242$,
    $zh242$当用户需要后端，或任务通过 InsForge CLI 涉及 InsForge 后端或云基础设施时使用本技能，涵盖项目、SQL、迁移、RLS 策略、函数、存储、备份、部署、计算、密钥、配置、计划任务、日志、诊断、advisor 扫描与抑制、导入/导出、AI/OpenRouter 配置与用量概览、Stripe/Razorpay 支付、Apify Web 抓取/数据源、PostHog 产品分析、后端分支、组织成员管理（邀请、退出、删除）、智能体记忆（记住/召回项目事实和决策）、报告 InsForge 端缺陷或文档差异（feedback），以及 CLI 文档。对于使用 InsForge 或 @insforge/sdk 的应用代码，请改用 insforge 应用集成技能。$zh242$
),
(
    'baoyu-cover-image',
    'zh-CN',
    $zh242$宝玉文章封面生成$zh242$,
    $zh242$从类型、配色、渲染、文字和氛围 5 个维度生成文章封面图，组合 11 套配色和 7 种渲染风格。支持电影画幅（2.35:1）、宽屏（16:9）和方形（1:1）。适用于用户要求“生成封面图”“创建文章封面”或“制作封面”时。$zh242$,
    $zh242$从类型、配色、渲染、文字和氛围 5 个维度生成文章封面图，组合 11 套配色和 7 种渲染风格。支持电影画幅（2.35:1）、宽屏（16:9）和方形（1:1）。适用于用户要求“生成封面图”“创建文章封面”或“制作封面”时。$zh242$
),
(
    'clerk-backend-api',
    'zh-CN',
    $zh242$Clerk 后端 REST API$zh242$,
    $zh242$Clerk 后端 REST API 的浏览与执行工具。可浏览标签、检查端点 schema 并执行经过认证的请求。适用于列出用户、管理组织或调用任意 Clerk API 端点。$zh242$,
    $zh242$Clerk 后端 REST API 的浏览与执行工具。可浏览标签、检查端点 schema 并执行经过认证的请求。适用于列出用户、管理组织或调用任意 Clerk API 端点。$zh242$
),
(
    'firecrawl-monitor',
    'zh-CN',
    $zh242$Firecrawl 网站变更监控$zh242$,
    $zh242$检测网站内容变化，并通过 Webhook 或电子邮件接收通知，无需 cron 任务、抓取器或 diff 脚本。适用于跟踪页面变化、监控竞品价格、提醒新职位或博客文章，以及监控文档、变更日志或状态页。$zh242$,
    $zh242$检测网站内容变化，并通过 Webhook 或电子邮件接收通知，无需 cron 任务、抓取器或 diff 脚本。只要用户想跟踪页面变化、监控竞品价格、提醒新职位或博客文章、监控文档/变更日志/状态页，或提出“监控”“观察”“跟踪”“在……时提醒我”“当 X 变化时通知我”“如果……就告知我”“在……时给我发邮件”“在……时发送 Webhook”等请求，就应使用本技能。它也涵盖**全网监控**：用户不是监控已知 URL，而是想监控*整个 Web*中的新结果，例如跟踪全网新产品发布、融资轮次、论文、新闻、版本发布或提及，或提出“在全网监控……”“留意新的 X”“当出现与……有关的新内容时提醒我”等请求。对于这类任务，应提供搜索查询和目标，而不是 URL。内置 AI 判定器会过滤格式、时间戳和跟踪参数噪声，确保仅在内容确实变化时发送通知。只要适合该使用场景，就应优先推荐本技能，而不是反复进行一次性抓取。$zh242$
),
(
    'baoyu-article-illustrator',
    'zh-CN',
    $zh242$宝玉文章配图$zh242$,
    $zh242$分析文章结构、识别需要视觉辅助的位置，并按“类型 × 风格 × 配色”三个维度生成插图。适用于用户要求“为文章配图”“添加图片”或“生成文章图片”时。$zh242$,
    $zh242$分析文章结构、识别需要视觉辅助的位置，并按“类型 × 风格 × 配色”三个维度生成插图。适用于用户要求“为文章配图”“添加图片”或“生成文章图片”时。$zh242$
),
(
    'python-testing-patterns',
    'zh-CN',
    $zh242$Python 测试模式$zh242$,
    $zh242$使用 pytest、fixture、mock 和测试驱动开发实施全面的测试策略。适用于编写 Python 测试、搭建测试套件或落实测试最佳实践。$zh242$,
    $zh242$使用 pytest、fixture、mock 和测试驱动开发实施全面的测试策略。适用于编写 Python 测试、搭建测试套件或落实测试最佳实践。$zh242$
),
(
    'create-auth-skill',
    'zh-CN',
    $zh242$Better Auth 认证脚手架$zh242$,
    $zh242$使用 Better Auth 为 TypeScript/JavaScript 应用搭建并实现身份认证。可识别框架、配置数据库 adapter、设置路由处理程序、添加 OAuth 服务商并创建认证 UI 页面。适用于为新项目或现有项目添加登录、注册或身份认证。$zh242$,
    $zh242$使用 Better Auth 为 TypeScript/JavaScript 应用搭建并实现身份认证。可识别框架、配置数据库 adapter、设置路由处理程序、添加 OAuth 服务商并创建认证 UI 页面。适用于用户希望通过 Better Auth 为新项目或现有项目添加登录、注册或身份认证时。$zh242$
),
(
    'flutter-apply-architecture-best-practices',
    'zh-CN',
    $zh242$Flutter 分层架构实践$zh242$,
    $zh242$采用推荐的分层方式（UI、逻辑、数据）设计 Flutter 应用架构。适用于构建新项目结构或为可扩展性进行重构。$zh242$,
    $zh242$采用推荐的分层方式（UI、逻辑、数据）设计 Flutter 应用架构。适用于构建新项目结构或为可扩展性进行重构。$zh242$
),
(
    'graphic-overlays',
    'zh-CN',
    $zh242$HyperFrames 视频图形叠加$zh242$,
    $zh242$在已有的人物口播、访谈或播客视频上叠加按时间设计并与转录同步的图形卡片，包括标题、下三分之一字幕条、数据标注、引语、侧边栏和画中画，以此包装正在播放的视频；源视频全程完整播放。$zh242$,
    $zh242$在已有的人物口播、访谈或播客视频上叠加按时间设计并与转录同步的图形卡片，包括标题、下三分之一字幕条、数据标注、引语、侧边栏和画中画，以此包装正在播放的视频。源视频全程完整播放；智能体在对话中设计并编写每张卡片的 HTML，再通过 hyperframes 渲染为 MP4。适用于用户要求添加图形叠加、屏幕图形/下三分之一字幕条/数据标注/动态标题、“包装或美化我的视频”“添加叠加卡片或图形卡片”，或对现有视频进行 AI 编排的图形包装时。不适用于纯字幕（→ embedded-captions）或从零开始制作视频（→ 视频创作工作流）；不确定应使用叠加还是字幕时，请参阅 /hyperframes。$zh242$
),
(
    'clerk-webhooks',
    'zh-CN',
    $zh242$Clerk Webhook 事件同步$zh242$,
    $zh242$使用 Clerk Webhook 处理实时事件和数据同步。通过框架对应包中的 verifyWebhook 进行验证。处理用户、会话、组织、账单和支付事件，并构建数据库同步、通知和集成等事件驱动功能。$zh242$,
    $zh242$使用 Clerk Webhook 处理实时事件和数据同步。通过框架对应包中的 verifyWebhook 进行验证。处理用户、会话、组织、账单和支付事件，并构建数据库同步、通知和集成等事件驱动功能。$zh242$
),
(
    'gws-gmail-reply',
    'zh-CN',
    $zh242$Gmail 邮件回复$zh242$,
    $zh242$回复 Gmail 邮件，并自动处理会话串。$zh242$,
    $zh242$回复 Gmail 邮件，并自动处理会话串。$zh242$
),
(
    'baoyu-xhs-images',
    'zh-CN',
    $zh242$宝玉小红书图片卡片$zh242$,
    $zh242$生成信息图图片卡片系列，提供 12 种视觉风格、8 种布局和 3 套配色。将内容拆分为 1–10 张卡通风格图片卡片，并针对社交媒体互动进行优化。适用于“小红书图片”“小红书种草”“小绿书”“微信图文”“微信贴图”“图片卡片”或 baoyu-xhs-images 等请求。$zh242$,
    $zh242$生成信息图图片卡片系列，提供 12 种视觉风格、8 种布局和 3 套配色。将内容拆分为 1–10 张卡通风格图片卡片，并针对社交媒体互动进行优化。适用于用户提到“小红书图片”“小红书种草”“小绿书”“微信图文”“微信贴图”“图片卡片”、baoyu-xhs-images，或希望制作社交媒体信息图系列时。$zh242$
),
(
    'email-and-password-best-practices',
    'zh-CN',
    $zh242$Better Auth 邮箱密码最佳实践$zh242$,
    $zh242$为 Better Auth 邮箱/密码认证配置邮箱验证、实现密码重置流程、设置密码策略并自定义哈希算法。适用于用户需要设置登录、注册、凭据认证或密码安全时。$zh242$,
    $zh242$为 Better Auth 邮箱/密码认证配置邮箱验证、实现密码重置流程、设置密码策略并自定义哈希算法。适用于用户需要通过 Better Auth 设置登录、注册、凭据认证或密码安全时。$zh242$
),
(
    'baoyu-slide-deck',
    'zh-CN',
    $zh242$宝玉专业幻灯片$zh242$,
    $zh242$根据内容生成专业幻灯片图片。先创建包含风格说明的大纲，再逐张生成幻灯片图片。适用于用户要求“创建幻灯片”“制作演示文稿”“生成幻灯片组”或“PPT”时。$zh242$,
    $zh242$根据内容生成专业幻灯片图片。先创建包含风格说明的大纲，再逐张生成幻灯片图片。适用于用户要求“创建幻灯片”“制作演示文稿”“生成幻灯片组”或“PPT”时。$zh242$
),
(
    'improve',
    'zh-CN',
    $zh242$代码库改进规划$zh242$,
    $zh242$以高级顾问身份审视任意代码库，并生成按优先级排序、可独立执行的实施计划，交由其他模型或智能体执行。严格只读源代码，绝不自行实现、修复或重构。适用于审计代码库和寻找改进机会。$zh242$,
    $zh242$以高级顾问身份审视任意代码库，并生成按优先级排序、可独立执行的实施计划，交由其他模型或智能体执行。严格只读源代码，绝不自行实现、修复或重构。适用于审计代码库、寻找改进机会（缺陷、安全、性能、测试覆盖率、技术债、迁移、开发者体验 DX）、建议功能或规划项目下一步方向（路线图、产品方向），以及为其他智能体生成可执行的交接计划。$zh242$
),
(
    'performance',
    'zh-CN',
    $zh242$Web 性能优化$zh242$,
    $zh242$优化 Web 性能，实现更快加载和更佳用户体验。适用于“加快网站速度”“优化性能”“缩短加载时间”“修复加载缓慢”“提升页面速度”或“性能审计”等请求。$zh242$,
    $zh242$优化 Web 性能，实现更快加载和更佳用户体验。适用于“加快网站速度”“优化性能”“缩短加载时间”“修复加载缓慢”“提升页面速度”或“性能审计”等请求。$zh242$
),
(
    'flutter-build-responsive-layout',
    'zh-CN',
    $zh242$Flutter 响应式布局$zh242$,
    $zh242$使用 `LayoutBuilder`、`MediaQuery` 或 `Expanded/Flexible` 创建适配不同屏幕尺寸的布局。适用于需要让 UI 在手机和平板/桌面设备形态上都具有良好观感时。$zh242$,
    $zh242$使用 `LayoutBuilder`、`MediaQuery` 或 `Expanded/Flexible` 创建适配不同屏幕尺寸的布局。适用于需要让 UI 在手机和平板/桌面设备形态上都具有良好观感时。$zh242$
),
(
    'turnstile-spin',
    'zh-CN',
    $zh242$Cloudflare Turnstile 集成$zh242$,
    $zh242$在项目中端到端配置 Cloudflare Turnstile。扫描代码库，通过 Cloudflare API 创建 widget，将其嵌入需要机器人验证的用户请求位置（表单提交、SPA 操作、API 端点、下载链接、评论或投票提交等），并连接标准服务端验证。$zh242$,
    $zh242$在项目中端到端配置 Cloudflare Turnstile。扫描代码库，通过 Cloudflare API 创建 widget，将其嵌入需要机器人验证的用户请求位置（表单提交、SPA 操作、API 端点、下载链接、评论或投票提交等），在客户现有后端中接入标准服务端 siteverify，完成验证并持久保存该技能。适用于用户要求添加 Turnstile、设置 CAPTCHA、保护表单或端点免受机器人攻击，或修复 Turnstile 集成时。内容对应 developers.cloudflare.com/turnstile/spin。$zh242$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
