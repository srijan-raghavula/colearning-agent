import { api, esc, state } from "../core.js";

export function renderLanding() {
  return `<section class="landing"><div class="landing-copy"><p class="eyebrow">A calmer way to learn</p><h1>Turn every question into <em>momentum.</em></h1><p class="lede">CoLearn brings your teacher’s course map, grounded AI tutor, and learning evidence into one focused workspace.</p><div class="actions"><a class="button primary" href="/signup">Create workspace <span>→</span></a><a class="button secondary" href="/login">Sign in</a></div><div class="trust-row"><span>✓ Teacher-grounded</span><span>✓ Adaptive tutor</span><span>✓ Progress that persists</span></div></div><div class="product-preview"><div class="preview-bar"><span></span><span></span><span></span><b>Today’s learning plan</b></div><div class="preview-body"><div class="preview-kicker">NEXT BEST ACTION</div><h2>Understand the idea, then try it yourself.</h2><div class="preview-task"><span class="task-icon">✦</span><div><b>Linear equations</b><small>Continue with a guided explanation</small></div><strong>68%</strong></div><div class="preview-task muted-task"><span class="task-icon">○</span><div><b>Graphing equations</b><small>Ready when you are</small></div><strong>—</strong></div><div class="preview-footer"><span>AI tutor is grounded in your course</span><span class="pulse"></span></div></div></div></section><section class="principles"><article><span>01</span><h2>One shared context</h2><p>Teachers shape the source of truth. Learners get a focused path with no context switching.</p></article><article class="accent"><span>02</span><h2>Agentic by default</h2><p>The tutor can explain, check understanding, and recommend the next useful step.</p></article><article><span>03</span><h2>Designed for progress</h2><p>Every conversation turns into durable evidence that improves the next session.</p></article></section>`;
}

export function renderAuth(signup) {
  if (!signup) return `<section class="auth-layout"><div class="auth-intro"><p class="eyebrow">Welcome back</p><h1>Pick up where you left off.</h1><p class="lede">Your course, conversations, and progress are waiting in one place.</p></div><section class="auth-card"><span class="brand-mark">C</span><h2>Sign in</h2><p class="muted">Use your CoLearn account to continue.</p><form id="auth-form" class="form-grid"><label>Email<input name="email" type="email" required autocomplete="username"></label><label>Password<input name="password" type="password" required minlength="8" maxlength="72" autocomplete="current-password"></label><p class="form-error" id="auth-error"></p><button class="button primary full" type="submit">Sign in <span>→</span></button></form><p class="auth-switch">New to CoLearn? <a href="/signup">Create a workspace</a></p></section></section>`;
  return `<section class="auth-layout"><div class="auth-intro"><p class="eyebrow">Build your learning OS</p><h1>A workspace that adapts to your role.</h1><p class="lede">Start with a teacher-led course, then let the agent help you decide what to do next.</p></div><section class="auth-card wizard-card" data-signup><span class="brand-mark">C</span><div class="step-indicator"><span class="active"></span><span></span><span></span><span></span></div><h2 id="wizard-title">Choose your workspace</h2><p class="muted">A few details help us shape the right experience.</p><form id="auth-form" class="form-grid"></form><p class="auth-switch">Already have an account? <a href="/login">Sign in</a></p></section></section>`;
}

export function bindAuth(app, signup, route) {
  const form = app.querySelector("#auth-form");
  if (!signup) {
    form.addEventListener("submit", async event => {
      event.preventDefault();
      const button = form.querySelector("button"), error = app.querySelector("#auth-error");
      button.disabled = true; error.textContent = "";
      try {
        const result = await api("/api/auth/login", { method: "POST", body: JSON.stringify(Object.fromEntries(new FormData(form))) });
        location.assign(result.user.role === "teacher" ? "/app/teacher" : "/app/student");
      } catch (err) { error.textContent = err.message; button.disabled = false; }
    });
    return;
  }
  const wizard = { step: 1, data: { role: "student" } };
  const render = () => {
    const steps = wizard.data.role === "student" ? 5 : 4;
    const body = wizard.step === 1
      ? `<fieldset class="wizard-options"><legend>I am joining as a...</legend><label class="role-option"><input type="radio" name="role" value="student" ${wizard.data.role === "student" ? "checked" : ""}><span><b>Student</b><small>Learn from a course map and an adaptive tutor.</small></span></label><label class="role-option"><input type="radio" name="role" value="teacher" ${wizard.data.role === "teacher" ? "checked" : ""}><span><b>Teacher</b><small>Author courses and guide learner momentum.</small></span></label></fieldset>`
      : wizard.step === 2 ? `<label>Name<input name="name" required maxlength="100" autocomplete="name" value="${esc(wizard.data.name || "")}"></label><label>Email<input name="email" type="email" required autocomplete="email" value="${esc(wizard.data.email || "")}"></label>`
        : wizard.step === 3 ? `<label>Password<input name="password" type="password" required minlength="8" maxlength="72" autocomplete="new-password"></label><p class="muted">Use at least 8 characters.</p>`
          : wizard.data.role === "student" && wizard.step === 4 ? `<label>Academic program<input name="academic_program" required maxlength="120" placeholder="BSc Computer Science" value="${esc(wizard.data.academic_program || "")}"></label><label>Branch<input name="branch" required maxlength="120" placeholder="Systems" value="${esc(wizard.data.branch || "")}"></label>`
            : wizard.data.role === "student" ? `<label>Semester<input name="semester" type="number" min="1" max="20" required value="${esc(wizard.data.semester || "")}"></label><label>Academic year<input name="academic_year" required maxlength="40" placeholder="2026" value="${esc(wizard.data.academic_year || "")}"></label>`
              : `<label>Department<input name="department" required maxlength="120" placeholder="Mathematics" value="${esc(wizard.data.department || "")}"></label><label>Teaching title<input name="teaching_title" required maxlength="80" placeholder="Professor" value="${esc(wizard.data.teaching_title || "")}"></label>`;
    form.innerHTML = `${body}<p class="form-error" id="auth-error"></p><div class="wizard-actions">${wizard.step > 1 ? `<button class="button secondary" type="button" data-back>Back</button>` : ""}<button class="button primary" type="submit">${wizard.step === steps ? "Create workspace" : "Continue"} <span>→</span></button></div>`;
    app.querySelector("#wizard-title").textContent = wizard.step === 1 ? "Choose your workspace" : wizard.step === 2 ? "Tell us who you are" : wizard.step === 3 ? "Secure your account" : "Shape your workspace";
    app.querySelectorAll(".step-indicator span").forEach((item, index) => item.classList.toggle("active", index < wizard.step));
    form.addEventListener("submit", submit);
    form.querySelector("[data-back]")?.addEventListener("click", () => { wizard.step -= 1; render(); });
    form.querySelectorAll("input[name=role]").forEach(input => input.addEventListener("change", () => { wizard.data.role = input.value; wizard.step = 1; render(); }));
  };
  async function submit(event) {
    event.preventDefault();
    Object.assign(wizard.data, Object.fromEntries(new FormData(form)));
    if (wizard.step < steps) { wizard.step += 1; render(); return; }
    const button = form.querySelector("button[type=submit]"), error = app.querySelector("#auth-error");
    button.disabled = true; error.textContent = "";
    try {
      const result = await api("/api/auth/signup", { method: "POST", body: JSON.stringify({ ...wizard.data, semester: Number(wizard.data.semester || 0) }) });
      location.assign(result.user.role === "teacher" ? "/app/teacher" : "/app/student");
    } catch (err) { error.textContent = err.message; button.disabled = false; }
  }
  render();
}
