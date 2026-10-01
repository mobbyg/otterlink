const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);
let selectedId = 0;
let assets = [];
let editorContent = null;
let editorSelectedIndex = -1;
let editorDrag = null;
let editorResize = null;
let editorButtonColor = '#efa00b';

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

function formatBytes(bytes) {
  const value = Number(bytes);
  if (!Number.isFinite(value) || value < 0) return '—';
  if (value < 1024) return value + ' B';
  if (value < 1024 * 1024) return (value / 1024).toFixed(1) + ' KB';
  if (value < 1024 * 1024 * 1024) return (value / (1024 * 1024)).toFixed(1) + ' MB';
  return (value / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
}

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

function defaultContent() {
  return {
    hero: { title: 'Welcome to Otter Link', body: 'Welcome to Otter Link.' },
    announcements: [],
    services: [],
    footer: ''
  };
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
  $('screen-content').value = JSON.stringify(defaultContent(), null, 2);
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

function parseEditorJSON() {
  try {
    const content = JSON.parse($('screen-content').value);
    return content && typeof content === 'object' ? content : {};
  } catch (error) {
    showError('Content must be valid JSON before using the visual editor.');
    return null;
  }
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
  let content = parseEditorJSON();
  if (!content) return;
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
  canvas.style.backgroundRepeat = '';
  const background = content.background || {};
  const assetID = Number(background.asset || 0);
  const elements = Array.isArray(content.elements) ? content.elements : [];
  canvas.classList.toggle('template-preview', elements.length > 0);
  canvas.style.position = 'relative';
  canvas.style.padding = elements.length > 0 ? '0' : '';
  if (assetID > 0) loadPreviewBackground(assetID, canvas, background.fit || 'cover');
  if (elements.length > 0) {
    renderPreviewElements(elements, canvas);
    return;
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
      const tile = document.createElement('div');      tile.className = 'preview-service';
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

function clamp01(value, fallback = 0) {
  const number = Number(value);
  if (!Number.isFinite(number)) return fallback;
  return Math.max(0, Math.min(1, number));
}

function clampSize(value, fallback = 0.1) {
  const number = Number(value);
  if (!Number.isFinite(number)) return fallback;
  return Math.max(0.01, Math.min(1, number));
}

function renderPreviewElements(elements, canvas) {
  for (const item of elements) {
    const type = String(item.type || '').toLowerCase();
    if (type !== 'text' && type !== 'button' && type !== 'image') continue;
    const element = document.createElement(type === 'button' ? 'button' : type === 'image' ? 'img' : 'div');
    if (type === 'button') {
      element.type = 'button';
      element.disabled = true;
      const assetID = Number(item.asset || 0);
      if (assetID > 0) {
        const image = document.createElement('img');
        image.alt = item.alt || item.text || '';
        image.draggable = false;
        loadPreviewAssetImage(assetID, image);
        image.style.width = '100%';
        image.style.height = '100%';
        image.style.objectFit = item.fit === 'cover' ? 'cover' : 'contain';
        image.style.display = 'block';
        image.style.pointerEvents = 'none';
        element.textContent = '';
        element.appendChild(image);
      } else {
        element.textContent = item.text || '';
        element.style.background = item.background || '#efa00b';
        element.style.color = item.color || '#591f0a';
        element.style.border = '1px solid rgba(255,255,255,.35)';
        element.style.borderRadius = '8px';
      }
    }
    if (type === 'image') {
      const assetID = Number(item.asset || 0);
      if (assetID < 1) continue;
      element.alt = item.alt || '';
      element.draggable = false;
      loadPreviewAssetImage(assetID, element);
      element.style.objectFit = item.fit === 'cover' ? 'cover' : 'contain';
    } else if (type !== 'button') {
      element.textContent = item.text || '';
    }
    element.style.position = 'absolute';
    element.style.left = (clamp01(item.x) * 100) + '%';
    element.style.top = (clamp01(item.y) * 100) + '%';
    element.style.width = (clampSize(item.width, type === 'button' ? 0.22 : 0.30) * 100) + '%';
    element.style.height = (clampSize(item.height, type === 'button' ? 0.09 : 0.12) * 100) + '%';
    if (type === 'text') {
      element.style.padding = '10px';
      element.style.overflow = 'hidden';
      element.style.whiteSpace = 'pre-wrap';
      element.style.background = 'transparent';
    }
    canvas.appendChild(element);
  }
}

async function loadPreviewAssetImage(assetID, image) {
  try {
    const response = await fetch('/api/content/assets/' + assetID, {
      headers: token ? { Authorization: 'Bearer ' + token } : {}
    });
    if (!response.ok) throw new Error('Unable to load asset');
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    image.addEventListener('load', () => URL.revokeObjectURL(url), { once: true });
    image.src = url;
  } catch (error) {
    image.alt = image.alt || 'Unable to load image asset ' + assetID;
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

function assetLabel(asset) {
  return `#${asset.id} — ${asset.name || asset.mime || 'Asset'}`;
}

function populateAssetSelect(select, includeNone = true) {
  select.innerHTML = '';
  if (includeNone) {
    const none = document.createElement('option');
    none.value = '0';
    none.textContent = 'None';
    select.appendChild(none);
  }  for (const asset of assets) {
    const option = document.createElement('option');
    option.value = String(asset.id);
    option.textContent = assetLabel(asset);
    select.appendChild(option);
  }
}

function normalizeEditorContent(content) {
  const normalized = content && typeof content === 'object' ? JSON.parse(JSON.stringify(content)) : {};
  if (!normalized.background || typeof normalized.background !== 'object') normalized.background = {};
  if (!Array.isArray(normalized.elements)) normalized.elements = [];
  return normalized;
}

function startGraphicalLayout() {
  const content = parseEditorJSON();
  if (!content) return;
  if (!Array.isArray(content.elements)) content.elements = [];
  if (!content.background || typeof content.background !== 'object') {
    content.background = { asset: 0, fit: 'cover' };
  }
  $('screen-content').value = JSON.stringify(content, null, 2);
  openVisualEditor();
}

function openVisualEditor() {
  const content = parseEditorJSON();
  if (!content) return;
  editorContent = normalizeEditorContent(content);
  editorSelectedIndex = -1;
  populateAssetSelect($('editor-background-asset'));
  populateAssetSelect($('editor-asset'));
  const background = editorContent.background || {};
  $('editor-background-asset').value = String(Number(background.asset || 0));
  if (!$('editor-background-asset').value) $('editor-background-asset').value = '0';
  $('editor-background-fit').value = background.fit === 'contain' ? 'contain' : 'cover';
  $('visual-editor-panel').classList.remove('hidden');
  renderVisualEditor();
}

function closeVisualEditor(apply) {
  if (apply) applyVisualEditor();
  editorContent = null;
  editorSelectedIndex = -1;
  editorDrag = null;
  $('visual-editor-panel').classList.add('hidden');
}

function applyVisualEditor() {
  if (!editorContent) return;
  $('screen-content').value = JSON.stringify(editorContent, null, 2);
}

function selectEditorElement(index) {
  if (!editorContent || !Array.isArray(editorContent.elements)) return;
  if (index < 0 || index >= editorContent.elements.length) {
    editorSelectedIndex = -1;
  } else {
    editorSelectedIndex = index;
  }
  renderVisualEditor();
}

function elementDefaults(type) {
  const defaults = {
    text: { type: 'text', text: 'New text', x: 0.5, y: 0.2, width: 0.3, height: 0.1, font_size: 32, color: '#ffffff', align: 'left', weight: 700 },
    image: { type: 'image', asset: Number(assets[0]?.id || 0), x: 0.5, y: 0.2, width: 0.2, height: 0.2, fit: 'contain' },
    button: { type: 'button', asset: 0, text: 'Button', x: 0.5, y: 0.4, width: 0.22, height: 0.09, background: '#efa00b', color: '#591f0a', destination: { type: 'service', service: 'chat' } }
  };
  return JSON.parse(JSON.stringify(defaults[type]));
}

function addEditorElement(type) {
  if (!editorContent) return;
  if (!Array.isArray(editorContent.elements)) editorContent.elements = [];
  editorContent.elements.push(elementDefaults(type));
  editorSelectedIndex = editorContent.elements.length - 1;
  renderVisualEditor();
}

function renderVisualEditor() {
  if (!editorContent) return;
  const canvas = $('visual-editor-canvas');
  canvas.innerHTML = '';
  const background = editorContent.background || {};
  const backgroundID = Number(background.asset || 0);
  canvas.style.backgroundImage = '';
  canvas.style.backgroundSize = background.fit === 'contain' ? 'contain' : 'cover';
  canvas.style.backgroundPosition = 'center';
  canvas.style.backgroundRepeat = 'no-repeat';
  if (backgroundID > 0) loadEditorBackground(backgroundID, canvas);

  const elements = Array.isArray(editorContent.elements) ? editorContent.elements : [];
  elements.forEach((item, index) => {
    const type = String(item.type || '').toLowerCase();
    if (!['text', 'button', 'image'].includes(type)) return;
    const wrapper = document.createElement('div');
    wrapper.className = 'editor-element' + (index === editorSelectedIndex ? ' selected' : '');
    wrapper.dataset.index = String(index);
    wrapper.style.left = (clamp01(item.x) * 100) + '%';
    wrapper.style.top = (clamp01(item.y) * 100) + '%';
    wrapper.style.width = (clampSize(item.width, type === 'button' ? 0.22 : 0.30) * 100) + '%';
    wrapper.style.height = (clampSize(item.height, type === 'button' ? 0.09 : 0.12) * 100) + '%';

    let visual;
    if (type === 'button') {
      visual = document.createElement('button');
      visual.type = 'button';
      visual.disabled = true;
    } else if (type === 'image') {
      visual = document.createElement('img');
    } else {
      visual = document.createElement('div');
    }
    visual.className = 'editor-element-visual';
    if (type === 'text') {
      visual.textContent = item.text || 'Text';
      visual.style.whiteSpace = 'pre-wrap';
      visual.style.overflow = 'visible';
      visual.style.color = item.color || '#ffffff';
      visual.style.fontSize = Math.max(8, Math.min(200, Number(item.font_size) || 32)) + 'px';
      visual.style.fontWeight = String(item.weight || 700);
      visual.style.textAlign = item.align || 'left';
      visual.style.lineHeight = '1.1';
      visual.style.padding = '6px';
    } else {
      const assetID = Number(item.asset || 0);
      if (assetID > 0) {
        const image = document.createElement('img');
        image.alt = item.alt || item.text || '';
        image.draggable = false;
        image.style.width = '100%';
        image.style.height = '100%';
        image.style.objectFit = item.fit === 'cover' ? 'cover' : 'contain';
        image.style.display = 'block';
        image.style.pointerEvents = 'none';
        loadPreviewAssetImage(assetID, image);
        visual.appendChild(image);
      } else {
        visual.textContent = type === 'button' ? 'Button' : 'Image';
      }
    }
    wrapper.appendChild(visual);
    wrapper.addEventListener('pointerdown', (event) => beginEditorDrag(event, index));
    wrapper.addEventListener('click', (event) => {
      event.stopPropagation();
      selectEditorElement(index);
    });
    if (index === editorSelectedIndex) {
      const handle = document.createElement('div');
      handle.className = 'editor-resize-handle';
      handle.title = 'Drag to resize';
      handle.setAttribute('aria-label', 'Resize element');
      handle.addEventListener('pointerdown', (event) => beginEditorResize(event, index));
      wrapper.appendChild(handle);
    }
    canvas.appendChild(wrapper);
  });

  updateInspector();
}

async function loadEditorBackground(assetID, canvas) {
  try {
    const response = await fetch('/api/content/assets/' + assetID, {
      headers: token ? { Authorization: 'Bearer ' + token } : {}
    });
    if (!response.ok) return;
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    canvas.style.backgroundImage = 'url("' + url + '")';
  } catch (error) {
    // Keep the editor usable if an asset disappears.
  }
}

function beginEditorDrag(event, index) {
  if (!editorContent) return;
  if (event.button !== undefined && event.button !== 0) return;
  event.preventDefault();
  event.stopPropagation();
  editorSelectedIndex = index;
  const item = editorContent.elements[index];
  editorDrag = {
    index,
    startX: event.clientX,
    startY: event.clientY,
    x: clamp01(item.x),
    y: clamp01(item.y)
  };
  event.currentTarget.setPointerCapture?.(event.pointerId);
}

function handleEditorPointerMove(event) {
  if (!editorContent) return;
  const canvas = $('visual-editor-canvas');
  const rect = canvas.getBoundingClientRect();
  if (!rect.width || !rect.height) return;

  if (editorResize) {
    const item = editorContent.elements[editorResize.index];
    if (!item) return;
    const dw = (event.clientX - editorResize.startX) / rect.width;
    const dh = (event.clientY - editorResize.startY) / rect.height;
    const maxWidth = Math.max(0.01, 1 - editorResize.x);
    const maxHeight = Math.max(0.01, 1 - editorResize.y);
    item.width = Number(Math.min(maxWidth, Math.max(0.01, editorResize.width + dw)).toFixed(4));
    item.height = Number(Math.min(maxHeight, Math.max(0.01, editorResize.height + dh)).toFixed(4));
    renderVisualEditor();
    return;
  }

  if (!editorDrag) return;
  const item = editorContent.elements[editorDrag.index];
  const dx = (event.clientX - editorDrag.startX) / rect.width;
  const dy = (event.clientY - editorDrag.startY) / rect.height;
  item.x = Number(clamp01(editorDrag.x + dx).toFixed(4));
  item.y = Number(clamp01(editorDrag.y + dy).toFixed(4));
  renderVisualEditor();
}

function endEditorDrag() {
  editorDrag = null;
}

function endEditorPointerInteraction() {
  editorDrag = null;
  editorResize = null;
}

function beginEditorResize(event, index) {
  if (!editorContent || !editorContent.elements[index]) return;
  event.preventDefault();
  event.stopPropagation();
  editorSelectedIndex = index;
  const item = editorContent.elements[index];
  editorResize = {
    index,
    startX: event.clientX,
    startY: event.clientY,
    width: clampSize(item.width, 0.3),
    height: clampSize(item.height, 0.1),
    x: clamp01(item.x),
    y: clamp01(item.y)
  };
  event.currentTarget.setPointerCapture?.(event.pointerId);
  renderVisualEditor();
}

function endEditorResize() {
  editorResize = null;
}


function updateInspector() {
  const properties = $('editor-properties');
  const empty = $('editor-no-selection');
  const index = editorSelectedIndex;
  if (!editorContent || index < 0 || !editorContent.elements[index]) {
    properties.classList.add('hidden');
    empty.classList.remove('hidden');
    return;
  }

  const item = editorContent.elements[index];
  const type = String(item.type || '').toLowerCase();
  empty.classList.add('hidden');
  properties.classList.remove('hidden');
  $('editor-type').value = type;
  $('editor-text').value = item.text || '';
  $('editor-font-size').value = Math.max(8, Math.min(200, Number(item.font_size) || 32));
  $('editor-text-color').value = item.color || '#ffffff';
  $('editor-text-align').value = item.align || 'left';
  $('editor-font-weight').value = String(item.weight || 700);
  $('editor-asset').value = String(Number(item.asset || 0));
  if (!$('editor-asset').value) $('editor-asset').value = '0';
  $('editor-x').value = clamp01(item.x).toFixed(2);
  $('editor-y').value = clamp01(item.y).toFixed(2);
  $('editor-width').value = clampSize(item.width, type === 'button' ? 0.22 : 0.30).toFixed(2);
  $('editor-height').value = clampSize(item.height, type === 'button' ? 0.09 : 0.12).toFixed(2);
  $('editor-fit').value = item.fit === 'cover' ? 'cover' : 'contain';
  const service = item.destination?.type === 'service' ? String(item.destination.service || '') : '';
  $('editor-service').value = service;
  $('editor-text').disabled = type === 'image';
  $('editor-font-size').disabled = type !== 'text';
  $('editor-text-color').disabled = type !== 'text';
  $('editor-text-align').disabled = type !== 'text';
  $('editor-font-weight').disabled = type !== 'text';
  $('editor-asset').disabled = type === 'text';
  $('editor-fit').disabled = type === 'text';
  $('editor-service').disabled = type !== 'button';
  $('editor-button-background').value = item.background || '#efa00b';
  $('editor-button-color').value = item.color || '#591f0a';
  const graphical = Number(item.asset || 0) > 0;
  $('editor-button-background').disabled = type !== 'button' || graphical;
  $('editor-button-color').disabled = type !== 'button' || graphical;
}

function updateSelectedProperty(property, value) {
  if (!editorContent || editorSelectedIndex < 0) return;
  const item = editorContent.elements[editorSelectedIndex];
  if (!item) return;
  if (property === 'x' || property === 'y') item[property] = Number(clamp01(value).toFixed(4));
  else if (property === 'width' || property === 'height') item[property] = Number(clampSize(value, property === 'width' ? 0.3 : 0.1).toFixed(4));
  else item[property] = value;
  renderVisualEditor();
}

function updateSelectedService(value) {
  if (!editorContent || editorSelectedIndex < 0) return;
  const item = editorContent.elements[editorSelectedIndex];
  if (!item) return;
  if (!value) {
    delete item.destination;
  } else {
    item.destination = { type: 'service', service: value };
  }
  renderVisualEditor();
}

function deleteSelectedElement() {
  if (!editorContent || editorSelectedIndex < 0) return;
  editorContent.elements.splice(editorSelectedIndex, 1);
  editorSelectedIndex = Math.min(editorSelectedIndex, editorContent.elements.length - 1);
  renderVisualEditor();
}

async function renderAssetRow(asset) {
  const row = document.createElement('tr');
  row.innerHTML = '<td>' + asset.id + '</td>' +
    '<td>' + escapeHTML(asset.name) + '</td>' +
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
    assets = result.assets || [];
    const body = $('assets');
    body.innerHTML = '';
    for (const asset of assets) body.appendChild(await renderAssetRow(asset));
    if (editorContent) {
      populateAssetSelect($('editor-background-asset'));
      populateAssetSelect($('editor-asset'));
      renderVisualEditor();
    }
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
  closeVisualEditor(false);
}
$('close-screen').addEventListener('click', closeEditor);
$('screen-panel').querySelector('.screen-modal-backdrop').addEventListener('click', closeEditor);
document.addEventListener('keydown', (event) => {
  if (event.key !== 'Escape') return;
  if (!$('visual-editor-panel').classList.contains('hidden')) closeVisualEditor(true);
  else if (!$('screen-preview').classList.contains('hidden')) $('screen-preview').classList.add('hidden');
  else if (!$('screen-panel').classList.contains('hidden')) closeEditor();
});
$('save-screen').addEventListener('click', saveScreen);
$('preview-screen').addEventListener('click', previewScreen);
$('visual-editor').addEventListener('click', openVisualEditor);
$('close-preview').addEventListener('click', () => $('screen-preview').classList.add('hidden'));
$('screen-preview').querySelector('.screen-preview-backdrop').addEventListener('click', () => $('screen-preview').classList.add('hidden'));
$('close-visual-editor').addEventListener('click', () => closeVisualEditor(true));
$('close-visual-editor-bottom').addEventListener('click', () => closeVisualEditor(true));
$('visual-editor-panel').querySelector('.screen-preview-backdrop').addEventListener('click', () => closeVisualEditor(true));
$('apply-visual-editor').addEventListener('click', applyVisualEditor);
$('add-text').addEventListener('click', () => addEditorElement('text'));
$('add-image').addEventListener('click', () => addEditorElement('image'));
$('add-button').addEventListener('click', () => addEditorElement('button'));
$('delete-element').addEventListener('click', deleteSelectedElement);
$('editor-background-asset').addEventListener('change', (event) => {
  if (!editorContent) return;
  editorContent.background = editorContent.background || {};
  editorContent.background.asset = Number(event.target.value || 0);
  renderVisualEditor();
});
$('editor-background-fit').addEventListener('change', (event) => {
  if (!editorContent) return;
  editorContent.background = editorContent.background || {};
  editorContent.background.fit = event.target.value === 'contain' ? 'contain' : 'cover';
  renderVisualEditor();
});
$('editor-text').addEventListener('input', (event) => updateSelectedProperty('text', event.target.value));
$('editor-font-size').addEventListener('change', (event) => updateSelectedProperty('font_size', Math.max(8, Math.min(200, Number(event.target.value) || 32))));
$('editor-text-color').addEventListener('input', (event) => updateSelectedProperty('color', event.target.value));
$('editor-text-align').addEventListener('change', (event) => updateSelectedProperty('align', event.target.value));
$('editor-font-weight').addEventListener('change', (event) => updateSelectedProperty('weight', Number(event.target.value || 700)));
$('editor-asset').addEventListener('change', (event) => updateSelectedProperty('asset', Number(event.target.value || 0)));
$('editor-x').addEventListener('change', (event) => updateSelectedProperty('x', event.target.value));
$('editor-y').addEventListener('change', (event) => updateSelectedProperty('y', event.target.value));
$('editor-width').addEventListener('change', (event) => updateSelectedProperty('width', event.target.value));
$('editor-height').addEventListener('change', (event) => updateSelectedProperty('height', event.target.value));
$('editor-fit').addEventListener('change', (event) => updateSelectedProperty('fit', event.target.value === 'cover' ? 'cover' : 'contain'));
$('editor-service').addEventListener('change', (event) => updateSelectedService(event.target.value));
$('editor-button-background').addEventListener('input', (event) => updateSelectedProperty('background', event.target.value));
$('editor-button-color').addEventListener('input', (event) => updateSelectedProperty('color', event.target.value));
$('visual-editor-canvas').addEventListener('pointermove', handleEditorPointerMove);
$('visual-editor-canvas').addEventListener('pointerup', endEditorPointerInteraction);
$('visual-editor-canvas').addEventListener('pointercancel', endEditorPointerInteraction);
$('visual-editor-canvas').addEventListener('click', () => selectEditorElement(-1));
$('upload-asset').addEventListener('click', uploadAsset);

$('delete-screen').addEventListener('click', () => {
  if (selectedId) deleteScreen({ id: selectedId, title: $('screen-title').value });
});

refreshScreens();
refreshAssets();