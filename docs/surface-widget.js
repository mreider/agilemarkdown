// Three-surface screenshot widget controller. Pages opt in by
// including this file and one or more `<figure class="surface-shot">`
// blocks. Each switcher starts on VS Code where it has a board to
// show; a moment with no board screenshot keeps the fallback surface
// build-site rendered into its data-active. Clicking a tab switches
// every switcher that offers that surface; the choice isn't remembered.

(function () {
  function applySurface(surface) {
    document.querySelectorAll('figure.surface-shot').forEach(function (fig) {
      // Skip switchers that don't offer this surface (e.g. a moment
      // with no board screenshot has no VS Code tab) so they keep the
      // surface they rendered with instead of going blank.
      if (!fig.querySelector('[role="tab"][data-surface="' + surface + '"]')) return;
      fig.dataset.active = surface;
      fig.querySelectorAll('[role="tab"]').forEach(function (t) {
        var on = t.dataset.surface === surface;
        t.setAttribute('aria-selected', on ? 'true' : 'false');
        t.setAttribute('tabindex', on ? '0' : '-1');
      });
    });
  }

  function wireFigure(fig) {
    var tabs = fig.querySelectorAll('[role="tab"]');
    tabs.forEach(function (tab, idx) {
      tab.addEventListener('click', function () { applySurface(tab.dataset.surface); });
      tab.addEventListener('keydown', function (e) {
        var dir = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0;
        if (!dir) return;
        e.preventDefault();
        var next = tabs[(idx + dir + tabs.length) % tabs.length];
        applySurface(next.dataset.surface);
        next.focus();
      });
    });
  }

  function init() {
    document.querySelectorAll('figure.surface-shot').forEach(wireFigure);
    // Default every switcher to VS Code where available; switchers
    // without a board keep the fallback rendered into their HTML.
    applySurface('extension');
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
