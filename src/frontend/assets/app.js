import { api, state } from "./app/core.js";
import { renderShell, renderError, bindShell } from "./app/shell.js";
import { renderLanding, renderAuth, bindAuth } from "./app/pages/auth.js";
import { renderProfile, bindProfile } from "./app/pages/profile.js";
import { renderStudentDashboard, renderWorkspace, bindStudentDashboard, bindWorkspace } from "./app/pages/student.js";
import { renderTeacherDashboard, bindTeacherDashboard } from "./app/pages/teacher.js";

const app = document.querySelector("#app");

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

function render(content, title) {
  app.innerHTML = renderShell(content, title, state);
  bindShell(app, route);
}

async function route() {
  try {
    await loadIdentity();
    const path = location.pathname.replace(/^\/app/, "") || "/";
    if (path === "/login") {
      render(renderAuth(false), "Sign in");
      bindAuth(app, false, route);
      return;
    }
    if (path === "/signup") {
      render(renderAuth(true), "Create account");
      bindAuth(app, true, route);
      return;
    }
    if (!state.user) {
      render(renderLanding(), "Learn with CoLearn");
      return;
    }
    if (path === "/profile") {
      render(renderProfile(), "Your profile");
      bindProfile(app, route);
      return;
    }
    if (path.startsWith("/teacher") && state.user.role === "teacher") {
      const content = await renderTeacherDashboard();
      render(content, "Teacher workspace");
      bindTeacherDashboard(app, route);
      return;
    }
    if (path.startsWith("/student/workspace") && state.user.role === "student") {
      const content = await renderWorkspace();
      render(content, "Learning space");
      bindWorkspace(app, route);
      return;
    }
    if (state.user.role === "student") {
      const content = await renderStudentDashboard();
      render(content, "Student overview");
      bindStudentDashboard(app, route);
      return;
    }
    const content = await renderTeacherDashboard();
    render(content, "Teacher workspace");
    bindTeacherDashboard(app, route);
  } catch (error) {
    render(renderError(error), "Something went wrong");
  }
}

window.addEventListener("popstate", route);
document.addEventListener("click", event => {
  const link = event.target.closest("a");
  if (!link || link.target || link.origin !== location.origin || link.pathname.startsWith("/api/")) return;
  if (link.pathname.startsWith("/teacher/courses/")) return;
  event.preventDefault();
  history.pushState({}, "", link.href);
  route();
});

route();
