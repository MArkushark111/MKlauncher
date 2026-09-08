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

/* News Admin */
function loadNewsAdmin() {
    api('GET', '/api/news').then(data => {
        const list = document.getElementById('news-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No news posts</p>';
            return;
        }
        data.forEach(n => {
            list.innerHTML += `<div class="archive-item" style="flex-direction:column;gap:4px">
                <div style="display:flex;justify-content:space-between;width:100%">
                    <span class="name">${n.title} ${n.is_pinned ? '<span class="badge badge-yellow">PINNED</span>' : ''}</span>
                    <button class="btn btn-danger btn-sm" onclick="deleteNews(${n.id})">DELETE</button>
                </div>
                <span style="color:var(--text-dim);font-size:12px">${n.content.substring(0, 100)}${n.content.length > 100 ? '...' : ''}</span>
                <span style="color:var(--text-dim);font-size:11px">${n.created_at || ''}</span>
            </div>`;
        });
    });
}

function createNews() {
    const title = document.getElementById('news-title').value.trim();
    const content = document.getElementById('news-content').value.trim();
    const image_url = document.getElementById('news-image').value.trim();
    const is_pinned = document.getElementById('news-pinned').value === '1';
    if (!title || !content) { alert('Title and content required'); return; }
    api('POST', '/api/news', { title, content, image_url, is_pinned }).then(() => {
        document.getElementById('news-title').value = '';
        document.getElementById('news-content').value = '';
        document.getElementById('news-image').value = '';
        loadNewsAdmin();
    });
}

function deleteNews(id) {
    if (!confirm('Delete this news post?')) return;
    api('DELETE', '/api/news/' + id).then(() => loadNewsAdmin());
}

/* Categories Admin */
function loadCategoriesAdmin() {
    api('GET', '/api/categories').then(data => {
        const list = document.getElementById('categories-list');
        list.innerHTML = '';
        if (!data) return;
        data.forEach(c => {
            list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                <span class="name">${c.icon} ${c.name}</span>
                <span style="color:var(--text-dim);font-size:12px">ID: ${c.id}</span>
            </div>`;
        });
    });
}

function createCategory() {
    const name = document.getElementById('new-cat-name').value.trim();
    const icon = document.getElementById('new-cat-icon').value.trim();
    if (!name) { alert('Name required'); return; }
    api('POST', '/api/admin/categories', { name, icon }).then(() => {
        document.getElementById('new-cat-name').value = '';
        document.getElementById('new-cat-icon').value = '';
        loadCategoriesAdmin();
    });
}

/* Featured Admin */
function loadFeaturedAdmin() {
    api('GET', '/api/games/featured').then(data => {
        const list = document.getElementById('featured-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No featured games</p>';
            return;
        }
        data.forEach(g => {
            list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                <span class="name">${g.name}</span>
                <button class="btn btn-danger btn-sm" onclick="removeFeatured(${g.id})">REMOVE</button>
            </div>`;
        });
    });
    api('GET', '/api/games').then(data => {
        const sel = document.getElementById('featured-game-select');
        sel.innerHTML = '';
        if (data) data.forEach(g => {
            sel.innerHTML += `<option value="${g.id}">${g.name}</option>`;
        });
    });
}

function setFeatured() {
    const gameId = document.getElementById('featured-game-select').value;
    api('POST', '/api/games/' + gameId + '/featured', { featured: true }).then(() => loadFeaturedAdmin());
}

function removeFeatured(gameId) {
    api('POST', '/api/games/' + gameId + '/featured', { featured: false }).then(() => loadFeaturedAdmin());
}

/* Mods Admin */
function loadModsAdmin() {
    api('GET', '/api/games').then(games => {
        if (!games || games.length === 0) return;
        const list = document.getElementById('mods-list');
        list.innerHTML = '';
        let loaded = 0;
        games.forEach(g => {
            api('GET', '/api/games/' + g.id + '/mods').then(mods => {
                if (mods && mods.length > 0) {
                    mods.forEach(m => {
                        list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                            <div>
                                <span class="name">${m.name}</span>
                                <span style="color:var(--text-dim);font-size:12px;margin-left:8px">by ${m.username || 'unknown'} | ${m.downloads} downloads</span>
                            </div>
                            <button class="btn btn-danger btn-sm" onclick="deleteMod(${m.id})">DELETE</button>
                        </div>`;
                    });
                }
                loaded++;
                if (loaded === games.length && list.innerHTML === '') {
                    list.innerHTML = '<p style="color:var(--text-dim)">No mods uploaded</p>';
                }
            });
        });
    });
}

function deleteMod(id) {
    if (!confirm('Delete this mod?')) return;
    api('DELETE', '/api/mods/' + id).then(() => loadModsAdmin());
}

/* Servers Admin */
function loadServersAdmin() {
    api('GET', '/api/games').then(games => {
        if (!games || games.length === 0) return;
        const sel = document.getElementById('server-game-select');
        sel.innerHTML = '';
        games.forEach(g => { sel.innerHTML += `<option value="${g.id}">${g.name}</option>`; });
    });
    api('GET', '/api/games').then(games => {
        const list = document.getElementById('servers-list');
        list.innerHTML = '';
        if (!games) return;
        let loaded = 0;
        games.forEach(g => {
            api('GET', '/api/games/' + g.id + '/servers').then(servers => {
                if (servers && servers.length > 0) {
                    servers.forEach(s => {
                        const statusColor = s.current_players > 0 ? 'var(--accent)' : 'var(--text-dim)';
                        list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                            <div>
                                <span class="name">${s.name}</span>
                                <span style="color:var(--text-dim);font-size:12px;margin-left:8px">${s.ip}:${s.port} | ${s.current_players}/${s.max_players}</span>
                            </div>
                            <button class="btn btn-danger btn-sm" onclick="deleteServer(${s.id})">DELETE</button>
                        </div>`;
                    });
                }
                loaded++;
                if (loaded === games.length && list.innerHTML === '') {
                    list.innerHTML = '<p style="color:var(--text-dim)">No servers configured</p>';
                }
            });
        });
    });
}

