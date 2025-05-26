document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const email = e.target.email.value.trim();
  const password = e.target.password.value.trim();

  const res = await fetch('http://localhost:5000/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json();

  const resultEl = document.getElementById('login-result');

  if (res.ok) {
    localStorage.setItem('user', JSON.stringify(data.user));
    resultEl.textContent = 'Login successful! Redirecting...';
    setTimeout(() => {
      window.location.href = 'dashboard.html';
    }, 1000);
  } else {
    resultEl.textContent = data.message || 'Login failed';
  }
});
