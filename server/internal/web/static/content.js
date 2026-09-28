const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);
let selectedId = 0;

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

function showError(message) { $('content-error').textContent = message || ''; }

function formatSchedule(screen) {
  if (!screen.start_at && !screen.end_at) return 'Permanent';
  return `${screen.start_at || 'Now'} → ${screen.end_at || 'No end'}`;
}

function renderRow(screen) {
  const row = document.createElement('tr');
  row.innerHTML = `<td>${escapeHTML(screen.slug)}</td><td>${escapeHTML(screen.title)}</td><td>${screen.published ? 'Yes' : 'Draft'}</td><td>${escapeHTML(formatSchedule(screen))}</td><td>${screen.version}</td><td><button class="secondary edit-screen">Edit</button> <button class="danger delete-screen">Delete</button></td>`;
  row.querySelector('.edit-screen').addEventListener('click', () => editScreen(screen));
  row.querySelector('.delete-screen').addEventListener('click', () => deleteScreen(screen));
  return row;
}

async function refreshScreens() {
  showError('');
  try {
    const result = await request('/api/admin/content/screens');
    const body = $('screens');
    body.innerHTML = '';
    for (const screen of result.screens || []) body.appendChild(renderRow(screen));
  } catch (error) {
    showError(error.message || String(error));
  }
}

function clearForm() {
  selectedId = 0;
  $('screen-detail-title').textContent = 'New Screen';
  $('screen-slug').value = '';
  $('screen-title').value = '';
  $('screen-priority').value = '0';
  $('screen-published').checked = true;
  $('screen-start').value = '';
  $('screen-end').value = '';
  $('screen-content').value = JSON.stringify({
    hero: { title: 'Welcome to Otter Link', body: 'Welcome to Otter Link.' },
    announcements: [],
    services: [],
    footer: ''
  }, null, 2);
  $('delete-screen').disabled = true;
}

function editScreen(screen) {
  selectedId = screen.id;
  $('screen-detail-title').textContent = `Edit ${screen.title}`;
  $('screen-slug').value = screen.slug;
  $('screen-title').value = screen.title;
  $('screen-priority').value = screen.priority;
  $('screen-published').checked = !!screen.published;
  $('screen-start').value = toLocalInput(screen.start_at);
  $('screen-end').value = toLocalInput(screen.end_at);
  $('screen-content').value = JSON.stringify(screen.content, null, 2);
  $('delete-screen').disabled = false;
  $('screen-panel').classList.remove('hidden');
}

function toLocalInput(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const pad = (value) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function toRFC3339(value) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
}

async function saveScreen() {
  showError('');
  let content;
  try {
    content = JSON.parse($('screen-content').value);
  } catch (error) {
    showError('Content must be valid JSON.');
    return;
  }

  try {
    await request('/api/admin/content/screens', {
      method: 'PUT',
      body: JSON.stringify({
        slug: $('screen-slug').value.trim(),
        title: $('screen-title').value.trim(),
        published: $('screen-published').checked,
        start_at: toRFC3339($('screen-start').value),
        end_at: toRFC3339($('screen-end').value),
        priority: Number($('screen-priority').value || 0),
        content
      })
    });
    await refreshScreens();
    clearForm();
    $('screen-panel').classList.add('hidden');
  } catch (error) {
    showError(error.message || String(error));
  }
}

async function deleteScreen(screen) {
  if (!confirm(`Delete screen "${screen.title}"?`)) return;
  try {
    await request(`/api/admin/content/screens/${screen.id}`, { method: 'DELETE' });
    await refreshScreens();
    if (selectedId === screen.id) {
      clearForm();
      $('screen-panel').classList.add('hidden');
    }
  } catch (error) {
    showError(error.message || String(error));
  }
}

function escapeHTML(value) {
  return String(value ?? '').replace(/[&<>"']/g, (char) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;'
  }[char]));
}

$('new-screen').addEventListener('click', () => {
  clearForm();
  $('screen-panel').classList.remove('hidden');
  $('screen-slug').focus();
});
$('close-screen').addEventListener('click', () => $('screen-panel').classList.add('hidden'));
$('save-screen').addEventListener('click', saveScreen);
$('delete-screen').addEventListener('click', () => {
  if (selectedId) deleteScreen({ id: selectedId, title: $('screen-title').value });
});
refreshScreens();
