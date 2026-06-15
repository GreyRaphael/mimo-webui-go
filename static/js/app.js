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
