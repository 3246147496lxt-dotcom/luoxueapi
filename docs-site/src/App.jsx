import { useEffect, useMemo, useRef, useState } from "react";
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
import { loadPublicBrandSettings, splitBrandApiSuffix } from "./brand-settings.js";
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

function usePublicBrand() {
  const [brand, setBrand] = useState(() => ({
    name: siteConfig.brandName,
    logo: siteConfig.logo,
  }));
  const latestBrand = useRef(brand);

  useEffect(() => {
    let disposed = false;
    let activeController = null;

    async function refreshBrand() {
      activeController?.abort();
      const controller = new AbortController();
      activeController = controller;

      const nextBrand = await loadPublicBrandSettings({
        fetchImpl: window.fetch.bind(window),
        // Keep the last known-good identity if a background refresh fails.
        fallback: latestBrand.current,
        signal: controller.signal,
      });

      if (!disposed && activeController === controller && !controller.signal.aborted) {
        latestBrand.current = nextBrand;
        setBrand(nextBrand);
      }
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "visible") void refreshBrand();
    }

    void refreshBrand();
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      disposed = true;
      activeController?.abort();
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, []);

  return brand;
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

function BrandWordmark({ name }) {
  const { base, apiSuffix } = splitBrandApiSuffix(name);

  return (
    <span className="brand-name" aria-hidden="true">
      {base ? <span>{base}</span> : null}
      {apiSuffix ? <span className="brand-name-api">{apiSuffix}</span> : null}
    </span>
  );
}

function Header({ authState, brand, theme, onToggleTheme, menuOpen, onToggleMenu, scrolled }) {
  return (
    <header className={`site-header ${scrolled ? "is-scrolled" : ""}`}>
      <div className="header-frame">
        <nav className="top-nav" aria-label="主导航">
          <a className="brand" href={siteConfig.mainSiteUrl} aria-label={brand.name}>
            <img src={brand.logo} alt="" />
            <BrandWordmark name={brand.name} />
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
            <button
              className="icon-button menu-button"
              type="button"
              onClick={onToggleMenu}
              aria-label="切换导航菜单"
              aria-expanded={menuOpen}
              aria-controls="mobile-navigation"
            >
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
    <div
      id="mobile-navigation"
      className={`mobile-menu ${open ? "open" : ""}`}
      aria-hidden={!open}
    >
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

function Step({ step, index, tutorialId }) {
  return (
    <li
      id={`${tutorialId}-step-${index + 1}`}
      className="tutorial-step"
      data-step-index={index}
    >
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
          <Step
            key={`${tutorial.id}-step-${index}`}
            step={step}
            index={index}
            tutorialId={tutorial.id}
          />
        ))}
      </ol>
    </section>
  );
}

function StepOutline({ tutorial, activeIndex, onSelect }) {
  return (
    <aside className="step-outline desktop-step-outline">
      <nav aria-label="本页导览">
        <p>本页导览</p>
        <ol>
          {tutorial.steps.map((step, index) => (
            <li key={`${tutorial.id}-outline-${index}`}>
              <button
                type="button"
                className={index === activeIndex ? "active" : ""}
                aria-current={index === activeIndex ? "step" : undefined}
                onClick={() => onSelect(index)}
                title={step.title}
              >
                <span>{index + 1}.</span>
                <span>{step.title}</span>
              </button>
            </li>
          ))}
        </ol>
      </nav>
    </aside>
  );
}

