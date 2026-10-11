// 备忘录系统 - 前端应用
const API_BASE = '/api';

// 状态管理
const state = {
    currentView: 'notes', // notes, trash, editor
    currentNoteId: null,
    currentTagFilter: null,
    searchKeyword: '',
    notes: [],
    tags: [],
    trashNotes: [],
    pagination: { page: 1, page_size: 10, total: 0, total_page: 1 },
    trashPagination: { page: 1, page_size: 10, total: 0, total_page: 1 },
    user: null,
    currentNote: null,
};

// 工具函数
function getToken() {
    return localStorage.getItem('token');
}

function setToken(token) {
    localStorage.setItem('token', token);
}

function clearToken() {
    localStorage.removeItem('token');
    localStorage.removeItem('username');
}

function showToast(msg, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.className = 'toast ' + type;
    toast.style.display = 'block';
    setTimeout(() => { toast.style.display = 'none'; }, 3000);
}

function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

function formatDate(dateStr) {
    const d = new Date(dateStr);
    return d.getFullYear() + '-' + 
        String(d.getMonth() + 1).padStart(2, '0') + '-' + 
        String(d.getDate()).padStart(2, '0') + ' ' +
        String(d.getHours()).padStart(2, '0') + ':' +
        String(d.getMinutes()).padStart(2, '0');
}

// API 请求
async function apiRequest(path, options = {}) {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
    };
    if (token) {
        headers['Authorization'] = 'Bearer ' + token;
    }

    const res = await fetch(API_BASE + path, {
        ...options,
        headers,
    });

    if (res.status === 401) {
        clearToken();
        window.location.href = '/login';
        return null;
    }

    const data = await res.json();
    if (!res.ok) {
        throw new Error(data.error || '请求失败');
    }
    return data;
}

// 初始化
function init() {
    const token = getToken();
    if (!token) {
        window.location.href = '/login';
        return;
    }
    state.user = { username: localStorage.getItem('username') };
    renderApp();
    loadTags();
    loadNotes();
}