function addServer() {
    const gameId = document.getElementById('server-game-select').value;
    const name = document.getElementById('server-name').value.trim();
    const ip = document.getElementById('server-ip').value.trim();
    const port = parseInt(document.getElementById('server-port').value);
    const max = parseInt(document.getElementById('server-max').value) || 32;
    if (!name || !ip || !port) { alert('Fill all fields'); return; }
    api('POST', '/api/games/' + gameId + '/servers', { name, ip, port, max_players: max }).then(() => {
        document.getElementById('server-name').value = '';
        document.getElementById('server-ip').value = '';
        document.getElementById('server-port').value = '';
        loadServersAdmin();
    });
}

function deleteServer(id) {
    if (!confirm('Delete this server?')) return;
    api('DELETE', '/api/servers/' + id).then(() => loadServersAdmin());
}

/* Launcher Updates */
function loadLauncherAdmin() {
    api('GET', '/api/launcher/version').then(data => {
        const el = document.getElementById('launcher-current');
        if (data.has_update) {
            el.innerHTML = `<div class="archive-item" style="justify-content:space-between">
                <div>
                    <span class="name">v${data.version}</span>
                    <span style="color:var(--text-dim);font-size:12px;margin-left:8px">${data.file_size ? (data.file_size / 1048576).toFixed(1) + ' MB' : ''}</span>
                    <span style="color:var(--text-dim);font-size:12px;margin-left:8px">${data.created_at || ''}</span>
                </div>
            </div>
            ${data.changelog ? '<div style="margin-top:8px;color:var(--text-dim);font-size:12px">' + data.changelog + '</div>' : ''}`;
        } else {
            el.innerHTML = '<p style="color:var(--text-dim)">No updates uploaded yet</p>';
        }
    });
}

function uploadLauncherUpdate() {
    const version = document.getElementById('launcher-version').value.trim();
    const changelog = document.getElementById('launcher-changelog').value.trim();
    const fileInput = document.getElementById('launcher-file');
    if (!version) { alert('Enter version number'); return; }
    if (!fileInput.files.length) { alert('Select setup file'); return; }
    const formData = new FormData();
    formData.append('version', version);
    formData.append('changelog', changelog);
    formData.append('file', fileInput.files[0]);
    fetch('/api/admin/launcher/upload', {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + authToken },
        body: formData
    }).then(r => r.json()).then(data => {
        if (data.success) {
            document.getElementById('launcher-version').value = '';
            document.getElementById('launcher-changelog').value = '';
            fileInput.value = '';
            loadLauncherAdmin();
            alert('Launcher v' + version + ' uploaded! All launchers will update on next start.');
        } else {
            alert(data.error || 'Upload failed');
        }
    }).catch(e => alert('Upload failed: ' + e));
}

/* Add Game (URL) - Multi-step wizard */
function loadURLGameCategories() {
    api('GET', '/api/categories').then(data => {
        const sel = document.getElementById('url-game-category');
        sel.innerHTML = '';
        if (data) data.forEach(c => {
            sel.innerHTML += `<option value="${c.name}">${c.icon} ${c.name}</option>`;
        });
    });
    urlStepNext(1);
}

