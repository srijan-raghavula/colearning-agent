(() => {
  for (const form of document.querySelectorAll('[data-auth-form]')) {
    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const button = form.querySelector('[data-auth-submit]');
      const error = form.querySelector('[data-auth-error]');
      const originalLabel = button.textContent;
      button.disabled = true;
      button.textContent = 'Please wait…';
      error.textContent = '';

      try {
        const payload = Object.fromEntries(new FormData(form).entries());
        const response = await fetch(form.action, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify(payload),
        });
        const result = await response.json();
        if (!response.ok || !result.user) {
          throw new Error(result.error || 'We could not sign you in. Check your details and try again.');
        }
        window.location.assign(result.user.role === 'teacher' ? '/teacher' : '/student');
      } catch (err) {
        error.textContent = err instanceof Error ? err.message : 'Something went wrong. Please try again.';
        button.disabled = false;
        button.textContent = originalLabel;
      }
    });
  }
})();
