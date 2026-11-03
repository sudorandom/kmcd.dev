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
    if (treeNav.scrollHeight > treeNav.clientHeight) {
      const linkRect = target.link.getBoundingClientRect();
      const navRect = treeNav.getBoundingClientRect();
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
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>s</kbd></dt><dd>Go to Stuff</dd></div>
              <div><dt><kbd>g</kbd> <span class="kbd-then">+</span> <kbd>m</kbd></dt><dd>Go to About Me</dd></div>
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

    // Handle 'g' chord sequences (g h, g p, g d, g s, g m, g g)
    if (gPending) {
      clearGPending();
      if (e.key === "h") {
        e.preventDefault();
        window.location.href = "/";
      } else if (e.key === "p") {
        e.preventDefault();
        window.location.href = "/posts/";
      } else if (e.key === "d") {
        e.preventDefault();
        window.location.href = "/prompts/";
      } else if (e.key === "s") {
        e.preventDefault();
        window.location.href = "/stuff/";
      } else if (e.key === "m") {
        e.preventDefault();
        window.location.href = "/me/";
      } else if (e.key === "g") {
        e.preventDefault();
        window.scrollTo({ top: 0, behavior: "smooth" });
      }
      return;
    }

    if (e.key === "g") {
      gPending = true;
      gTimer = setTimeout(clearGPending, 1000);
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

function initAll() {
  initImageFadeIn();
  initArchiveTags();
  initThemeAndPaletteControls();
  initTocScrollSpy();
  initKeyboardShortcuts();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAll);
} else {
  initAll();
}
