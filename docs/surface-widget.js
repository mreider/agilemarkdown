// Three-surface screenshot widget controller. Pages opt in by
// including this file and one or more `<figure class="surface-shot">`
// blocks. Choice persists across pages via localStorage so a reader
// switching to "cli" on the landing keeps that view through the
// tutorials.

(function () {
  var KEY = 'am.surface';
  var DEFAULT = 'extension';
  var current = DEFAULT;
  try {
    var saved = localStorage.getItem(KEY);
    if (saved === 'cli' || saved === 'extension' || saved === 'claude') {
      current = saved;
    }
  } catch (_) {}

  function applySurface(surface) {
    current = surface;
    try { localStorage.setItem(KEY, surface); } catch (_) {}
    document.querySelectorAll('figure.surface-shot').forEach(function (fig) {
      fig.dataset.active = surface;
      fig.querySelectorAll('[role="tab"]').forEach(function (t) {
        var on = t.dataset.surface === surface;
        t.setAttribute('aria-selected', on ? 'true' : 'false');
        t.setAttribute('tabindex', on ? '0' : '-1');
      });
    });
    document.querySelectorAll('.surface-switcher [data-surface]').forEach(function (b) {
      b.setAttribute('aria-checked', b.dataset.surface === surface ? 'true' : 'false');
    });
  }

  function wireFigure(fig) {
    var tabs = fig.querySelectorAll('[role="tab"]');
    tabs.forEach(function (tab, idx) {
      tab.addEventListener('click', function () { applySurface(tab.dataset.surface); });
      tab.addEventListener('keydown', function (e) {
        var key = e.key;
        var dir = key === 'ArrowRight' ? 1 : key === 'ArrowLeft' ? -1 : 0;
        if (!dir) return;
        e.preventDefault();
        var next = tabs[(idx + dir + tabs.length) % tabs.length];
        applySurface(next.dataset.surface);
        next.focus();
      });
    });
  }

  function wireSwitcher(btn) {
    btn.addEventListener('click', function () { applySurface(btn.dataset.surface); });
  }

  function init() {
    document.querySelectorAll('figure.surface-shot').forEach(wireFigure);
    document.querySelectorAll('.surface-switcher [data-surface]').forEach(wireSwitcher);
    applySurface(current);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
