export const state = {
  user: null,
  dashboard: null,
  teacherDashboard: null,
  curriculum: { modules: [] },
  session: null,
  turns: [],
  theme: localStorage.getItem("colearn-theme") || "light"
};

export const esc = (value = "") => String(value).replace(/[&<>"']/g, char => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
}[char]));

export const initials = (name = "") => name.split(/\s+/).filter(Boolean).map(part => part[0]).slice(0, 2).join("").toUpperCase() || "?";
export const modeLabel = mode => ({ guided: "Guided", socratic: "Socratic", direct: "Direct", practice: "Practice", hint: "Hint", diagnostic: "Diagnostic" }[mode] || mode);
export const curriculumLessons = curriculum => (curriculum?.modules || []).flatMap(module => (module.chapters || []).flatMap(chapter => chapter.lessons || []));

export async function api(path, options = {}) {
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

export function notify(message, type = "success") {
  const node = document.createElement("div");
  node.className = `toast ${type}`;
  node.setAttribute("role", type === "error" ? "alert" : "status");
  node.textContent = message;
  document.body.append(node);
  setTimeout(() => node.remove(), 4500);
}

export function progressRail(concepts, title) {
  const items = (concepts || []).slice(0, 6);
  const average = items.length ? Math.round(items.reduce((sum, item) => sum + (item.progress?.confidence || 0), 0) / items.length) : 0;
  return `<aside class="card progress-card"><div class="progress-ring" style="--progress:${average * 3.6}deg"><b>${average}%</b></div><h2>${esc(title)}</h2><p class="muted">${items.length ? "Confidence across the active learning path." : "Progress will appear as learners begin."}</p>${items.map(item => `<div class="progress-item"><span>${esc(item.concept.title)}</span><strong>${item.progress.confidence}%</strong><i><em style="width:${item.progress.confidence}%"></em></i></div>`).join("")}</aside>`;
}

export function curriculumSidebar(curriculum, courseID, role, activeLessonID = "") {
  const modules = curriculum?.modules || [];
  return `<aside class="curriculum-sidebar"><div class="sidebar-heading"><div><p class="eyebrow">Course map</p><h2>Contents</h2></div><span class="sidebar-count">${curriculumLessons(curriculum).length} lessons</span></div><div class="sidebar-tree">${modules.length ? modules.map(module => `<details class="sidebar-module" open><summary><span class="sidebar-index">${String(module.order).padStart(2, "0")}</span><span>${esc(module.title)}</span></summary>${(module.chapters || []).map(chapter => `<div class="sidebar-chapter"><small>${esc(chapter.title)}</small>${(chapter.lessons || []).map(lesson => role === "teacher" ? `<a class="sidebar-lesson ${lesson.id === activeLessonID ? "active" : ""}" href="/teacher/courses/${encodeURIComponent(courseID)}/curriculum"><span class="lesson-dot"></span><span>${esc(lesson.title)}</span></a>` : `<button class="sidebar-lesson ${lesson.id === activeLessonID ? "active" : ""}" type="button" data-open-lesson="${esc(lesson.id)}"><span class="lesson-dot"></span><span>${esc(lesson.title)}</span></button>`).join("")}</div>`).join("")}</details>`).join("") : `<div class="sidebar-empty"><b>No curriculum yet</b><span>Your course map will appear here.</span></div>`}</div>${role === "teacher" && courseID ? `<a class="button secondary sidebar-action" href="/teacher/courses/${encodeURIComponent(courseID)}/curriculum">Open curriculum studio</a>` : ""}</aside>`;
}

export function lessonDetail(curriculum, lessonID) {
  const lesson = curriculumLessons(curriculum).find(item => item.id === lessonID);
  if (!lesson) return `<div class="empty"><b>Select a lesson</b><span>Choose a lesson from the course map to see its content.</span></div>`;
  return `<div class="lesson-detail"><p class="eyebrow">Lesson ${String(lesson.order).padStart(2, "0")}</p><h2>${esc(lesson.title)}</h2><p class="muted">${esc(lesson.summary || "Work through this lesson at your own pace.")}</p>${lesson.contents?.length ? `<div class="lesson-content-cards">${lesson.contents.map(content => `<article><span class="tag">${esc(content.kind.replaceAll("_", " "))}</span><h3>${esc(content.title)}</h3><p>${esc(content.body)}</p></article>`).join("")}</div>` : `<div class="notice">Lesson content is being prepared by your teacher.</div>`}</div>`;
}
