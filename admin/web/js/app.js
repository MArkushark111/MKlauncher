let authToken = localStorage.getItem('mk_token') || '';
let allGames = [];
let browseCallback = null;

const API = '';

function api(method, path, body, isForm = false) {
    const opts = {
        method,
        headers: {},
    };
    if (authToken) opts.headers['Authorization'] = 'Bearer ' + authToken;
    if (body && !isForm) {
        opts.headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(body);
    } else if (body && isForm) {
        opts.body = body;
    }
    return fetch(API + path, opts).then(r => r.json());
}

function authenticate() {
    const pass = document.getElementById('passcode-input').value;
    const btn = document.getElementById('auth-btn');
    const err = document.getElementById('auth-error');

    btn.querySelector('.btn-text').textContent = '';
    btn.querySelector('.btn-loader').classList.remove('hidden');
    err.classList.add('hidden');

    api('POST', '/api/auth', { passcode: pass }).then(data => {
        btn.querySelector('.btn-text').textContent = 'AUTHENTICATE';
        btn.querySelector('.btn-loader').classList.add('hidden');

        if (data.success && data.token) {
            authToken = data.token;
            localStorage.setItem('mk_token', authToken);
            showAdminScreen();
        } else {
            err.textContent = data.error || 'Invalid passcode';
            err.classList.remove('hidden');
        }
    }).catch(e => {
        btn.querySelector('.btn-text').textContent = 'AUTHENTICATE';
        btn.querySelector('.btn-loader').classList.add('hidden');
        err.textContent = 'Connection failed';
        err.classList.remove('hidden');
    });
}

document.getElementById('passcode-input').addEventListener('keydown', e => {
    if (e.key === 'Enter') authenticate();
});

function showAdminScreen() {
    document.getElementById('auth-screen').classList.remove('active');
    document.getElementById('auth-screen').classList.add('hidden');
    document.getElementById('admin-screen').classList.remove('hidden');
    document.getElementById('admin-screen').classList.add('active');
    loadDashboard();
}

function logout() {
    authToken = '';
    localStorage.removeItem('mk_token');
    document.getElementById('admin-screen').classList.remove('active');
    document.getElementById('admin-screen').classList.add('hidden');
    document.getElementById('auth-screen').classList.remove('hidden');
    document.getElementById('auth-screen').classList.add('active');
}

function showTab(tab) {
    document.querySelectorAll('.tab').forEach(t => { t.classList.add('hidden'); t.classList.remove('active'); });
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));

    document.getElementById('tab-' + tab).classList.remove('hidden');
    document.getElementById('tab-' + tab).classList.add('active');

    event.currentTarget.classList.add('active');

    switch(tab) {
        case 'dashboard': loadDashboard(); break;
        case 'games': loadGames(); break;
        case 'archives': loadArchives(); break;
        case 'admins': loadAdmins(); break;
        case 'config': loadConfig(); break;
        case 'notifications': loadNotifications(); break;
    }
}

function loadDashboard() {
    api('GET', '/api/stats').then(data => {
        document.getElementById('stat-games').textContent = data.total_games || 0;
        document.getElementById('stat-downloads').textContent = data.total_downloads || 0;
        document.getElementById('stat-today').textContent = data.today_downloads || 0;
        document.getElementById('stat-week').textContent = data.week_downloads || 0;

        const topList = document.getElementById('top-games-list');
        topList.innerHTML = '';
        if (data.top_games && data.top_games.length > 0) {
            data.top_games.forEach(g => {
                topList.innerHTML += `<div class="archive-item">
                    <span class="name">${g.name}</span>
                    <span class="size">${g.downloads} downloads</span>
                </div>`;
            });
        } else {
            topList.innerHTML = '<p style="color:var(--text-dim)">No downloads yet</p>';
        }
    });

    api('GET', '/api/health').then(data => {
        const el = document.getElementById('health-status');
        el.innerHTML = `<p><span class="label">Status: </span><span style="color:var(--accent)">${data.status}</span></p>
            <p><span class="label">Uptime: </span>${data.uptime}</p>
            <p><span class="label">Games: </span>${data.total_games}</p>
            <p><span class="label">Admins: </span>${data.total_admins}</p>`;
    });
}

function loadGames() {
    api('GET', '/api/games').then(data => {
        allGames = data || [];
        renderGames(allGames);
    });
}

