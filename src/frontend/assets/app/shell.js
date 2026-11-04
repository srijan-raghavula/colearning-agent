import { api, esc, initials, state } from "./core.js";

export function renderShell(content, title) {
  document.title = `${title} · CoLearn`;
  document.documentElement.dataset.theme = state.theme;
  const student = state.user?.role === "student";
  const active = location.pathname.includes("workspace") ? "learn" : location.pathname.includes("profile") ? "profile" : "overview";
  const nav = state.user ? `<nav class="primary-nav" aria-label="Primary navigation"><a class="${active === "overview" ? "active" : ""}" href="${student ? "/app/student" : "/app/teacher"}"><span>⌂</span>Overview</a>${student ? `<a class="${active === "learn" ? "active" : ""}" href="/app/student/workspace"><span>◈</span>Learning</a>` : `<a class="${location.hash === "#course-setup" ? "active" : ""}" href="/app/teacher#course-setup"><span>▣</span>Course setup</a>`}<a class="${active === "profile" ? "active" : ""}" href="/app/profile"><span>◎</span>Profile</a></nav>` : "";
  return `<div class="app-shell"><header class="topbar"><a class="brand" href="/"><span class="brand-mark">C</span><span><b>CoLearn</b><small>Human + AI learning OS</small></span></a>${nav}<div class="account">${state.user ? `<button class="icon-button" type="button" data-action="theme" aria-label="Toggle theme">${state.theme === "dark" ? "☀" : "☾"}</button><span class="avatar">${initials(state.user.name)}</span><span class="account-name">${esc(state.user.name)}</span><button class="link-button" data-action="logout">Sign out</button>` : ""}</div></header><main class="page">${content}</main></div>`;
}

export function renderError(error) {
  return `<section class="error-state card"><span class="status-icon">!</span><p class="eyebrow">Workspace unavailable</p><h1>We could not load this view</h1><p>${esc(error.message)}</p><div class="error-actions"><button class="button primary" type="button" data-retry>Try again</button><a class="button secondary" href="/">Return home</a></div></section>`;
}

export function bindShell(app, route) {
  app.querySelector('[data-action="theme"]')?.addEventListener("click", () => {
    state.theme = state.theme === "dark" ? "light" : "dark";
    localStorage.setItem("colearn-theme", state.theme);
    route();
  });
  app.querySelector('[data-action="logout"]')?.addEventListener("click", async () => {
    try { await fetch("/logout", { credentials: "same-origin" }); } finally { location.assign("/"); }
  });
  app.querySelector("[data-retry]")?.addEventListener("click", route);
}
