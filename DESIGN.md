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
- 旧公共 `/home` 是视觉基准；登录、注册、管理端和用户端只是同一语言在不同信息密度下的应用。
- `PublicSiteLayout` 与 `AuthLayout` 默认进入 Snow Clay；`AppLayout` 默认提供 Snow Clay 画布、顶栏和侧栏，`home-clay` 只负责管理端高密度内容适配，不能用于给用户页批量套管理端规则。

## 不可变规则

1. 紫色负责主操作、当前选择和键盘焦点。
2. 冰蓝只用于落雪品牌、信息提示和少量数据强调，不得作为产品主交互色。
3. 绿色、琥珀色和红色分别承担成功、警告和危险语义。
4. `#14B8A6`、`#0D9488`、`#0F766E` 等旧青绿色主主题已经退役。
5. 产品自带界面文案通过 i18n 翻译；管理员或用户提供的自定义文案必须原样展示，不得因语言、主题或视觉迁移被翻译、改写或替换。
6. 页面可以调整信息密度和布局，但不得另建色板、字体体系、材质模型或组件词汇。