function renderGames(games) {
    const grid = document.getElementById('games-list');
    grid.innerHTML = '';
    if (!games || games.length === 0) {
        grid.innerHTML = '<p style="color:var(--text-dim)">No games found</p>';
        return;
    }
    games.forEach(g => {
        const coverImg = g.cover_url ? `<img src="${g.cover_url}" style="width:100%;height:160px;object-fit:cover">` : '<div class="game-card-cover">&#127918;</div>';
        grid.innerHTML += `<div class="game-card">
            ${coverImg}
            <div class="game-card-body">
                <h4>${g.name}</h4>
                <p>v${g.version} | ${g.category || 'Uncategorized'}</p>
                <p>${g.download_count} downloads</p>
            </div>
            <div class="game-card-actions">
                <button class="btn btn-secondary btn-sm" onclick="editGame(${g.id})">EDIT</button>
                <button class="btn btn-danger btn-sm" onclick="deleteGame(${g.id}, '${g.name.replace(/'/g, "\\'")}')">DELETE</button>
            </div>
        </div>`;
    });
}

function filterGames() {
    const q = document.getElementById('game-search').value.toLowerCase();
    const filtered = allGames.filter(g => g.name.toLowerCase().includes(q) || (g.tags || '').toLowerCase().includes(q));
    renderGames(filtered);
}

function deleteGame(id, name) {
    if (!confirm(`Delete "${name}" and all its files?`)) return;
    api('DELETE', '/api/games/' + id).then(data => {
        if (data.error) { alert(data.error); return; }
        loadGames();
    });
}

function editGame(id) {
    api('GET', '/api/games/' + id).then(game => {
        const modal = document.getElementById('edit-modal');
        const form = document.getElementById('edit-form');
        form.innerHTML = `
            <input type="hidden" id="edit-id" value="${game.id}">
            <div class="form-group"><label>NAME</label><input type="text" id="edit-name" value="${game.name}"></div>
            <div class="form-group"><label>DESCRIPTION</label><textarea id="edit-desc">${game.description || ''}</textarea></div>
            <div class="form-row">
                <div class="form-group"><label>VERSION</label><input type="text" id="edit-version" value="${game.version}"></div>
                <div class="form-group"><label>CATEGORY</label><input type="text" id="edit-category" value="${game.category || ''}"></div>
            </div>
            <div class="form-group"><label>TAGS</label><input type="text" id="edit-tags" value="${game.tags || ''}"></div>
            <div class="form-group"><label>GAME FOLDER</label><input type="text" id="edit-folder" value="${game.game_folder || ''}"></div>
            <div class="form-group"><label>EXE PATH</label><input type="text" id="edit-exe" value="${game.exe_path || ''}"></div>
            <div class="form-group"><label>COVER URL</label><input type="text" id="edit-cover" value="${game.cover_url || ''}"></div>
            <div class="form-group"><label>BACKGROUND URL</label><input type="text" id="edit-bg" value="${game.background_url || ''}"></div>
            <div class="form-group"><label>LOGO URL</label><input type="text" id="edit-logo" value="${game.logo_url || ''}"></div>
            <div class="form-group"><label>WIDE COVER URL</label><input type="text" id="edit-wide" value="${game.wide_cover_url || ''}"></div>
            <button class="btn btn-primary" onclick="saveGameEdit()">SAVE CHANGES</button>
            <div class="form-row" style="margin-top:16px">
                <div class="form-group"><label>NEW VERSION</label><input type="text" id="new-version" placeholder="e.g. 1.1.0"></div>
                <div class="form-group"><label>ARCHIVE</label><input type="file" id="new-version-archive" accept=".zip,.rar,.7z,.tar,.gz,.tar.gz,.tgz"></div>
                <div class="form-group" style="align-self:flex-end"><button class="btn btn-secondary" onclick="uploadGameVersion(${game.id})">UPLOAD VERSION</button></div>
            </div>`;
        modal.classList.remove('hidden');
    });
}

function saveGameEdit() {
    const id = document.getElementById('edit-id').value;
    const fd = new FormData();
    fd.append('name', document.getElementById('edit-name').value);
    fd.append('description', document.getElementById('edit-desc').value);
    fd.append('version', document.getElementById('edit-version').value);
    fd.append('category', document.getElementById('edit-category').value);
    fd.append('tags', document.getElementById('edit-tags').value);
    fd.append('game_folder', document.getElementById('edit-folder').value);
    fd.append('exe_path', document.getElementById('edit-exe').value);

    fetch(API + '/api/games/' + id, {
        method: 'PUT',
        headers: { 'Authorization': 'Bearer ' + authToken },
        body: fd
    }).then(r => r.json()).then(data => {
        if (data.error) { alert(data.error); return; }
        closeEditModal();
        loadGames();
    });
}

