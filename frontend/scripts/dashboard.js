const API = 'http://localhost:8080'; // ← change to your Go server address

// ── AUTH CHECK ────────────────────────────────────────────────────────────
const userStr = localStorage.getItem('user');
if (!userStr) { alert('Please login first'); window.location.href = 'login.html'; }
const user = JSON.parse(userStr);
const token = localStorage.getItem('token');
document.getElementById('username').textContent = user.username;

// ── TOAST ─────────────────────────────────────────────────────────────────
function toast(msg, type = 'success') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = `show ${type}`;
  clearTimeout(el._t);
  el._t = setTimeout(() => el.className = '', 3000);
}

// ── AUTH HEADERS ──────────────────────────────────────────────────────────
function authHeaders() {
  return { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }; // 
}

// ── STATE ─────────────────────────────────────────────────────────────────
let groups = [];
let currentGroup = null;    // { id, name }
let currentMembers = [];    // models.Member[]
let currentExpenses = [];   // expense with splits[]

// ── TABS ──────────────────────────────────────────────────────────────────
document.querySelectorAll('.tab-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');
    if (btn.dataset.tab === 'balances') loadBalances();
    if (btn.dataset.tab === 'settle') { loadBalances(); loadHistory(); }
  });
});

// ── LOAD GROUPS ───────────────────────────────────────────────────────────
async function loadGroups() {
  const res = await fetch(`${API}/api/groups/creator/${user.id}`, { headers: authHeaders() });
   if (res.status === 401) {
    localStorage.removeItem('user');
    localStorage.removeItem('token');
    alert('Session expired. Please log in again.');
    window.location.href = 'login.html';
    return;
  }
  if (!res.ok) { toast('Error loading groups', 'error'); return; }
  groups = await res.json() || [];

  const ul = document.getElementById('groups-ul');
  ul.innerHTML = '';
  if (groups.length === 0) {
    ul.innerHTML = '<li style="cursor:default; color: var(--muted); font-size: 0.85rem;">No groups yet</li>';
    return;
  }
  groups.forEach(g => {
    const li = document.createElement('li');
    li.textContent = g.name;
    li.addEventListener('click', () => openGroup(g));
    ul.appendChild(li);
  });
}

// ── OPEN GROUP ────────────────────────────────────────────────────────────
async function openGroup(group) {
  currentGroup = group;

  // Mark active in sidebar
  document.querySelectorAll('#groups-ul li').forEach(li => {
    li.classList.toggle('active', li.textContent === group.name);
  });

  document.getElementById('empty-state').style.display = 'none';
  document.getElementById('group-detail').style.display = 'block';
  document.getElementById('group-name').textContent = group.name;

  // Reset to overview tab
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
  document.querySelector('[data-tab="overview"]').classList.add('active');
  document.getElementById('tab-overview').classList.add('active');

  await Promise.all([loadMembers(), loadExpenses()]);
}

// ── LOAD MEMBERS ──────────────────────────────────────────────────────────
async function loadMembers() {
  const res = await fetch(`${API}/api/groups/${currentGroup.id}/members`);
  if (!res.ok) { toast('Error loading members', 'error'); return; }
  currentMembers = await res.json() || [];

  // Render members list
  const ul = document.getElementById('members-ul');
  ul.innerHTML = '';
  currentMembers.forEach(m => {
    const li = document.createElement('li');
    li.innerHTML = `
          <div class="avatar">${m.name.charAt(0)}</div>
          <div class="member-info">
            <span class="member-name">${m.name}</span>
            <span class="member-phone">${m.phone}</span>
          </div>`;
    ul.appendChild(li);
  });

  // Populate payer select in add expense form
  const payerSel = document.getElementById('expense-payer');
  payerSel.innerHTML = currentMembers.map(m =>
    `<option value="${m.id}">${m.name}</option>`
  ).join('');

  // Populate settle selects
  ['settle-from', 'settle-to'].forEach(id => {
    document.getElementById(id).innerHTML = currentMembers.map(m =>
      `<option value="${m.id}">${m.name}</option>`
    ).join('');
  });

  renderSplitInputs(document.querySelector('input[name="split-type"]:checked')?.value || 'equal');
}

// ── LOAD EXPENSES ─────────────────────────────────────────────────────────
async function loadExpenses() {
  const res = await fetch(`${API}/api/expenses/${currentGroup.id}`);
  if (!res.ok) { toast('Error loading expenses', 'error'); return; }
  currentExpenses = await res.json() || [];

  const ul = document.getElementById('expenses-ul');
  ul.innerHTML = '';
  if (currentExpenses.length === 0) {
    ul.innerHTML = '<li style="color: var(--muted); font-size: 0.85rem; padding: 0.5rem 0;">No expenses yet</li>';
    return;
  }

  currentExpenses.slice(0, 8).forEach(exp => {
    const payer = currentMembers.find(m => m.id === exp.paidById);
    const li = document.createElement('li');
    li.innerHTML = `
          <div class="expense-info">
            <span class="expense-desc">${exp.description}</span>
            <span class="expense-meta">Paid by ${payer ? payer.name : 'Unknown'}</span>
          </div>
          <span class="expense-amount">₹${exp.amount.toFixed(2)}</span>`;
    ul.appendChild(li);
  });
}

