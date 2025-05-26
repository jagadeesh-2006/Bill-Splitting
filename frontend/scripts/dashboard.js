
window.addEventListener('DOMContentLoaded', async () => {
  const userStr = localStorage.getItem('user');
  if (!userStr) {
    alert('Please login first');
    return window.location.href = 'login.html';
  }

  const user = JSON.parse(userStr);
  document.getElementById('username').textContent = user.username;

  const ul = document.getElementById('groups-ul');
  const groupDetailDiv = document.getElementById('group-detail');
  const groupNameEl = document.getElementById('group-name');
  const membersUl = document.getElementById('members-ul');
  const expensesUl = document.getElementById('expenses-ul');
  const expenseForm = document.getElementById('expense-form');
  const expensePayerSelect = document.getElementById('expense-payer');
  const backBtn = document.getElementById('back-btn');
  const splitInputsDiv = document.getElementById('split-inputs');

  let groups = [];
  let currentGroup = null;

  async function loadGroups() {
    const res = await fetch('http://localhost:5000/api/groups/user-groups/' + user._id);
    if (!res.ok) {
      alert('Error fetching groups');
      return;
    }
    groups = await res.json();

    ul.innerHTML = '';
    if (groups.length === 0) {
      ul.innerHTML = '<li>No groups found</li>';
    } else {
      groups.forEach(g => {
        const li = document.createElement('li');
        li.textContent = g.name;
        li.style.cursor = 'pointer';
        li.addEventListener('click', () => showGroupDetails(g));
        ul.appendChild(li);
      });
    }
  }

  function renderSplitInputs(splitType) {
    splitInputsDiv.innerHTML = '';
    if (!currentGroup) return;

    if (splitType === 'percentage' || splitType === 'custom') {
      currentGroup.members.forEach(m => {
        const div = document.createElement('div');
        div.style.marginBottom = '0.5rem';

        const label = document.createElement('label');
        label.htmlFor = `${splitType}-${m._id}`;
        label.textContent = `${m.username}: `;

        const input = document.createElement('input');
        input.type = 'number';
        input.min = '0';
        input.step = 'any';
        input.id = `${splitType}-${m._id}`;
        input.required = true;
        input.placeholder = splitType === 'percentage' ? 'Percentage %' : 'Amount';
        if (splitType === 'percentage') input.max = '100';

        div.appendChild(label);
        div.appendChild(input);
        splitInputsDiv.appendChild(div);
      });
    }
  }

  async function showGroupDetails(group) {
    currentGroup = group;
    groupNameEl.textContent = group.name;

    membersUl.innerHTML = '';
    group.members.forEach(m => {
      const li = document.createElement('li');
      li.textContent = `${m.username} (${m.phone || m.email || 'No contact'})`;
      membersUl.appendChild(li);
    });

    const res = await fetch(`http://localhost:5000/api/expenses/group/${group._id}`);
    expensesUl.innerHTML = '';
    if (!res.ok) {
      expensesUl.innerHTML = '<li>Error loading expenses</li>';
    } else {
      const expenses = await res.json();
      if (expenses.length === 0) {
        expensesUl.innerHTML = '<li>No expenses yet</li>';
      } else {
        expenses.forEach(exp => {
          const payer = group.members.find(m => m._id === exp.paidBy);
          const payerName = payer ? payer.username : 'Unknown';

          let splitInfo = '';
          if (exp.splitBetween?.length === exp.splitAmounts?.length) {
            splitInfo = exp.splitBetween.map((userId, idx) => {
              const user = group.members.find(m => m._id === userId);
              const name = user ? user.username : 'Unknown';
              return `${name}: $${exp.splitAmounts[idx].toFixed(2)}`;
            }).join(', ');
          }

          const li = document.createElement('li');
          li.textContent = `${exp.description}: $${exp.amount.toFixed(2)} paid by ${payerName}. Split => ${splitInfo}`;
          expensesUl.appendChild(li);
        });
      }
    }

    expensePayerSelect.innerHTML = '';
    group.members.forEach(m => {
      const option = document.createElement('option');
      option.value = m._id;
      option.textContent = m.username;
      expensePayerSelect.appendChild(option);
    });

    document.querySelectorAll('input[name="split-type"]').forEach(input => {
      input.checked = input.value === 'equal';
    });
    renderSplitInputs('equal');

    document.getElementById('groups-list').style.display = 'none';
    groupDetailDiv.style.display = 'block';
  }

  expenseForm.querySelectorAll('input[name="split-type"]').forEach(input => {
    input.addEventListener('change', e => {
      renderSplitInputs(e.target.value);
    });
  });

  expenseForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!currentGroup) return;

    const description = document.getElementById('expense-description').value.trim();
    const amount = parseFloat(document.getElementById('expense-amount').value);
    const payer = expensePayerSelect.value;
    const splitType = document.querySelector('input[name="split-type"]:checked')?.value;

    if (!description || isNaN(amount) || !payer || !splitType) {
      alert('Please fill all fields and select split type');
      return;
    }

    let splitBetween = [];
    let splitAmounts = [];

    if (splitType === 'equal') {
      splitBetween = currentGroup.members.map(m => m._id);
      const share = parseFloat((amount / splitBetween.length).toFixed(2));
      splitAmounts = splitBetween.map(() => share);
    } else if (splitType === 'percentage') {
      let totalPercent = 0;
      for (const m of currentGroup.members) {
        const val = parseFloat(document.getElementById(`percentage-${m._id}`)?.value || 0);
        if (isNaN(val) || val < 0) {
          alert(`Invalid percentage for ${m.username}`);
          return;
        }
        totalPercent += val;
        splitBetween.push(m._id);
        splitAmounts.push((val / 100) * amount);
      }
      if (Math.round(totalPercent) !== 100) {
        alert('Total percentage must be 100');
        return;
      }
      splitAmounts = splitAmounts.map(v => parseFloat(v.toFixed(2)));
    } else if (splitType === 'custom') {
      let total = 0;
      for (const m of currentGroup.members) {
        const val = parseFloat(document.getElementById(`custom-${m._id}`)?.value || 0);
        if (isNaN(val) || val < 0) {
          alert(`Invalid amount for ${m.username}`);
          return;
        }
        total += val;
        splitBetween.push(m._id);
        splitAmounts.push(val);
      }
      if (Math.abs(total - amount) > 0.01) {
        alert('Total split amounts must equal total amount');
        return;
      }
    }

    const res = await fetch('http://localhost:5000/api/expenses/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        groupId: currentGroup._id,
        description,
        amount,
        payer,
        splitBetween,
        splitAmounts,
      }),
    });

    if (!res.ok) {
      const data = await res.json();
      alert(data.message || 'Failed to add expense');
      return;
    }

    expenseForm.reset();
    renderSplitInputs('equal');

    const updatedGroupRes = await fetch('http://localhost:5000/api/groups/' + currentGroup._id);
    if (updatedGroupRes.ok) {
      currentGroup = await updatedGroupRes.json();
      showGroupDetails(currentGroup);
    } else {
      loadGroups();
      groupDetailDiv.style.display = 'none';
      document.getElementById('groups-list').style.display = 'block';
    }
  });

  backBtn.addEventListener('click', () => {
    groupDetailDiv.style.display = 'none';
    document.getElementById('groups-list').style.display = 'block';
  });

  document.getElementById('logout-btn').addEventListener('click', () => {
    localStorage.removeItem('user');
    window.location.href = 'login.html';
  });

  await loadGroups();
  document.getElementById('group-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('new-group-name').value.trim();
    const phonesRaw = document.getElementById('member-phones').value.trim();

    if (!name || !phonesRaw) {
      alert('Please provide group name and member phone numbers');
      return;
    }

    // Split and clean phone numbers
    const memberPhones = phonesRaw.split(',').map(p => p.trim()).filter(p => p.length > 0);
    
    if (memberPhones.length === 0) {
      alert('No valid phone numbers provided');
      return;
    }

    // Send correct payload keys expected by backend
    const res = await fetch('http://localhost:5000/api/groups/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        memberPhones,     // <-- this key is important to match backend
        creatorId: user._id  // <-- this key is important to match backend
      }),
    });

    const data = await res.json();
    if (!res.ok) {
      alert(data.message || 'Failed to create group');
    } else {
      alert('Group created successfully!');
      document.getElementById('group-form').reset();
      await loadGroups();
    }
  });

});
