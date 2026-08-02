---
name: "Luoxue API Snow Clay"
description: "落雪 API 全产品唯一有效的紫色 Snow Clay 设计语言入口"
status: "active"
sourceOfTruth: ".superdesign/design-system.md"
runtimeTokens: "frontend/src/styles/luoxue-clay-tokens.css"
componentContracts: "frontend/src/styles/luoxue-clay-components.css"
visualGroundTruth:
  - "frontend/src/views/HomeView.clay.css"
  - "frontend/src/components/public/PublicSiteLayout.vue#public-site-page--snow"
  - "frontend/src/views/public/QuotaViewerLandingView.vue#/quota-viewer"
  - "frontend/src/views/public/QuotaViewerLandingView.css#/quota-viewer"
colors:
  canvas: "#F4F1FA"
  surface: "#FFFFFF"
  recessed: "#EFEBF5"
  text: "#332F3A"
  textSecondary: "#635F69"
  primaryViolet: "#7C3AED"
  primaryVioletDeep: "#5B21B6"
  brandIceBlue: "#0B8BED"
  semanticSuccess: "#047857"
---

# Luoxue API Snow Clay

本文件是供工具和协作者读取的根入口，不复制第二套设计规范。

- 完整设计规范以 [`.superdesign/design-system.md`](./.superdesign/design-system.md) 为唯一权威。
- 运行时色彩、字体、圆角与阴影以 `frontend/src/styles/luoxue-clay-tokens.css` 为唯一权威。
- 按钮、输入框、选择器、卡片、表格、分页与状态组件统一由 `frontend/src/styles/luoxue-clay-components.css` 应用语义 Token。
- 旧公共 `/home` 仍是整体视觉基准；已完成且最终视觉审阅为 `PASS / ship` 的公开 `/quota-viewer` 是 Snow Clay 的新视觉基准之一。登录、注册、管理端和用户端只是同一语言在不同信息密度下的应用。
- `PublicSiteLayout` 与 `AuthLayout` 默认进入 Snow Clay；`AppLayout` 默认提供 Snow Clay 画布和全高侧栏。桌面端不设置全局顶栏，页面语境由内容区的语义 `h1` 承担，账户、钱包、订阅、偏好与帮助统一收进左下角账户菜单；移动端仅保留用于打开导航的极简栏。`home-clay` 只负责管理端高密度内容适配，不能用于给用户页批量套管理端规则。

## 公开 `/quota-viewer` 可复用边界

- 首屏采用左侧价值主张、右侧产品证据的 split 构图；右侧以薰衣草色额度仪表作为主证据，不用通用插画替代。
- 健康、平稳、提醒与重置四种状态必须同时改变圆环和完整卡面氛围，不得只更换数字或局部颜色。
- 周重置只显示相对倒计时；月到期只显示日/月，不混用两套时间表达。
- 完整路径固定为五步：账户 → 安装 → 配对 → 桌面 → 续费。移动端五步必须全部直接可见，不折叠、不依赖横向滑动发现。
- 登录后由本站发起一次性安装包下载，不将 GitHub release 页作为下载中转或主行动落点。
- 主标题保持中文自然字距；桌面断点下稳定为两行，并且不得将“桌面”拆字断行。

## 不可变规则

1. 紫色负责主操作、当前选择和键盘焦点。
2. 冰蓝只用于落雪品牌、信息提示和少量数据强调，不得作为产品主交互色。
3. 绿色、琥珀色和红色分别承担成功、警告和危险语义。
4. `#14B8A6`、`#0D9488`、`#0F766E` 等旧青绿色主主题已经退役。
5. 产品自带界面文案通过 i18n 翻译；管理员或用户提供的自定义文案必须原样展示，不得因语言、主题或视觉迁移被翻译、改写或替换。
6. 页面可以调整信息密度和布局，但不得另建色板、字体体系、材质模型或组件词汇。
7. 每个认证页面必须在内容区提供明确的语义 `h1`；紧凑工作区可使用仅供辅助技术读取的标题，但不得依赖全局导航代替页面标题。
