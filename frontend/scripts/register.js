document.getElementById('register-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const username = e.target.username.value.trim();
  const email = e.target.email.value.trim();
  const password = e.target.password.value.trim();
  const phone = e.target.phone.value.trim();

  const res = await fetch('http://localhost:5000/api/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, email, password, phone }),
  });

  const data = await res.json();

  const resultEl = document.getElementById('register-result');
  if (res.ok) {
    resultEl.textContent = 'Registration successful! You can now login.';
    e.target.reset();
  } else {
    resultEl.textContent = data.message || 'Registration failed';
  }
});