function closeEditModal() { document.getElementById('edit-modal').classList.add('hidden'); }

/* Add Game Form */
let currentStep = 1;

function nextStep(step) {
    if (step === 2 && !document.getElementById('game-name').value.trim()) {
        alert('Game name is required');
        return;
    }
    document.getElementById('add-form-step-' + currentStep).classList.add('hidden');
    document.getElementById('add-form-step-' + step).classList.remove('hidden');
    document.getElementById('step-' + step).classList.add('active');
    currentStep = step;
    if (step === 4) buildSummary();
}

function prevStep(step) {
    document.getElementById('add-form-step-' + currentStep).classList.add('hidden');
    document.getElementById('add-form-step-' + step).classList.remove('hidden');
    currentStep = step;
}

function handleArchiveSelect(input) {
    if (input.files.length > 0) {
        const file = input.files[0];
        document.getElementById('archive-name').textContent = file.name;
        document.getElementById('archive-size').textContent = formatSize(file.size);
        document.getElementById('archive-info').classList.remove('hidden');
    }
}

function previewImage(input, previewId) {
    if (input.files && input.files[0]) {
        const reader = new FileReader();
        reader.onload = e => {
            const img = document.getElementById(previewId);
            img.src = e.target.result;
            img.classList.remove('hidden');
            img.parentElement.querySelector('.img-placeholder').style.display = 'none';
        };
        reader.readAsDataURL(input.files[0]);
    }
}

function browseArchive() {
    openBrowseModal(path => {
        document.getElementById('game-folder').value = path;
    });
}

function browseForExe() {
    openBrowseModal(path => {
        if (path.toLowerCase().endsWith('.exe')) {
            document.getElementById('game-exe').value = path;
        } else {
            alert('Please select a .exe file');
        }
    });
}

function openBrowseModal(callback) {
    browseCallback = callback;
    const modal = document.getElementById('browse-modal');
    modal.classList.remove('hidden');
    browsePath('/storage/archives');
}

let currentBrowsePath = '';

function browsePath(path) {
    currentBrowsePath = path;
    document.getElementById('browse-path').textContent = path;

    fetch(API + '/api/archive/browse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + authToken },
        body: JSON.stringify({ path })
    }).then(r => r.json()).then(data => {
        const items = document.getElementById('browse-items');
        items.innerHTML = '';

        if (path !== '/storage/archives') {
            const parent = path.substring(0, path.lastIndexOf('/'));
            items.innerHTML += `<div class="browse-item" onclick="browsePath('${parent}')">
                <span class="icon">&#128194;</span>
                <span class="item-name">..</span>
            </div>`;
        }

        if (data.entries) {
            data.entries.sort((a, b) => b.isDir - a.isDir);
            data.entries.forEach(item => {
                const icon = item.isDir ? '&#128193;' : '&#128196;';
                const size = item.isDir ? '' : formatSize(item.size);
                const clickAction = item.isDir
                    ? `browsePath('${item.path.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}')`
                    : `selectBrowseItem('${item.path.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}')`;
                items.innerHTML += `<div class="browse-item" onclick="${clickAction}">
                    <span class="icon">${icon}</span>
                    <span class="item-name">${item.name}</span>
                    <span class="item-size">${size}</span>
                </div>`;
            });
        }
    });
}

function selectBrowseItem(path) {
    if (browseCallback) browseCallback(path);
    closeModal();
}

function closeModal() {
    document.getElementById('browse-modal').classList.add('hidden');
}

function buildSummary() {
    const summary = document.getElementById('game-summary');
    const name = document.getElementById('game-name').value;
    const desc = document.getElementById('game-desc').value;
    const ver = document.getElementById('game-version').value;
    const cat = document.getElementById('game-category').value;
    const tags = document.getElementById('game-tags').value;
    const folder = document.getElementById('game-folder').value;
    const exe = document.getElementById('game-exe').value;
    const archive = document.getElementById('archive-file').files[0];

    summary.innerHTML = `
        <p><span class="label">Name: </span>${name}</p>
        <p><span class="label">Description: </span>${desc || 'None'}</p>
        <p><span class="label">Version: </span>${ver}</p>
        <p><span class="label">Category: </span>${cat || 'None'}</p>
        <p><span class="label">Tags: </span>${tags || 'None'}</p>
        <p><span class="label">Archive: </span>${archive ? archive.name : 'None'}</p>
        <p><span class="label">Game Folder: </span>${folder || 'Root'}</p>
        <p><span class="label">Executable: </span>${exe || 'None'}</p>`;
}

