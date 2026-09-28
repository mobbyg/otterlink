const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);
let selectedUsername = '';
let selectedChannel = null;

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


let selectedKeyword = '';

function showKeywordError(message) { $('keyword-error').textContent = message || ''; }

async function refreshKeywordTargets() {
  try {
    const channels = await request('/api/admin/chat/channels');
    const chat = $('keyword-chat');
    const currentChat = chat.value;
    chat.innerHTML = '<option value="">None</option>';
    for (const channel of channels.channels || []) {
      const option = document.createElement('option');
      option.value = channel.id;
      option.textContent = channel.name;
      chat.appendChild(option);
    }
    if ([...chat.options].some(o => o.value === currentChat)) chat.value = currentChat;
  } catch (error) { showKeywordError(error.message || String(error)); }

  try {
    const now = new Date();
    const result = await request(`/api/admin/events?year=${now.getFullYear()}&month=${now.getMonth()+1}`);
    const event = $('keyword-event');
    const currentEvent = event.value;
    event.innerHTML = '<option value="">None</option>';
    for (const item of result.events || []) {
      const option = document.createElement('option');
      option.value = item.id;
      option.textContent = `${item.title} — ${formatDate(item.start_at)}`;
      event.appendChild(option);
    }
    if ([...event.options].some(o => o.value === currentEvent)) event.value = currentEvent;
  } catch (error) { showKeywordError(error.message || String(error)); }
}

function keywordTargets(keyword) {
  return (keyword.targets || []).map(target => `${target.type}: ${target.label}`).join(', ') || 'None';
}

function renderKeywordRow(keyword) {
  const row = document.createElement('tr');
  row.innerHTML = `<td>${escapeHTML(keyword.keyword)}</td><td>${escapeHTML(keyword.display_name)}</td><td>${escapeHTML(keyword.description || '')}</td><td>${escapeHTML(keywordTargets(keyword))}</td><td><button class="secondary keyword-edit">Edit</button> <button class="danger keyword-delete">Delete</button></td>`;
  row.querySelector('.keyword-edit').addEventListener('click', () => editKeyword(keyword));
  row.querySelector('.keyword-delete').addEventListener('click', () => deleteKeyword(keyword));
  return row;
}

async function refreshKeywords() {
  showKeywordError('');
  try {
    const result = await request('/api/admin/keywords');
    const body = $('keywords'); body.innerHTML = '';
    for (const keyword of result.keywords || []) body.appendChild(renderKeywordRow(keyword));
  } catch (error) { showKeywordError(error.message || String(error)); }
}

function clearKeywordForm() {
  selectedKeyword = '';
  $('keyword-name').value = '';
  $('keyword-display-name').value = '';
  $('keyword-description').value = '';
  $('keyword-chat').value = '';
  $('keyword-event').value = '';
  $('keyword-name').disabled = false;
  showKeywordError('');
}

