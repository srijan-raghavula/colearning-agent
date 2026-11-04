const app = document.querySelector("#app");
const state = { user: null, dashboard: null, teacherDashboard: null, curriculum: { modules: [] }, session: null, turns: [], loading: false, theme: localStorage.getItem("colearn-theme") || "light" };

const esc = (value = "") => String(value).replace(/[&<>"']/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char]));
const initials = name => name.split(/\s+/).filter(Boolean).map(part => part[0]).slice(0, 2).join("").toUpperCase() || "?";
const modeLabel = mode => ({ guided: "Guided", socratic: "Socratic", direct: "Direct", practice: "Practice", hint: "Hint", diagnostic: "Diagnostic" }[mode] || mode);
const curriculumLessons = curriculum => (curriculum?.modules || []).flatMap(module => (module.chapters || []).flatMap(chapter => chapter.lessons || []));

function curriculumSidebar(curriculum, courseID, role, activeLessonID = "") {
  const modules = curriculum?.modules || [];
  return `<aside class="curriculum-sidebar"><div class="sidebar-heading"><div><p class="eyebrow">Course notebook</p><h2>Contents</h2></div><span class="sidebar-count">${curriculumLessons(curriculum).length} lessons</span></div><div class="sidebar-tree">${modules.length ? modules.map(module => `<details class="sidebar-module" open><summary><span class="sidebar-index">${String(module.order).padStart(2, "0")}</span><span>${esc(module.title)}</span></summary>${(module.chapters || []).map(chapter => `<div class="sidebar-chapter"><small>${esc(chapter.title)}</small>${(chapter.lessons || []).map(lesson => role === "teacher" ? `<button class="sidebar-lesson ${lesson.id === activeLessonID ? "active" : ""}" type="button" data-teacher-curriculum="${esc(courseID)}"><span class="lesson-dot"></span><span>${esc(lesson.title)}</span></button>` : `<button class="sidebar-lesson ${lesson.id === activeLessonID ? "active" : ""}" type="button" data-open-lesson="${esc(lesson.id)}"><span class="lesson-dot"></span><span>${esc(lesson.title)}</span></button>`).join("")}</div>`).join("")}</details>`).join("") : `<div class="sidebar-empty"><b>No curriculum yet</b><span>Your teacher's modules and lessons will appear here.</span></div>`}</div>${role === "teacher" && courseID ? `<a class="button secondary sidebar-action" href="/teacher/courses/${encodeURIComponent(courseID)}/curriculum">Open Curriculum Studio →</a>` : ""}</aside>`;
}

function progressRail(concepts, title) {
  const items = (concepts || []).slice(0, 6);
  const average = items.length ? Math.round(items.reduce((sum, item) => sum + (item.progress?.confidence || 0), 0) / items.length) : 0;
  return `<aside class="card progress-rail"><div class="progress-ring" style="--progress:${average * 3.6}deg"><b>${average}%</b></div><h2>${title}</h2><p class="muted">${items.length ? "Confidence across the active learning path." : "Progress will appear as learners begin."}</p>${items.map(item => `<div class="progress-item"><span>${esc(item.concept.title)}</span><strong>${item.progress.confidence}%</strong><i><em style="width:${item.progress.confidence}%"></em></i></div>`).join("")}</aside>`;
}

function profilePage() {
  const profile = state.user.profile || {};
  const student = state.user.role === "student";
  shell(`<div class="heading"><div><p class="eyebrow">Account workspace</p><h1>Your profile</h1><p class="lede">Keep the shared account identity and your role-specific learning context in sync.</p></div></div>
    <section class="profile-layout"><article class="card profile-summary"><span class="avatar profile-avatar">${initials(state.user.name)}</span><h2>${esc(state.user.name)}</h2><p class="muted">${esc(state.user.email)}</p><span class="tag">${esc(state.user.role)}</span><p class="profile-id">Account ID · ${esc(state.user.id)}</p></article>
    <article class="card"><h2>Profile details</h2><form id="profile-form" class="form-grid"><label>Full name<input name="name" required maxlength="100" value="${esc(state.user.name)}"></label>${student ? `<label>Academic program<input name="academic_program" maxlength="120" value="${esc(profile.academic_program || "")}"></label><label>Branch<input name="branch" maxlength="120" value="${esc(profile.branch || "")}"></label><label>Semester<input name="semester" type="number" min="0" max="20" value="${Number(profile.semester || 0)}"></label><label>Academic year<input name="academic_year" maxlength="40" value="${esc(profile.academic_year || "")}"></label>` : `<label>Department<input name="department" maxlength="120" value="${esc(profile.department || "")}"></label><label>Teaching title<input name="teaching_title" maxlength="80" value="${esc(profile.teaching_title || "")}"></label>`}<p class="form-error" id="profile-error" role="status"></p><button class="button primary" type="submit">Save profile</button></form></article></section>`, "Your profile");
  document.querySelector("#profile-form").addEventListener("submit", async event => {
    event.preventDefault();
    const form = event.currentTarget, button = form.querySelector("button"), error = document.querySelector("#profile-error");
    button.disabled = true; error.textContent = "";
    try {
      const values = Object.fromEntries(new FormData(form)); values.semester = Number(values.semester || 0);
      const result = await api("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(values) });
      state.user = result.user;
      notify("Profile saved");
      profilePage();
    } catch (err) { error.textContent = err.message; button.disabled = false; }
  });
}

function lessonDetail(curriculum, lessonID) {
  const lesson = curriculumLessons(curriculum).find(item => item.id === lessonID);
  if (!lesson) return `<div class="empty"><b>Select a lesson</b><span>Choose a lesson from the course notebook to see its content.</span></div>`;
  return `<div class="lesson-detail"><p class="eyebrow">Lesson ${String(lesson.order).padStart(2, "0")}</p><h2>${esc(lesson.title)}</h2><p class="muted">${esc(lesson.summary || "Work through this lesson at your own pace.")}</p>${lesson.contents?.length ? `<div class="lesson-content-cards">${lesson.contents.map(content => `<article><span class="tag">${esc(content.kind.replaceAll("_", " "))}</span><h3>${esc(content.title)}</h3><p>${esc(content.body)}</p></article>`).join("")}</div>` : `<div class="notice">Lesson content is being prepared by your teacher.</div>`}</div>`;
}

function bindCurriculum(courseID, curriculum) {
  document.querySelectorAll("[data-open-lesson]").forEach(button => button.addEventListener("click", () => {
    const lessonID = button.dataset.openLesson;
    document.querySelectorAll("[data-open-lesson]").forEach(item => item.classList.toggle("active", item === button));
    const detail = document.querySelector("#lesson-detail");
    if (detail) detail.innerHTML = lessonDetail(curriculum, lessonID);
    history.replaceState({}, "", `/app/student?course_id=${encodeURIComponent(courseID)}&lesson_id=${encodeURIComponent(lessonID)}`);
  }));
  document.querySelectorAll("[data-teacher-curriculum]").forEach(button => button.addEventListener("click", () => location.assign(`/teacher/courses/${encodeURIComponent(button.dataset.teacherCurriculum)}/curriculum`)));
}

function applyLearningResult(result) {
  if (!result?.progress || !state.dashboard) return;
  state.dashboard.featured_progress = result.progress;
  state.dashboard.concepts = (state.dashboard.concepts || []).map(card =>
    card.concept.id === result.progress.concept_id ? { ...card, progress: result.progress, state: result.progress.status } : card);
}

async function api(path, options = {}) {
  const headers = { Accept: "application/json", ...(options.headers || {}) };
  if (!(options.body instanceof FormData)) headers["Content-Type"] = "application/json";
  const response = await fetch(path, { credentials: "same-origin", ...options, headers });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(body.error || `Request failed (${response.status})`);
    error.status = response.status;
    throw error;
  }
  return body;
}

function shell(content, title = "CoLearn") {
  document.title = title === "CoLearn" ? "CoLearn" : `${title} · CoLearn`;
  document.documentElement.dataset.theme = state.theme;
  const roleClass = state.user ? `role-${esc(state.user.role)}` : "role-guest";
  const studentOverviewActive = location.pathname === "/app/student" && !location.pathname.includes("/workspace");
  const studentLearningActive = location.pathname.includes("/workspace");
  const teacherOverviewActive = location.pathname === "/app/teacher" && !location.hash;
  const teacherSetupActive = location.hash === "#course-setup";
  const navigation = state.user?.role === "teacher"
    ? `<nav class="role-nav" aria-label="Teacher workspace"><a class="${teacherOverviewActive ? "active" : ""}" href="/app/teacher">Class overview</a><a class="${teacherSetupActive ? "active" : ""}" href="/app/teacher#course-setup">Course setup</a></nav>`
    : `<nav class="role-nav" aria-label="Student workspace"><a class="${studentOverviewActive ? "active" : ""}" href="/app/student">Overview</a><a class="${studentLearningActive ? "active" : ""}" href="/app/student/workspace">Learning space</a></nav>`;
  app.innerHTML = `<div class="app-shell ${roleClass}">
    <header class="topbar"><a class="brand" href="/"><span class="brand-mark">C</span><span><b>CoLearn</b><small>Human + AI learning</small></span></a>
      ${state.user ? `${navigation}
      <div class="account"><button class="theme-toggle" type="button" data-action="theme">${state.theme === "dark" ? "☀ Light" : "☾ Dark"}</button><a class="profile-link" href="/app/profile">Profile</a><span class="avatar">${initials(state.user.name)}</span><span>${esc(state.user.name)}</span><button class="link-button" data-action="logout">Sign out</button></div>` : ""}
    </header><main class="page">${content}</main>
  </div>`;
  app.querySelector('[data-action="theme"]')?.addEventListener("click", () => {
    state.theme = state.theme === "dark" ? "light" : "dark";
    localStorage.setItem("colearn-theme", state.theme);
    shell(content, title);
  });
  app.querySelector('[data-action="logout"]')?.addEventListener("click", async () => { try { await fetch("/logout", { credentials: "same-origin" }); } finally { location.assign("/"); } });
  app.querySelector("[data-retry]")?.addEventListener("click", () => route());
}

function landing() {
  shell(`<section class="hero"><div><p class="eyebrow">Learning that keeps going</p><h1>Understand more.<br><em>Keep moving.</em></h1><p class="lede">CoLearn pairs a teacher’s curriculum with a configurable AI tutor. Ask questions, practice at your pace, and see your understanding grow over time.</p><div class="actions"><a class="button primary" href="/signup">Create your workspace →</a><a class="button secondary" href="/login">Sign in</a></div><p class="trust">✓ Teacher-approved context · adaptive tutor modes · formative checks</p></div><div class="hero-card"><div class="mock-top"><i></i><i></i><i></i></div><div class="mock-body"><p class="eyebrow">YOUR LEARNING PATH</p><h2>One small step at a time</h2><div class="mock-item"><b>Linear equations</b><span>68% confidence</span></div><div class="mock-item"><b>Graphing equations</b><span>Needs support</span></div></div></div></section><section class="features"><article><b>01 · Teacher-led</b><span>Concepts and source material stay grounded in what your teacher provides.</span></article><article><b>02 · Adaptive</b><span>Choose a tutor mode that matches how you want to learn.</span></article><article><b>03 · Continuous</b><span>Sessions and evidence stay connected between visits.</span></article></section>`, "Learn with CoLearn");
}

function authPage(signup = false) {
  if (!signup) {
    shell(`<section class="auth-card"><span class="brand-mark">C</span><p class="eyebrow">Welcome back</p><h1>Sign in to CoLearn</h1><p class="muted">Continue where your learning left off.</p><form id="auth-form" class="form-grid"><label>Email<input name="email" type="email" required autocomplete="username"></label><label>Password<input name="password" type="password" required minlength="8" maxlength="72" autocomplete="current-password"></label><p class="form-error" id="auth-error"></p><button class="button primary full" type="submit">Sign in</button></form><p class="auth-switch">New to CoLearn? <a href="/signup">Create one</a></p></section>`, "Sign in");
    bindAuthForm(false);
    return;
  }
  const wizard = { step: 1, data: { role: "student" } };
  const render = () => {
    const role = wizard.data.role;
    const steps = role === "student" ? 5 : 4;
    const body = wizard.step === 1
      ? `<fieldset class="wizard-options"><legend>Choose your workspace</legend><label class="role-option"><input type="radio" name="role" value="student" ${role === "student" ? "checked" : ""}><span><b>Student</b><small>Learn with a course explorer and tutor.</small></span></label><label class="role-option"><input type="radio" name="role" value="teacher" ${role === "teacher" ? "checked" : ""}><span><b>Teacher</b><small>Build courses and support learners.</small></span></label></fieldset>`
      : wizard.step === 2
        ? `<div class="wizard-fields"><label>Name<input name="name" required maxlength="100" autocomplete="name" value="${esc(wizard.data.name || "")}"></label><label>Email<input name="email" type="email" required autocomplete="email" value="${esc(wizard.data.email || "")}"></label></div>`
        : wizard.step === 3
          ? `<div class="wizard-fields"><label>Password<input name="password" type="password" required minlength="8" maxlength="72" autocomplete="new-password"></label><p class="muted">Use at least 8 characters. Your session token will keep this workspace signed in.</p></div>`
          : role === "student" && wizard.step === 4
            ? `<div class="wizard-fields"><label>Academic program<input name="academic_program" required maxlength="120" placeholder="BSc Computer Science" value="${esc(wizard.data.academic_program || "")}"></label><label>Branch<input name="branch" required maxlength="120" placeholder="Systems" value="${esc(wizard.data.branch || "")}"></label></div>`
            : role === "student"
              ? `<div class="wizard-fields"><label>Semester<input name="semester" type="number" min="1" max="20" required value="${esc(wizard.data.semester || "")}"></label><label>Academic year<input name="academic_year" required maxlength="40" placeholder="2026" value="${esc(wizard.data.academic_year || "")}"></label></div>`
              : `<div class="wizard-fields"><label>Department<input name="department" required maxlength="120" placeholder="Mathematics" value="${esc(wizard.data.department || "")}"></label><label>Teaching title<input name="teaching_title" required maxlength="80" placeholder="Professor" value="${esc(wizard.data.teaching_title || "")}"></label></div>`;
    shell(`<section class="auth-card wizard-card"><span class="brand-mark">C</span><p class="eyebrow">Create your workspace · ${wizard.step}/${steps}</p><h1>${wizard.step === 1 ? "Start with your role" : wizard.step === 2 ? "Tell us who you are" : wizard.step === 3 ? "Secure your account" : "Shape your workspace"}</h1><p class="muted">We only ask for two or three details at a time.</p><form id="auth-form" class="form-grid">${body}<p class="form-error" id="auth-error"></p><div class="wizard-actions">${wizard.step > 1 ? `<button class="button secondary" type="button" data-wizard-back>Back</button>` : ""}<button class="button primary" type="submit">${wizard.step === steps ? "Create workspace" : "Continue"}</button></div></form><p class="auth-switch">Already have an account? <a href="/login">Sign in</a></p></section>`, "Create account");
    const form = document.querySelector("#auth-form");
    form.addEventListener("submit", async event => {
      event.preventDefault();
      const values = Object.fromEntries(new FormData(form));
      Object.assign(wizard.data, values);
      if (wizard.step < steps) {
        wizard.step += 1;
        render();
        return;
      }
      const button = form.querySelector("button[type=submit]"), error = document.querySelector("#auth-error");
      button.disabled = true; error.textContent = "";
      try {
        const result = await api("/api/auth/signup", { method: "POST", body: JSON.stringify({ ...wizard.data, semester: Number(wizard.data.semester || 0) }) });
        location.assign(result.user.role === "teacher" ? "/app/teacher" : "/app/student");
      } catch (err) { error.textContent = err.message; button.disabled = false; }
    });
    form.querySelector("[data-wizard-back]")?.addEventListener("click", () => { wizard.step -= 1; render(); });
    form.querySelectorAll("input[name=role]").forEach(input => input.addEventListener("change", () => { wizard.data.role = input.value; wizard.step = 1; render(); }));
  };
  render();
}

function bindAuthForm(signup) {
  document.querySelector("#auth-form").addEventListener("submit", async event => {
    event.preventDefault();
    const form = event.currentTarget, button = form.querySelector("button"), error = document.querySelector("#auth-error");
    button.disabled = true; error.textContent = "";
    try { const result = await api("/api/auth/login", { method: "POST", body: JSON.stringify(Object.fromEntries(new FormData(form))) }); location.assign(result.user.role === "teacher" ? "/app/teacher" : "/app/student"); }
    catch (err) { error.textContent = err.message; button.disabled = false; }
  });
}

async function loadIdentity() {
  try {
    state.user = (await api("/api/auth/me")).user;
  } catch (error) {
    if (error.status === 401) {
      state.user = null;
      return;
    }
    throw error;
  }
}

async function studentDashboard(courseID = new URLSearchParams(location.search).get("course_id")) {
  const data = await api(`/api/v1/student/dashboard${courseID ? `?course_id=${encodeURIComponent(courseID)}` : ""}`); state.dashboard = data;
  if (!data.course?.id) {
    const pending = (data.courses || []).filter(course => course.status !== "approved");
    shell(`<section class="empty-state card"><p class="eyebrow">Student workspace</p><h1>Your learning space is ready</h1><p class="lede">Join an approved course to unlock the curriculum, tutor, and progress view.</p>${pending.length ? `<div class="enrollment-list">${pending.map(course => `<div class="support"><b>${esc(course.course_title)}</b><span>${esc(course.subject)} · ${esc(course.status)}</span></div>`).join("")}</div>` : `<p class="muted">No course has been assigned yet. Ask your teacher to approve your enrollment before starting a lesson.</p>`}<a class="button primary" href="/student/courses">Browse courses →</a></section>`, "Choose a course");
    return;
  }
  state.curriculum = await api(`/api/v1/student/courses/${encodeURIComponent(data.course.id)}/curriculum`);
  const d = data, progress = d.featured_progress || {};
  const availableCourses = (d.courses || []).filter(course => course.status === "approved");
  const selectedLessonID = new URLSearchParams(location.search).get("lesson_id") || curriculumLessons(state.curriculum)[0]?.id || "";
  shell(`<div class="heading"><div><p class="eyebrow">${esc(d.course.subject)} · ${esc(d.course.title)}</p><h1>Good morning, ${esc(d.student_name)}</h1><p class="lede">Your next small step is ready. Keep the conversation going.</p></div>${availableCourses.length ? `<label class="course-picker">Learning space<select id="student-course-picker">${availableCourses.map(course => `<option value="${esc(course.course_id)}" ${course.course_id === d.course.id ? "selected" : ""}>${esc(course.course_title)} · ${esc(course.teacher_name)}</option>`).join("")}</select></label>` : ""}</div>
    <div class="workspace-shell">${curriculumSidebar(state.curriculum, d.course.id, "student", selectedLessonID)}<section class="workspace-main"><div id="lesson-detail" class="card lesson-detail-card">${lessonDetail(state.curriculum, selectedLessonID)}</div>
    <section class="resume"><div><p class="eyebrow">Recommended next lesson</p><h2>${esc(d.featured_concept.title)}</h2><p>${esc(d.featured_concept.summary)}</p><b>${progress.confidence}% confidence</b><small>Next: ${esc(progress.next_action)}</small></div><button class="button light" data-start="${esc(d.featured_concept.id)}">${d.active_session ? "Resume conversation" : "Start this lesson"} →</button></section>
    <section class="metrics"><div><b>${d.stats.concepts_started}</b><span>Concepts started</span></div><div><b>${d.stats.concepts_secure}</b><span>Secure concepts</span></div><div><b>${d.stats.learning_minutes}</b><span>Learning minutes</span></div><div><b>${d.turns.length}</b><span>Recent turns</span></div></section>
    <section class="dashboard-columns"><article class="card"><div class="card-head"><div><h2>Your learning path</h2><p class="muted">Choose a lesson from the primary action above. This path shows your progress at a glance.</p></div></div><div class="path">${d.concepts.map(card => `<div class="path-row path-row-static"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></div>`).join("")}</div></article>${progressRail(d.concepts, "Your progress")}<aside class="card"><h2>Teacher context</h2><p class="teacher-name">${esc(d.course.teacher_name)}</p><p class="muted">${esc(d.course.description)}</p><span class="tag">${esc(d.policy.source_policy)}</span></aside></section></section></div>`);
  bindCurriculum(d.course.id, state.curriculum);
  document.querySelectorAll("[data-start]").forEach(button => button.addEventListener("click", () => startSession(button)));
  document.querySelector("#student-course-picker")?.addEventListener("change", event => {
    history.replaceState({}, "", `/app/student?course_id=${encodeURIComponent(event.target.value)}`);
    studentDashboard(event.target.value);
  });
}

async function startSession(button) {
  const conceptID = button.dataset.start;
  button.disabled = true;
  button.setAttribute("aria-busy", "true");
  const originalLabel = button.textContent;
  button.textContent = "Opening…";
  try { const session = await api("/api/v1/student/sessions", { method: "POST", body: JSON.stringify({ course_id: state.dashboard.course.id, concept_id: conceptID, mode: state.dashboard.policy.default_mode }) }); location.assign(`/app/student/workspace?session_id=${encodeURIComponent(session.id)}`); }
  catch (err) { notify(err.message, "error"); button.disabled = false; button.removeAttribute("aria-busy"); button.textContent = originalLabel; }
}

async function workspace() {
  const params = new URLSearchParams(location.search);
  if (!state.dashboard) state.dashboard = await api(`/api/v1/student/dashboard${params.get("course_id") ? `?course_id=${encodeURIComponent(params.get("course_id"))}` : ""}`);
  const id = params.get("session_id") || state.dashboard?.active_session?.id;
  if (!id) return studentDashboard();
  const data = await api(`/api/v1/student/sessions/${encodeURIComponent(id)}`); state.session = data.session; state.turns = data.turns;
  const concept = state.dashboard?.concepts.find(item => item.concept.id === data.session.concept_id)?.concept || { title: "Learning space", summary: "" };
  renderWorkspace(concept);
}

function renderWorkspace(concept, notice = "") {
  shell(`<div class="heading"><div><p class="eyebrow">${esc(state.dashboard?.course.title || "CoLearn")} · Learning space</p><h1>${esc(concept.title)}</h1><p class="lede">${esc(concept.summary)}</p></div></div>${notice ? `<div class="notice">Next suggested action: ${esc(notice)}</div>` : ""}<section class="workspace"><article class="card conversation"><div id="messages">${state.turns.length ? state.turns.map(turn => `<div class="message ${turn.role === "tutor" ? "tutor" : "student"}"><span class="message-avatar">${turn.role === "tutor" ? "✦" : initials(state.user.name)}</span><div><small>${turn.role === "tutor" ? "CoLearn tutor" : "You"} · ${esc(modeLabel(turn.mode))}</small><p>${esc(turn.body)}</p>${turn.sources?.length ? `<div class="sources">${turn.sources.map(source => source.reference ? `<a href="${esc(source.reference)}" target="_blank" rel="noopener noreferrer">${source.kind === "external" ? "↗" : "✦"} ${esc(source.title)}</a>` : `<span>${source.kind === "external" ? "↗" : "✦"} ${esc(source.title)}</span>`).join("")}</div>` : ""}</div></div>`).join("") : `<div class="empty"><b>Start with what you are wondering.</b><span>Ask about a step, request a hint, or tell the tutor what feels unclear.</span></div>`}</div><form id="turn-form" class="composer"><textarea name="message" required placeholder="Ask your tutor something…"></textarea><div><select name="mode">${(state.dashboard?.policy.allowed_modes || ["guided"]).map(mode => `<option value="${mode}" ${mode === state.session.mode ? "selected" : ""}>${modeLabel(mode)}</option>`).join("")}</select><button class="button primary" type="submit">Send →</button></div></form></article><aside class="card"><h2>Learning brief</h2><p class="muted">${esc(state.dashboard?.course.description || "")}</p><hr><b>Tutor mode</b><p class="muted">${modeLabel(state.session.mode)}</p><b>Next suggested action</b><p>${esc(state.dashboard?.featured_progress.next_action || "Keep exploring")}</p><button class="button secondary full" id="agent-next">Plan my next step</button><button class="button secondary full" id="check-button">Check my understanding</button></aside></section>`);
  document.querySelector("#turn-form").addEventListener("submit", sendTurn);
  document.querySelector("#agent-next").addEventListener("click", runAgentNext);
  document.querySelector("#check-button").addEventListener("click", checkUnderstanding);
}

async function sendTurn(event) {
  event.preventDefault(); const form = event.currentTarget, button = form.querySelector("button"); button.disabled = true;
  try { const selectedMode = form.mode.value; const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/turns`, { method: "POST", body: JSON.stringify({ message: form.message.value, mode: selectedMode }) }); state.session = result.session; applyLearningResult(result); state.turns.push({ role: "student", body: form.message.value, mode: selectedMode }, result.turn); form.reset(); form.mode.value = state.session.mode; renderWorkspace(state.dashboard.concepts.find(item => item.concept.id === state.session.concept_id)?.concept || {}, result.next_action); }
  catch (err) { notify(err.message, "error"); button.disabled = false; }
}
async function checkUnderstanding() { try { const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/understanding-checks`, { method: "POST", body: "{}" }); notify(`${result.summary} Next: ${result.next_action}`); } catch (err) { notify(err.message, "error"); } }
async function runAgentNext() { const button = document.querySelector("#agent-next"); button.disabled = true; button.textContent = "Planning…"; try { const result = await api("/api/v1/student/agent/next", { method: "POST", body: JSON.stringify({ session_id: state.session.id, mode: state.session.mode }) }); state.session = result.session; applyLearningResult(result); state.turns.push(result.turn); renderWorkspace(state.dashboard.concepts.find(item => item.concept.id === state.session.concept_id)?.concept || {}, result.next_action); } catch (err) { notify(err.message, "error"); button.disabled = false; button.textContent = "Plan my next step"; } }

async function teacherDashboard(courseID = new URLSearchParams(location.search).get("course_id")) {
  const d = await api(`/api/v1/teacher/dashboard${courseID ? `?course_id=${encodeURIComponent(courseID)}` : ""}`); state.teacherDashboard = d;
  state.curriculum = d.course.id ? await api(`/api/v1/teacher/courses/${encodeURIComponent(d.course.id)}/curriculum`) : { modules: [] };
  const concepts = d.concepts || [];
  const firstConcept = concepts[0]?.concept?.id || "";
  const materials = d.materials || [];
  shell(`<div class="heading"><div><p class="eyebrow">Teacher workspace · ${esc(d.course.subject || "Your course")}</p><h1>${esc(d.course.title || "Set up your course")}</h1><p class="lede">One place to understand learner momentum and keep the tutor grounded in your teaching.</p></div>${(d.courses || []).length ? `<label class="course-picker">Course<select id="teacher-course-picker">${d.courses.map(course => `<option value="${esc(course.id)}" ${course.id === d.course.id ? "selected" : ""}>${esc(course.title)}</option>`).join("")}</select></label>` : ""}</div>
    <div class="workspace-shell">${curriculumSidebar(state.curriculum, d.course.id, "teacher")}<section class="workspace-main">
    ${!d.course.id ? `<section class="card empty-state"><h2>Create your first course</h2><p>Give learners a clear starting point, then add concepts and teacher-grounded material.</p><form id="course-form" class="form-grid course-form"><label>Course title<input name="title" required maxlength="160" placeholder="e.g. Algebra foundations"></label><label>Subject<input name="subject" required maxlength="100" placeholder="e.g. Mathematics"></label><label>Description<textarea name="description" rows="4" maxlength="4000" placeholder="What will learners understand?"></textarea><button class="button primary" type="submit">Create course</button><p class="form-error" id="course-error" role="status"></p></form></section>` : `<section class="metrics"><div><b>${d.stats.active_learners}</b><span>Active learners</span></div><div><b>${d.stats.concepts_published}</b><span>Published concepts</span></div><div><b>${d.stats.average_confidence}%</b><span>Average confidence</span></div><div><b>${d.stats.needs_support}</b><span>Needs support</span></div></section>
    <section class="dashboard-columns" id="overview"><article class="card"><div class="card-head"><div><h2>Concept health</h2><p class="muted">Where the class is building confidence.</p></div></div><div class="path">${concepts.length ? concepts.map(card => `<div class="path-row"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></div>`).join("") : `<div class="empty"><b>No concepts yet</b><span>Add course content to create the learning path.</span></div>`}    </div></article>${progressRail(concepts, "Class progress")}<article class="card"><h2>Support needed</h2><p class="muted">Signals worth your attention, not grades.</p>${d.support_needed?.length ? d.support_needed.map(item => `<div class="support"><b>${esc(item.student_name)}</b><span>${esc(item.concept_title)} · ${item.confidence}%</span><small>${esc(item.next_action)}</small></div>`).join("") : `<div class="empty"><b>No urgent support signals</b><span>Your learners are moving steadily.</span></div>`}</article></section>
    <section class="teacher-setup" id="course-setup"><article class="card"><div class="card-head"><div><h2>Course setup</h2><p class="muted">Keep the tutor’s context current without leaving this workspace.</p></div><span class="tag">${materials.length} materials</span></div>${firstConcept ? "" : `<div class="setup-empty"><b>Build the course outline first</b><span>Concepts and lessons are required before tutor sources can be attached.</span><a class="button secondary" href="/teacher/courses/${encodeURIComponent(d.course.id)}/curriculum">Open Curriculum Studio →</a></div>`}<div class="setup-grid" ${firstConcept ? "" : "hidden"}><form id="material-form" class="form-grid"><h3>Publish material</h3><input type="hidden" name="course_id" value="${esc(d.course.id)}"><label>Concept<select name="concept_id" required>${concepts.map(card => `<option value="${esc(card.concept.id)}">${esc(card.concept.title)}</option>`).join("")}</select></label><label>Title<input name="title" required maxlength="200" placeholder="e.g. Worked example"></label><label>Explanation<textarea name="body" required rows="5" placeholder="What should the tutor use?"></textarea></label><label>Source reference<input name="source_ref" maxlength="500" placeholder="Optional link or citation"></label><button class="button primary" type="submit" ${firstConcept ? "" : "disabled"}>Publish material</button><p class="form-error" id="material-error" role="status"></p>    </form><form id="pdf-material-form" class="form-grid"><h3>Upload PDF material</h3><input type="hidden" name="course_id" value="${esc(d.course.id)}"><label>Concept<select name="concept_id" required>${concepts.map(card => `<option value="${esc(card.concept.id)}">${esc(card.concept.title)}</option>`).join("")}</select></label><label>PDF file<input type="file" name="file" accept="application/pdf,.pdf" required></label><button class="button secondary" type="submit">Upload and ground tutor</button><p class="form-error" id="pdf-material-error" role="status"></p></form><form id="policy-form" class="form-grid"><h3>Tutor policy</h3><label>Default mode<select name="default_mode">${(d.policy.allowed_modes || []).map(mode => `<option value="${esc(mode)}" ${mode === d.policy.default_mode ? "selected" : ""}>${modeLabel(mode)}</option>`).join("")}</select></label><label>Source policy<input name="source_policy" value="${esc(d.policy.source_policy || "")}" maxlength="200"></label><label class="checkbox-field"><input type="checkbox" name="allow_external_sources" ${d.policy.allow_external_sources ? "checked" : ""}> Allow labeled external sources</label><button class="button secondary" type="submit">Save policy</button><p class="form-error" id="policy-error" role="status"></p></form></div><div class="material-list">${materials.length ? materials.map(material => `<div class="material-row"><div><b>${esc(material.title)}</b><span>${esc(material.source.title)} · v${material.version}</span></div><span class="tag">${material.published ? "Published" : "Draft"}</span></div>`).join("") : `<div class="empty"><b>No materials published</b><span>Give the tutor a source of truth for this course.</span></div>`}</div></article></section></section></div>`}`);
  bindCurriculum(d.course.id, state.curriculum);
  document.querySelector("#material-form")?.addEventListener("submit", publishMaterial);
  document.querySelector("#pdf-material-form")?.addEventListener("submit", uploadPDFMaterial);
  document.querySelector("#policy-form")?.addEventListener("submit", savePolicy);
  document.querySelector("#course-form")?.addEventListener("submit", createCourse);
  document.querySelector("#teacher-course-picker")?.addEventListener("change", event => {
    history.replaceState({}, "", `/app/teacher?course_id=${encodeURIComponent(event.target.value)}`);
    teacherDashboard(event.target.value);
  });
  if (location.hash) document.querySelector(location.hash)?.scrollIntoView({ block: "start" });
}

async function createCourse(event) {
  event.preventDefault();
  const form = event.currentTarget, button = form.querySelector("button"), error = form.querySelector(".form-error");
  button.disabled = true; error.textContent = "";
  try { await api("/api/v1/teacher/courses", { method: "POST", body: JSON.stringify(Object.fromEntries(new FormData(form))) }); notify("Course created"); await teacherDashboard(); }
  catch (err) { error.textContent = err.message; button.disabled = false; }
}

async function publishMaterial(event) {
  event.preventDefault();
  const form = event.currentTarget, button = form.querySelector("button"), error = form.querySelector(".form-error");
  button.disabled = true; error.textContent = "";
  try { await api("/api/v1/teacher/materials", { method: "POST", body: JSON.stringify(Object.fromEntries(new FormData(form))) }); notify("Material published"); await teacherDashboard(); }
  catch (err) { error.textContent = err.message; button.disabled = false; }
}

async function uploadPDFMaterial(event) {
  event.preventDefault();
  const form = event.currentTarget, button = form.querySelector("button"), error = form.querySelector(".form-error");
  button.disabled = true; error.textContent = "";
  try { await api("/api/v1/teacher/materials/pdf", { method: "POST", body: new FormData(form) }); notify("PDF extracted and published"); await teacherDashboard(); }
  catch (err) { error.textContent = err.message; button.disabled = false; }
}

async function savePolicy(event) {
  event.preventDefault();
  const form = event.currentTarget, button = form.querySelector("button"), error = form.querySelector(".form-error");
  button.disabled = true; error.textContent = "";
  try {
    const policy = state.teacherDashboard.policy;
    await api(`/api/v1/teacher/courses/${encodeURIComponent(state.teacherDashboard.course.id)}/tutor-policy`, { method: "PUT", body: JSON.stringify({ course_id: state.teacherDashboard.course.id, default_mode: form.default_mode.value, allowed_modes: policy.allowed_modes, allow_external_sources: form.allow_external_sources.checked, source_policy: form.source_policy.value }) });
    notify("Tutor policy saved"); await teacherDashboard();
  } catch (err) { error.textContent = err.message; button.disabled = false; }
}

function notify(message, type = "success") { const node = document.createElement("div"); node.className = `toast ${type}`; node.setAttribute("role", type === "error" ? "alert" : "status"); node.setAttribute("aria-live", type === "error" ? "assertive" : "polite"); node.textContent = message; document.body.append(node); setTimeout(() => node.remove(), 4500); }

async function route() {
  try {
    await loadIdentity();
    const path = location.pathname.replace(/^\/app/, "") || "/";
    if (path === "/login") return authPage(false);
    if (path === "/signup") return authPage(true);
    if (!state.user) return landing();
    if (path === "/profile") return profilePage();
    if (path.startsWith("/teacher")) return state.user.role === "teacher" ? teacherDashboard() : landing();
    if (path.startsWith("/student/workspace")) return state.user.role === "student" ? workspace() : landing();
    return state.user.role === "student" ? studentDashboard() : teacherDashboard();
  } catch (err) { shell(`<section class="error-state"><h1>Something went wrong</h1><p>${esc(err.message)}</p><div class="error-actions"><button class="button primary" type="button" data-retry>Try again</button><a class="button secondary" href="/">Return home</a></div></section>`); }
}
window.addEventListener("popstate", route);
document.addEventListener("click", event => { const link = event.target.closest("a"); if (link?.origin === location.origin && !link.target && !link.pathname.startsWith("/api/") && !link.pathname.startsWith("/teacher/courses/")) { event.preventDefault(); history.pushState({}, "", link.href); route(); } });
route();
