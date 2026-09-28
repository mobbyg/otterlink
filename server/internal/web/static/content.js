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


function showAssetError(message) { $('asset-error').textContent = message || ''; }

async function previewScreen() {
  showError('');
  let content;
  try {
    content = JSON.parse($('screen-content').value);
  } catch (error) {
    showError('Content must be valid JSON before it can be previewed.');
    return;
  }
  $('screen-preview-title').textContent = $('screen-title').value.trim() || 'Screen Preview';
  renderPreview(content);
  $('screen-preview').classList.remove('hidden');
}

function renderPreview(content) {
  const canvas = $('screen-preview-canvas');
  canvas.innerHTML = '';
  canvas.style.backgroundImage = '';
  canvas.style.backgroundSize = '';
  canvas.style.backgroundPosition = '';
  const background = content.background || {};
  const assetID = Number(background.asset || 0);
  if (assetID > 0) {
    loadPreviewBackground(assetID, canvas, background.fit || 'cover');
  }
  const hero = content.hero || {};
  const heroEl = document.createElement('section');
  heroEl.className = 'preview-hero';
  if (hero.icon) {
    const icon = document.createElement('div');
    icon.className = 'preview-hero-icon';
    icon.textContent = hero.icon;
    heroEl.appendChild(icon);
  }
  const heroText = document.createElement('div');
  const title = document.createElement('h3');
  title.textContent = hero.title || 'Welcome to Otter Link';
  const body = document.createElement('p');
  body.textContent = hero.body || '';
  heroText.append(title, body);
  heroEl.appendChild(heroText);
  canvas.appendChild(heroEl);

  const announcements = Array.isArray(content.announcements) ? content.announcements : [];
  if (announcements.length) {
    const section = document.createElement('section');
    section.className = 'preview-section';
    const heading = document.createElement('h4');
    heading.textContent = "What's New";
    section.appendChild(heading);
    for (const item of announcements) {
      const card = document.createElement('div');
      card.className = 'preview-announcement';
      const text = document.createElement('div');
      const title = document.createElement('strong');
      title.textContent = item.title || '';
      const body = document.createElement('p');
      body.textContent = item.body || '';
      text.append(title, body);
      card.appendChild(text);
      if (item.action_label) {
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = item.action_label;
        button.disabled = true;
        card.appendChild(button);
      }
      section.appendChild(card);
    }
    canvas.appendChild(section);
  }

  const services = Array.isArray(content.services) ? content.services : [];
  if (services.length) {
    const section = document.createElement('section');
    section.className = 'preview-section';
    const heading = document.createElement('h4');
    heading.textContent = 'Explore Otter Link';
    section.appendChild(heading);
    const grid = document.createElement('div');
    grid.className = 'preview-services';
    for (const item of services) {
      const tile = document.createElement('div');
      tile.className = 'preview-service';
      const title = document.createElement('strong');
      title.textContent = (item.icon ? item.icon + '  ' : '') + (item.title || '');
      const description = document.createElement('p');
      description.textContent = item.description || '';
      tile.append(title, description);
      grid.appendChild(tile);
    }
    section.appendChild(grid);
    canvas.appendChild(section);
  }

  if (content.footer) {
    const footer = document.createElement('div');
    footer.className = 'preview-footer';
    footer.textContent = content.footer;
    canvas.appendChild(footer);
  }
}

async function loadPreviewBackground(assetID, canvas, fit) {
  try {
    const response = await fetch('/api/content/assets/' + assetID, {
      headers: token ? { Authorization: 'Bearer ' + token } : {}
    });
    if (!response.ok) throw new Error('Unable to load background asset');
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    canvas.style.backgroundImage = 'url("' + url + '")';
    canvas.style.backgroundSize = fit === 'contain' ? 'contain' : 'cover';
    canvas.style.backgroundPosition = 'center';
    canvas.style.backgroundRepeat = 'no-repeat';
  } catch (error) {
    showError(error.message || String(error));
  }
}

function formatBytes(size) {
  const value = Number(size) || 0;
  if (value < 1024) return value + ' B';
  if (value < 1024 * 1024) return (value / 1024).toFixed(1) + ' KB';
  return (value / (1024 * 1024)).toFixed(1) + ' MB';
}

function renderAssetRow(asset) {
  const row = document.createElement('tr');
  row.innerHTML = '<td>' + escapeHTML(asset.name) + '</td>' +
    '<td>' + escapeHTML(asset.mime) + '</td>' +
    '<td>' + formatBytes(asset.size) + '</td>' +
    '<td><code>' + escapeHTML(asset.sha256.slice(0, 16)) + '…</code></td>' +
    '<td><button class="danger delete-asset">Delete</button></td>';
  row.querySelector('.delete-asset').addEventListener('click', () => deleteAsset(asset));
  return row;
}

async function refreshAssets() {
  showAssetError('');
  try {
    const result = await request('/api/admin/content/assets');
    const body = $('assets');
    body.innerHTML = '';
    for (const asset of result.assets || []) body.appendChild(renderAssetRow(asset));
  } catch (error) {
    showAssetError(error.message || String(error));
  }
}

async function uploadAsset() {
  showAssetError('');
  const input = $('asset-file');
  const file = input.files && input.files[0];
  if (!file) {
    showAssetError('Choose an image first.');
    return;
  }

  const form = new FormData();
  form.append('asset', file, file.name);
  try {
    const response = await fetch('/api/admin/content/assets', {
      method: 'POST',
      headers: token ? { Authorization: 'Bearer ' + token } : {},
      body: form
    });
    if (!response.ok) throw new Error((await response.text()) || 'Upload failed');
    input.value = '';
    await refreshAssets();
  } catch (error) {
    showAssetError(error.message || String(error));
  }
}

async function deleteAsset(asset) {
  if (!confirm('Delete asset "' + asset.name + '"?')) return;
  showAssetError('');
  try {
    await request('/api/admin/content/assets/' + asset.id, { method: 'DELETE' });
    await refreshAssets();
  } catch (error) {
    showAssetError(error.message || String(error));
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
function closeEditor() {
  $('screen-panel').classList.add('hidden');
}
$('close-screen').addEventListener('click', closeEditor);
$('screen-panel').querySelector('.screen-modal-backdrop').addEventListener('click', closeEditor);
document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && !$('screen-preview').classList.contains('hidden')) $('screen-preview').classList.add('hidden');
  else if (event.key === 'Escape' && !$('screen-panel').classList.contains('hidden')) closeEditor();
});
$('save-screen').addEventListener('click', saveScreen);
$('preview-screen').addEventListener('click', previewScreen);
$('close-preview').addEventListener('click', () => $('screen-preview').classList.add('hidden'));
$('screen-preview').querySelector('.screen-preview-backdrop').addEventListener('click', () => $('screen-preview').classList.add('hidden'));
$('delete-screen').addEventListener('click', () => {
  if (selectedId) deleteScreen({ id: selectedId, title: $('screen-title').value });
});
refreshScreens();

$('upload-asset').addEventListener('click', uploadAsset);
refreshAssets();
