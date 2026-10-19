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
function escapeHtml(value) {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;')
    .replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

function markdownInlineToHtml(value) {
  const protectedParts = [];
  const protect = (html) => {
    const marker = `\u0000${protectedParts.length}\u0000`;
    protectedParts.push(html);
    return marker;
  };
  let text = escapeHtml(value);
  text = text.replace(/`([^`]+)`/g, (_match, code) => protect(`<code>${code}</code>`));
  text = text.replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+|mailto:[^\s)]+)\)/g,
    (_match, label, url) => protect(`<a href="${escapeHtml(url)}">${label}</a>`));
  text = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  text = text.replace(/__(.+?)__/g, '<strong>$1</strong>');
  text = text.replace(/\*(.+?)\*/g, '<em>$1</em>');
  text = text.replace(/_(.+?)_/g, '<em>$1</em>');
  return text.replace(/\u0000(\d+)\u0000/g, (_match, index) => protectedParts[Number(index)]);
}

function markdownSelectionToHtml(markdown) {
  const lines = markdown.replace(/\r\n?/g, '\n').split('\n');
  const blocks = [];
  let paragraph = [];
  let listItems = [];

  const flushParagraph = () => {
    if (paragraph.length) {
      blocks.push(`<p>${paragraph.map(markdownInlineToHtml).join('<br>')}</p>`);
      paragraph = [];
    }
  };
  const flushList = () => {
    if (listItems.length) {
      blocks.push(`<ul>${listItems.map((item) => `<li>${markdownInlineToHtml(item)}</li>`).join('')}</ul>`);
      listItems = [];
    }
  };

  for (const line of lines) {
    const heading = line.match(/^\s{0,3}(#{1,6})\s+(.+)$/);
    const bullet = line.match(/^\s{0,3}[-*+]\s+(.+)$/);
    if (!line.trim()) {
      flushParagraph();
      flushList();
    } else if (heading) {
      flushParagraph();
      flushList();
      const level = heading[1].length;
      blocks.push(`<h${level}>${markdownInlineToHtml(heading[2])}</h${level}>`);
    } else if (bullet) {
      flushParagraph();
      listItems.push(bullet[1]);
    } else {
      flushList();
      paragraph.push(line);
    }
  }
  flushParagraph();
  flushList();
  return blocks.join('\n');
}

function insertMarkdown(kind) {
  const field = $('article-body');
  const start = field.selectionStart ?? field.value.length;
  const end = field.selectionEnd ?? start;
  const selected = field.value.slice(start, end);
  let before = '';
  let after = '';
  let replacement = selected;
  switch (kind) {
    case 'h1': before = '# '; replacement = selected || 'Heading'; after = ''; break;
    case 'h2': before = '## '; replacement = selected || 'Subtitle'; after = ''; break;
    case 'bold': before = '**'; replacement = selected || 'bold text'; after = '**'; break;
    case 'italic': before = '*'; replacement = selected || 'italic text'; after = '*'; break;
    case 'list': before = '- '; replacement = (selected || 'List item').replace(/\n/g, '\n- '); after = ''; break;
    case 'align-left':
    case 'align-center':
    case 'align-right': {
      const alignment = kind.slice('align-'.length);
      const text = selected || 'Text to align';
      // Markdown inside a raw HTML block is not parsed as Markdown by QTextDocument.
      // Convert the selected Markdown to HTML before wrapping it, preserving bold,
      // italics, links, headings and lists in the aligned content.
      const formatted = markdownSelectionToHtml(text);
      const insertion = `<div style="text-align: ${alignment};">\n${formatted}\n</div>`;
      field.setRangeText(insertion, start, end, 'select');
      field.focus();
      field.dispatchEvent(new Event('input', { bubbles: true }));
      return;
    }
    case 'link': before = '['; replacement = selected || 'link text'; after = '](https://example.com)'; break;
    default: return;
  }
  const insertion = before + replacement + after;
  field.setRangeText(insertion, start, end, 'select');
  field.focus();
  field.dispatchEvent(new Event('input', { bubbles: true }));
}
document.querySelectorAll('[data-markdown]').forEach((button) => {
  button.addEventListener('click', () => insertMarkdown(button.dataset.markdown));
});
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