function urlStepNext(step) {
    if (step === 2) {
        if (!document.getElementById('url-game-name').value.trim()) { alert('Enter game name'); return; }
        if (document.getElementById('url-game-status').value === 'coming_soon') { step = 3; }
    }
    if (step === 4) {
        const urls = [];
        document.querySelectorAll('#url-download-list .url-row').forEach(row => {
            const link = row.querySelector('.url-link-input').value.trim();
            if (link) urls.push({ url: link, label: row.querySelector('.url-label-input').value.trim() || 'Mirror' });
        });
        if (urls.length === 0) { alert('Add at least one download URL'); return; }
        if (!document.getElementById('url-game-exe').value.trim()) { alert('Enter exe path'); return; }
        const summary = `
            <b>Name:</b> ${document.getElementById('url-game-name').value}<br>
            <b>Version:</b> ${document.getElementById('url-game-version').value || '1.0.0'}<br>
            <b>Category:</b> ${document.getElementById('url-game-category').value}<br>
            <b>Description:</b> ${document.getElementById('url-game-desc').value || 'N/A'}<br>
            <b>Tags:</b> ${document.getElementById('url-game-tags').value || 'N/A'}<br>
            <b>Download URLs:</b> ${urls.length} link(s)<br>${urls.map(u => `&nbsp;&nbsp;- ${u.label}: ${u.url}`).join('<br>')}<br>
            <b>Exe Path:</b> ${document.getElementById('url-game-exe').value}<br>
            <b>Cover:</b> ${document.getElementById('url-game-cover').files[0] ? document.getElementById('url-game-cover').files[0].name : 'None'}<br>
            <b>Background:</b> ${document.getElementById('url-game-bg').files[0] ? document.getElementById('url-game-bg').files[0].name : 'None'}
        `;
        document.getElementById('url-confirm-summary').innerHTML = summary;
    }
    for (let i = 1; i <= 4; i++) {
        document.getElementById('url-step-' + i).style.display = (i === step) ? 'block' : 'none';
    }
    const dots = document.querySelectorAll('#url-step-indicator .step-dot');
    dots.forEach((d, idx) => { d.style.background = (idx < step) ? 'var(--accent)' : '#333'; });
}

function urlStepBack(step) {
    if (step === 2 && document.getElementById('url-game-status').value === 'coming_soon') { step = 1; }
    urlStepNext(step);
}

function addUrlRow() {
    const list = document.getElementById('url-download-list');
    const row = document.createElement('div');
    row.className = 'url-row';
    row.style.cssText = 'display:flex;gap:8px;margin-bottom:8px';
    row.innerHTML = `
        <input type="text" class="url-link-input" placeholder="https://example.com/mirror.zip" style="flex:3">
        <input type="text" class="url-label-input" placeholder="Label" style="flex:1" value="Mirror">
        <button class="btn" onclick="this.parentElement.remove()" style="padding:6px 12px">X</button>
    `;
    list.appendChild(row);
}

let browseTreeData = null;
let browseCurrentPath = '';

function browseLocalFolder(input) {
    if (!input.files.length) return;
    const status = document.getElementById('browse-url-status');
    const label = document.getElementById('browse-file-label');

    const files = Array.from(input.files);
    const rootFolder = files[0].webkitRelativePath.split('/')[0];
    label.textContent = rootFolder;

    const structure = {};
    let totalSize = 0;
    files.forEach(f => {
        totalSize += f.size;
        const parts = f.webkitRelativePath.split('/');
        let current = structure;
        for (let i = 1; i < parts.length; i++) {
            if (i === parts.length - 1) {
                current[parts[i]] = { _size: f.size, _isFile: true };
            } else {
                if (!current[parts[i]]) current[parts[i]] = {};
                current = current[parts[i]];
            }
        }
    });

    const sizeMB = (totalSize / (1024 * 1024)).toFixed(1);
    document.getElementById('url-game-size').value = sizeMB;

    status.style.display = 'block';
    status.style.color = 'var(--accent)';
    status.textContent = rootFolder + ' — ' + sizeMB + ' MB — ' + files.length + ' files';

    const container = document.getElementById('url-tree-container');
    const tree = document.getElementById('url-tree');
    container.style.display = 'block';
    tree.innerHTML = '';
    document.getElementById('selected-folder-label').textContent = rootFolder;
    document.getElementById('selected-exe-label').textContent = 'none';

    renderLocalTree(structure, tree, 0, rootFolder);
}

function renderLocalTree(obj, parent, depth, basePath) {
    const entries = Object.keys(obj).sort((a, b) => {
        const aIsFile = obj[a]._isFile;
        const bIsFile = obj[b]._isFile;
        if (aIsFile !== bIsFile) return aIsFile ? 1 : -1;
        return a.localeCompare(b);
    });

    entries.forEach(name => {
        const item = obj[name];
        const isFile = item._isFile;
        const div = document.createElement('div');
        const pad = depth * 16;
        div.style.cssText = 'padding-left:' + pad + 'px;cursor:pointer;padding:3px 6px;margin:1px 0;border-radius:3px;font-size:12px;';

        if (isFile) {
            const isExe = name.toLowerCase().endsWith('.exe');
            const size = item._size;
            const sizeStr = size > 1024*1024 ? ' (' + (size / (1024*1024)).toFixed(1) + ' MB)' : (size > 1024 ? ' (' + (size / 1024).toFixed(0) + ' KB)' : '');
            div.textContent = (isExe ? '🎮 ' : '📄 ') + name + sizeStr;
            div.style.color = isExe ? 'var(--accent)' : 'var(--text)';
            div.onmouseenter = () => div.style.background = 'rgba(0,255,136,0.15)';
            div.onmouseleave = () => { if (!div.dataset.selected) div.style.background = 'transparent'; };
            div.dataset.type = 'exe';
            div.onclick = () => {
                document.querySelectorAll('#url-tree div[data-type="exe"]').forEach(d => { d.style.background = 'transparent'; d.dataset.selected = ''; });
                div.style.background = 'rgba(0,255,136,0.3)';
                div.dataset.selected = '1';
                document.getElementById('url-game-exe').value = name;
                document.getElementById('selected-exe-label').textContent = name;
            };
        } else {
            div.textContent = '📁 ' + name + '/';
            div.style.color = 'var(--text)';
            div.onmouseenter = () => div.style.background = 'rgba(0,255,136,0.1)';
            div.onmouseleave = () => { if (!div.dataset.selected) div.style.background = 'transparent'; };
            div.dataset.type = 'folder';
            div.onclick = () => {
                document.querySelectorAll('#url-tree div[data-type="folder"]').forEach(d => { d.style.background = 'transparent'; d.dataset.selected = ''; });
                div.style.background = 'rgba(0,255,136,0.3)';
                div.dataset.selected = '1';
                document.getElementById('selected-folder-label').textContent = basePath + '/' + name;
                document.getElementById('selected-exe-label').textContent = 'none';
                document.getElementById('url-game-exe').value = '';
            };
        }
        parent.appendChild(div);

        if (!isFile) {
            renderLocalTree(item, parent, depth + 1, basePath + '/' + name);
        }
    });
}

