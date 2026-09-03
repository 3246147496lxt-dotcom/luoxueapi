-- Reviewed Simplified Chinese catalog copy for published skills.sh ranks 838-918.
-- Keep upstream Skill packages and immutable marketplace slugs unchanged.
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    'review-pr',
    'zh-CN',
    $zh245$审查 Pull Request$zh245$,
    $zh245$审查 Pull Request diff，并将结构化反馈写入 review.json，供工作流发布。适用于通过 pr_diff.txt、pr_description.txt 等本地产物审查已检出的 PR，并生成机器可读的审查结果，而不是直接发布到 GitHub。$zh245$,
    $zh245$审查 Pull Request diff，并将结构化反馈写入 review.json，供工作流发布。适用于通过 pr_diff.txt、pr_description.txt 等本地产物审查已检出的 PR，并生成机器可读的审查结果，而不是直接发布到 GitHub。$zh245$
),
(
    'mobile-android-design',
    'zh-CN',
    $zh245$Android 原生界面设计$zh245$,
    $zh245$掌握 Material Design 3 和 Jetpack Compose 模式，用于构建 Android 原生应用。适用于设计 Android 界面、实现 Compose UI，或遵循 Google Material Design 指南的场景。$zh245$,
    $zh245$掌握 Material Design 3 和 Jetpack Compose 模式，用于构建 Android 原生应用。适用于设计 Android 界面、实现 Compose UI，或遵循 Google Material Design 指南的场景。$zh245$
),
(
    'spec-driven-implementation',
    'zh-CN',
    $zh245$规格驱动实现$zh245$,
    $zh245$针对大型功能推行规格优先工作流：实现前编写 PRODUCT.md，必要时编写 TECH.md，并随实现演进持续更新两份规格。适用于启动重要功能、规划智能体驱动的实现，或用户希望将产品与技术规格纳入版本控制时。$zh245$,
    $zh245$针对大型功能推行规格优先工作流：实现前编写 PRODUCT.md，必要时编写 TECH.md，并随实现演进持续更新两份规格。适用于启动重要功能、规划智能体驱动的实现，或用户希望将产品与技术规格提交到源代码版本控制时。$zh245$
),
(
    'architecture-patterns',
    'zh-CN',
    $zh245$后端架构模式$zh245$,
    $zh245$实施经过验证的后端架构模式，包括整洁架构、六边形架构和领域驱动设计。适用于为新微服务设计整洁架构、将单体重构为有界上下文、实施六边形架构或洋葱架构，以及调试应用层之间的循环依赖。$zh245$,
    $zh245$实施经过验证的后端架构模式，包括整洁架构、六边形架构和领域驱动设计。适用于为新微服务设计整洁架构、将单体重构为有界上下文、实施六边形架构或洋葱架构，以及调试应用层之间的循环依赖。$zh245$
),
(
    'write-product-spec',
    'zh-CN',
    $zh245$编写产品规格$zh245$,
    $zh245$为 Warp 中重要的面向用户功能编写 PRODUCT.md 规格，重点描述详细行为与验证方式。适用于用户要求产品规格、预期行为文档或 PRD，希望在实现前定义功能行为，或功能规模较大、行为存在歧义而书面规格有助于实现或审查时。$zh245$,
    $zh245$为 Warp 中重要的面向用户功能编写 PRODUCT.md 规格，重点描述详细行为与验证方式。适用于用户要求产品规格、预期行为文档或 PRD，希望在实现前定义功能行为，或功能规模较大、行为存在歧义而书面规格有助于实现或审查时。$zh245$
),
(
    'api-and-interface-design',
    'zh-CN',
    $zh245$API 与接口设计$zh245$,
    $zh245$指导设计稳定的 API 和接口。适用于设计 API、模块边界或任何公共接口，也适用于创建 REST 或 GraphQL 端点、定义模块间类型契约，或划分前端与后端边界。$zh245$,
    $zh245$指导设计稳定的 API 和接口。适用于设计 API、模块边界或任何公共接口，也适用于创建 REST 或 GraphQL 端点、定义模块间类型契约，或划分前端与后端边界。$zh245$
),
(
    'dbs',
    'zh-CN',
    $zh245$DBS 商业工具箱$zh245$,
    $zh245$dontbesilent 商业工具箱主入口，提供新手教程、任务前路由和任务后导航。用户不知道该用哪个 DBS 技能、要求分析商业问题或询问下一步时使用。$zh245$,
    $zh245$dontbesilent 商业工具箱主入口，提供新手教程、任务前路由和任务后导航。用户不知道该用哪个 DBS 技能、要求分析商业问题或询问下一步时使用。$zh245$
),
(
    'firestore-security-rules-auditor',
    'zh-CN',
    $zh245$Firestore 安全规则审计$zh245$,
    $zh245$评估 Firestore 安全规则的安全程度。当 Firestore 安全规则更新时使用，以确保生成的规则极其安全且稳健。$zh245$,
    $zh245$评估 Firestore 安全规则的安全程度。当 Firestore 安全规则更新时使用，以确保生成的规则极其安全且稳健。$zh245$
),
(
    'write-tech-spec',
    'zh-CN',
    $zh245$编写技术规格$zh245$,
    $zh245$研究当前代码库和实现约束后，为 Warp 的重要功能编写 TECH.md 规格。适用于用户要求与产品规格关联的技术规格、实施计划或架构文档时。$zh245$,
    $zh245$研究当前代码库和实现约束后，为 Warp 的重要功能编写 TECH.md 规格。适用于用户要求与产品规格关联的技术规格、实施计划或架构文档时。$zh245$
),
(
    'core-web-vitals',
    'zh-CN',
    $zh245$Core Web Vitals 优化$zh245$,
    $zh245$优化 Core Web Vitals（LCP、INP、CLS），改善页面体验与搜索排名。适用于“改善 Core Web Vitals”“修复 LCP”“降低 CLS”“优化 INP”“优化页面体验”或“修复布局偏移”等请求。$zh245$,
    $zh245$优化 Core Web Vitals（LCP、INP、CLS），改善页面体验与搜索排名。适用于“改善 Core Web Vitals”“修复 LCP”“降低 CLS”“优化 INP”“优化页面体验”或“修复布局偏移”等请求。$zh245$
),
(
    'resolve-merge-conflicts',
    'zh-CN',
    $zh245$解决 Git 合并冲突$zh245$,
    $zh245$只提取未解决路径、冲突区块和紧凑 diff 来解决 Git 合并冲突，避免将整个文件载入上下文。适用于 merge、rebase、cherry-pick 或 stash pop 因冲突停止、`git status` 显示未合并路径，或文件中包含冲突标记时。$zh245$,
    $zh245$只提取未解决路径、冲突区块和紧凑 diff 来解决 Git 合并冲突，避免将整个文件载入上下文。适用于 merge、rebase、cherry-pick 或 stash pop 因冲突停止、`git status` 显示未合并路径，或文件中包含冲突标记时。$zh245$
),
(
    'fix-errors',
    'zh-CN',
    $zh245$修复 Warp Rust 错误$zh245$,
    $zh245$修复 Warp Rust 代码库中的编译错误、lint 问题和测试失败。涵盖提交前检查、WASM 特有错误和运行指定测试。适用于用户遇到构建错误、clippy 或 fmt 失败、测试失败，或需要在 PR 前运行或解读提交前检查时。$zh245$,
    $zh245$修复 Warp Rust 代码库中的编译错误、lint 问题和测试失败。涵盖提交前检查、WASM 特有错误和运行指定测试。适用于用户遇到构建错误、clippy 或 fmt 失败、测试失败，或需要在 PR 前运行或解读提交前检查时。$zh245$
),
(
    'fixing-motion-performance',
    'zh-CN',
    $zh245$修复动画性能$zh245$,
    $zh245$审计并修复动画性能问题，包括反复触发布局计算、合成器属性、滚动关联动画和模糊效果。适用于动画卡顿、转场不流畅，或审查 CSS/JS 动画性能时。$zh245$,
    $zh245$审计并修复动画性能问题，包括反复触发布局计算、合成器属性、滚动关联动画和模糊效果。适用于动画卡顿、转场不流畅，或审查 CSS/JS 动画性能时。$zh245$
),
(
    'implement-specs',
    'zh-CN',
    $zh245$按规格实现功能$zh245$,
    $zh245$根据已批准的 PRODUCT.md 和 TECH.md 实现功能，并在实现演进过程中，让规格与代码始终在同一个 PR 中保持一致。适用于产品与技术规格获批、下一步需要构建功能时。$zh245$,
    $zh245$根据已批准的 PRODUCT.md 和 TECH.md 实现功能，并在实现演进过程中，让规格与代码始终在同一个 PR 中保持一致。适用于产品与技术规格获批、下一步需要构建功能时。$zh245$
),
(
    'create-pr',
    'zh-CN',
    $zh245$创建 Pull Request$zh245$,
    $zh245$在 Warp 仓库中为当前分支创建 Pull Request。适用于用户提到开启 PR、创建 Pull Request、提交改动供审查，或准备合并代码时。$zh245$,
    $zh245$在 Warp 仓库中为当前分支创建 Pull Request。适用于用户提到开启 PR、创建 Pull Request、提交改动供审查，或准备合并代码时。$zh245$
),
(
    'update-skill',
    'zh-CN',
    $zh245$创建或更新技能$zh245$,
    $zh245$通过生成、编辑或完善此仓库中的 SKILL.md 文件来创建或更新技能。适用于编写新技能，或修改现有技能的结构、frontmatter 或指导内容时。$zh245$,
    $zh245$通过生成、编辑或完善此仓库中的 SKILL.md 文件来创建或更新技能。适用于编写新技能，或修改现有技能的结构、frontmatter 或指导内容时。$zh245$
),
(
    'diagnose-ci-failures',
    'zh-CN',
    $zh245$诊断 CI 失败$zh245$,
    $zh245$使用 GitHub CLI 诊断 PR 的 CI 失败、提取错误日志并生成修复计划。适用于用户要求检查 CI 状态、提取 CI 问题、分诊测试失败，或调查 PR 构建失败时。$zh245$,
    $zh245$使用 GitHub CLI 诊断 PR 的 CI 失败、提取错误日志并生成修复计划。适用于用户要求检查 CI 状态、提取 CI 问题、分诊测试失败，或调查 PR 构建失败时。$zh245$
),
(
    'gpt-image-2-c21144c1b9',
    'zh-CN',
    $zh245$Pilio GPT Image 2 图像创作$zh245$,
    $zh245$通过统一的 Pilio 开发者 API 使用 Pilio GPT Image 2 创建或编辑图像。适用于用户需要文生图、基于提示词编辑图像、重设风格、转换产品照片，或根据一张或多张本地参考图进行合成时。$zh245$,
    $zh245$通过统一的 Pilio 开发者 API 使用 Pilio GPT Image 2 创建或编辑图像。适用于用户需要文生图、基于提示词编辑图像、重设风格、转换产品照片，或根据一张或多张本地参考图进行合成时。$zh245$
),
(
    'mobile-ios-design',
    'zh-CN',
    $zh245$iOS 原生界面设计$zh245$,
    $zh245$掌握 iOS Human Interface Guidelines 和 SwiftUI 模式，用于构建 iOS 原生应用。适用于设计 iOS 界面、实现 SwiftUI 视图，或确保应用遵循 Apple 设计原则时。$zh245$,
    $zh245$掌握 iOS Human Interface Guidelines 和 SwiftUI 模式，用于构建 iOS 原生应用。适用于设计 iOS 界面、实现 SwiftUI 视图，或确保应用遵循 Apple 设计原则时。$zh245$
),
(
    'idea-refine',
    'zh-CN',
    $zh245$创意提炼与压力测试$zh245$,
    $zh245$通过结构化的发散与收敛思考，将原始想法提炼为清晰、可执行的概念。适用于想法仍然模糊、制定计划前需要检验假设，或希望先扩展选项再收敛到一个方案时。触发请求包括“构思”“完善这个想法”或“压力测试我的计划”。$zh245$,
    $zh245$通过结构化的发散与收敛思考，将原始想法提炼为清晰、可执行的概念。适用于想法仍然模糊、制定计划前需要检验假设，或希望先扩展选项再收敛到一个方案时。触发请求包括“构思”“完善这个想法”或“压力测试我的计划”。$zh245$
),
(
    'using-agent-skills',
    'zh-CN',
    $zh245$发现与调用智能体技能$zh245$,
    $zh245$发现并调用智能体技能。适用于开始会话，或需要确定哪个技能适合当前任务时。这是管理其他所有技能如何被发现和调用的元技能。$zh245$,
    $zh245$发现并调用智能体技能。适用于开始会话，或需要确定哪个技能适合当前任务时。这是管理其他所有技能如何被发现和调用的元技能。$zh245$
),
(
    'baoyu-post-to-weibo',
    'zh-CN',
    $zh245$发布内容到微博$zh245$,
    $zh245$向微博发布内容。支持含文字、图片和视频的普通微博，也支持通过 Chrome CDP 以 Markdown 输入发布头条文章。适用于用户要求“发微博”“发布微博”“分享到微博”“写微博”或“微博头条文章”时。$zh245$,
    $zh245$向微博发布内容。支持含文字、图片和视频的普通微博，也支持通过 Chrome CDP 以 Markdown 输入发布头条文章。适用于用户要求“发布到微博”“发微博”“发布微博”“分享到微博”“写微博”或“微博头条文章”时。$zh245$
),
(
    'laravel-specialist',
    'zh-CN',
    $zh245$Laravel 开发专家$zh245$,
    $zh245$构建和配置 Laravel 10+ 应用，包括创建 Eloquent 模型与关系、实现 Sanctum 身份认证、配置 Horizon 队列、使用 API Resource 设计 RESTful API，以及使用 Livewire 构建响应式界面。$zh245$,
    $zh245$构建和配置 Laravel 10+ 应用，包括创建 Eloquent 模型与关系、实现 Sanctum 身份认证、配置 Horizon 队列、使用 API Resource 设计 RESTful API，以及使用 Livewire 构建响应式界面。适用于创建 Laravel 模型、配置队列 worker、实现 Sanctum 认证流程、构建 Livewire 组件、优化 Eloquent 查询，或为 Laravel 功能编写 Pest/PHPUnit 测试时。$zh245$
),
(
    'brandalf',
    'zh-CN',
    $zh245$Brandalf 品牌资产指南$zh245$,
    $zh245$指导创建、修订和审查带有 Warp 或 Oz 品牌的资产。适用于发布页、文档、HTML/CSS 组件、UI 模型、提示词、社交媒体素材、文案、演示文稿，或任何应在视觉和语调上鲜明体现 Warp 或 Oz 的品牌交付物。$zh245$,
    $zh245$指导创建、修订和审查带有 Warp 或 Oz 品牌的资产。适用于发布页、文档、HTML/CSS 组件、UI 模型、提示词、社交媒体素材、文案、演示文稿，或任何应在视觉和语调上鲜明体现 Warp 或 Oz 的品牌交付物。$zh245$
),
(
    'shadcn-ui-9d55f8588a',
    'zh-CN',
    $zh245$shadcn/ui 组件开发$zh245$,
    $zh245$提供完整的 shadcn/ui 组件库模式，包括安装、配置和实现无障碍 React 组件。适用于设置 shadcn/ui、安装组件、使用 React Hook Form 和 Zod 构建表单，以及使用 Tailwind CSS 自定义主题。$zh245$,
    $zh245$提供完整的 shadcn/ui 组件库模式，包括安装、配置和实现无障碍 React 组件。适用于设置 shadcn/ui、安装组件、使用 React Hook Form 和 Zod 构建表单、使用 Tailwind CSS 自定义主题，或实现按钮、对话框、下拉菜单、表格和复杂表单布局等 UI 模式。$zh245$
),
(
    'dbs-content',
    'zh-CN',
    $zh245$DBS 内容诊断$zh245$,
    $zh245$诊断如何把已经确定的选题做成好内容。用户要求设计内容、检查文案或改善内容表达时使用。$zh245$,
    $zh245$诊断如何把已经确定的选题做成好内容。用户要求设计内容、检查文案或改善内容表达时使用。$zh245$
),
(
    'pr-walkthrough',
    'zh-CN',
    $zh245$Pull Request 可视化走查$zh245$,
    $zh245$为 Pull Request 生成静态交互式 D3 走查页面。适用于用户需要可缩放的 PR 地图、以图形或画布理解 PR，或希望用其他方式可视化 PR 的系统组件、数据流、代码依赖和用户操作时。$zh245$,
    $zh245$为 Pull Request 生成静态交互式 D3 走查页面。适用于用户需要可缩放的 PR 地图、以图形或画布理解 PR，或希望用其他方式可视化 PR 的系统组件、数据流、代码依赖和用户操作时。$zh245$
),
(
    'dbs-diagnosis',
    'zh-CN',
    $zh245$DBS 商业诊断$zh245$,
    $zh245$用问诊和体检两种模式诊断商业问题与商业模式。用户希望拆解业务、检查商业模式或消解具体商业困境时使用。$zh245$,
    $zh245$用问诊和体检两种模式诊断商业问题与商业模式。用户希望拆解业务、检查商业模式或消解具体商业困境时使用。$zh245$
),
(
    'dbs-benchmark',
    'zh-CN',
    $zh245$DBS 对标分析$zh245$,
    $zh245$用五重过滤法寻找值得模仿的对标，并排除主体差异造成的噪音。用户要求找对标、选择模仿对象或分析竞争参照时使用。$zh245$,
    $zh245$用五重过滤法寻找值得模仿的对标，并排除主体差异造成的噪音。用户要求找对标、选择模仿对象或分析竞争参照时使用。$zh245$
),
(
    'stripe-directory',
    'zh-CN',
    $zh245$Stripe Directory 服务目录$zh245$,
    $zh245$适用于用户希望针对特定行业、工作流、痛点、能力或待完成任务，寻找企业、软件、服务商或合作伙伴时；智能体需要以编程方式购买或使用服务时也应使用。通过 Stripe Directory 建立简短且相关的候选名单。$zh245$,
    $zh245$适用于用户希望针对特定行业、工作流、痛点、能力或待完成任务，寻找企业、软件、服务商或合作伙伴时；智能体需要以编程方式购买或使用服务时也应使用。即使用户没有明确提到 Stripe Directory，也应使用它建立简短且相关的候选名单。$zh245$
),
(
    'git-workflow-and-versioning',
    'zh-CN',
    $zh245$Git 工作流与版本管理$zh245$,
    $zh245$规范 Git 工作流实践。适用于进行任何代码改动，以及提交、创建分支、解决冲突或组织多个并行工作流时。也适用于发布版本、选择语义化版本升级级别、打标签或编写变更日志。$zh245$,
    $zh245$规范 Git 工作流实践。适用于进行任何代码改动，以及提交、创建分支、解决冲突或组织多个并行工作流时。也适用于发布版本、选择语义化版本升级级别、打标签或编写变更日志。$zh245$
),
(
    'dbs-deconstruct',
    'zh-CN',
    $zh245$DBS 概念拆解$zh245$,
    $zh245$用维特根斯坦与奥派经济学方法拆解模糊的商业概念。用户要求解释一个词、澄清概念边界或识别概念混用时使用。$zh245$,
    $zh245$用维特根斯坦与奥派经济学方法拆解模糊的商业概念。用户要求解释一个词、澄清概念边界或识别概念混用时使用。$zh245$
),
(
    'agent-browser-d62e8bba15',
    'zh-CN',
    $zh245$智能体浏览器自动化$zh245$,
    $zh245$通过 inference.sh 为 AI 智能体提供浏览器自动化。可导航网页、使用 @e 引用与元素交互、截图和录制视频；支持 Web 抓取、填写表单、点击、输入、拖放、上传文件和执行 JavaScript。适用于 Web 自动化、数据提取、测试、智能体浏览和研究。$zh245$,
    $zh245$通过 inference.sh 为 AI 智能体提供浏览器自动化。可导航网页、使用 @e 引用与元素交互、截图和录制视频。能力包括 Web 抓取、填写表单、点击、输入、拖放、上传文件和执行 JavaScript。适用于 Web 自动化、数据提取、测试、智能体浏览和研究。触发词包括：浏览器、Web 自动化、抓取、导航、点击、填写表单、截图、浏览 Web、playwright、无头浏览器、Web 智能体、上网和录制视频。$zh245$
),
(
    'browser-testing-with-devtools',
    'zh-CN',
    $zh245$Chrome DevTools 浏览器测试$zh245$,
    $zh245$通过 Chrome DevTools MCP 在真实浏览器中测试。适用于构建或调试任何在浏览器中运行的内容，以及需要检查 DOM、捕获控制台错误、分析网络请求、分析性能，或使用真实运行时数据验证视觉输出时。$zh245$,
    $zh245$通过 Chrome DevTools MCP 在真实浏览器中测试。适用于构建或调试任何在浏览器中运行的内容，以及需要检查 DOM、捕获控制台错误、分析网络请求、分析性能，或使用真实运行时数据验证视觉输出时。需要预先配置 chrome-devtools MCP 服务器。$zh245$
),
(
    'prompt-engineering-patterns',
    'zh-CN',
    $zh245$提示词工程模式$zh245$,
    $zh245$适用于用户要求“优化提示词”“提升提示词效果”“设计提示词模板”“编写更好的提示词”“调试提示词问题”“使用思维链”“结构化提示”或“少样本提示”，以及希望为生产级 LLM 应用采用高级提示词工程模式时。$zh245$,
    $zh245$适用于用户要求“优化提示词”“提升提示词效果”“设计提示词模板”“编写更好的提示词”“调试提示词问题”“使用思维链”“结构化提示”或“少样本提示”，以及希望为生产级 LLM 应用采用高级提示词工程模式时。$zh245$
),
(
    'ci-cd-and-automation',
    'zh-CN',
    $zh245$CI/CD 流水线自动化$zh245$,
    $zh245$自动配置 CI/CD 流水线。适用于设置或修改构建与部署流水线，以及需要自动执行质量门禁、在 CI 中配置测试运行器或制定部署策略时。$zh245$,
    $zh245$自动配置 CI/CD 流水线。适用于设置或修改构建与部署流水线，以及需要自动执行质量门禁、在 CI 中配置测试运行器或制定部署策略时。$zh245$
),
(
    'error-handling-patterns',
    'zh-CN',
    $zh245$错误处理模式$zh245$,
    $zh245$掌握跨语言错误处理模式，包括异常、Result 类型、错误传播和优雅降级，以构建有韧性的应用。适用于实现错误处理、设计 API 或提升应用可靠性时。$zh245$,
    $zh245$掌握跨语言错误处理模式，包括异常、Result 类型、错误传播和优雅降级，以构建有韧性的应用。适用于实现错误处理、设计 API 或提升应用可靠性时。$zh245$
),
(
    'council',
    'zh-CN',
    $zh245$多模型智能体评议会$zh245$,
    $zh245$运行模型多样化的子智能体评议会，从多个角度调查同一问题、比较发现并形成最终建议。适用于用户要求评议会、第二意见、由多个智能体/模型评估一个问题、并行调查、红队/蓝队对比，或协助在多个技术方案间做决定时。$zh245$,
    $zh245$运行模型多样化的子智能体评议会，从多个角度调查同一问题、比较发现并形成最终建议。适用于用户要求评议会、第二意见、由多个智能体/模型评估一个问题、并行调查、红队/蓝队对比，或协助在多个相互竞争的技术方案间做决定时。$zh245$
),
(
    'pnpm',
    'zh-CN',
    $zh245$pnpm 包管理$zh245$,
    $zh245$采用严格依赖解析的 Node.js 包管理器。适用于运行 pnpm 专用命令、通过 pnpm-workspace.yaml 配置工作区，或使用 catalog、patch、override、配置依赖和全局虚拟存储管理依赖时。$zh245$,
    $zh245$采用严格依赖解析的 Node.js 包管理器。适用于运行 pnpm 专用命令、通过 pnpm-workspace.yaml 配置工作区，或使用 catalog、patch、override、配置依赖和全局虚拟存储管理依赖时。$zh245$
),
(
    'dbs-hook',
    'zh-CN',
    $zh245$DBS 短视频开头诊断$zh245$,
    $zh245$诊断短视频开头的问题并生成优化方案。用户要求修改开头、提高开场吸引力或降低开头流失时使用。$zh245$,
    $zh245$诊断短视频开头的问题并生成优化方案。用户要求修改开头、提高开场吸引力或降低开头流失时使用。$zh245$
),
(
    'shipping-and-launch',
    'zh-CN',
    $zh245$生产发布准备$zh245$,
    $zh245$为生产发布做好准备。适用于准备部署到生产环境，以及需要发布前检查清单、配置监控、规划分阶段发布或制定回滚策略时。$zh245$,
    $zh245$为生产发布做好准备。适用于准备部署到生产环境，以及需要发布前检查清单、配置监控、规划分阶段发布或制定回滚策略时。$zh245$
),
(
    'java-springboot',
    'zh-CN',
    $zh245$Spring Boot 开发实践$zh245$,
    $zh245$获取使用 Spring Boot 开发应用的最佳实践。$zh245$,
    $zh245$获取使用 Spring Boot 开发应用的最佳实践。$zh245$
),
(
    'best-practices',
    'zh-CN',
    $zh245$现代 Web 开发最佳实践$zh245$,
    $zh245$应用现代 Web 开发最佳实践，提升安全性、兼容性和代码质量。适用于“应用最佳实践”“安全审计”“现代化代码”“代码质量审查”或“检查漏洞”等请求。$zh245$,
    $zh245$应用现代 Web 开发最佳实践，提升安全性、兼容性和代码质量。适用于“应用最佳实践”“安全审计”“现代化代码”“代码质量审查”或“检查漏洞”等请求。$zh245$
),
(
    'sanity-best-practices',
    'zh-CN',
    $zh245$Sanity 开发最佳实践$zh245$,
    $zh245$Sanity 开发最佳实践，涵盖 schema 设计、GROQ 查询、TypeGen、Visual Editing、图像、Portable Text、Studio 结构、本地化、迁移、Sanity Functions、Webhook、Blueprints，以及 Next.js、Nuxt、Astro、Remix、SvelteKit、Angular 等框架集成。$zh245$,
    $zh245$Sanity 开发最佳实践，涵盖 schema 设计、GROQ 查询、TypeGen、Visual Editing、图像、Portable Text、Studio 结构、本地化、迁移、Sanity Functions、Webhook、Blueprints，以及 Next.js、Nuxt、Astro、Remix、SvelteKit、Angular、Hydrogen 和 App SDK 等框架集成。凡涉及 Sanity schema、defineType 或 defineField、GROQ 或 defineQuery、内容建模、Presentation 或预览配置、Sanity 驱动的前端集成、事件驱动内容自动化、documentEventHandler、defineDocumentFunction、defineMediaLibraryAssetFunction、@sanity/functions、@sanity/blueprints、sanity.blueprint.ts，或审查和修复 Sanity 代码库时，都应使用本技能。$zh245$
),
(
    'source-driven-development',
    'zh-CN',
    $zh245$官方来源驱动开发$zh245$,
    $zh245$让每项实现决策都以官方文档为依据。适用于希望获得权威、有来源引用且不采用过时模式的代码，或使用任何重视正确性的框架和库进行构建时。$zh245$,
    $zh245$让每项实现决策都以官方文档为依据。适用于希望获得权威、有来源引用且不采用过时模式的代码，或使用任何重视正确性的框架和库进行构建时。$zh245$
),
(
    'baoyu-youtube-transcript',
    'zh-CN',
    $zh245$宝玉 YouTube 字幕提取$zh245$,
    $zh245$通过 URL 或视频 ID 下载 YouTube 视频转录稿/字幕和封面图。支持多语言、翻译、章节和说话人识别，并缓存原始数据以便快速重新格式化。适用于获取 YouTube 转录稿、下载字幕或提取视频封面。$zh245$,
    $zh245$通过 URL 或视频 ID 下载 YouTube 视频转录稿/字幕和封面图。支持多语言、翻译、章节和说话人识别，并缓存原始数据以便快速重新格式化。适用于用户要求“获取 YouTube 转录稿”“下载字幕”“获取字幕”“YouTube 字幕”“YouTube 封面”“视频封面”，或提供 YouTube URL 并希望提取转录稿/字幕文本或封面图时。$zh245$
),
(
    'dbs-action',
    'zh-CN',
    $zh245$DBS 行动阻滞诊断$zh245$,
    $zh245$用阿德勒心理学框架诊断执行阻滞。用户知道该做什么却迟迟不做、反复拖延或行动中断时使用。$zh245$,
    $zh245$用阿德勒心理学框架诊断执行阻滞。用户知道该做什么却迟迟不做、反复拖延或行动中断时使用。$zh245$
),
(
    'deprecation-and-migration',
    'zh-CN',
    $zh245$弃用与迁移管理$zh245$,
    $zh245$管理弃用和迁移。适用于移除旧系统、API 或功能，将用户从一种实现迁移到另一种实现，或决定维护还是终止现有代码时。$zh245$,
    $zh245$管理弃用和迁移。适用于移除旧系统、API 或功能，将用户从一种实现迁移到另一种实现，或决定维护还是终止现有代码时。$zh245$
),
(
    'check-impl-against-spec',
    'zh-CN',
    $zh245$对照规格检查实现$zh245$,
    $zh245$将 Pull Request 的实现与 spec_context.md 中的规格上下文进行比较，并把任何实质性不匹配写入 review.json。适用于 PR 审查期间已有获批规格或仓库规格上下文时。$zh245$,
    $zh245$将 Pull Request 的实现与 spec_context.md 中的规格上下文进行比较，并把任何实质性不匹配写入 review.json。适用于 PR 审查期间已有获批规格或仓库规格上下文时。$zh245$
),
(
    'reproduce-bug-report',
    'zh-CN',
    $zh245$复现 UI 缺陷报告$zh245$,
    $zh245$启动具有计算机操作能力的 Oz 云智能体，复现以 UI 为主的缺陷报告、采集视觉证据（默认录屏），并报告复现结果。适用于调查 issue、工单、支持报告或提示词中描述的特定交互或视觉缺陷时。$zh245$,
    $zh245$启动具有计算机操作能力的 Oz 云智能体，复现以 UI 为主的缺陷报告、采集视觉证据（默认录屏），并报告复现结果。适用于调查 issue、工单、支持报告或提示词中描述的特定交互或视觉缺陷时。$zh245$
),
(
    'marketing-loops',
    'zh-CN',
    $zh245$循环营销工作流$zh245$,
    $zh245$适用于用户希望建立定期、自运行的营销工作流：由 AI 智能体按每周、每日或事件触发的节奏重复执行，而不是只完成一次任务。用户提到“营销循环”“循环营销工作流”“自动化营销”等概念时也应使用。$zh245$,
    $zh245$适用于用户希望建立定期、自运行的营销工作流：由 AI 智能体按每周、每日或事件触发的节奏重复执行，而不是只完成一次任务。用户提到“营销循环”“循环营销工作流”“自动化我的营销”“营销自动驾驶”“每周营销复盘”“广告疲劳检查”“内容刷新循环”“流失监控”“排名下降提醒”“常驻营销”“营销自动化工作流”或“每周运行此任务”时也应使用。本技能用于选择、调整和安排一个持续运行并编排其他营销技能的营销循环。一次性营销创意请参阅 marketing-ideas；专门的实验循环请参阅 ab-testing。$zh245$
),
(
    'opencli-usage',
    'zh-CN',
    $zh245$OpenCLI 使用总览$zh245$,
    $zh245$在任何 OpenCLI 会话开始时使用。这是 `opencli` 功能、adapter 发现方式、通用 flag 与输出格式，以及下一步应加载哪个专项技能的顶层地图。智能体询问“opencli 能做什么？”或“如何找到正确命令？”时应指向此技能。$zh245$,
    $zh245$在任何 OpenCLI 会话开始时使用。这是 `opencli` 功能、adapter 发现方式、通用 flag 与输出格式，以及下一步应加载哪个专项技能的顶层地图。智能体询问“opencli 能做什么？”或“如何找到正确命令？”时应指向此技能。$zh245$
),
(
    'python-design-patterns',
    'zh-CN',
    $zh245$Python 设计模式$zh245$,
    $zh245$Python 设计模式，包括 KISS、关注点分离、单一职责和组合优于继承。适用于从零设计新服务或组件并选择职责分层方式，或重构过大的上帝类或单体函数时。$zh245$,
    $zh245$Python 设计模式，包括 KISS、关注点分离、单一职责和组合优于继承。适用于从零设计新服务或组件并选择职责分层方式；重构膨胀过大的上帝类或单体函数；决定新增抽象还是接受重复；审查 Pull Request 中紧耦合、内部类型泄漏等结构问题；为新类层次选择继承或组合；以及代码库因 I/O 与业务逻辑纠缠而难以测试时。$zh245$
),
(
    'gemini-api-dev',
    'zh-CN',
    $zh245$Gemini API 应用开发$zh245$,
    $zh245$适用于使用 Gemini API 托管模型（包括 Gemini 和 Gemma 4）构建应用，处理多模态内容（文本、图像、音频、视频）、实现函数调用、使用结构化输出，或需要最新模型规格时。涵盖 SDK 使用、模型选择和 API 能力。$zh245$,
    $zh245$适用于使用 Gemini API 托管模型（包括 Gemini 和 Gemma 4）构建应用，处理多模态内容（文本、图像、音频、视频）、实现函数调用、使用结构化输出，或需要最新模型规格时。涵盖 SDK 使用（Python 的 google-genai、JavaScript/TypeScript 的 @google/genai、Java 的 com.google.genai:google-genai、Go 的 google.golang.org/genai）、模型选择和 API 能力。$zh245$
),
(
    'firecrawl-research-index',
    'zh-CN',
    $zh245$Firecrawl 研究论文索引$zh245$,
    $zh245$在 Firecrawl 研究论文索引中查找能回答研究问题的论文。该论文摘要语料库以生物医学和生命科学文献（PubMed、bioRxiv、medRxiv）为主，同时包含 arXiv 上计算机科学、物理和数学领域的预印本，并采用语义搜索与扩展。$zh245$,
    $zh245$在 Firecrawl 研究论文索引中查找能回答研究问题的论文。该论文摘要语料库以生物医学和生命科学文献（PubMed、bioRxiv、medRxiv）为主，同时包含 arXiv 上计算机科学、物理和数学领域的预印本；它使用语义搜索、语义与结构扩展及正文验证。适用于任何文献查找和论文检索任务，包括临床、生物医学、药物、基因、疾病和其他生命科学问题，无论答案是一篇论文还是完整的多篇论文集合。只能通过 `firecrawl_research_*` MCP 工具或 `firecrawl research` CLI 子命令访问该索引。调用 `firecrawl_search` 并将 `categories` 选项设为 `["research"]` 是另一项功能：它只把普通 Web 搜索筛选到研究相关网站（列表包括 PubMed、bioRxiv、medRxiv、arXiv 和出版商网站），返回这些网站的页面结果，并不会查询本索引中的论文记录。$zh245$
),
(
    'aso-audit',
    'zh-CN',
    $zh245$应用商店 ASO 审计$zh245$,
    $zh245$适用于用户希望审计或优化 App Store 或 Google Play 商品页，以及提到“ASO 审计”“应用商店优化”“优化我的应用商品页”“提升应用可见度”“应用商店排名”“审计我的商品页”等请求时。$zh245$,
    $zh245$适用于用户希望审计或优化 App Store 或 Google Play 商品页。用户提到“ASO 审计”“应用商店优化”“优化我的应用商品页”“提升应用可见度”“应用商店排名”“审计我的商品页”“为什么没人下载我的应用”“提升应用转化率”“应用关键词优化”或“将我的应用与竞品比较”时也应使用。当用户提供 App Store 或 Google Play URL 并希望改进商品页时使用。$zh245$
),
(
    'design-system',
    'zh-CN',
    $zh245$设计系统与令牌架构$zh245$,
    $zh245$令牌架构、组件规范和幻灯片生成。涵盖三层令牌（原始→语义→组件）、CSS 变量、间距与排版尺度、组件规范和战略性幻灯片创作。适用于设计令牌、系统化设计和符合品牌规范的演示文稿。$zh245$,
    $zh245$令牌架构、组件规范和幻灯片生成。涵盖三层令牌（原始→语义→组件）、CSS 变量、间距与排版尺度、组件规范和战略性幻灯片创作。适用于设计令牌、系统化设计和符合品牌规范的演示文稿。$zh245$
),
(
    'web-design-guidelines-08dca22aaa',
    'zh-CN',
    $zh245$Web 界面规范审查$zh245$,
    $zh245$审查 UI 代码是否符合 Web Interface Guidelines。适用于“审查我的 UI”“检查无障碍性”“审计设计”“审查 UX”或“按最佳实践检查我的网站”等请求。$zh245$,
    $zh245$审查 UI 代码是否符合 Web Interface Guidelines。适用于“审查我的 UI”“检查无障碍性”“审计设计”“审查 UX”或“按最佳实践检查我的网站”等请求。$zh245$
),
(
    'dbs-xhs-title',
    'zh-CN',
    $zh245$DBS 小红书标题生成$zh245$,
    $zh245$从 75 个经过验证的小红书标题公式中选择并生成合适标题。用户要求起小红书标题、改标题或选择标题公式时使用。$zh245$,
    $zh245$从 75 个经过验证的小红书标题公式中选择并生成合适标题。用户要求起小红书标题、改标题或选择标题公式时使用。$zh245$
),
(
    'playwright-generate-test',
    'zh-CN',
    $zh245$生成 Playwright 测试$zh245$,
    $zh245$通过 Playwright MCP 根据场景生成 Playwright 测试。$zh245$,
    $zh245$通过 Playwright MCP 根据场景生成 Playwright 测试。$zh245$
),
(
    'baseline-ui',
    'zh-CN',
    $zh245$UI 基础清理$zh245$,
    $zh245$通过修复间距、层级、排版和细小布局问题，快速清理粗糙的 UI 代码。适用于界面需要快速整理或润色时。$zh245$,
    $zh245$通过修复间距、层级、排版和细小布局问题，快速清理粗糙的 UI 代码。适用于界面需要快速整理或润色时。$zh245$
),
(
    'modern-javascript-patterns',
    'zh-CN',
    $zh245$现代 JavaScript 模式$zh245$,
    $zh245$掌握 ES6+ 特性，包括 async/await、解构、展开运算符、箭头函数、Promise、模块、迭代器、生成器和函数式编程模式，以编写简洁高效的 JavaScript 代码。适用于重构遗留代码、实施现代模式或优化 JavaScript 应用。$zh245$,
    $zh245$掌握 ES6+ 特性，包括 async/await、解构、展开运算符、箭头函数、Promise、模块、迭代器、生成器和函数式编程模式，以编写简洁高效的 JavaScript 代码。适用于重构遗留代码、实施现代模式或优化 JavaScript 应用。$zh245$
),
(
    'vue-router-best-practices',
    'zh-CN',
    $zh245$Vue Router 最佳实践$zh245$,
    $zh245$Vue Router 4 模式、导航守卫、路由参数，以及路由与组件生命周期之间的交互。$zh245$,
    $zh245$Vue Router 4 模式、导航守卫、路由参数，以及路由与组件生命周期之间的交互。$zh245$
),
(
    'javascript-testing-patterns',
    'zh-CN',
    $zh245$JavaScript 测试模式$zh245$,
    $zh245$使用 Jest、Vitest 和 Testing Library 实施全面的测试策略，覆盖单元测试、集成测试和端到端测试，并结合 mock、fixture 和测试驱动开发。适用于编写 JavaScript/TypeScript 测试、搭建测试基础设施或实施 TDD/BDD 工作流。$zh245$,
    $zh245$使用 Jest、Vitest 和 Testing Library 实施全面的测试策略，覆盖单元测试、集成测试和端到端测试，并结合 mock、fixture 和测试驱动开发。适用于编写 JavaScript/TypeScript 测试、搭建测试基础设施或实施 TDD/BDD 工作流。$zh245$
),
(
    'design',
    'zh-CN',
    $zh245$综合设计工具$zh245$,
    $zh245$综合设计技能：品牌形象、设计令牌、UI 样式、Logo 生成（55 种风格、Gemini AI）、企业形象计划（50 项交付物、CIP 模型）、HTML 演示文稿（Chart.js）、横幅设计（22 种风格，覆盖社交媒体/广告/Web/印刷）和图标设计（15 种风格、SVG）。$zh245$,
    $zh245$综合设计技能：品牌形象、设计令牌、UI 样式、Logo 生成（55 种风格、Gemini AI）、企业形象计划（50 项交付物、CIP 模型）、HTML 演示文稿（Chart.js）、横幅设计（22 种风格，覆盖社交媒体/广告/Web/印刷）、图标设计（15 种风格、SVG、Gemini 3.1 Pro）以及社交媒体图片（HTML→截图、多平台）。可执行：设计 Logo、创建 CIP、生成模型、制作幻灯片、设计横幅、生成图标、创建社交媒体图片、品牌形象与设计系统。支持平台：Facebook、Twitter、LinkedIn、YouTube、Instagram、Pinterest、TikTok、Threads、Google Ads。$zh245$
),
(
    'interview-me',
    'zh-CN',
    $zh245$逐问式需求访谈$zh245$,
    $zh245$挖掘用户真正想要的内容，而不是他们以为自己应该想要的内容。通过每次一个问题的访谈，直至对底层意图达到约 95% 的把握。适用于请求信息不足、用户明确要求访谈或压力测试，或在计划、规格和代码出现前发现自己正默默补全模糊需求时。$zh245$,
    $zh245$挖掘用户真正想要的内容，而不是他们以为自己应该想要的内容。通过每次一个问题的访谈，直至对底层意图达到约 95% 的把握。适用于请求信息不足（例如只说“为我构建 X”，却没说明“为谁”或“为何现在做”）、用户明确要求“采访我”“严格盘问我”“我们确定吗？”“压力测试我的想法”，或在任何计划、规格和代码出现前发现自己正默默补全模糊需求时。$zh245$
),
(
    'dbs-ai-check',
    'zh-CN',
    $zh245$DBS AI 写作痕迹检测$zh245$,
    $zh245$扫描文案中的 AI 写作特征并输出检测报告，默认只诊断不改写。用户要求检查 AI 味、AI 痕迹或机器化表达时使用。$zh245$,
    $zh245$扫描文案中的 AI 写作特征并输出检测报告，默认只诊断不改写。用户要求检查 AI 味、AI 痕迹或机器化表达时使用。$zh245$
),
(
    'the-news',
    'zh-CN',
    $zh245$全球头版新闻$zh245$,
    $zh245$为智能体提供 20 个国家头版新闻的实时与历史访问能力，用于突发新闻、时事和跨媒体比较分析。$zh245$,
    $zh245$为智能体提供 20 个国家头版新闻的实时与历史访问能力，用于突发新闻、时事和跨媒体比较分析。$zh245$
),
(
    'opencli-browser',
    'zh-CN',
    $zh245$OpenCLI 浏览器控制$zh245$,
    $zh245$适用于智能体需要通过 opencli 驱动真实 Chrome 窗口，包括检查页面、填写表单、操作已登录流程或临时提取数据时。涵盖选择器优先的目标契约、复合表单字段、失效引用处理、网络捕获，以及 CLI 返回的智能体原生信封结构。$zh245$,
    $zh245$适用于智能体需要通过 opencli 驱动真实 Chrome 窗口，包括检查页面、填写表单、操作已登录流程或临时提取数据时。涵盖选择器优先的目标契约、复合表单字段、失效引用处理、网络捕获，以及 CLI 返回的智能体原生信封结构。不适用于编写 adapter；该任务请参阅 opencli-adapter-author。$zh245$
),
(
    'doubt-driven-development',
    'zh-CN',
    $zh245$质疑驱动开发$zh245$,
    $zh245$让每一项非琐碎决策在成立前都接受一次使用全新上下文的对抗性审查。适用于正确性比速度更重要、处理陌生代码、风险较高（生产环境、安全敏感逻辑、不可逆操作），或现在验证一个看似确定的结果比日后调试更省成本时。$zh245$,
    $zh245$让每一项非琐碎决策在成立前都接受一次使用全新上下文的对抗性审查。适用于正确性比速度更重要、处理陌生代码、风险较高（生产环境、安全敏感逻辑、不可逆操作），或现在验证一个看似确定的结果比日后调试更省成本时。$zh245$
),
(
    'agent-development',
    'zh-CN',
    $zh245$Claude Code 智能体开发$zh245$,
    $zh245$适用于用户要求“创建智能体”“添加智能体”“编写子智能体”“智能体 frontmatter”“何时使用 description”“智能体示例”“智能体工具”“智能体颜色”“自主智能体”，或需要智能体结构、系统提示词与触发条件方面的指导时。$zh245$,
    $zh245$适用于用户要求“创建智能体”“添加智能体”“编写子智能体”“智能体 frontmatter”“何时使用 description”“智能体示例”“智能体工具”“智能体颜色”“自主智能体”，或需要有关智能体结构、系统提示词、触发条件，以及 Claude Code 插件智能体开发最佳实践的指导时。$zh245$
),
(
    'respond-to-pr-comments-in-blocklist',
    'zh-CN',
    $zh245$回复并解决 PR 审查评论$zh245$,
    $zh245$以交互方式逐条引导用户处理 PR 审查评论，收集每条评论的决定，然后在用户批准预览后，将智能体撰写的回复发布到 GitHub 并解决审查线程。仅适用于用户希望回复或解决 GitHub 审查线程时。$zh245$,
    $zh245$以交互方式逐条引导用户处理 PR 审查评论，收集每条评论的决定，然后在用户批准预览后，将智能体撰写的回复发布到 GitHub 并解决审查线程。仅适用于用户希望回复或解决 GitHub 审查线程时。如果用户只想获取或显示评论，请使用 `pr-comments`；如果只想修改代码而不向 GitHub 回发任何内容，也应跳过本技能。$zh245$
),
(
    'animate-afd97d0495',
    'zh-CN',
    $zh245$从零构建动画$zh245$,
    $zh245$从零构建动画，并按决定动画观感的顺序作出选择：是否应该动画、目的是什么、用什么工具和属性、采用何种曲线与时长、如何被打断以及如何退出；随后编写实现。适用于为元素添加动画或动态、让组件更有生气，或构建转场时。$zh245$,
    $zh245$从零构建动画，并按决定动画观感的顺序作出选择：是否应该动画、目的是什么、用什么工具和属性、采用何种曲线与时长、如何被打断以及如何退出；随后编写实现。适用于为元素添加动画或动态、让组件更有生气，或构建转场时。若要评议现有动画，请使用 review-animations；若要审计整个代码库，请使用 improve-animations。$zh245$
),
(
    'create-readme',
    'zh-CN',
    $zh245$创建项目 README$zh245$,
    $zh245$为项目创建 README.md 文件。$zh245$,
    $zh245$为项目创建 README.md 文件。$zh245$
),
(
    'rust-async-patterns',
    'zh-CN',
    $zh245$Rust 异步编程模式$zh245$,
    $zh245$掌握使用 Tokio、async trait、错误处理和并发模式进行 Rust 异步编程。适用于构建异步 Rust 应用、实现并发系统或调试异步代码时。$zh245$,
    $zh245$掌握使用 Tokio、async trait、错误处理和并发模式进行 Rust 异步编程。适用于构建异步 Rust 应用、实现并发系统或调试异步代码时。$zh245$
),
(
    'validate-changes-match-specs',
    'zh-CN',
    $zh245$验证改动符合规格$zh245$,
    $zh245$验证分支或 Pull Request 的实现是否符合新引入的产品、技术、安全及相关规格。适用于审查或收尾规格驱动的改动，并解决已提交规格与实现之间的不匹配时。$zh245$,
    $zh245$验证分支或 Pull Request 的实现是否符合新引入的产品、技术、安全及相关规格。适用于审查或收尾规格驱动的改动，并解决已提交规格与实现之间的不匹配时。$zh245$
),
(
    'lark-whiteboard-cli',
    'zh-CN',
    $zh245$飞书画板 CLI 绘图$zh245$,
    $zh245$当用户要求或使用飞书画板绘制架构图、流程图、思维导图、时序图或其他可视化图表时使用本技能，作为使用 whiteboard-cli 设计图表布局的指南。$zh245$,
    $zh245$当用户要求或使用飞书画板绘制架构图、流程图、思维导图、时序图或其他可视化图表时使用本技能，作为使用 whiteboard-cli 设计图表布局的指南。$zh245$
),
(
    'dbs-chatroom',
    'zh-CN',
    $zh245$DBS 专家聊天室$zh245$,
    $zh245$根据话题推荐或接受用户指定的专家，模拟多角色对话并总结分歧。用户要求定向聊天室、专家讨论或继续当前聊天室时使用。$zh245$,
    $zh245$根据话题推荐或接受用户指定的专家，模拟多角色对话并总结分歧。用户要求定向聊天室、专家讨论或继续当前聊天室时使用。$zh245$
)
ON CONFLICT (slug, locale) DO UPDATE
SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
