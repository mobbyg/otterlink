const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);
let selectedUsername = '';

async function request(path, options = {}) {
  const headers = {
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  };
  const response = await fetch(path, { ...options, headers });
  if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
  if (response.status === 204) return null;
  return response.json();
}

function showError(message) { $('error').textContent = message || ''; }
function showDetailError(message) { $('detail-error').textContent = message || ''; }

function formatDate(value) {
  return value ? new Date(value).toLocaleString() : 'Never';
}

function renderUserRow(user) {
  const row = document.createElement('tr');
  row.className = 'admin-user-row';
  row.title = 'Open account details';
  row.innerHTML = `
    <td>${escapeHTML(user.username)}</td>
    <td>${escapeHTML(user.display_name)}</td>
    <td><span class="admin-badge">${escapeHTML(user.role)}</span></td>
    <td>${escapeHTML(user.status)}</td>
    <td><span class="presence-dot ${user.online ? 'online' : ''}"></span>${user.online ? 'Online' : 'Offline'}</td>
    <td>${user.session_count}</td>
    <td>${escapeHTML(formatDate(user.last_activity))}</td>`;
  row.addEventListener('click', () => openUser(user.username));
  return row;
}

async function refresh() {
  showError('');
  try {
    const result = await request('/api/admin/users');
    const body = $('users');
    body.innerHTML = '';
    for (const user of result.users || []) body.appendChild(renderUserRow(user));
  } catch (error) {
    showError(error.message || String(error));
  }
}

async function openUser(username) {
  showDetailError('');
  try {
    const user = await request(`/api/admin/users/${encodeURIComponent(username)}`);
    selectedUsername = user.username;
    $('detail-title').textContent = user.display_name || user.username;
    $('detail-username').value = user.username;
    $('detail-display-name').value = user.display_name || '';
    $('detail-email').value = user.email || '';
    $('detail-role').value = user.role;
    $('detail-status').value = user.status;
    $('detail-meta').innerHTML = `${user.online ? 'Online' : 'Offline'} • ${user.session_count} active session${user.session_count === 1 ? '' : 's'} • Last activity: ${escapeHTML(formatDate(user.last_activity))} • Created ${escapeHTML(formatDate(user.created_at))}`;
    $('user-panel').classList.remove('hidden');
  } catch (error) {
    showError(error.message || String(error));
  }
}

async function saveUser() {
  if (!selectedUsername) return;
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}`, {
      method: 'PATCH',
      body: JSON.stringify({
        display_name: $('detail-display-name').value,
        email: $('detail-email').value,
        role: $('detail-role').value,
        status: $('detail-status').value
      })
    });
    await refresh();
    await openUser(selectedUsername);
  } catch (error) {
    showDetailError(error.message || String(error));
  }
}

async function resetPassword() {
  if (!selectedUsername) return;
  const password = window.prompt(`Enter a new password for ${selectedUsername}. It must be at least 12 characters:`);
  if (password === null) return;
  if (password.length < 12) {
    showDetailError('Password must be at least 12 characters.');
    return;
  }
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}/password`, {
      method: 'POST',
      body: JSON.stringify({ password })
    });
    window.alert('Password reset. Existing sessions for this account have been signed out.');
    await refresh();
    await openUser(selectedUsername);
  } catch (error) {
    showDetailError(error.message || String(error));
  }
}

async function revokeSessions() {
  if (!selectedUsername) return;
  if (!window.confirm(`Sign out all active sessions for '${selectedUsername}'?`)) return;
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}/sessions/revoke`, { method: 'POST' });
    await refresh();
    await openUser(selectedUsername);
    window.alert('All active sessions have been signed out.');
  } catch (error) {
    showDetailError(error.message || String(error));
  }
}

async function deleteUser() {
  if (!selectedUsername) return;
  if (!window.confirm(`Delete the account '${selectedUsername}'? This cannot be undone.`)) return;
  const password = window.prompt('Enter your admin password to confirm account deletion:');
  if (password === null) return;
  if (!password) {
    showDetailError('Admin password is required to delete an account.');
    return;
  }

  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}`, {
      method: 'DELETE',
      body: JSON.stringify({ password })
    });
    closeDetail();
    await refresh();
  } catch (error) {
    showDetailError(error.message || String(error));
  }
}

function closeDetail() {
  selectedUsername = '';
  $('user-panel').classList.add('hidden');
  showDetailError('');
}

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (character) => ({
    '&':'&amp;', '<':'&lt;', '>':'&gt;', "'":'&#39;', '"':'&quot;'
  }[character]));
}

$('refresh').addEventListener('click', refresh);
$('close-detail').addEventListener('click', closeDetail);
$('save-user').addEventListener('click', saveUser);
$('reset-password').addEventListener('click', resetPassword);
$('revoke-sessions').addEventListener('click', revokeSessions);
$('delete-user').addEventListener('click', deleteUser);
refresh();