function addGameURL() {
    const name = document.getElementById('url-game-name').value.trim();
    const version = document.getElementById('url-game-version').value.trim();
    const category = document.getElementById('url-game-category').value;
    const status = document.getElementById('url-game-status').value;
    const desc = document.getElementById('url-game-desc').value.trim();
    const tags = document.getElementById('url-game-tags').value.trim();
    const exe_path = document.getElementById('url-game-exe').value.trim();
    if (!name) { alert('Game name is required'); return; }
    if (status !== 'coming_soon' && !exe_path) { alert('Exe path is required'); return; }
    const urls = [];
    document.querySelectorAll('#url-download-list .url-row').forEach(row => {
        const link = row.querySelector('.url-link-input').value.trim();
        if (link) urls.push({ url: link, label: row.querySelector('.url-label-input').value.trim() || 'Mirror' });
    });
    if (status !== 'coming_soon' && urls.length === 0) { alert('Add at least one download URL'); return; }
    const formData = new FormData();
    formData.append('name', name);
    formData.append('version', version || '1.0.0');
    formData.append('category', category);
    formData.append('description', desc);
    formData.append('tags', tags);
    formData.append('download_url', urls[0].url);
    formData.append('exe_path', exe_path);
    formData.append('status', document.getElementById('url-game-status').value);
    const fileSizeMB = parseFloat(document.getElementById('url-game-size').value) || 0;
    formData.append('file_size', String(Math.round(fileSizeMB * 1024 * 1024)));
    if (urls.length > 1) formData.append('mirror_urls', JSON.stringify(urls.slice(1)));
    const coverFile = document.getElementById('url-game-cover').files[0];
    const bgFile = document.getElementById('url-game-bg').files[0];
    if (coverFile) formData.append('cover', coverFile);
    if (bgFile) formData.append('background', bgFile);
    fetch('/api/games/url', {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + authToken },
        body: formData
    }).then(r => r.json()).then(data => {
        if (data.id) {
            document.getElementById('url-game-name').value = '';
            document.getElementById('url-game-desc').value = '';
            document.getElementById('url-game-tags').value = '';
            document.getElementById('url-game-exe').value = '';
            document.getElementById('url-download-list').innerHTML = `
                <div class="url-row" style="display:flex;gap:8px;margin-bottom:8px">
                    <input type="text" class="url-link-input" placeholder="https://example.com/game.zip" style="flex:3">
                    <input type="text" class="url-label-input" placeholder="Label" style="flex:1" value="Main">
                </div>`;
            urlStepNext(1);
            alert('Game "' + name + '" added with ' + urls.length + ' download link(s)!');
        } else {
            alert(data.error || 'Failed to add game');
        }
    }).catch(e => alert('Error: ' + e));
}

function loadReportsAdmin() {
    api('GET', '/api/reports').then(data => {
        const list = document.getElementById('reports-list');
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No reports yet</p>';
            return;
        }
        list.innerHTML = '';
        data.forEach(r => {
            const statusColor = r.status === 'open' ? '#ff4444' : r.status === 'replied' ? '#00ff88' : '#ffaa00';
            list.innerHTML += `
                <div style="background:#111;border:1px solid #2a2a2a;border-radius:8px;padding:16px;margin-bottom:12px">
                    <div style="display:flex;justify-content:space-between;margin-bottom:8px">
                        <span style="color:#00ff88;font-weight:bold">${r.game_name}</span>
                        <span style="color:${statusColor};font-size:12px;text-transform:uppercase">${r.status}</span>
                    </div>
                    <p style="color:#888;font-size:12px;margin:4px 0">By ${r.username} | ${r.created_at}</p>
                    <p style="color:#ccc;font-size:13px;margin:8px 0">${r.description}</p>
                    ${r.admin_reply ? `<p style="color:#00ff88;font-size:12px;margin:8px 0"><b>Reply:</b> ${r.admin_reply}</p>` : ''}
                    <div style="display:flex;gap:8px;margin-top:8px">
                        <input type="text" id="report-reply-${r.id}" placeholder="Admin reply..." style="flex:1;padding:6px 10px;background:#1a1a1a;border:1px solid #333;border-radius:4px;color:#e0e0e0;font-size:12px">
                        <button class="btn btn-sm" onclick="replyReport(${r.id})" style="padding:6px 12px;font-size:12px">REPLY</button>
                        <button class="btn btn-sm" onclick="resolveReport(${r.id})" style="padding:6px 12px;font-size:12px;background:#00ff88;color:#000">RESOLVED</button>
                    </div>
                </div>`;
        });
    });
}