// ── LOAD BALANCES ─────────────────────────────────────────────────────────
async function loadBalances() {
  if (!currentGroup) return;
  const res = await fetch(`${API}/api/groups/${currentGroup.id}/balances`);
  if (!res.ok) { toast('Error loading balances', 'error'); return; }
  const balances = await res.json() || [];

  const grid = document.getElementById('balances-grid');
  grid.innerHTML = '';

  // Also populate settle selects with balance info
  balances.forEach(b => {
    const cls = b.amount > 0.01 ? 'positive' : b.amount < -0.01 ? 'negative' : 'zero';
    const label = b.amount > 0.01 ? 'is owed' : b.amount < -0.01 ? 'owes' : 'settled up';
    const card = document.createElement('div');
    card.className = `balance-card ${cls}`;
    card.innerHTML = `
          <div class="balance-name">${b.memberName}</div>
          <div class="balance-amount">₹${Math.abs(b.amount).toFixed(2)}</div>
          <div class="balance-label">${label}</div>`;
    grid.appendChild(card);
  });

  if (balances.length === 0) {
    grid.innerHTML = '<p style="color: var(--muted); font-size: 0.875rem;">No balance data yet.</p>';
  }
}

// ── LOAD HISTORY ──────────────────────────────────────────────────────────
async function loadHistory() {
  if (!currentGroup) return;
  const res = await fetch(`${API}/api/groups/${currentGroup.id}/settlements`);
  if (!res.ok) { toast('Error loading history', 'error'); return; }
  const history = await res.json() || [];

  const container = document.getElementById('history-list');
  container.innerHTML = '';
  if (history.length === 0) {
    container.innerHTML = '<p style="color: var(--muted); font-size: 0.875rem;">No settlements yet.</p>';
    return;
  }

  history.forEach(s => {
    const date = new Date(s.paidAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
    const item = document.createElement('div');
    item.className = 'history-item';
    item.innerHTML = `
          <div class="history-arrow">
            <span class="history-from">${s.fromName}</span>
            <span class="history-arrow-icon">→</span>
            <span class="history-to">${s.toName}</span>
          </div>
          <div class="history-right">
            <div class="history-amount">₹${s.amount.toFixed(2)}</div>
            ${s.note ? `<div class="history-note">${s.note}</div>` : ''}
            <div class="history-date">${date}</div>
          </div>`;
    container.appendChild(item);
  });
}

// ── SPLIT INPUTS ──────────────────────────────────────────────────────────
function renderSplitInputs(splitType) {
  const container = document.getElementById('split-inputs');
  container.innerHTML = '';
  if (splitType === 'equal' || currentMembers.length === 0) return;

  currentMembers.forEach(m => {
    const row = document.createElement('div');
    row.className = 'split-member-row';
    row.innerHTML = `
          <span class="split-member-name">${m.name}</span>
          <input type="number" min="0" step="any"
            id="split-${m.id}"
            placeholder="${splitType === 'percentage' ? '% share' : '₹ amount'}"
            required />`;
    container.appendChild(row);
  });
}

document.querySelectorAll('input[name="split-type"]').forEach(r => {
  r.addEventListener('change', e => renderSplitInputs(e.target.value));
});

// ── ADD EXPENSE FORM ──────────────────────────────────────────────────────
document.getElementById('expense-form').addEventListener('submit', async e => {
  e.preventDefault();
  if (!currentGroup) return;

  const description = document.getElementById('expense-description').value.trim();
  const amount = parseFloat(document.getElementById('expense-amount').value);
  const paidById = parseInt(document.getElementById('expense-payer').value);
  const splitType = document.querySelector('input[name="split-type"]:checked').value;

  if (!description || isNaN(amount) || amount <= 0) {
    toast('Please fill all fields', 'error'); return;
  }

  let splitBetween = [];

  if (splitType === 'equal') {
    const share = parseFloat((amount / currentMembers.length).toFixed(2));
    splitBetween = currentMembers.map(m => ({ memberId: m.id, amountOwed: share }));
  } else if (splitType === 'percentage') {
    let total = 0;
    for (const m of currentMembers) {
      const pct = parseFloat(document.getElementById(`split-${m.id}`)?.value || 0);
      if (isNaN(pct) || pct < 0) { toast(`Invalid % for ${m.name}`, 'error'); return; }
      total += pct;
      splitBetween.push({ memberId: m.id, amountOwed: parseFloat(((pct / 100) * amount).toFixed(2)) });
    }
    if (Math.round(total) !== 100) { toast('Percentages must add up to 100', 'error'); return; }
  } else {
    let total = 0;
    for (const m of currentMembers) {
      const amt = parseFloat(document.getElementById(`split-${m.id}`)?.value || 0);
      if (isNaN(amt) || amt < 0) { toast(`Invalid amount for ${m.name}`, 'error'); return; }
      total += amt;
      splitBetween.push({ memberId: m.id, amountOwed: amt });
    }
    if (Math.abs(total - amount) > 0.01) { toast('Split amounts must equal total', 'error'); return; }
  }

  const btn = document.getElementById('add-expense-btn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';

  const res = await fetch(`${API}/api/expenses`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ groupId: currentGroup.id, description, amount, paidById, splitBetween }),
  });

  btn.disabled = false;
  btn.textContent = 'Add Expense';

  if (!res.ok) {
    const d = await res.json();
    toast(d.message || 'Failed to add expense', 'error'); return;
  }

  toast('Expense added!');
  document.getElementById('expense-form').reset();
  renderSplitInputs('equal');
  await loadExpenses();

  // Switch to overview
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
  document.querySelector('[data-tab="overview"]').classList.add('active');
  document.getElementById('tab-overview').classList.add('active');
});

