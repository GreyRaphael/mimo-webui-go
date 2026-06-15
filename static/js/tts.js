/* ============================================================
   MiMo WebUI – TTS Page JavaScript
   ============================================================ */

(function () {
    'use strict';

    // ---- State ----
    let currentMode = 'preset'; // 'preset' | 'design' | 'clone'
    let isGenerating = false;
    let abortController = null;
    let sampleFileId = null;
    let audioContext = null;
    let pcmPlaybackNode = null;

    // ---- DOM refs ----
    let modeTabsEl, presetPanelEl, designPanelEl, clonePanelEl;
    let textInputEl, generateBtnEl, resultAreaEl, audioPlayerEl;
    let statusEl, voiceSelectEl, styleInputEl, voiceDescInputEl;
    let sampleUploadEl, sampleNameEl;

    document.addEventListener('DOMContentLoaded', init);

    function init() {
        modeTabsEl = document.querySelectorAll('[data-tts-mode]');
        presetPanelEl = document.getElementById('preset-panel');
        designPanelEl = document.getElementById('design-panel');
        clonePanelEl = document.getElementById('clone-panel');
        textInputEl = document.getElementById('tts-text');
        generateBtnEl = document.getElementById('tts-generate-btn');
        resultAreaEl = document.getElementById('tts-result');
        audioPlayerEl = document.getElementById('tts-audio-player');
        statusEl = document.getElementById('tts-status');
        voiceSelectEl = document.getElementById('tts-voice');
        styleInputEl = document.getElementById('tts-style-instruction');
        voiceDescInputEl = document.getElementById('tts-voice-description');
        sampleUploadEl = document.getElementById('tts-sample-upload');
        sampleNameEl = document.getElementById('tts-sample-name');

        // Mode tab switching
        modeTabsEl.forEach(tab => {
            tab.addEventListener('click', () => switchMode(tab.dataset.ttsMode));
        });

        if (generateBtnEl) generateBtnEl.addEventListener('click', generate);
        if (sampleUploadEl) sampleUploadEl.addEventListener('change', handleSampleUpload);

        // Default mode
        switchMode('preset');
    }

    // ---- Mode switching ----

    function switchMode(mode) {
        currentMode = mode;

        // Update tab styling
        modeTabsEl.forEach(tab => {
            if (tab.dataset.ttsMode === mode) {
                tab.classList.add('bg-blue-600', 'text-white');
                tab.classList.remove('bg-gray-700', 'text-gray-300');
            } else {
                tab.classList.remove('bg-blue-600', 'text-white');
                tab.classList.add('bg-gray-700', 'text-gray-300');
            }
        });

        // Show/hide panels
        [presetPanelEl, designPanelEl, clonePanelEl].forEach(el => {
            if (el) el.classList.add('hidden');
        });

        switch (mode) {
            case 'preset':
                presetPanelEl?.classList.remove('hidden');
                break;
            case 'design':
                designPanelEl?.classList.remove('hidden');
                break;
            case 'clone':
                clonePanelEl?.classList.remove('hidden');
                break;
        }
    }

    // ---- Sample file upload for voice cloning ----

    async function handleSampleUpload(e) {
        const file = e.target.files?.[0];
        if (!file) return;

        if (!file.type.startsWith('audio/')) {
            showToast('请上传音频文件', 'warn');
            return;
        }

        try {
            setStatus('正在上传样本音频...');
            const result = await uploadFile(file);
            sampleFileId = result.file_id;
            if (sampleNameEl) sampleNameEl.textContent = file.name;
            setStatus('样本上传完成');
            showToast('样本上传成功', 'success', 2000);
        } catch (err) {
            showToast('上传失败: ' + err.message, 'error');
            setStatus('上传失败');
        }

        if (sampleUploadEl) sampleUploadEl.value = '';
    }

    // ---- Generate TTS ----

    async function generate() {
        if (isGenerating) {
            // Abort
            if (abortController) abortController.abort();
            isGenerating = false;
            updateGenerateBtn();
            setStatus('已取消');
            return;
        }

        const text = textInputEl?.value?.trim();
        if (!text) {
            showToast('请输入要合成的文本', 'warn');
            return;
        }

        // Build request body based on mode
        const body = { text, stream: true, audio_format: 'pcm', model_variant: currentMode };

        switch (currentMode) {
            case 'preset':
                body.voice = voiceSelectEl?.value || 'male-1';
                break;
            case 'design':
                body.style_instruction = styleInputEl?.value?.trim() || '';
                body.voice_description = voiceDescInputEl?.value?.trim() || '';
                break;
            case 'clone':
                if (!sampleFileId) {
                    showToast('请先上传样本音频', 'warn');
                    return;
                }
                body.sample_file_id = sampleFileId;
                break;
        }

        isGenerating = true;
        updateGenerateBtn();
        setStatus('正在生成...');
        if (resultAreaEl) resultAreaEl.innerHTML = '';

        // Collect PCM16 chunks
        const pcmChunks = [];

        abortController = sseStream('/api/tts', {
            body,
            onMessage(chunk) {
                // chunk is base64-encoded PCM16 data
                if (chunk && chunk !== '[DONE]') {
                    try {
                        const binary = atob(chunk);
                        const bytes = new Uint8Array(binary.length);
                        for (let i = 0; i < binary.length; i++) {
                            bytes[i] = binary.charCodeAt(i);
                        }
                        pcmChunks.push(bytes);
                    } catch {
                        // Not valid base64, might be status text
                    }
                }
            },
            onDone() {
                isGenerating = false;
                updateGenerateBtn();
                abortController = null;

                if (pcmChunks.length > 0) {
                    setStatus('生成完成，正在播放...');
                    playPCMChunks(pcmChunks, 24000);
                } else {
                    setStatus('未收到音频数据');
                }
            },
            onError(err) {
                isGenerating = false;
                updateGenerateBtn();
                abortController = null;
                setStatus('生成失败');
                showToast('TTS生成失败: ' + err, 'error');
            },
        });
    }

    // ---- PCM16 to WAV conversion ----

    function pcm16ToWav(pcmData, sampleRate) {
        const numChannels = 1;
        const bitsPerSample = 16;
        const byteRate = sampleRate * numChannels * (bitsPerSample / 8);
        const blockAlign = numChannels * (bitsPerSample / 8);
        const dataSize = pcmData.length;
        const buffer = new ArrayBuffer(44 + dataSize);
        const view = new DataView(buffer);

        // RIFF header
        writeString(view, 0, 'RIFF');
        view.setUint32(4, 36 + dataSize, true);
        writeString(view, 8, 'WAVE');

        // fmt chunk
        writeString(view, 12, 'fmt ');
        view.setUint32(16, 16, true);           // chunk size
        view.setUint16(20, 1, true);            // PCM format
        view.setUint16(22, numChannels, true);
        view.setUint32(24, sampleRate, true);
        view.setUint32(28, byteRate, true);
        view.setUint16(32, blockAlign, true);
        view.setUint16(34, bitsPerSample, true);

        // data chunk
        writeString(view, 36, 'data');
        view.setUint32(40, dataSize, true);

        // Copy PCM data
        new Uint8Array(buffer, 44).set(pcmData);

        return new Blob([buffer], { type: 'audio/wav' });
    }

    function writeString(view, offset, str) {
        for (let i = 0; i < str.length; i++) {
            view.setUint8(offset + i, str.charCodeAt(i));
        }
    }

    // ---- Audio playback ----

    function playPCMChunks(chunks, sampleRate) {
        // Concatenate all chunks
        const totalLength = chunks.reduce((sum, c) => sum + c.length, 0);
        const merged = new Uint8Array(totalLength);
        let offset = 0;
        for (const chunk of chunks) {
            merged.set(chunk, offset);
            offset += chunk.length;
        }

        // Convert PCM16 to WAV and play
        const wavBlob = pcm16ToWav(merged, sampleRate);
        const url = URL.createObjectURL(wavBlob);

        if (audioPlayerEl) {
            audioPlayerEl.src = url;
            audioPlayerEl.classList.remove('hidden');
            audioPlayerEl.play().catch(() => {
                setStatus('自动播放被浏览器阻止，请手动点击播放');
            });
        }

        // Also create a download link
        if (resultAreaEl) {
            const downloadLink = document.createElement('a');
            downloadLink.href = url;
            downloadLink.download = 'tts_output.wav';
            downloadLink.className = 'text-blue-400 hover:text-blue-300 text-sm mt-2 inline-block';
            downloadLink.textContent = '⬇ 下载音频';
            resultAreaEl.appendChild(downloadLink);
        }

        setStatus('播放中');
    }

    /**
 * Play real-time PCM16 via Web Audio API (for very low-latency streaming).
 * @param {Uint8Array} pcmChunk - Raw PCM16 little-endian data
 * @param {number} sampleRate
 */
    function playPCMRealtime(pcmChunk, sampleRate) {
        if (!audioContext) {
            audioContext = new (window.AudioContext || window.webkitAudioContext)({ sampleRate });
        }

        // Convert PCM16 to Float32
        const numSamples = pcmChunk.length / 2;
        const float32 = new Float32Array(numSamples);
        for (let i = 0; i < numSamples; i++) {
            const int16 = (pcmChunk[i * 2 + 1] << 8) | pcmChunk[i * 2];
            float32[i] = (int16 < 0x8000 ? int16 : int16 - 0x10000) / 32768;
        }

        const audioBuffer = audioContext.createBuffer(1, numSamples, sampleRate);
        audioBuffer.getChannelData(0).set(float32);

        const source = audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(audioContext.destination);
        source.start();
    }

    // ---- UI helpers ----

    function setStatus(text) {
        if (statusEl) statusEl.textContent = text;
    }

    function updateGenerateBtn() {
        if (!generateBtnEl) return;
        if (isGenerating) {
            generateBtnEl.textContent = '停止';
            generateBtnEl.classList.remove('bg-blue-600');
            generateBtnEl.classList.add('bg-red-600');
        } else {
            generateBtnEl.textContent = '生成语音';
            generateBtnEl.classList.remove('bg-red-600');
            generateBtnEl.classList.add('bg-blue-600');
        }
    }
})();
