(function () {
  var doc = document.documentElement;

  function store(key, value) {
    try {
      localStorage.setItem(key, value);
    } catch (e) {}
  }

  function setThemeColor() {
    var meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.setAttribute("content", doc.getAttribute("data-theme") === "light" ? "#f3f5f9" : "#0c1015");
  }

  setThemeColor();

  function reducedMotion() {
    try {
      return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    } catch (e) {
      return false;
    }
  }

  function applyTheme(next) {
    doc.setAttribute("data-theme", next);
    store("im.site.theme", next);
    setThemeColor();
  }

  var generation = 0;

  function changeTheme(next, ev) {
    if (doc.getAttribute("data-theme") === next) return;
    var mine = ++generation;
    doc.classList.add("theme-switching");
    var done = function () {
      if (mine === generation) doc.classList.remove("theme-switching");
    };
    if (typeof document.startViewTransition !== "function" || reducedMotion()) {
      applyTheme(next);
      requestAnimationFrame(done);
      return;
    }
    var x = ev.clientX;
    var y = ev.clientY;
    if (ev.detail === 0 || (x === 0 && y === 0)) {
      var rect = ev.currentTarget.getBoundingClientRect();
      x = rect.left + rect.width / 2;
      y = rect.top + rect.height / 2;
    }
    var radius = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
    var transition;
    try {
      transition = document.startViewTransition(function () {
        applyTheme(next);
      });
    } catch (e) {
      done();
      applyTheme(next);
      return;
    }
    transition.finished.then(done, done);
    transition.ready
      .then(function () {
        doc.animate(
          { clipPath: ["circle(0px at " + x + "px " + y + "px)", "circle(" + radius + "px at " + x + "px " + y + "px)"] },
          { duration: 1000, easing: "ease-in-out", pseudoElement: "::view-transition-new(root)" },
        );
      })
      .catch(function () {});
  }

  var toggle = document.querySelector(".theme-toggle");
  if (toggle) {
    toggle.addEventListener("click", function (ev) {
      changeTheme(doc.getAttribute("data-theme") === "light" ? "dark" : "light", ev);
    });
  }

  var toasts = document.querySelector(".toasts");

  function toast(text, tone) {
    if (!toasts) return;
    var item = document.createElement("div");
    item.className = "toast " + tone;
    item.setAttribute("role", tone === "error" ? "alert" : "status");
    var label = document.createElement("span");
    label.textContent = text;
    var close = document.createElement("button");
    close.type = "button";
    close.className = "btn btn-ghost btn-sm btn-icon";
    close.setAttribute("aria-label", toasts.getAttribute("data-close"));
    close.innerHTML = toasts.querySelector('template[data-icon="close"]').innerHTML;
    item.innerHTML = toasts.querySelector('template[data-icon="' + tone + '"]').innerHTML;
    item.appendChild(label);
    item.appendChild(close);
    var remove = function () {
      if (item.parentNode) item.parentNode.removeChild(item);
    };
    close.addEventListener("click", remove);
    var shown = toasts.querySelectorAll(".toast");
    if (shown.length > 3) shown[0].parentNode.removeChild(shown[0]);
    toasts.appendChild(item);
    if (tone !== "error") setTimeout(remove, 5000);
  }

  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest && ev.target.closest(".copy-btn");
    if (!btn || !toasts) return;
    var value = btn.getAttribute("data-copy");
    var fail = function () {
      toast(toasts.getAttribute("data-error"), "error");
    };
    try {
      navigator.clipboard.writeText(value).then(function () {
        toast(toasts.getAttribute("data-ok"), "ok");
      }, fail);
    } catch (e) {
      fail();
    }
  });

  var header = document.querySelector(".site-header");
  var navToggle = document.querySelector(".nav-toggle");
  if (header && navToggle) {
    navToggle.addEventListener("click", function () {
      var open = header.classList.toggle("nav-open");
      navToggle.setAttribute("aria-expanded", open ? "true" : "false");
    });
  }

  var menus = document.querySelectorAll(".lang-menu, .more-menu");
  document.addEventListener("click", function (ev) {
    for (var m = 0; m < menus.length; m++) {
      if (menus[m].open && !menus[m].contains(ev.target)) menus[m].open = false;
    }
    var link = ev.target.closest && ev.target.closest("a[data-lang]");
    if (link) store("im.site.lang", link.getAttribute("data-lang"));
  });
  document.addEventListener("keydown", function (ev) {
    if (ev.key !== "Escape") return;
    for (var m = 0; m < menus.length; m++) {
      if (menus[m].open) {
        menus[m].open = false;
        menus[m].querySelector("summary").focus();
      }
    }
    if (header && header.classList.contains("nav-open")) {
      header.classList.remove("nav-open");
      navToggle.setAttribute("aria-expanded", "false");
    }
  });

  function detectOs() {
    var ua = navigator.userAgent || "";
    var data = navigator.userAgentData;
    var platform = (data && data.platform) || navigator.platform || "";
    if (/android|iphone|ipad|ipod|mobile/i.test(ua) || (data && data.mobile)) return "phone";
    if (/mac/i.test(platform) && navigator.maxTouchPoints > 1) return "phone";
    if (/win/i.test(platform) || /windows/i.test(ua)) return "windows";
    if (/mac/i.test(platform) || /mac os x/i.test(ua)) return "mac";
    if (/linux|x11|cros/i.test(platform + " " + ua)) return /aarch64|arm/i.test(platform + " " + ua) ? "pi" : "linux";
    return null;
  }

  var os = detectOs();

  var labels = document.querySelectorAll("[data-os-label]");
  for (var i = 0; i < labels.length; i++) {
    var label = os && labels[i].getAttribute("data-label-" + os);
    if (label) labels[i].textContent = label;
  }

  function show(selector, visible) {
    var el = document.querySelector(selector);
    if (el) el.hidden = !visible;
  }

  function openPlatform(id) {
    var card = document.getElementById("os-" + id);
    if (!card) return null;
    var details = card.querySelector("details.install");
    if (details) details.open = true;
    return card;
  }

  if (document.body.getAttribute("data-page") === "download") {
    if (os === "phone") {
      show('[data-os-msg="phone"]', true);
    } else if (os) {
      var card = openPlatform(os);
      if (card) {
        card.classList.add("is-detected");
        card.parentNode.insertBefore(card, card.parentNode.firstChild);
        show('[data-os-msg="none"]', false);
        show('[data-os-msg="' + os + '"]', true);
      }
      if (os === "mac" && navigator.userAgentData && navigator.userAgentData.getHighEntropyValues) {
        navigator.userAgentData
          .getHighEntropyValues(["architecture"])
          .then(function (v) {
            var arch = v.architecture === "arm" ? "arm" : v.architecture === "x86" ? "intel" : null;
            var group = arch && document.querySelector('[data-arch="' + arch + '"]');
            if (group) group.classList.add("is-detected");
          })
          .catch(function () {});
      }
    }
    if (/^#os-/.test(location.hash)) openPlatform(location.hash.slice(4));
  }
})();
