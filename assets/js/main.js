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
let currentPalette = "monochrome";

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
  let initialPalette = (window.localStorage && window.localStorage.getItem("palette")) || "monochrome";
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

  if (optionsContainer) {
    optionsContainer.querySelectorAll("button").forEach(btn => btn.setAttribute("tabindex", "-1"));
  }

  function resetIdleTimer() {
    clearTimeout(idleTimer);
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
    clearTimeout(collapseTimer);
    collapseTimer = setTimeout(() => {
      collapseCapsule();
    }, delay);
  }

  function cancelCollapse() {
    clearTimeout(collapseTimer);
    resetIdleTimer();
  }

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

  // 3. Palette dots inside options: click only (no hover preview)
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

function initAll() {
  initThemeAndPaletteControls();
  initTocScrollSpy();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAll);
} else {
  initAll();
}