function editKeyword(keyword) {
  selectedKeyword = keyword.keyword;
  $('keyword-name').value = keyword.keyword;
  $('keyword-name').disabled = true;
  $('keyword-display-name').value = keyword.display_name || '';
  $('keyword-description').value = keyword.description || '';
  $('keyword-chat').value = '';
  $('keyword-event').value = '';
  for (const target of keyword.targets || []) {
    if (target.type === 'chat') $('keyword-chat').value = String(target.id);
    if (target.type === 'event') $('keyword-event').value = String(target.id);
  }
  showKeywordError('');
  $('keywords-section').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

async function saveKeyword() {
  const keyword = $('keyword-name').value.trim();
  const displayName = $('keyword-display-name').value.trim();
  const description = $('keyword-description').value.trim();
  const targets = [];
  if ($('keyword-chat').value) {
    const option = $('keyword-chat').selectedOptions[0];
    targets.push({ type: 'chat', id: Number($('keyword-chat').value), label: option.textContent });
  }
  if ($('keyword-event').value) {
    const option = $('keyword-event').selectedOptions[0];
    targets.push({ type: 'event', id: Number($('keyword-event').value), label: option.textContent });
  }
  if (!keyword || !displayName) {
    showKeywordError('Keyword and display name are required.');
    return;
  }
  if (!targets.length) {
    showKeywordError('Select at least one existing service.');
    return;
  }
  showKeywordError('');
  try {
    await request(`/api/admin/keywords/${encodeURIComponent(keyword)}`, {
      method: 'PUT',
      body: JSON.stringify({ display_name: displayName, description, targets })
    });
    clearKeywordForm();
    await refreshKeywords();
    await refreshKeywordTargets();
    await refreshAudit();
  } catch (error) { showKeywordError(error.message || String(error)); }
}

async function deleteKeyword(keyword) {
  if (!window.confirm(`Delete service keyword '${keyword.keyword}'?`)) return;
  showKeywordError('');
  try {
    await request(`/api/admin/keywords/${encodeURIComponent(keyword.keyword)}`, { method: 'DELETE' });
    if (selectedKeyword === keyword.keyword) clearKeywordForm();
    await refreshKeywords();
    await refreshAudit();
  } catch (error) { showKeywordError(error.message || String(error)); }
}

function showEventError(message) { $('event-error').textContent = message || ''; }
function localEventValue(value) { if (!value) return ''; return new Date(value).toISOString(); }
function eventWhen(event) {
  if (event.all_day) return event.start_at === event.end_at ? event.start_at : event.start_at + ' → ' + event.end_at;
  return formatDate(event.start_at) + ' → ' + formatDate(event.end_at);
}
function renderEventRow(event) {
  const row = document.createElement('tr');
  row.innerHTML = `<td>${escapeHTML(eventWhen(event))}</td><td>${escapeHTML(event.title)}</td><td>${escapeHTML(event.description || '')}</td><td>${escapeHTML(event.created_by)}</td><td><button class="secondary delete-event">Delete</button></td>`;
  row.querySelector('.delete-event').addEventListener('click', async () => {
    if (!window.confirm(`Delete server event '${event.title}'?`)) return;
    try { await request(`/api/admin/events/${event.id}`, { method: 'DELETE' }); await refreshEvents(); }
    catch (error) { showEventError(error.message || String(error)); }
  });
  return row;
}
async function refreshEvents() {
  showEventError('');
  const now = new Date();
  const monthValue = $('event-month').value || `${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}`;
  const [year, month] = monthValue.split('-').map(Number);
  try {
    const result = await request(`/api/admin/events?year=${year}&month=${month}`);
    const body = $('events'); body.innerHTML = '';
    for (const event of result.events || []) body.appendChild(renderEventRow(event));
  } catch (error) { showEventError(error.message || String(error)); }
}
async function createEvent() {
  const title = $('event-title').value.trim();
  const description = $('event-description').value.trim();
  const allDay = $('event-all-day').checked;
  const start = $('event-start').value;
  const end = $('event-end').value;
  if (!title || !start || !end) { showEventError('Title, start, and end are required.'); return; }
  const payload = { title, description, all_day: allDay };
  if (allDay) {
    payload.start_at = start.slice(0,10); payload.end_at = end.slice(0,10);
  } else {
    payload.start_at = localEventValue(start); payload.end_at = localEventValue(end);
  }
  try {
    await request('/api/admin/events', { method: 'POST', body: JSON.stringify(payload) });
    $('event-title').value=''; $('event-description').value=''; $('event-start').value=''; $('event-end').value='';
    await refreshEvents(); await refreshAudit();
  } catch (error) { showEventError(error.message || String(error)); }
}

function showError(message) { $('error').textContent = message || ''; }
function showDetailError(message) { $('detail-error').textContent = message || ''; }
function showAuditError(message) { $('audit-error').textContent = message || ''; }
function showChannelError(message) { $('channel-error').textContent = message || ''; }
function showChannelDetailError(message) { $('channel-detail-error').textContent = message || ''; }
function formatDate(value) { return value ? new Date(value).toLocaleString() : 'Never'; }

function renderUserRow(user) {
  const row = document.createElement('tr');
  row.className = 'admin-user-row';
  row.title = 'Open account details';
  row.innerHTML = `<td>${escapeHTML(user.username)}</td><td>${escapeHTML(user.display_name)}</td><td><span class="admin-badge">${escapeHTML(user.role)}</span></td><td>${escapeHTML(user.status)}</td><td><span class="presence-dot ${user.online ? 'online' : ''}"></span>${user.online ? 'Online' : 'Offline'}</td><td>${user.session_count}</td><td>${escapeHTML(formatDate(user.last_activity))}</td>`;
  row.addEventListener('click', () => openUser(user.username));
  return row;
}

function renderAuditRow(event) {
  const row = document.createElement('tr');
  row.innerHTML = `<td>${escapeHTML(formatDate(event.created_at))}</td><td>${escapeHTML(event.actor_username)}</td><td><span class="admin-badge">${escapeHTML(event.action)}</span></td><td>${escapeHTML(event.target_username || '—')}</td><td>${escapeHTML(event.result)}</td><td>${escapeHTML(event.details || '')}</td>`;
  return row;
}

function renderChannelRow(channel) {
  const row = document.createElement('tr');
  row.innerHTML = `<td>${escapeHTML(channel.name)}</td><td>${escapeHTML(channel.creator || '—')}</td><td>${escapeHTML(channel.original_mod || '—')}</td><td>${channel.allow_ops_to_create_ops ? 'Yes' : 'No'}</td><td><button class="secondary channel-manage">Manage</button></td>`;
  row.querySelector('.channel-manage').addEventListener('click', () => openChannel(channel));
  return row;
}

async function refresh() {
  showError('');
  try {
    const result = await request('/api/admin/users');
    const body = $('users'); body.innerHTML = '';
    for (const user of result.users || []) body.appendChild(renderUserRow(user));
  } catch (error) { showError(error.message || String(error)); }
}

async function refreshAudit() {
  showAuditError('');
  try {
    const result = await request('/api/admin/audit');
    const body = $('audit-events'); body.innerHTML = '';
    for (const event of result.events || []) body.appendChild(renderAuditRow(event));
  } catch (error) { showAuditError(error.message || String(error)); }
}

async function refreshChannels() {
  showChannelError('');
  try {
    const result = await request('/api/admin/chat/channels');
    const body = $('channels'); body.innerHTML = '';
    for (const channel of result.channels || []) body.appendChild(renderChannelRow(channel));
  } catch (error) { showChannelError(error.message || String(error)); }
}

async function createChannel() {
  const name = $('channel-name').value.trim();
  if (!name) { showChannelError('Enter a channel name.'); return; }
  showChannelError('');
  try {
    await request('/api/admin/chat/channels', { method: 'POST', body: JSON.stringify({ name, allow_ops_to_create_ops: $('channel-allow-ops').checked }) });
    $('channel-name').value = '';
    $('channel-allow-ops').checked = false;
    await refreshChannels(); await refreshAudit();
  } catch (error) { showChannelError(error.message || String(error)); }
}

function openChannel(channel) {
  selectedChannel = channel;
  $('channel-detail-title').textContent = channel.name;
  $('channel-user').value = '';
  $('channel-role').value = 'mod';
  showChannelDetailError('');
  $('channel-panel').classList.remove('hidden');
}

function closeChannel() {
  selectedChannel = null;
  $('channel-panel').classList.add('hidden');
  showChannelDetailError('');
}

async function saveChannelRole() {
  if (!selectedChannel) return;
  const username = $('channel-user').value.trim();
  if (!username) { showChannelDetailError('Enter a username.'); return; }
  try {
    await request(`/api/admin/chat/channels/${selectedChannel.id}/users/${encodeURIComponent(username)}/role`, { method: 'POST', body: JSON.stringify({ role: $('channel-role').value }) });
    await refreshChannels(); await refreshAudit();
    window.alert('Channel role updated.');
  } catch (error) { showChannelDetailError(error.message || String(error)); }
}

async function deleteChannel() {
  if (!selectedChannel) return;
  if (!window.confirm(`Delete the permanent channel '${selectedChannel.name}'? This cannot be undone.`)) return;
  try {
    await request(`/api/admin/chat/channels/${selectedChannel.id}`, { method: 'DELETE' });
    closeChannel(); await refreshChannels(); await refreshAudit();
  } catch (error) { showChannelDetailError(error.message || String(error)); }
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
  } catch (error) { showError(error.message || String(error)); }
}

