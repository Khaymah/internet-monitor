(function () {
  var layout = document.querySelector(".cl-layout");
  if (!layout) return;
  var strings = JSON.parse(layout.getAttribute("data-strings"));
  var num = new Intl.NumberFormat(strings.intl, { maximumFractionDigits: 0 });
  var releases = Array.prototype.slice.call(document.querySelectorAll(".cl-release"));
  var input = document.querySelector(".cl-search input");
  var chips = document.querySelectorAll(".cl-filter .chip");
  var count = document.querySelector(".cl-count");
  var toggleAll = document.querySelector(".cl-toggle-all");
  var empty = document.querySelector(".cl-empty");
  var filter = "all";

  function normalize(text) {
    return String(text).toLocaleLowerCase().normalize("NFKD").replace(/[ً-ٰٟ]/g, "");
  }

  function reduced() {
    try {
      return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    } catch (e) {
      return true;
    }
  }

  function parts(li) {
    return {
      button: li.querySelector(".cl-card-head button"),
      panel: li.querySelector(".cl-panel"),
      card: li.querySelector(".cl-card"),
    };
  }

  function setOpen(li, open) {
    var p = parts(li);
    p.button.setAttribute("aria-expanded", open ? "true" : "false");
    p.panel.hidden = !open;
    if (open) p.card.setAttribute("data-open", "");
    else p.card.removeAttribute("data-open");
  }

  function isOpen(li) {
    return li.querySelector(".cl-card-head button").getAttribute("aria-expanded") === "true";
  }

  var remembered = {};
  releases.forEach(function (li) {
    remembered[li.id] = isOpen(li);
  });

  function setCount(el, n) {
    var nodes = el.childNodes;
    nodes[nodes.length - 1].textContent = num.format(n);
  }

  function apply() {
    var q = normalize((input.value || "").trim());
    var searching = q !== "" || filter !== "all";
    var shownReleases = 0;
    var shownChanges = 0;
    releases.forEach(function (li) {
      var visible = 0;
      if (li.hasAttribute("data-preview")) {
        visible = filter === "all" && (q === "" || normalize(li.textContent).indexOf(q) >= 0) ? 1 : 0;
      } else {
        var versionHit = q !== "" && normalize(li.getAttribute("data-search")).indexOf(q) >= 0;
        var groups = li.querySelectorAll(".cl-group");
        for (var g = 0; g < groups.length; g++) {
          var kind = groups[g].getAttribute("data-kind");
          var items = groups[g].querySelectorAll("li");
          var n = 0;
          for (var i = 0; i < items.length; i++) {
            var hit = (filter === "all" || filter === kind) && (q === "" || versionHit || normalize(items[i].textContent).indexOf(q) >= 0);
            items[i].hidden = !hit;
            if (hit) n++;
          }
          groups[g].hidden = n === 0;
          setCount(groups[g].querySelector(".cl-group-count"), n);
          var badge = li.querySelector('.cl-badge[data-kind="' + kind + '"]');
          badge.hidden = n === 0;
          setCount(badge, n);
          visible += n;
        }
      }
      li.hidden = visible === 0;
      var link = document.querySelector('.cl-map a[href="#' + li.id + '"]');
      if (link) link.hidden = li.hidden;
      if (visible) {
        shownReleases++;
        shownChanges += visible;
      }
      parts(li).button.disabled = searching;
      setOpen(li, searching || remembered[li.id]);
    });
    document.querySelectorAll(".cl-month").forEach(function (m) {
      m.hidden = !m.querySelector(".cl-release:not([hidden])");
    });
    document.querySelectorAll(".cl-map-month").forEach(function (m) {
      m.hidden = !m.querySelector("a:not([hidden])");
    });
    count.textContent = searching
      ? strings.results.replace("{changes}", num.format(shownChanges)).replace("{releases}", num.format(shownReleases))
      : "";
    empty.hidden = shownReleases > 0;
    document.querySelector(".cl-map").hidden = shownReleases === 0;
    toggleAll.hidden = searching || shownReleases === 0;
    updateToggleAll();
  }

  function updateToggleAll() {
    var all = releases.every(function (li) {
      return remembered[li.id];
    });
    toggleAll.textContent = all ? strings.collapse : strings.expand;
    return all;
  }

  releases.forEach(function (li) {
    parts(li).button.addEventListener("click", function () {
      remembered[li.id] = !remembered[li.id];
      setOpen(li, remembered[li.id]);
      updateToggleAll();
    });
  });

  toggleAll.addEventListener("click", function () {
    var all = updateToggleAll();
    releases.forEach(function (li) {
      remembered[li.id] = !all;
      setOpen(li, !all);
    });
    updateToggleAll();
  });

  input.addEventListener("input", apply);
  for (var c = 0; c < chips.length; c++) {
    chips[c].addEventListener("click", function (ev) {
      filter = ev.currentTarget.getAttribute("data-filter");
      for (var k = 0; k < chips.length; k++) chips[k].setAttribute("aria-pressed", chips[k] === ev.currentTarget ? "true" : "false");
      apply();
    });
  }
  document.querySelector(".cl-clear").addEventListener("click", function () {
    input.value = "";
    filter = "all";
    for (var k = 0; k < chips.length; k++) chips[k].setAttribute("aria-pressed", k === 0 ? "true" : "false");
    apply();
  });

  function openTarget(id, smooth) {
    var li = document.getElementById(id);
    if (!li || !li.classList.contains("cl-release")) return;
    remembered[id] = true;
    setOpen(li, true);
    updateToggleAll();
    requestAnimationFrame(function () {
      li.scrollIntoView({ behavior: smooth && !reduced() ? "smooth" : "auto", block: "start" });
    });
  }

  document.querySelectorAll(".cl-map a").forEach(function (a) {
    a.addEventListener("click", function (ev) {
      ev.preventDefault();
      var id = a.getAttribute("href").slice(1);
      history.replaceState(null, "", "#" + id);
      openTarget(id, true);
    });
  });

  if (location.hash) openTarget(decodeURIComponent(location.hash.slice(1)), false);

  if (typeof IntersectionObserver !== "undefined") {
    var seen = {};
    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (e) {
          if (e.isIntersecting) seen[e.target.id] = e.boundingClientRect.top;
          else delete seen[e.target.id];
        });
        var first = Object.keys(seen).sort(function (a, b) {
          return seen[a] - seen[b];
        })[0];
        if (!first) return;
        document.querySelectorAll(".cl-map a").forEach(function (a) {
          if (a.getAttribute("href") === "#" + first) a.setAttribute("aria-current", "location");
          else a.removeAttribute("aria-current");
        });
      },
      { rootMargin: "-80px 0px -55% 0px" },
    );
    releases.forEach(function (li) {
      observer.observe(li);
    });
  }

  apply();
})();
