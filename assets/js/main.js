/**
 * Theming & Palette System.
 * Supports OS preference, manual theme (light/dark), palette selection,
 * and an expandable capsule control (click-only, no hover preview).
 */
let currentTheme = "dark";

// Detect the color scheme the operating system prefers.
function detectOSColorTheme() {
  const chosenTheme = window.localStorage && window.localStorage.getItem("theme");
  if (chosenTheme === "dark" || chosenTheme === "light") {
    currentTheme = chosenTheme;
  } else if (window.matchMedia("(prefers-color-scheme: light)").matches) {
    currentTheme = "light";
  } else {
    currentTheme = "dark";
  }
  document.documentElement.setAttribute("data-theme", currentTheme);
}

function setTheme(theme) {
  if (!theme) return;
  currentTheme = theme;
  try {
    if (window.localStorage) {
      window.localStorage.setItem("theme", theme);
    }
  } catch (err) {
    console.warn("Could not write theme to localStorage", err);
  }
  document.documentElement.setAttribute("data-theme", theme);
}

function switchTheme() {
  const nextTheme = currentTheme === "dark" ? "light" : "dark";
  setTheme(nextTheme);
}

/**
 * Palette Switcher.
 */
let currentPalette = "orange";

function updatePaletteActiveUI(activePalette) {
  document.querySelectorAll(".palette-dot-btn").forEach(btn => {
    const target = btn.getAttribute("data-palette-target");
    if (target === activePalette) {
      btn.classList.add("active");
      btn.setAttribute("aria-checked", "true");
    } else {
      btn.classList.remove("active");
      btn.setAttribute("aria-checked", "false");
    }
  });
}

function setPalette(paletteName) {
  if (!paletteName) return;
  currentPalette = paletteName;
  if (window.localStorage) {
    try {
      window.localStorage.setItem("palette", paletteName);
    } catch (e) {
      console.warn("Could not write palette to localStorage", e);
    }
  }
  document.documentElement.setAttribute("data-palette", paletteName);
  updatePaletteActiveUI(paletteName);
}

/**
 * Initialize theme, palette, and capsule trigger/auto-collapse behaviors.
 */
