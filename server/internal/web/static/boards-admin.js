(() => {
  const token = localStorage.getItem('otterlink-token') || '';
  const $ = id => document.getElementById(id);
  let editing = 0;
  let currentBoards = [];
  let currentUsers = [];
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
    const item=document.createElement('li');
    item.className='board-admin-item';
    const details=document.createElement('div');
    details.className='board-admin-details';
    const name=document.createElement('strong'); name.textContent=board.name;
    const desc=document.createElement('p'); desc.className='status'; desc.textContent=board.description||'No description';
    const stats=document.createElement('p'); stats.className='board-admin-stats'; stats.textContent=String(board.thread_count||0)+' threads · '+String(board.unread_count||0)+' unread';
    details.append(name,desc,stats);
    const actions=document.createElement('div'); actions.className='admin-actions board-admin-actions';
    const edit=document.createElement('button'); edit.className='secondary'; edit.textContent='Edit';
    edit.addEventListener('click',()=>{editing=board.id;$('board-name').value=board.name;$('board-description').value=board.description||'';$('board-rules').value=board.rules||'';$('save-board').textContent='Save Changes';$('cancel-board-edit').hidden=false;$('boards-section').scrollIntoView({behavior:'smooth',block:'start'});$('board-name').focus();});
    const del=document.createElement('button'); del.className='danger'; del.textContent='Delete';
    del.addEventListener('click',async()=>{if(!confirm('Delete board “'+board.name+'” and all its threads and posts? This cannot be undone.'))return;error();try{await request('/api/admin/boards/'+board.id,{method:'DELETE'});if(editing===board.id)reset();await refresh();}catch(e){error(e.message||String(e));}});
    actions.append(edit,del); item.append(details,actions); return item;
  }
  async function refreshUsers(){
    try{
      const data=await request('/api/admin/users');
      currentUsers=(data.users||[]).slice().sort((a,b)=>a.username.localeCompare(b.username,undefined,{sensitivity:'base'}));
      renderUsers();
    }catch(e){error(e.message||String(e));}
  }
  function renderUsers(){
    const list=$('board-user-list'); if(!list)return;
    const filter=$('board-user-filter').value.trim().toLocaleLowerCase();
    const previous=list.value;
    list.replaceChildren();
    const users=currentUsers.filter(user=>user.username.toLocaleLowerCase().includes(filter)||(user.display_name||'').toLocaleLowerCase().includes(filter));
    for(const user of users){
      const option=document.createElement('option');
      option.value=user.username;
      option.textContent=user.display_name&&user.display_name!==user.username?user.display_name+' (@'+user.username+')':'@'+user.username;
      list.appendChild(option);
    }
    if([...list.options].some(option=>option.value===previous))list.value=previous;
    $('board-user-count').textContent=users.length+' of '+currentUsers.length;
  }
  function useSelectedUser(){
    const list=$('board-user-list');
    if(!list.value)return;
    $('board-moderator-username').value=list.value;
    $('board-moderator-username').focus();
  }
  async function refresh(){
    error();
    try{
      const data=await request('/api/admin/boards');
      currentBoards=data.boards||[];
      const list=$('boards'); list.replaceChildren();
      for(const board of currentBoards)list.appendChild(row(board));
      $('board-count').textContent=currentBoards.length+' board'+(currentBoards.length===1?'':'s');
      if(currentBoards.length===0){const empty=document.createElement('li');empty.className='status board-empty';empty.textContent='No boards created yet.';list.appendChild(empty);}
      const select=$('board-moderator-board');const previous=select.value;select.replaceChildren();
      for(const board of currentBoards){const option=document.createElement('option');option.value=String(board.id);option.textContent=board.name;select.appendChild(option);}
      if([...select.options].some(option=>option.value===previous))select.value=previous;
    }catch(e){error(e.message||String(e));}
  }
  $('save-board').addEventListener('click',async()=>{const name=$('board-name').value.trim();if(!name){error('Board name is required.');return;}error();const payload={name,description:$('board-description').value,rules:$('board-rules').value};try{await request(editing?'/api/admin/boards/'+editing:'/api/admin/boards',{method:editing?'PUT':'POST',body:JSON.stringify(payload)});reset();await refresh();}catch(e){error(e.message||String(e));}});
  $('cancel-board-edit').addEventListener('click',reset);
  async function setModerator(enabled){const board=$('board-moderator-board').value;const username=$('board-moderator-username').value.trim();if(!board||!username){error('Choose a board and enter a username.');return;}error();try{await request('/api/admin/boards/'+board+'/moderators/'+encodeURIComponent(username),{method:'PUT',body:JSON.stringify({enabled})});$('board-moderator-username').value='';await refresh();}catch(e){error(e.message||String(e));}}
  $('appoint-board-moderator').addEventListener('click',()=>setModerator(true));
  $('remove-board-moderator').addEventListener('click',()=>setModerator(false));
  $('refresh-boards').addEventListener('click',async()=>{await refresh();await refreshUsers();});
  $('board-user-filter').addEventListener('input',renderUsers);
  $('board-user-list').addEventListener('change',useSelectedUser);
  $('board-user-list').addEventListener('dblclick',useSelectedUser);
  $('use-board-user').addEventListener('click',useSelectedUser);
  refresh();
  refreshUsers();
})();