function replyReport(id) {
    const reply = document.getElementById('report-reply-' + id).value.trim();
    if (!reply) { alert('Enter a reply'); return; }
    api('POST', '/api/admin/reports/reply', { report_id: id, reply: reply, status: 'replied' }).then(() => loadReportsAdmin());
}

function resolveReport(id) {
    api('POST', '/api/admin/reports/reply', { report_id: id, reply: '', status: 'resolved' }).then(() => loadReportsAdmin());
}

function logout() {
    authToken = '';
    localStorage.removeItem('mk_token');
    document.getElementById('admin-screen').classList.remove('active');
    document.getElementById('admin-screen').classList.add('hidden');
    document.getElementById('auth-screen').classList.remove('hidden');
    document.getElementById('auth-screen').classList.add('active');
}

function showTab(tab, event) {
    document.querySelectorAll('.tab').forEach(t => { t.classList.add('hidden'); t.classList.remove('active'); });
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));

    document.getElementById('tab-' + tab).classList.remove('hidden');
    document.getElementById('tab-' + tab).classList.add('active');

    if (event && event.currentTarget) event.currentTarget.classList.add('active');

    switch(tab) {
        case 'dashboard': loadDashboard(); break;
        case 'games': loadGames(); break;
        case 'add-game-url': loadURLGameCategories(); break;
        case 'archives': loadArchives(); break;
        case 'admins': loadAdmins(); break;
        case 'users': loadUsers(); break;
        case 'news': loadNewsAdmin(); break;
        case 'categories': loadCategoriesAdmin(); break;
        case 'featured': loadFeaturedAdmin(); break;
        case 'mods': loadModsAdmin(); break;
        case 'servers': loadServersAdmin(); break;
        case 'reports': loadReportsAdmin(); break;
        case 'launcher': loadLauncherAdmin(); break;
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
        allGames = Array.isArray(data) ? data : [];
        renderGames(allGames);
    }).catch(e => {
        allGames = [];
        renderGames([]);
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
        const safeName = g.name.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '&quot;');
        grid.innerHTML += `<div class="game-card">
            ${coverImg}
            <div class="game-card-body">
                <h4>${g.name}</h4>
                <p>v${g.version} | ${g.category || 'Uncategorized'}</p>
                <p>${g.download_count} downloads</p>
            </div>
            <div class="game-card-actions">
                <button class="btn btn-primary btn-sm" onclick="updateGameVersion(${g.id})">UPDATE</button>
                <button class="btn btn-secondary btn-sm" onclick="editGame(${g.id})">EDIT</button>
                <button class="btn btn-danger btn-sm" onclick="deleteGame(${g.id}, '${safeName}')">DELETE</button>
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
    if (!confirm('Delete "' + name + '" and all its files?\n\nThis cannot be undone.')) return;
    api('DELETE', '/api/games/' + id).then(data => {
        if (data.error) { alert('Error: ' + data.error); return; }
        alert('"' + name + '" deleted');
        loadGames();
    }).catch(e => {
        alert('Delete failed: ' + e.message);
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

function updateGameVersion(id) {
    const game = allGames.find(g => g.id === id);
    if (!game) return;
    const modal = document.getElementById('edit-modal');
    const form = document.getElementById('edit-form');
    form.innerHTML = `
        <h3 style="color:var(--accent);margin-bottom:16px">UPDATE VERSION: ${game.name}</h3>
        <p style="color:var(--text-secondary);margin-bottom:16px">Current version: v${game.version}</p>
        <div class="form-group"><label>NEW VERSION *</label><input type="text" id="new-version" placeholder="e.g. 1.1.0"></div>
        <div class="form-group"><label>ARCHIVE FILE *</label><input type="file" id="new-version-archive" accept=".zip,.rar,.7z,.tar,.gz,.tar.gz,.tgz"></div>
        <div id="version-progress" class="progress-container hidden">
            <div class="progress-bar"><div class="progress-fill" id="version-progress-fill"></div></div>
            <p id="version-progress-text">Uploading...</p>
        </div>
        <button class="btn btn-primary" id="upload-version-btn" onclick="uploadGameVersion(${game.id})">UPLOAD VERSION</button>`;
    modal.classList.remove('hidden');
}

/* Add Game Form */
let currentStep = 1;

function nextStep(step) {
    if (step === 2 && !document.getElementById('game-name').value.trim()) {
        alert('Game name is required');
        return;
    }
    if (step === 2 && document.getElementById('game-status').value === 'coming_soon') {
        step = 3;
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

function handleFolderSelect(input) {
    if (input.files.length === 0) return;
    const firstPath = input.files[0].webkitRelativePath || input.files[0].name;
    const root = firstPath.split('/')[0];
    document.getElementById('game-folder').value = root;
    document.getElementById('archive-name').textContent = `${input.files.length} files from ${root}`;
    document.getElementById('archive-size').textContent = formatSize(Array.from(input.files).reduce((sum, file) => sum + file.size, 0));
    document.getElementById('archive-info').classList.remove('hidden');
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
    const folderFiles = document.getElementById('game-folder-files').files;
    openBrowseModal(path => {
        document.getElementById('game-folder').value = path;
    });
    if (folderFiles.length > 0) {
        openLocalBrowse(folderFiles);
    } else {
        browsePath('/storage/archives');
    }
}

function browseForExe() {
    openBrowseModal(path => {
        if (path.toLowerCase().endsWith('.exe')) {
            document.getElementById('game-exe').value = path;
        } else {
            alert('Please select a .exe file');
        }
    });
    const folderFiles = document.getElementById('game-folder-files').files;
    if (folderFiles.length > 0) {
        openLocalBrowse(folderFiles);
    } else {
        browsePath('/storage/archives');
    }
}

function openBrowseModal(callback) {
    browseCallback = callback;
    const modal = document.getElementById('browse-modal');
    modal.classList.remove('hidden');
}

let localBrowseFiles = [];

function openLocalBrowse(files) {
    localBrowseFiles = Array.from(files);
    const root = (localBrowseFiles[0].webkitRelativePath || localBrowseFiles[0].name).split('/')[0];
    browseLocalPath(root);
}

function browseLocalPath(path) {
    currentBrowsePath = path;
    document.getElementById('browse-path').textContent = path + ' (selected folder)';
    const items = document.getElementById('browse-items');
    items.innerHTML = '';
    const folders = new Map();
    const visibleFiles = [];

    for (const file of localBrowseFiles) {
        const relative = file.webkitRelativePath || file.name;
        if (relative === path || !relative.startsWith(path + '/')) continue;
        const remainder = relative.substring(path.length + 1);
        const parts = remainder.split('/');
        const child = parts[0];
        if (parts.length > 1) {
            folders.set(child, path + '/' + child);
        } else {
            visibleFiles.push({name: child, path: relative, isDir: false, size: file.size});
        }
    }

    const entries = [
        ...Array.from(folders, ([name, path]) => ({name, path, isDir: true, size: 0})),
        ...visibleFiles,
    ];
    if (entries.length === 0) {
        items.innerHTML = '<p style="color:var(--text-dim)">No files found</p>';
        return;
    }
    if (path.includes('/')) {
        const parent = path.substring(0, path.lastIndexOf('/'));
        items.innerHTML += `<div class="browse-item" onclick="browseLocalPath('${parent.replace(/'/g, "\\'")}')">
            <span class="icon">&#128194;</span><span class="item-name">..</span>
        </div>`;
    }
    entries.forEach(item => {
        const escapedPath = item.path.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
        const clickAction = item.isDir
            ? `browseLocalPath('${escapedPath}')`
            : `selectBrowseItem('${escapedPath}')`;
        items.innerHTML += `<div class="browse-item" onclick="${clickAction}">
            <span class="icon">${item.isDir ? '&#128193;' : '&#128196;'}</span>
            <span class="item-name">${item.name}</span>
            <span class="item-size">${item.isDir ? '' : formatSize(item.size)}</span>
        </div>`;
    });
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

        if (data.error) {
            items.innerHTML = `<p style="color:var(--danger)">${data.error}</p>`;
            return;
        }

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
                const isArchive = /\.(zip|rar|7z|tar|gz|tgz)$/i.test(item.name);
                const clickAction = item.isDir || isArchive
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
    const status = document.getElementById('game-status').value;
    const tags = document.getElementById('game-tags').value;
    const folder = document.getElementById('game-folder').value;
    const exe = document.getElementById('game-exe').value;
    summary.innerHTML = `
        <p><span class="label">Name: </span>${name}</p>
        <p><span class="label">Description: </span>${desc || 'None'}</p>
        <p><span class="label">Version: </span>${ver}</p>
        <p><span class="label">Category: </span>${cat || 'None'}</p>
        <p><span class="label">Status: </span>${status}</p>
        <p><span class="label">Tags: </span>${tags || 'None'}</p>
        <p><span class="label">Folder files: </span>${document.getElementById('game-folder-files').files.length}</p>
        <p><span class="label">Game Folder: </span>${folder || 'Root'}</p>
        <p><span class="label">Executable: </span>${exe || 'None'}</p>`;
}

let currentUploadSession = '';

async function uploadFileWithRetry(session, file, path, index, total) {
    const chunkSize = 8 * 1024 * 1024;
    let offset = 0;
    for (let attempt = 1; attempt <= 5; attempt++) {
        try {
            if (offset >= file.size) {
                filesUploadedBytes += file.size;
                return;
            }
            const chunk = file.slice(offset, Math.min(offset + chunkSize, file.size));
            const form = new FormData();
            form.append('path', path);
            form.append('offset', String(offset));
            form.append('total_size', String(file.size));
            form.append('file', chunk, path);
            const response = await fetch(API + '/api/uploads/' + encodeURIComponent(session) + '/file', {
                method: 'POST',
                headers: { 'Authorization': 'Bearer ' + authToken },
                body: form
            });
            const responseText = await response.text();
            let data = {};
            try {
                data = responseText ? JSON.parse(responseText) : {};
            } catch (parseError) {
                throw new Error(responseText || `HTTP ${response.status}`);
            }
            if (!response.ok || data.error) {
                if (response.status === 409 && Number.isFinite(data.offset)) {
                    offset = data.offset;
                    continue;
                }
                throw new Error(data.error || `HTTP ${response.status}`);
            }
            offset = Number(data.offset);
            const completed = filesUploadedBytes + offset;
            document.getElementById('add-progress-fill').style.width = Math.round(completed / filesTotalBytes * 100) + '%';
            document.getElementById('add-progress-text').textContent = `Uploading ${index + 1} / ${total}: ${path}`;
            if (offset >= file.size) {
                filesUploadedBytes += file.size;
                return;
            }
            attempt = 0;
        } catch (error) {
            if (attempt === 5) throw error;
            document.getElementById('add-progress-text').textContent = `Connection lost. Retrying ${path} (${attempt}/5)...`;
            await new Promise(resolve => setTimeout(resolve, Math.min(15000, 1000 * 2 ** (attempt - 1))));
        }
    }
}

let filesUploadedBytes = 0;
let filesTotalBytes = 0;

async function uploadImageFile(session, fileInputId, pathPrefix) {
    const input = document.getElementById(fileInputId);
    if (!input || !input.files || !input.files[0]) return '';
    const file = input.files[0];
    const ext = file.name.split('.').pop();
    const path = pathPrefix + '.' + ext;
    const form = new FormData();
    form.append('path', path);
    form.append('offset', '0');
    form.append('total_size', String(file.size));
    form.append('file', file, path);
    const response = await fetch(API + '/api/uploads/' + encodeURIComponent(session) + '/file', {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + authToken },
        body: form
    });
    const data = await response.json();
    if (!response.ok || data.error) throw new Error(data.error || 'Image upload failed');
    return path;
}

async function submitGame() {
    const btn = document.getElementById('add-game-btn');
    const progress = document.getElementById('add-progress');
    btn.disabled = true;
    progress.classList.remove('hidden');

    const folderFiles = document.getElementById('game-folder-files').files;
    const isComingSoon = document.getElementById('game-status').value === 'coming_soon';
    if (folderFiles.length === 0 && !isComingSoon) {
        alert('Select a game folder first');
        btn.disabled = false;
        progress.classList.add('hidden');
        return;
    }
    const session = currentUploadSession || (crypto.randomUUID ? crypto.randomUUID() : Date.now().toString(36) + Math.random().toString(36).slice(2));
    currentUploadSession = session;

    try {
        if (!isComingSoon) {
            const files = Array.from(folderFiles);
            const paths = files.map(file => file.webkitRelativePath || file.name);
            filesUploadedBytes = 0;
            filesTotalBytes = files.reduce((totalBytes, file) => totalBytes + file.size, 0);
            document.getElementById('add-progress-text').textContent = 'Uploading game files...';
            for (let index = 0; index < files.length; index++) {
                await uploadFileWithRetry(session, files[index], paths[index], index, files.length);
            }
        }

        document.getElementById('add-progress-text').textContent = 'Uploading images...';
        const coverPath = await uploadImageFile(session, 'cover-file', 'cover');
        const bgPath = await uploadImageFile(session, 'bg-file', 'bg');
        const logoPath = await uploadImageFile(session, 'logo-file', 'logo');
        const widePath = await uploadImageFile(session, 'wide-file', 'wide');

        const response = await fetch(API + '/api/uploads/' + encodeURIComponent(session) + '/finalize', {
            method: 'POST',
            headers: { 'Authorization': 'Bearer ' + authToken, 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name: document.getElementById('game-name').value,
                description: document.getElementById('game-desc').value,
                version: document.getElementById('game-version').value,
                category: document.getElementById('game-category').value,
                tags: document.getElementById('game-tags').value,
                game_folder: document.getElementById('game-folder').value,
                exe_path: document.getElementById('game-exe').value,
                status: document.getElementById('game-status').value,
                files: isComingSoon ? [] : paths,
                cover_path: coverPath,
                bg_path: bgPath,
                logo_path: logoPath,
                wide_path: widePath
            })
        });
        const data = await response.json();
        if (!response.ok || data.error) throw new Error(data.error || `HTTP ${response.status}`);
        alert('Game "' + data.name + '" added successfully!');
        currentUploadSession = '';
        resetAddForm();
        showTab('games');
    } catch (error) {
        alert('Upload paused: ' + error.message + '. Submit again to retry missing files.');
    } finally {
        btn.disabled = false;
        progress.classList.add('hidden');
    }
}

function resetAddForm() {
    document.getElementById('game-name').value = '';
    document.getElementById('game-desc').value = '';
    document.getElementById('game-version').value = '1.0.0';
    document.getElementById('game-category').value = '';
    document.getElementById('game-status').value = 'released';
    document.getElementById('game-tags').value = '';
    document.getElementById('game-folder').value = '';
    document.getElementById('game-exe').value = '';
    document.getElementById('game-folder-files').value = '';
    currentUploadSession = '';
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
            const safeName = a.name.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
            list.innerHTML += `<div class="archive-item">
                <span class="name">${a.name}</span>
                <span class="size">${formatSize(a.size)}</span>
                <button class="btn btn-danger btn-sm" onclick="deleteArchive('${safeName}')">DELETE</button>
            </div>`;
        });
    });
}

function deleteArchive(name) {
    if (!confirm('Delete archive "' + name + '"?\n\nThis cannot be undone.')) return;
    api('POST', '/api/archives/delete', { name: name }).then(data => {
        if (data.error) { alert('Error: ' + data.error); return; }
        alert('Archive deleted');
        loadArchives();
    }).catch(e => {
        alert('Delete failed: ' + e.message);
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
/* Users */
function loadUsers() {
    api('GET', '/api/admin/users').then(data => {
        const list = document.getElementById('users-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No users registered yet</p>';
            return;
        }
        data.forEach(u => {
            const statusColor = u.is_banned ? 'var(--danger)' : 'var(--accent)';
            const statusText = u.is_banned ? 'BANNED' : 'Active';
            const banBtn = u.is_banned
                ? `<button class="btn btn-primary btn-sm" onclick="toggleBan(${u.id}, false)">UNBAN</button>`
                : `<button class="btn btn-danger btn-sm" onclick="toggleBan(${u.id}, true)">BAN</button>`;
            list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                <div>
                    <span class="name">${u.username}</span>
                    <span style="color:${statusColor};font-size:11px;margin-left:8px">${statusText}</span>
                    <span style="color:var(--text-dim);font-size:11px;margin-left:8px">${u.display_name || ''}</span>
                </div>
                ${banBtn}
            </div>`;
        });
    });
}