function initThemeAndPaletteControls() {
  let initialPalette = (window.localStorage && window.localStorage.getItem("palette")) || "orange";
  if (initialPalette === "planetscale-orange") {
    initialPalette = "orange";
    try { window.localStorage.setItem("palette", "orange"); } catch (e) {}
  }
  currentPalette = initialPalette;
  document.documentElement.setAttribute("data-palette", currentPalette);
  updatePaletteActiveUI(currentPalette);

  detectOSColorTheme();

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", (e) => e.matches && detectOSColorTheme());
  window
    .matchMedia("(prefers-color-scheme: light)")
    .addEventListener("change", (e) => e.matches && detectOSColorTheme());

  // Standalone theme toggles outside the capsule (if any)
  document.querySelectorAll(".theme-toggle").forEach(toggle => {
    if (!toggle.closest(".theme-control-capsule")) {
      toggle.addEventListener("click", switchTheme);
    }
  });

  const capsule = document.querySelector(".theme-control-capsule");
  if (!capsule) return;

  const triggerBtn = capsule.querySelector(".capsule-trigger-btn");
  const optionsContainer = capsule.querySelector(".capsule-options");
  const themeToggleBtn = capsule.querySelector(".theme-toggle-btn");
  const paletteBtns = capsule.querySelectorAll(".palette-dot-btn");

  let collapseTimer = null;
  let idleTimer = null;

  const mobileQuery = window.matchMedia("(max-width: 684px)");
  const isMobileCapsule = () => mobileQuery.matches;

  if (optionsContainer) {
    optionsContainer.querySelectorAll("button").forEach(btn => btn.setAttribute("tabindex", "-1"));
  }

  function resetIdleTimer() {
    clearTimeout(idleTimer);
    if (isMobileCapsule()) return;
    if (capsule.classList.contains("expanded")) {
      idleTimer = setTimeout(() => {
        collapseCapsule();
      }, 5000);
    }
  }

  function expandCapsule() {
    capsule.classList.add("expanded");
    if (triggerBtn) {
      triggerBtn.setAttribute("aria-expanded", "true");
      triggerBtn.setAttribute("title", "Close appearance settings");
    }
    if (optionsContainer) {
      optionsContainer.setAttribute("aria-hidden", "false");
      optionsContainer.querySelectorAll("button").forEach(btn => btn.setAttribute("tabindex", "0"));
    }
    resetIdleTimer();
  }

  function collapseCapsule() {
    if (isMobileCapsule()) return;
    clearTimeout(collapseTimer);
    clearTimeout(idleTimer);
    capsule.classList.remove("expanded");
    if (triggerBtn) {
      triggerBtn.setAttribute("aria-expanded", "false");
      triggerBtn.setAttribute("title", "Theme and appearance settings");
    }
    if (optionsContainer) {
      optionsContainer.setAttribute("aria-hidden", "true");
      optionsContainer.querySelectorAll("button").forEach(btn => btn.setAttribute("tabindex", "-1"));
    }
    if (capsule.contains(document.activeElement) && document.activeElement !== triggerBtn) {
      triggerBtn.focus();
    }
  }

  function scheduleCollapse(delay = 2000) {
    if (isMobileCapsule()) return;
    clearTimeout(collapseTimer);
    collapseTimer = setTimeout(() => {
      collapseCapsule();
    }, delay);
  }

  function cancelCollapse() {
    clearTimeout(collapseTimer);
    resetIdleTimer();
  }

  function updateMobileState() {
    if (isMobileCapsule()) {
      clearTimeout(collapseTimer);
      clearTimeout(idleTimer);
      capsule.classList.add("expanded");
      if (triggerBtn) {
        triggerBtn.setAttribute("aria-expanded", "true");
      }
      if (optionsContainer) {
        optionsContainer.setAttribute("aria-hidden", "false");
        optionsContainer.querySelectorAll("button").forEach(btn => btn.setAttribute("tabindex", "0"));
      }
    } else {
      collapseCapsule();
    }
  }

  mobileQuery.addEventListener("change", updateMobileState);
  updateMobileState();

  // 1. Trigger button click expands/collapses the options
  if (triggerBtn) {
    triggerBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      if (capsule.classList.contains("expanded")) {
        collapseCapsule();
      } else {
        expandCapsule();
      }
    });
  }

  // If clicked anywhere on capsule while collapsed, expand it
  capsule.addEventListener("click", (e) => {
    if (!capsule.classList.contains("expanded")) {
      expandCapsule();
    }
  });

  // 2. Dark/Light toggle button inside options: click only (no hover preview)
  if (themeToggleBtn) {
    themeToggleBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      cancelCollapse();
      const nextTheme = currentTheme === "dark" ? "light" : "dark";
      setTheme(nextTheme);
      scheduleCollapse(1200);
    });
  }

  // 3. Palette dots inside options: click anywhere in the square section
  paletteBtns.forEach(btn => {
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      cancelCollapse();
      const target = btn.getAttribute("data-palette-target");
      if (target) {
        setPalette(target);
        scheduleCollapse(1200);
      }
    });
  });

  const paletteSwitcher = capsule.querySelector(".palette-switcher");
  if (paletteSwitcher) {
    paletteSwitcher.addEventListener("click", (e) => {
      if (e.target.closest(".palette-dot-btn")) return;
      const clickX = e.clientX;
      let closestBtn = null;
      let minDistance = Infinity;
      paletteBtns.forEach(pBtn => {
        const rect = pBtn.getBoundingClientRect();
        const center = rect.left + rect.width / 2;
        const dist = Math.abs(clickX - center);
        if (dist < minDistance) {
          minDistance = dist;
          closestBtn = pBtn;
        }
      });
      if (closestBtn) {
        cancelCollapse();
        const target = closestBtn.getAttribute("data-palette-target");
        if (target) {
          setPalette(target);
          scheduleCollapse(1200);
        }
      }
    });
  }

  // 4. Mouse tracking across capsule for keepalive and auto-collapse
  capsule.addEventListener("mouseenter", () => {
    cancelCollapse();
  });

  capsule.addEventListener("mousemove", () => {
    cancelCollapse();
  });

  capsule.addEventListener("mouseleave", () => {
    if (capsule.classList.contains("expanded")) {
      scheduleCollapse(2000);
    }
  });

  // Keep alive when focused, collapse when focus leaves
  capsule.addEventListener("focusin", () => {
    cancelCollapse();
    resetIdleTimer();
  });

  capsule.addEventListener("focusout", (e) => {
    if (!capsule.contains(e.relatedTarget)) {
      scheduleCollapse(1200);
    }
  });

  // 5. Outside clicks & escape key
  document.addEventListener("click", (e) => {
    if (!capsule.contains(e.target) && capsule.classList.contains("expanded")) {
      collapseCapsule();
    }
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && capsule.classList.contains("expanded")) {
      collapseCapsule();
    }
  });
}

