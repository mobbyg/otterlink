(() => {
  const token = localStorage.getItem('otterlink-token') || '';
  const $ = id => document.getElementById(id);
  let editing = 0;
  let currentBoards = [];
  async function request(path, options = {}) {
    const headers = { ...(options.body ? {'Content-Type':'application/json'} : {}), ...(token ? {Authorization:'Bearer '+token} : {}) };
    const response = await fetch(path, {...options, headers});
    if (!response.ok) throw new Error((await response.text()) || ('HTTP '+response.status));
    if (response.status === 204) return null;
    return response.json();
  }
  function error(message='') { $('boards-error').textContent = message; }
  function reset() { editing=0; $('board-name').value=''; $('board-description').value=''; $('board-rules').value=''; $('save-board').textContent='Create Board'; $('cancel-board-edit').hidden=true; }
  function row(board) {
    const tr=document.createElement('tr');
    const name=document.createElement('td'); name.textContent=board.name;
    const desc=document.createElement('td'); desc.textContent=board.description||'';
    const threads=document.createElement('td'); threads.textContent=String(board.thread_count||0);
    const unread=document.createElement('td'); unread.textContent=String(board.unread_count||0);
    const actions=document.createElement('td');
    const edit=document.createElement('button'); edit.className='secondary'; edit.textContent='Edit';
    edit.addEventListener('click',()=>{editing=board.id;$('board-name').value=board.name;$('board-description').value=board.description||'';$('board-rules').value=board.rules||'';$('save-board').textContent='Save Changes';$('cancel-board-edit').hidden=false;$('boards-section').scrollIntoView({behavior:'smooth',block:'start'});});
    const del=document.createElement('button'); del.className='danger'; del.textContent='Delete';
    del.addEventListener('click',async()=>{if(!confirm('Delete board “'+board.name+'” and all its threads and posts? This cannot be undone.'))return;error();try{await request('/api/admin/boards/'+board.id,{method:'DELETE'});await refresh();}catch(e){error(e.message||String(e));}});
    actions.append(edit,document.createTextNode(' '),del);tr.append(name,desc,threads,unread,actions);return tr;
  }
  async function refresh(){error();try{const data=await request('/api/admin/boards');currentBoards=data.boards||[];const tbody=$('boards');tbody.replaceChildren();for(const b of currentBoards)tbody.appendChild(row(b));const select=$('board-moderator-board');const previous=select.value;select.replaceChildren();for(const b of currentBoards){const option=document.createElement('option');option.value=String(b.id);option.textContent=b.name;select.appendChild(option);}if([...select.options].some(o=>o.value===previous))select.value=previous;}catch(e){error(e.message||String(e));}}
  $('save-board').addEventListener('click',async()=>{const name=$('board-name').value.trim();if(!name){error('Board name is required.');return;}error();const payload={name,description:$('board-description').value,rules:$('board-rules').value};try{await request(editing?'/api/admin/boards/'+editing:'/api/admin/boards',{method:editing?'PUT':'POST',body:JSON.stringify(payload)});reset();await refresh();}catch(e){error(e.message||String(e));}});
  $('cancel-board-edit').addEventListener('click',reset);
  async function setModerator(enabled){const board=$('board-moderator-board').value;const username=$('board-moderator-username').value.trim();if(!board||!username){error('Choose a board and enter a username.');return;}error();try{await request('/api/admin/boards/'+board+'/moderators/'+encodeURIComponent(username),{method:'PUT',body:JSON.stringify({enabled})});$('board-moderator-username').value='';await refresh();}catch(e){error(e.message||String(e));}}
  $('appoint-board-moderator').addEventListener('click',()=>setModerator(true));
  $('remove-board-moderator').addEventListener('click',()=>setModerator(false));
  $('refresh-boards').addEventListener('click',refresh);
  refresh();
})();