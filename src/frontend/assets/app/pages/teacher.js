import { api, esc, state, curriculumSidebar, progressRail, notify } from "../core.js";

export async function renderTeacherDashboard() {
  const courseID = new URLSearchParams(location.search).get("course_id");
  state.teacherDashboard = await api(`/api/v1/teacher/dashboard${courseID ? `?course_id=${encodeURIComponent(courseID)}` : ""}`);
  const data = state.teacherDashboard;
  state.curriculum = data.course.id ? await api(`/api/v1/teacher/courses/${encodeURIComponent(data.course.id)}/curriculum`) : { modules: [] };
  const concepts = data.concepts || [], materials = data.materials || [], firstConcept = concepts[0]?.concept?.id || "";
  return `<section class="page-heading"><div><p class="eyebrow">Teacher workspace · ${esc(data.course.subject || "Your course")}</p><h1>${esc(data.course.title || "Set up your course")}</h1><p class="lede">See learner momentum and keep the tutor grounded in your teaching.</p></div>${(data.courses || []).length ? `<label class="course-picker">Active course<select id="teacher-course-picker">${data.courses.map(course => `<option value="${esc(course.id)}" ${course.id === data.course.id ? "selected" : ""}>${esc(course.title)}</option>`).join("")}</select></label>` : ""}</section><div class="workspace-shell">${curriculumSidebar(state.curriculum, data.course.id, "teacher")}<section class="workspace-main">${!data.course.id ? `<section class="card empty-state"><span class="status-icon">+</span><h2>Create your first course</h2><p>Give learners a clear starting point, then add concepts and teacher-grounded material.</p><form id="course-form" class="form-grid course-form"><label>Course title<input name="title" required maxlength="160" placeholder="Algebra foundations"></label><label>Subject<input name="subject" required maxlength="100" placeholder="Mathematics"></label><label>Description<textarea name="description" rows="4" maxlength="4000"></textarea><button class="button primary" type="submit">Create course <span>→</span></button><p class="form-error" id="course-error"></p></form></section>` : `<section class="metrics"><div><b>${data.stats.active_learners}</b><span>Active learners</span></div><div><b>${data.stats.concepts_published}</b><span>Published concepts</span></div><div><b>${data.stats.average_confidence}%</b><span>Average confidence</span></div><div><b>${data.stats.needs_support}</b><span>Needs support</span></div></section><section class="dashboard-columns"><article class="card"><div class="card-title"><div><p class="eyebrow">Class signal</p><h2>Concept health</h2><p class="muted">Where the class is building confidence.</p></div></div><div class="path">${concepts.length ? concepts.map(card => `<div class="path-row"><span class="number">${String(card.concept.order).padStart(2, "0")}</span><span><b>${esc(card.concept.title)}</b><small>${esc(card.concept.summary)}</small></span><strong class="status ${esc(card.progress.status)}">${card.progress.confidence}%</strong></div>`).join("") : `<div class="empty"><b>No concepts yet</b><span>Open curriculum studio to create the learning path.</span></div>`}</div></article>${progressRail(concepts, "Class progress")}<article class="card"><p class="eyebrow">Attention queue</p><h2>Support needed</h2><p class="muted">Signals worth your attention, not grades.</p>${data.support_needed?.length ? data.support_needed.map(item => `<div class="support"><b>${esc(item.student_name)}</b><span>${esc(item.concept_title)} · ${item.confidence}%</span><small>${esc(item.next_action)}</small></div>`).join("") : `<div class="empty"><b>No urgent signals</b><span>Your learners are moving steadily.</span></div>`}</article></section><section class="teacher-setup" id="course-setup"><article class="card"><div class="card-title"><div><p class="eyebrow">Ground the agent</p><h2>Course setup</h2><p class="muted">Keep the tutor’s context current without leaving this workspace.</p></div><span class="tag">${materials.length} materials</span></div>${firstConcept ? "" : `<div class="setup-empty"><b>Build the course outline first</b><span>Concepts and lessons are required before sources can be attached.</span><a class="button secondary" href="/teacher/courses/${encodeURIComponent(data.course.id)}/curriculum">Open curriculum studio</a></div>`}<div class="setup-grid" ${firstConcept ? "" : "hidden"}><form id="material-form" class="form-grid"><h3>Publish material</h3><input type="hidden" name="course_id" value="${esc(data.course.id)}"><label>Concept<select name="concept_id" required>${concepts.map(card => `<option value="${esc(card.concept.id)}">${esc(card.concept.title)}</option>`).join("")}</select></label><label>Title<input name="title" required maxlength="200"></label><label>Explanation<textarea name="body" required rows="5"></textarea></label><label>Source reference<input name="source_ref" maxlength="500"></label><button class="button primary" type="submit">Publish material</button><p class="form-error" id="material-error"></p></form><form id="pdf-material-form" class="form-grid"><h3>Upload PDF</h3><input type="hidden" name="course_id" value="${esc(data.course.id)}"><label>Concept<select name="concept_id" required>${concepts.map(card => `<option value="${esc(card.concept.id)}">${esc(card.concept.title)}</option>`).join("")}</select></label><label>PDF file<input type="file" name="file" accept="application/pdf,.pdf" required></label><button class="button secondary" type="submit">Upload and ground tutor</button><p class="form-error" id="pdf-material-error"></p></form><form id="policy-form" class="form-grid"><h3>Tutor policy</h3><label>Default mode<select name="default_mode">${(data.policy.allowed_modes || []).map(mode => `<option value="${esc(mode)}" ${mode === data.policy.default_mode ? "selected" : ""}>${esc(mode)}</option>`).join("")}</select></label><label>Source policy<input name="source_policy" value="${esc(data.policy.source_policy || "")}" maxlength="200"></label><label class="checkbox-field"><input type="checkbox" name="allow_external_sources" ${data.policy.allow_external_sources ? "checked" : ""}> Allow labeled external sources</label><button class="button secondary" type="submit">Save policy</button><p class="form-error" id="policy-error"></p></form></div><div class="material-list">${materials.length ? materials.map(material => `<div class="material-row"><div><b>${esc(material.title)}</b><span>${esc(material.source.title)} · v${material.version}</span></div><span class="tag">${material.published ? "Published" : "Draft"}</span></div>`).join("") : `<div class="empty"><b>No materials published</b><span>Give the tutor a source of truth.</span></div>`}</div></article></section></section>`}</div>`;
}

