(() => {
  const gameById = (id) => document.getElementById(id);
  const gameToken = () => localStorage.getItem('otterlink-token') || '';

  async function gameRequest(path, options = {}) {
    const headers = {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(gameToken() ? { Authorization: `Bearer ${gameToken()}` } : {})
    };
    const response = await fetch(path, { ...options, headers });
    if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
    if (response.status === 204) return null;
    return response.json();
  }

  function gameError(message) {
    gameById('games-error').textContent = message || '';
  }

  function clearGameForm() {
    gameById('game-id').value = '';
    gameById('game-slug').value = '';
    gameById('game-title').value = '';
    gameById('game-author').value = '';
    gameById('game-category').value = 'Interactive Fiction';
    gameById('game-story-file').value = '';
    gameById('game-story-upload').value = '';
    gameById('game-upload-status').textContent = '';
    gameById('game-description').value = '';
    gameById('game-published').checked = true;
    gameById('save-game').textContent = 'Add Game';
    gameError('');
  }

  function editGame(game) {
    gameById('game-id').value = game.id;
    gameById('game-slug').value = game.slug;
    gameById('game-title').value = game.title;
    gameById('game-author').value = game.author || '';
    gameById('game-category').value = game.category || '';
    gameById('game-story-file').value = game.story_file;
    gameById('game-description').value = game.description || '';
    gameById('game-published').checked = !!game.published;
    gameById('save-game').textContent = 'Save Changes';
    gameError('');
    gameById('games-section').scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  function gameRow(game) {
    const row = document.createElement('tr');
    const values = [
      game.title,
      game.category || 'Uncategorized',
      game.engine,
      game.story_file,
      game.published ? 'Yes' : 'No',
      game.available ? 'Available' : 'Unavailable'
    ];
    for (const value of values) {
      const cell = document.createElement('td');
      cell.textContent = value || '';
      row.appendChild(cell);
    }
    const actions = document.createElement('td');
    const edit = document.createElement('button');
    edit.className = 'secondary';
    edit.textContent = 'Edit';
    edit.addEventListener('click', () => editGame(game));
    const remove = document.createElement('button');
    remove.className = 'danger';
    remove.textContent = 'Delete';
    remove.addEventListener('click', async () => {
      if (!confirm(`Delete ${game.title} and stop any active sessions?`)) return;
      try {
        gameError('');
        await gameRequest(`/api/admin/games/${game.id}`, { method: 'DELETE' });
        await refreshGames();
        if (gameById('game-id').value === String(game.id)) clearGameForm();
      } catch (error) {
        gameError(error.message || String(error));
      }
    });
    actions.append(edit, document.createTextNode(' '), remove);
    row.appendChild(actions);
    return row;
  }

  async function uploadStoryFile() {
    const file = gameById('game-story-upload').files[0];
    if (!file) {
      gameById('game-upload-status').textContent = 'Choose a story file first.';
      return;
    }
    if (file.size > 20 * 1024 * 1024) {
      gameById('game-upload-status').textContent = 'Story files are limited to 20 MiB.';
      return;
    }
    const form = new FormData();
    form.append('story_file', file, file.name);
    const headers = gameToken() ? { Authorization: `Bearer ${gameToken()}` } : {};
    try {
      gameError('');
      gameById('game-upload-status').textContent = 'Uploading…';
      const response = await fetch('/api/admin/games/files', { method: 'POST', headers, body: form });
      if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
      const result = await response.json();
      const companion = result.filename.toLowerCase().endsWith('.blb');
      if (!companion) gameById('game-story-file').value = result.filename;
      gameById('game-story-upload').value = '';
      gameById('game-upload-status').textContent = companion
        ? `Uploaded companion file ${result.filename} (${result.bytes} bytes). Set Story filename to the matching .z1–.z8 story before saving the catalog entry.`
        : `Uploaded ${result.filename} (${result.bytes} bytes). Save the catalog entry to publish it.`;
      await refreshGames();
    } catch (error) {
      gameById('game-upload-status').textContent = '';
      gameError(error.message || String(error));
    }
  }

  async function refreshGames() {
    gameError('');
    try {
      const result = await gameRequest('/api/admin/games');
      gameById('games-runtime-status').textContent = result.runtime_available
        ? 'Z-machine runtime available (dfrotz found).'
        : 'Z-machine runtime unavailable. Install the frotz package on the server before launching games.';
      const body = gameById('games-list');
      body.replaceChildren();
      const games = result.games || [];
      if (!games.length) {
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 7;
        cell.textContent = 'No games registered yet.';
        row.appendChild(cell);
        body.appendChild(row);
        return;
      }
      for (const game of games) body.appendChild(gameRow(game));
    } catch (error) {
      gameError(error.message || String(error));
    }
  }

  gameById('refresh-games').addEventListener('click', refreshGames);
  gameById('upload-game-story').addEventListener('click', uploadStoryFile);
  gameById('clear-game-form').addEventListener('click', clearGameForm);
  gameById('save-game').addEventListener('click', async () => {
    const id = gameById('game-id').value;
    const requestBody = {
      slug: gameById('game-slug').value,
      title: gameById('game-title').value,
      author: gameById('game-author').value,
      category: gameById('game-category').value,
      description: gameById('game-description').value,
      engine: 'zmachine',
      story_file: gameById('game-story-file').value,
      published: gameById('game-published').checked
    };
    try {
      gameError('');
      await gameRequest(id ? `/api/admin/games/${id}` : '/api/admin/games', {
        method: id ? 'PUT' : 'POST',
        body: JSON.stringify(requestBody)
      });
      clearGameForm();
      await refreshGames();
    } catch (error) {
      gameError(error.message || String(error));
    }
  });

  clearGameForm();
  refreshGames();
})();
