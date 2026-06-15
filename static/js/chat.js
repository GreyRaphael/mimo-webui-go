/* ============================================================
   MiMo WebUI – Chat Page JavaScript
   ============================================================ */

(function () {
    'use strict';

    // ---- State ----
    let currentSessionId = null;
    let sessions = [];
    let isStreaming = false;
    let abortController = null;

    // ---- DOM refs (resolved on DOMContentLoaded) ----
    let sessionListEl, chatMessagesEl, chatInputEl, sendBtnEl;
    let newSessionBtnEl, chatTitleEl, attachBtnEl, fileInputEl;
    let mediaPreviewEl, mediaPreviewImgEl, removeMediaBtnEl;

    // Pending media attachment
    let pendingMedia = null; // { file_id, media_type, url (preview) }

    document.addEventListener('DOMContentLoaded', init);

    function init() {
        sessionListEl = document.getElementById('session-list');
        chatMessagesEl = document.getElementById('chat-messages');
        chatInputEl = document.getElementById('chat-input');
        sendBtnEl = document.getElementById('send-btn');
        newSessionBtnEl = document.getElementById('new-session-btn');
        chatTitleEl = document.getElementById('chat-title');
        attachBtnEl = document.getElementById('attach-btn');
        fileInputEl = document.getElementById('file-input');
        mediaPreviewEl = document.getElementById('media-preview');
        mediaPreviewImgEl = document.getElementById('media-preview-img');
        removeMediaBtnEl = document.getElementById('remove-media-btn');

        if (sendBtnEl) sendBtnEl.addEventListener('click', sendMessage);
        if (chatInputEl) {
            chatInputEl.addEventListener('keydown', e => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    sendMessage();
                }
            });
        }
        if (newSessionBtnEl) newSessionBtnEl.addEventListener('click', createSession);
        if (attachBtnEl) attachBtnEl.addEventListener('click', () => fileInputEl?.click());
        if (fileInputEl) fileInputEl.addEventListener('change', handleFileAttach);
        if (removeMediaBtnEl) removeMediaBtnEl.addEventListener('click', removePendingMedia);

        loadSessions();

        // If URL contains a session ID, load it
        const pathParts = window.location.pathname.split('/');
        if (pathParts.length >= 3 && pathParts[1] === 'chat' && pathParts[2]) {
            switchSession(pathParts[2]);
        }
    }

    // ---- Session management ----

    async function loadSessions() {
        try {
            const resp = await fetch('/api/sessions');
            if (!resp.ok) throw new Error('Failed to load sessions');
            sessions = await resp.json();
            renderSessionList();
        } catch (err) {
            console.error('loadSessions:', err);
        }
    }

    function renderSessionList() {
        if (!sessionListEl) return;
        sessionListEl.innerHTML = '';

        if (!sessions || sessions.length === 0) {
            sessionListEl.innerHTML = '<div class="p-3 text-sm text-gray-500">暂无会话</div>';
            return;
        }

        sessions.forEach(s => {
            const div = document.createElement('div');
            div.className = `flex items-center justify-between px-3 py-2 rounded-lg cursor-pointer hover:bg-gray-700 transition ${s.id === currentSessionId ? 'bg-gray-700' : ''}`;
            div.innerHTML = `
                <span class="truncate flex-1 text-sm">${escapeHtml(s.title || '新对话')}</span>
                <button class="delete-session ml-2 text-gray-500 hover:text-red-400 text-xs" data-id="${s.id}" title="删除">✕</button>
            `;
            div.addEventListener('click', (e) => {
                if (e.target.classList.contains('delete-session')) return;
                switchSession(s.id);
            });
            div.querySelector('.delete-session').addEventListener('click', (e) => {
                e.stopPropagation();
                deleteSession(s.id);
            });
            sessionListEl.appendChild(div);
        });
    }

    async function createSession() {
        try {
            const resp = await fetch('/api/sessions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: '新对话', model: 'mimo-v2.5' }),
            });
            if (!resp.ok) throw new Error('Failed to create session');
            const session = await resp.json();
            sessions.unshift(session);
            renderSessionList();
            switchSession(session.id);
        } catch (err) {
            showToast('创建会话失败', 'error');
        }
    }

    async function deleteSession(id) {
        if (!confirm('确定删除该会话？')) return;
        try {
            const resp = await fetch(`/api/sessions/${id}`, { method: 'DELETE' });
            if (!resp.ok && resp.status !== 204) throw new Error('Failed to delete');
            sessions = sessions.filter(s => s.id !== id);
            if (currentSessionId === id) {
                currentSessionId = null;
                if (chatMessagesEl) chatMessagesEl.innerHTML = '';
                if (chatTitleEl) chatTitleEl.textContent = '选择或创建会话';
            }
            renderSessionList();
            showToast('会话已删除', 'success');
        } catch (err) {
            showToast('删除会话失败', 'error');
        }
    }

    function switchSession(id) {
        currentSessionId = id;
        const session = sessions.find(s => s.id === id);
        if (chatTitleEl) chatTitleEl.textContent = session?.title || '对话';

        // Update URL without reload
        history.replaceState(null, '', `/chat/${id}`);
        renderSessionList();
        loadMessages(id);
    }

    // ---- Messages ----

    async function loadMessages(sessionId) {
        if (!chatMessagesEl) return;
        try {
            const resp = await fetch(`/api/sessions/${sessionId}/messages`);
            if (!resp.ok) throw new Error('Failed to load messages');
            const messages = await resp.json();
            chatMessagesEl.innerHTML = '';
            (messages || []).forEach(m => appendMessage(m.role, m.content, m.media_type, m.media_url));
            scrollToBottom();
        } catch (err) {
            console.error('loadMessages:', err);
        }
    }

    function appendMessage(role, content, mediaType, mediaUrl) {
        if (!chatMessagesEl) return;
        const wrapper = document.createElement('div');
        wrapper.className = `flex flex-col ${role === 'user' ? 'items-end' : 'items-start'} mb-3`;

        let mediaHtml = '';
        if (mediaUrl && mediaType === 'image') {
            mediaHtml = `<img src="${escapeHtml(mediaUrl)}" class="media-preview mb-2" alt="attachment">`;
        } else if (mediaUrl && mediaType === 'audio') {
            mediaHtml = `<audio controls src="${escapeHtml(mediaUrl)}" class="mb-2 max-w-xs"></audio>`;
        } else if (mediaUrl && mediaType === 'video') {
            mediaHtml = `<video controls src="${escapeHtml(mediaUrl)}" class="media-preview mb-2"></video>`;
        }

        const bubbleClass = role === 'user' ? 'msg-bubble msg-user' : 'msg-bubble msg-assistant';
        wrapper.innerHTML = `
            ${mediaHtml}
            <div class="${bubbleClass}">
                <div class="msg-content" style="display:none;">${escapeHtml(content || '')}</div>
            </div>
        `;

        chatMessagesEl.appendChild(wrapper);

        // Render markdown for assistant messages
        const contentEl = wrapper.querySelector('.msg-content');
        if (contentEl && content) {
            if (role === 'assistant') {
                renderMarkdown(contentEl);
            } else {
                contentEl.style.display = '';
            }
        }

        scrollToBottom();
        return wrapper;
    }

    function createStreamingBubble() {
        if (!chatMessagesEl) return null;
        const wrapper = document.createElement('div');
        wrapper.className = 'flex flex-col items-start mb-3';
        wrapper.id = 'streaming-bubble';
        wrapper.innerHTML = `
            <div class="msg-bubble msg-assistant">
                <div class="msg-content md-content"></div>
                <div class="typing-indicator"><span></span><span></span><span></span></div>
            </div>
        `;
        chatMessagesEl.appendChild(wrapper);
        scrollToBottom();
        return wrapper.querySelector('.msg-content');
    }

    function appendStreamChunk(contentEl, chunk) {
        if (!contentEl) return;
        // Remove typing indicator on first chunk
        const indicator = contentEl.parentElement.querySelector('.typing-indicator');
        if (indicator) indicator.remove();

        contentEl.textContent += chunk;
        // Debounce markdown rendering
        clearTimeout(contentEl._renderTimer);
        contentEl._renderTimer = setTimeout(() => {
            renderMarkdown(contentEl);
        }, 100);
        scrollToBottom();
    }

    function finalizeStreamBubble(contentEl) {
        if (!contentEl) return;
        clearTimeout(contentEl._renderTimer);
        const indicator = contentEl.parentElement.querySelector('.typing-indicator');
        if (indicator) indicator.remove();
        renderMarkdown(contentEl);
        const wrapper = document.getElementById('streaming-bubble');
        if (wrapper) wrapper.removeAttribute('id');
    }

    // ---- Send message ----

    async function sendMessage() {
        if (!currentSessionId) {
            showToast('请先创建或选择会话', 'warn');
            return;
        }
        if (isStreaming) return;

        const content = chatInputEl?.value?.trim();
        if (!content && !pendingMedia) return;

        // Clear input
        if (chatInputEl) chatInputEl.value = '';

        // Build request body
        const body = { content: content || '' };
        if (pendingMedia) {
            body.media_url = pendingMedia.url || pendingMedia.file_id;
            body.media_type = pendingMedia.media_type;
        }

        // Show user message in UI
        appendMessage('user', content, pendingMedia?.media_type, pendingMedia?.url);
        removePendingMedia();

        isStreaming = true;
        updateSendBtn();

        const contentEl = createStreamingBubble();
        let fullContent = '';

        abortController = sseStream(`/api/sessions/${currentSessionId}/messages`, {
            body,
            onMessage(chunk) {
                fullContent += chunk;
                appendStreamChunk(contentEl, chunk);
            },
            onDone() {
                finalizeStreamBubble(contentEl);
                isStreaming = false;
                updateSendBtn();
                abortController = null;
            },
            onError(err) {
                finalizeStreamBubble(contentEl);
                isStreaming = false;
                updateSendBtn();
                abortController = null;
                showToast('发送失败: ' + err, 'error');
            },
        });
    }

    function updateSendBtn() {
        if (!sendBtnEl) return;
        if (isStreaming) {
            sendBtnEl.disabled = true;
            sendBtnEl.innerHTML = '<span class="spinner"></span>';
        } else {
            sendBtnEl.disabled = false;
            sendBtnEl.textContent = '发送';
        }
    }

    // ---- File attachment ----

    async function handleFileAttach(e) {
        const file = e.target.files?.[0];
        if (!file) return;

        try {
            showToast('正在上传文件...', 'info', 2000);
            const result = await uploadFile(file);

            const mediaType = file.type.startsWith('image/') ? 'image'
                : file.type.startsWith('audio/') ? 'audio'
                : file.type.startsWith('video/') ? 'video'
                : 'file';

            pendingMedia = {
                file_id: result.file_id,
                media_type: mediaType,
                url: URL.createObjectURL(file),
                mime_type: result.mime_type,
            };

            // Show preview
            if (mediaPreviewEl) {
                mediaPreviewEl.classList.remove('hidden');
                if (mediaType === 'image' && mediaPreviewImgEl) {
                    mediaPreviewImgEl.src = pendingMedia.url;
                    mediaPreviewImgEl.classList.remove('hidden');
                } else {
                    if (mediaPreviewImgEl) mediaPreviewImgEl.classList.add('hidden');
                    // Show a file name indicator
                    const label = mediaPreviewEl.querySelector('.media-label');
                    if (label) label.textContent = file.name;
                }
            }

            showToast('文件上传成功', 'success', 2000);
        } catch (err) {
            showToast('文件上传失败: ' + err.message, 'error');
        }

        // Reset file input
        if (fileInputEl) fileInputEl.value = '';
    }

    function removePendingMedia() {
        pendingMedia = null;
        if (mediaPreviewEl) mediaPreviewEl.classList.add('hidden');
        if (mediaPreviewImgEl) mediaPreviewImgEl.src = '';
    }

    // ---- Utilities ----

    function scrollToBottom() {
        if (chatMessagesEl) {
            chatMessagesEl.scrollTop = chatMessagesEl.scrollHeight;
        }
    }

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // Expose for other scripts
    window.MiMoChat = {
        sendMessage,
        createSession,
        switchSession,
        loadSessions,
    };
})();
