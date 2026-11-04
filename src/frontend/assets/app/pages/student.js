import { api, esc, initials, modeLabel, state, curriculumLessons, curriculumSidebar, lessonDetail, progressRail, notify } from "../core.js";

export async function renderStudentDashboard() {
  const courseID = new URLSearchParams(location.search).get("course_id");
  state.dashboard = await api(`/api/v1/student/dashboard${courseID ? `?course_id=${encodeURIComponent(courseID)}` : ""}`);
  const data = state.dashboard;
  if (!data.course?.id) {
    const pending = (data.courses || []).filter(course => course.status !== "approved");
    return `<section class="empty-state card"><span class="status-icon">◎</span><p class="eyebrow">Student workspace</p><h1>Your learning space is ready</h1><p class="lede">Join an approved course to unlock the course map, tutor, and progress view.</p>${pending.length ? `<div class="enrollment-list">${pending.map(course => `<div class="support"><b>${esc(course.course_title)}</b><span>${esc(course.subject)} · ${esc(course.status)}</span></div>`).join("")}</div>` : `<p class="muted">No course has been assigned yet. Ask your teacher to approve your enrollment.</p>`}<a class="button primary" href="/student/courses">Browse courses <span>→</span></a></section>`;
  }
  state.curriculum = await api(`/api/v1/student/courses/${encodeURIComponent(data.course.id)}/curriculum`);
  const selectedLessonID = new URLSearchParams(location.search).get("lesson_id") || curriculumLessons(state.curriculum)[0]?.id || "";
  const courses = (data.courses || []).filter(course => course.status === "approved");
  const progress = data.featured_progress || {};
  return `<section class="page-heading"><div><p class="eyebrow">${esc(data.course.subject)} · ${esc(data.course.title)}</p><h1>Good morning, ${esc(data.student_name)}</h1><p class="lede">Your agent has a next best action ready.</p></div>${courses.length ? `<label class="course-picker">Active course<select id="student-course-picker">${courses.map(course => `<option value="${esc(course.course_id)}" ${course.course_id === data.course.id ? "selected" : ""}>${esc(course.course_title)}</option>`).join("")}</select></label>` : ""}</section><div class="workspace-shell">${curriculumSidebar(state.curriculum, data.course.id, "student", selectedLessonID)}<section class="workspace-main"><div id="lesson-detail" class="card lesson-detail-card">${lessonDetail(state.curriculum, selectedLessonID)}</div><section class="agent-banner"><div><p class="eyebrow">Agent recommendation</p><h2>${esc(data.featured_concept.title)}</h2><p>${esc(data.featured_concept.summary)}</p><div class="agent-meta"><span>${progress.confidence}% confidence</span><span>Next: ${esc(progress.next_action)}</span></div></div><button class="button light" data-start="${esc(data.featured_concept.id)}">${data.active_session ? "Resume session" : "Start session"} <span>→</span></button></section><section class="metrics"><div><b>${data.stats.concepts_started}</b><span>Concepts started</span></div><div><b>${data.stats.concepts_secure}</b><span>Concepts secure</span></div><div><b>${data.stats.learning_minutes}</b><span>Learning minutes</span></div><div><b>${data.turns.length}</b><span>Recent turns</span></div></section><section class="dashboard-columns"><article class="card"><div class="card-title"><div><p class="eyebrow">Course map</p><h2>Your learning path</h2><p class="muted">A shared view of the concepts you are building.</p></div></div><div class="path">${data.concepts.map(card => `<div class="path-row"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></div>`).join("")}</div></article>${progressRail(data.concepts, "Your progress")}<aside class="card context-card"><p class="eyebrow">Teacher context</p><h2>${esc(data.course.teacher_name)}</h2><p class="muted">${esc(data.course.description)}</p><span class="tag">${esc(data.policy.source_policy)}</span></aside></section></section></div>`;
}

export async function renderWorkspace() {
  const params = new URLSearchParams(location.search);
  if (!state.dashboard) state.dashboard = await api(`/api/v1/student/dashboard${params.get("course_id") ? `?course_id=${encodeURIComponent(params.get("course_id"))}` : ""}`);
  const id = params.get("session_id") || state.dashboard?.active_session?.id;
  if (!id) return renderStudentDashboard();
  const data = await api(`/api/v1/student/sessions/${encodeURIComponent(id)}`);
  state.session = data.session; state.turns = data.turns;
  const concept = state.dashboard?.concepts.find(item => item.concept.id === data.session.concept_id)?.concept || { title: "Learning space", summary: "" };
  return workspaceMarkup(concept);
}

