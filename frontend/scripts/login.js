const API = 'http://localhost:8080';

// If already logged in, skip straight to dashboard
if (localStorage.getItem('token')) {
  window.location.href = 'dashboard.html';
}

document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();

  const email    = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  const result   = document.getElementById('login-result');
  const btn      = e.target.querySelector('.btn-submit');

  result.className = '';
  result.textContent = '';

  if (!email || !password) {
    result.className = 'error';
    result.textContent = 'Please fill in all fields.';
    return;
  }

  btn.textContent = 'Logging in…';
  btn.disabled = true;

  try {
    const res = await fetch(`${API}/api/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    const data = await res.json();

    if (!res.ok) {
      result.className = 'error';
      result.textContent = data.message || 'Login failed. Please try again.';
      return;
    }

    // Save token and user to localStorage — dashboard reads these
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));

    result.className = 'success';
    result.textContent = 'Logged in! Redirecting…';

    setTimeout(() => window.location.href = 'dashboard.html', 600);

  } catch (err) {
    result.className = 'error';
    result.textContent = 'Could not reach server. Is it running?';
  } finally {
    btn.textContent = 'Log in';
    btn.disabled = false;
  }
});