async function saveUser() {
  if (!selectedUsername) return;
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}`, { method: 'PATCH', body: JSON.stringify({ display_name: $('detail-display-name').value, email: $('detail-email').value, role: $('detail-role').value, status: $('detail-status').value }) });
    await refresh(); await refreshAudit(); await openUser(selectedUsername);
  } catch (error) { showDetailError(error.message || String(error)); }
}

async function resetPassword() {
  if (!selectedUsername) return;
  const password = window.prompt(`Enter a new password for ${selectedUsername}. It must be at least 12 characters:`);
  if (password === null) return;
  if (password.length < 12) { showDetailError('Password must be at least 12 characters.'); return; }
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}/password`, { method: 'POST', body: JSON.stringify({ password }) });
    window.alert('Password reset. Existing sessions for this account have been signed out.');
    await refresh(); await refreshAudit(); await openUser(selectedUsername);
  } catch (error) { showDetailError(error.message || String(error)); }
}

async function revokeSessions() {
  if (!selectedUsername) return;
  if (!window.confirm(`Sign out all active sessions for '${selectedUsername}'?`)) return;
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}/sessions/revoke`, { method: 'POST' });
    await refresh(); await refreshAudit(); await openUser(selectedUsername); window.alert('All active sessions have been signed out.');
  } catch (error) { showDetailError(error.message || String(error)); }
}

async function deleteUser() {
  if (!selectedUsername) return;
  if (!window.confirm(`Delete the account '${selectedUsername}'? This cannot be undone.`)) return;
  const password = window.prompt('Enter your admin password to confirm account deletion:');
  if (password === null) return;
  if (!password) { showDetailError('Admin password is required to delete an account.'); return; }
  showDetailError('');
  try {
    await request(`/api/admin/users/${encodeURIComponent(selectedUsername)}`, { method: 'DELETE', body: JSON.stringify({ password }) });
    closeDetail(); await refresh(); await refreshAudit();
  } catch (error) { showDetailError(error.message || String(error)); await refreshAudit(); }
}

function closeDetail() { selectedUsername = ''; $('user-panel').classList.add('hidden'); showDetailError(''); }
function escapeHTML(value) { return String(value ?? '').replace(/[&<>'"]/g, (character) => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', "'":'&#39;', '"':'&quot;' }[character])); }

$('refresh').addEventListener('click', refresh);
$('refresh-audit').addEventListener('click', refreshAudit);
$('refresh-channels').addEventListener('click', refreshChannels);
$('create-channel').addEventListener('click', createChannel);
$('close-detail').addEventListener('click', closeDetail);
$('save-user').addEventListener('click', saveUser);
$('reset-password').addEventListener('click', resetPassword);
$('revoke-sessions').addEventListener('click', revokeSessions);
$('delete-user').addEventListener('click', deleteUser);
$('close-channel-detail').addEventListener('click', closeChannel);
$('save-channel-role').addEventListener('click', saveChannelRole);
$('delete-channel').addEventListener('click', deleteChannel);
$('refresh-events').addEventListener('click', refreshEvents);
$('event-month').addEventListener('change', refreshEvents);
$('create-event').addEventListener('click', createEvent);
refresh();
refreshChannels();
refreshAudit();

refreshEvents();

$('refresh-keywords').addEventListener('click', async () => { await refreshKeywordTargets(); await refreshKeywords(); });
$('save-keyword').addEventListener('click', saveKeyword);
$('clear-keyword').addEventListener('click', clearKeywordForm);
refreshKeywordTargets();
refreshKeywords();