/**
 * Table of Contents Scroll-Spy
 * Highlights the TOC item corresponding to the section currently being read by the user.
 */
function initTocScrollSpy() {
  const treeNav = document.querySelector("aside.tree-nav");
  if (!treeNav) return;

  const tocLinks = Array.from(treeNav.querySelectorAll("a[href^='#']"));
  if (tocLinks.length === 0) return;

  // Map each link to its target element in the document
  const targets = tocLinks
    .map(link => {
      const href = link.getAttribute("href");
      if (!href || href === "#") return null;
      try {
        const id = decodeURIComponent(href.slice(1));
        const el = document.getElementById(id);
        if (el) {
          return { link, el, id };
        }
      } catch (e) {
        // Ignore malformed href
      }
      return null;
    })
    .filter(Boolean);

  if (targets.length === 0) return;

  let currentActive = null;

  function setActive(target) {
    if (!target || target === currentActive) return;
    currentActive = target;

    tocLinks.forEach(link => {
      link.classList.remove("active");
      const li = link.closest("li");
      if (li) {
        li.classList.remove("active-item");
        li.classList.remove("active-parent");
      }
    });

    target.link.classList.add("active");
    const li = target.link.closest("li");
    if (li) {
      li.classList.add("active-item");
      let parentLi = li.parentElement ? li.parentElement.closest("li") : null;
      while (parentLi) {
        parentLi.classList.add("active-parent");
        parentLi = parentLi.parentElement ? parentLi.parentElement.closest("li") : null;
      }
    }

    // Auto-scroll the tree-nav container if needed so active item is in view
    const scrollBox = treeNav.querySelector(".tree-nav-card") || treeNav;
    if (scrollBox.scrollHeight > scrollBox.clientHeight) {
      const linkRect = target.link.getBoundingClientRect();
      const navRect = scrollBox.getBoundingClientRect();
      if (linkRect.top < navRect.top + 30 || linkRect.bottom > navRect.bottom - 30) {
        target.link.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    }
  }

  function onScroll() {
    const scrollBottom = window.innerHeight + window.scrollY;
    const docHeight = document.documentElement.scrollHeight;

    // 1. If at or near the very bottom of the page, activate the last target
    if (scrollBottom >= docHeight - 50) {
      setActive(targets[targets.length - 1]);
      return;
    }

    // 2. Otherwise find the heading that has passed the reading threshold (140px from top)
    const threshold = 140;
    let active = null;

    for (let i = 0; i < targets.length; i++) {
      const rect = targets[i].el.getBoundingClientRect();
      if (rect.top <= threshold) {
        active = targets[i];
      } else {
        break;
      }
    }

    // 3. If above all headings, default to the first heading
    if (!active && targets.length > 0) {
      active = targets[0];
    }

    if (active) {
      setActive(active);
    }
  }

  let ticking = false;
  window.addEventListener("scroll", () => {
    if (!ticking) {
      window.requestAnimationFrame(() => {
        onScroll();
        ticking = false;
      });
      ticking = true;
    }
  }, { passive: true });

  window.addEventListener("resize", onScroll, { passive: true });

  // Instant feedback on click
  tocLinks.forEach(link => {
    link.addEventListener("click", () => {
      const match = targets.find(t => t.link === link);
      if (match) {
        setActive(match);
      }
    });
  });

  // Initial check on load
  onScroll();
}

function initArchiveTags() {
  const cloud = document.getElementById("archive-tag-cloud");
  const button = cloud?.querySelector(".archive-tags-toggle");
  if (!button) return;

  const extraTags = Array.from(cloud.children).filter(item => !item.classList.contains("archive-tags-control")).slice(12);
  const setExpanded = expanded => {
    extraTags.forEach(tag => { tag.hidden = !expanded; });
    button.setAttribute("aria-expanded", String(expanded));
    button.textContent = expanded ? "Show fewer tags" : button.dataset.expandLabel;
  };

  setExpanded(false);
  button.parentElement.hidden = false;
  const browse = cloud.closest(".archive-browse");
  let animation;
  button.addEventListener("click", () => {
    const startHeight = browse?.getBoundingClientRect().height;
    animation?.cancel();
    setExpanded(button.getAttribute("aria-expanded") !== "true");
    if (browse && !window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      const endHeight = browse.getBoundingClientRect().height;
      animation = browse.animate([
        { height: `${startHeight}px`, overflow: "clip" },
        { height: `${endHeight}px`, overflow: "clip" },
      ], { duration: 280, easing: "ease-in-out" });
    }
  });
}

function initImageFadeIn() {
  document.querySelectorAll("img.img-fade-in").forEach(img => {
    if (img.complete && img.naturalWidth > 0) {
      img.classList.add("is-loaded");
    } else {
      img.addEventListener("load", () => img.classList.add("is-loaded"), { once: true });
      img.addEventListener("error", () => img.classList.add("is-loaded"), { once: true });
    }
  });
}

function initKeyboardShortcuts() {
  let gPending = false;
  let gTimer = null;
  let modalEl = null;

  function ensureModal() {
    if (modalEl) return modalEl;
    modalEl = document.createElement("div");
    modalEl.className = "shortcuts-modal-backdrop";
    modalEl.setAttribute("role", "dialog");
    modalEl.setAttribute("aria-modal", "true");
    modalEl.setAttribute("aria-labelledby", "shortcuts-modal-title");
    modalEl.hidden = true;
    modalEl.innerHTML = `
      <div class="shortcuts-modal">
        <div class="shortcuts-modal__header">
          <h2 id="shortcuts-modal-title">Keyboard Shortcuts</h2>
          <button type="button" class="shortcuts-modal__close" aria-label="Close shortcuts menu">&times;</button>
        </div>
        <div class="shortcuts-modal__grid">
          <div class="shortcuts-modal__section">
            <h3>Navigation</h3>
            <dl class="shortcuts-list">
              <div><dt><kbd>[</kbd> <span class="kbd-then">/</span> <kbd>]</kbd></dt><dd>Prev / next post</dd></div>
              <div><dt><kbd>Shift</kbd> <span class="kbd-then">+</span> <kbd>[</kbd> <span class="kbd-then">/</span> <kbd>]</kbd></dt><dd>Prev / next in series</dd></div>
              <div><dt><kbd>/</kbd></dt><dd>Search site</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>h</kbd></dt><dd>Go to Home</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>p</kbd></dt><dd>Go to Posts</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>d</kbd></dt><dd>Go to Daily Prompts</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>s</kbd></dt><dd>Go to Projects</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>m</kbd></dt><dd>Go to About Me</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>r</kbd></dt><dd>Go to Random Page</dd></div>
            </dl>
          </div>
          <div class="shortcuts-modal__section">
            <h3>Reading &amp; Actions</h3>
            <dl class="shortcuts-list">
              <div><dt><kbd>?</kbd></dt><dd>Toggle this help menu</dd></div>
              <div><dt><kbd>j</kbd> <span class="kbd-then">/</span> <kbd>k</kbd></dt><dd>Scroll down / up</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>g</kbd></dt><dd>Scroll to top</dd></div>
              <div><dt><kbd>G</kbd></dt><dd>Scroll to bottom</dd></div>
              <div><dt><kbd>t</kbd></dt><dd>Toggle dark / light mode</dd></div>
              <div><dt><kbd>Esc</kbd></dt><dd>Close menu / unfocus input</dd></div>
            </dl>
          </div>
        </div>
      </div>
    `;
    document.body.appendChild(modalEl);

    modalEl.querySelector(".shortcuts-modal__close")?.addEventListener("click", () => {
      toggleModal(false);
    });
    modalEl.addEventListener("click", (ev) => {
      if (ev.target === modalEl) {
        toggleModal(false);
      }
    });
    return modalEl;
  }

  function toggleModal(forceState) {
    const el = ensureModal();
    const nextState = typeof forceState === "boolean" ? forceState : el.hidden;
    el.hidden = !nextState;
  }

  window.toggleShortcutsModal = toggleModal;

  function clearGPending() {
    gPending = false;
    if (gTimer) {
      clearTimeout(gTimer);
      gTimer = null;
    }
  }

  document.addEventListener("keydown", (e) => {
    if (e.defaultPrevented || e.metaKey || e.ctrlKey) return;

    const target = e.target;
    const isInput =
      target &&
      (target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable);

    if (e.key === "Escape") {
      if (modalEl && !modalEl.hidden) {
        e.preventDefault();
        toggleModal(false);
        return;
      }
      if (isInput && typeof target.blur === "function") {
        target.blur();
      }
      clearGPending();
      return;
    }

    if (isInput) return;

    // '?' toggles the shortcuts modal
    if (e.key === "?") {
      e.preventDefault();
      clearGPending();
      toggleModal();
      return;
    }

    // '/' navigates to /search/ (or focuses search box if already on /search/)
    if (e.key === "/") {
      e.preventDefault();
      clearGPending();
      const existingSearchInput = document.querySelector(
        "#search .pagefind-ui__search-input, #search input[type='text'], #search input"
      );
      if (existingSearchInput) {
        existingSearchInput.focus();
        if (typeof existingSearchInput.select === "function") {
          existingSearchInput.select();
        }
      } else {
        window.location.href = "/search/";
      }
      return;
    }

    // Shift + '[' ('{') -> Previous post in current series
    if (e.key === "{" || (e.code === "BracketLeft" && e.shiftKey && !e.altKey)) {
      clearGPending();
      const seriesPrev = document.querySelector(".post-series-box .series-step.is-prev a");
      if (seriesPrev && seriesPrev.href) {
        e.preventDefault();
        window.location.href = seriesPrev.href;
      }
      return;
    }

    // Shift + ']' ('}') -> Next post in current series
    if (e.key === "}" || (e.code === "BracketRight" && e.shiftKey && !e.altKey)) {
      clearGPending();
      const seriesNext = document.querySelector(".post-series-box .series-step.is-next a");
      if (seriesNext && seriesNext.href) {
        e.preventDefault();
        window.location.href = seriesNext.href;
      }
      return;
    }

    // '[' or physical BracketLeft -> Previous post / page
    if (e.key === "[" || (e.code === "BracketLeft" && !e.shiftKey && !e.altKey)) {
      clearGPending();
      const prevLink = document.querySelector(".pagination .button.previous a");
      if (prevLink && prevLink.href) {
        e.preventDefault();
        window.location.href = prevLink.href;
      }
      return;
    }

    // ']' or physical BracketRight -> Next post / page
    if (e.key === "]" || (e.code === "BracketRight" && !e.shiftKey && !e.altKey)) {
      clearGPending();
      const nextLink = document.querySelector(".pagination .button.next a");
      if (nextLink && nextLink.href) {
        e.preventDefault();
        window.location.href = nextLink.href;
      }
      return;
    }

    if (e.altKey) return;

    // Ignore key auto-repeat on 'g' while holding 'g' for a chord like g+r
    if (gPending && e.repeat && e.key.toLowerCase() === "g") {
      return;
    }

    // Handle 'g' chord sequences (g h, g p, g d, g s, g m, g r, g g)
    if (gPending) {
      const k = e.key.toLowerCase();
      clearGPending();
      if (k === "h") {
        e.preventDefault();
        window.location.href = "/";
      } else if (k === "p") {
        e.preventDefault();
        window.location.href = "/posts/";
      } else if (k === "d") {
        e.preventDefault();
        window.location.href = "/prompts/";
      } else if (k === "s") {
        e.preventDefault();
        window.location.href = "/projects/";
      } else if (k === "m") {
        e.preventDefault();
        window.location.href = "/me/";
      } else if (k === "r") {
        e.preventDefault();
        window.location.href = "/random/";
      } else if (k === "g") {
        e.preventDefault();
        window.scrollTo({ top: 0, behavior: "smooth" });
      }
      return;
    }

    if (e.key === "g" && !e.shiftKey) {
      gPending = true;
      gTimer = setTimeout(clearGPending, 1200);
      return;
    }

    if (e.key === "G") {
      e.preventDefault();
      window.scrollTo({ top: document.documentElement.scrollHeight, behavior: "smooth" });
      return;
    }

    if (e.key === "j") {
      window.scrollBy({ top: 240, behavior: "smooth" });
      return;
    }

    if (e.key === "k") {
      window.scrollBy({ top: -240, behavior: "smooth" });
      return;
    }

    if (e.key === "t") {
      e.preventDefault();
      switchTheme();
      return;
    }
  });
}

function initMascotScrollMood() {
  const mascots = Array.from(document.querySelectorAll(".toc-mascot"));
  const panels = Array.from(document.querySelectorAll("[data-nibble-panel]"));
  const restoreBars = Array.from(document.querySelectorAll("[data-nibble-restore-bar]"));
  const restoreBtns = Array.from(document.querySelectorAll(".nibble-restore-btn, [data-nibble-restore]"));

  function syncRestoreButtons() {
    let isHidden = false;
    try {
      isHidden = localStorage.getItem("kmcd-hide-nibble") === "1";
    } catch (e) {}
    restoreBars.forEach(bar => {
      bar.hidden = !isHidden;
    });
    restoreBtns.forEach(btn => {
      if (btn.classList.contains("nibble-restore-btn")) {
        btn.hidden = !isHidden;
      }
    });
  }

  syncRestoreButtons();

  function updateMood() {
    const maxScroll = Math.max(1, document.documentElement.scrollHeight - window.innerHeight);
    const progress = Math.min(1, Math.max(0, window.scrollY / maxScroll));
    const mood = progress >= 0.88 ? "3" : progress >= 0.58 ? "2" : progress >= 0.25 ? "1" : "0";

    mascots.forEach(el => {
      if (el.getAttribute("data-mood") === "bye") return;
      if (el.getAttribute("data-mood") !== mood) {
        el.setAttribute("data-mood", mood);
      }
    });
  }

  const farewells = ["Bye! o/", "See ya! o/", "FIN/ACK! o/", "EOF! o/"];
  const greetings = ["Hi again! o/", "SYN/ACK! o/", "Back online!", "Beep! o/"];
  const interactionCounts = { trips: 0, pokes: 0 };
  function readInteractionCount(kind) {
    try {
      const stored = localStorage.getItem(`kmcd-nibble-${kind}`);
      const count = Number(stored);
      if (stored !== null && Number.isSafeInteger(count) && count >= 0) {
        interactionCounts[kind] = count;
      }
    } catch (err) {}
    return interactionCounts[kind];
  }
  function countInteraction(kind) {
    const count = Math.min(Number.MAX_SAFE_INTEGER, readInteractionCount(kind) + 1);
    interactionCounts[kind] = count;
    try {
      localStorage.setItem(`kmcd-nibble-${kind}`, String(count));
    } catch (err) {}
  }
  const playfulTricks = [
    { id: "flip", trick: "flip", mood: "3", text: "DO A BARREL ROLL!", duration: 750 },
    { id: "disco", trick: "disco", mood: "3", text: "OVERCLOCKING!!", duration: 1500 },
    { id: "glitch", trick: "glitch", mood: "2", text: "segfault (core dumped)", duration: 1100 },
  ];
  const nextTrickThreshold = () => 5 + Math.floor(Math.random() * 6);
  let clicksUntilTrick = nextTrickThreshold();
  let trickPending = false;

  mascots.forEach(mascot => {
    let trickTimer = null;
    mascot.querySelector(".toc-mascot-link")?.addEventListener("click", () => {
      countInteraction("trips");
    });
    mascot.addEventListener("contextmenu", async (e) => {
      e.preventDefault();
      // Every poke counts, even while a trick is waiting for its connection check.
      countInteraction("pokes");
      const panel = mascot.closest(".toc-mascot-panel");
      const speech = panel ? panel.querySelector(".nibble-speech") : null;
      if (!speech || panel.hidden || mascot.getAttribute("data-mood") === "bye" || trickPending) return;
      if (--clicksUntilTrick > 0) return;
      clicksUntilTrick = nextTrickThreshold();
      trickPending = true;

      if (trickTimer) clearTimeout(trickTimer);
      mascot.removeAttribute("data-trick");
      speech.hidden = true;

      let observations = [{ id: "offline", text: "Your connection went quiet. Still there?" }];
      const started = performance.now();
      try {
        const response = await fetch("/http/", {
          method: "HEAD",
          cache: "no-store",
          signal: AbortSignal.timeout(5000),
        });
        if (!response.ok) throw new Error("Connection check failed");
        const elapsed = Math.round(performance.now() - started);
        const city = response.headers.get("x-kmcd-city")?.trim();
        const ip = response.headers.get("x-kmcd-ip")?.trim();
        observations = [{ id: "latency", text: `The server answered in ${elapsed} ms. I'm still listening.` }];
        observations.push({ id: "ping", trick: "jump", mood: "3", text: `PONG! time=${elapsed}ms`, duration: 1400 });
        if (city) observations.push({ id: "city", text: `How are things in ${city}?` });
        if (ip) observations.push({ id: "ip", text: `${ip}. That's you, isn't it?` });
      } catch (err) {
        // Missing edge headers or an unavailable endpoint should never invent personal details.
      } finally {
        trickPending = false;
      }

      // A slow response must not interrupt Nibble's farewell or reveal a hidden panel.
      if (panel.hidden || mascot.getAttribute("data-mood") === "bye") return;
      const trips = readInteractionCount("trips");
      const pokes = readInteractionCount("pokes");
      const actions = [...observations, ...playfulTricks,
        { id: "trips", trick: "jump", mood: "3", text: `Random trip odometer: ${trips}. Where to next?` },
        { id: "pokes", trick: "glitch", mood: "2", text: `Poke counter: ${pokes}. Yes, I'm counting. Every. Single. One.` },
      ];
      const item = actions[Math.floor(Math.random() * actions.length)];
      if (item.trick) {
        void mascot.offsetWidth; // Restart the animation, including after rapid clicks.
        mascot.setAttribute("data-trick", item.trick);
      }
      mascot.setAttribute("data-mood", item.mood || "2");
      speech.textContent = item.text;
      speech.hidden = false;

      trickTimer = setTimeout(() => {
        mascot.removeAttribute("data-trick");
        if (mascot.getAttribute("data-mood") === "bye") return;
        speech.hidden = true;
        mascot.removeAttribute("data-mood");
        updateMood();
      }, item.duration || 5500);
    });
  });

  panels.forEach(panel => {
    const closeBtn = panel.querySelector(".nibble-close-btn");
    if (!closeBtn) return;

    closeBtn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();

      const mascot = panel.querySelector(".toc-mascot");
      const speech = panel.querySelector(".nibble-speech");
      const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

      if (mascot) {
        mascot.setAttribute("data-mood", "bye");
      }
      if (speech) {
        speech.textContent = farewells[Math.floor(Math.random() * farewells.length)];
        speech.hidden = false;
      }

      const waveDuration = reducedMotion ? 450 : 900;
      const fadeDuration = reducedMotion ? 0 : 260;

      setTimeout(() => {
        panel.classList.add("is-dismissing");
        setTimeout(() => {
          try {
            localStorage.setItem("kmcd-hide-nibble", "1");
          } catch (err) {}
          panels.forEach(p => {
            p.hidden = true;
            p.classList.remove("is-dismissing");
          });
          syncRestoreButtons();
        }, fadeDuration);
      }, waveDuration);
    });
  });

  restoreBtns.forEach(btn => {
    btn.addEventListener("click", () => {
      const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      try {
        localStorage.removeItem("kmcd-hide-nibble");
      } catch (err) {}
      syncRestoreButtons();
      const allPanels = Array.from(document.querySelectorAll(".toc-mascot-panel"));
      allPanels.forEach(p => {
        p.hidden = false;
        p.classList.remove("is-dismissing");
        const speech = p.querySelector(".nibble-speech");
        const mascot = p.querySelector(".toc-mascot");
        if (mascot) {
          mascot.setAttribute("data-mood", "bye");
        }
        if (speech) {
          speech.textContent = greetings[Math.floor(Math.random() * greetings.length)];
          speech.hidden = false;
        }
        setTimeout(() => {
          if (speech) speech.hidden = true;
          if (mascot) mascot.removeAttribute("data-mood");
          updateMood();
        }, reducedMotion ? 450 : 900);
      });
    });
  });

  if (mascots.length === 0) return;

  let ticking = false;
  window.addEventListener("scroll", () => {
    if (!ticking) {
      window.requestAnimationFrame(() => {
        updateMood();
        ticking = false;
      });
      ticking = true;
    }
  }, { passive: true });

  window.addEventListener("resize", updateMood, { passive: true });
  updateMood();
}

function initAll() {
  initImageFadeIn();
  initArchiveTags();
  initThemeAndPaletteControls();
  initTocScrollSpy();
  initMascotScrollMood();
  initKeyboardShortcuts();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAll);
} else {
  initAll();
}
