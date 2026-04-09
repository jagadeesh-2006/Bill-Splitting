const API = 'http://localhost:8080';

// If already logged in, skip straight to dashboard
if (localStorage.getItem('token')) {
  window.location.href = 'dashboard.html';
}

document.getElementById('register-form').addEventListener('submit', async (e) => {
  e.preventDefault();

  const username = document.getElementById('username').value.trim();
  const phone    = document.getElementById('phone').value.trim();
  const email    = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  const result   = document.getElementById('register-result');
  const btn      = e.target.querySelector('.btn-submit');

  result.className = '';
  result.textContent = '';

  // Basic validation
  if (!username || !phone || !email || !password) {
    result.className = 'error';
    result.textContent = 'Please fill in all fields.';
    return;
  }

  if (password.length < 6) {
    result.className = 'error';
    result.textContent = 'Password must be at least 6 characters.';
    return;
  }

  // Basic phone check — digits only, 7–15 chars
  if (!/^\+?[0-9]{10}$/.test(phone)) {
    result.className = 'error';
    result.textContent = 'Enter a valid phone number (digits only, 10 chars).';
    return;
  }

  btn.textContent = 'Creating account…';
  btn.disabled = true;

  try {
    const res = await fetch(`${API}/api/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, phone, email, password }),
    });

    const data = await res.json();

    if (!res.ok) {
      result.className = 'error';
      result.textContent = data.message || 'Registration failed. Please try again.';
      return;
    }

    result.className = 'success';
    result.textContent = 'Account created! Redirecting to login…';

    setTimeout(() => window.location.href = 'login.html', 1000);

  } catch (err) {
    result.className = 'error';
    result.textContent = 'Could not reach server. Is it running?';
  } finally {
    btn.textContent = 'Create account';
    btn.disabled = false;
  }
});