function searchUsers() {
    const q = document.getElementById('user-search').value.trim();
    if (!q) { loadUsers(); return; }
    api('GET', '/api/users/search?q=' + encodeURIComponent(q)).then(data => {
        const list = document.getElementById('users-list');
        list.innerHTML = '';
        if (!data || data.length === 0) {
            list.innerHTML = '<p style="color:var(--text-dim)">No users found</p>';
            return;
        }
        data.forEach(u => {
            const statusColor = u.is_banned ? 'var(--danger)' : 'var(--accent)';
            const statusText = u.is_banned ? 'BANNED' : 'Active';
            const banBtn = u.is_banned
                ? `<button class="btn btn-primary btn-sm" onclick="toggleBan(${u.id}, false)">UNBAN</button>`
                : `<button class="btn btn-danger btn-sm" onclick="toggleBan(${u.id}, true)">BAN</button>`;
            list.innerHTML += `<div class="archive-item" style="justify-content:space-between">
                <div>
                    <span class="name">${u.username}</span>
                    <span style="color:${statusColor};font-size:11px;margin-left:8px">${statusText}</span>
                </div>
                ${banBtn}
            </div>`;
        });
    });
}

function toggleBan(userId, ban) {
    const action = ban ? 'ban' : 'unban';
    if (!confirm(`Are you sure you want to ${action} this user?`)) return;
    api('POST', '/api/admin/users/' + userId + '/ban', { banned: ban }).then(() => {
        loadUsers();
    });
}

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

