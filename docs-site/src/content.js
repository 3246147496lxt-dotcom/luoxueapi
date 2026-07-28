export const PRODUCTION_MAIN_SITE_URL = "https://luoxueapi.cc";
export const DEVELOPMENT_MAIN_SITE_URL = "http://127.0.0.1:4178";

function normalizeMainSiteUrl(value) {
  if (typeof value !== "string" || !value.trim()) return "";

  try {
    const parsed = new URL(value.trim());
    if (
      !["http:", "https:"].includes(parsed.protocol)
      || parsed.username
      || parsed.password
      || parsed.search
      || parsed.hash
    ) {
      return "";
    }

    return parsed.toString().replace(/\/+$/, "");
  } catch {
    return "";
  }
}

export function resolveMainSiteUrl(env = {}) {
  const fallback = env?.DEV ? DEVELOPMENT_MAIN_SITE_URL : PRODUCTION_MAIN_SITE_URL;
  return normalizeMainSiteUrl(env?.VITE_MAIN_SITE_URL) || fallback;
}

const VITE_ENV = import.meta.env || {};
const MAIN_SITE_URL = resolveMainSiteUrl(VITE_ENV);
const DOCS_BASE_URL = VITE_ENV.BASE_URL || "/tutorial-docs/";

const siteLink = (path = "") => `${MAIN_SITE_URL}${path}`;
const docsAsset = (path) => `${DOCS_BASE_URL}${path.replace(/^\/+/, "")}`;
const API_BASE_URL = siteLink("/v1");

export const siteConfig = {
  brandName: "落雪API",
  logo: docsAsset("assets/brand-mark.svg"),
  pageTitle: "文档中心",
  mainSiteUrl: MAIN_SITE_URL,
  dashboardUrl: siteLink("/dashboard"),
  loginUrl: siteLink("/login"),
  profileUrl: siteLink("/profile"),
  supportContact: "客服 QQ：2456772148",
  supportValue: "2456772148",
  navigation: [
    { label: "首页", href: siteLink("/home"), external: true },
    { label: "控制台", href: siteLink("/dashboard"), external: true },
    { label: "API 密钥", href: siteLink("/keys"), external: true },
    { label: "文档", href: "#quick-start", active: true },
    { label: "个人中心", href: siteLink("/profile"), external: true },
  ],
};

