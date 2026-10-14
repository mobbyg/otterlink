const token = localStorage.getItem('otterlink-token') || '';
const $ = (id) => document.getElementById(id);
let articles = [];

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

function showError(message = '') { $('native-news-error').textContent = message; }
function formatDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
function makeCell(row, text) {
  const cell = document.createElement('td');
  cell.textContent = text;
  row.appendChild(cell);
  return cell;
}
function renderArticle(article) {
  const row = document.createElement('tr');
  makeCell(row, article.title);
  const slugCell = makeCell(row, article.slug);
  slugCell.title = 'Stable article destination';
  makeCell(row, article.published ? 'Published' : 'Draft');
  makeCell(row, article.published ? 'Yes' : 'No');
  makeCell(row, formatDate(article.updated_at));
  const actions = makeCell(row, '');
  const edit = document.createElement('button');
  edit.type = 'button'; edit.className = 'secondary'; edit.textContent = 'Edit';
  edit.addEventListener('click', () => editArticle(article));
  actions.appendChild(edit);
  const copy = document.createElement('button');
  copy.type = 'button'; copy.className = 'secondary'; copy.textContent = 'Copy key';
  copy.addEventListener('click', async () => {
    try { await navigator.clipboard.writeText(article.slug); copy.textContent = 'Copied'; }
    catch { showError('Clipboard access was blocked; copy the slug from the table instead.'); }
  });
  actions.appendChild(copy);
  return row;
}
async function refreshArticles() {
  showError('');
  try {
    const result = await request('/api/admin/native-news');
    articles = result.articles || [];
    const body = $('native-news-articles');
    body.replaceChildren();
    for (const article of articles) body.appendChild(renderArticle(article));
  } catch (error) { showError(error.message || String(error)); }
}
function clearEditor() {
  $('article-id').value = '0';
  $('article-title').value = '';
  $('article-slug').value = '';
  $('article-slug').disabled = false;
  $('article-body').value = '';
  $('article-published').checked = false;
  $('editor-heading').textContent = 'New Article';
  $('article-status').textContent = 'Draft';
  $('article-preview').classList.add('hidden');
}
function editArticle(article) {
  $('article-id').value = String(article.id);
  $('article-title').value = article.title || '';
  $('article-slug').value = article.slug || '';
  $('article-slug').disabled = true;
  $('article-body').value = article.body || '';
  $('article-published').checked = Boolean(article.published);
  $('editor-heading').textContent = 'Edit Article';
  $('article-status').textContent = article.published ? 'Published · linkable' : 'Draft · not linkable';
  $('article-preview').classList.add('hidden');
  $('article-editor').scrollIntoView({ behavior: 'smooth', block: 'start' });
}
$('new-article').addEventListener('click', () => { clearEditor(); $('article-title').focus(); });
$('clear-article').addEventListener('click', clearEditor);
$('refresh-articles').addEventListener('click', refreshArticles);
$('article-published').addEventListener('change', () => {
  $('article-status').textContent = $('article-published').checked ? 'Published · linkable' : 'Draft · not linkable';
});
$('preview-article').addEventListener('click', () => {
  $('preview-title').textContent = $('article-title').value;
  $('preview-body').textContent = $('article-body').value;
  $('article-preview').classList.remove('hidden');
});
$('article-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  showError('');
  const id = Number($('article-id').value || 0);
  const payload = {
    id,
    slug: $('article-slug').value.trim(),
    title: $('article-title').value.trim(),
    body: $('article-body').value,
    published: $('article-published').checked
  };
  try {
    const article = await request('/api/admin/native-news', {
      method: 'PUT', body: JSON.stringify(payload)
    });
    await refreshArticles();
    editArticle(article);
    $('article-status').textContent = article.published ? 'Published · linkable' : 'Draft · not linkable';
  } catch (error) { showError(error.message || String(error)); }
});
clearEditor();
refreshArticles();