function MobileStepOutline({ tutorial, activeIndex, onSelect }) {
  return (
    <div className="mobile-step-outline">
      <label htmlFor="mobile-step-select">本页导览</label>
      <select
        id="mobile-step-select"
        aria-label="本页导览，当前步骤"
        value={activeIndex}
        onChange={(event) => onSelect(Number(event.target.value))}
      >
        {tutorial.steps.map((step, index) => (
          <option key={`${tutorial.id}-mobile-outline-${index}`} value={index}>
            {index + 1}. {step.title}
          </option>
        ))}
      </select>
    </div>
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
  const brand = usePublicBrand();
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
  const [activeStepIndex, setActiveStepIndex] = useState(0);

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
    document.title = `${siteConfig.pageTitle} · ${brand.name}`;

    let favicon = document.querySelector('link[rel="icon"]');
    if (!favicon) {
      favicon = document.createElement("link");
      favicon.rel = "icon";
      document.head.appendChild(favicon);
    }
    favicon.type = brand.logo.startsWith("data:image/svg+xml") || /\.svg(?:[?#].*)?$/iu.test(brand.logo)
      ? "image/svg+xml"
      : brand.logo.startsWith("data:image/png") || /\.png(?:[?#].*)?$/iu.test(brand.logo)
        ? "image/png"
        : "image/x-icon";
    favicon.href = brand.logo;
  }, [brand]);

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

  useEffect(() => {
    document.getElementById(`tab-${activeTutorial.id}`)?.scrollIntoView({
      behavior: "auto",
      block: "nearest",
      inline: "nearest",
    });
  }, [activeTutorial.id]);

  useEffect(() => {
    setActiveStepIndex(0);

    const stepElements = activeTutorial.steps
      .map((_, index) => document.getElementById(`${activeTutorial.id}-step-${index + 1}`))
      .filter(Boolean);
    if (stepElements.length === 0) return undefined;

    let frameId = 0;
    function syncActiveStep() {
      frameId = 0;
      const configuredGuideLine = Number.parseFloat(
        window.getComputedStyle(document.documentElement)
          .getPropertyValue("--docs-step-anchor-offset"),
      );
      const guideLine = Number.isFinite(configuredGuideLine) ? configuredGuideLine : 140;
      const atPageEnd = window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 4;
      let nextIndex = 0;

      stepElements.forEach((element, index) => {
        if (element.getBoundingClientRect().top <= guideLine) nextIndex = index;
      });

      if (atPageEnd) nextIndex = stepElements.length - 1;
      setActiveStepIndex(nextIndex);
    }

    function requestSync() {
      if (frameId) return;
      frameId = window.requestAnimationFrame(syncActiveStep);
    }

    syncActiveStep();
    window.addEventListener("scroll", requestSync, { passive: true });
    window.addEventListener("resize", requestSync);
    return () => {
      window.removeEventListener("scroll", requestSync);
      window.removeEventListener("resize", requestSync);
      if (frameId) window.cancelAnimationFrame(frameId);
    };
  }, [activeTutorial.id, activeTutorial.steps]);

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
    const nextTab = document.getElementById(`tab-${nextTutorial.id}`);
    nextTab?.focus();
    nextTab?.scrollIntoView({ behavior: "auto", block: "nearest", inline: "nearest" });
  }

  function scrollToStep(index) {
    const step = document.getElementById(`${activeTutorial.id}-step-${index + 1}`);
    if (!step) return;

    setActiveStepIndex(index);
    const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    step.scrollIntoView({ behavior: reduceMotion ? "auto" : "smooth", block: "start" });
  }

  return (
    <div className="app-shell">
      <Header
        authState={authState}
        brand={brand}
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

          <div className="reading-grid">
            <StepOutline
              tutorial={activeTutorial}
              activeIndex={activeStepIndex}
              onSelect={scrollToStep}
            />

            <div className="reading-column">
              <MobileStepOutline
                tutorial={activeTutorial}
                activeIndex={activeStepIndex}
                onSelect={scrollToStep}
              />
              <TutorialPanel tutorial={activeTutorial} />
            </div>

            <div className="reading-balance" aria-hidden="true" />
          </div>

          <footer className="docs-footer">
            <div>
              <img src={brand.logo} alt="" />
              <p>{brand.name} 新用户接入与使用指南。</p>
            </div>
            <SupportContact />
          </footer>
        </div>
      </main>
    </div>
  );
}