export const tutorials = [
  {
    id: "quick-start",
    tabLabel: "新手教程",
    icon: "key",
    description: "创建 API 密钥，并将对应模型导入 CC Switch。",
    steps: [
      {
        title: "进入控制台",
        description: "进入仪表盘后，确认余额、订阅状态和常用入口是否正常显示。",
        image: {
          src: docsAsset("assets/dashboard-overview.png"),
          alt: "落雪API 控制台仪表盘概览",
        },
      },
      {
        title: "打开 API 密钥",
        description: "在仪表盘左侧导航中点击“API 密钥”。",
        image: {
          src: docsAsset("assets/api-key-navigation.png"),
          alt: "落雪API 仪表盘侧边栏中的 API 密钥入口",
        },
      },
      {
        title: "创建 API 密钥",
        description: "在 API 密钥页面点击“创建密钥”。",
        image: {
          src: docsAsset("assets/api-key-create.png"),
          alt: "API 密钥页面右上角的创建密钥按钮",
        },
      },
      {
        title: "填写名称并选择分组",
        description: "名称可以随意填写，按自己的使用需求选择对应分组，然后保存修改。",
        note: {
          tone: "warning",
          text: "分组一定要选，不能选择 default，也不能留空。",
        },
        notePlacement: "after-image",
        image: {
          src: docsAsset("assets/api-key-name-group.png"),
          alt: "创建密钥窗口中的名称与分组字段",
        },
      },
      {
        title: "将配置导出到 CC Switch",
        description: "回到 API 密钥列表，点击导入到CCS。",
        link: { label: "下载 CC Switch", href: "https://ccswitch.io/zh" },
      },
      {
        title: "点击导入",
        description: "确认导入信息无误后，点击“导入”。",
        image: {
          src: docsAsset("assets/cc-switch-confirm-import.png"),
          alt: "CC Switch 供应商配置确认窗口中的导入按钮",
        },
      },
      {
        title: "点击启用后重启Codex",
        description: "导入完成后，在 CC Switch 的供应商列表中点击“启用”，再重启 Codex，让新配置生效。",
        image: {
          src: docsAsset("assets/cc-switch-enable-codex.png"),
          alt: "CC Switch 供应商列表中的启用按钮",
        },
      },
    ],
  },
  {
    id: "desktop",
    tabLabel: "Desktop 客户端",
    icon: "terminal",
    description: "在 macOS 上安装落雪API Desktop，完成网页配对，并让 Codex 通过本地安全网关使用所选线路。",
    steps: [
      {
        title: "下载 macOS 安装包",
        description: "下载适用于 macOS 13 及以上系统的 Universal DMG；同一个安装包同时支持 Apple Silicon 与 Intel Mac。",
        link: {
          label: "下载落雪API Desktop",
          href: siteLink("/api/v1/public/desktop/releases/macos/latest/download"),
        },
        note: {
          tone: "warning",
          text: "当前安装包没有 Apple 公证，首次打开时 macOS 会显示安全拦截，这是预期行为。请只从落雪API官方地址下载。",
        },
      },
      {
        title: "安装到应用程序",
        description: "打开下载完成的 DMG，将“落雪API Desktop”拖入“应用程序”文件夹，再从“应用程序”中打开。",
      },
      {
        title: "允许首次打开",
        description: "如 macOS 阻止打开，请进入“系统设置 → 隐私与安全”，在页面下方找到落雪API Desktop 的拦截提示，点击“仍要打开”，再按系统提示确认。",
        note: {
          text: "不要关闭系统的整体安全检查，也不要执行来源不明的终端命令。只需要为本次下载的落雪API Desktop 单独授权。",
        },
      },
      {
        title: "在网页批准设备",
        description: "App 会打开落雪API登录页。登录后核对设备名称和申请权限并批准；配对完成后凭据只保存在这台 Mac 的钥匙串中。",
        note: {
          text: "每个账号最多绑定 5 台设备。Desktop 不会读取或保存你的网页登录密码。",
        },
      },
      {
        title: "选择线路和默认模型",
        description: "回到 App，选择一个可用的 OpenAI 分组及默认模型并启用接管。若余额不足、没有可用分组、配置无法解析或检测到 CC Switch 接管，App 会停止修改并给出处理入口。",
      },
      {
        title: "重启 Codex 并验证",
        description: "按 App 提示重启 Codex Desktop，或复制 CLI 启动命令。发送一条简短消息；首个 Responses 请求成功后，Desktop 总览会显示网关运行状态和本机用量。关闭主窗口后 App 会继续在菜单栏运行。",
        note: {
          text: "卸载前请先在 Desktop 设置中执行“恢复并准备卸载”，确保 Codex 配置和开机启动均已恢复。",
        },
      },
    ],
  },
  {
    id: "clients",
    tabLabel: "客户端配置",
    icon: "client",
    description: "站内会根据密钥分组生成对应配置，下面只说明各类客户端的地址规则。",
    steps: [
      {
        title: "先确认密钥所属分组",
        description: "打开 API 密钥页面并检查分组。没有分组的密钥无法生成正确配置，也不能正常调用模型。",
        link: { label: "打开 API 密钥", href: siteLink("/keys") },
      },
      {
        title: "OpenAI Compatible 客户端",
        description: "Cherry Studio、Chatbox 等兼容客户端通常需要填写带 /v1 的 Base URL，再选择 GET /v1/models 返回的模型。",
        code: {
          label: "OpenAI Compatible",
          value: `Base URL：${API_BASE_URL}
API Key：sk-your-api-key
模型：从 /v1/models 返回结果中选择`,
        },
      },
      {
        title: "Claude Code、Codex 与 Gemini CLI",
        description: "这些 CLI 使用的变量和配置文件不同，且地址通常不需要手动追加 /v1。请在“使用密钥”窗口选择对应客户端和操作系统后复制。",
        code: {
          label: "地址规则",
          value: `Claude Code  ANTHROPIC_BASE_URL=${MAIN_SITE_URL}
Codex CLI   base_url = "${MAIN_SITE_URL}"
Gemini CLI  GOOGLE_GEMINI_BASE_URL=${MAIN_SITE_URL}`,
        },
        note: {
          text: "如已有 ~/.claude、~/.codex 或其他客户端配置，请先备份再合并，避免覆盖原有服务商和模型设置。",
        },
      },
      {
        title: "CC Switch 与 OpenCode",
        description: "CC Switch 可在密钥页使用一键导入；OpenCode 可从“使用密钥”窗口复制对应 provider 配置。完成后先发送一条简短消息验证。",
      },
    ],
  },
  {
    id: "api",
    tabLabel: "API 接入",
    icon: "api",
    description: "根据密钥分组选择相应协议，并始终通过请求头传递密钥。",
    steps: [
      {
        title: "确认协议与端点",
        description: "同一个站点支持多种兼容协议，但具体可用端点取决于密钥分组和管理员配置。",
        code: {
          label: "常用端点",
          value: `GET  ${API_BASE_URL}/models
POST ${API_BASE_URL}/chat/completions
POST ${API_BASE_URL}/responses
POST ${API_BASE_URL}/messages
GET  ${MAIN_SITE_URL}/v1beta/models
GET  ${API_BASE_URL}/usage`,
        },
      },
      {
        title: "发送 OpenAI Compatible 测试请求",
        description: "以下请求适用于支持 Chat Completions 的分组。请把模型名替换为当前密钥实际可用的模型。",
        code: {
          label: "cURL",
          value: `curl ${API_BASE_URL}/chat/completions \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "从 /v1/models 中选择",
    "messages": [{"role": "user", "content": "请回复 OK"}]
  }'`,
        },
      },
      {
        title: "根据状态码排查",
        description: "联系支持前，请记录请求时间、端点、模型名和状态码，但不要发送完整密钥。",
        code: {
          label: "常见状态码",
          value: `401  密钥无效、已停用，或认证头填写错误
403  余额不足、密钥过期、无有效订阅，或分组/IP 权限受限
404  当前分组不支持该模型或端点
429  触发密钥额度、订阅、速率或并发限制
503  当前没有可用的上游账号`,
        },
      },
      {
        title: "查看用量记录",
        description: "请求成功后可在“使用记录”中核对模型、Token、费用和状态；异常时再携带必要信息联系客服。",
        link: { label: "查看使用记录", href: siteLink("/usage") },
      },
    ],
  },
  {
    id: "recharge",
    tabLabel: "充值帮助",
    icon: "wallet",
    description: "充值方式、手续费和到账倍率以支付页面当时显示为准。",
    steps: [
      {
        title: "选择充值或兑换",
        description: "如已开通在线支付，可进入“充值/订阅”；持有兑换码时可直接在兑换页面使用。",
        link: { label: "打开充值/订阅", href: siteLink("/purchase") },
        note: {
          text: "若充值入口没有显示，通常表示管理员暂未开启支付功能；此时可使用兑换码或联系客服。",
        },
      },
      {
        title: "付款前核对订单",
        description: "确认充值账户、到账金额、支付金额、手续费和支付方式后再付款。不要向站外提供的个人账户转账。",
        note: {
          tone: "warning",
          text: "支付方式和费率可能动态调整，静态教程不会承诺固定最低金额、手续费或到账倍率。",
        },
      },
      {
        title: "确认到账状态",
        description: "支付后等待订单状态变为“已完成”，再到仪表盘确认余额或订阅。长时间未到账时请保留订单号。",
        link: { label: "查看我的订单", href: siteLink("/orders") },
      },
      {
        title: "联系支持",
        description: "反馈问题时请提供订单号、支付时间、金额和页面状态，不要发送密码或完整 API 密钥。",
        code: {
          label: "客服联系方式",
          value: "QQ：2456772148",
        },
      },
    ],
  },
];
