const app = document.querySelector("#app");
const state = { user: null, dashboard: null, session: null, turns: [], loading: false };

const esc = (value = "") => String(value).replace(/[&<>"']/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char]));
const initials = name => name.split(/\s+/).filter(Boolean).map(part => part[0]).slice(0, 2).join("").toUpperCase() || "?";
const modeLabel = mode => ({ guided: "Guided", socratic: "Socratic", direct: "Direct", practice: "Practice", hint: "Hint", diagnostic: "Diagnostic" }[mode] || mode);

async function api(path, options = {}) {
  const response = await fetch(path, { credentials: "same-origin", headers: { "Content-Type": "application/json", Accept: "application/json", ...(options.headers || {}) }, ...options });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || `Request failed (${response.status})`);
  return body;
}

function shell(content, title = "CoLearn") {
  document.title = `${title} · CoLearn`;
  app.innerHTML = `<div class="app-shell">
    <header class="topbar"><a class="brand" href="/"><span class="brand-mark">C</span><span><b>CoLearn</b><small>Human + AI learning</small></span></a>
      ${state.user ? `<nav><a href="/app/student">Overview</a><a href="/app/student/workspace">Learning space</a>${state.user.role === "teacher" ? `<a href="/app/teacher">Teacher dashboard</a>` : ""}</nav>
      <div class="account"><span class="avatar">${initials(state.user.name)}</span><span>${esc(state.user.name)}</span><button class="link-button" data-action="logout">Sign out</button></div>` : `<div class="top-actions"><a class="button secondary" href="/login">Sign in</a><a class="button primary" href="/signup">Get started</a></div>`}
    </header><main class="page">${content}</main>
  </div>`;
  app.querySelector('[data-action="logout"]')?.addEventListener("click", async () => { await fetch("/logout", { credentials: "same-origin" }); location.assign("/"); });
}

function landing() {
  shell(`<section class="hero"><div><p class="eyebrow">Learning that keeps going</p><h1>Understand more.<br><em>Keep moving.</em></h1><p class="lede">CoLearn pairs a teacher’s curriculum with a configurable AI tutor. Ask questions, practice at your pace, and see your understanding grow over time.</p><div class="actions"><a class="button primary" href="/signup">Open your workspace →</a><a class="button secondary" href="/login">Sign in</a></div><p class="trust">✓ Teacher-approved context · adaptive tutor modes · formative checks</p></div><div class="hero-card"><div class="mock-top"><i></i><i></i><i></i></div><div class="mock-body"><p class="eyebrow">YOUR LEARNING PATH</p><h2>One small step at a time</h2><div class="mock-item"><b>Linear equations</b><span>68% confidence</span></div><div class="mock-item"><b>Graphing equations</b><span>Needs support</span></div></div></div></section><section class="features"><article><b>01 · Teacher-led</b><span>Concepts and source material stay grounded in what your teacher provides.</span></article><article><b>02 · Adaptive</b><span>Choose a tutor mode that matches how you want to learn.</span></article><article><b>03 · Continuous</b><span>Sessions and evidence stay connected between visits.</span></article></section>`, "Learn with CoLearn");
}

function authPage(signup = false) {
  shell(`<section class="auth-card"><span class="brand-mark">C</span><p class="eyebrow">${signup ? "Start learning" : "Welcome back"}</p><h1>${signup ? "Create your workspace" : "Sign in to CoLearn"}</h1><p class="muted">${signup ? "Join a teacher-grounded learning space." : "Continue where your learning left off."}</p><form id="auth-form" class="form-grid">${signup ? `<label>Name<input name="name" required maxlength="100" autocomplete="name"></label>` : ""}<label>Email<input name="email" type="email" required autocomplete="username"></label><label>Password<input name="password" type="password" required minlength="8" maxlength="72" autocomplete="${signup ? "new-password" : "current-password"}"></label>${signup ? `<label>Role<select name="role"><option value="student">Student</option><option value="teacher">Teacher</option></select></label>` : ""}<p class="form-error" id="auth-error"></p><button class="button primary full" type="submit">${signup ? "Create account" : "Sign in"}</button></form><p class="auth-switch">${signup ? "Already have an account?" : "New to CoLearn?"} <a href="${signup ? "/login" : "/signup"}">${signup ? "Sign in" : "Create one"}</a></p></section>`, signup ? "Create account" : "Sign in");
  document.querySelector("#auth-form").addEventListener("submit", async event => {
    event.preventDefault();
    const form = event.currentTarget, button = form.querySelector("button"), error = document.querySelector("#auth-error");
    button.disabled = true; error.textContent = "";
    try { const result = await api(`/api/auth/${signup ? "signup" : "login"}`, { method: "POST", body: JSON.stringify(Object.fromEntries(new FormData(form))) }); location.assign(result.user.role === "teacher" ? "/app/teacher" : "/app/student"); }
    catch (err) { error.textContent = err.message; button.disabled = false; }
  });
}

async function loadIdentity() {
  try { state.user = (await api("/api/auth/me")).user; } catch { state.user = null; }
}

async function studentDashboard() {
  const data = await api("/api/v1/student/dashboard"); state.dashboard = data;
  const d = data, progress = d.featured_progress || {};
  shell(`<div class="heading"><div><p class="eyebrow">${esc(d.course.subject)} · ${esc(d.course.title)}</p><h1>Good morning, ${esc(d.student_name)}</h1><p class="lede">Your next small step is ready. Keep the conversation going.</p></div></div>
    <section class="resume"><div><p class="eyebrow">Recommended next lesson</p><h2>${esc(d.featured_concept.title)}</h2><p>${esc(d.featured_concept.summary)}</p><b>${progress.confidence}% confidence</b><small>Next: ${esc(progress.next_action)}</small></div><button class="button light" data-start="${esc(d.featured_concept.id)}">${d.active_session ? "Resume conversation" : "Start this lesson"} →</button></section>
    <section class="metrics"><div><b>${d.stats.concepts_started}</b><span>Concepts started</span></div><div><b>${d.stats.concepts_secure}</b><span>Secure concepts</span></div><div><b>${d.stats.learning_minutes}</b><span>Learning minutes</span></div><div><b>${d.turns.length}</b><span>Recent turns</span></div></section>
    <section class="grid-two"><article class="card"><div class="card-head"><div><h2>Your learning path</h2><p class="muted">A sequence of ideas, not a checklist.</p></div></div><div class="path">${d.concepts.map(card => `<button class="path-row" data-start="${esc(card.concept.id)}"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></button>`).join("")}</div></article><aside class="card"><h2>Teacher context</h2><p class="teacher-name">${esc(d.course.teacher_name)}</p><p class="muted">${esc(d.course.description)}</p><span class="tag">${esc(d.policy.source_policy)}</span></aside></section>`);
  document.querySelectorAll("[data-start]").forEach(button => button.addEventListener("click", () => startSession(button.dataset.start)));
}

async function startSession(conceptID) {
  try { const session = await api("/api/v1/student/sessions", { method: "POST", body: JSON.stringify({ course_id: state.dashboard.course.id, concept_id: conceptID, mode: state.dashboard.policy.default_mode }) }); location.assign(`/app/student/workspace?session_id=${encodeURIComponent(session.id)}`); }
  catch (err) { notify(err.message, "error"); }
}

async function workspace() {
  if (!state.dashboard) state.dashboard = await api("/api/v1/student/dashboard");
  const id = new URLSearchParams(location.search).get("session_id") || state.dashboard?.active_session?.id;
  if (!id) return studentDashboard();
  const data = await api(`/api/v1/student/sessions/${encodeURIComponent(id)}`); state.session = data.session; state.turns = data.turns;
  const concept = state.dashboard?.concepts.find(item => item.concept.id === data.session.concept_id)?.concept || { title: "Learning space", summary: "" };
  renderWorkspace(concept);
}

function renderWorkspace(concept, notice = "") {
  shell(`<div class="heading"><div><p class="eyebrow">${esc(state.dashboard?.course.title || "CoLearn")} · Learning space</p><h1>${esc(concept.title)}</h1><p class="lede">${esc(concept.summary)}</p></div><a class="button secondary" href="/app/student">← Overview</a></div>${notice ? `<div class="notice">${esc(notice)}</div>` : ""}<section class="workspace"><article class="card conversation"><div id="messages">${state.turns.length ? state.turns.map(turn => `<div class="message ${turn.role === "tutor" ? "tutor" : "student"}"><span class="message-avatar">${turn.role === "tutor" ? "✦" : initials(state.user.name)}</span><div><small>${turn.role === "tutor" ? "CoLearn tutor" : "You"} · ${esc(modeLabel(turn.mode))}</small><p>${esc(turn.body)}</p>${turn.sources?.length ? `<div class="sources">${turn.sources.map(source => `<span>${source.kind === "external" ? "↗" : "✦"} ${esc(source.title)}</span>`).join("")}</div>` : ""}</div></div>`).join("") : `<div class="empty"><b>Start with what you are wondering.</b><span>Ask about a step, request a hint, or tell the tutor what feels unclear.</span></div>`}</div><form id="turn-form" class="composer"><textarea name="message" required placeholder="Ask your tutor something…"></textarea><div><select name="mode">${(state.dashboard?.policy.allowed_modes || ["guided"]).map(mode => `<option value="${mode}" ${mode === state.session.mode ? "selected" : ""}>${modeLabel(mode)}</option>`).join("")}</select><button class="button primary" type="submit">Send →</button></div></form></article><aside class="card"><h2>Learning brief</h2><p class="muted">${esc(state.dashboard?.course.description || "")}</p><hr><b>Tutor mode</b><p class="muted">${modeLabel(state.session.mode)}</p><b>Next suggested action</b><p>${esc(state.dashboard?.featured_progress.next_action || "Keep exploring")}</p><button class="button secondary full" id="check-button">Check my understanding</button></aside></section>`);
  document.querySelector("#turn-form").addEventListener("submit", sendTurn);
  document.querySelector("#check-button").addEventListener("click", checkUnderstanding);
}

async function sendTurn(event) {
  event.preventDefault(); const form = event.currentTarget, button = form.querySelector("button"); button.disabled = true;
  try { const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/turns`, { method: "POST", body: JSON.stringify({ message: form.message.value, mode: form.mode.value }) }); state.session = result.session; state.turns.push({ role: "student", body: form.message.value, mode: form.mode.value }, result.turn); form.reset(); renderWorkspace(state.dashboard.concepts.find(item => item.concept.id === state.session.concept_id)?.concept || {}, result.next_action); }
  catch (err) { notify(err.message, "error"); button.disabled = false; }
}
async function checkUnderstanding() { try { const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/understanding-checks`, { method: "POST", body: "{}" }); notify(`${result.summary} Next: ${result.next_action}`); } catch (err) { notify(err.message, "error"); } }