// ── SETTLE UP FORM ────────────────────────────────────────────────────────
document.getElementById('settle-form').addEventListener('submit', async e => {
  e.preventDefault();
  if (!currentGroup) return;

  const fromMember = parseInt(document.getElementById('settle-from').value);
  const toMember = parseInt(document.getElementById('settle-to').value);
  const amount = parseFloat(document.getElementById('settle-amount').value);
  const note = document.getElementById('settle-note').value.trim();

  if (fromMember === toMember) { toast('Cannot settle with yourself', 'error'); return; }
  if (!amount || amount <= 0) { toast('Enter a valid amount', 'error'); return; }

  const btn = document.getElementById('settle-btn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';

  const res = await fetch(`${API}/api/groups/${currentGroup.id}/settle`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ fromMember, toMember, amount, note }),
  });

  btn.disabled = false;
  btn.textContent = 'Record Settlement';

  if (!res.ok) {
    const d = await res.json();
    toast(d.message || 'Failed to record settlement', 'error'); return;
  }

  toast('Settlement recorded!');
  document.getElementById('settle-form').reset();
  await Promise.all([loadBalances(), loadHistory()]);
});

// ── MEMBER INPUT ROWS (create group form) ─────────────────────────────────
let memberRowCount = 0;

function addMemberRow(name = '', phone = '') {
  memberRowCount++;
  const id = memberRowCount;
  const list = document.getElementById('members-input-list');
  const row = document.createElement('div');
  row.className = 'member-row';
  row.id = `member-row-${id}`;
  row.innerHTML = `
        <input type="text"  placeholder="Name"  class="member-name-input"  value="${name}"  required />
        <input type="tel"   placeholder="Phone" class="member-phone-input" value="${phone}" required />
        <button type="button" class="remove-member-btn" onclick="removeMemberRow(${id})">×</button>`;
  list.appendChild(row);
}

function removeMemberRow(id) {
  const row = document.getElementById(`member-row-${id}`);
  if (row) row.remove();
}

document.getElementById('add-member-btn').addEventListener('click', () => addMemberRow());

// Start with 2 empty rows
addMemberRow(); addMemberRow();

// ── CREATE GROUP FORM ─────────────────────────────────────────────────────
document.getElementById('group-form').addEventListener('submit', async e => {
  e.preventDefault();

  const name = document.getElementById('new-group-name').value.trim();
  const rows = document.querySelectorAll('.member-row');
  const members = [];

  for (const row of rows) {
    const n = row.querySelector('.member-name-input').value.trim();
    const p = row.querySelector('.member-phone-input').value.trim();
    if (!n || !p) { toast('Each member needs a name and phone', 'error'); return; }
    members.push({ name: n, phone: p });
  }

  if (!name) { toast('Group name is required', 'error'); return; }
  if (members.length < 1) { toast('Add at least one member', 'error'); return; }

  const btn = document.getElementById('create-group-btn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';

  const res = await fetch(`${API}/api/groups`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ name, members }),
  });

  btn.disabled = false;
  btn.textContent = 'Create group';

  if (!res.ok) {
    const d = await res.json();
    toast(d.message || 'Failed to create group', 'error'); return;
  }

  toast('Group created!');
  document.getElementById('group-form').reset();
  document.getElementById('members-input-list').innerHTML = '';
  memberRowCount = 0;
  addMemberRow(); addMemberRow();
  await loadGroups();
});

// ── BACK BUTTON ───────────────────────────────────────────────────────────
document.getElementById('back-btn').addEventListener('click', () => {
  document.getElementById('group-detail').style.display = 'none';
  document.getElementById('empty-state').style.display = 'flex';
  document.querySelectorAll('#groups-ul li').forEach(li => li.classList.remove('active'));
  currentGroup = null;
});

// ── LOGOUT ────────────────────────────────────────────────────────────────
document.getElementById('logout-btn').addEventListener('click', () => {
  localStorage.removeItem('user');
  localStorage.removeItem('token');
  window.location.href = 'login.html';
});

// ── INIT ──────────────────────────────────────────────────────────────────
loadGroups();
