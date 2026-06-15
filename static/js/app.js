// MiMo WebUI - Main Application JavaScript

// Highlight active nav link based on current path
document.addEventListener('DOMContentLoaded', function() {
    const path = window.location.pathname;
    document.querySelectorAll('.nav-link').forEach(function(link) {
        const href = link.getAttribute('href');
        if (href && (path === href || path.startsWith(href + '/'))) {
            link.classList.add('active', 'bg-gray-700', 'text-white');
        }
    });
});

// ============================================================
// IndexedDB helper for persisting file blobs across page navigation
// ============================================================
const _DB_NAME = 'mimo-webui-files';
const _DB_STORE = 'files';

function _openFileDB() {
    return new Promise((resolve, reject) => {
        const req = indexedDB.open(_DB_NAME, 1);
        req.onupgradeneeded = () => req.result.createObjectStore(_DB_STORE);
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
    });
}

async function saveFileToDB(key, file) {
    const db = await _openFileDB();
    return new Promise((resolve, reject) => {
        const tx = db.transaction(_DB_STORE, 'readwrite');
        tx.objectStore(_DB_STORE).put(file, key);
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
    });
}

async function loadFileFromDB(key) {
    const db = await _openFileDB();
    return new Promise((resolve, reject) => {
        const tx = db.transaction(_DB_STORE, 'readonly');
        const req = tx.objectStore(_DB_STORE).get(key);
        req.onsuccess = () => resolve(req.result || null);
        req.onerror = () => reject(req.error);
    });
}

async function removeFileFromDB(key) {
    const db = await _openFileDB();
    return new Promise((resolve, reject) => {
        const tx = db.transaction(_DB_STORE, 'readwrite');
        tx.objectStore(_DB_STORE).delete(key);
        tx.oncomplete = () => resolve();
        tx.onerror = () => reject(tx.error);
    });
}
