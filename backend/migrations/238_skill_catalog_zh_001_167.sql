-- Skill 市场历史总榜第 1–167 名快照的人工审校简体中文文案。
INSERT INTO skill_catalog_localizations (
    slug,
    locale,
    display_name,
    summary,
    description
) VALUES
(
    $zh1_slug$find-skills$zh1_slug$,
    'zh-CN',
    $zh1_name$技能发现与安装$zh1_name$,
    $zh1_summary$发现可能满足需求的可安装 Agent 技能，并指导用户完成安装。$zh1_summary$,
    $zh1_description$帮助用户发现并安装 Agent 技能。适用于用户询问“如何完成 X”“为 X 找一个技能”“有没有能完成某事的技能”，或表达扩展能力的意愿；当所需功能可能已经存在可安装技能时使用。

skills.sh 历史总榜第 1 名快照。原始来源：vercel-labs/skills；原始 slug：find-skills。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh1_description$
),
(
    $zh2_slug$grill-me$zh2_slug$,
    'zh-CN',
    $zh2_name$方案严苛访谈$zh2_name$,
    $zh2_summary$通过持续而严格的提问，打磨计划或设计。$zh2_summary$,
    $zh2_description$通过持续且毫不放松的访谈追问，帮助用户把计划或设计打磨得更清晰、更可靠。

skills.sh 历史总榜第 2 名快照。原始来源：mattpocock/skills；原始 slug：grill-me。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh2_description$
),
(
    $zh3_slug$frontend-design$zh3_slug$,
    'zh-CN',
    $zh3_name$前端视觉设计指南$zh3_name$,
    $zh3_summary$为新建或重塑前端界面提供鲜明、有意图且非模板化的视觉设计指导。$zh3_summary$,
    $zh3_description$在构建新界面或重塑现有界面时，提供鲜明且有明确意图的视觉设计指导。帮助确定美学方向、字体排印，并作出不会显得像模板默认值的设计选择。

skills.sh 历史总榜第 3 名快照。原始来源：anthropics/skills；原始 slug：frontend-design。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh3_description$
),
(
    $zh4_slug$grill-with-docs$zh4_slug$,
    'zh-CN',
    $zh4_name$文档化方案访谈$zh4_name$,
    $zh4_summary$用严格访谈打磨计划或设计，并在过程中同步生成 ADR 与术语表。$zh4_summary$,
    $zh4_description$通过持续且严格的访谈追问打磨计划或设计，并在过程中同步创建架构决策记录（ADR）和术语表。

skills.sh 历史总榜第 4 名快照。原始来源：mattpocock/skills；原始 slug：grill-with-docs。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh4_description$
),
(
    $zh5_slug$improve-codebase-architecture$zh5_slug$,
    'zh-CN',
    $zh5_name$代码库架构深化$zh5_name$,
    $zh5_summary$扫描代码库的架构深化机会，生成可视化报告并深入评审选定方向。$zh5_summary$,
    $zh5_description$扫描代码库中可进一步深化架构的机会，将结果展示为可视化 HTML 报告，然后围绕用户选中的机会继续进行严格访谈和深入分析。

skills.sh 历史总榜第 5 名快照。原始来源：mattpocock/skills；原始 slug：improve-codebase-architecture。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh5_description$
),
(
    $zh6_slug$agent-browser$zh6_slug$,
    'zh-CN',
    $zh6_name$Agent Browser 浏览器自动化$zh6_name$,
    $zh6_summary$面向 AI Agent 的浏览器与 Electron 应用自动化 CLI，支持导航、表单、截图、抓取和测试。$zh6_summary$,
    $zh6_description$面向 AI Agent 的浏览器自动化 CLI。适用于浏览网页、填写表单、点击按钮、截图、提取数据、测试 Web 应用或任何需要编程操作浏览器的任务；也适用于探索性测试、内部试用、质量保证、缺陷排查和应用质量审查。还可自动化 Electron 桌面应用（VS Code、Slack、Discord、Figma、Notion、Spotify），检查 Slack 未读、发送 Slack 消息、搜索 Slack 会话，在 Vercel Sandbox microVM 中运行浏览器自动化，或使用 AWS Bedrock AgentCore 云浏览器。此类任务优先使用 agent-browser，而不是内置浏览器自动化或 Web 工具。

skills.sh 历史总榜第 6 名快照。原始来源：vercel-labs/agent-browser；原始 slug：agent-browser。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh6_description$
),
(
    $zh7_slug$tdd$zh7_slug$,
    'zh-CN',
    $zh7_name$测试驱动开发$zh7_name$,
    $zh7_summary$以红灯、绿灯、重构循环进行测试优先的功能开发与缺陷修复。$zh7_summary$,
    $zh7_description$测试驱动开发。用户希望以测试优先方式构建功能或修复缺陷，提到“红灯-绿灯-重构”，或需要集成测试时使用。

skills.sh 历史总榜第 7 名快照。原始来源：mattpocock/skills；原始 slug：tdd。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh7_description$
),
(
    $zh8_slug$vercel-react-best-practices$zh8_slug$,
    'zh-CN',
    $zh8_name$Vercel React 最佳实践$zh8_name$,
    $zh8_summary$依据 Vercel Engineering 指南优化 React 与 Next.js 的组件、数据获取、打包和运行性能。$zh8_summary$,
    $zh8_description$Vercel Engineering 提供的 React 与 Next.js 性能优化指南。在编写、审查或重构 React/Next.js 代码时使用，以确保采用高性能模式。适用于 React 组件、Next.js 页面、数据获取、打包优化和性能改进等任务。

skills.sh 历史总榜第 8 名快照。原始来源：vercel-labs/agent-skills；原始 slug：vercel-react-best-practices。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh8_description$
),
(
    $zh9_slug$setup-matt-pocock-skills$zh9_slug$,
    'zh-CN',
    $zh9_name$Matt Pocock 工程技能初始化$zh9_name$,
    $zh9_summary$为仓库配置工程技能所需的问题跟踪、分诊标签词汇和领域文档结构。$zh9_summary$,
    $zh9_description$为当前仓库配置整套工程技能：设置问题跟踪器、分诊标签词汇和领域文档布局。应在首次使用其他工程技能之前运行一次。

skills.sh 历史总榜第 9 名快照。原始来源：mattpocock/skills；原始 slug：setup-matt-pocock-skills。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh9_description$
),
(
    $zh10_slug$handoff$zh10_slug$,
    'zh-CN',
    $zh10_name$对话交接文档$zh10_name$,
    $zh10_summary$把当前对话压缩成可供另一位 Agent 继续工作的交接文档。$zh10_summary$,
    $zh10_description$将当前对话浓缩成一份交接文档，使另一位 Agent 能够接续处理。

skills.sh 历史总榜第 10 名快照。原始来源：mattpocock/skills；原始 slug：handoff。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh10_description$
),
(
    $zh11_slug$triage$zh11_slug$,
    'zh-CN',
    $zh11_name$问题分诊与任务简报$zh11_name$,
    $zh11_summary$按分诊状态机处理问题和外部 PR，完成分类、核验、追问及 Agent 就绪简报。$zh11_summary$,
    $zh11_description$通过一套分诊角色状态机推进问题和外部 PR：进行分类、核验，必要时开展严格追问，并编写可由 Agent 直接执行的任务简报。

skills.sh 历史总榜第 11 名快照。原始来源：mattpocock/skills；原始 slug：triage。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh11_description$
),
(
    $zh12_slug$lark-doc$zh12_slug$,
    'zh-CN',
    $zh12_name$飞书云文档操作$zh12_name$,
    $zh12_summary$读取、创建和编辑飞书云文档，并处理图片、附件与思维笔记。$zh12_summary$,
    $zh12_description$飞书云文档（Docx / Wiki）内容操作：读取、创建、编辑文档，插入或下载图片附件，以及操作思维笔记。用户提供文档 URL/token（包括 doubao.com 的 /docx/、/wiki/）时使用；按 URL 路径/token 而非域名路由。文档内嵌资源按读取参考中的统一规则分流。文档评论走 lark-drive；表格或 Base 内部数据操作不在本 Skill。

skills.sh 历史总榜第 12 名快照。原始来源：open.feishu.cn；原始 slug：lark-doc。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh12_description$
),
(
    $zh13_slug$lark-approval$zh13_slug$,
    'zh-CN',
    $zh13_name$飞书审批$zh13_name$,
    $zh13_summary$查询和处理飞书审批待办、已办与实例，并搜索定义和发起原生审批。$zh13_summary$,
    $zh13_description$飞书审批：查询和处理审批待办、已办和实例，搜索可发起的审批定义、查看定义详情并发起原生审批实例。用户要处理审批任务、查看审批实例、搜索或发起审批时使用。审批待办不是飞书任务；非审批类待办走 lark-task。不负责创建审批定义；第三方审批定义不走原生提单。

skills.sh 历史总榜第 13 名快照。原始来源：open.feishu.cn；原始 slug：lark-approval。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh13_description$
),
(
    $zh14_slug$lark-base$zh14_slug$,
    'zh-CN',
    $zh14_name$飞书多维表格$zh14_name$,
    $zh14_summary$管理飞书多维表格的表、字段、记录、视图、公式、表单、仪表盘和角色权限。$zh14_summary$,
    $zh14_description$飞书多维表格（Base）操作：建表、字段、记录、视图、统计、公式与 lookup、表单、仪表盘、workflow 和角色权限；遇到 Base、多维表格、bitable 或 /base/ 链接时使用。文件导入或导出转 lark-drive，认证与授权转 lark-shared。

skills.sh 历史总榜第 14 名快照。原始来源：open.feishu.cn；原始 slug：lark-base。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh14_description$
),
(
    $zh15_slug$lark-drive$zh15_slug$,
    'zh-CN',
    $zh15_name$飞书云空间$zh15_name$,
    $zh15_summary$管理飞书云空间文件、目录、权限、评论、版本、密级标签及本地文件导入。$zh15_summary$,
    $zh15_description$飞书云空间（云盘/云存储）：管理 Drive 文件和文件夹，包含上传与下载、创建文件夹、复制、移动、删除、查看元数据、查询权限设置、评论、权限、订阅、标题、版本、飞书文档密级标签（secure labels）和本地文件导入。用户需要整理云盘目录、处理云空间资源 URL/token、判断链接类型、真实 token 或标题，或导入 Word、Markdown、Excel、CSV、PPTX、.base 为 docx、sheet、bitable、slides 时使用；doubao.com 云空间 URL/token 也按资源路径和 token 路由，不回退 WebFetch。不负责文档内容编辑（走 lark-doc）、表格或 Base 表内数据操作（走 lark-sheets/lark-base）、知识空间节点或成员管理（走 lark-wiki），以及原生 Markdown 文件读写、patch 或 diff（走 lark-markdown）。

skills.sh 历史总榜第 15 名快照。原始来源：open.feishu.cn；原始 slug：lark-drive。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh15_description$
),
(
    $zh16_slug$lark-shared$zh16_slug$,
    'zh-CN',
    $zh16_name$Lark CLI 认证与权限$zh16_name$,
    $zh16_summary$处理 lark-cli 的安装、登录状态、身份、业务域权限、授权范围和撤销授权。$zh16_summary$,
    $zh16_description$用于 lark-cli 的安装与认证任务：登录、状态查询和注销，用户与机器人身份，业务域权限（--domain，包括 all、docs、drive），缺失的授权范围、撤销授权，以及处理 _notice JSON。

skills.sh 历史总榜第 16 名快照。原始来源：open.feishu.cn；原始 slug：lark-shared。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh16_description$
),
(
    $zh17_slug$lark-calendar$zh17_slug$,
    'zh-CN',
    $zh17_name$飞书日历$zh17_name$,
    $zh17_summary$查看和管理飞书日程、参会人、忙闲、推荐时段及会议室。$zh17_summary$,
    $zh17_description$飞书日历：管理日历日程和会议室。查看或搜索日程、创建或更新日程、管理参会人、查询忙闲和推荐时段、预定会议室。用户需要查看日程安排、创建或修改会议、查询或预定会议室时使用。不负责查询过去的视频会议记录（走 lark-vc）或待办任务（走 lark-task）。

skills.sh 历史总榜第 17 名快照。原始来源：open.feishu.cn；原始 slug：lark-calendar。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh17_description$
),
(
    $zh18_slug$lark-im$zh18_slug$,
    'zh-CN',
    $zh18_name$飞书即时通讯$zh18_name$,
    $zh18_summary$收发和搜索飞书消息，管理群聊、文件、表情、加急、交互卡片及会话数据。$zh18_summary$,
    $zh18_description$飞书即时通讯：收发消息和管理群聊。发送和回复消息、搜索聊天记录、管理群聊成员、上传下载图片和文件、管理表情回复、发送应用内、短信或电话加急、发送和处理交互卡片（Interactive Card）、监听卡片按钮回调（card.action.trigger）。当用户需要发消息、查看或搜索聊天记录、下载聊天中的文件、查看群成员、搜索群、创建群聊或话题群、管理标记数据、管理 Feed 置顶（添加、移除或查询置顶会话）、管理标签数据或处理卡片回调时使用。

skills.sh 历史总榜第 18 名快照。原始来源：open.feishu.cn；原始 slug：lark-im。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh18_description$
),
(
    $zh19_slug$lark-contact$zh19_slug$,
    'zh-CN',
    $zh19_name$飞书通讯录$zh19_name$,
    $zh19_summary$在姓名、邮箱与 open_id 之间解析飞书联系人，并查询部门和个人信息。$zh19_summary$,
    $zh19_description$飞书/Lark 通讯录：按姓名或邮箱解析 open_id，按 open_id 反查姓名、部门、邮箱、联系方式、个人状态和签名，并按关键词搜索当前用户可见的机器人或智能体。用户提到一个名字并准备继续发消息或安排日程，或拿到 open_id 后要查询具体信息时使用。不负责遍历部门树、按部门列出员工或生成组织架构图；此类需求使用原生 OpenAPI。

skills.sh 历史总榜第 19 名快照。原始来源：open.feishu.cn；原始 slug：lark-contact。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh19_description$
),
(
    $zh20_slug$lark-sheets$zh20_slug$,
    'zh-CN',
    $zh20_name$飞书电子表格$zh20_name$,
    $zh20_summary$创建和编辑飞书电子表格，管理结构、单元格、公式、图表、透视表与财务模型。$zh20_summary$,
    $zh20_description$飞书电子表格：创建和操作电子表格。支持创建表格，管理工作表与行列结构（增删、合并、调整尺寸、隐藏、冻结），读写单元格（值、公式、样式、批注、单元格图片），查找替换和多操作批量更新，以及创建和维护图表、透视表、条件格式、筛选器、迷你图、浮动图片等对象。当用户需要创建电子表格、管理工作表、批量读写或编辑数据、统计汇总与可视化、表格美化、公式计算（含 Excel 公式迁移）、金融或财务建模（DCF、三张表、预算、敏感性分析等）时使用。若用户想按名称或关键词搜索云空间中的表格文件，请改用 lark-drive 的 drive +search 定位资源。用户给出 doubao.com 的 /sheets/ URL/token 时，也应直接使用本 Skill；按 URL 路径模式和 token 路由，不因域名不同而回退 WebFetch。

skills.sh 历史总榜第 20 名快照。原始来源：open.feishu.cn；原始 slug：lark-sheets。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh20_description$
),
(
    $zh21_slug$lark-wiki$zh21_slug$,
    'zh-CN',
    $zh21_name$飞书知识库$zh21_name$,
    $zh21_summary$管理飞书知识空间、成员、文档节点、快捷方式及层级结构。$zh21_summary$,
    $zh21_description$飞书知识库：管理知识空间、空间成员和文档节点。创建和查询知识空间、查看和管理空间成员、管理节点层级结构、在知识库中组织文档和快捷方式。用户需要在知识库中查找或创建文档、浏览知识空间结构、查看或管理空间成员、移动或复制节点时使用。用户给出 doubao.com 的 /wiki/ URL/token 时，也应直接使用本 Skill，不因域名不同而回退 WebFetch；路由依据是 URL 路径模式和 token。不负责上传文件到知识库节点下（走 lark-drive），也不负责编辑文档、表格或 Base 内容（走 lark-doc、lark-sheets、lark-base）。

skills.sh 历史总榜第 21 名快照。原始来源：open.feishu.cn；原始 slug：lark-wiki。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh21_description$
),
(
    $zh22_slug$lark-attendance$zh22_slug$,
    'zh-CN',
    $zh22_name$飞书考勤记录$zh22_name$,
    $zh22_summary$查询当前用户自己的飞书考勤打卡记录。$zh22_summary$,
    $zh22_description$飞书考勤打卡：查询当前用户自己的考勤打卡记录。

skills.sh 历史总榜第 22 名快照。原始来源：open.feishu.cn；原始 slug：lark-attendance。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh22_description$
),
(
    $zh23_slug$lark-event$zh23_slug$,
    'zh-CN',
    $zh23_name$飞书实时事件$zh23_name$,
    $zh23_summary$订阅、消费并流式处理飞书消息、审批、任务、会议、妙记和画板等实时事件。$zh23_summary$,
    $zh23_description$用于监听、订阅和消费 Lark/飞书实时事件：通过 lark-cli event consume <EventKey> 以 NDJSON 流式输出事件，覆盖即时消息、表情回复、群聊变更、审批状态、任务更新、视频会议开始、加入与结束、妙记生成、画板更新等。适用于 Lark 机器人、实时消息处理、长期运行的订阅器，以及流式 webhook 或推送处理器。支持通过 --max-events 和 --timeout 限定运行范围，并提供写入 stderr 的就绪标记契约，便于 AI Agent 作为子进程调用。

skills.sh 历史总榜第 23 名快照。原始来源：open.feishu.cn；原始 slug：lark-event。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh23_description$
),
(
    $zh24_slug$lark-minutes$zh24_slug$,
    'zh-CN',
    $zh24_name$飞书妙记$zh24_name$,
    $zh24_summary$搜索和编辑飞书妙记，处理音视频、产物内容、说话人、关键词与访问权限。$zh24_summary$,
    $zh24_description$飞书妙记：搜索妙记、查看基础信息、下载或上传音视频、读取或编辑妙记产物内容、修改标题、替换说话人或关键词，并申请妙记查看或编辑权限。给出 minute_token、本地音视频文件，需要查询、修改或转换妙记产物，或用户明确要主动申请妙记权限时使用。本地音视频转纪要或逐字稿优先使用本 Skill，不使用 ffmpeg 或 whisper 本地转写。不负责获取会议关联妙记，也不负责仅按自然语言标题定位纪要。

skills.sh 历史总榜第 24 名快照。原始来源：open.feishu.cn；原始 slug：lark-minutes。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh24_description$
),
(
    $zh25_slug$lark-openapi-explorer$zh25_slug$,
    'zh-CN',
    $zh25_name$飞书 OpenAPI 探索$zh25_name$,
    $zh25_summary$从飞书官方文档中查找并调用尚未由 lark-cli 封装的原生 OpenAPI。$zh25_summary$,
    $zh25_description$飞书/Lark 原生 OpenAPI 探索：从官方文档库中查找未经 CLI 封装的原生 OpenAPI 接口。用户需求无法由现有 lark-* Skill 或 lark-cli 已注册命令满足，需要查找并调用原生飞书 OpenAPI 时使用。

skills.sh 历史总榜第 25 名快照。原始来源：open.feishu.cn；原始 slug：lark-openapi-explorer。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh25_description$
),
(
    $zh26_slug$lark-whiteboard$zh26_slug$,
    'zh-CN',
    $zh26_name$飞书画板$zh26_name$,
    $zh26_summary$查询、导出和编辑飞书云文档中的画板及原始节点结构。$zh26_summary$,
    $zh26_description$飞书画板：查询和编辑飞书云文档中的画板。支持将画板导出为预览图片、导出原始节点结构，并用多种格式更新画板内容。用户需要查看画板内容、导出画板图片或编辑画板时使用。不负责飞书云文档内容编辑（lark-doc），也不负责文档内嵌电子表格或 Base（lark-sheets、lark-base）。

skills.sh 历史总榜第 26 名快照。原始来源：open.feishu.cn；原始 slug：lark-whiteboard。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh26_description$
),
(
    $zh27_slug$lark-mail$zh27_slug$,
    'zh-CN',
    $zh27_name$飞书邮箱$zh27_name$,
    $zh27_summary$起草、收发、搜索和管理飞书邮件、文件夹、标签、联系人与收信规则。$zh27_summary$,
    $zh27_description$飞书邮箱：当用户需要起草、发送、回复或转发邮件，查阅或搜索邮件，管理邮件文件夹、邮件标签和邮件联系人，监听新邮件或设置收信规则时使用。仅处理邮件意图；不用于文档、表格、日历、认证设置、纯通讯录查询或即时通讯任务。

skills.sh 历史总榜第 27 名快照。原始来源：open.feishu.cn；原始 slug：lark-mail。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh27_description$
),
(
    $zh28_slug$lark-task$zh28_slug$,
    'zh-CN',
    $zh28_name$飞书任务$zh28_name$,
    $zh28_summary$管理飞书任务、清单、子任务、协作者、附件和任务智能体。$zh28_summary$,
    $zh28_description$飞书任务：管理任务、清单和任务智能体。创建待办任务、查看和更新任务状态、拆分子任务、组织任务清单、分配协作成员、上传任务附件、注册或注销任务智能体、更新任务智能体主页数据，并写入智能体任务记录。用户需要创建待办事项、查看任务列表、跟踪任务进度、管理项目清单、给他人分配任务、为任务上传附件、注册或注销任务智能体、更新智能体主页数据或写入任务记录时使用。

skills.sh 历史总榜第 28 名快照。原始来源：open.feishu.cn；原始 slug：lark-task。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh28_description$
),
(
    $zh29_slug$lark-slides$zh29_slug$,
    'zh-CN',
    $zh29_name$飞书幻灯片$zh29_name$,
    $zh29_summary$创建和编辑飞书幻灯片，读取内容并管理、删除或局部替换页面。$zh29_summary$,
    $zh29_description$飞书幻灯片：创建和编辑幻灯片。创建演示文稿、读取幻灯片内容，并管理页面的创建、删除、读取和局部替换。用户需要创建或编辑幻灯片、读取或修改单个页面时使用。用户给出 doubao.com 的 /slides/ URL/token 时，也应直接使用本 Skill；按 URL 路径模式和 token 路由，不因域名不同而回退 WebFetch。不负责云文档内容编辑（走 lark-doc）、云文档中的独立画板对象（走 lark-whiteboard），以及上传或下载普通文件（走 lark-drive）。

skills.sh 历史总榜第 29 名快照。原始来源：open.feishu.cn；原始 slug：lark-slides。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh29_description$
),
(
    $zh30_slug$lark-skill-maker$zh30_slug$,
    'zh-CN',
    $zh30_name$飞书 Skill 制作器$zh30_name$,
    $zh30_summary$把飞书 API 原子操作或多步流程封装成可复用的 lark-cli 自定义 Skill。$zh30_summary$,
    $zh30_description$创建 lark-cli 的自定义 Skill。用户需要把飞书 API 操作封装成可复用 Skill，包括包装原子 API 或编排多步流程时使用。

skills.sh 历史总榜第 30 名快照。原始来源：open.feishu.cn；原始 slug：lark-skill-maker。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh30_description$
),
(
    $zh31_slug$lark-vc$zh31_slug$,
    'zh-CN',
    $zh31_name$飞书视频会议$zh31_name$,
    $zh31_summary$查询进行中与历史飞书会议、会中实时内容、会议纪要和参会人快照。$zh31_summary$,
    $zh31_description$飞书视频会议：查询进行中的会议列表（含会议 ID），读取会中实时内容（发言、聊天、共享等），发送会中消息，以及搜索历史会议、查询会议纪要（总结、待办、章节、逐字稿）和参会人快照。Agent 真实入会或离会走 lark-vc-agent；查询未来日程走 lark-calendar。

skills.sh 历史总榜第 31 名快照。原始来源：open.feishu.cn；原始 slug：lark-vc。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh31_description$
),
(
    $zh32_slug$lark-workflow-meeting-summary$zh32_slug$,
    'zh-CN',
    $zh32_name$飞书会议纪要工作流$zh32_name$,
    $zh32_summary$汇总指定时间范围内的飞书会议纪要，并生成结构化报告或会议周报。$zh32_summary$,
    $zh32_description$会议纪要整理工作流：汇总指定时间范围内的会议纪要并生成结构化报告。用户需要整理会议纪要、生成会议周报或回顾一段时间内的会议内容时使用。

skills.sh 历史总榜第 32 名快照。原始来源：open.feishu.cn；原始 slug：lark-workflow-meeting-summary。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh32_description$
),
(
    $zh33_slug$lark-workflow-standup-report$zh33_slug$,
    'zh-CN',
    $zh33_name$飞书日程待办摘要$zh33_name$,
    $zh33_summary$组合飞书日程与任务命令，生成今天、明天或本周的日程及未完成任务摘要。$zh33_summary$,
    $zh33_description$日程待办摘要：编排 calendar +agenda 和 task +get-my-tasks，生成指定日期的日程与未完成任务摘要。适用于了解今天、明天或本周的安排。

skills.sh 历史总榜第 33 名快照。原始来源：open.feishu.cn；原始 slug：lark-workflow-standup-report。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh33_description$
),
(
    $zh34_slug$prototype$zh34_slug$,
    'zh-CN',
    $zh34_name$设计原型验证$zh34_name$,
    $zh34_summary$快速构建一次性原型，用于验证状态模型、交互逻辑或界面方向。$zh34_summary$,
    $zh34_description$构建一次性原型来回答设计问题。适用于用户希望快速验证状态模型或逻辑是否合理，或探索界面应呈现何种形态时。

skills.sh 历史总榜第 34 名快照。原始来源：mattpocock/skills；原始 slug：prototype。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh34_description$
),
(
    $zh35_slug$lark-okr$zh35_slug$,
    'zh-CN',
    $zh35_name$飞书 OKR 管理$zh35_name$,
    $zh35_summary$查看和编辑飞书 OKR 周期、目标、关键结果、对齐关系、量化指标及进展记录。$zh35_summary$,
    $zh35_description$飞书 OKR：管理目标与关键结果。查看和编辑 OKR 周期、目标、关键结果、对齐关系、量化指标和进展记录。用户需要查看或创建 OKR、管理目标和关键结果、查看对齐关系时使用。不负责待办任务管理（lark-task）、日程或会议安排（lark-calendar）以及绩效评估。

skills.sh 历史总榜第 35 名快照。原始来源：open.feishu.cn；原始 slug：lark-okr。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh35_description$
),
(
    $zh36_slug$web-design-guidelines$zh36_slug$,
    'zh-CN',
    $zh36_name$Web 界面规范审查$zh36_name$,
    $zh36_summary$依据 Web 界面指南审查 UI 代码的可访问性、设计质量、用户体验和最佳实践。$zh36_summary$,
    $zh36_description$审查 UI 代码是否符合 Web 界面指南。用户要求审查 UI、检查可访问性、进行设计审计、评审用户体验，或核对网站是否遵循最佳实践时使用。

skills.sh 历史总榜第 36 名快照。原始来源：vercel-labs/agent-skills；原始 slug：web-design-guidelines。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh36_description$
),
(
    $zh37_slug$lark-markdown$zh37_slug$,
    'zh-CN',
    $zh37_name$飞书 Markdown 文件$zh37_name$,
    $zh37_summary$查看、创建、上传、编辑和比较飞书云空间中的原生 Markdown 文件。$zh37_summary$,
    $zh37_description$飞书 Markdown：查看、创建、上传、编辑和比较 Markdown 文件。用户需要创建或编辑 Markdown 文件，读取、修改、局部 patch 或比较差异时使用。不负责把 Markdown 导入为飞书在线文档，也不负责文件搜索、权限、评论、移动、删除等云空间管理操作。

skills.sh 历史总榜第 37 名快照。原始来源：open.feishu.cn；原始 slug：lark-markdown。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh37_description$
),
(
    $zh38_slug$microsoft-foundry$zh38_slug$,
    'zh-CN',
    $zh38_name$Microsoft Foundry Agent 全流程$zh38_name$,
    $zh38_summary$使用 azd 端到端部署、评估、优化、微调和管理 Microsoft Foundry Agent。$zh38_summary$,
    $zh38_description$使用 azd 对 Microsoft Foundry Agent 进行端到端部署、评估、微调和管理：包括托管 Agent 的脚手架、运行与部署，提示词 Agent 创建，批量与持续评估，提示词优化器，Agent Optimizer 脚手架，agent.yaml，基于追踪的数据集整理，以及 SFT、DPO、RFT 模型微调。适用于 azd ai agent、azd provision 或 deploy、部署与调用 Agent、添加工具、持续评估和监控、Agent CI/CD、优化提示词或 Agent 指令、模型部署、Foundry 项目、RBAC 与角色分配、权限、配额、容量、区域、Agent 和部署故障排查、AI Services、创建 Foundry 资源、知识索引、定制部署、接入、可用性、训练数据、评分器、蒸馏、微调模型和大文件上传。不用于 Azure Functions、App Service 或通用 Azure 部署；这些任务应使用对应的专用 Skill。

skills.sh 历史总榜第 38 名快照。原始来源：microsoft/azure-skills；原始 slug：microsoft-foundry。打包时排除了 86 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh38_description$
),
(
    $zh39_slug$azure-diagnostics$zh39_slug$,
    'zh-CN',
    $zh39_name$Azure 生产故障诊断$zh39_name$,
    $zh39_summary$利用 AppLens、Azure Monitor 和资源运行状况，对 Azure 生产故障进行安全分诊与根因排查。$zh39_summary$,
    $zh39_description$使用 AppLens、Azure Monitor、资源运行状况和安全分诊流程调试 Azure 生产问题。适用于 App Service 高 CPU 或部署失败、Container Apps 和 Functions 故障、AKS 故障、虚拟机 RDP 或 Linux SSH 连接失败、黑屏、密码重置、NSG 或防火墙阻断、kubectl 无法连接、kube-system 或 CoreDNS 失败、Pod Pending 或 CrashLoop、节点未就绪、升级失败、日志与 KQL 分析、镜像拉取失败、冷启动、健康探针失败、资源错误根因，以及 Event Hubs、Service Bus、消息 SDK、AMQP 连接、消息锁和死信等问题。

skills.sh 历史总榜第 39 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-diagnostics。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh39_description$
),
(
    $zh40_slug$azure-ai$zh40_slug$,
    'zh-CN',
    $zh40_name$Azure AI 服务$zh40_name$,
    $zh40_summary$处理 Azure AI Search、Speech、OpenAI 和 Document Intelligence 的搜索、语音、转写与 OCR 任务。$zh40_summary$,
    $zh40_description$用于 Azure AI 服务，包括 Search、Speech、OpenAI 和 Document Intelligence。支持普通搜索、向量与混合搜索、语义搜索、语音转文字、文字转语音、转写和 OCR。用户提到 AI Search、查询搜索、向量搜索、混合或语义搜索、语音识别、语音合成、转写、OCR 或文字转语音时使用。

skills.sh 历史总榜第 40 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-ai。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh40_description$
),
(
    $zh41_slug$azure-deploy$zh41_slug$,
    'zh-CN',
    $zh41_name$Azure 已就绪应用部署$zh41_name$,
    $zh41_summary$执行已完成准备和验证的 Azure 应用部署，并为 azd、Terraform、Bicep 等命令提供错误恢复。$zh41_summary$,
    $zh41_description$为已经准备完成、具备 .azure/deployment-plan.md 和基础设施文件的应用执行 Azure 部署。不要在用户要求创建新应用时使用；此类任务应先使用 azure-prepare。本 Skill 运行 azd up、azd deploy、terraform apply 和 az deployment 等命令，并内置错误恢复；要求已有 azure-prepare 生成的部署计划和 azure-validate 验证通过状态。适用于执行部署、推送到生产或云端、正式上线、Bicep 部署、Terraform 应用和发布到 Azure。不适用于“创建并部署”“构建并部署”“新建应用”“搭建基础设施”或“使用 Terraform 创建并部署到 Azure”；这些请求应使用 azure-prepare。

skills.sh 历史总榜第 41 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-deploy。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh41_description$
),
(
    $zh42_slug$azure-prepare$zh42_slug$,
    'zh-CN',
    $zh42_name$Azure azd 部署准备$zh42_name$,
    $zh42_summary$为 azd 部署流程生成 azure.yaml、Bicep 或 Terraform 基础设施及 Dockerfile。$zh42_summary$,
    $zh42_description$为基于 azd 的 Azure 项目准备部署：生成 Azure Developer CLI（azd）工作流所需的 azure.yaml、Bicep 或 Terraform 基础设施，以及 Dockerfile。仅当用户明确要用 azd 部署，或项目已有 azure.yaml 时使用。不用于非 azd 部署、Python App Service 纯代码部署（使用 python-appservice-deploy）或跨云迁移（使用 azure-cloud-migrate）。适用于准备 azd 应用、创建 azure.yaml、搭建 azd 基础设施、以 azd 现代化 Azure 应用、Functions、定时与 Service Bus 触发器、事件驱动函数、托管身份、生成 Bicep 或 Terraform，以及创建并部署到 Azure。

skills.sh 历史总榜第 42 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-prepare。打包时排除了 64 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh42_description$
),
(
    $zh43_slug$azure-validate$zh43_slug$,
    'zh-CN',
    $zh43_name$Azure 部署前校验$zh43_name$,
    $zh43_summary$在部署前深度检查 Azure 配置、基础设施、RBAC、托管身份权限和各项前置条件。$zh43_summary$,
    $zh43_description$执行 Azure 部署就绪性预检。部署前深入检查配置、Bicep 或 Terraform 基础设施、RBAC 角色分配、托管身份权限和前置条件。适用于验证应用、检查部署准备、运行预检、核对配置、判断是否可部署、验证 azure.yaml、Bicep、Azure Functions 或无服务器部署、部署前测试、排查部署错误、核对 RBAC 与角色分配、审查托管身份权限、what-if 分析，以及验证 Container Apps 部署。

skills.sh 历史总榜第 43 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-validate。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh43_description$
),
(
    $zh44_slug$azure-storage$zh44_slug$,
    'zh-CN',
    $zh44_name$Azure 存储服务$zh44_name$,
    $zh44_summary$讲解和操作 Azure Blob、文件共享、队列、表存储和 Data Lake，并比较存储访问层级。$zh44_summary$,
    $zh44_description$覆盖 Azure 存储服务，包括 Blob Storage、File Shares、Queue Storage、Table Storage 和 Data Lake。解答热、凉、冷、归档等访问层级的适用场景与比较，提供对象存储、SMB 文件共享、异步消息、NoSQL 键值存储和大数据分析能力，并包含生命周期管理。适用于 Blob、文件共享、队列、表、数据湖、文件上传、Blob 下载、存储账户、访问层级及其比较和生命周期管理。不用于 SQL 数据库、Cosmos DB（使用 azure-prepare），也不用于 Event Hubs 或 Service Bus 消息（使用 azure-messaging）。

skills.sh 历史总榜第 44 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-storage。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh44_description$
),
(
    $zh45_slug$entra-app-registration$zh45_slug$,
    'zh-CN',
    $zh45_name$Microsoft Entra 应用注册$zh45_name$,
    $zh45_summary$指导 Microsoft Entra ID 应用注册、OAuth 2.0 认证、API 权限、服务主体和 MSAL 集成。$zh45_summary$,
    $zh45_description$指导 Microsoft Entra ID 应用注册、OAuth 2.0 身份验证和 MSAL 集成。适用于创建应用注册、注册 Azure AD 应用、配置 OAuth、搭建认证、添加 API 权限、生成服务主体、编写 MSAL 示例、控制台应用认证、Entra ID 设置和 Azure AD 身份验证。不用于 Key Vault 密钥审计（使用 azure-keyvault-expiration-audit）或通用 Azure 资源安全指导。

skills.sh 历史总榜第 45 名快照。原始来源：microsoft/azure-skills；原始 slug：entra-app-registration。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh45_description$
),
(
    $zh46_slug$appinsights-instrumentation$zh46_slug$,
    'zh-CN',
    $zh46_name$Azure Application Insights 埋点$zh46_name$,
    $zh46_summary$为 Web 应用接入 Azure Application Insights，提供遥测模式、SDK 配置和 APM 最佳实践。$zh46_summary$,
    $zh46_description$指导 Web 应用接入 Azure Application Insights，提供遥测模式、SDK 安装配置和参考资料。适用于应用埋点、App Insights SDK、遥测模式、Application Insights 概念与示例，以及 APM 最佳实践。

skills.sh 历史总榜第 46 名快照。原始来源：microsoft/azure-skills；原始 slug：appinsights-instrumentation。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh46_description$
),
(
    $zh47_slug$azure-resource-lookup$zh47_slug$,
    'zh-CN',
    $zh47_name$Azure 资源查询$zh47_name$,
    $zh47_summary$跨订阅或资源组列出、查找和查看 Azure 资源，并支持标签、孤立资源与 Resource Graph 查询。$zh47_summary$,
    $zh47_description$跨订阅或资源组列出、查找和展示 Azure 资源，可处理列出网站、Web Apps、App Services、虚拟机、存储账户、Container Apps 和查看现有资源等请求。适用于资源清单、按标签查找、标签分析、发现孤立资源（不含成本分析）、未挂载磁盘、按类型计数、跨订阅查询和 Azure Resource Graph。不用于部署或修改资源（使用 azure-deploy）、成本优化（使用 azure-cost）或非 Azure 云。

skills.sh 历史总榜第 47 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-resource-lookup。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh47_description$
),
(
    $zh48_slug$azure-compliance$zh48_slug$,
    'zh-CN',
    $zh48_name$Azure 合规与安全审计$zh48_name$,
    $zh48_summary$使用 azqr 和 Key Vault 到期检查执行 Azure 最佳实践、合规与安全态势审计。$zh48_summary$,
    $zh48_description$使用 azqr 和 Key Vault 到期检查执行 Azure 合规与安全审计，覆盖最佳实践评估、资源审查、策略与合规验证以及安全态势检查。适用于合规扫描、安全审计、运行 azqr 前的准备、Azure 最佳实践、Key Vault 到期检查、过期证书、即将到期的密钥、孤立资源和合规评估。

skills.sh 历史总榜第 48 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-compliance。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh48_description$
),
(
    $zh49_slug$azure-aigateway$zh49_slug$,
    'zh-CN',
    $zh49_name$Azure AI Gateway 配置$zh49_name$,
    $zh49_summary$将 Azure API Management 配置为 AI 模型、MCP 工具和 Agent 的治理网关。$zh49_summary$,
    $zh49_description$把 Azure API Management 配置为 AI 模型、MCP 工具和 Agent 的 AI Gateway。适用于语义缓存、token 限制、内容安全、负载均衡、AI 模型治理、MCP 限流、越狱检测、接入 Azure OpenAI 或 AI Foundry 模型、测试 AI Gateway、LLM 策略、AI 后端配置、token 指标、成本控制、把 API 转为 MCP，以及向网关导入 OpenAPI。

skills.sh 历史总榜第 49 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-aigateway。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh49_description$
),
(
    $zh50_slug$azure-kusto$zh50_slug$,
    'zh-CN',
    $zh50_name$Azure Kusto 数据分析$zh50_name$,
    $zh50_summary$使用 KQL 查询和分析 Azure Data Explorer 中的日志、遥测、时序与 IoT 数据。$zh50_summary$,
    $zh50_description$使用 KQL 查询和分析 Azure Data Explorer（Kusto/ADX）中的数据，适用于日志分析、遥测和时间序列。用户需要编写 KQL、查询 Kusto 数据库或 ADX 集群、分析日志、时序数据、IoT 遥测或进行异常检测时使用。

skills.sh 历史总榜第 50 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-kusto。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh50_description$
),
(
    $zh51_slug$azure-resource-visualizer$zh51_slug$,
    'zh-CN',
    $zh51_name$Azure 资源架构可视化$zh51_name$,
    $zh51_summary$分析 Azure 资源组，并生成展示资源关系和基础设施拓扑的 Mermaid 架构图。$zh51_summary$,
    $zh51_description$分析 Azure 资源组并生成详细的 Mermaid 架构图，展示各项资源之间的关系。适用于创建架构图、可视化 Azure 资源、展示资源关系、生成 Mermaid 图、分析资源组、绘制资源架构、查看资源拓扑或映射 Azure 基础设施。

skills.sh 历史总榜第 51 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-resource-visualizer。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh51_description$
),
(
    $zh52_slug$azure-messaging$zh52_slug$,
    'zh-CN',
    $zh52_name$Azure 消息服务排障$zh52_name$,
    $zh52_summary$排查 Azure Event Hubs 与 Service Bus SDK 的连接、认证、消息处理、锁和配置问题。$zh52_summary$,
    $zh52_description$排查并解决 Azure Event Hubs 和 Service Bus 消息 SDK 问题，包括连接失败、认证错误、消息处理异常和 SDK 配置。适用于 Event Hubs 或 Service Bus SDK 错误、AMQP 连接、Event Processor Host、消息锁丢失或过期、锁续期与批处理、发送超时、接收端断开、SDK 日志、消费者和队列问题、主题订阅、checkpoint、收不到消息、死信、会话锁、空闲超时、连接失活、link detach、重连缓慢、会话错误、重复事件、offset 重置和批量接收。

skills.sh 历史总榜第 52 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-messaging。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh52_description$
),
(
    $zh53_slug$lark-vc-agent$zh53_slug$,
    'zh-CN',
    $zh53_name$飞书视频会议 Agent$zh53_name$,
    $zh53_summary$让应用机器人加入或离开进行中的飞书会议，并读取会中事件、发送消息和表情。$zh53_summary$,
    $zh53_description$飞书视频会议会中能力：让应用机器人真实加入或离开正在进行的会议，读取当前身份可见的会中事件，并发送会中文本消息或表情。适用于用户询问正在进行的会议发生了什么、谁在发言、是否共享内容，或需要发现当前可读的进行中会议 ID。不负责已结束会议搜索、参会人快照、纪要、逐字稿或录制查询；这些任务使用 lark-vc。

skills.sh 历史总榜第 53 名快照。原始来源：open.feishu.cn；原始 slug：lark-vc-agent。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh53_description$
),
(
    $zh54_slug$azure-rbac$zh54_slug$,
    'zh-CN',
    $zh54_name$Azure RBAC 最小权限$zh54_name$,
    $zh54_summary$为身份选择最小权限 Azure RBAC 角色，并生成 CLI 命令和 Bicep 角色分配代码。$zh54_summary$,
    $zh54_description$帮助用户为身份选择具备最小权限的 Azure RBAC 角色，并生成对应的 CLI 命令和 Bicep 代码，同时说明授予角色所需的权限。适用于 Bicep 角色分配、角色选择、最小权限、Blob 读取角色、托管身份角色、自定义角色定义、给身份分配角色、授予访问权限所需角色，以及执行角色分配所需权限。

skills.sh 历史总榜第 54 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-rbac。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh54_description$
),
(
    $zh55_slug$remotion-best-practices$zh55_slug$,
    'zh-CN',
    $zh55_name$Remotion 技能路由$zh55_name$,
    $zh55_summary$为所有 Remotion 相关技能提供统一路由入口。$zh55_summary$,
    $zh55_description$所有 Remotion 技能的统一路由器，用于把请求分派给合适的 Remotion 能力。

skills.sh 历史总榜第 55 名快照。原始来源：remotion-dev/skills；原始 slug：remotion-best-practices。打包时排除了 24 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh55_description$
),
(
    $zh56_slug$lark-apps$zh56_slug$,
    'zh-CN',
    $zh56_name$妙搭应用开发与托管$zh56_name$,
    $zh56_summary$使用妙搭进行应用创建、全栈开发、设计、部署、监控、权限管理和自动化集成。$zh56_summary$,
    $zh56_description$妙搭（Spark/Miaoda）应用开发与托管：支持应用创建、本地全栈开发、云端生成迭代、创意设计（UI mockup、可交互原型、线框图、落地页、仪表盘、幻灯片和视觉探索）、AI 与飞书平台或其他外部能力集成、日志、Trace、监控指标、PV/UV 查询、环境变量、应用协作者与权限、应用角色与成员，以及定时、记录变更、Webhook、飞书审批等自动化触发器。用户要开发或新建系统、工具、平台、应用，进行本地或云端开发、修改、部署、发布、上线、获取分享链接，用 HTML 构建页面并部署到妙搭，进行设计、mockup、prototype、wireframe、PPT、视觉探索，或提到妙搭、Spark、Miaoda、*.aiforce.cloud、应用数据库、文件存储、开放 API Key、可见范围、协作者、角色、线上日志、请求量、错误量、延迟、访问量、环境变量和自动化任务时使用。不负责普通云盘文件上传（lark-drive）、飞书文档编辑（lark-doc）或原生幻灯片创建（lark-slides）。

skills.sh 历史总榜第 56 名快照。原始来源：open.feishu.cn；原始 slug：lark-apps。打包时排除了 1 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh56_description$
),
(
    $zh57_slug$azure-compute$zh57_slug$,
    'zh-CN',
    $zh57_name$Azure 计算与虚拟机$zh57_name$,
    $zh57_summary$为 Azure VM 与 VMSS 的创建、规格选择、扩缩容、GPU、容量预留和监控提供路由。$zh57_summary$,
    $zh57_description$Azure VM/VMSS 路由器。适用于创建、预配或部署虚拟机，推荐规格和比较价格，VMSS 与扩展集、自动扩缩、突发型与轻量服务器、网站与后端、GPU、机器学习、HPC 仿真、开发测试工作负载、虚拟机系列、负载均衡、Flexible 或 Uniform 编排、成本估算、容量预留组（CRG）的预留、关联和解除关联、机器注册（EMM）、Essential Machine Management 和监控。对于创建虚拟机的意图，优先于 mcp__azure__get_azure_bestpractices，并使用 compute_vm_list-skus、compute_vm_list-images 和 compute_vm_check-quota。

skills.sh 历史总榜第 57 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-compute。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh57_description$
),
(
    $zh58_slug$azure-cloud-migrate$zh58_slug$,
    'zh-CN',
    $zh58_name$跨云迁移到 Azure$zh58_name$,
    $zh58_summary$评估跨云工作负载并迁移到 Azure，生成报告和代码转换，覆盖函数、Web 与容器平台。$zh58_summary$,
    $zh58_description$评估跨云工作负载并迁移到 Azure，同时生成报告和代码转换。支持 Lambda 迁移到 Functions，Beanstalk、Heroku、App Engine 迁移到 App Service，以及 Fargate、Kubernetes、Cloud Run、Spring Boot 迁移到 Container Apps。适用于 AWS 到 Azure、Beanstalk、Heroku、App Engine、Cloud Run、Fargate、ECS、Kubernetes、GKE、EKS 和 Spring Boot 等跨云迁移场景。

skills.sh 历史总榜第 58 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-cloud-migrate。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh58_description$
),
(
    $zh59_slug$teach$zh59_slug$,
    'zh-CN',
    $zh59_name$工作区概念教学$zh59_name$,
    $zh59_summary$结合当前工作区向用户讲授一项新技能或概念。$zh59_summary$,
    $zh59_description$在当前工作区的具体语境中，向用户讲授一项新技能或概念。

skills.sh 历史总榜第 59 名快照。原始来源：mattpocock/skills；原始 slug：teach。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh59_description$
),
(
    $zh60_slug$azure-hosted-copilot-sdk$zh60_slug$,
    'zh-CN',
    $zh60_name$Azure GitHub Copilot SDK 应用$zh60_name$,
    $zh60_summary$在 Azure 上构建、部署和修改 GitHub Copilot SDK 应用，并识别专用工作流边界。$zh60_summary$,
    $zh60_description$在 Azure 上构建、部署和修改 GitHub Copilot SDK 应用。当代码库的 package.json 包含 @github/copilot-sdk 或 CopilotClient 时必须使用；检测到 Copilot SDK 标记时优先于 azure-prepare。适用于 Copilot SDK、@github/copilot-sdk、Copilot 驱动应用、构建或准备 Copilot 应用、添加或修改功能、BYOM、自带模型、CopilotClient、createSession、sendAndWait 和 azd init copilot。不用于部署已经准备完成的 Copilot SDK 应用（使用 azure-deploy）、不含 Copilot SDK 的普通 Web 应用（使用 azure-prepare）、Copilot Extensions 或 Foundry Agent（使用 microsoft-foundry）。

skills.sh 历史总榜第 60 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-hosted-copilot-sdk。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh60_description$
),
(
    $zh61_slug$caveman$zh61_slug$,
    'zh-CN',
    $zh61_name$Caveman 超精简沟通$zh61_name$,
    $zh61_summary$以保持技术准确性的超精简表达减少输出 token，并支持多档强度与文言模式。$zh61_summary$,
    $zh61_description$一种超精简沟通模式，在保持完整技术准确性的同时，以实测约 65% 的幅度减少输出 token。支持 lite、full（默认）、ultra、wenyan-lite、wenyan-full 和 wenyan-ultra 等强度。用户要求“caveman mode”“像穴居人一样说话”“少用 token”“简短表达”，调用 /caveman，或明确要求提高 token 效率时使用。

skills.sh 历史总榜第 61 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh61_description$
),
(
    $zh62_slug$grilling$zh62_slug$,
    'zh-CN',
    $zh62_name$方案压力访谈$zh62_name$,
    $zh62_summary$通过持续而严格的追问，对计划、决策或想法进行压力测试。$zh62_summary$,
    $zh62_description$围绕计划、决策或想法持续进行严格追问。适用于用户希望压力测试自己的思路，或使用任何与“grill”相关的触发说法时。

skills.sh 历史总榜第 62 名快照。原始来源：mattpocock/skills；原始 slug：grilling。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh62_description$
),
(
    $zh63_slug$lark-doc-larksuite-cli$zh63_slug$,
    'zh-CN',
    $zh63_name$飞书云文档操作（LarkSuite CLI 版）$zh63_name$,
    $zh63_summary$通过 LarkSuite CLI 读取、创建和编辑飞书云文档，并处理图片、附件与思维笔记。$zh63_summary$,
    $zh63_description$飞书云文档（Docx / Wiki）内容操作：读取、创建、编辑文档，插入或下载图片附件，以及操作思维笔记。用户提供文档 URL/token（包括 doubao.com 的 /docx/、/wiki/）时使用；按 URL 路径/token 而非域名路由。文档内嵌资源按读取参考中的统一规则分流。文档评论走 lark-drive；表格或 Base 内部数据操作不在本 Skill。

skills.sh 历史总榜第 63 名快照。原始来源：larksuite/cli；原始 slug：lark-doc。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh63_description$
),
(
    $zh64_slug$lark-base-larksuite-cli$zh64_slug$,
    'zh-CN',
    $zh64_name$飞书多维表格（LarkSuite CLI 版）$zh64_name$,
    $zh64_summary$通过 LarkSuite CLI 管理飞书多维表格的表、字段、记录、视图、公式、表单和权限。$zh64_summary$,
    $zh64_description$飞书多维表格（Base）操作：建表、字段、记录、视图、统计、公式与 lookup、表单、仪表盘、workflow 和角色权限；遇到 Base、多维表格、bitable 或 /base/ 链接时使用。文件导入或导出转 lark-drive，认证与授权转 lark-shared。

skills.sh 历史总榜第 64 名快照。原始来源：larksuite/cli；原始 slug：lark-base。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh64_description$
),
(
    $zh65_slug$lark-shared-larksuite-cli$zh65_slug$,
    'zh-CN',
    $zh65_name$Lark CLI 认证与权限（LarkSuite CLI 版）$zh65_name$,
    $zh65_summary$处理 lark-cli 的安装、登录状态、身份、业务域权限、授权范围和撤销授权。$zh65_summary$,
    $zh65_description$用于 lark-cli 的安装与认证任务：登录、状态查询和注销，用户与机器人身份，业务域权限（--domain，包括 all、docs、drive），缺失的授权范围、撤销授权，以及处理 _notice JSON。

skills.sh 历史总榜第 65 名快照。原始来源：larksuite/cli；原始 slug：lark-shared。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh65_description$
),
(
    $zh66_slug$lark-drive-larksuite-cli$zh66_slug$,
    'zh-CN',
    $zh66_name$飞书云空间（LarkSuite CLI 版）$zh66_name$,
    $zh66_summary$通过 LarkSuite CLI 管理飞书云空间文件、目录、权限、评论、版本及本地文件导入。$zh66_summary$,
    $zh66_description$飞书云空间（云盘/云存储）：管理 Drive 文件和文件夹，包含上传与下载、创建文件夹、复制、移动、删除、查看元数据、查询权限设置、评论、权限、订阅、标题、版本、飞书文档密级标签（secure labels）和本地文件导入。用户需要整理云盘目录、处理云空间资源 URL/token、判断链接类型、真实 token 或标题，或导入 Word、Markdown、Excel、CSV、PPTX、.base 为 docx、sheet、bitable、slides 时使用；doubao.com 云空间 URL/token 也按资源路径和 token 路由，不回退 WebFetch。不负责文档内容编辑（走 lark-doc）、表格或 Base 表内数据操作（走 lark-sheets/lark-base）、知识空间节点或成员管理（走 lark-wiki），以及原生 Markdown 文件读写、patch 或 diff（走 lark-markdown）。

skills.sh 历史总榜第 66 名快照。原始来源：larksuite/cli；原始 slug：lark-drive。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh66_description$
),
(
    $zh67_slug$lark-im-larksuite-cli$zh67_slug$,
    'zh-CN',
    $zh67_name$飞书即时通讯（LarkSuite CLI 版）$zh67_name$,
    $zh67_summary$通过 LarkSuite CLI 收发和搜索消息、管理群聊、文件、表情、加急、交互卡片及会话数据。$zh67_summary$,
    $zh67_description$飞书即时通讯：收发消息和管理群聊。发送和回复消息、搜索聊天记录、管理群聊成员、上传下载图片和文件、管理表情回复、发送应用内、短信或电话加急、发送和处理交互卡片（Interactive Card）、监听卡片按钮回调（card.action.trigger）。当用户需要发消息、查看或搜索聊天记录、下载聊天中的文件、查看群成员、搜索群、创建群聊或话题群、管理标记数据、管理 Feed 置顶（添加、移除或查询置顶会话）、管理标签数据或处理卡片回调时使用。

skills.sh 历史总榜第 67 名快照。原始来源：larksuite/cli；原始 slug：lark-im。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh67_description$
),
(
    $zh68_slug$lark-wiki-larksuite-cli$zh68_slug$,
    'zh-CN',
    $zh68_name$飞书知识库（LarkSuite CLI 版）$zh68_name$,
    $zh68_summary$通过 LarkSuite CLI 管理飞书知识空间、成员、文档节点及层级结构。$zh68_summary$,
    $zh68_description$飞书知识库：管理知识空间、空间成员和文档节点。创建和查询知识空间、查看和管理空间成员、管理节点层级结构、在知识库中组织文档和快捷方式。当用户需要在知识库中查找或创建文档、浏览知识空间结构、查看或管理空间成员、移动或复制节点时使用。用户给出 doubao.com 的 /wiki/ URL/token 时，也应直接使用本 Skill，不因域名不同而回退 WebFetch；路由依据是 URL 路径模式和 token。不负责上传文件到知识库节点下（走 lark-drive），也不负责编辑文档、表格或 Base 内容（走 lark-doc、lark-sheets、lark-base）。

skills.sh 历史总榜第 68 名快照。原始来源：larksuite/cli；原始 slug：lark-wiki。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh68_description$
),
(
    $zh69_slug$lark-sheets-larksuite-cli$zh69_slug$,
    'zh-CN',
    $zh69_name$飞书电子表格（LarkSuite CLI 版）$zh69_name$,
    $zh69_summary$通过 LarkSuite CLI 创建和编辑电子表格，管理结构、单元格、公式、图表、透视表与财务模型。$zh69_summary$,
    $zh69_description$飞书电子表格：创建和操作电子表格。支持创建表格，管理工作表与行列结构（增删、合并、调整尺寸、隐藏、冻结），读写单元格（值、公式、样式、批注、单元格图片），查找替换和多操作批量更新，以及创建和维护图表、透视表、条件格式、筛选器、迷你图、浮动图片等对象。当用户需要创建电子表格、管理工作表、批量读写或编辑数据、统计汇总与可视化、表格美化、公式计算（含 Excel 公式迁移）、金融或财务建模（DCF、三张表、预算、敏感性分析等）时使用。若用户想按名称或关键词搜索云空间中的表格文件，请改用 lark-drive 的 drive +search 定位资源。用户给出 doubao.com 的 /sheets/ URL/token 时，也应直接使用本 Skill；按 URL 路径模式和 token 路由，不因域名不同而回退 WebFetch。

skills.sh 历史总榜第 69 名快照。原始来源：larksuite/cli；原始 slug：lark-sheets。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh69_description$
),
(
    $zh70_slug$lark-whiteboard-larksuite-cli$zh70_slug$,
    'zh-CN',
    $zh70_name$飞书画板（LarkSuite CLI 版）$zh70_name$,
    $zh70_summary$通过 LarkSuite CLI 查询、导出和编辑飞书云文档中的画板。$zh70_summary$,
    $zh70_description$飞书画板：查询和编辑飞书云文档中的画板。支持将画板导出为预览图片、导出原始节点结构，并用多种格式更新画板内容。用户需要查看画板内容、导出画板图片或编辑画板时使用。不负责飞书云文档内容编辑（lark-doc），也不负责文档内嵌电子表格或 Base（lark-sheets、lark-base）。

skills.sh 历史总榜第 70 名快照。原始来源：larksuite/cli；原始 slug：lark-whiteboard。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh70_description$
),
(
    $zh71_slug$lark-task-larksuite-cli$zh71_slug$,
    'zh-CN',
    $zh71_name$飞书任务（LarkSuite CLI 版）$zh71_name$,
    $zh71_summary$通过 LarkSuite CLI 管理任务、清单、子任务、协作者、附件和任务智能体。$zh71_summary$,
    $zh71_description$飞书任务：管理任务、清单和任务智能体。创建待办任务、查看和更新任务状态、拆分子任务、组织任务清单、分配协作成员、上传任务附件、注册或注销任务智能体、更新任务智能体主页数据，并写入智能体任务记录。用户需要创建待办事项、查看任务列表、跟踪任务进度、管理项目清单、给他人分配任务、为任务上传附件、注册或注销任务智能体、更新智能体主页数据或写入任务记录时使用。

skills.sh 历史总榜第 71 名快照。原始来源：larksuite/cli；原始 slug：lark-task。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh71_description$
),
(
    $zh72_slug$lark-calendar-larksuite-cli$zh72_slug$,
    'zh-CN',
    $zh72_name$飞书日历（LarkSuite CLI 版）$zh72_name$,
    $zh72_summary$通过 LarkSuite CLI 查看和管理日程、参会人、忙闲、推荐时段及会议室。$zh72_summary$,
    $zh72_description$飞书日历：管理日历日程和会议室。查看或搜索日程、创建或更新日程、管理参会人、查询忙闲和推荐时段、预定会议室。用户需要查看日程安排、创建或修改会议、查询或预定会议室时使用。不负责查询过去的视频会议记录（走 lark-vc）或待办任务（走 lark-task）。

skills.sh 历史总榜第 72 名快照。原始来源：larksuite/cli；原始 slug：lark-calendar。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh72_description$
),
(
    $zh73_slug$lark-mail-larksuite-cli$zh73_slug$,
    'zh-CN',
    $zh73_name$飞书邮箱（LarkSuite CLI 版）$zh73_name$,
    $zh73_summary$通过 LarkSuite CLI 起草、收发、搜索和管理飞书邮件、文件夹、标签、联系人及收信规则。$zh73_summary$,
    $zh73_description$飞书邮箱：当用户需要起草、发送、回复或转发邮件，查阅或搜索邮件，管理邮件文件夹、邮件标签和邮件联系人，监听新邮件或设置收信规则时使用。仅处理邮件意图；不用于文档、表格、日历、认证设置、纯通讯录查询或即时通讯任务。

skills.sh 历史总榜第 73 名快照。原始来源：larksuite/cli；原始 slug：lark-mail。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh73_description$
),
(
    $zh74_slug$lark-minutes-larksuite-cli$zh74_slug$,
    'zh-CN',
    $zh74_name$飞书妙记（LarkSuite CLI 版）$zh74_name$,
    $zh74_summary$通过 LarkSuite CLI 搜索和编辑妙记，处理音视频、说话人、关键词和访问权限。$zh74_summary$,
    $zh74_description$飞书妙记：搜索妙记、查看基础信息、下载或上传音视频、读取或编辑妙记产物内容、修改标题、替换说话人或关键词，并申请妙记查看或编辑权限。给出 minute_token、本地音视频文件，需要查询、修改或转换妙记产物，或用户明确要主动申请妙记权限时使用。本地音视频转纪要或逐字稿优先使用本 Skill，不使用 ffmpeg 或 whisper 本地转写。不负责获取会议关联妙记，也不负责仅按自然语言标题定位纪要。

skills.sh 历史总榜第 74 名快照。原始来源：larksuite/cli；原始 slug：lark-minutes。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh74_description$
),
(
    $zh75_slug$lark-vc-larksuite-cli$zh75_slug$,
    'zh-CN',
    $zh75_name$飞书视频会议（LarkSuite CLI 版）$zh75_name$,
    $zh75_summary$通过 LarkSuite CLI 查询进行中与历史会议、会中内容、会议纪要和参会人快照。$zh75_summary$,
    $zh75_description$飞书视频会议：查询进行中的会议列表（含会议 ID），读取会中实时内容（发言、聊天、共享等），发送会中消息，以及搜索历史会议、查询会议纪要（总结、待办、章节、逐字稿）和参会人快照。Agent 真实入会或离会走 lark-vc-agent；查询未来日程走 lark-calendar。

skills.sh 历史总榜第 75 名快照。原始来源：larksuite/cli；原始 slug：lark-vc。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh75_description$
),
(
    $zh76_slug$lark-event-larksuite-cli$zh76_slug$,
    'zh-CN',
    $zh76_name$飞书实时事件（LarkSuite CLI 版）$zh76_name$,
    $zh76_summary$通过 LarkSuite CLI 订阅、消费并流式处理飞书消息、审批、任务、会议、妙记和画板事件。$zh76_summary$,
    $zh76_description$用于监听、订阅和消费 Lark/飞书实时事件：通过 lark-cli event consume <EventKey> 以 NDJSON 流式输出事件，覆盖即时消息、表情回复、群聊变更、审批状态、任务更新、视频会议开始、加入与结束、妙记生成、画板更新等。适用于 Lark 机器人、实时消息处理、长期运行的订阅器，以及流式 webhook 或推送处理器。支持通过 --max-events 和 --timeout 限定运行范围，并提供写入 stderr 的就绪标记契约，便于 AI Agent 作为子进程调用。

skills.sh 历史总榜第 76 名快照。原始来源：larksuite/cli；原始 slug：lark-event。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh76_description$
),
(
    $zh77_slug$lark-contact-larksuite-cli$zh77_slug$,
    'zh-CN',
    $zh77_name$飞书通讯录（LarkSuite CLI 版）$zh77_name$,
    $zh77_summary$通过 LarkSuite CLI 在姓名、邮箱与 open_id 之间解析联系人，并查询部门和个人信息。$zh77_summary$,
    $zh77_description$飞书/Lark 通讯录：按姓名或邮箱解析 open_id，按 open_id 反查姓名、部门、邮箱、联系方式、个人状态和签名，并按关键词搜索当前用户可见的机器人或智能体。用户提到一个名字并准备继续发消息或安排日程，或拿到 open_id 后要查询具体信息时使用。不负责遍历部门树、按部门列出员工或生成组织架构图；此类需求使用原生 OpenAPI。

skills.sh 历史总榜第 77 名快照。原始来源：larksuite/cli；原始 slug：lark-contact。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh77_description$
),
(
    $zh78_slug$lark-workflow-meeting-summary-larksuite-cli$zh78_slug$,
    'zh-CN',
    $zh78_name$飞书会议纪要工作流（LarkSuite CLI 版）$zh78_name$,
    $zh78_summary$汇总指定时间范围内的飞书会议纪要，并生成结构化报告或会议周报。$zh78_summary$,
    $zh78_description$会议纪要整理工作流：汇总指定时间范围内的会议纪要并生成结构化报告。用户需要整理会议纪要、生成会议周报或回顾一段时间内的会议内容时使用。

skills.sh 历史总榜第 78 名快照。原始来源：larksuite/cli；原始 slug：lark-workflow-meeting-summary。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh78_description$
),
(
    $zh79_slug$lark-workflow-standup-report-larksuite-cli$zh79_slug$,
    'zh-CN',
    $zh79_name$飞书日程待办摘要（LarkSuite CLI 版）$zh79_name$,
    $zh79_summary$组合日程与任务命令，生成今天、明天或本周的日程及未完成任务摘要。$zh79_summary$,
    $zh79_description$日程待办摘要：编排 calendar +agenda 和 task +get-my-tasks，生成指定日期的日程与未完成任务摘要。适用于了解今天、明天或本周的安排。

skills.sh 历史总榜第 79 名快照。原始来源：larksuite/cli；原始 slug：lark-workflow-standup-report。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh79_description$
),
(
    $zh80_slug$video-edit$zh80_slug$,
    'zh-CN',
    $zh80_name$RunComfy 视频编辑路由$zh80_name$,
    $zh80_summary$按编辑意图在 RunComfy 模型目录中选择合适的视频编辑模型，并提供针对性的提示词模式。$zh80_summary$,
    $zh80_description$在 RunComfy 上编辑现有视频。本 Skill 会按用户意图智能匹配模型：Wan 2.7 Edit-Video 适合通用风格重塑、背景或包装替换并保持身份与动作；Kling 2.6 Pro Motion Control 可把参考视频的精确动作迁移到目标角色；Lucy Edit Restyle 适合轻量、身份稳定的风格或服装替换。内置各模型已记录的提示词模式，减少因选错模型而反复迭代。通过本地 RunComfy CLI 调用 runcomfy run <vendor>/<model>/<endpoint>。用户提出视频编辑、重塑视频、替换视频背景、动作控制、视频服装替换或任何明确的视频转换请求时触发。

skills.sh 历史总榜第 80 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：video-edit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh80_description$
),
(
    $zh81_slug$lark-openapi-explorer-larksuite-cli$zh81_slug$,
    'zh-CN',
    $zh81_name$飞书 OpenAPI 探索（LarkSuite CLI 版）$zh81_name$,
    $zh81_summary$从飞书官方文档中查找并调用尚未由 lark-cli 封装的原生 OpenAPI。$zh81_summary$,
    $zh81_description$飞书/Lark 原生 OpenAPI 探索：从官方文档库中查找未经 CLI 封装的原生 OpenAPI 接口。用户需求无法由现有 lark-* Skill 或 lark-cli 已注册命令满足，需要查找并调用原生飞书 OpenAPI 时使用。

skills.sh 历史总榜第 81 名快照。原始来源：larksuite/cli；原始 slug：lark-openapi-explorer。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh81_description$
),
(
    $zh82_slug$lark-skill-maker-larksuite-cli$zh82_slug$,
    'zh-CN',
    $zh82_name$飞书 Skill 制作器（LarkSuite CLI 版）$zh82_name$,
    $zh82_summary$把飞书 API 原子操作或多步流程封装成可复用的 lark-cli 自定义 Skill。$zh82_summary$,
    $zh82_description$创建 lark-cli 的自定义 Skill。用户需要把飞书 API 操作封装成可复用 Skill，包括包装原子 API 或编排多步流程时使用。

skills.sh 历史总榜第 82 名快照。原始来源：larksuite/cli；原始 slug：lark-skill-maker。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh82_description$
),
(
    $zh83_slug$image-to-video$zh83_slug$,
    'zh-CN',
    $zh83_name$RunComfy 图片转视频$zh83_name$,
    $zh83_summary$按动画意图选择合适的 RunComfy 图片转视频模型，兼顾原生音频、身份保持、口型同步和多模态参考。$zh83_summary$,
    $zh83_description$在 RunComfy 上让任意静态图片动起来。本 Skill 按用户意图智能选择图片转视频模型：HappyHorse 1.0 I2V 适合通用动画，具备 Arena 第一名表现、原生音频和身份保持；Wan 2.7 可通过 audio_url 实现自定义配音口型同步；Seedance 2.0 Pro 可结合图片、参考视频和参考音频进行多模态动画。内置各模型已记录的提示词模式，减少因选错模型而反复迭代。通过本地 RunComfy CLI 调用 runcomfy run <vendor>/<model>/image-to-video 或对应端点变体。用户提出图片转视频、i2v、让图片动起来，或任何把静态图转换为视频的明确请求时触发。

skills.sh 历史总榜第 83 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：image-to-video。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh83_description$
),
(
    $zh84_slug$nano-banana-2$zh84_slug$,
    'zh-CN',
    $zh84_name$Google Nano Banana 2 图像生成$zh84_name$,
    $zh84_summary$在 RunComfy 上使用 Google Nano Banana 2 生成图像，提供该模型的提示词模式、优势、分辨率定价、安全容忍度及模型路由建议。$zh84_summary$,
    $zh84_description$在 RunComfy 上使用 Google Nano Banana 2（Gemini 家族的 Flash 级文生图模型）生成图像。内置该模型已记录的提示词模式，相比对同一模型直接使用普通提示词，可获得更精准的输出。说明 Nano Banana 2 的优势（快速迭代、图内文字渲染、可预测的构图，以及可选的联网事实依据）、各分辨率档位的定价、安全容忍度调节方式，以及何时应改用 Nano Banana Pro、GPT Image 2、Flux 2 或 Seedream。通过本地 RunComfy CLI 调用 `runcomfy run google/nano-banana-2/text-to-image`。当用户提到“nano banana”“nano-banana-2”“nano banana 2”“google image gen”“gemini image”，或明确要求使用此模型生成图像时触发。

skills.sh 历史总榜第 84 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：nano-banana-2。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh84_description$
),
(
    $zh85_slug$nano-banana-edit$zh85_slug$,
    'zh-CN',
    $zh85_name$Google Nano Banana 2 图像编辑$zh85_name$,
    $zh85_summary$在 RunComfy 上使用 Google Nano Banana 2 编辑图像，涵盖主体身份保持、背景替换、空间化局部编辑和最多 20 张输入的批量编辑。$zh85_summary$,
    $zh85_description$在 RunComfy 上使用 Google Nano Banana 2 的图生图编辑端点修改图像。说明 Nano Banana Edit 的优势（保持主体身份、替换背景、用空间语言限定局部修改，以及最多支持 20 张输入的多图批量编辑）、调用结构，以及何时应改用 GPT Image 2 Edit、Flux Kontext 或 Nano Banana 2 文生图。通过本地 RunComfy CLI 调用 `runcomfy run google/nano-banana-2/edit`。当用户提到“nano banana edit”“edit with nano banana”“image edit nano banana”，或明确要求使用此模型编辑图像时触发。

skills.sh 历史总榜第 85 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：nano-banana-edit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh85_description$
),
(
    $zh86_slug$image-edit$zh86_slug$,
    'zh-CN',
    $zh86_name$智能图像编辑$zh86_name$,
    $zh86_summary$根据用户意图，在 RunComfy 图像编辑模型中智能选择 Nano Banana Edit、GPT Image 2 Edit、Flux Kontext Pro 或 Z-Image Turbo Inpaint。$zh86_summary$,
    $zh86_description$在 RunComfy 上编辑图像。本 Skill 是智能路由器，会将用户意图匹配到 RunComfy 目录中的合适编辑模型：Nano Banana Edit（最多批量处理 20 张图，默认保持主体身份）、OpenAI GPT Image 2 Edit（多语言图内文字改写、多参考图合成、精准布局）、Flux Kontext Pro（基于单张参考图的高保真局部编辑），或 Z-Image Turbo Inpaint（由蒙版驱动的精准区域编辑）。内置各模型已记录的提示词模式，避免因选错模型而浪费迭代，并获得更精准的编辑结果。通过本地 RunComfy CLI 调用 `runcomfy run <vendor>/<model>/edit`。当用户提到“image edit”“edit image”“image-to-image”“i2i”“swap background”“remove object”“rewrite headline”，或明确要求编辑单张或一批图像时触发。

skills.sh 历史总榜第 86 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：image-edit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh86_description$
),
(
    $zh87_slug$flux-kontext$zh87_slug$,
    'zh-CN',
    $zh87_name$Flux 1 Kontext Pro 图像编辑$zh87_name$,
    $zh87_summary$在 RunComfy 上使用 Flux 1 Kontext Pro 进行高保真局部图像编辑，提供模型提示词模式、调用结构和替代模型建议。$zh87_summary$,
    $zh87_description$在 RunComfy 上使用 Flux 1 Kontext Pro（Black Forest Labs 的精准局部图像编辑模型）编辑图像。内置该模型已记录的提示词模式，相比直接使用普通提示词，可获得更精准的输出。说明 Flux Kontext 的优势（基于单张参考图的精准局部编辑、强提示词控制、一致的高保真输出）、调用结构（单张图像加提示词），以及何时应改用 Nano Banana Edit、GPT Image 2 Edit 或 Flux 2 Klein。通过本地 RunComfy CLI 调用 `runcomfy run blackforestlabs/flux-1-kontext/pro/edit`。当用户提到“flux kontext”“flux-kontext”“flux 1 kontext”“kontext”“BFL kontext”，或明确要求使用此模型编辑图像时触发。

skills.sh 历史总榜第 87 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：flux-kontext。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh87_description$
),
(
    $zh88_slug$wan-2-7$zh88_slug$,
    'zh-CN',
    $zh88_name$Wan 2.7 文生视频$zh88_name$,
    $zh88_summary$在 RunComfy 上使用 Wan 2.7 生成视频，涵盖多参考条件、音频驱动口型同步、提示词扩展及参数与模型路由建议。$zh88_summary$,
    $zh88_description$在 RunComfy 上使用 Wan 2.7（Wan-AI 的旗舰动态模型）生成文生视频。说明 Wan 2.7 的优势（多参考条件控制、通过 `audio_url` 实现音频驱动的口型同步、更顺畅的转场和提示词扩展）、时长、分辨率与宽高比参数结构，以及何时应改用 HappyHorse 1.0、Seedance 2.0、Kling 或 LTX 2。通过本地 RunComfy CLI 调用 `runcomfy run wan-ai/wan-2-7/text-to-video`。当用户提到“wan”“wan 2.7”“wan-2-7”“wan video”，或明确要求使用此模型生成视频时触发。

skills.sh 历史总榜第 88 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：wan-2-7。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh88_description$
),
(
    $zh89_slug$gpt-image-edit$zh89_slug$,
    'zh-CN',
    $zh89_name$OpenAI GPT Image 2 图像编辑$zh89_name$,
    $zh89_summary$在 RunComfy 上使用 OpenAI GPT Image 2 编辑图像，支持主体保持、多语言图内文字编辑、最多 10 张参考图与精准排版。$zh89_summary$,
    $zh89_description$在 RunComfy 上使用 OpenAI GPT Image 2（ChatGPT Images 2.0 的 `/edit` 端点）编辑图像。内置该模型已记录的提示词模式，相比直接使用普通提示词，可获得更精准的输出。说明 GPT Image Edit 的优势（主体保持措辞、多语言图内文字编辑、最多 10 张参考图、精准布局与排版）、调用结构，以及何时应改用 Nano Banana Edit、Flux Kontext 或 GPT Image 2 文生图。通过本地 RunComfy CLI 调用 `runcomfy run openai/gpt-image-2/edit`。当用户提到“gpt image edit”“gpt-image-edit”“chatgpt image edit”“edit with gpt image 2”，或明确要求使用此模型编辑图像时触发。

skills.sh 历史总榜第 89 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：gpt-image-edit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh89_description$
),
(
    $zh90_slug$seedance-v2$zh90_slug$,
    'zh-CN',
    $zh90_name$Seedance 2.0 Pro 视频生成$zh90_name$,
    $zh90_summary$在 RunComfy 上使用 ByteDance Seedance 2.0 Pro 生成电影感短视频，支持多模态参考、同步原生音频和自然口型同步。$zh90_summary$,
    $zh90_description$在 RunComfy 上使用 ByteDance Seedance 2.0 Pro 生成电影感短视频。说明 Seedance 2.0 Pro 的优势（最多支持 9 张图像、3 段视频和 3 段音频的多模态参考，带自然口型同步的同步原生音频，以及电影感动作优化）、4 至 15 秒的时长结构，以及何时应改用 HappyHorse 1.0、Wan 2.7 或 Kling。通过本地 RunComfy CLI 调用 `runcomfy run bytedance/seedance-v2/pro`。当用户提到“seedance”“seedance 2”“seedance v2”“seedance pro”“bytedance video”，或明确要求使用此模型生成视频时触发。

skills.sh 历史总榜第 90 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：seedance-v2。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh90_description$
),
(
    $zh91_slug$happyhorse-1-0$zh91_slug$,
    'zh-CN',
    $zh91_name$HappyHorse 1.0 文生视频$zh91_name$,
    $zh91_summary$在 RunComfy 上使用 HappyHorse 1.0 生成文生视频，涵盖原生 1080p、同步音频、多镜头角色一致性和六种语言提示词。$zh91_summary$,
    $zh91_description$在 RunComfy 上使用 HappyHorse 1.0 生成文生视频。说明 HappyHorse 1.0 的优势（Artificial Analysis Video Arena 排名第一、原生 1080p 与同步原生音频、多镜头角色一致性、支持六种语言的提示词）、时长、宽高比与分辨率参数结构，以及何时应改用 Wan 2.7、Seedance 2 或 LTX 2。通过本地 RunComfy CLI 调用 `runcomfy run happyhorse/happyhorse-1-0/text-to-video`。当用户提到“happyhorse”“happy horse”“happyhorse 1.0”“happyhorse video”，或明确要求使用此模型生成视频时触发。

skills.sh 历史总榜第 91 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：happyhorse-1-0。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh91_description$
),
(
    $zh92_slug$flux-2-klein$zh92_slug$,
    'zh-CN',
    $zh92_name$Flux 2 Klein 图像生成$zh92_name$,
    $zh92_summary$在 RunComfy 上使用 Flux 2 Klein 快速生成图像，涵盖多参考品牌风格、提示词模式、步数策略及 9B 与 4B 版本取舍。$zh92_summary$,
    $zh92_description$在 RunComfy 上使用 Flux 2 Klein（Black Forest Labs 的 Flux 2 蒸馏高速版本）生成图像。内置该模型已记录的提示词模式，相比直接使用普通提示词，可获得更精准的输出。说明 Flux 2 Klein 的优势（亚秒级延迟、多参考品牌风格、以主体为先的声明式提示词）、步数策略（快速迭代使用 4 至 8 步，精修约 25 步）、9B 与 4B 版本的取舍，以及何时应改用 Flux 2 Pro、Seedream 5 或 GPT Image 2。通过本地 RunComfy CLI 调用 `runcomfy run blackforestlabs/flux-2-klein/9b/text-to-image`（或 `/4b/`）。当用户提到“flux 2 klein”“flux-2-klein”“flux klein”“BFL flux 2”，或明确要求使用此模型生成图像时触发。

skills.sh 历史总榜第 92 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：flux-2-klein。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh92_description$
),
(
    $zh93_slug$azure-quotas$zh93_slug$,
    'zh-CN',
    $zh93_name$Azure 配额管理$zh93_name$,
    $zh93_summary$跨资源提供商查询和管理 Azure 配额与用量，用于部署规划、容量验证、区域选择和配额提升申请。$zh93_summary$,
    $zh93_description$跨资源提供商查询和管理 Azure 配额与用量，用于部署规划、容量验证和区域选择。适用于“检查配额”“服务限制”“当前用量”“申请提高配额”“配额超限”“验证容量”“区域可用性”“预配限制”“vCPU 限制”或“我的订阅还有多少可用 vCPU”等请求。

skills.sh 历史总榜第 93 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-quotas。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh93_description$
),
(
    $zh94_slug$domain-modeling$zh94_slug$,
    'zh-CN',
    $zh94_name$领域建模$zh94_name$,
    $zh94_summary$构建并完善项目的领域模型，明确领域术语与通用语言，并记录相关架构决策。$zh94_summary$,
    $zh94_description$构建并完善项目的领域模型。当用户希望明确领域术语或通用语言、记录架构决策，或其他 Skill 需要维护领域模型时使用。

skills.sh 历史总榜第 94 名快照。原始来源：mattpocock/skills；原始 slug：domain-modeling。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh94_description$
),
(
    $zh95_slug$azure-upgrade$zh95_slug$,
    'zh-CN',
    $zh95_name$Azure 工作负载升级$zh95_name$,
    $zh95_summary$评估并升级 Azure 工作负载的计划、层级或 SKU，或在源代码中现代化 Azure SDK 依赖。$zh95_summary$,
    $zh95_description$评估 Azure 工作负载并在不同计划、层级或 SKU 之间升级，或在源代码中将 Azure SDK 依赖现代化。适用于将 Consumption 升级到 Flex Consumption、升级 Azure Functions 计划、更改托管计划或 Function App SKU、将 App Service 迁移到 Container Apps、将旧版 Azure Java SDK（com.microsoft.azure）现代化为 com.azure，以及将 Azure Cache for Redis（ACR/ACRE）迁移到 Azure Managed Redis（AMR）。

skills.sh 历史总榜第 95 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-upgrade。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh95_description$
),
(
    $zh96_slug$lark-approval-larksuite-cli$zh96_slug$,
    'zh-CN',
    $zh96_name$飞书审批（larksuite/cli）$zh96_name$,
    $zh96_summary$查询和处理飞书审批待办、已办与实例，搜索可发起的审批定义、查看定义详情并发起原生审批实例。$zh96_summary$,
    $zh96_description$飞书审批：查询和处理审批待办、已办与实例，搜索可发起审批定义、查看定义详情并发起原生审批实例。当用户要处理审批任务、查看审批实例、搜索或发起审批时使用。审批待办不是飞书任务；非审批类待办走 lark-task。不负责创建审批定义；三方审批定义不走原生提单。

skills.sh 历史总榜第 96 名快照。原始来源：larksuite/cli；原始 slug：lark-approval。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh96_description$
),
(
    $zh97_slug$kling-3-0$zh97_slug$,
    'zh-CN',
    $zh97_name$Kling 3.0 视频生成$zh97_name$,
    $zh97_summary$在 RunComfy 上使用 Kling 3.0 生成视频，覆盖 Standard、Pro、4K 三档渲染和文生视频、图生视频两种模式。$zh97_summary$,
    $zh97_description$在 RunComfy 上使用 Kling 3.0 生成视频。Kling 3.0（又称 Kling V3.0）是 Kuaishou Technology 的第三代多镜头视频模型，支持原生同步音频，并能在不同镜头间保持角色身份一致。本 Skill 覆盖 Kling 3.0 的全部六个端点，包括 Standard、Pro、4K 三档渲染，以及文生视频、图生视频两种模式。通过本地 RunComfy CLI 调用 `runcomfy run kling/kling-3.0/<tier>/<mode>`。当用户提到“kling”“kling 3.0”“kling v3”“kling pro”“kling 4k”“kling text to video”“kling image to video”，或明确要求使用 Kling 3.0 生成或制作动画时触发。

skills.sh 历史总榜第 97 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：kling-3-0。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh97_description$
),
(
    $zh98_slug$codebase-design$zh98_slug$,
    'zh-CN',
    $zh98_name$代码库设计$zh98_name$,
    $zh98_summary$提供设计深模块的共享术语，用于改善模块接口、寻找深化机会、确定边界并提升可测试性与 AI 可导航性。$zh98_summary$,
    $zh98_description$提供设计深模块的共享术语。当用户希望设计或改进模块接口、寻找模块深化机会、决定边界应放在哪里、让代码更易测试或更便于 AI 导航，或其他 Skill 需要使用深模块术语时使用。

skills.sh 历史总榜第 98 名快照。原始来源：mattpocock/skills；原始 slug：codebase-design。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh98_description$
),
(
    $zh99_slug$paper-context-resolver$zh99_slug$,
    'zh-CN',
    $zh99_name$论文上下文解析$zh99_name$,
    $zh99_summary$为 README 优先的深度学习仓库复现补齐关键论文细节，仅在仓库资料留下狭窄且影响复现的缺口时使用。$zh99_summary$,
    $zh99_description$Rigor 论文上下文助手，用于 README 优先的深度学习仓库复现。仅当 README 和仓库文件留下一个范围明确、对复现至关重要的缺口，并且任务需要从论文的一手资料中查明数据集划分、预处理、评估协议、检查点映射或运行时假设等具体细节，同时记录冲突时使用。不要用于一般性的论文摘要、仓库扫描、环境搭建、命令执行、仅按标题查找论文，也不要默认以论文内容取代 README 指引。

skills.sh 历史总榜第 99 名快照。原始来源：lllllllama/rigorpilot-skills；原始 slug：paper-context-resolver。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh99_description$
),
(
    $zh100_slug$hyperframes-cli$zh100_slug$,
    'zh-CN',
    $zh100_name$HyperFrames 命令行工具$zh100_name$,
    $zh100_summary$指导使用 HyperFrames CLI 完成本地与云端的开发、检查、预览、渲染、发布和故障诊断流程。$zh100_summary$,
    $zh100_description$使用 HyperFrames CLI 开发循环：init、add、catalog、capture、lint、check、snapshot、compare、grade-compare、preview、play、present、beats、keyframes、单个或批量 render、publish、cloud、cloudrun、feedback、lambda、doctor、browser、info、upgrade、skills、compositions、docs、benchmark、telemetry、transcribe、auth、tts 和 remove-background。诊断构建或渲染失败时也应使用。validate、inspect 和 layout 是已弃用的别名，请改用 check。覆盖本地渲染、HeyGen 托管云、AWS Lambda 和 Google Cloud Run 渲染。

skills.sh 历史总榜第 100 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes-cli。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh100_description$
),
(
    $zh101_slug$diagnosing-bugs$zh101_slug$,
    'zh-CN',
    $zh101_name$疑难问题诊断$zh101_name$,
    $zh101_summary$为棘手 Bug 和性能退化提供系统化诊断循环，适用于排错、调试、异常、失败或速度变慢等问题。$zh101_summary$,
    $zh101_description$用于棘手 Bug 和性能退化的诊断循环。当用户要求“诊断”或“调试”，或报告功能损坏、抛出异常、运行失败或速度变慢时使用。

skills.sh 历史总榜第 101 名快照。原始来源：mattpocock/skills；原始 slug：diagnosing-bugs。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh101_description$
),
(
    $zh102_slug$repo-intake-and-plan$zh102_slug$,
    'zh-CN',
    $zh102_name$仓库接入与复现计划$zh102_name$,
    $zh102_summary$扫描 README 与常见项目文件，提取文档命令、分类运行候选项，并为深度学习仓库生成最小可信复现计划。$zh102_summary$,
    $zh102_description$Rigor 接入助手，用于 README 优先的深度学习仓库复现。当任务明确要求扫描仓库、阅读 README 和常见项目文件、提取已记录的命令、分类推理、评估与训练候选项，并向主编排器返回最小可信复现计划时使用。不要用于环境搭建、资产下载、命令执行、最终报告、论文查找或端到端编排。

skills.sh 历史总榜第 102 名快照。原始来源：lllllllama/rigorpilot-skills；原始 slug：repo-intake-and-plan。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh102_description$
),
(
    $zh103_slug$minimal-run-and-audit$zh103_slug$,
    'zh-CN',
    $zh103_name$最小化运行与审计$zh103_name$,
    $zh103_summary$从选定的冒烟测试、推理或评估命令中采集并规范化证据，生成标准化的 `repro_outputs/` 文件。$zh103_summary$,
    $zh103_description$Rigor 运行 Skill，用于 README 优先的深度学习仓库复现。当任务明确要求从选定的冒烟测试或文档中的推理、评估命令采集并规范化证据，写入标准化的 `repro_outputs/` 文件，并在仓库文件发生变化时附上补丁说明时使用。不要用于执行训练、初始仓库接入、通用环境搭建、论文查找、目标选择、隐蔽改变科学含义，或单独承担端到端编排。

skills.sh 历史总榜第 103 名快照。原始来源：lllllllama/rigorpilot-skills；原始 slug：minimal-run-and-audit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh103_description$
),
(
    $zh104_slug$env-and-assets-bootstrap$zh104_slug$,
    'zh-CN',
    $zh104_name$环境与资产初始化$zh104_name$,
    $zh104_summary$在运行 README 所记录的深度学习仓库之前，准备保守的 Conda 优先环境、检查点和数据集路径假设及缓存说明。$zh104_summary$,
    $zh104_description$Rigor 搭建 Skill，用于 README 优先的深度学习仓库复现。当任务明确要求在运行 README 所记录的仓库前，准备保守的 Conda 优先环境、检查点与数据集路径假设、缓存位置提示和搭建说明时使用。不要用于仓库扫描、完整编排、论文解读、最终运行报告，或与具体复现目标无关的通用环境搭建。

skills.sh 历史总榜第 104 名快照。原始来源：lllllllama/rigorpilot-skills；原始 slug：env-and-assets-bootstrap。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh104_description$
),
(
    $zh105_slug$codex-pet$zh105_slug$,
    'zh-CN',
    $zh105_name$Codex Pet 生成器$zh105_name$,
    $zh105_summary$根据一张参考图，在 RunComfy 上生成兼容 Codex 的 `spritesheet.webp` 和 `pet.json` 自定义桌面宠物。$zh105_summary$,
    $zh105_description$RunComfy 上的 Codex Pet 生成器。根据一张参考图构建兼容 Codex 的 Codex Pet `spritesheet.webp` 和 `pet.json`，将其放入 `${CODEX_HOME:-$HOME/.codex}/pets/<name>/` 后，Codex 就会把它作为自定义 Codex Pet，与 8 个内置宠物一同加载。本 Skill 会生成 Codex 所要求的精确图集：1536×1872 PNG/WebP、8 列×9 行、每格 192×208，共 9 种动画状态——idle、running-right、running-left、waving、jumping、failed、waiting、running 和 review。它先通过本地 RunComfy CLI 以 `runcomfy run openai/gpt-image-2/edit` 调用一次 OpenAI GPT Image 2 Edit，生成标准 Codex Pet 姿势，再通过 ImageMagick 的微变换以编程方式组装全部 9 行动画；无需 Codex Pro、`$imagegen` 或 OPENAI_API_KEY，只需 RUNCOMFY_TOKEN。当用户提到“codex pet”“create codex pet”“make codex pet”“hatch codex pet”“/hatch image”“desktop pet codex”“codex pets”“spritesheet.webp”，或明确要求为 OpenAI Codex 构建自定义宠物时触发。

skills.sh 历史总榜第 105 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：codex-pet。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh105_description$
),
(
    $zh106_slug$to-prd$zh106_slug$,
    'zh-CN',
    $zh106_name$从对话生成 PRD$zh106_name$,
    $zh106_summary$将当前对话上下文整理为产品需求文档（PRD），并以 GitHub Issue 的形式提交。$zh106_summary$,
    $zh106_description$将当前对话上下文整理为产品需求文档（PRD），并以 GitHub Issue 的形式提交。当用户希望根据当前上下文创建 PRD 时使用。

skills.sh 历史总榜第 106 名快照。原始来源：mattpocock/skills；原始 slug：to-prd。打包时排除了 0 个上游文件。该排行榜条目在仓库 HEAD 中不存在，已从历史提交 aaf3050857a8d00c710c382c87a29b81341370aa 恢复。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh106_description$
),
(
    $zh107_slug$lark-slides-larksuite-cli$zh107_slug$,
    'zh-CN',
    $zh107_name$飞书幻灯片（larksuite/cli）$zh107_name$,
    $zh107_summary$创建和编辑飞书幻灯片，支持创建演示文稿、读取内容，以及创建、删除、读取和局部替换单个页面。$zh107_summary$,
    $zh107_description$飞书幻灯片：创建和编辑幻灯片。创建演示文稿、读取幻灯片内容、管理幻灯片页面（创建、删除、读取、局部替换）。当用户需要创建或编辑幻灯片、读取或修改单个页面时使用。当用户给出 doubao.com 的 /slides/ URL/token 时，也应直接使用本 Skill，不要因为域名不是飞书而回退到 WebFetch；路由依据是 URL 路径模式和 token，而不是域名。不负责：云文档内容编辑（走 lark-doc）、云文档里的独立画板对象（走 lark-whiteboard）、上传或下载普通文件（走 lark-drive）。

skills.sh 历史总榜第 107 名快照。原始来源：larksuite/cli；原始 slug：lark-slides。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh107_description$
),
(
    $zh108_slug$hyperframes$zh108_slug$,
    'zh-CN',
    $zh108_name$HyperFrames 总入口$zh108_name$,
    $zh108_summary$处理所有 HyperFrames 视频、动画和动态图形任务的必读入口，负责恢复项目状态、识别意图并路由到对应工作流。$zh108_summary$,
    $zh108_description$必读入口：凡是制作、创建、编辑、动画化或渲染视频、动画或动态图形的请求，都应先阅读本 Skill，包括宣传片、解说视频、字幕短片、标题卡、叠加层、幻灯片或交互式演示、Remotion 移植，以及任何 HyperFrames HTML 合成。检查、诊断、验证、预览、发布或批量渲染现有 HyperFrames 项目时也应使用。输入可以是网站 URL、GitHub PR、Figma 设计或 URL、文本或简报、现有素材或音乐。它会恢复项目状态，在适用时捕获用户意图，选择并安装所属工作流，然后路由领域能力。除非用户明确为交付物选择其他框架，或只要求录制浏览器会话，否则默认使用 HyperFrames 作为输出框架。

skills.sh 历史总榜第 108 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh108_description$
),
(
    $zh109_slug$lark-attendance-larksuite-cli$zh109_slug$,
    'zh-CN',
    $zh109_name$飞书考勤打卡（larksuite/cli）$zh109_name$,
    $zh109_summary$查询当前用户自己的飞书考勤打卡记录。$zh109_summary$,
    $zh109_description$飞书考勤打卡：查询自己的考勤打卡记录。

skills.sh 历史总榜第 109 名快照。原始来源：larksuite/cli；原始 slug：lark-attendance。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh109_description$
),
(
    $zh110_slug$azure-enterprise-infra-planner$zh110_slug$,
    'zh-CN',
    $zh110_name$Azure 企业基础设施规划$zh110_name$,
    $zh110_summary$根据工作负载描述设计并预配符合 WAF 的企业级 Azure 网络、身份、安全、合规和多资源拓扑。$zh110_summary$,
    $zh110_description$根据工作负载描述设计并预配企业级 Azure 基础设施。面向规划网络、身份、安全、合规和符合 WAF 的多资源拓扑的云架构师与平台工程师。直接生成 Bicep 或 Terraform，不使用 azd。适用于“规划 Azure 基础设施”“设计 Azure landing zone”“设计 hub-spoke 网络”“规划多区域灾难恢复拓扑”“设置 VNet、防火墙和专用终结点”“订阅范围的 Bicep 部署”“为 VM 工作负载配置 Azure Backup”等请求。以应用为中心的工作流优先使用 azure-prepare。

skills.sh 历史总榜第 110 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-enterprise-infra-planner。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh110_description$
),
(
    $zh111_slug$to-issues$zh111_slug$,
    'zh-CN',
    $zh111_name$方案拆分为 GitHub Issues$zh111_name$,
    $zh111_summary$使用示踪弹垂直切片，将计划、规格或 PRD 拆分为可独立认领的 GitHub Issues。$zh111_summary$,
    $zh111_description$使用示踪弹垂直切片，将计划、规格或 PRD 拆分为可独立认领的 GitHub Issues。当用户希望把计划转换为 Issues、创建实现任务票或将工作拆解为 Issues 时使用。

skills.sh 历史总榜第 111 名快照。原始来源：mattpocock/skills；原始 slug：to-issues。打包时排除了 0 个上游文件。该排行榜条目在仓库 HEAD 中不存在，已从历史提交 aaf3050857a8d00c710c382c87a29b81341370aa 恢复。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh111_description$
),
(
    $zh112_slug$design-taste-frontend$zh112_slug$,
    'zh-CN',
    $zh112_name$高品味前端设计$zh112_name$,
    $zh112_summary$面向落地页、作品集和重设计的反模板化前端 Skill，根据简报判断设计方向并交付具有真实设计系统的界面。$zh112_summary$,
    $zh112_description$面向落地页、作品集和重设计的反粗制滥造前端 Skill。代理会阅读简报，判断正确的设计方向，并交付不显模板化的界面；适用时建立真实设计系统，重设计时先做审计，并执行严格的开发前检查。

skills.sh 历史总榜第 112 名快照。原始来源：leonxlnx/taste-skill；原始 slug：design-taste-frontend。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh112_description$
),
(
    $zh113_slug$skill-creator$zh113_slug$,
    'zh-CN',
    $zh113_name$Skill 创建器$zh113_name$,
    $zh113_summary$创建、修改和优化 Skill，并通过评测、基准测试和方差分析衡量性能与触发准确率。$zh113_summary$,
    $zh113_description$创建新 Skill，修改和改进现有 Skill，并衡量 Skill 性能。当用户希望从零创建 Skill、编辑或优化现有 Skill、运行评测来测试 Skill、通过方差分析对 Skill 性能做基准测试，或优化 Skill 描述以提高触发准确率时使用。

skills.sh 历史总榜第 113 名快照。原始来源：anthropics/skills；原始 slug：skill-creator。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh113_description$
),
(
    $zh114_slug$ai-video-generation$zh114_slug$,
    'zh-CN',
    $zh114_name$AI 视频生成$zh114_name$,
    $zh114_summary$通过 RunComfy CLI 在完整视频模型目录中智能路由，为文生视频、图生视频和视频续写选择合适模型。$zh114_summary$,
    $zh114_description$通过 `runcomfy` CLI 在 RunComfy 上生成 AI 视频。本 Skill 是覆盖完整视频模型目录的智能路由器：HappyHorse 1.0（Arena 第一、原生音频）、Wan-AI Wan 2-7（开放权重、音频驱动口型同步）、ByteDance Seedance v2、1-5、1-0（多模态电影感）、Kling 3.0、2-6、Google Veo 3-1、MiniMax Hailuo 2-3 和 ByteDance Dreamina 3-0。覆盖文生视频（t2v）、图生视频（i2v）以及 Veo 的视频续写端点。它会根据用户的实际意图选择合适模型，例如 Arena 第一的质量、多镜头角色身份一致性、原生音频、电影感运动、最快生成、15 秒以内短片或最长时长，并提供各模型已记录的提示词模式和最小 `runcomfy run` 调用。当用户要求“生成视频”“制作视频”“文生视频”“t2v”“图生视频”“i2v”“制作动画”“AI 视频”“让某物动起来”“根据提示词生成视频”“根据图像生成视频”，或明确要求从提示词或静态图生成视频片段时触发。

skills.sh 历史总榜第 114 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：ai-video-generation。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh114_description$
),
(
    $zh115_slug$ask-matt$zh115_slug$,
    'zh-CN',
    $zh115_name$询问 Matt$zh115_name$,
    $zh115_summary$判断当前情况适合使用哪个 Skill 或工作流，作为本仓库中所有 Skill 的路由器。$zh115_summary$,
    $zh115_description$询问当前情况适合使用哪个 Skill 或工作流。作为本仓库中各 Skill 的路由器。

skills.sh 历史总榜第 115 名快照。原始来源：mattpocock/skills；原始 slug：ask-matt。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh115_description$
),
(
    $zh116_slug$ai-image-generation$zh116_slug$,
    'zh-CN',
    $zh116_name$AI 图像生成与编辑$zh116_name$,
    $zh116_summary$通过 RunComfy CLI 在完整图像模型目录中智能路由，为文生图、图生图和图像编辑选择合适模型。$zh116_summary$,
    $zh116_description$通过 `runcomfy` CLI 在 RunComfy 上生成和编辑图像。本 Skill 是覆盖完整图像模型目录的智能路由器：FLUX 2（Klein 9B/4B、Pro、Dev、Flash、Turbo、Max）、Google Nano Banana 2 与 Pro、OpenAI GPT Image 2、ByteDance Seedream 5、4-5、4-0 与 Dreamina 4-0、Alibaba Qwen Image、Z-Image Turbo，以及 Wan 2-7。覆盖文生图（t2i）和图生图或编辑（i2i）端点。它会根据用户的实际意图选择合适模型，例如精准排版、写实人像、亚秒级迭代、多参考品牌风格或开放权重工作流，并提供各模型已记录的提示词模式和最小 `runcomfy run` 调用。当用户要求“生成图像”“制作图片”“文生图”“AI 图像”“生成一张……的图”“图生图”“i2i”，或明确要求创建或重绘图像时触发。

skills.sh 历史总榜第 116 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：ai-image-generation。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh116_description$
),
(
    $zh117_slug$azure-kubernetes$zh117_slug$,
    'zh-CN',
    $zh117_name$Azure Kubernetes 规划$zh117_name$,
    $zh117_summary$规划、创建并配置可用于生产的 Azure Kubernetes Service（AKS）集群，覆盖网络、安全、扩缩容、升级与成本。$zh117_summary$,
    $zh117_description$规划、创建并配置可用于生产的 Azure Kubernetes Service（AKS）集群。覆盖 Day-0 清单、SKU 选择（Automatic 或 Standard）、网络选项（专用 API Server、Azure CNI Overlay、出站配置）、安全，以及运维（自动扩缩容、升级策略、成本分析）。适用于创建或预配 AKS 环境、启用 AKS 可观测性、设计 AKS 网络、选择 AKS SKU、保护或优化 AKS、AKS Spot 节点、AKS Cluster Autoscaler、调整 AKS Pod 规模、Pod 资源规格、资源过度预配的 AKS Pod、Pod 资源请求与限制、Vertical Pod Autoscaler 和 VPA 建议等请求。

skills.sh 历史总榜第 117 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-kubernetes。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh117_description$
),
(
    $zh118_slug$runcomfy-cli$zh118_slug$,
    'zh-CN',
    $zh118_name$RunComfy 命令行工具$zh118_name$,
    $zh118_summary$指导安装、认证和使用 RunComfy CLI，从命令行发现并调用数百个图像、视频和音频模型端点。$zh118_summary$,
    $zh118_description$从命令行运行 RunComfy 上的任意模型。`runcomfy` CLI 以单个二进制和一套认证提供数百个模型端点，包括图像生成、图像编辑、视频生成、图生视频、口型同步、换脸、视频编辑、局部重绘、扩图、续写、ControlNet、重打光、放大、LoRA 训练等。提交请求、轮询状态并下载输出。本 Skill 会教代理如何安装、认证、发现模型参数结构、调用模型、使用流式、轮询或免等待模式、在 JSON 输出模式下编写脚本，以及处理错误。当用户提到“runcomfy cli”“install runcomfy”“runcomfy login”“runcomfy run”“runcomfy whoami”“runcomfy api”，或明确要求从脚本或终端调用 RunComfy 模型时触发。相关 Skill（ai-image-generation、ai-video-generation、image-edit、video-edit、face-swap、lipsync、image-to-video、image-inpainting、image-outpainting、video-extend、controlnet-pose、relight）均通过此 CLI 调度。

skills.sh 历史总榜第 118 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：runcomfy-cli。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh118_description$
),
(
    $zh119_slug$ai-avatar-video$zh119_slug$,
    'zh-CN',
    $zh119_name$AI 数字人视频$zh119_name$,
    $zh119_summary$通过 RunComfy CLI 创建 AI 数字人、口播和口型同步视频，并根据全身、肖像、配音或电影感场景选择模型。$zh119_summary$,
    $zh119_description$通过 `runcomfy` CLI 在 RunComfy 上创建 AI 数字人、口播和口型同步视频。可在 ByteDance OmniHuman（音频驱动的全身数字人）、Wan-AI Wan 2-7（通过肖像上的 `audio_url` 实现音频驱动口型同步）、HappyHorse 1.0（Arena 第一，支持带原生音频的 t2v 与 i2v）以及 Seedance v2 Pro（参考音频加参考主体的多模态电影感视频）之间路由。根据用户的实际意图选择合适模型，例如 UGC 旁白、虚拟主持人、配音产品演示、口型同步角色或对话场景，并提供各模型已记录的提示词模式和最小 `runcomfy run` 调用。当用户提到“talking head”“lip sync”“avatar video”“make X speak”“audio to video”“audio driven avatar”“virtual presenter”“AI spokesperson”“dubbed video”“UGC avatar”“HeyGen alternative”“Synthesia alternative”“digital human”“make this portrait talk”“video from voiceover”，或明确要求让画面中的人物说话时触发。

skills.sh 历史总榜第 119 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：ai-avatar-video。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh119_description$
),
(
    $zh120_slug$face-swap$zh120_slug$,
    'zh-CN',
    $zh120_name$人脸与角色替换$zh120_name$,
    $zh120_summary$通过 RunComfy CLI 在图像或视频中替换人脸与角色，并按静态图、视频、全角色或仅人脸等意图选择模型。$zh120_summary$,
    $zh120_description$通过 `runcomfy` CLI 在 RunComfy 上将人脸或角色替换进视频或图像。可在社区版 Wan 2-2 Animate（音频驱动的角色动画与身份替换）、GPT Image 2 Edit（通过参考图合成，对静态图做单次精准换脸）、Nano Banana Edit（批量且保持身份的替换）、Flux Kontext（单参考图高保真局部人脸编辑）以及 Kling 2-6 Motion Control Pro（把一段表演的动作迁移到目标角色）之间路由。根据用户的实际意图选择合适模型，包括单张静态图还是视频、完整角色还是仅人脸、对话场景还是无声动作。当用户提到“face swap”“swap face”“deepfake”“face replacement”“character swap”“head swap”“put X's face on Y”“make this video star X”“replace the actor in this video”“swap the character in the photo”“deepfake video”“ReActor alternative”，或明确要求用另一个身份替换现有身份时触发。

skills.sh 历史总榜第 120 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：face-swap。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh120_description$
),
(
    $zh121_slug$lark-note$zh121_slug$,
    'zh-CN',
    $zh121_name$飞书会议纪要直查$zh121_name$,
    $zh121_summary$在已知 `note_id` 时查询飞书会议纪要详情、展示类型、关联文档 token 和 unified 原始逐字记录。$zh121_summary$,
    $zh121_description$飞书会议纪要（Note）直查：已知 note_id 时查询纪要详情、展示类型、关联文档 token，并读取 unified 原始逐字记录。当用户已持有 note_id，或从文档显式 vc-node-id 获得 note_id 时使用。不负责会议、日程或妙记定位、文档标题搜索或 Docx 正文读取。

skills.sh 历史总榜第 121 名快照。原始来源：open.feishu.cn；原始 slug：lark-note。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh121_description$
),
(
    $zh122_slug$supabase-postgres-best-practices$zh122_slug$,
    'zh-CN',
    $zh122_name$Supabase Postgres 最佳实践$zh122_name$,
    $zh122_summary$Supabase 维护的通用 Postgres 最佳实践，覆盖数据库架构、迁移、安全、SQL 编写、性能诊断和运维。$zh122_summary$,
    $zh122_description$由 Supabase 维护、适用于任意运行环境的 Postgres 最佳实践。凡是要编写或修改 Postgres 数据库中的内容，都应先加载本 Skill，包括创建或修改表和列（含字段类型选择）、架构设计、迁移与声明式架构文件、RLS 策略及其验证测试、索引、触发器、数据库函数、队列与定时任务（pg_cron、pgmq）、向量或语义搜索（pgvector），以及恢复转储（pg_restore）或导入数据。诊断慢查询、高 CPU、超时、EXPLAIN 计划、连接耗尽、锁、膨胀，或数据行对错误用户或租户可见时也应加载。它不只是性能指南；架构、迁移、安全和 SQL 编写任务同样需要遵守这些规则，即使只改一列或只写一条查询。

skills.sh 历史总榜第 122 名快照。原始来源：supabase/agent-skills；原始 slug：supabase-postgres-best-practices。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh122_description$
),
(
    $zh123_slug$video-inpainting$zh123_slug$,
    'zh-CN',
    $zh123_name$视频局部重绘$zh123_name$,
    $zh123_summary$通过 RunComfy CLI 跨视频帧编辑指定区域，可移除持续出现的对象、清理线缆或水印，并以匹配的运动替换区域。$zh123_summary$,
    $zh123_description$通过 `runcomfy` CLI 在 RunComfy 上跨视频帧编辑区域：移除在多帧中持续出现的对象、清理线缆或水印，或用匹配的运动替换某一区域。可在 Wan 2-7 edit-video（默认方案，以提示词和空间语言驱动区域编辑）、Lucy Edit Restyle（保持身份稳定、感知区域的重绘）和 Seedream 4-0 edit-sequential（将视频片段视为帧堆栈时）之间路由。根据修改是由自然语言驱动、需要锁定身份，还是需要把逐帧静态局部重绘串成视频来选择方案。当用户提到“video inpaint”“video inpainting”“remove from video”“mask region in video”“clean up video”“remove object from clip”“video patch”“frame-by-frame edit”“remove watermark from video”“remove passing person”，或明确要求跨视频帧编辑某一区域时触发。

skills.sh 历史总榜第 123 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：video-inpainting。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh123_description$
),
(
    $zh124_slug$lark-okr-larksuite-cli$zh124_slug$,
    'zh-CN',
    $zh124_name$飞书 OKR（larksuite/cli）$zh124_name$,
    $zh124_summary$管理飞书 OKR 周期、目标、关键结果、对齐关系、量化指标和进展记录。$zh124_summary$,
    $zh124_description$飞书 OKR：管理目标与关键结果。查看和编辑 OKR 周期、目标、关键结果、对齐关系、量化指标和进展记录。当用户需要查看或创建 OKR、管理目标和关键结果、查看对齐关系时使用。不负责：待办任务管理（lark-task）、日程或会议安排（lark-calendar）、绩效评估。

skills.sh 历史总榜第 124 名快照。原始来源：larksuite/cli；原始 slug：lark-okr。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh124_description$
),
(
    $zh125_slug$image-inpainting$zh125_slug$,
    'zh-CN',
    $zh125_name$图像局部重绘$zh125_name$,
    $zh125_summary$通过 RunComfy CLI 进行蒙版驱动的图像局部重绘，支持对象与水印移除、区域替换和瑕疵清理。$zh125_summary$,
    $zh125_description$通过 `runcomfy` CLI 在 RunComfy 上进行蒙版驱动的图像局部重绘。当提供蒙版时，路由到 Tongyi MAI Z-Image Turbo Inpainting 专用端点，该端点支持蒙版、强度和控制比例；没有蒙版、必须用语言描述区域时，则路由到能够保持主体身份的编辑模型（Nano Banana 2 Edit、GPT Image 2 Edit、FLUX Kontext Pro）。适用于对象移除、水印移除、区域替换、瑕疵清理，以及任何由二值蒙版定义目标区域的受控局部编辑。当用户提到“inpaint”“inpainting”“image inpaint”“remove from image”“fill region”“mask-driven edit”“remove watermark”“remove object”“patch the photo”“fill the hole”，或明确要求编辑静态图中的特定蒙版区域时触发。

skills.sh 历史总榜第 125 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：image-inpainting。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh125_description$
),
(
    $zh126_slug$controlnet-pose$zh126_slug$,
    'zh-CN',
    $zh126_name$ControlNet 姿态控制$zh126_name$,
    $zh126_summary$通过 RunComfy CLI 按姿态、骨架、动作、深度或 Canny 参考条件生成图像与视频，并智能选择对应模型。$zh126_summary$,
    $zh126_description$通过 `runcomfy` CLI 在 RunComfy 上进行姿态条件生成。可在 Kling 2-6 Motion Control Pro 或 Standard（把参考视频的动作与走位迁移到目标角色）、社区版 Wan 2-2 Animate（带姿态条件的音频驱动角色动画）和 Z-Image Turbo ControlNet LoRA（根据 OpenPose、DWPose、Canny 或深度控制图生成姿态受控图像）之间路由。根据视频还是静态图、风格化还是真实感来选择合适方案。当用户提到“controlnet”“control net”“pose control”“openpose”“DWPose”“transfer pose”“motion control”“pose driven”“character pose”“depth control”“canny edge”“use this pose”，或明确要求以姿态、骨架、动作、深度或 Canny 参考作为生成条件时触发。

skills.sh 历史总榜第 126 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：controlnet-pose。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh126_description$
),
(
    $zh127_slug$lipsync$zh127_slug$,
    'zh-CN',
    $zh127_name$口型同步$zh127_name$,
    $zh127_summary$通过 RunComfy CLI 将人脸口型与指定音轨同步，并按肖像、现有视频或脚本生成等意图选择模型端点。$zh127_summary$,
    $zh127_description$通过 `runcomfy` CLI 在 RunComfy 上让人脸口型与指定音轨同步。可在 ByteDance OmniHuman（由肖像和音频生成音频驱动的全身数字人）、Sync Labs sync v2 或 Pro（把先进的口型同步应用到视频）、Kling lipsync（音频转视频，以及带同步语音的文生视频）和 Creatify lipsync 之间路由。本 Skill 会根据用户的实际意图选择合适端点：静态肖像加音频（数字人模式）、源视频加音频（替换现有素材中的口型），或根据脚本生成并同步。当用户提到“lip sync”“lipsync”“make this video speak”“match audio to mouth”“dub video”“sync lips to voice”“Sync Labs”“voiceover sync”，或明确要求用一段音频驱动人脸口型时触发。

skills.sh 历史总榜第 127 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：lipsync。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh127_description$
),
(
    $zh128_slug$video-extend$zh128_slug$,
    'zh-CN',
    $zh128_name$视频续写$zh128_name$,
    $zh128_summary$通过 RunComfy CLI 延长或续写现有视频，使用 Google Veo 3-1 保持动作、光照与主体身份一致。$zh128_summary$,
    $zh128_description$通过 `runcomfy` CLI 在 RunComfy 上延长或续写现有视频片段。路由到 Google Veo 3-1 的 `extend-video` 或 `fast/extend-video` 端点：选择源视频，并用提示词描述接下来发生的内容，模型会生成在动作、光照和主体身份上与原片一致的续写片段。适用于用户希望延长一段较短的 Veo 视频，或从一个种子片段逐镜头串联叙事。当用户提到“extend video”“continue video”“longer video”“video extend”“make this clip longer”“Veo extend”“chain video shots”“video continuation”，或明确要求在现有视频之后增加更多画面时触发。

skills.sh 历史总榜第 128 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：video-extend。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh128_description$
),
(
    $zh129_slug$elevenlabs-music-generation$zh129_slug$,
    'zh-CN',
    $zh129_name$ElevenLabs 音乐生成$zh129_name$,
    $zh129_summary$通过 RunComfy CLI 使用 ElevenLabs Music 生成 5 秒至 5 分钟的完整歌曲或器乐，支持分段控制与多语言人声。$zh129_summary$,
    $zh129_description$通过 `runcomfy` CLI 在 RunComfy 上使用 ElevenLabs Music 生成完整歌曲和器乐。ElevenLabs Music 可将风格描述和结构化歌词转为录音室品质的 44.1 kHz 立体声音频，时长从 5 秒到 5 分钟，并支持段落级控制（Intro、Verse、Chorus、Bridge）、多语言人声和适合商业使用的输出。可以生成伴奏、完整人声歌曲、广告短曲、播客片头、游戏循环音乐或器乐铺底。通过本地 RunComfy CLI 调用 `runcomfy run elevenlabs/elevenlabs/music-generation`。当用户提到“generate music”“make a song”“AI music”“background music”“instrumental track”“ElevenLabs Music”“soundtrack”“jingle”“theme music”“royalty-free music”“compose”，或明确要求根据文字描述生成音乐或歌曲时触发。

skills.sh 历史总榜第 129 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：elevenlabs-music-generation。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh129_description$
),
(
    $zh130_slug$image-outpainting$zh130_slug$,
    'zh-CN',
    $zh130_name$图像扩图$zh130_name$,
    $zh130_summary$通过 RunComfy CLI 向原画布外扩展静态图像、补全画外内容或调整宽高比，同时保持原始内容。$zh130_summary$,
    $zh130_description$通过 `runcomfy` CLI 在 RunComfy 上进行图像扩图：把静态图扩展到原画布之外、补全相机未拍到的内容，或在保持原始内容的同时更改宽高比，例如从正方形变为 16:9、从竖图变为横图。可在 Nano Banana 2 Edit（默认方案，由空间语言驱动）、GPT Image 2 Edit（多参考图和参考风格匹配）、FLUX Kontext Pro（单次生成，最大限度保留原图）以及各品牌编辑端点（Seedream、Dreamina、Qwen、FLUX 2）之间路由。根据扩图是由自然语言、参考图还是锁定品牌风格驱动来选择方案。当用户提到“outpaint”“outpainting”“extend image canvas”“expand the image”“fill in around the photo”“uncrop”“change aspect ratio”“extend frame”“wide-screen from square”，或明确要求在现有静态图周围增加画布时触发。

skills.sh 历史总榜第 130 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：image-outpainting。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh130_description$
),
(
    $zh131_slug$relight$zh131_slug$,
    'zh-CN',
    $zh131_name$图像重打光$zh131_name$,
    $zh131_summary$通过 RunComfy CLI 改变静态图的布光、色温、方向或氛围，并按需求选择专用重打光或身份保持模型。$zh131_summary$,
    $zh131_description$通过 `runcomfy` CLI 在 RunComfy 上为静态图重新打光，改变布光方案、色温、方向或氛围。优先路由到 Qwen Edit 2509 专用的 `relight` LoRA 端点；当用自然语言描述光线即可满足需求时，则回退到能够保持主体身份的编辑端点（Nano Banana 2 Edit、GPT Image 2 Edit、FLUX Kontext Pro）。适用于产品重打光（摄影棚柔光箱变为窗光）、人像氛围调整（阴天变为黄金时刻）或调色变化。当用户提到“relight”“relighting”“change the lighting”“make it golden hour”“studio lighting”“rim light”“blue hour”“soft window light”“change light direction”“color temperature”，或明确要求改变静态图的光照方式时触发。

skills.sh 历史总榜第 131 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：relight。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh131_description$
),
(
    $zh132_slug$video-outpainting$zh132_slug$,
    'zh-CN',
    $zh132_name$视频扩图$zh132_name$,
    $zh132_summary$通过 RunComfy CLI 扩展视频的空间画布、转换横竖宽高比，并在保持中央动作的同时补充画外环境。$zh132_summary$,
    $zh132_description$通过 `runcomfy` CLI 在 RunComfy 上进行视频扩图：扩展视频的空间画布、更改宽高比（例如从 9:16 竖屏转为 16:9 横屏，或反向转换），并在保留中央动作的同时补充原画框之外的环境。通过 Wan 2-7 edit-video 进行由提示词塑造的空间扩展；当用于重点交付、接缝质量至关重要时，会引导代理使用专用 ComfyUI 扩图工作流。当用户提到“video outpaint”“video outpainting”“extend video canvas”“expand video frame”“uncrop video”“aspect ratio change”“vertical to horizontal video”“16:9 from 9:16”“TikTok to YouTube”，或明确要求把视频空间扩展到原画框之外时触发。

skills.sh 历史总榜第 132 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：video-outpainting。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh132_description$
),
(
    $zh133_slug$implement$zh133_slug$,
    'zh-CN',
    $zh133_name$按规范实现$zh133_name$,
    $zh133_summary$根据规格说明或一组任务票实现具体工作。$zh133_summary$,
    $zh133_description$根据规格说明或一组任务票实现具体工作。

skills.sh 历史总榜第 133 名快照。原始来源：mattpocock/skills；原始 slug：implement。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh133_description$
),
(
    $zh134_slug$ai-music$zh134_slug$,
    'zh-CN',
    $zh134_name$AI 音乐生成与编辑$zh134_name$,
    $zh134_summary$通过 RunComfy CLI 在音乐模型目录中智能路由，支持高品质或低成本音乐生成、歌曲局部重绘和前后续写。$zh134_summary$,
    $zh134_description$通过 `runcomfy` CLI 在 RunComfy 上生成 AI 音乐。本 Skill 是音乐模型目录的智能路由器，可路由到 ElevenLabs AI Music Generation（高品质 44.1 kHz 立体声人声曲目，5 秒至 5 分钟，每秒 0.0083 美元）和 ACE Step 或 ACE Step 1.5（StepFun-AI 开放权重、标签驱动作曲、多语言歌词，每秒 0.0002 至 0.0003 美元，约便宜 27 倍），以及 ACE Step audio-inpaint（重新生成现有曲目中的指定时间段）和 ACE Step audio-outpaint（从前面或后面延长曲目）。它会根据用户的实际意图选择合适模型，例如高品质人声 Hook、低成本背景音乐库、多语言流行歌曲、修复不理想的副歌，或把 30 秒草稿延长为 2 分钟版本，并提供各模型已记录的提示词模式和最小 `runcomfy run` 调用。当用户提到“generate music”“make a song”“AI music”“background music”“instrumental track”“soundtrack”“jingle”“theme music”“royalty-free music”“compose”“music with lyrics”“extend music”“fix this song”“inpaint music”，或明确要求生成或编辑音乐时触发。

skills.sh 历史总榜第 134 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：ai-music。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh134_description$
),
(
    $zh135_slug$ace-step$zh135_slug$,
    'zh-CN',
    $zh135_name$ACE Step 音乐生成与编辑$zh135_name$,
    $zh135_summary$通过 RunComfy CLI 使用 ACE Step 生成、局部重绘和续写音乐，支持标签驱动作曲、多语言歌词和最长 4 分钟立体声。$zh135_summary$,
    $zh135_description$通过 `runcomfy` CLI 在 RunComfy 上使用 ACE Step 生成、局部重绘和续写音乐。ACE Step 是 StepFun-AI 的开放权重音乐基础模型，支持由标签驱动的作曲（流派、情绪、乐器）、带段落标记的多语言歌词、5 秒至 4 分钟立体声输出，价格为每秒 0.0002 至 0.0003 美元（约比 ElevenLabs Music 便宜 27 倍）。提供四个端点：ACE Step text-to-audio（默认）、ACE Step 1.5 text-to-audio（支持 50 多种语言的歌词，并改进了结构化歌词处理）、ACE Step audio-inpaint（重新生成现有曲目中的指定时间段）以及 ACE Step audio-outpaint（从前面或后面延长现有曲目）。当用户提到“ace step”“ace-step”“acestep”“ACE music”“open music model”“cheap AI music”“inpaint audio”“audio inpaint”“extend music”“audio outpaint”“lengthen track”“music with tags”，或明确要求使用 ACE Step 生成或编辑音乐时触发。

skills.sh 历史总榜第 135 名快照。原始来源：prime-skills/runcomfy-agent-skills；原始 slug：ace-step。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh135_description$
),
(
    $zh136_slug$hyperframes-registry$zh136_slug$,
    'zh-CN',
    $zh136_name$HyperFrames 组件注册表$zh136_name$,
    $zh136_summary$在 HyperFrames 合成中发现、安装并接入注册表区块与组件，覆盖目录检索、安装位置、代码合并和上游贡献。$zh136_summary$,
    $zh136_description$在 HyperFrames 合成中安装、发现并接入注册表区块与组件。运行 `hyperframes add` 或 `hyperframes catalog`、安装单个条目或符合某个标签的全部区块、把已安装条目接入 `index.html`，或处理 `hyperframes.json` 时使用。覆盖条目发现、安装位置、区块子合成接线、组件代码片段合并，以及编写新块或新组件并贡献到上游的流程（创意 → 脚手架 → 验证 → PR）。

skills.sh 历史总榜第 136 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes-registry。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh136_description$
),
(
    $zh137_slug$brainstorming$zh137_slug$,
    'zh-CN',
    $zh137_name$创意工作前置构思$zh137_name$,
    $zh137_summary$在创建功能、构建组件、增加能力或修改行为之前，先探索用户意图、需求与设计。$zh137_summary$,
    $zh137_description$进行任何创意工作之前都必须使用，包括创建功能、构建组件、增加能力或修改行为。在实现之前探索用户意图、需求和设计。

skills.sh 历史总榜第 137 名快照。原始来源：obra/superpowers；原始 slug：brainstorming。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh137_description$
),
(
    $zh138_slug$lark-markdown-larksuite-cli$zh138_slug$,
    'zh-CN',
    $zh138_name$飞书 Markdown（larksuite/cli）$zh138_name$,
    $zh138_summary$查看、创建、上传、编辑和比较 Markdown 文件，支持读取、局部补丁和差异比较。$zh138_summary$,
    $zh138_description$飞书 Markdown：查看、创建、上传、编辑和比较 Markdown 文件。当用户需要创建或编辑 Markdown 文件、读取、修改、局部 patch 或比较差异时使用。不负责将 Markdown 导入为飞书在线文档，也不负责文件搜索、权限、评论、移动、删除等云空间管理操作。

skills.sh 历史总榜第 138 名快照。原始来源：larksuite/cli；原始 slug：lark-markdown。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh138_description$
),
(
    $zh139_slug$azure-cost$zh139_slug$,
    'zh-CN',
    $zh139_name$Azure 成本管理$zh139_name$,
    $zh139_summary$查询 Azure 成本、预测支出并通过资源优化减少浪费，不用于部署、预配、诊断或安全审计。$zh139_summary$,
    $zh139_description$Azure 成本管理：查询成本、预测支出，并通过优化减少浪费。适用于“Azure 成本”“Azure 账单”“成本明细”“我花了多少钱”“预测支出”“优化成本”“减少支出”“孤立资源”“调整 VM 规格”“成本激增”“降低存储成本”“AKS 成本”等请求。不要用于部署资源、预配、诊断或安全审计。

skills.sh 历史总榜第 139 名快照。原始来源：microsoft/azure-skills；原始 slug：azure-cost。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh139_description$
),
(
    $zh140_slug$writing-great-skills$zh140_slug$,
    'zh-CN',
    $zh140_name$编写高质量 Skill$zh140_name$,
    $zh140_summary$提供高质量编写与编辑 Skill 的术语和原则，使 Skill 的行为和触发更可预测。$zh140_summary$,
    $zh140_description$高质量编写和编辑 Skill 的参考资料，提供让 Skill 行为可预测的术语与原则。

skills.sh 历史总榜第 140 名快照。原始来源：mattpocock/skills；原始 slug：writing-great-skills。打包时排除了 0 个上游文件。该排行榜条目在仓库 HEAD 中不存在，已从历史提交 af6d6922c3e2b5288eef155346cbe319e4ed3bd0 恢复。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh140_description$
),
(
    $zh141_slug$ui-ux-pro-max$zh141_slug$,
    'zh-CN',
    $zh141_name$UI/UX Pro Max 设计智库$zh141_name$,
    $zh141_summary$面向 Web 与移动端的 UI/UX 设计智库，提供可检索的风格、配色、字体、产品类型、动效、图标与图表数据库。$zh141_summary$,
    $zh141_description$面向 Web 和移动端的 UI/UX 设计智库。可检索的本地数据库包含 84 种风格、192 套配色、74 组字体搭配、192 种产品类型、98 条 UX 指南、104 个图标条目、16 个 GSAP 动效预设，以及覆盖 22 种技术栈的 25 类图表：React、Next.js、Vue、Nuxt、Svelte、Astro、SwiftUI、React Native、Flutter、Tailwind、shadcn/ui、Jetpack Compose、Angular、Laravel、JavaFX、WPF、WinUI、Avalonia、Uno Platform、UWP、Three.js 和 HTML/CSS。设计、构建或审查 UI 时使用，包括页面、组件、配色方案、排版、布局、无障碍、动画或数据可视化。

skills.sh 历史总榜第 141 名快照。原始来源：nextlevelbuilder/ui-ux-pro-max-skill；原始 slug：ui-ux-pro-max。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh141_description$
),
(
    $zh142_slug$code-review$zh142_slug$,
    'zh-CN',
    $zh142_name$代码审查$zh142_name$,
    $zh142_summary$从指定提交、分支、标签或 merge-base 起，按代码规范和需求符合度两个维度并行审查变更。$zh142_summary$,
    $zh142_description$从一个固定点（commit、branch、tag 或 merge-base）开始，沿两个维度审查后续变更：规范维度检查代码是否遵循仓库记录的编码标准；规格维度检查代码是否符合来源 Issue 或规格的要求。两个审查会由子代理并行执行，并将结果并排报告。当用户希望审查分支、PR、进行中的变更，或要求“审查从 X 以来的变更”时使用。

skills.sh 历史总榜第 142 名快照。原始来源：mattpocock/skills；原始 slug：code-review。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh142_description$
),
(
    $zh143_slug$caveman-commit$zh143_slug$,
    'zh-CN',
    $zh143_name$Caveman 极简提交信息$zh143_name$,
    $zh143_summary$生成超精简的 Conventional Commits 提交信息，在保留意图和原因的同时去除噪声。$zh143_summary$,
    $zh143_description$超精简提交信息生成器。在保留意图和推理的同时去除提交信息中的噪声。采用 Conventional Commits 格式；主题不超过 50 个字符，仅在原因不明显时添加正文。当用户要求“编写提交信息”“commit message”“generate commit”“/commit”，或调用 `/caveman-commit` 时使用。暂存变更时自动触发。

skills.sh 历史总榜第 143 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman-commit。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh143_description$
),
(
    $zh144_slug$caveman-review$zh144_slug$,
    'zh-CN',
    $zh144_name$Caveman 极简代码审查$zh144_name$,
    $zh144_summary$生成超精简且可执行的代码审查意见，每条仅包含位置、问题和修复建议。$zh144_summary$,
    $zh144_description$生成超精简的代码审查意见。在保留可执行信号的同时去除 PR 反馈中的噪声。每条意见只有一行：位置、问题、修复。当用户要求“审查此 PR”“代码审查”“审查差异”“/review”，或调用 `/caveman-review` 时使用。审查 Pull Request 时自动触发。

skills.sh 历史总榜第 144 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman-review。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh144_description$
),
(
    $zh145_slug$caveman-compress$zh145_slug$,
    'zh-CN',
    $zh145_name$Caveman 上下文压缩$zh145_name$,
    $zh145_summary$将 CLAUDE.md、待办和偏好等自然语言记忆文件压缩为 Caveman 格式，以节省输入 Token。$zh145_summary$,
    $zh145_description$将自然语言记忆文件（CLAUDE.md、待办、偏好）压缩为 Caveman 格式，以节省输入 Token。保留所有技术实质、代码、URL 和结构。压缩版本会覆盖原文件，并将便于人类阅读的备份保存为 `FILE.original.md`。触发方式：`/caveman-compress FILEPATH` 或“压缩记忆文件”。

skills.sh 历史总榜第 145 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman-compress。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh145_description$
),
(
    $zh146_slug$lark-vc-agent-larksuite-cli$zh146_slug$,
    'zh-CN',
    $zh146_name$飞书视频会议会中能力（larksuite/cli）$zh146_name$,
    $zh146_summary$让应用机器人加入或离开进行中的飞书视频会议，并读取会中事件、发送文字消息或表情。$zh146_summary$,
    $zh146_description$飞书视频会议会中能力：用于让应用机器人真实加入或离开正在进行的会议，并读取当前身份可见的会中事件、发送会中文字消息或会中表情。适用于用户询问正在开的会议发生了什么、谁在发言、是否共享内容，或需要发现当前可读的进行中会议 ID。不负责已结束会议搜索、参会人快照、纪要、逐字稿或录制查询，这些使用 lark-vc Skill。

skills.sh 历史总榜第 146 名快照。原始来源：larksuite/cli；原始 slug：lark-vc-agent。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh146_description$
),
(
    $zh147_slug$caveman-help$zh147_slug$,
    'zh-CN',
    $zh147_name$Caveman 帮助速查$zh147_name$,
    $zh147_summary$展示 Caveman 所有模式、Skill 和命令的单次速查卡，不会进入持久模式。$zh147_summary$,
    $zh147_description$Caveman 所有模式、Skill 和命令的速查卡。仅展示一次，不会进入持久模式。触发方式：`/caveman-help`、“caveman help”“what caveman commands”或“how do I use caveman”。

skills.sh 历史总榜第 147 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman-help。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh147_description$
),
(
    $zh148_slug$vercel-composition-patterns$zh148_slug$,
    'zh-CN',
    $zh148_name$Vercel React 组合模式$zh148_name$,
    $zh148_summary$提供可扩展的 React 组合模式，用于重构布尔属性泛滥、构建灵活组件库和设计可复用 API。$zh148_summary$,
    $zh148_description$可扩展的 React 组合模式。在重构布尔属性泛滥的组件、构建灵活的组件库或设计可复用 API 时使用。涉及复合组件、render props、context provider 或组件架构的任务会触发本 Skill。包含 React 19 API 变更。

skills.sh 历史总榜第 148 名快照。原始来源：vercel-labs/agent-skills；原始 slug：vercel-composition-patterns。打包时排除了 0 个上游文件。本快照中未包含许可证文件；版权归原作者所有，使用须遵守源仓库的许可证。$zh148_description$
),
(
    $zh149_slug$wayfinder$zh149_slug$,
    'zh-CN',
    $zh149_name$大型工作路线图$zh149_name$,
    $zh149_summary$把超出单次代理会话容量的大型工作规划为共享的决策任务图，并逐一解决，直至通往目标的路径清晰。$zh149_summary$,
    $zh149_description$规划规模巨大、无法容纳在单次代理会话中的工作：在 Issue Tracker 上将其组织为共享的决策任务图，并逐一解决，直到通往目标的路径清晰。

skills.sh 历史总榜第 149 名快照。原始来源：mattpocock/skills；原始 slug：wayfinder。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh149_description$
),
(
    $zh150_slug$research$zh150_slug$,
    'zh-CN',
    $zh150_name$基于一手资料的研究$zh150_name$,
    $zh150_summary$依据高可信一手资料调查问题，并将研究发现保存为仓库中的 Markdown 文件。$zh150_summary$,
    $zh150_description$依据高可信一手资料调查问题，并将研究发现保存为仓库中的 Markdown 文件。当用户希望研究某个主题、收集文档或 API 事实，或把阅读调研工作委派给后台代理时使用。

skills.sh 历史总榜第 150 名快照。原始来源：mattpocock/skills；原始 slug：research。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh150_description$
),
(
    $zh151_slug$resolving-merge-conflicts$zh151_slug$,
    'zh-CN',
    $zh151_name$解决 Git 合并冲突$zh151_name$,
    $zh151_summary$解决正在进行的 Git merge 或 rebase 冲突。$zh151_summary$,
    $zh151_description$需要解决正在进行的 Git merge 或 rebase 冲突时使用。

skills.sh 历史总榜第 151 名快照。原始来源：mattpocock/skills；原始 slug：resolving-merge-conflicts。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh151_description$
),
(
    $zh152_slug$shadcn$zh152_slug$,
    'zh-CN',
    $zh152_name$shadcn 组件与项目管理$zh152_name$,
    $zh152_summary$管理 shadcn 组件与项目，支持添加、搜索、修复、调试、样式设计和 UI 组合，并提供文档与用法示例。$zh152_summary$,
    $zh152_description$管理 shadcn 组件和项目，包括添加、搜索、修复、调试、样式设计和组合 UI，也包括聊天界面。提供项目上下文、组件文档和用法示例。处理 shadcn/ui、组件注册表、预设、`--preset` 代码，或任何包含 `components.json` 文件的项目时使用。“shadcn init”“create an app with --preset”或“switch to --preset”等请求也会触发。

skills.sh 历史总榜第 152 名快照。原始来源：shadcn/ui；原始 slug：shadcn。打包时排除了 2 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh152_description$
),
(
    $zh153_slug$to-spec$zh153_slug$,
    'zh-CN',
    $zh153_name$从对话生成规格$zh153_name$,
    $zh153_summary$将当前对话直接综合为规格说明并发布到项目 Issue Tracker，无需额外访谈。$zh153_summary$,
    $zh153_description$将当前对话整理为规格说明，并发布到项目 Issue Tracker。无需访谈，只需综合已经讨论过的内容。

skills.sh 历史总榜第 153 名快照。原始来源：mattpocock/skills；原始 slug：to-spec。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh153_description$
),
(
    $zh154_slug$high-end-visual-design$zh154_slug$,
    'zh-CN',
    $zh154_name$高端视觉设计$zh154_name$,
    $zh154_summary$指导 AI 按高端设计机构的标准选择字体、间距、阴影、卡片结构和动画，避免廉价或泛化的默认设计。$zh154_summary$,
    $zh154_description$教 AI 像高端设计机构一样设计。明确规定让网站显得高级的字体、间距、阴影、卡片结构和动画，并阻止所有会让 AI 设计显得廉价或千篇一律的常见默认做法。

skills.sh 历史总榜第 154 名快照。原始来源：leonxlnx/taste-skill；原始 slug：high-end-visual-design。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh154_description$
),
(
    $zh155_slug$airunway-aks-setup$zh155_slug$,
    'zh-CN',
    $zh155_name$在 AKS 上部署 AI Runway$zh155_name$,
    $zh155_summary$从裸集群到模型运行，完成 AKS 集群验证、控制器安装、GPU 评估、提供商配置和首次部署。$zh155_summary$,
    $zh155_description$在 AKS 上设置 AI Runway，从裸集群一直到模型运行。覆盖集群验证、控制器安装、GPU 评估、提供商配置和首次部署。适用于“setup AI Runway”“onboard AKS cluster”“install AI Runway”“airunway setup”“deploy model to AKS”“GPU inference on AKS”“KAITO setup on AKS”“run LLM on AKS”“vLLM on AKS”“set up model serving on AKS”“AI Runway controller”等请求。

skills.sh 历史总榜第 155 名快照。原始来源：microsoft/azure-skills；原始 slug：airunway-aks-setup。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh155_description$
),
(
    $zh156_slug$to-tickets$zh156_slug$,
    'zh-CN',
    $zh156_name$方案拆分为任务票$zh156_name$,
    $zh156_summary$将计划、规格或当前对话拆分为示踪弹式任务票，声明各自的阻塞关系并发布到已配置的 Tracker。$zh156_summary$,
    $zh156_description$将计划、规格或当前对话拆分为一组示踪弹式任务票，每张票都声明其阻塞边，并发布到已配置的 Tracker：在本地，每张票使用一个文件并以文本记录边；在真实 Tracker 上，则使用原生阻塞链接。

skills.sh 历史总榜第 156 名快照。原始来源：mattpocock/skills；原始 slug：to-tickets。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh156_description$
),
(
    $zh157_slug$redesign-existing-projects$zh157_slug$,
    'zh-CN',
    $zh157_name$现有项目高端重设计$zh157_name$,
    $zh157_summary$审计并升级现有网站与应用，识别泛化的 AI 设计模式，在不破坏功能的前提下应用高端设计标准。$zh157_summary$,
    $zh157_description$把现有网站和应用升级到高端品质。审计当前设计，识别泛化的 AI 设计模式，并在不破坏功能的前提下应用高端设计标准。适用于任何 CSS 框架或原生 CSS。

skills.sh 历史总榜第 157 名快照。原始来源：leonxlnx/taste-skill；原始 slug：redesign-existing-projects。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh157_description$
),
(
    $zh158_slug$hyperframes-core$zh158_slug$,
    'zh-CN',
    $zh158_name$HyperFrames 核心规范$zh158_name$,
    $zh158_summary$定义 HyperFrames 可渲染项目的合成规范，覆盖时序属性、轨道、子合成、媒体播放、确定性渲染和验证。$zh158_summary$,
    $zh158_description$HyperFrames 合成规范，用于构建一个可渲染项目。处理合成结构、`data-*` 时序属性、`class="clip"`、轨道、子合成、变量、框架托管的媒体播放、确定性渲染规则和验证时使用。还覆盖 Tailwind 项目，以及 STORYBOARD.md 与 SCRIPT.md 计划格式。编写合成 HTML 前应先阅读。

skills.sh 历史总榜第 158 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes-core。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh158_description$
),
(
    $zh159_slug$hyperframes-animation$zh159_slug$,
    'zh-CN',
    $zh159_name$HyperFrames 动画$zh159_name$,
    $zh159_summary$汇总 HyperFrames 动画知识，包括原子动效、多阶段场景、转场、动效设计，以及七种运行时适配器。$zh159_summary$,
    $zh159_description$HyperFrames 的全部动画知识，包括原子动效规则、多阶段场景蓝图、场景转场、更广泛的动效设计技巧，以及七种运行时适配器：默认使用 GSAP，另有 Lottie、Three.js、Anime.js、CSS keyframes、Web Animations API 和 TypeGPU。用于任何运动或动画任务：选择 2 至 4 条规则进行组合、加载蓝图，或查阅特定运行时 API，例如 GSAP 缓动、Lottie 播放器或 Three.js mixer。还覆盖审计现有合成的动作编排（动画地图），以及 24 种具名文字动画效果。遵循 HyperFrames 原生规则：单条暂停时间线、可安全 seek、结果确定。

skills.sh 历史总榜第 159 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes-animation。打包时排除了 22 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh159_description$
),
(
    $zh160_slug$media-use$zh160_slug$,
    'zh-CN',
    $zh160_name$HyperFrames 媒体工作台$zh160_name$,
    $zh160_summary$作为 HyperFrames 项目的统一媒体入口，解析、生成、处理并复用音乐、音效、图像、图标、品牌素材、语音和调色资产。$zh160_summary$,
    $zh160_description$Agent Media OS 是 HyperFrames 项目中满足所有媒体需求的统一 Skill。把 BGM、SFX、图像、图标、品牌 Logo、语音、调色或 LUT 解析为冻结的本地文件，或可直接粘贴的代码块加台账记录，只需一个动词 `resolve`；目录缺少素材时，通过 TTS、音乐或图像模型生成；通过统一音频引擎制作旁白、转录、字幕和背景移除；对媒体执行剪切、重构画幅或变换；并在项目间复用资产。面对“实拍素材显得暗、平、无聊”“应该有复古、摄像机、印刷或 ASCII 感”“需要隐私处理”或“需要媒体揭示效果”等模糊反馈时也应使用。

skills.sh 历史总榜第 160 名快照。原始来源：heygen-com/hyperframes；原始 slug：media-use。打包时排除了 53 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh160_description$
),
(
    $zh161_slug$lark-apps-larksuite-cli$zh161_slug$,
    'zh-CN',
    $zh161_name$妙搭应用开发与托管（larksuite/cli）$zh161_name$,
    $zh161_summary$使用妙搭（Spark/Miaoda）创建、设计、开发、部署和托管应用，并管理集成、监控、权限、角色与自动化触发器。$zh161_summary$,
    $zh161_description$妙搭（Spark/Miaoda）应用开发与托管：应用创建、本地全栈开发、云端生成迭代、创意设计（UI mockup、可交互原型、线框图、落地页、仪表盘、幻灯片 deck、视觉探索）、AI 相关能力和飞书平台能力或其他外部能力集成、日志、Trace、监控指标、PV、UV 查询、环境变量管理、应用协作者与协作权限设置、应用角色与成员管理、自动化触发器（定时、记录变更、Webhook、飞书审批）。当用户要开发或新建一个系统、工具、平台或应用，要进行本地开发、云端开发、修改、部署、发布、上线或获取可分享链接，要用 HTML 制作页面或网站并部署到妙搭，要进行设计、design、mockup、prototype、wireframe、制作 PPT、deck 或视觉探索，或提到妙搭、Spark、Miaoda（应用运行时域名形如 `*.aiforce.cloud`）、应用数据库、应用文件存储、开放 API Key、可见范围、应用协作者或开发权限、应用角色或角色成员、线上日志、接口请求量、错误量、延迟、访问量、环境变量、为妙搭应用配置自动化任务、定时触发或审批通过后自动触发时使用。不负责普通云盘文件上传（lark-drive）、飞书文档编辑（lark-doc）或原生幻灯片创建（lark-slides）。

skills.sh 历史总榜第 161 名快照。原始来源：larksuite/cli；原始 slug：lark-apps。打包时排除了 1 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh161_description$
),
(
    $zh162_slug$cavecrew$zh162_slug$,
    'zh-CN',
    $zh162_name$CaveCrew 子代理委派$zh162_name$,
    $zh162_summary$指导何时把代码定位、小范围编辑或差异审查委派给 Caveman 风格子代理，并用压缩输出节省主上下文。$zh162_summary$,
    $zh162_description$把工作委派给 Caveman 风格子代理的决策指南。告诉主线程何时应派生 `cavecrew-investigator`（定位代码）、`cavecrew-builder`（编辑 1 至 2 个文件）或 `cavecrew-reviewer`（审查差异），而不是在线程内直接完成或使用普通的 `Explore`。子代理输出会经过 Caveman 压缩，因此注入主上下文的工具结果约小 60%，让主上下文在长会话中维持更久。触发方式：“delegate to subagent”“use cavecrew”“spawn investigator/builder/reviewer”“save context”“compressed agent output”。

skills.sh 历史总榜第 162 名快照。原始来源：juliusbrussee/caveman；原始 slug：cavecrew。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh162_description$
),
(
    $zh163_slug$caveman-stats$zh163_slug$,
    'zh-CN',
    $zh163_name$Caveman Token 统计$zh163_name$,
    $zh163_summary$直接读取 Claude Code 会话日志，显示当前会话的真实 Token 用量和预计节省量。$zh163_summary$,
    $zh163_description$显示当前会话的真实 Token 用量和预计节省量。直接读取 Claude Code 会话日志，不使用 AI 估算。由 `/caveman-stats` 触发。输出由 mode-tracker hook 注入，模型自身不计算这些数字。

skills.sh 历史总榜第 163 名快照。原始来源：juliusbrussee/caveman；原始 slug：caveman-stats。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh163_description$
),
(
    $zh164_slug$just-scrape$zh164_slug$,
    'zh-CN',
    $zh164_name$ScrapeGraph AI 网页抓取$zh164_name$,
    $zh164_summary$通过 ScrapeGraph AI CLI 搜索、抓取、爬取网页、提取结构化数据并监控页面变化。$zh164_summary$,
    $zh164_description$通过 ScrapeGraph AI CLI 搜索、抓取、爬取网页、提取结构化数据并监控页面。当用户要求搜索 Web、抓取网页、获取某个 URL 的内容、从网站提取 JSON、爬取文档或网站分区、监控页面变化、检查请求历史、查询 ScrapeGraph 额度，或验证 API 配置时使用。

skills.sh 历史总榜第 164 名快照。原始来源：scrapegraphai/just-scrape；原始 slug：just-scrape。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh164_description$
),
(
    $zh165_slug$minimalist-ui$zh165_slug$,
    'zh-CN',
    $zh165_name$极简主义 UI$zh165_name$,
    $zh165_summary$创建干净的编辑风界面，采用温暖单色、字体对比、扁平便当网格和柔和低饱和色，避免渐变与厚重阴影。$zh165_summary$,
    $zh165_description$干净的编辑风界面。采用温暖的单色调、字体对比、扁平便当网格和柔和低饱和色。不使用渐变，也不使用厚重阴影。

skills.sh 历史总榜第 165 名快照。原始来源：leonxlnx/taste-skill；原始 slug：minimalist-ui。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh165_description$
),
(
    $zh166_slug$hyperframes-creative$zh166_slug$,
    'zh-CN',
    $zh166_name$HyperFrames 创意指导$zh166_name$,
    $zh166_summary$为 HyperFrames 视频提供非动画类创意指导，覆盖设计规格、配色、排版、旁白、节拍、音频响应视觉和品牌风格。$zh166_summary$,
    $zh166_description$面向 HyperFrames 视频的非动画类创意指导。用于处理设计规格（frame.md、design.md）、配色、排版、旁白、节拍规划、音频响应视觉、合成模式，以及品牌或风格决策。原子动效模式和场景蓝图请使用 `hyperframes-animation`。

skills.sh 历史总榜第 166 名快照。原始来源：heygen-com/hyperframes；原始 slug：hyperframes-creative。打包时排除了 6 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh166_description$
),
(
    $zh167_slug$anti-ui-slop$zh167_slug$,
    'zh-CN',
    $zh167_name$反通用化 UI 设计$zh167_name$,
    $zh167_summary$阻止 AI 编码代理交付千篇一律的 UI，为设计、前端实现、重设计、审查和发布前打磨提供产品专属方向。$zh167_summary$,
    $zh167_description$阻止 AI 编码代理交付千篇一律的 UI。适用于 Web 或移动端 UI 设计、前端实现、重设计、UI 审查和发布前打磨，帮助 Codex、Claude Code、Cursor、Copilot 或其他代理获得符合具体产品的设计方向、完整交互状态和严格的完工门槛。

skills.sh 历史总榜第 167 名快照。原始来源：uizze.com；原始 slug：anti-ui-slop。打包时排除了 0 个上游文件。版权归原作者所有；使用和再分发须遵守随附或源仓库中的许可证。$zh167_description$
)
ON CONFLICT (slug, locale) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    summary = EXCLUDED.summary,
    description = EXCLUDED.description,
    updated_at = NOW();
