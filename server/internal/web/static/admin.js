const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);

async function request(path) {
  const headers = token ? { Authorization: `Bearer ${token}` } : {};
  const response = await fetch(path, { headers });
  if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
  return response.json();
}

function showError(message) {
  $('error').textContent = message || '';
}

async function refresh() {
  showError('');
  try {
    const result = await request('/api/admin/users');
    const body = $('users');
    body.innerHTML = '';
    for (const user of result.users || []) {
      const row = document.createElement('tr');
      row.innerHTML = `
        <td>${escapeHTML(user.username)}</td>
        <td>${escapeHTML(user.display_name)}</td>
        <td><span class="admin-badge">${escapeHTML(user.role)}</span></td>
        <td>${escapeHTML(user.status)}</td>
        <td><span class="presence-dot ${user.online ? 'online' : ''}"></span>${user.online ? 'Online' : 'Offline'}</td>
        <td>${escapeHTML(new Date(user.created_at).toLocaleString())}</td>`;
      body.appendChild(row);
    }
  } catch (error) {
    showError(error.message || String(error));
  }
}

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (character) => ({
    '&':'&amp;', '<':'&lt;', '>':'&gt;', "'":'&#39;', '"':'&quot;'
  }[character]));
}

$('refresh').addEventListener('click', refresh);
refresh();
