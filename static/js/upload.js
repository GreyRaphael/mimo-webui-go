/* ============================================================
   MiMo WebUI – File Upload JavaScript
   ============================================================ */

/**
 * Upload a file to the server with optional progress tracking.
 * @param {File} file - The file to upload
 * @param {object} [options]
 * @param {function} [options.onProgress] - Called with (percent: number) during upload
 * @returns {Promise<{file_id: string, mime_type: string, size_bytes: number, temp_path: string}>}
 */
async function uploadFile(file, options) {
    options = options || {};

    const formData = new FormData();
    formData.append('file', file);

    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/upload');

        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable && options.onProgress) {
                const percent = Math.round((e.loaded / e.total) * 100);
                options.onProgress(percent);
            }
        });

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    resolve(JSON.parse(xhr.responseText));
                } catch {
                    reject(new Error('Invalid JSON response'));
                }
            } else {
                try {
                    const err = JSON.parse(xhr.responseText);
                    reject(new Error(err.error || `Upload failed: HTTP ${xhr.status}`));
                } catch {
                    reject(new Error(`Upload failed: HTTP ${xhr.status}`));
                }
            }
        });

        xhr.addEventListener('error', () => reject(new Error('Network error')));
        xhr.addEventListener('abort', () => reject(new Error('Upload aborted')));

        xhr.send(formData);
    });
}

/**
 * Set up a drag-and-drop file upload zone.
 * @param {string|Element} dropzone - CSS selector or DOM element for the dropzone
 * @param {object} options
 * @param {string[]} [options.accept] - Accepted MIME types (e.g. ['image/*', 'audio/*'])
 * @param {number} [options.maxSizeMB] - Max file size in MB
 * @param {function} options.onUpload - Called with (result, file) after successful upload
 * @param {function} [options.onError] - Called with (error: Error) on failure
 * @param {function} [options.onProgress] - Called with (percent: number) during upload
 * @param {boolean} [options.multiple=false] - Allow multiple files
 * @returns {{ destroy: function, getInput: function }}
 */
function setupDropzone(dropzone, options) {
    const el = typeof dropzone === 'string' ? document.querySelector(dropzone) : dropzone;
    if (!el) throw new Error('Dropzone element not found');

    const {
        accept,
        maxSizeMB,
        onUpload,
        onError,
        onProgress,
        multiple = false,
    } = options;

    // Create hidden file input
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.multiple = multiple;
    if (accept && accept.length) {
        fileInput.accept = accept.join(',');
    }
    fileInput.style.display = 'none';
    el.appendChild(fileInput);

    // Click to open file picker
    function handleClick(e) {
        if (e.target === fileInput) return;
        fileInput.click();
    }

    // Drag events
    function handleDragOver(e) {
        e.preventDefault();
        e.stopPropagation();
        el.classList.add('dragover');
    }

    function handleDragLeave(e) {
        e.preventDefault();
        e.stopPropagation();
        el.classList.remove('dragover');
    }

    function handleDrop(e) {
        e.preventDefault();
        e.stopPropagation();
        el.classList.remove('dragover');

        const files = e.dataTransfer?.files;
        if (files && files.length) {
            processFiles(multiple ? Array.from(files) : [files[0]]);
        }
    }

    function handleFileChange() {
        if (fileInput.files && fileInput.files.length) {
            processFiles(multiple ? Array.from(fileInput.files) : [fileInput.files[0]]);
            fileInput.value = '';
        }
    }

    function processFiles(files) {
        files.forEach(file => {
            // Validate file type
            if (accept && accept.length) {
                const isAccepted = accept.some(pattern => {
                    if (pattern.endsWith('/*')) {
                        return file.type.startsWith(pattern.replace('/*', '/'));
                    }
                    return file.type === pattern || file.name.toLowerCase().endsWith(pattern);
                });
                if (!isAccepted) {
                    const err = new Error(`不支持的文件类型: ${file.type || '未知'}`);
                    if (onError) onError(err);
                    else showToast(err.message, 'warn');
                    return;
                }
            }

            // Validate file size
            if (maxSizeMB && file.size > maxSizeMB * 1024 * 1024) {
                const err = new Error(`文件大小超过 ${maxSizeMB} MB 限制`);
                if (onError) onError(err);
                else showToast(err.message, 'warn');
                return;
            }

            // Upload
            uploadFile(file, { onProgress })
                .then(result => {
                    if (onUpload) onUpload(result, file);
                })
                .catch(err => {
                    if (onError) onError(err);
                    else showToast('上传失败: ' + err.message, 'error');
                });
        });
    }

    // Bind events
    el.addEventListener('click', handleClick);
    el.addEventListener('dragover', handleDragOver);
    el.addEventListener('dragleave', handleDragLeave);
    el.addEventListener('drop', handleDrop);
    fileInput.addEventListener('change', handleFileChange);

    return {
        destroy() {
            el.removeEventListener('click', handleClick);
            el.removeEventListener('dragover', handleDragOver);
            el.removeEventListener('dragleave', handleDragLeave);
            el.removeEventListener('drop', handleDrop);
            fileInput.removeEventListener('change', handleFileChange);
            fileInput.remove();
        },
        getInput() {
            return fileInput;
        },
    };
}