function wipeServer() {
    if (!confirm('WARNING: This will permanently delete ALL games, archives, covers, and reset the database.\n\nThis CANNOT be undone. Type "yes" in the next prompt to confirm.')) return;
    const confirm2 = prompt('Type YES to confirm full server wipe:');
    if (confirm2 !== 'YES') { alert('Wipe cancelled.'); return; }
    api('POST', '/api/server/wipe').then(data => {
        if (data.error) { alert('Error: ' + data.error); return; }
        alert('Server wiped successfully. Refreshing...');
        location.reload();
    }).catch(e => {
        alert('Wipe failed: ' + e.message);
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

    const progress = document.getElementById('version-progress');
    const btn = document.getElementById('upload-version-btn');
    if (progress) progress.classList.remove('hidden');
    if (btn) btn.disabled = true;

    const form = new FormData();
    form.append('version', version);
    form.append('archive', archive);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', API + '/api/games/' + id + '/versions');
    xhr.setRequestHeader('Authorization', 'Bearer ' + authToken);

    xhr.upload.onprogress = function(e) {
        if (e.lengthComputable && progress) {
            const pct = Math.round(e.loaded / e.total * 100);
            const fill = document.getElementById('version-progress-fill');
            const text = document.getElementById('version-progress-text');
            if (fill) fill.style.width = pct + '%';
            if (text) text.textContent = `Uploading... ${pct}% (${formatSize(e.loaded)} / ${formatSize(e.total)})`;
        }
    };

    xhr.onload = function() {
        if (btn) btn.disabled = false;
        if (progress) progress.classList.add('hidden');
        try {
            const data = JSON.parse(xhr.responseText);
            if (data.error) { alert(data.error); return; }
            alert('Version uploaded: ' + version);
            closeEditModal();
            loadGames();
        } catch(e) {
            alert('Version upload failed');
        }
    };

    xhr.onerror = function() {
        if (btn) btn.disabled = false;
        if (progress) progress.classList.add('hidden');
        alert('Version upload failed');
    };

    xhr.send(form);
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
