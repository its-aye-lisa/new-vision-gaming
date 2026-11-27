(function () {
  "use strict";

  function fallbackCopy(text) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.className = "sr-only";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
    } catch (_) {
      // Ignore copy fallback failure
    }
    document.body.removeChild(ta);
  }

  document.addEventListener("click", function (ev) {
    var input = ev.target.closest("[data-select-on-click]");
    if (input && typeof input.select === "function") {
      input.select();
      return;
    }

    var btn = ev.target.closest("[data-copy-text]");
    if (!btn) return;
    var text = btn.getAttribute("data-copy-text") || "";
    if (!text) return;

    var orig = btn.getAttribute("data-copy-label") || btn.textContent;
    btn.setAttribute("data-copy-label", orig);

    var done = function () {
      btn.textContent = "Copied!";
      window.setTimeout(function () {
        btn.textContent = orig;
      }, 2000);
    };

    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      navigator.clipboard.writeText(text).then(done).catch(function () {
        fallbackCopy(text);
        done();
      });
    } else {
      fallbackCopy(text);
      done();
    }
  });
})();