/**
 * Generate a preview for an uploaded file.
 * @param {File} file
 * @returns {Promise<{type: string, url: string, element: Element}>}
 */
function generatePreview(file) {
    return new Promise((resolve, reject) => {
        const type = file.type.startsWith('image/') ? 'image'
            : file.type.startsWith('audio/') ? 'audio'
            : file.type.startsWith('video/') ? 'video'
            : 'unknown';

        if (type === 'unknown') {
            reject(new Error('无法预览此文件类型'));
            return;
        }

        const url = URL.createObjectURL(file);
        let element;

        switch (type) {
            case 'image':
                element = document.createElement('img');
                element.src = url;
                element.className = 'media-preview';
                element.alt = file.name;
                break;
            case 'audio':
                element = document.createElement('audio');
                element.src = url;
                element.controls = true;
                element.className = 'max-w-xs';
                break;
            case 'video':
                element = document.createElement('video');
                element.src = url;
                element.controls = true;
                element.className = 'media-preview';
                break;
        }

        element.addEventListener('load', () => resolve({ type, url, element }), { once: true });
        element.addEventListener('loadeddata', () => resolve({ type, url, element }), { once: true });
        element.addEventListener('error', () => {
            URL.revokeObjectURL(url);
            reject(new Error('预览加载失败'));
        }, { once: true });

        // For images that might already be cached
        if (type === 'image' && element.complete) {
            resolve({ type, url, element });
        }
    });
}

// ---- Multimodal page helpers ----

/**
 * Set up a multimodal understanding page (image/audio/video).
 * @param {object} config
 * @param {string} config.formId - ID of the form element
 * @param {string} config.dropzoneId - ID of the dropzone element
 * @param {string} config.promptId - ID of the prompt textarea
 * @param {string} config.resultId - ID of the result container
 * @param {string} config.submitId - ID of the submit button
 * @param {string} config.apiEndpoint - API endpoint (e.g. '/api/image')
 * @param {string[]} config.acceptTypes - Accepted MIME types
 * @param {string} config.mediaType - 'image' | 'audio' | 'video'
 */
function setupMultimodalPage(config) {
    const {
        dropzoneId,
        promptId,
        resultId,
        submitId,
        apiEndpoint,
        acceptTypes,
        mediaType,
    } = config;

    let uploadedFileId = null;
    let isProcessing = false;
    let abortController = null;

    const resultEl = document.getElementById(resultId);
    const submitBtn = document.getElementById(submitId);
    const promptEl = document.getElementById(promptId);

    // Setup dropzone
    setupDropzone(`#${dropzoneId}`, {
        accept: acceptTypes,
        maxSizeMB: mediaType === 'video' ? 200 : (mediaType === 'audio' ? 50 : 20),
        onUpload(result, file) {
            uploadedFileId = result.file_id;
            showToast('文件上传成功', 'success', 2000);

            // Show preview
            const previewContainer = document.getElementById(`${dropzoneId}-preview`);
            if (previewContainer) {
                previewContainer.innerHTML = '';
                generatePreview(file).then(({ element }) => {
                    previewContainer.appendChild(element);
                }).catch(() => {});
            }
        },
        onError(err) {
            showToast(err.message, 'error');
        },
    });

    // Submit
    if (submitBtn) {
        submitBtn.addEventListener('click', async () => {
            if (isProcessing) {
                if (abortController) abortController.abort();
                isProcessing = false;
                updateBtn();
                return;
            }

            if (!uploadedFileId) {
                showToast('请先上传文件', 'warn');
                return;
            }

            const prompt = promptEl?.value?.trim() || '';
            isProcessing = true;
            updateBtn();
            if (resultEl) resultEl.innerHTML = '<div class="spinner spinner-lg mx-auto mt-8"></div>';

            let fullContent = '';

            abortController = sseStream(apiEndpoint, {
                body: { file_id: uploadedFileId, prompt },
                onMessage(chunk) {
                    if (!resultEl) return;
                    if (!fullContent) resultEl.innerHTML = '';
                    fullContent += chunk;
                    resultEl.textContent = fullContent;
                },
                onDone() {
                    isProcessing = false;
                    updateBtn();
                    abortController = null;
                    if (resultEl && fullContent) renderMarkdown(resultEl);
                },
                onError(err) {
                    isProcessing = false;
                    updateBtn();
                    abortController = null;
                    if (resultEl) resultEl.innerHTML = `<p class="text-red-400">错误: ${escapeHtml(err)}</p>`;
                    showToast('处理失败: ' + err, 'error');
                },
            });
        });
    }

    function updateBtn() {
        if (!submitBtn) return;
        if (isProcessing) {
            submitBtn.textContent = '停止';
            submitBtn.classList.remove('bg-blue-600');
            submitBtn.classList.add('bg-red-600');
        } else {
            submitBtn.textContent = '提交';
            submitBtn.classList.remove('bg-red-600');
            submitBtn.classList.add('bg-blue-600');
        }
    }

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
}