function workspaceMarkup(concept, notice = "") {
  return `<section class="page-heading"><div><p class="eyebrow">${esc(state.dashboard?.course.title || "CoLearn")} · Active session</p><h1>${esc(concept.title)}</h1><p class="lede">${esc(concept.summary)}</p></div><span class="session-status"><i></i> Agent online</span></section>${notice ? `<div class="notice">Next suggested action: ${esc(notice)}</div>` : ""}<section class="workspace"><article class="card conversation"><div class="conversation-head"><div><p class="eyebrow">Tutor conversation</p><h2>Think out loud</h2></div><span class="tag">${modeLabel(state.session.mode)} mode</span></div><div id="messages">${state.turns.length ? state.turns.map(turn => `<div class="message ${turn.role === "tutor" ? "tutor" : "student"}"><span class="message-avatar">${turn.role === "tutor" ? "✦" : initials(state.user.name)}</span><div><small>${turn.role === "tutor" ? "CoLearn tutor" : "You"} · ${esc(modeLabel(turn.mode))}</small><p>${esc(turn.body)}</p>${turn.sources?.length ? `<div class="sources">${turn.sources.map(source => source.reference ? `<a href="${esc(source.reference)}" target="_blank" rel="noopener noreferrer">↗ ${esc(source.title)}</a>` : `<span>✦ ${esc(source.title)}</span>`).join("")}</div>` : ""}</div></div>`).join("") : `<div class="empty"><b>Start with what you are wondering.</b><span>Ask for an explanation, a hint, or a worked example.</span></div>`}</div><form id="turn-form" class="composer"><textarea name="message" required placeholder="Ask your tutor something..."></textarea><div><select name="mode">${(state.dashboard?.policy.allowed_modes || ["guided"]).map(mode => `<option value="${mode}" ${mode === state.session.mode ? "selected" : ""}>${modeLabel(mode)}</option>`).join("")}</select><button class="button primary" type="submit">Send <span>→</span></button></div></form></article><aside class="card agent-panel"><div class="agent-orb">✦</div><p class="eyebrow">Agent controls</p><h2>Let the tutor steer.</h2><p class="muted">${esc(state.dashboard?.course.description || "")}</p><div class="agent-detail"><span>Current mode</span><b>${modeLabel(state.session.mode)}</b></div><div class="agent-detail"><span>Suggested next</span><b>${esc(state.dashboard?.featured_progress.next_action || "Keep exploring")}</b></div><button class="button secondary full" id="agent-next">Plan my next step</button><button class="button secondary full" id="check-button">Check understanding</button></aside></section>`;
}

export function bindStudentDashboard(app, route) {
  app.querySelectorAll("[data-open-lesson]").forEach(button => button.addEventListener("click", () => {
    const lessonID = button.dataset.openLesson;
    app.querySelectorAll("[data-open-lesson]").forEach(item => item.classList.toggle("active", item === button));
    app.querySelector("#lesson-detail").innerHTML = lessonDetail(state.curriculum, lessonID);
    history.replaceState({}, "", `/app/student?course_id=${encodeURIComponent(state.dashboard.course.id)}&lesson_id=${encodeURIComponent(lessonID)}`);
  }));
  app.querySelector("[data-start]")?.addEventListener("click", startSession);
  app.querySelector("#student-course-picker")?.addEventListener("change", event => {
    history.replaceState({}, "", `/app/student?course_id=${encodeURIComponent(event.target.value)}`);
    route();
  });
}

async function startSession(event) {
  const button = event.currentTarget;
  button.disabled = true;
  try {
    const session = await api("/api/v1/student/sessions", { method: "POST", body: JSON.stringify({ course_id: state.dashboard.course.id, concept_id: button.dataset.start, mode: state.dashboard.policy.default_mode }) });
    location.assign(`/app/student/workspace?session_id=${encodeURIComponent(session.id)}`);
  } catch (error) { notify(error.message, "error"); button.disabled = false; }
}

export function bindWorkspace(app, route) {
  app.querySelector("#turn-form").addEventListener("submit", sendTurn);
  app.querySelector("#agent-next").addEventListener("click", runAgentNext);
  app.querySelector("#check-button").addEventListener("click", checkUnderstanding);
}

async function sendTurn(event) {
  event.preventDefault();
  const form = event.currentTarget, button = form.querySelector("button");
  button.disabled = true;
  try {
    const selectedMode = form.mode.value;
    const message = form.message.value;
    const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/turns`, { method: "POST", body: JSON.stringify({ message, mode: selectedMode }) });
    state.session = result.session; state.turns.push({ role: "student", body: message, mode: selectedMode }, result.turn);
    const concept = state.dashboard.concepts.find(item => item.concept.id === state.session.concept_id)?.concept || {};
    document.querySelector("#app main").innerHTML = workspaceMarkup(concept, result.next_action);
    bindWorkspace(document.querySelector("#app"), () => {});
  } catch (error) { notify(error.message, "error"); button.disabled = false; }
}

async function checkUnderstanding() {
  try {
    const result = await api(`/api/v1/student/sessions/${encodeURIComponent(state.session.id)}/understanding-checks`, { method: "POST", body: "{}" });
    notify(`${result.summary} Next: ${result.next_action}`);
  } catch (error) { notify(error.message, "error"); }
}

async function runAgentNext() {
  const button = document.querySelector("#agent-next");
  button.disabled = true; button.textContent = "Planning...";
  try {
    const result = await api("/api/v1/student/agent/next", { method: "POST", body: JSON.stringify({ session_id: state.session.id, mode: state.session.mode }) });
    state.session = result.session; state.turns.push(result.turn);
    const concept = state.dashboard.concepts.find(item => item.concept.id === state.session.concept_id)?.concept || {};
    document.querySelector("#app main").innerHTML = workspaceMarkup(concept, result.next_action);
    bindWorkspace(document.querySelector("#app"), () => {});
  } catch (error) { notify(error.message, "error"); button.disabled = false; button.textContent = "Plan my next step"; }
}