function submitGame() {
    const btn = document.getElementById('add-game-btn');
    const progress = document.getElementById('add-progress');
    btn.disabled = true;
    progress.classList.remove('hidden');

    const fd = new FormData();
    fd.append('name', document.getElementById('game-name').value);
    fd.append('description', document.getElementById('game-desc').value);
    fd.append('version', document.getElementById('game-version').value);
    fd.append('category', document.getElementById('game-category').value);
    fd.append('tags', document.getElementById('game-tags').value);
    fd.append('game_folder', document.getElementById('game-folder').value);
    fd.append('exe_path', document.getElementById('game-exe').value);

    const archiveFile = document.getElementById('archive-file').files[0];
    if (archiveFile) fd.append('archive', archiveFile);

    const coverFile = document.getElementById('cover-file').files[0];
    if (coverFile) fd.append('cover', coverFile);

    const bgFile = document.getElementById('bg-file').files[0];
    if (bgFile) fd.append('background', bgFile);

    const logoFile = document.getElementById('logo-file').files[0];
    if (logoFile) fd.append('logo', logoFile);

    const wideFile = document.getElementById('wide-file').files[0];
    if (wideFile) fd.append('wide_cover', wideFile);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', API + '/api/games');
    xhr.setRequestHeader('Authorization', 'Bearer ' + authToken);

    xhr.upload.onprogress = e => {
        if (e.lengthComputable) {
            const pct = Math.round(e.loaded / e.total * 100);
            document.getElementById('add-progress-fill').style.width = pct + '%';
            document.getElementById('add-progress-text').textContent = `Uploading... ${pct}% (${formatSize(e.loaded)} / ${formatSize(e.total)})`;
        }
    };

    xhr.onload = function() {
        btn.disabled = false;
        progress.classList.add('hidden');
        const data = JSON.parse(xhr.responseText);
        if (data.error) {
            alert('Error: ' + data.error);
            return;
        }
        alert('Game "' + data.name + '" added successfully!');
        resetAddForm();
        showTab('games');
    };

    xhr.onerror = function() {
        btn.disabled = false;
        progress.classList.add('hidden');
        alert('Upload failed');
    };

    xhr.send(fd);
}

function resetAddForm() {
    document.getElementById('game-name').value = '';
    document.getElementById('game-desc').value = '';
    document.getElementById('game-version').value = '1.0.0';
    document.getElementById('game-category').value = '';
    document.getElementById('game-tags').value = '';
    document.getElementById('game-folder').value = '';
    document.getElementById('game-exe').value = '';
    ['cover-preview', 'bg-preview', 'logo-preview', 'wide-preview'].forEach(id => {
        const el = document.getElementById(id);
        el.classList.add('hidden');
        el.src = '';
    });
    prevStep(1);
}

/* Archives */
function loadArchives() {
    api('GET', '/api/archives').then(data => {
        const list = document.getElementById('archives-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No archives found</p>';
            return;
        }
        data.forEach(a => {
            list.innerHTML += `<div class="archive-item">
                <span class="name">${a.name}</span>
                <span class="size">${formatSize(a.size)}</span>
            </div>`;
        });
    });
}

function uploadArchiveDirect(input) {
    if (input.files.length === 0) return;
    const file = input.files[0];
    const fd = new FormData();
    fd.append('archive', file);

    const progress = document.getElementById('upload-progress');
    progress.classList.remove('hidden');

    const xhr = new XMLHttpRequest();
    xhr.open('POST', API + '/api/archives/upload');
    xhr.setRequestHeader('Authorization', 'Bearer ' + authToken);

    xhr.upload.onprogress = e => {
        if (e.lengthComputable) {
            const pct = Math.round(e.loaded / e.total * 100);
            document.getElementById('upload-progress-fill').style.width = pct + '%';
            document.getElementById('upload-progress-text').textContent = `Uploading... ${pct}% (${formatSize(e.loaded)} / ${formatSize(e.total)})`;
        }
    };

    xhr.onload = function() {
        progress.classList.add('hidden');
        const data = JSON.parse(xhr.responseText);
        if (data.error) { alert(data.error); return; }
        alert('Archive uploaded!');
        loadArchives();
    };

    xhr.onerror = function() {
        progress.classList.add('hidden');
        alert('Upload failed');
    };

    xhr.send(fd);
    input.value = '';
}