async function teacherDashboard() {
  const d = await api("/api/v1/teacher/dashboard");
  shell(`<div class="heading"><div><p class="eyebrow">Teacher workspace · ${esc(d.course.subject)}</p><h1>${esc(d.course.title)}</h1><p class="lede">See where learners are building confidence and keep the path grounded.</p></div></div><section class="metrics"><div><b>${d.stats.active_learners}</b><span>Active learners</span></div><div><b>${d.stats.concepts_published}</b><span>Published concepts</span></div><div><b>${d.stats.average_confidence}%</b><span>Average confidence</span></div><div><b>${d.stats.needs_support}</b><span>Needs support</span></div></section><section class="grid-two"><article class="card"><h2>Concept health</h2><div class="path">${d.concepts.map(card => `<div class="path-row"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></div>`).join("")}</div></article><article class="card"><h2>Support needed</h2>${d.support_needed?.length ? d.support_needed.map(item => `<div class="support"><b>${esc(item.student_name)}</b><span>${esc(item.concept_title)} · ${item.confidence}%</span><small>${esc(item.next_action)}</small></div>`).join("") : `<div class="empty"><b>No urgent support signals</b><span>Your learners are moving steadily.</span></div>`}</article></section>`);
}

function notify(message, type = "success") { const node = document.createElement("div"); node.className = `toast ${type}`; node.textContent = message; document.body.append(node); setTimeout(() => node.remove(), 4500); }

async function route() {
  try {
    await loadIdentity();
    const path = location.pathname.replace(/^\/app/, "") || "/";
    if (path === "/login") return authPage(false);
    if (path === "/signup") return authPage(true);
    if (!state.user) return landing();
    if (path.startsWith("/teacher")) return state.user.role === "teacher" ? teacherDashboard() : landing();
    if (path.startsWith("/student/workspace")) return state.user.role === "student" ? workspace() : landing();
    return state.user.role === "student" ? studentDashboard() : teacherDashboard();
  } catch (err) { shell(`<section class="error-state"><h1>Something went wrong</h1><p>${esc(err.message)}</p><a class="button secondary" href="/">Return home</a></section>`); }
}
window.addEventListener("popstate", route);
document.addEventListener("click", event => { const link = event.target.closest("a"); if (link?.origin === location.origin && !link.target && !link.pathname.startsWith("/api/")) { event.preventDefault(); history.pushState({}, "", link.href); route(); } });
route();
