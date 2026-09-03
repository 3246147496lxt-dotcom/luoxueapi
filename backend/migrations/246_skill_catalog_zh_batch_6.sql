-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 919-1000.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'dbs-slowisfast',
    'zh-CN',
    $zh246$DBS 慢即是快$zh246$,
    $zh246$识别关键环节中的贪快与必要摩擦，寻找能长期积累资产的方法。用户担心推进过快、反复返工或希望设计长期复利路径时使用。$zh246$,
    $zh246$识别关键环节中的贪快与必要摩擦，寻找能长期积累资产的方法。用户担心推进过快、反复返工或希望设计长期复利路径时使用。$zh246$
),
(
    'pinia',
    'zh-CN',
    $zh246$Pinia 状态管理$zh246$,
    $zh246$Pinia 是 Vue 官方状态管理库，类型安全且可扩展。适用于定义 Store、处理 state/getter/action，或在 Vue 应用中实现 Store 模式。$zh246$,
    $zh246$Pinia 是 Vue 官方状态管理库，类型安全且可扩展。定义 Store、处理 state/getter/action，或在 Vue 应用中实现 Store 模式时使用。$zh246$
),
(
    'dbs-chatroom-austrian',
    'zh-CN',
    $zh246$DBS 奥派聊天室$zh246$,
    $zh246$由哈耶克、米塞斯与 Claude 从奥派经济学视角展开多角色讨论。用户要求进入奥派聊天室或用奥派分析问题时使用。$zh246$,
    $zh246$由哈耶克、米塞斯与 Claude 从奥派经济学视角展开多角色讨论。用户要求进入奥派聊天室或用奥派分析问题时使用。$zh246$
),
(
    'sql-optimization-patterns',
    'zh-CN',
    $zh246$SQL 优化模式$zh246$,
    $zh246$掌握 SQL 查询优化、索引策略与 EXPLAIN 分析，显著提升数据库性能并消除慢查询。适用于调试慢查询、设计数据库架构或优化应用性能。$zh246$,
    $zh246$掌握 SQL 查询优化、索引策略与 EXPLAIN 分析，显著提升数据库性能并消除慢查询。调试慢查询、设计数据库架构或优化应用性能时使用。$zh246$
),
(
    'gpt-image-2-e2d4ead7e6',
    'zh-CN',
    $zh246$GPT Image 2 图像生成$zh246$,
    $zh246$完整覆盖兼容 OpenAI 的 GPT Image 2，包括 images/generations、images/edits，以及使用 image_generation 工具的 Responses；适用于文生图、蒙版编辑、多图批处理、流式输出、partial_images 和图文混合流程。$zh246$,
    $zh246$完整覆盖兼容 OpenAI 的 GPT Image 2，包括 images/generations、images/edits，以及使用 image_generation 工具的 Responses。当一次性图像辅助工具不足以满足需求时使用，涵盖文生图、蒙版编辑、多图批处理、流式输出、partial_images 和图文混合 Responses 流程。读取 .env 并遵循进程环境变量；可与任何兼容 OpenAI 的网关配合使用。$zh246$
),
(
    'planning-with-files-zh',
    'zh-CN',
    $zh246$中文文件式规划$zh246$,
    $zh246$基于 Manus 风格的文件规划系统，用于组织和跟踪复杂任务进度。创建 task_plan.md、findings.md 和 progress.md 三个文件。适用于规划、拆解或组织多步骤项目、研究任务或需要超过 5 次工具调用的工作，并支持 /clear 后自动恢复会话。$zh246$,
    $zh246$基于 Manus 风格的文件规划系统，用于组织和跟踪复杂任务的进度。创建 task_plan.md、findings.md 和 progress.md 三个文件。当用户要求规划、拆解或组织多步骤项目、研究任务，或需要超过 5 次工具调用的工作时使用。支持 /clear 后自动恢复会话。触发词：任务规划、项目计划、制定计划、分解任务、多步骤规划、进度跟踪、文件规划、帮我规划、拆解项目。$zh246$
),
(
    'golang-pro',
    'zh-CN',
    $zh246$Go 专家开发$zh246$,
    $zh246$使用 goroutine 与 channel 实现 Go 并发模式，通过 gRPC 或 REST 设计和构建微服务，借助 pprof 优化性能，并以泛型、接口和稳健错误处理落实惯用 Go 风格。$zh246$,
    $zh246$使用 goroutine 与 channel 实现 Go 并发模式，通过 gRPC 或 REST 设计和构建微服务，借助 pprof 优化 Go 应用性能，并以泛型、接口和稳健的错误处理落实惯用 Go 风格。构建需要并发编程、微服务架构或高性能系统的 Go 应用时使用。遇到 goroutine、channel、Go 泛型、gRPC 集成、CLI 工具、基准测试或表驱动测试时调用。$zh246$
),
(
    'skill-development',
    'zh-CN',
    $zh246$Skill 开发$zh246$,
    $zh246$当用户希望创建 Skill、向插件添加 Skill、编写新 Skill、改进 Skill 说明、组织 Skill 内容，或需要了解 Skill 结构、渐进式披露及 Claude Code 插件 Skill 开发最佳实践时使用。$zh246$,
    $zh246$当用户希望“创建 Skill”“向插件添加 Skill”“编写新 Skill”“改进 Skill 说明”“组织 Skill 内容”，或需要 Skill 结构、渐进式披露及 Claude Code 插件 Skill 开发最佳实践方面的指导时，应使用本 Skill。$zh246$
),
(
    'dbs-agent-migration',
    'zh-CN',
    $zh246$DBS Agent 迁移$zh246$,
    $zh246$审计项目规则文件、识别真源、统一命名并生成桥接，把项目迁移成多端一致的 Agent 工作台。用户要求迁移 Claude Code、Codex、Grok、通用 Agent 或整理 AGENTS.md 时使用。$zh246$,
    $zh246$审计项目规则文件、识别真源、统一命名并生成桥接，把项目迁移成多端一致的 Agent 工作台。用户要求迁移 Claude Code、Codex、Grok、通用 Agent 或整理 AGENTS.md 时使用。$zh246$
),
(
    'marketing-council',
    'zh-CN',
    $zh246$营销顾问委员会$zh246$,
    $zh246$针对营销问题提供多位专家的模拟顾问委员会视角，成员包括 Seth Godin、David Ogilvy、Eugene Schwartz、April Dunford、Rory Sutherland、Alex Hormozi、Byron Sharp 等传奇营销人。$zh246$,
    $zh246$当用户希望就营销问题获得多位专家视角时使用：它模拟一个由 Seth Godin、David Ogilvy、Eugene Schwartz、April Dunford、Rory Sutherland、Alex Hormozi、Byron Sharp 等传奇营销人组成的顾问委员会。用户提到“营销委员会”“顾问委员会”“咨询委员会”“Seth Godin 会怎么说”“Ogilvy 会怎么想”“代入 Hormozi”“获得多个视角”“辩论一下”“让委员会审查”“营销导师”，或询问某位著名营销人会如何处理其问题时也应使用。委员会会依据每位顾问有据可查的方法论分别给出观点，指出彼此分歧并综合形成建议。执行胜出方向时，请转交给 positioning、offers、copywriting、ads 或其他相关 Skill。$zh246$
),
(
    'security-requirement-extraction',
    'zh-CN',
    $zh246$安全需求提取$zh246$,
    $zh246$从威胁模型和业务背景中推导安全需求。适用于把威胁转化为可执行需求、创建安全用户故事或构建安全测试用例。$zh246$,
    $zh246$从威胁模型和业务背景中推导安全需求。把威胁转化为可执行需求、创建安全用户故事或构建安全测试用例时使用。$zh246$
),
(
    'banner-design',
    'zh-CN',
    $zh246$横幅设计$zh246$,
    $zh246$为社交媒体、广告、网站首屏、创意素材和印刷品设计横幅，提供多种艺术指导方案与 AI 生成视觉内容。支持 Facebook、Twitter/X、LinkedIn、YouTube、Instagram、Google Display 等平台。$zh246$,
    $zh246$为社交媒体、广告、网站首屏、创意素材和印刷品设计横幅，提供多种艺术指导方案与 AI 生成视觉内容。操作：设计、创建、生成横幅。平台：Facebook、Twitter/X、LinkedIn、YouTube、Instagram、Google Display、网站首屏、印刷品。风格：极简、渐变、醒目排版、照片型、插画型、几何、复古、玻璃拟态、3D、霓虹、双色调、编辑设计、拼贴。使用 ui-ux-pro-max、frontend-design、ai-artist 和 ai-multimodal Skill。$zh246$
),
(
    'brand',
    'zh-CN',
    $zh246$品牌管理$zh246$,
    $zh246$涵盖品牌语调、视觉识别、信息框架、素材管理和品牌一致性。适用于品牌内容、语气风格、营销素材、品牌合规和风格指南。$zh246$,
    $zh246$涵盖品牌语调、视觉识别、信息框架、素材管理和品牌一致性。处理品牌内容、语气风格、营销素材、品牌合规或风格指南时启用。$zh246$
),
(
    'responsive-design',
    'zh-CN',
    $zh246$响应式设计$zh246$,
    $zh246$使用容器查询、流式排版、CSS Grid 和移动优先断点策略实现现代响应式布局。适用于构建自适应界面、实现流式布局或创建组件级响应行为。$zh246$,
    $zh246$使用容器查询、流式排版、CSS Grid 和移动优先断点策略实现现代响应式布局。构建自适应界面、实现流式布局或创建组件级响应行为时使用。$zh246$
),
(
    'fixing-accessibility',
    'zh-CN',
    $zh246$无障碍修复$zh246$,
    $zh246$审计并修复 HTML 无障碍问题，包括 ARIA 标签、键盘导航、焦点管理、颜色对比度和表单错误。适用于添加交互控件、表单、对话框或审查 WCAG 合规性。$zh246$,
    $zh246$审计并修复 HTML 无障碍问题，包括 ARIA 标签、键盘导航、焦点管理、颜色对比度和表单错误。添加交互控件、表单、对话框或审查 WCAG 合规性时使用。$zh246$
),
(
    'git-advanced-workflows',
    'zh-CN',
    $zh246$Git 高级工作流$zh246$,
    $zh246$掌握包括变基、Cherry-pick、二分查找、Worktree 和 Reflog 在内的高级 Git 工作流，以保持历史整洁并从各种异常中恢复。适用于复杂 Git 历史、功能分支协作和仓库故障排查。$zh246$,
    $zh246$掌握包括变基、Cherry-pick、二分查找、Worktree 和 Reflog 在内的高级 Git 工作流，以保持历史整洁并从任何异常中恢复。管理复杂 Git 历史、在功能分支上协作或排查仓库问题时使用。$zh246$
),
(
    'nuxt-ui',
    'zh-CN',
    $zh246$Nuxt UI 组件库$zh246$,
    $zh246$使用 @nuxt/ui v4 构建界面：提供 125 个以上具备无障碍能力的 Vue 组件，并支持 Tailwind CSS 主题。适用于创建界面、定制品牌主题、构建表单、仪表板、文档站和聊天界面。$zh246$,
    $zh246$使用 @nuxt/ui v4 构建界面：提供 125 个以上具备无障碍能力的 Vue 组件，并支持 Tailwind CSS 主题。创建界面、定制主题以匹配品牌、构建表单，或组合仪表板、文档站点和聊天界面等布局时使用。$zh246$
),
(
    'dotnet-backend-patterns',
    'zh-CN',
    $zh246$.NET 后端模式$zh246$,
    $zh246$掌握用于构建稳健 API、MCP 服务器和企业应用的 C#/.NET 后端开发模式，涵盖 async/await、依赖注入、Entity Framework Core、Dapper、配置、缓存和 xUnit 测试。$zh246$,
    $zh246$掌握用于构建稳健 API、MCP 服务器和企业应用的 C#/.NET 后端开发模式。涵盖 async/await、依赖注入、Entity Framework Core、Dapper、配置、缓存和使用 xUnit 进行测试。开发 .NET 后端、审查 C# 代码或设计 API 架构时使用。$zh246$
),
(
    'wecomcli-doc',
    'zh-CN',
    $zh246$企业微信文档$zh246$,
    $zh246$企业微信文档管理 Skill，支持新建普通文档、读取 Markdown 内容及使用 Markdown 覆写正文，可通过 docid 或文档 URL 定位。在线表格、智能表格和智能文档分别使用对应的 wecomcli Skill。$zh246$,
    $zh246$企业微信文档（doc）管理 Skill。提供普通文档的新建、内容读取（Markdown）和内容覆写能力。适用场景：（1）从零新建空白文档；（2）以 Markdown 格式读取文档完整内容；（3）用 Markdown 覆写文档正文。支持通过 docid 或文档 URL 定位文档。当用户提到“企业微信文档”“企微文档”“创建文档”“写个文档”，或链接形如 `https://doc.weixin.qq.com/doc/xxx` 时触发。注意：在线表格（`/sheet/*`）请使用 `wecomcli-sheet`；智能表格（`/smartsheet/*`）请使用 `wecomcli-smartsheet`；智能文档/智能主页（`/smartpage/*`）请使用 `wecomcli-smartpage`。$zh246$
),
(
    'feature-sliced-design',
    'zh-CN',
    $zh246$Feature-Sliced Design 架构$zh246$,
    $zh246$Feature-Sliced Design（FSD）v2.1 官方 Skill，用于在前端项目中应用 FSD 方法论，包括分层组织、代码归属、静态素材放置、Slice 分组、公共 API、导入边界、跨导入和框架集成等。$zh246$,
    $zh246$Feature-Sliced Design（FSD）v2.1 官方 Skill，用于在前端项目中应用该方法论。以下任务应使用：通过 FSD 分层组织项目结构；决定代码归属；放置静态素材（图像、图标、字体、PDF）；将关系紧密的 Slice 分组；定义公共 API 与导入边界；解决跨导入或评估 @x 模式；决定是否创建或删除某个 Entity；评估是否根本不需要 entities 层；决定页面布局归属，或是否使用 widgets 层（不建议）；决定逻辑应留在本地还是抽取；从 FSD v2.0 或非 FSD 代码库迁移；将 FSD 与框架集成（Next.js App Router 和 Pages Router、Nuxt、Vite、Astro）；或在 FSD 中实现身份验证、API 处理、Redux 和 TanStack Query（React Query）等常见模式。$zh246$
),
(
    'wecomcli-msg',
    'zh-CN',
    $zh246$企业微信消息$zh246$,
    $zh246$企业微信消息 Skill，支持查询会话列表、拉取文本/图片/文件/语音/视频消息记录、获取多媒体文件和发送文本消息。适用于查看聊天记录、收发消息及查看群聊中的图片或文件。$zh246$,
    $zh246$企业微信消息 Skill。提供会话列表查询、消息记录拉取（支持文本、图片、文件、语音和视频）、多媒体文件获取和文本消息发送能力。当用户需要“查看消息”“看聊天记录”“发消息给某人”“最近有什么消息”“给群里发消息”或“看看发了什么图片/文件”时触发。$zh246$
),
(
    'slides',
    'zh-CN',
    $zh246$HTML 演示文稿$zh246$,
    $zh246$使用 Chart.js、设计 Token、响应式布局、文案公式和情境化幻灯片策略创建具有战略性的 HTML 演示文稿。$zh246$,
    $zh246$使用 Chart.js、设计 Token、响应式布局、文案公式和情境化幻灯片策略创建具有战略性的 HTML 演示文稿。$zh246$
),
(
    'wecomcli-todo',
    'zh-CN',
    $zh246$企业微信待办$zh246$,
    $zh246$企业微信待办管理 Skill，支持创建、更新、删除待办，更改用户状态，获取列表及批量查看详情。适用于分派任务、标记完成、接受或拒绝待办、设置提醒、修改时间、查看详情及列出近期待办。$zh246$,
    $zh246$企业微信待办事项管理 Skill，支持创建待办、更新待办、更改用户在待办中的状态、获取待办列表、批量获取待办详情和删除待办。当用户说“帮我创建一个待办”“把这个任务分派给张三”“标记待办完成”“接受/拒绝这个待办”“把我在这个待办里的状态改成已完成”“删掉那个待办”“帮我建个提醒”“更新一下待办内容”“把提醒时间改到下周”“看看这个待办的详情”“待办分派给谁了”“看看我有哪些待办”或“列一下我最近的待办”等，需要对待办进行读写操作时使用。$zh246$
),
(
    'wecomcli-contact',
    'zh-CN',
    $zh246$企业微信通讯录$zh246$,
    $zh246$查询当前用户可见范围内的企业微信通讯录成员，支持按姓名或别名在本地筛选匹配，返回 userid、姓名和别名。仅返回当前用户有权限查看的成员，并非全量成员。$zh246$,
    $zh246$企业微信通讯录成员查询 Skill，获取当前用户可见范围内的通讯录成员，支持按姓名或别名在本地筛选匹配。返回 userid、姓名和别名。注意：仅返回当前用户有权限查看的成员，并非全量成员。$zh246$
),
(
    'wecomcli-schedule',
    'zh-CN',
    $zh246$企业微信日程$zh246$,
    $zh246$企业微信日程管理 Skill，支持查询与查看详情、创建日程并设置提醒和参与人、修改或取消日程、增删参与人，以及查询多人闲忙状态并分析共同空闲时段。$zh246$,
    $zh246$企业微信日程管理 Skill，适用于用户对企业微信日程的各类管理需求。当用户需要：（1）查询指定时间范围内的日程列表，或获取日程详细信息（标题、时间、地点、参与者等）；（2）创建新日程并设置提醒、参与人等；（3）修改已有日程的标题、时间、地点等信息或取消日程；（4）添加或移除日程参与人；（5）查询多个成员的闲忙状态并分析共同空闲时段以安排会议时，使用本 Skill。$zh246$
),
(
    'wecomcli-meeting',
    'zh-CN',
    $zh246$企业微信会议$zh246$,
    $zh246$企业微信会议 Skill，支持创建预约会议、查询会议列表、获取详情、取消会议和更新会议成员。适用于预约、安排、查看、查找或取消会议，以及添加、移除或修改参与人。$zh246$,
    $zh246$企业微信会议 Skill，支持创建预约会议、查询会议列表、获取会议详情、取消会议和更新会议成员。当用户需要“创建会议”“预约会议”“约会议”“安排会议”“查看会议”“查询会议列表”“会议详情”“什么时候开会”“有哪些会议”“查找会议”“取消会议”“删除会议”“修改会议成员”“添加会议参与人”或“移除会议成员”时触发。$zh246$
),
(
    'deepline-analytics',
    'zh-CN',
    $zh246$Deepline 商业分析$zh246$,
    $zh246$使用 Deepline 回答商业分析、RevOps、GTM 指标、销售管线、收入、漏斗、客户或数据仓库问题。支持 Snowflake 及语义层查询；潜客开发、数据丰富、联系人查找、外联和个性化流程请改用 deepline-gtm。$zh246$,
    $zh246$使用 Deepline 回答商业分析、RevOps、GTM 指标、销售管线、收入、漏斗、客户或数据仓库问题时使用本 Skill。触发短语包括“查询 Snowflake”“分析销售管线”“ACV 总额”“按季度拆分”“使用语义层”“运行语义查询”，或任何对 snowflake_get_semantic_layer / snowflake_run_semantic_query 的使用。跳过潜客开发、数据丰富、联系人查找、外联或个性化工作流；这些场景请使用 deepline-gtm。$zh246$
),
(
    'vue-testing-best-practices',
    'zh-CN',
    $zh246$Vue 测试最佳实践$zh246$,
    $zh246$用于 Vue.js 测试，涵盖 Vitest、Vue Test Utils、组件测试、Mock、测试模式，以及使用 Playwright 进行端到端测试。$zh246$,
    $zh246$用于 Vue.js 测试。涵盖 Vitest、Vue Test Utils、组件测试、Mock、测试模式，以及使用 Playwright 进行端到端测试。$zh246$
),
(
    'taste-design',
    'zh-CN',
    $zh246$Taste 语义设计系统$zh246$,
    $zh246$面向 Google Stitch 的语义设计系统 Skill，生成适合代理使用的 DESIGN.md，强制执行高级且拒绝千篇一律的 UI 标准，包括严格排版、校准色彩、非对称布局、持续微动效和硬件加速性能。$zh246$,
    $zh246$面向 Google Stitch 的语义设计系统 Skill。生成适合代理使用的 DESIGN.md 文件，强制执行高级且拒绝千篇一律的 UI 标准，包括严格的排版、经过校准的色彩、非对称布局、持续微动效和硬件加速性能。$zh246$
),
(
    'health',
    'zh-CN',
    $zh246$工程健康审计$zh246$,
    $zh246$在预算约束下借助代理开展工程健康审计，检查指令/配置漂移、Hook/MCP、验证器覆盖面和 AI 可维护性。适用于审计 Claude、Codex、Pi、Agent 指令、MCP、Hook 或 AI 可维护性漂移。$zh246$,
    $zh246$在预算约束下借助代理开展工程健康审计，检查指令/配置漂移、Hook/MCP、验证器覆盖面和 AI 可维护性。当用户以任何语言要求审计 Claude、Codex、Pi、Agent 指令、MCP 或 Hook、验证器覆盖率，或 AI 可维护性漂移时使用。不适用于调试应用代码或审查 PR。$zh246$
),
(
    'dbs-report',
    'zh-CN',
    $zh246$DBS 诊断报告$zh246$,
    $zh246$把多次 dbs-save 存档合并成可交付的 Markdown 报告。用户要求汇总诊断、整理报告或生成可分享材料时使用。$zh246$,
    $zh246$把多次 dbs-save 存档合并成可交付的 Markdown 报告。用户要求汇总诊断、整理报告或生成可分享材料时使用。$zh246$
),
(
    'rust-best-practices',
    'zh-CN',
    $zh246$Rust 最佳实践$zh246$,
    $zh246$基于 Apollo GraphQL 最佳实践手册的惯用 Rust 编码指南，适用于编写、审查或重构 Rust 代码，选择借用、克隆与所有权模式，使用 Result 处理错误，优化性能及编写测试或文档。$zh246$,
    $zh246$基于 Apollo GraphQL 最佳实践手册的惯用 Rust 编码指南。以下场景使用：（1）编写新的 Rust 代码或函数；（2）审查或重构现有 Rust 代码；（3）在借用、克隆或所有权模式之间做选择；（4）使用 Result 类型实现错误处理；（5）优化 Rust 代码性能；（6）为 Rust 项目编写测试或文档。$zh246$
),
(
    'dbs-save',
    'zh-CN',
    $zh246$DBS 保存诊断$zh246$,
    $zh246$把当前诊断的关键状态保存到本地，并查看或设置存档位置。用户要求保存结论、跨会话续接或修改存档位置时使用。$zh246$,
    $zh246$把当前诊断的关键状态保存到本地，并查看或设置存档位置。用户要求保存结论、跨会话续接或修改存档位置时使用。$zh246$
),
(
    'dbs-restore',
    'zh-CN',
    $zh246$DBS 恢复诊断$zh246$,
    $zh246$恢复由 dbs-save 保存的最近诊断状态。用户要求接着上次、查看此前结论或继续未完成诊断时使用。$zh246$,
    $zh246$恢复由 dbs-save 保存的最近诊断状态。用户要求接着上次、查看此前结论或继续未完成诊断时使用。$zh246$
),
(
    'web-access',
    'zh-CN',
    $zh246$网页访问$zh246$,
    $zh246$所有联网操作必须通过本 Skill 处理，包括搜索、网页抓取、登录后操作和网络交互。适用于搜索信息、查看网页、操作需登录的网站、抓取社交媒体、读取动态页面及任何需要真实浏览器的网络任务。$zh246$,
    $zh246$所有联网操作必须通过本 Skill 处理，包括搜索、网页抓取、登录后操作、网络交互等。触发场景：用户要求搜索信息、查看网页内容、访问需要登录的网站、操作网页界面、抓取社交媒体内容（小红书、微博、Twitter 等）、读取动态渲染页面，以及任何需要真实浏览器环境的网络任务。$zh246$
),
(
    'tavily-research',
    'zh-CN',
    $zh246$Tavily 深度研究$zh246$,
    $zh246$通过 Tavily CLI 开展带引用的全面 AI 研究。适用于深度研究、详细报告、对比、市场分析、文献综述和多来源综合，返回以网页来源为依据的结构化报告，耗时约 30–120 秒。$zh246$,
    $zh246$通过 Tavily CLI 开展带引用的全面 AI 研究。当用户需要深度研究、详细报告、对比、市场分析、文献综述，或说“研究”“调查”“深入分析”“比较 X 与 Y”“某领域的市场情况如何”，或需要带明确引用的多来源综合时使用。返回一份以网页来源为依据的结构化报告。耗时约 30–120 秒。快速事实查找请改用 tavily-search。$zh246$
),
(
    'opencli-autofix',
    'zh-CN',
    $zh246$OpenCLI 适配器自动修复$zh246$,
    $zh246$OpenCLI 命令失败时自动修复损坏的适配器，引导收集 Trace 产物、修补适配器、重试，并在修复验证通过后提交上游 GitHub Issue；适用于任何 AI 代理。$zh246$,
    $zh246$OpenCLI 命令失败时自动修复损坏的适配器。当 opencli 命令失败时加载本 Skill：它会引导收集 Trace 产物、修补适配器、重试，并在修复验证通过后提交上游 GitHub Issue。适用于任何 AI 代理。$zh246$
),
(
    'okx-dex-market',
    'zh-CN',
    $zh246$OKX DEX 市场数据$zh246$,
    $zh246$硬性限制：绝不用于预测市场或 Polymarket UpDown 查询；具名 DApp 加时间范围，或 BTC/ETH/SOL/XRP/BNB/DOGE/HYPE 涨跌查询应转到 okx-dapp-discovery。其余场景只读访问六组链上 DEX 数据。$zh246$,
    $zh246$硬性限制：绝不用于预测市场或 Polymarket UpDown 查询。当具名 DApp（Polymarket/Aave/Hyperliquid/PancakeSwap/Morpho）与时间范围同时出现，或查询 BTC/ETH/SOL/XRP/BNB/DOGE/HYPE 的涨跌/updown 时，转到 okx-dapp-discovery。除此之外，仅提供六组只读链上 DEX 数据：TOKEN（搜索、热门、流动性、持有者/巨鲸、风险元数据、集群/持仓集中度、交易历史、顶级交易者）；MARKET（价格、K 线/OHLC、指数价格、钱包 PnL/胜率、交易历史）；SIGNAL（聪明钱/KOL/巨鲸跟踪、买入信号、排行榜/牛人榜）；SOCIAL（新闻、情绪、Token 热度、KOL 排行榜）；TRENCHES（pump.fun/Meme 发布/新盘/扫链、开发者声誉、Bundle/Sniper 检测/捆绑狙击者、共同投资者——仅只读；买入/Snipe 请转到 okx-dapp-discovery）；WS（onchainos ws CLI 或自定义 WebSocket 脚本）。还负责全部六组数据的 Market API 付款/x402、配额，以及 MARKET_API_*_OVER_QUOTA/confirming:true。$zh246$
),
(
    'dbs-goal',
    'zh-CN',
    $zh246$DBS 目标澄清$zh246$,
    $zh246$提取已有约束，只追问影响执行或验收的信息，把模糊愿望和目标整理成可行动、可检查的交付物。用户要求澄清目标、检查任务是否说清或定义交付结果时使用。$zh246$,
    $zh246$提取已有约束，只追问影响执行或验收的信息，把模糊愿望和目标整理成可行动、可检查的交付物。用户要求澄清目标、检查任务是否说清或定义交付结果时使用。$zh246$
),
(
    'antfu',
    'zh-CN',
    $zh246$Anthony Fu 工具规范$zh246$,
    $zh246$Anthony Fu 为 JavaScript/TypeScript 项目制定的鲜明工具偏好与规范。适用于搭建新项目、配置 ESLint/Prettier 替代方案、Monorepo、发布库，或用户提到 Anthony Fu 的偏好时。$zh246$,
    $zh246$Anthony Fu 为 JavaScript/TypeScript 项目制定的鲜明工具偏好与规范。搭建新项目、配置 ESLint/Prettier 替代方案、设置 Monorepo、发布库，或用户提到 Anthony Fu 的偏好时使用。$zh246$
),
(
    'database-migration',
    'zh-CN',
    $zh246$数据库迁移$zh246$,
    $zh246$跨 ORM 和平台执行数据库迁移，涵盖零停机策略、数据转换和回滚流程。适用于迁移数据库、变更架构、执行数据转换或实施零停机部署策略。$zh246$,
    $zh246$跨 ORM 和平台执行数据库迁移，采用零停机策略、数据转换和回滚流程。迁移数据库、变更架构、执行数据转换或实施零停机部署策略时使用。$zh246$
),
(
    'conventional-commit',
    'zh-CN',
    $zh246$约定式提交$zh246$,
    $zh246$使用结构化 XML 格式生成约定式提交消息的提示词与工作流，依据 Conventional Commits 规范指导创建标准化、描述清晰的提交消息，并提供说明、示例和验证。$zh246$,
    $zh246$使用结构化 XML 格式生成约定式提交消息的提示词与工作流。依据 Conventional Commits 规范，指导用户创建标准化、描述清晰的提交消息，其中包含说明、示例和验证。$zh246$
),
(
    'find-qualified-titles',
    'zh-CN',
    $zh246$查找合格职位$zh246$,
    $zh246$根据 ICP 在已知公司域名中查找真实任职者，尤其适用于查找目标公司的全部职位名称、合格职位、RevOps 或营销运营买家，或付费人员搜索前应先精确发现职位名称的场景。$zh246$,
    $zh246$根据 ICP 在已知公司域名中查找真实任职者时使用，尤其适用于“查找这些公司的所有职位名称”“查找合格职位”“查找 RevOps 或营销运营买家”等请求，或应在付费人员搜索之前先精确发现职位名称的场景。$zh246$
),
(
    'flutter-animations',
    'zh-CN',
    $zh246$Flutter 动画$zh246$,
    $zh246$添加、修复、重构、调试、测试或解释 Flutter 动画与运动效果，涵盖隐式与显式动画、Hero 转场、错落或序列动画、物理运动、手势、弹簧、滚动、曲线、性能、无障碍和生命周期错误。$zh246$,
    $zh246$添加、修复、重构、调试、测试或解释 Flutter 动画与运动效果。处理 AnimatedContainer、AnimatedOpacity、AnimatedSwitcher 和 TweenAnimationBuilder 等隐式动画；使用 AnimationController、Tween、CurvedAnimation、AnimatedWidget、AnimatedBuilder 和内置 Transition 的显式动画；Hero/共享元素路由转场；错落或序列动画；基于物理的运动、手势、弹簧、抛动、滚动物理、曲线、性能、无障碍、减少动态效果，以及动画生命周期错误时使用。$zh246$
),
(
    'proactive-agent',
    'zh-CN',
    $zh246$主动型 AI 代理$zh246$,
    $zh246$把 AI 代理从被动执行任务者转变为能预判需求并持续改进的主动伙伴。现已包含 WAL Protocol、用于跨上下文存续的 Working Buffer、Compaction Recovery 和经过实战检验的安全模式，是 Hal Stack 的一部分。$zh246$,
    $zh246$把 AI 代理从被动执行任务者转变为能预判需求并持续改进的主动伙伴。现已包含 WAL Protocol、用于跨上下文存续的 Working Buffer、Compaction Recovery，以及经过实战检验的安全模式。属于 Hal Stack 🦞。$zh246$
),
(
    'clerk-cli',
    'zh-CN',
    $zh246$Clerk CLI 管理$zh246$,
    $zh246$操作 Clerk CLI（`clerk`）完成身份验证、用户/组织/会话管理、模拟身份、本地 Webhook 测试、部署验证、实例配置、环境密钥、功能开关，以及 Clerk Backend、Platform 或 Frontend API 调用。$zh246$,
    $zh246$操作 Clerk CLI（`clerk` 二进制文件）完成身份验证、用户/组织/会话管理、模拟身份、本地 Webhook 测试、部署验证、实例配置、环境密钥、功能开关，以及任何 Clerk Backend、Platform 或 Frontend API 调用。用户提到 Clerk 管理任务、“列出 Clerk 用户”“模拟某个用户”“在本地测试 Webhook”“启用组织”“启用计费”、`clerk env pull`、`clerk doctor`、`clerk deploy`、`clerk api`，或任何临时 Clerk API 请求时使用。优先使用 CLI 而非原始 HTTP：它会自动处理身份验证、密钥解析、应用/实例定位和格式化。$zh246$
),
(
    'smart-search',
    'zh-CN',
    $zh246$智能搜索$zh246$,
    $zh246$基于 opencli 命令的智能搜索路由器。当用户希望使用 OpenCLI、CLI 或 API 搜索、查询、查找或研究信息，尤其涉及指定网站、社交媒体、技术资料、新闻、购物、旅游、求职、金融或中文内容时，务必使用本 Skill。$zh246$,
    $zh246$基于 opencli 命令的智能搜索路由器。当用户希望使用 OpenCLI、CLI 或 API 搜索、查询、查找或研究信息，尤其涉及指定网站、社交媒体、技术资料、新闻、购物、旅游、求职、金融或中文内容时，务必使用本 Skill。$zh246$
),
(
    'tailwind-css-patterns',
    'zh-CN',
    $zh246$Tailwind CSS 模式$zh246$,
    $zh246$提供全面的 Tailwind CSS 实用工具优先样式模式，涵盖响应式设计、布局工具、Flexbox、Grid、间距、排版、颜色和现代 CSS 最佳实践。$zh246$,
    $zh246$提供全面的 Tailwind CSS 实用工具优先样式模式，涵盖响应式设计、布局工具、Flexbox、Grid、间距、排版、颜色和现代 CSS 最佳实践。为 React/Vue/Svelte 组件设置样式、构建响应式布局、实现设计系统或优化 CSS 工作流时使用。$zh246$
),
(
    'observability-and-instrumentation',
    'zh-CN',
    $zh246$可观测性与埋点$zh246$,
    $zh246$为代码添加埋点，使生产行为可见且可诊断。适用于添加日志、指标、追踪或告警；发布生产功能并需要证明其正常工作；或生产问题已出现但现有数据无法说明发生了什么。$zh246$,
    $zh246$为代码添加埋点，使生产行为可见且可诊断。添加日志、指标、追踪或告警时使用。发布任何会在生产环境运行的功能，并且需要证据证明其正常工作时使用。当生产问题被报告，但无法从现有数据判断发生了什么时也应使用。$zh246$
),
(
    'browser-mcp-agent',
    'zh-CN',
    $zh246$浏览器 MCP 代理$zh246$,
    $zh246$通过 MCP 工具调用为 AI 代理提供独立真实浏览器，可启动、导航、点击、填写、截图、提取文本和运行 JS，并使用内核级真实设备指纹与持久配置文件，让会话持续登录且网站看到一致设备。$zh246$,
    $zh246$通过 MCP 工具调用为 AI 代理提供独立真实浏览器：启动、导航、点击、填写、截图、提取文本和运行 JS；配备内核级真实设备指纹与持久配置文件，让会话在多次运行之间保持登录，且网页看到的是同一台一致设备，而非无头构建。无需编写 Playwright 或 SDK 代码。当代理需要自行操作网站；计算机操作/浏览器操作方案需要采集的真实指纹而非合成指纹；代理会话总是丢失登录状态；或需要比较托管的代理浏览器服务时使用。以下请求也适用：“MCP 浏览器”“浏览器 MCP 服务器”“让我的代理浏览网页”“代理浏览器控制”“browser-use MCP”“计算机操作浏览器”“Browserbase 替代方案”“Steel 浏览器替代方案”“检测到无头浏览器”。支持 Node（npx）或 Python；支持 Windows x64、macOS Intel 与 Apple Silicon、Linux x64/arm64。SDK 与 REST 参考为 anti-detect-browser；账户隔离参考 multi-account-isolation。$zh246$
),
(
    'multi-account-isolation',
    'zh-CN',
    $zh246$多账户隔离$zh246$,
    $zh246$验证浏览器配置文件是否真正彼此隔离，包括时区与出口 IP、WebRTC 代理暴露、Canvas/WebGL 哈希稳定性，以及身份画像、Cookie Jar 和地址是否共享。适用于多账户或测试身份隔离自检。$zh246$,
    $zh246$验证浏览器配置文件是否真正彼此隔离，而不是想当然：确认每个配置文件的时区与其自身出口 IP 一致；WebRTC 只暴露代理；同一配置文件重启后的 Canvas 与 WebGL 哈希保持一致；任意两个配置文件都不共享身份画像、Cookie Jar 或地址。当你在同一台机器上运行多个自有账户或测试身份并需要检查配置；某个配置文件测试无异常但仍感觉有问题；需要选择检测套件（CreepJS、whoer、browserleaks WebRTC、pixelscan、liarjs）；审计供应商运行时如何处理 API 与代理凭据；或询问浏览器隔离完全无法覆盖哪些层面时使用。以下请求也适用：“配置文件隔离检查”“指纹一致性测试”“时区不匹配”“WebRTC 泄漏”“Canvas 哈希不稳定”“账户关联”“临时配置文件”“防关联”“多账号”“隔离自检”。SDK 为 anti-detect-browser；MCP 为 browser-mcp-agent。$zh246$
),
(
    'security-review',
    'zh-CN',
    $zh246$安全审查$zh246$,
    $zh246$适用于添加身份验证、处理用户输入、使用密钥、创建 API 端点，或实现支付及敏感功能的场景，提供全面的安全检查清单与模式。$zh246$,
    $zh246$添加身份验证、处理用户输入、使用密钥、创建 API 端点，或实现支付及敏感功能时使用本 Skill。提供全面的安全检查清单与模式。$zh246$
),
(
    'unocss',
    'zh-CN',
    $zh246$UnoCSS 原子引擎$zh246$,
    $zh246$UnoCSS 是即时原子化 CSS 引擎，也是 Tailwind CSS 的超集。适用于配置 UnoCSS、编写实用工具规则与快捷方式，或使用 Wind、Icons、Attributify 等预设。$zh246$,
    $zh246$UnoCSS 是即时原子化 CSS 引擎，也是 Tailwind CSS 的超集。配置 UnoCSS、编写实用工具规则或快捷方式，或使用 Wind、Icons、Attributify 等预设时使用。$zh246$
),
(
    'tavily-best-practices',
    'zh-CN',
    $zh246$Tavily 最佳实践$zh246$,
    $zh246$以最佳实践构建生产级 Tavily 集成。为使用 Claude Code、Cursor 等编码助手的开发者提供参考文档，用于在代理工作流、RAG 系统或自主代理中实现网页搜索、内容提取、爬取和研究。$zh246$,
    $zh246$以最佳实践构建生产级 Tavily 集成。为使用编码助手（Claude Code、Cursor 等）的开发者提供参考文档，用于在代理工作流、RAG 系统或自主代理中实现网页搜索、内容提取、爬取和研究。$zh246$
),
(
    'apify-ultimate-scraper',
    'zh-CN',
    $zh246$Apify 全能抓取器$zh246$,
    $zh246$适用于任何平台的通用 AI 网页抓取器，可从 Instagram、Facebook、TikTok、YouTube、LinkedIn、X/Twitter、Google Maps、Google Search、Google Trends、Reddit、Airbnb、Yelp 等 15 个以上平台提取数据。$zh246$,
    $zh246$适用于任何平台的通用 AI 网页抓取器。可从 Instagram、Facebook、TikTok、YouTube、LinkedIn、X/Twitter、Google Maps、Google Search、Google Trends、Reddit、Airbnb、Yelp 等 15 个以上平台抓取数据。适用于潜客开发、品牌监测、竞争对手分析、网红发现、趋势研究、内容分析、受众分析、评论分析、SEO 情报、招聘或任何数据提取任务。$zh246$
),
(
    'postgresql-optimization',
    'zh-CN',
    $zh246$PostgreSQL 优化$zh246$,
    $zh246$专注 PostgreSQL 独有特性、高级数据类型和专属能力的开发助手，涵盖 JSONB 操作、数组、自定义类型、范围/几何类型、全文搜索、窗口函数及 PostgreSQL 扩展生态。$zh246$,
    $zh246$专注 PostgreSQL 独有特性、高级数据类型和 PostgreSQL 专属能力的开发助手。涵盖 JSONB 操作、数组类型、自定义类型、范围/几何类型、全文搜索、窗口函数和 PostgreSQL 扩展生态。$zh246$
),
(
    'resend',
    'zh-CN',
    $zh246$Resend 邮件 API$zh246$,
    $zh246$用于 Resend 邮件 API，包括发送单封或批量交易邮件、通过 Webhook 接收邮件、模板、投递事件、域名、联系人、广播、API 密钥、自动化、事件、请求日志和 SDK。$zh246$,
    $zh246$使用 Resend 邮件 API 时使用本 Skill，包括发送交易邮件（单封或批量）、通过 Webhook 接收来信、管理邮件模板、跟踪投递事件，以及管理域名、联系人、广播、Webhook、API 密钥、自动化、事件、查看 API 请求日志或设置 Resend SDK。用户只要提到 Resend，即使只是“使用 Resend 发送一封邮件”这样的简单任务，也始终应使用本 Skill；其中包含防止常见生产问题的关键注意事项，如幂等键、Webhook 验证和模板变量语法。$zh246$
),
(
    'async-python-patterns',
    'zh-CN',
    $zh246$Python 异步模式$zh246$,
    $zh246$掌握 Python asyncio、并发编程与 async/await 模式，以构建高性能应用。适用于异步 API、并发系统或需要非阻塞操作的 I/O 密集型应用。$zh246$,
    $zh246$掌握 Python asyncio、并发编程与 async/await 模式，以构建高性能应用。构建异步 API、并发系统或需要非阻塞操作的 I/O 密集型应用时使用。$zh246$
),
(
    'chrome-devtools',
    'zh-CN',
    $zh246$Chrome DevTools 自动化$zh246$,
    $zh246$使用 Chrome DevTools MCP 进行专家级浏览器自动化、调试和性能分析。适用于与网页交互、截取屏幕截图、分析网络流量和性能剖析。$zh246$,
    $zh246$使用 Chrome DevTools MCP 进行专家级浏览器自动化、调试和性能分析。与网页交互、截取屏幕截图、分析网络流量和进行性能剖析时使用。$zh246$
),
(
    'playwright-explore-website',
    'zh-CN',
    $zh246$Playwright 网站探索$zh246$,
    $zh246$使用 Playwright MCP 探索网站以开展测试。$zh246$,
    $zh246$使用 Playwright MCP 探索网站以开展测试。$zh246$
),
(
    'deep-agents-memory',
    'zh-CN',
    $zh246$Deep Agents 记忆$zh246$,
    $zh246$当 Deep Agent 需要记忆、持久化或文件系统访问时调用，涵盖 StateBackend（临时）、StoreBackend（持久）、FilesystemMiddleware，以及用于路由的 CompositeBackend。$zh246$,
    $zh246$当 Deep Agent 需要记忆、持久化或文件系统访问时，调用本 Skill。涵盖 StateBackend（临时）、StoreBackend（持久）、FilesystemMiddleware，以及用于路由的 CompositeBackend。$zh246$
),
(
    'dbs-learning',
    'zh-CN',
    $zh246$DBS 连续学习$zh246$,
    $zh246$把课题拆成连续学习文章，并根据用户反馈调整后续深度、角度和节奏。用户要求系统学习一个主题、继续下一篇或按反馈调整课程时使用。$zh246$,
    $zh246$把课题拆成连续学习文章，并根据用户反馈调整后续深度、角度和节奏。用户要求系统学习一个主题、继续下一篇或按反馈调整课程时使用。$zh246$
),
(
    'dotnet-best-practices',
    'zh-CN',
    $zh246$.NET 最佳实践$zh246$,
    $zh246$确保解决方案或项目中的 .NET/C# 代码符合最佳实践。$zh246$,
    $zh246$确保解决方案或项目中的 .NET/C# 代码符合最佳实践。$zh246$
),
(
    'github-issues',
    'zh-CN',
    $zh246$GitHub Issue 管理$zh246$,
    $zh246$使用 MCP 工具创建、更新和管理 GitHub Issue，支持缺陷报告、功能请求、任务、标签、负责人、里程碑、自定义字段、类型、工作流、关联与依赖，以及阻塞关系。$zh246$,
    $zh246$使用 MCP 工具创建、更新和管理 GitHub Issue。当用户希望创建缺陷报告、功能请求或任务 Issue，更新现有 Issue，添加标签/负责人/里程碑，设置 Issue 字段（日期、优先级、自定义字段），设置 Issue 类型，管理 Issue 工作流，关联 Issue，添加依赖，或跟踪被阻塞/阻塞关系时使用。以下请求会触发：“创建 Issue”“提交缺陷”“请求功能”“更新 Issue X”“设置优先级”“设置开始日期”“关联 Issue”“添加依赖”“被阻塞”“正在阻塞”，以及任何 GitHub Issue 管理任务。$zh246$
),
(
    'github-actions-templates',
    'zh-CN',
    $zh246$GitHub Actions 模板$zh246$,
    $zh246$创建可用于生产的 GitHub Actions 工作流，以自动测试、构建和部署应用。适用于使用 GitHub Actions 配置 CI/CD、自动化开发工作流或创建可复用工作流模板。$zh246$,
    $zh246$创建可用于生产的 GitHub Actions 工作流，以自动测试、构建和部署应用。使用 GitHub Actions 配置 CI/CD、自动化开发工作流或创建可复用工作流模板时使用。$zh246$
),
(
    'ponytail-review',
    'zh-CN',
    $zh246$Ponytail 过度工程审查$zh246$,
    $zh246$只关注过度工程的代码审查，寻找可删除内容：重复实现标准库、不必要依赖、臆测式抽象和无效灵活性。每项发现仅一行，列出位置、应删内容及替代方案。$zh246$,
    $zh246$只关注过度工程的代码审查。寻找可删除内容：重复实现标准库、不必要的依赖、臆测式抽象和无效灵活性。每项发现仅用一行表示：位置、应删除的内容、替代方案。当用户说“审查过度工程”“哪些可以删除”“这是否过度设计”“简化审查”，或调用 /ponytail-review 时使用。它是以正确性为重点的代码审查的补充，只追查复杂性。$zh246$
),
(
    'fixing-metadata',
    'zh-CN',
    $zh246$HTML 元数据修复$zh246$,
    $zh246$审计并修复 HTML 元数据，包括页面标题、Meta Description、Canonical URL、Open Graph 标签、Twitter Card、Favicon、JSON-LD 结构化数据和 robots 指令。$zh246$,
    $zh246$审计并修复 HTML 元数据，包括页面标题、Meta Description、Canonical URL、Open Graph 标签、Twitter Card、Favicon、JSON-LD 结构化数据和 robots 指令。添加 SEO 元数据、修复社交分享预览、审查 Open Graph 标签、设置 Canonical URL，或发布需要正确 Meta 标签的新页面时使用。$zh246$
),
(
    'expo-brownfield',
    'zh-CN',
    $zh246$Expo 原生应用集成$zh246$,
    $zh246$开源框架：将 Expo 与 React Native 集成到现有原生 iOS 或 Android 应用。适用于棕地集成、在原生应用中嵌入 React Native、AAR/XCFramework，或向现有 Kotlin/Swift 项目添加 Expo。$zh246$,
    $zh246$开源框架。将 Expo 与 React Native 集成到现有原生 iOS 或 Android 应用。当用户提到棕地集成、在原生应用中嵌入 React Native、AAR/XCFramework，或向现有 Kotlin/Swift 项目添加 Expo 时使用。涵盖隔离式方案与集成式方案。$zh246$
),
(
    'dbs-good-question',
    'zh-CN',
    $zh246$DBS 好问题$zh246$,
    $zh246$把模糊问题改写成 Agent 可推理、可批评、可验证的问题说明书，并判断自动化解决程度。用户要求把问题说清、生成好问题或评估 Agent 可解性时使用。$zh246$,
    $zh246$把模糊问题改写成 Agent 可推理、可批评、可验证的问题说明书，并判断自动化解决程度。用户要求把问题说清、生成好问题或评估 Agent 可解性时使用。$zh246$
),
(
    'sql-optimization',
    'zh-CN',
    $zh246$SQL 全面优化$zh246$,
    $zh246$适用于所有 SQL 数据库（MySQL、PostgreSQL、SQL Server、Oracle）的通用性能优化助手，提供全面查询调优、索引策略、执行计划分析、分页优化、批量操作和性能监控指导。$zh246$,
    $zh246$适用于所有 SQL 数据库（MySQL、PostgreSQL、SQL Server、Oracle）的通用 SQL 性能优化助手，提供全面的查询调优、索引策略和数据库性能分析，并提供执行计划分析、分页优化、批量操作和性能监控指导。$zh246$
),
(
    'excel-automation',
    'zh-CN',
    $zh246$Excel 自动化$zh246$,
    $zh246$原始说明仅包含符号“>”。$zh246$,
    $zh246$原始说明仅包含符号“>”。$zh246$
),
(
    'backtesting-frameworks',
    'zh-CN',
    $zh246$交易策略回测框架$zh246$,
    $zh246$构建稳健的交易策略回测系统，正确处理前视偏差、幸存者偏差和交易成本。适用于开发交易算法、验证策略或构建回测基础设施。$zh246$,
    $zh246$构建稳健的交易策略回测系统，正确处理前视偏差、幸存者偏差和交易成本。开发交易算法、验证策略或构建回测基础设施时使用。$zh246$
),
(
    'ast-grep',
    'zh-CN',
    $zh246$ast-grep 结构化搜索$zh246$,
    $zh246$编写 ast-grep 规则以执行结构化代码搜索与分析的指南。适用于使用抽象语法树（AST）模式搜索代码库、查找特定代码结构，或执行超出简单文本搜索能力的复杂代码查询。$zh246$,
    $zh246$编写 ast-grep 规则以执行结构化代码搜索与分析的指南。当用户需要使用抽象语法树（AST）模式搜索代码库、查找特定代码结构，或执行超出简单文本搜索能力的复杂代码查询时使用。用户要求搜索代码模式、查找特定语言构造，或定位具有特定结构特征的代码时，应使用本 Skill。$zh246$
),
(
    'openapi-spec-generation',
    'zh-CN',
    $zh246$OpenAPI 规范生成$zh246$,
    $zh246$从代码、设计优先规范和验证模式生成并维护 OpenAPI 3.1 规范。适用于创建 API 文档、生成 SDK 或确保 API 契约合规。$zh246$,
    $zh246$从代码、设计优先规范和验证模式生成并维护 OpenAPI 3.1 规范。创建 API 文档、生成 SDK 或确保 API 契约合规时使用。$zh246$
),
(
    'architecture-decision-records',
    'zh-CN',
    $zh246$架构决策记录$zh246$,
    $zh246$按照技术决策文档最佳实践编写和维护架构决策记录（ADR）。适用于记录重大技术决策、审查过去的架构选择或建立决策流程。$zh246$,
    $zh246$按照技术决策文档最佳实践编写和维护架构决策记录（ADR）。记录重大技术决策、审查过去的架构选择或建立决策流程时使用。$zh246$
),
(
    'opencli-adapter-author',
    'zh-CN',
    $zh246$OpenCLI 适配器开发$zh246$,
    $zh246$为新网站编写 OpenCLI 适配器，或向现有网站添加新命令；从首次侦察、字段解码、适配器编码到验证提供端到端指导，并取代 opencli-oneshot / opencli-explorer。$zh246$,
    $zh246$为新网站编写 OpenCLI 适配器，或向现有网站添加新命令时使用。从首次侦察、字段解码、适配器编码到验证提供端到端指导。取代 opencli-oneshot / opencli-explorer。临时驱动浏览器（无需适配器）请改用 opencli-browser；了解 opencli 的总体方向请参见 opencli-usage。$zh246$
),
(
    'migrate-radix-to-base',
    'zh-CN',
    $zh246$Radix UI 迁移至 Base UI$zh246$,
    $zh246$将 React 项目和组件从 Radix UI 迁移到 Base UI。适用于迁移 Radix、改用 base-ui、转换 Radix Primitives，或切换 shadcn 项目的基础组件库；支持单个组件和整个项目。$zh246$,
    $zh246$将 React 项目和组件从 Radix UI 迁移到 Base UI。当用户要求从 Radix 迁移、改用 base-ui、转换 Radix Primitives，或切换 shadcn 项目的基础组件库时使用。既可处理单个组件（如“迁移 Accordion”），也可处理整个项目。$zh246$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
