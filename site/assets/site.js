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

  var toggle = document.querySelector(".theme-toggle");
  if (toggle) {
    toggle.addEventListener("click", function () {
      var next = doc.getAttribute("data-theme") === "light" ? "dark" : "light";
      doc.setAttribute("data-theme", next);
      store("im.site.theme", next);
      setThemeColor();
    });
  }

  var header = document.querySelector(".site-header");
  var navToggle = document.querySelector(".nav-toggle");
  if (header && navToggle) {
    navToggle.addEventListener("click", function () {
      var open = header.classList.toggle("nav-open");
      navToggle.setAttribute("aria-expanded", open ? "true" : "false");
    });
  }

  var langMenu = document.querySelector(".lang-menu");
  document.addEventListener("click", function (ev) {
    if (langMenu && langMenu.open && !langMenu.contains(ev.target)) langMenu.open = false;
    var link = ev.target.closest && ev.target.closest("a[data-lang]");
    if (link) store("im.site.lang", link.getAttribute("data-lang"));
  });
  document.addEventListener("keydown", function (ev) {
    if (ev.key !== "Escape") return;
    if (langMenu && langMenu.open) {
      langMenu.open = false;
      langMenu.querySelector("summary").focus();
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