// 渲染主应用
function renderApp() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="app-container">
            <aside class="sidebar">
                 <div class="sidebar-header">
                    <div class="logo-mini">📝</div>
                    <h2>备忘录</h2>
                </div>
                <div class="user-info">
                    <div class="user-avatar" id="userAvatar">${(state.user?.username || 'U').charAt(0).toUpperCase()}</div>
                    <div class="user-detail">
                        <div class="username">${escapeHtml(state.user?.username || '用户')}</div>
                        <button class="logout-btn" onclick="handleLogout()">退出登录</button>
                    </div>
                </div>
                <nav class="sidebar-nav">
                    <div class="nav-section">
                        <div class="nav-item ${state.currentView === 'notes' && !state.currentTagFilter ? 'active' : ''}" onclick="switchView('notes')">
                            <span class="nav-icon">📄</span>
                            <span>全部笔记</span>
                        </div>
                        <div class="nav-item" onclick="showNewTagModal()">
                            <span class="nav-icon">➕</span>
                            <span>新建标签</span>
                        </div>
                    </div>
                    <div class="nav-section">
                        <div class="nav-section-title">我的标签</div>
                        <div id="tagList"></div>
                    </div>
                    <div class="nav-section">
                        <div class="nav-section-title">其他</div>
                        <div class="nav-item ${state.currentView === 'trash' ? 'active' : ''}" onclick="switchView('trash')">
                            <span class="nav-icon">🗑️</span>
                            <span>回收站</span>
                        </div>
                    </div>
                </nav>
                <div class="sidebar-footer">
                    <button class="btn-new-note" onclick="createNewNote()">+ 新建笔记</button>
                </div>
            </aside>
            <main class="main-content" id="mainContent">
                <!-- 动态内容 -->
            </main>
        </div>
    `;
    renderTagList();
    renderMainContent();
}

function renderTagList() {
    const tagList = document.getElementById('tagList');
    if (!tagList) return;
    
    if (state.tags.length === 0) {
        tagList.innerHTML = '<div style="color:#bbb; font-size:13px; padding: 8px 12px;">暂无标签</div>';
        return;
    }
    
    tagList.innerHTML = state.tags.map(tag => `
        <div class="tag-item ${state.currentTagFilter === tag.id ? 'active' : ''}" onclick="filterByTag(${tag.id})">
            <span class="tag-dot" style="background:${tag.color}"></span>
            <span class="tag-name">${escapeHtml(tag.name)}</span>
            <span class="tag-count">${tag.note_count || 0}</span>
        </div>
    `).join('');
}

function renderMainContent() {
    const main = document.getElementById('mainContent');
    if (state.currentView === 'editor') {
        renderEditor(main);
    } else if (state.currentView === 'trash') {
        renderTrash(main);
    } else {
        renderNotesList(main);
    }
}

// 笔记列表视图
function renderNotesList(main) {
    let title = '全部笔记';
    if (state.currentTagFilter) {
        const tag = state.tags.find(t => t.id === state.currentTagFilter);
        if (tag) title = '#' + tag.name;
    }
    if (state.searchKeyword) title = '搜索: "' + state.searchKeyword + '"';

    main.innerHTML = `
        <div class="toolbar">
            <div class="search-box">
                <input type="text" id="searchInput" placeholder="搜索笔记标题或内容..." value="${escapeHtml(state.searchKeyword)}">
            </div>
            <div class="toolbar-actions">
                <button class="icon-btn primary" onclick="createNewNote()">+ 新建</button>
            </div>
        </div>
        <div class="notes-container">
            <div class="toolbar-title" style="margin-bottom: 20px; padding: 0 4px;">${title} (${state.pagination.total})</div>
            <div id="notesGrid" class="notes-grid"></div>
            <div id="pagination" class="pagination"></div>
        </div>
    `;

    const searchInput = document.getElementById('searchInput');
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            state.searchKeyword = e.target.value.trim();
            state.pagination.page = 1;
            loadNotes();
        }, 300);
    });

    const grid = document.getElementById('notesGrid');
    if (state.notes.length === 0) {
        grid.innerHTML = `
            <div class="empty-state" style="grid-column: 1/-1;">
                <div class="empty-state-icon">📝</div>
                <div class="empty-state-text">还没有笔记</div>
                <div class="empty-state-hint">点击"新建笔记"开始记录你的想法</div>
            </div>
        `;
    } else {
        grid.innerHTML = state.notes.map(note => {
            const tagsHtml = (note.tags || []).slice(0, 3).map(t => 
                `<span class="note-tag" style="background:${t.color || '#667eea'}">${escapeHtml(t.name)}</span>`
            ).join('');
            return `
                <div class="note-card" onclick="openNote(${note.id})">
                    ${note.is_public ? '<span class="public-badge">🌐 公开</span>' : ''}
                    <div class="note-card-title">${escapeHtml(note.title)}</div>
                    <div class="note-card-summary">${escapeHtml(note.summary || '暂无内容')}</div>
                    <div class="note-card-tags">${tagsHtml}</div>
                    <div class="note-card-footer">
                        <span>${formatDate(note.updated_at)}</span>
                        <div class="note-card-actions">
                            <button class="card-action-btn" onclick="event.stopPropagation(); shareNote(${note.id})">分享</button>
                            <button class="card-action-btn danger" onclick="event.stopPropagation(); deleteNote(${note.id})">删除</button>
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    }

    renderPagination('pagination', state.pagination, (page) => {
        state.pagination.page = page;
        loadNotes();
    });
}

function renderPagination(id, pagination, onChange) {
    const el = document.getElementById(id);
    if (!el || pagination.total_page <= 1) {
        if (el) el.innerHTML = '';
        return;
    }
    
    const { page, total_page } = pagination;
    let html = '';
    html += `<button class="page-btn" ${page <= 1 ? 'disabled' : ''} onclick="(${onChange.toString()})(${page - 1})">上一页</button>`;
    
    const maxBtns = 5;
    let start = Math.max(1, page - Math.floor(maxBtns / 2));
    let end = Math.min(total_page, start + maxBtns - 1);
    if (end - start + 1 < maxBtns) start = Math.max(1, end - maxBtns + 1);
    
    for (let i = start; i <= end; i++) {
        html += `<button class="page-btn ${i === page ? 'active' : ''}" onclick="(${onChange.toString()})(${i})">${i}</button>`;
    }
    
    html += `<button class="page-btn" ${page >= total_page ? 'disabled' : ''} onclick="(${onChange.toString()})(${page + 1})">下一页</button>`;
    el.innerHTML = html;
}