/* Admins */
function loadAdmins() {
    api('GET', '/api/admins').then(data => {
        const list = document.getElementById('admins-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No admins</p>';
            return;
        }
        data.forEach(a => {
            list.innerHTML += `<div class="admin-item">
                <span>${a.username}</span>
                <span style="color:var(--text-dim)">Created: ${new Date(a.created_at).toLocaleDateString()}</span>
                <button class="btn btn-danger btn-sm" onclick="removeAdmin('${a.username}')">REMOVE</button>
            </div>`;
        });
    });
}

function addAdmin() {
    const user = document.getElementById('new-admin-user').value.trim();
    const pass = document.getElementById('new-admin-pass').value.trim();
    if (!user || !pass) { alert('Username and passcode required'); return; }

    api('POST', '/api/admin/add', { username: user, passcode: pass }).then(data => {
        if (data.error) { alert(data.error); return; }
        document.getElementById('new-admin-user').value = '';
        document.getElementById('new-admin-pass').value = '';
        loadAdmins();
    });
}

function removeAdmin(username) {
    if (!confirm(`Remove admin "${username}"?`)) return;
    api('DELETE', '/api/admin/' + username).then(data => {
        if (data.error) { alert(data.error); return; }
        loadAdmins();
    });
}

/* Config */
function loadConfig() {
    api('GET', '/api/config').then(data => {
        document.getElementById('cfg-wan-host').value = data.wan_host || '';
        document.getElementById('cfg-wan-port').value = data.wan_port || '8080';
        document.getElementById('cfg-server-name').value = data.server_name || '';
        document.getElementById('cfg-max-upload').value = data.max_upload_mb || 4096;

        if (data.wan_host && data.wan_port) {
            document.getElementById('wan-display').textContent = data.wan_host + ':' + data.wan_port;
        }
    });
}

function saveConfig() {
    const config = {
        wan_host: document.getElementById('cfg-wan-host').value,
        wan_port: document.getElementById('cfg-wan-port').value,
        server_name: document.getElementById('cfg-server-name').value,
        max_upload_mb: parseInt(document.getElementById('cfg-max-upload').value) || 4096,
    };

    return api('PUT', '/api/config', config).then(data => {
        if (data.error) {
            alert(data.error);
            throw new Error(data.error);
        }
        loadConfig();
        return data;
    });
}

function applyAndRestartServer() {
    saveConfig().then(() => {
        if (!confirm('Restart the server now?')) return;
        api('POST', '/api/server/restart').then(data => {
            if (data.error) { alert(data.error); return; }
            alert('Server restart requested. Reconnect in a few seconds.');
        });
    });
}

function copyWanAddress() {
    const address = document.getElementById('wan-display').textContent.trim();
    if (!address || address.startsWith('Configure')) return;
    const fallbackCopy = () => {
        const input = document.createElement('textarea');
        input.value = address;
        document.body.appendChild(input);
        input.select();
        const copied = document.execCommand('copy');
        input.remove();
        if (copied) alert('WAN address copied');
        else alert('Unable to copy WAN address');
    };
    if (navigator.clipboard) {
        navigator.clipboard.writeText(address).then(() => alert('WAN address copied')).catch(fallbackCopy);
    } else {
        fallbackCopy();
    }
}

function uploadGameVersion(id) {
    const version = document.getElementById('new-version').value.trim();
    const archive = document.getElementById('new-version-archive').files[0];
    if (!version || !archive) {
        alert('Version and archive are required');
        return;
    }
    const form = new FormData();
    form.append('version', version);
    form.append('archive', archive);
    fetch(API + '/api/games/' + id + '/versions', {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + authToken },
        body: form
    }).then(r => r.json()).then(data => {
        if (data.error) { alert(data.error); return; }
        alert('Version uploaded');
        closeEditModal();
        loadGames();
    }).catch(() => alert('Version upload failed'));
}

/* Notifications */
function loadNotifications() {
    api('GET', '/api/notifications').then(data => {
        const list = document.getElementById('notifications-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No notifications</p>';
            return;
        }
        data.forEach(n => {
            list.innerHTML += `<div class="notification-item ${n.read ? '' : 'unread'}">
                <h4>${n.title}</h4>
                <p>${n.message}</p>
                <div class="time">${new Date(n.created_at).toLocaleString()}</div>
            </div>`;
        });
    });
}

/* Utilities */
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

/* Init */
if (authToken) {
    api('GET', '/api/health').then(() => {
        showAdminScreen();
    }).catch(() => {
        authToken = '';
        localStorage.removeItem('mk_token');
    });
}