export function bindTeacherDashboard(app, route) {
  app.querySelector("#course-form")?.addEventListener("submit", submitCourse);
  app.querySelector("#material-form")?.addEventListener("submit", submitMaterial);
  app.querySelector("#pdf-material-form")?.addEventListener("submit", uploadPDF);
  app.querySelector("#policy-form")?.addEventListener("submit", savePolicy);
  app.querySelector("#teacher-course-picker")?.addEventListener("change", event => {
    history.replaceState({}, "", `/app/teacher?course_id=${encodeURIComponent(event.target.value)}`);
    route();
  });
}

async function submitCourse(event) {
  event.preventDefault();
  await formRequest(event.currentTarget, "/api/v1/teacher/courses", "Course created");
}
async function submitMaterial(event) {
  event.preventDefault();
  await formRequest(event.currentTarget, "/api/v1/teacher/materials", "Material published");
}
async function uploadPDF(event) {
  event.preventDefault();
  await formRequest(event.currentTarget, "/api/v1/teacher/materials/pdf", "PDF extracted and published", true);
}
async function savePolicy(event) {
  event.preventDefault();
  const form = event.currentTarget;
  await formRequest(form, `/api/v1/teacher/courses/${encodeURIComponent(state.teacherDashboard.course.id)}/tutor-policy`, "Tutor policy saved", false, {
    course_id: state.teacherDashboard.course.id,
    default_mode: form.default_mode.value,
    allowed_modes: state.teacherDashboard.policy.allowed_modes,
    allow_external_sources: form.allow_external_sources.checked,
    source_policy: form.source_policy.value
  });
}
async function formRequest(form, path, success, multipart = false, payload = null) {
  const button = form.querySelector("button"), error = form.querySelector(".form-error");
  button.disabled = true; if (error) error.textContent = "";
  try {
    const body = payload ? JSON.stringify(payload) : multipart ? new FormData(form) : JSON.stringify(Object.fromEntries(new FormData(form)));
    await api(path, { method: path.includes("tutor-policy") ? "PUT" : "POST", body });
    notify(success);
    location.reload();
  } catch (err) { if (error) error.textContent = err.message; button.disabled = false; }
}
