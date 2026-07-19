import { useEffect, useMemo, useState } from "react";
import {
  BookOpen,
  Boxes,
  ChevronRight,
  CircleHelp,
  Check,
  Code2,
  Copy,
  ExternalLink,
  Image as ImageIcon,
  KeyRound,
  Menu,
  Moon,
  MonitorCog,
  Sun,
  Terminal,
  WalletCards,
  X,
} from "lucide-react";
import { readStoredAuth, verifyAuthState } from "./auth-state.js";
import { siteConfig, tutorials } from "./content.js";
import {
  loadPublishedDocumentation,
  resolveActiveTutorialId,
  subscribeToTutorialHashChanges,
} from "./documentation.js";

const AUTH_STORAGE_KEYS = new Set(["auth_token", "auth_user"]);

const tutorialIcons = {
  key: KeyRound,
  client: MonitorCog,
  api: Code2,
  wallet: WalletCards,
  image: ImageIcon,
  help: CircleHelp,
  book: BookOpen,
  code: Code2,
  terminal: Terminal,
};

function svgIconDataUrl(svg) {
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

function TutorialTabIcon({ tutorial }) {
  const [customFailed, setCustomFailed] = useState(false);
  const FallbackIcon = tutorialIcons[tutorial.icon] || Boxes;

  useEffect(() => {
    setCustomFailed(false);
  }, [tutorial.iconSvg]);

  if (tutorial.iconSvg && !customFailed) {
    return (
      <img
        className="tutorial-tab-icon"
        src={svgIconDataUrl(tutorial.iconSvg)}
        alt=""
        aria-hidden="true"
        onError={() => setCustomFailed(true)}
      />
    );
  }
  return <FallbackIcon size={16} aria-hidden="true" />;
}

function getInitialTheme() {
  const saved = window.localStorage.getItem("docs-theme");
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function useDocumentAuth() {
  const [authState, setAuthState] = useState(() => readStoredAuth(window.localStorage));

  useEffect(() => {
    let disposed = false;
    let activeController = null;
    let syncVersion = 0;

    async function syncAuthState() {
      activeController?.abort();
      const version = ++syncVersion;
      const controller = new AbortController();
      activeController = controller;

      setAuthState(readStoredAuth(window.localStorage));

      const verifiedState = await verifyAuthState({
        storage: window.localStorage,
        fetchImpl: window.fetch.bind(window),
        timeoutMs: 5000,
        signal: controller.signal,
      });

      if (!disposed && version === syncVersion) {
        setAuthState(verifiedState);
      }
    }

    function handleStorage(event) {
      if (event.storageArea === window.localStorage && AUTH_STORAGE_KEYS.has(event.key)) {
        void syncAuthState();
      }
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        void syncAuthState();
      }
    }

    void syncAuthState();
    window.addEventListener("storage", handleStorage);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      disposed = true;
      activeController?.abort();
      window.removeEventListener("storage", handleStorage);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, []);

  return authState;
}

function getUserLabel(user) {
  const username = typeof user?.username === "string" ? user.username.trim() : "";
  if (username) return username;

  const email = typeof user?.email === "string" ? user.email.trim() : "";
  return email.split("@")[0] || "已登录用户";
}

function getUserInitials(user) {
  return Array.from(getUserLabel(user)).slice(0, 2).join("").toUpperCase();
}

function getSafeAvatarUrl(user) {
  const value = typeof user?.avatar_url === "string" ? user.avatar_url.trim() : "";
  if (!value) return "";

  try {
    const url = new URL(value, window.location.origin);
    return url.protocol === "http:" || url.protocol === "https:" ? url.href : "";
  } catch {
    return "";
  }
}

function UserAvatar({ user }) {
  const avatarUrl = getSafeAvatarUrl(user);
  const label = getUserLabel(user);

  return (
    <span className="user-avatar" aria-hidden="true">
      {avatarUrl ? <img src={avatarUrl} alt="" referrerPolicy="no-referrer" /> : getUserInitials(user)}
    </span>
  );
}

function AccountActions({ authState, mobile = false, onNavigate }) {
  const authenticated = authState.status === "authenticated" && authState.user;
  const guest = authState.status === "guest";
  const primaryHref = guest ? siteConfig.loginUrl : siteConfig.dashboardUrl;
  const primaryLabel = guest ? "登录" : "进入控制台";

  if (mobile) {
    return (
      <div className="mobile-account" aria-live="polite">
        {authenticated && (
          <a className="mobile-profile" href={siteConfig.profileUrl} onClick={onNavigate}>
            <UserAvatar user={authState.user} />
            <span>{getUserLabel(authState.user)}</span>
            <ChevronRight size={16} />
          </a>
        )}
        <a className="mobile-console" href={primaryHref} onClick={onNavigate}>{primaryLabel}</a>
      </div>
    );
  }

  return (
    <div className="account-actions" aria-live="polite">
      <a className="console-button" href={primaryHref}>{primaryLabel}</a>
      {authenticated && (
        <a
          className="user-chip"
          href={siteConfig.profileUrl}
          aria-label={`个人中心：${getUserLabel(authState.user)}`}
          title={getUserLabel(authState.user)}
        >
          <UserAvatar user={authState.user} />
          <span className="user-name">{getUserLabel(authState.user)}</span>
        </a>
      )}
    </div>
  );
}

function CodeBlock({ children, label }) {
  const [copied, setCopied] = useState(false);

  async function copyCode() {
    await navigator.clipboard.writeText(children.trim());
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div className="code-block">
      <div className="code-toolbar">
        <span>{label || "示例"}</span>
        <button type="button" onClick={copyCode}>{copied ? "已复制" : "复制"}</button>
      </div>
      <pre><code>{children.trim()}</code></pre>
    </div>
  );
}

function StepNote({ note }) {
  return (
    <aside className={`note ${note.tone || "info"}`}>
      <CircleHelp size={17} />
      <span>{note.text}</span>
    </aside>
  );
}

function Header({ authState, theme, onToggleTheme, menuOpen, onToggleMenu, scrolled }) {
  return (
    <header className={`site-header ${scrolled ? "is-scrolled" : ""}`}>
      <div className="header-frame">
        <nav className="top-nav" aria-label="主导航">
          <a className="brand" href={siteConfig.mainSiteUrl}>
            <img src={siteConfig.logo} alt="" />
            <span>{siteConfig.brandName}</span>
          </a>

          <div className="desktop-nav">
            {siteConfig.navigation.map((item) => (
              <a
                key={item.label}
                href={item.href}
                className={item.active ? "active" : ""}
                target={item.external ? "_blank" : undefined}
                rel={item.external ? "noreferrer" : undefined}
              >
                {item.label}
              </a>
            ))}
            <span className="nav-divider" aria-hidden="true" />
            <button className="icon-button" type="button" onClick={onToggleTheme} aria-label="切换主题">
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </button>
            <AccountActions authState={authState} />
          </div>

          <div className="mobile-actions">
            <button className="icon-button" type="button" onClick={onToggleTheme} aria-label="切换主题">
              {theme === "dark" ? <Sun size={19} /> : <Moon size={19} />}
            </button>
            <button className="icon-button menu-button" type="button" onClick={onToggleMenu} aria-label="切换导航菜单">
              {menuOpen ? <X size={22} /> : <Menu size={22} />}
            </button>
          </div>
        </nav>
      </div>
    </header>
  );
}

function MobileMenu({ authState, open, onNavigate }) {
  return (
    <div className={`mobile-menu ${open ? "open" : ""}`} aria-hidden={!open}>
      <nav aria-label="手机导航">
        {siteConfig.navigation.map((item, index) => (
          <a
            key={item.label}
            href={item.href}
            target={item.external ? "_blank" : undefined}
            rel={item.external ? "noreferrer" : undefined}
            onClick={onNavigate}
            style={{ transitionDelay: open ? `${index * 42}ms` : "0ms" }}
          >
            {item.label}
            <ChevronRight size={16} />
          </a>
        ))}
      </nav>
      <AccountActions authState={authState} mobile onNavigate={onNavigate} />
    </div>
  );
}

function Step({ step, index }) {
  return (
    <li className="tutorial-step">
      <div className="step-number" aria-hidden="true">{index + 1}</div>
      <div className="step-body">
        <h2>{step.title}</h2>
        <p>{step.description}</p>

        {step.link && (
          <a className="inline-link" href={step.link.href} target="_blank" rel="noreferrer">
            {step.link.label}<ExternalLink size={14} />
          </a>
        )}

        {step.note && step.notePlacement !== "after-image" && <StepNote note={step.note} />}

        {step.code && <CodeBlock label={step.code.label}>{step.code.value}</CodeBlock>}

        {step.image && (
          <figure className="step-image">
            <img src={step.image.src} alt={step.image.alt} />
            {step.image.caption && <figcaption>{step.image.caption}</figcaption>}
          </figure>
        )}

        {step.note && step.notePlacement === "after-image" && <StepNote note={step.note} />}
      </div>
    </li>
  );
}

function TutorialPanel({ tutorial }) {
  return (
    <section
      id={`panel-${tutorial.id}`}
      className="tutorial-panel"
      role="tabpanel"
      aria-labelledby={`tab-${tutorial.id}`}
      tabIndex={0}
    >
      <div className="tutorial-intro">
        <p>{tutorial.description}</p>
      </div>
      <ol className="step-list">
        {tutorial.steps.map((step, index) => (
          <Step key={`${tutorial.id}-${step.title}`} step={step} index={index} />
        ))}
      </ol>
    </section>
  );
}

function SupportContact() {
  const [copied, setCopied] = useState(false);

  async function copyContact() {
    await navigator.clipboard.writeText(siteConfig.supportValue);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <button className="footer-support" type="button" onClick={copyContact}>
      {copied ? <Check size={14} /> : <Copy size={14} />}
      {copied ? "客服 QQ 已复制" : siteConfig.supportContact}
    </button>
  );
}

export function App() {
  const authState = useDocumentAuth();
  const [documentation, setDocumentation] = useState(() => ({
    tutorials,
    source: "bundled",
    version: null,
    publishedAt: null,
    refreshComplete: false,
  }));
  const visibleTutorials = documentation.tutorials;
  const [theme, setTheme] = useState(getInitialTheme);
  const [activeId, setActiveId] = useState(() => window.location.hash.replace(/^#/, ""));
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  const activeTutorial = useMemo(
    () => visibleTutorials.find((tutorial) => tutorial.id === activeId) || visibleTutorials[0],
    [activeId, visibleTutorials],
  );

  useEffect(() => {
    const controller = new AbortController();

    void loadPublishedDocumentation({
      fetchImpl: window.fetch.bind(window),
      fallbackTutorials: tutorials,
      signal: controller.signal,
    }).then((result) => {
      if (!controller.signal.aborted) {
        setDocumentation({ ...result, refreshComplete: true });
      }
    });

    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!documentation.refreshComplete) return;

    const nextId = resolveActiveTutorialId(visibleTutorials, window.location.hash);
    if (activeId !== nextId) setActiveId(nextId);
    if (window.location.hash !== `#${nextId}`) {
      window.history.replaceState(null, "", `#${nextId}`);
    }
  }, [activeId, documentation.refreshComplete, visibleTutorials]);

  useEffect(() => subscribeToTutorialHashChanges({
    target: window,
    tutorialList: visibleTutorials,
    onChange: setActiveId,
  }), [visibleTutorials]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem("docs-theme", theme);
  }, [theme]);

  useEffect(() => {
    function handleScroll() {
      setScrolled(window.scrollY > 18);
    }
    handleScroll();
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    document.body.classList.toggle("menu-open", menuOpen);
    return () => document.body.classList.remove("menu-open");
  }, [menuOpen]);

  function selectTutorial(id) {
    setActiveId(id);
    window.history.replaceState(null, "", `#${id}`);
  }

  function handleTabKeyDown(event, index) {
    const keys = ["ArrowLeft", "ArrowRight", "Home", "End"];
    if (!keys.includes(event.key)) return;

    event.preventDefault();
    let nextIndex = index;
    if (event.key === "ArrowLeft") nextIndex = (index - 1 + visibleTutorials.length) % visibleTutorials.length;
    if (event.key === "ArrowRight") nextIndex = (index + 1) % visibleTutorials.length;
    if (event.key === "Home") nextIndex = 0;
    if (event.key === "End") nextIndex = visibleTutorials.length - 1;

    const nextTutorial = visibleTutorials[nextIndex];
    selectTutorial(nextTutorial.id);
    document.getElementById(`tab-${nextTutorial.id}`)?.focus();
  }

  return (
    <div className="app-shell">
      <Header
        authState={authState}
        theme={theme}
        onToggleTheme={() => setTheme(theme === "dark" ? "light" : "dark")}
        menuOpen={menuOpen}
        onToggleMenu={() => setMenuOpen((value) => !value)}
        scrolled={scrolled}
      />
      <MobileMenu authState={authState} open={menuOpen} onNavigate={() => setMenuOpen(false)} />

      <main className="docs-main">
        <div className="docs-container">
          <section className="page-heading">
            <h1>{siteConfig.pageTitle}</h1>
          </section>

          <div className="tabs" role="tablist" aria-label="教程分类">
            {visibleTutorials.map((tutorial, index) => {
              const selected = tutorial.id === activeTutorial.id;
              return (
                <button
                  key={tutorial.id}
                  type="button"
                  role="tab"
                  id={`tab-${tutorial.id}`}
                  aria-selected={selected}
                  aria-controls={`panel-${tutorial.id}`}
                  tabIndex={selected ? 0 : -1}
                  className={selected ? "selected" : ""}
                  onClick={() => selectTutorial(tutorial.id)}
                  onKeyDown={(event) => handleTabKeyDown(event, index)}
                >
                  <TutorialTabIcon tutorial={tutorial} />
                  <span>{tutorial.tabLabel}</span>
                </button>
              );
            })}
          </div>

          <TutorialPanel tutorial={activeTutorial} />

          <footer className="docs-footer">
            <div>
              <img src={siteConfig.logo} alt="" />
              <p>{siteConfig.footerText}</p>
            </div>
            <SupportContact />
          </footer>
        </div>
      </main>
    </div>
  );
}
