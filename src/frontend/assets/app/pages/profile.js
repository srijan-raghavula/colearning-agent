import { api, esc, state, initials, notify } from "../core.js";

export function renderProfile() {
  const profile = state.user.profile || {};
  const student = state.user.role === "student";
  return `<section class="page-heading"><div><p class="eyebrow">Account settings</p><h1>Your profile</h1><p class="lede">Keep your identity and learning context current.</p></div></section><section class="profile-layout"><article class="card identity-card"><span class="avatar profile-avatar">${initials(state.user.name)}</span><h2>${esc(state.user.name)}</h2><p class="muted">${esc(state.user.email)}</p><span class="tag">${esc(state.user.role)}</span><p class="profile-id">Account ID · ${esc(state.user.id)}</p></article><article class="card"><div class="card-title"><div><p class="eyebrow">Personal details</p><h2>Workspace profile</h2></div></div><form id="profile-form" class="form-grid"><label>Full name<input name="name" required maxlength="100" value="${esc(state.user.name)}"></label>${student ? `<label>Academic program<input name="academic_program" maxlength="120" value="${esc(profile.academic_program || "")}"></label><label>Branch<input name="branch" maxlength="120" value="${esc(profile.branch || "")}"></label><label>Semester<input name="semester" type="number" min="0" max="20" value="${Number(profile.semester || 0)}"></label><label>Academic year<input name="academic_year" maxlength="40" value="${esc(profile.academic_year || "")}"></label>` : `<label>Department<input name="department" maxlength="120" value="${esc(profile.department || "")}"></label><label>Teaching title<input name="teaching_title" maxlength="80" value="${esc(profile.teaching_title || "")}"></label>`}<p class="form-error" id="profile-error"></p><button class="button primary" type="submit">Save changes <span>→</span></button></form></article></section>`;
}

export function bindProfile(app, route) {
  app.querySelector("#profile-form").addEventListener("submit", async event => {
    event.preventDefault();
    const form = event.currentTarget, button = form.querySelector("button"), error = app.querySelector("#profile-error");
    button.disabled = true; error.textContent = "";
    try {
      const values = Object.fromEntries(new FormData(form)); values.semester = Number(values.semester || 0);
      state.user = (await api("/api/v1/auth/profile", { method: "PUT", body: JSON.stringify(values) })).user;
      notify("Profile saved");
      route();
    } catch (err) { error.textContent = err.message; button.disabled = false; }
  });
}