// 回收站视图
function renderTrash(main) {
    main.innerHTML = `
        <div class="toolbar">
            <div class="toolbar-title">🗑️ 回收站</div>
            <div class="toolbar-actions">
                <button class="icon-btn" onclick="switchView('notes')">← 返回笔记</button>
            </div>
        </div>
        <div class="notes-container">
            <div id="trashList"></div>
            <div id="trashPagination" class="pagination"></div>
        </div>
    `;

    const list = document.getElementById('trashList');
    if (state.trashNotes.length === 0) {
        list.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">✨</div>
                <div class="empty-state-text">回收站是空的</div>
                <div class="empty-state-hint">删除的笔记会暂时保存在这里</div>
            </div>
        `;
    } else {
        list.innerHTML = state.trashNotes.map(note => `
            <div class="trash-item">
                <div class="trash-item-info">
                    <div class="trash-item-title">${escapeHtml(note.title)}</div>
                    <div class="trash-item-deleted">删除于 ${formatDate(note.deleted_at)}</div>
                </div>
                <div class="trash-item-actions">
                    <button class="icon-btn success" onclick="restoreNote(${note.id})">恢复</button>
                    <button class="icon-btn danger" onclick="permanentDeleteNote(${note.id})">永久删除</button>
                </div>
            </div>
        `).join('');
    }

    renderPagination('trashPagination', state.trashPagination, (page) => {
        state.trashPagination.page = page;
        loadTrash();
    });
}

// 编辑器视图
function renderEditor(main) {
    const note = state.currentNote || { title: '', content: '', tags: [], is_public: false, share_token: '' };
    const tagsHtml = (note.tags || []).map(t => 
        `<span class="tag-chip" style="background:${t.color}">
            ${escapeHtml(t.name)}
            <span class="tag-chip-remove" onclick="removeTagFromNote(${t.id})">×</span>
        </span>`
    ).join('');

    main.innerHTML = `
        <div class="editor-container">
            <div class="editor-header">
                <button class="back-btn" onclick="backToList()">← 返回</button>
                <input type="text" class="editor-title-input" id="noteTitle" placeholder="笔记标题..." value="${escapeHtml(note.title)}">
                <div style="display:flex; gap:8px;">
                    <button class="icon-btn primary" onclick="saveNote()">💾 保存</button>
                </div>
            </div>
            <div class="editor-main">
                <div class="editor-content-wrapper">
                    <div class="editor-toolbar">
                        <button class="toolbar-btn" onclick="execCmd('bold')" title="粗体"><b>B</b></button>
                        <button class="toolbar-btn" onclick="execCmd('italic')" title="斜体"><i>I</i></button>
                        <button class="toolbar-btn" onclick="execCmd('underline')" title="下划线"><u>U</u></button>
                        <button class="toolbar-btn" onclick="execCmd('strikeThrough')" title="删除线"><s>S</s></button>
                        <div class="toolbar-divider"></div>
                        <button class="toolbar-btn" onclick="execCmd('formatBlock', 'h1')" title="标题1">H1</button>
                        <button class="toolbar-btn" onclick="execCmd('formatBlock', 'h2')" title="标题2">H2</button>
                        <button class="toolbar-btn" onclick="execCmd('formatBlock', 'h3')" title="标题3">H3</button>
                        <button class="toolbar-btn" onclick="execCmd('formatBlock', 'p')" title="正文">P</button>
                        <div class="toolbar-divider"></div>
                        <button class="toolbar-btn" onclick="execCmd('insertUnorderedList')" title="无序列表">• 列表</button>
                        <button class="toolbar-btn" onclick="execCmd('insertOrderedList')" title="有序列表">1. 列表</button>
                        <button class="toolbar-btn" onclick="execCmd('formatBlock', 'blockquote')" title="引用">❝ 引用</button>
                        <div class="toolbar-divider"></div>
                        <button class="toolbar-btn" onclick="execCmd('justifyLeft')" title="左对齐">⬅</button>
                        <button class="toolbar-btn" onclick="execCmd('justifyCenter')" title="居中">⬌</button>
                        <button class="toolbar-btn" onclick="execCmd('justifyRight')" title="右对齐">➡</button>
                        <div class="toolbar-divider"></div>
                        <button class="toolbar-btn" onclick="insertLink()" title="插入链接">🔗</button>
                        <button class="toolbar-btn" onclick="insertImageBtn()" title="插入图片">🖼️</button>
                        <div class="toolbar-divider"></div>
                        <button class="toolbar-btn" onclick="execCmd('undo')" title="撤销">↶</button>
                        <button class="toolbar-btn" onclick="execCmd('redo')" title="重做">↷</button>
                    </div>
                    <div class="editor-content" id="editorContent" contenteditable="true" data-placeholder="开始记录你的想法..."></div>
                </div>
                <aside class="editor-sidebar">
                    <h3>标签</h3>
                    <div class="tags-editor" id="noteTagsEditor">${tagsHtml}</div>
                    <select class="tag-select" id="addTagSelect" onchange="addTagToNote(this.value); this.value=''; ">
                        <option value="">+ 添加标签</option>
                        ${state.tags.filter(t => !(note.tags || []).find(nt => nt.id === t.id)).map(t => 
                            `<option value="${t.id}">${escapeHtml(t.name)}</option>`
                        ).join('')}
                    </select>
                    
                    <h3 style="margin-top: 24px;">分享设置</h3>
                    <div class="setting-row">
                        <span class="setting-label">公开分享</span>
                        <label class="switch">
                            <input type="checkbox" id="publicSwitch" ${note.is_public ? 'checked' : ''} onchange="togglePublic()">
                            <span class="slider"></span>
                        </label>
                    </div>
                    ${note.is_public && note.share_token ? `
                        <div class="share-info">
                            <div class="share-link">${window.location.origin}/share/${note.share_token}</div>
                            <button class="share-copy-btn" onclick="copyShareLink('${note.share_token}')">复制链接</button>
                        </div>
                    ` : ''}
                    
                    <h3 style="margin-top: 24px;">笔记信息</h3>
                    <div class="note-meta">
                        ${note.id ? `
                            <div>创建: ${formatDate(note.created_at)}</div>
                            <div>更新: ${formatDate(note.updated_at)}</div>
                            <div>浏览: ${note.view_count || 0} 次</div>
                        ` : '<div style="color:#bbb;">尚未保存</div>'}
                    </div>
                </aside>
            </div>
        </div>
    `;

    document.getElementById('editorContent').innerHTML = note.content || '';
}

function execCmd(cmd, value = null) {
    document.execCommand(cmd, false, value);
    document.getElementById('editorContent').focus();
}

function insertLink() {
    const url = prompt('请输入链接地址:', 'https://');
    if (url) {
        execCmd('createLink', url);
    }
}

function insertImageBtn() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/*';
    input.onchange = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        
        const formData = new FormData();
        formData.append('file', file);
        
        try {
            const token = getToken();
            const res = await fetch(API_BASE + '/upload/image', {
                method: 'POST',
                headers: { 'Authorization': 'Bearer ' + token },
                body: formData,
            });
            const data = await res.json();
            
            if (res.ok) {
                execCmd('insertImage', data.url);
                showToast('图片上传成功');
            } else {
                showToast(data.error || '上传失败', 'error');
            }
        } catch (err) {
            showToast('上传失败: ' + err.message, 'error');
        }
    };
    input.click();
}

// 数据加载
async function loadNotes() {
    try {
        let url = `/notes?page=${state.pagination.page}&page_size=${state.pagination.page_size}`;
        if (state.searchKeyword) url += '&keyword=' + encodeURIComponent(state.searchKeyword);
        if (state.currentTagFilter) url += '&tag_id=' + state.currentTagFilter;
        
        const data = await apiRequest(url);
        if (data) {
            state.notes = data.notes;
            state.pagination = data.pagination;
            renderMainContent();
        }
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function loadTrash() {
    try {
        const data = await apiRequest(`/trash?page=${state.trashPagination.page}&page_size=${state.trashPagination.page_size}`);
        if (data) {
            state.trashNotes = data.notes;
            state.trashPagination = data.pagination;
            renderMainContent();
        }
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function loadTags() {
    try {
        const data = await apiRequest('/tags');
        if (data) {
            state.tags = data.tags;
            renderTagList();
        }
    } catch (err) {
        console.error('加载标签失败:', err);
    }
}

// 操作函数
function switchView(view) {
    state.currentView = view;
    state.currentNoteId = null;
    state.currentNote = null;
    if (view === 'notes') {
        state.currentTagFilter = null;
        state.searchKeyword = '';
        state.pagination.page = 1;
        loadNotes();
    } else if (view === 'trash') {
        state.trashPagination.page = 1;
        loadTrash();
    }
    renderApp();
}

function filterByTag(tagId) {
    state.currentView = 'notes';
    state.currentTagFilter = tagId;
    state.pagination.page = 1;
    renderApp();
    loadNotes();
}

function createNewNote() {
    state.currentView = 'editor';
    state.currentNoteId = null;
    state.currentNote = { title: '', content: '', tags: [], is_public: false, share_token: '' };
    renderApp();
    setTimeout(() => document.getElementById('noteTitle').focus(), 100);
}

async function openNote(id) {
    try {
        const data = await apiRequest('/notes/' + id);
        if (data) {
            state.currentView = 'editor';
            state.currentNoteId = id;
            state.currentNote = data.note;
            renderApp();
        }
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function backToList() {
    state.currentView = 'notes';
    state.currentNoteId = null;
    state.currentNote = null;
    renderApp();
    loadNotes();
}

async function saveNote() {
    const title = document.getElementById('noteTitle').value.trim();
    const content = document.getElementById('editorContent').innerHTML;
    
    if (!title) {
        showToast('请输入笔记标题', 'error');
        return;
    }

    const tagIds = (state.currentNote?.tags || []).map(t => t.id);
    const isPublic = document.getElementById('publicSwitch')?.checked || false;

    try {
        let data;
        if (state.currentNoteId) {
            data = await apiRequest('/notes/' + state.currentNoteId, {
                method: 'PUT',
                body: JSON.stringify({ title, content, tag_ids: tagIds, is_public: isPublic }),
            });
        } else {
            data = await apiRequest('/notes', {
                method: 'POST',
                body: JSON.stringify({ title, content, tag_ids: tagIds, is_public: isPublic }),
            });
            state.currentNoteId = data.note.id;
        }
        
        if (data) {
            state.currentNote = data.note;
            showToast('保存成功');
            loadTags(); // 更新标签计数
        }
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function deleteNote(id) {
    if (!confirm('确定要将这篇笔记移到回收站吗？')) return;
    
    try {
        await apiRequest('/notes/' + id, { method: 'DELETE' });
        showToast('已移至回收站');
        loadNotes();
        loadTags();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function restoreNote(id) {
    try {
        await apiRequest('/trash/' + id + '/restore', { method: 'POST' });
        showToast('笔记已恢复');
        loadTrash();
        loadTags();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function permanentDeleteNote(id) {
    if (!confirm('确定要永久删除这篇笔记吗？此操作不可恢复！')) return;
    
    try {
        await apiRequest('/trash/' + id, { method: 'DELETE' });
        showToast('已永久删除');
        loadTrash();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

async function togglePublic() {
    if (!state.currentNoteId) {
        showToast('请先保存笔记', 'error');
        document.getElementById('publicSwitch').checked = !document.getElementById('publicSwitch').checked;
        return;
    }
    
    try {
        const data = await apiRequest('/notes/' + state.currentNoteId + '/toggle-public', { method: 'POST' });
        if (data) {
            state.currentNote.is_public = data.is_public;
            state.currentNote.share_token = data.share_token;
            showToast(data.is_public ? '已开启公开分享' : '已取消公开分享');
            renderEditor(document.getElementById('mainContent'));
        }
    } catch (err) {
        showToast(err.message, 'error');
        document.getElementById('publicSwitch').checked = !document.getElementById('publicSwitch').checked;
    }
}

function shareNote(id) {
    openNote(id);
    setTimeout(() => {
        const switchEl = document.getElementById('publicSwitch');
        if (switchEl && !switchEl.checked) {
            togglePublic();
        }
    }, 500);
}

function copyShareLink(token) {
    const url = window.location.origin + '/share/' + token;
    navigator.clipboard.writeText(url).then(() => {
        showToast('链接已复制');
    }).catch(() => {
        showToast('复制失败，请手动复制', 'error');
    });
}

function addTagToNote(tagId) {
    tagId = parseInt(tagId);
    if (!tagId) return;
    const tag = state.tags.find(t => t.id === tagId);
    if (!tag) return;
    if (!state.currentNote.tags) state.currentNote.tags = [];
    if (!state.currentNote.tags.find(t => t.id === tagId)) {
        state.currentNote.tags.push(tag);
        renderEditor(document.getElementById('mainContent'));
    }
}

function removeTagFromNote(tagId) {
    state.currentNote.tags = state.currentNote.tags.filter(t => t.id !== tagId);
    renderEditor(document.getElementById('mainContent'));
}

// 标签管理
function showNewTagModal() {
    const name = prompt('请输入标签名称:');
    if (!name || !name.trim()) return;
    
    const color = prompt('请输入标签颜色（十六进制，如 #3b82f6）:', '#3b82f6');
    
    apiRequest('/tags', {
        method: 'POST',
        body: JSON.stringify({ name: name.trim(), color: color || '#3b82f6' }),
    }).then(data => {
        showToast('标签创建成功');
        loadTags();
    }).catch(err => {
        showToast(err.message, 'error');
    });
}

// 登出
function handleLogout() {
    if (!confirm('确定要退出登录吗？')) return;
    apiRequest('/logout', { method: 'POST' }).finally(() => {
        clearToken();
        window.location.href = '/login';
    });
}

// 启动应用
init();
