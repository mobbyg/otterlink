const state = { token: localStorage.getItem('otterlink-token') || '', user: null };
const $ = (id) => document.getElementById(id);

async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (options.body && typeof options.body !== 'string') { headers['Content-Type'] = 'application/json'; options.body = JSON.stringify(options.body); }
  const response = await fetch(path, { ...options, headers });
  if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
  if (response.status === 204) return null;
  return response.json();
}

function showApp() { $('auth').classList.add('hidden'); $('app').classList.remove('hidden'); $('logout').classList.remove('hidden'); }
function showAuth() { $('auth').classList.remove('hidden'); $('app').classList.add('hidden'); $('logout').classList.add('hidden'); }
function fail(id, error) { $(id).textContent = error.message || String(error); }

$('login').addEventListener('submit', async (event) => {
  event.preventDefault(); $('login-error').textContent = '';
  const form = new FormData(event.target);
  try {
    const result = await request('/api/auth/login', { method:'POST', body:{ username:form.get('username'), password:form.get('password') } });
    state.token = result.token; localStorage.setItem('otterlink-token', state.token); await start();
  } catch (error) { fail('login-error', error); }
});

$('register').addEventListener('submit', async (event) => {
  event.preventDefault(); $('register-error').textContent = '';
  const form = new FormData(event.target);
  try {
    await request('/api/auth/register', { method:'POST', body:{ username:form.get('username'), display_name:form.get('display_name'), email:form.get('email'), password:form.get('password') } });
    $('login').elements.username.value = form.get('username');
    $('login').elements.password.value = form.get('password');
    $('register-error').textContent = 'Account created. Connect when ready.';
  } catch (error) { fail('register-error', error); }
});

$('add-buddy').addEventListener('submit', async (event) => {
  event.preventDefault();
  const username = new FormData(event.target).get('username');
  try { await request('/api/buddies', { method:'POST', body:{ username } }); event.target.reset(); await refresh(); }
  catch (error) { alert(error.message); }
});

$('send-chat').addEventListener('submit', async (event) => {
  event.preventDefault(); const input = event.target.elements.message;
  try { await request('/api/chat', { method:'POST', body:{ message:input.value } }); input.value=''; await refreshChat(); }
  catch (error) { alert(error.message); }
});

$('logout').addEventListener('click', async () => {
  try { await request('/api/auth/logout', { method:'POST' }); } catch (_) {}
  state.token=''; localStorage.removeItem('otterlink-token'); state.user=null; showAuth();
});

async function refresh() { await Promise.all([refreshBuddies(), refreshPresence(), refreshChat()]); }
async function refreshBuddies() {
  const result = await request('/api/buddies'); const online = new Set((await request('/api/presence')).users.map(u => u.username.toLowerCase()));
  $('buddies').innerHTML = '';
  for (const buddy of result.buddies) {
    const li = document.createElement('li'); li.className = online.has(buddy.username.toLowerCase()) ? 'online' : '';
    li.textContent = buddy.display_name || buddy.username;
    const name = buddy.username;
    li.title = name;
    li.addEventListener('dblclick', async () => { if (confirm(`Remove ${name} from buddies?`)) { await request(`/api/buddies?username=${encodeURIComponent(name)}`, {method:'DELETE'}); await refresh(); } });
    $('buddies').appendChild(li);
  }
}
async function refreshPresence() {
  const result = await request('/api/presence'); $('online').innerHTML='';
  for (const user of result.users) { const li=document.createElement('li'); li.className='online'; li.textContent=user.display_name || user.username; $('online').appendChild(li); }
}
async function refreshChat() {
  const result = await request('/api/chat'); const box=$('chat'); box.innerHTML='';
  for (const message of result.messages || []) {
    const row=document.createElement('article'); row.className='msg';
    const who=document.createElement('strong'); who.textContent=message.from.display_name || message.from.username;
    const time=document.createElement('time'); time.textContent=new Date(message.timestamp).toLocaleTimeString([], {hour:'numeric',minute:'2-digit'});
    const text=document.createElement('p'); text.textContent=message.message;
    row.append(who,time,text); box.appendChild(row);
  }
  box.scrollTop=box.scrollHeight;
}

async function start() {
  try {
    state.user = await request('/api/me'); $('identity').textContent = state.user.display_name || state.user.username; showApp(); await refresh();
  } catch (_) { state.token=''; localStorage.removeItem('otterlink-token'); showAuth(); }
}

if (state.token) start(); else showAuth();
setInterval(() => { if (state.token) refresh().catch(() => {}); }, 3000);
