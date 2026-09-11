/**
 * Theming.
 *
 * Supports the preferred color scheme of the operation system as well as
 * the theme choice of the user.
 *
 */
const themeToggles = document.querySelectorAll(".theme-toggle");

// Detect the color scheme the operating system prefers.
function detectOSColorTheme() {
  const chosenTheme = window.localStorage && window.localStorage.getItem("theme");
  const chosenThemeIsDark = chosenTheme == "dark";
  const chosenThemeIsLight = chosenTheme == "light";

  if (chosenThemeIsDark) {
    document.documentElement.setAttribute("data-theme", "dark");
  } else if (chosenThemeIsLight) {
    document.documentElement.setAttribute("data-theme", "light");
  } else if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
    document.documentElement.setAttribute("data-theme", "dark");
  } else {
    document.documentElement.setAttribute("data-theme", "dark");
  }
}

// Switch the theme.
let isThemeToggling = false;
function switchTheme(e) {
  if (e) {
    if (e.stopPropagation) e.stopPropagation();
    if (e.stopImmediatePropagation) e.stopImmediatePropagation();
  }
  if (isThemeToggling) return;
  isThemeToggling = true;
  setTimeout(() => { isThemeToggling = false; }, 200);

  const currentTheme = document.documentElement.getAttribute("data-theme") || "dark";
  const newTheme = currentTheme === "dark" ? "light" : "dark";

  try {
    if (window.localStorage) {
      window.localStorage.setItem("theme", newTheme);
    }
  } catch (err) {
    console.warn("Could not write theme to localStorage", err);
  }

  document.documentElement.setAttribute("data-theme", newTheme);
}

// Event listener
if (themeToggles.length > 0) {
  themeToggles.forEach(toggle => {
    toggle.addEventListener("click", switchTheme, false);
  });
  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", (e) => e.matches && detectOSColorTheme());
  window
    .matchMedia("(prefers-color-scheme: light)")
    .addEventListener("change", (e) => e.matches && detectOSColorTheme());

  detectOSColorTheme();
} else {
  localStorage.removeItem("theme");
}

/**
 * Palette Switcher with Live Hover Preview & Auto-Revert.
 */
let committedPalette = "monochrome";
let previewedPalette = "monochrome";

function updatePaletteActiveUI(activePalette, originPalette) {
  document.querySelectorAll(".palette-dot-btn").forEach(btn => {
    const target = btn.getAttribute("data-palette-target");
    if (target === activePalette) {
      btn.classList.add("active");
      btn.setAttribute("aria-checked", "true");
    } else {
      btn.classList.remove("active");
      btn.setAttribute("aria-checked", "false");
    }

    // If previewing a different palette, mark the original committed button with a subtle ring
    if (originPalette && target === originPalette && activePalette !== originPalette) {
      btn.classList.add("committed-origin");
    } else {
      btn.classList.remove("committed-origin");
    }
  });
}

function previewPalette(paletteName) {
  if (!paletteName || previewedPalette === paletteName) return;
  previewedPalette = paletteName;
  document.documentElement.setAttribute("data-palette", paletteName);
  updatePaletteActiveUI(paletteName, committedPalette);
}

function commitPalette(paletteName) {
  if (!paletteName) return;
  committedPalette = paletteName;
  previewedPalette = paletteName;
  if (window.localStorage) {
    try {
      window.localStorage.setItem("palette", paletteName);
    } catch (e) {
      console.warn("Could not write palette to localStorage", e);
    }
  }
  document.documentElement.setAttribute("data-palette", paletteName);
  updatePaletteActiveUI(paletteName, paletteName);
}

function revertPalette() {
  if (previewedPalette === committedPalette) return;
  previewedPalette = committedPalette;
  document.documentElement.setAttribute("data-palette", committedPalette);
  updatePaletteActiveUI(committedPalette, committedPalette);
}

function initPalette() {
  let initial = (window.localStorage && window.localStorage.getItem("palette")) || "monochrome";
  if (initial === "planetscale-orange") {
    initial = "orange";
    if (window.localStorage) {
      try { window.localStorage.setItem("palette", "orange"); } catch (e) {}
    }
  }
  committedPalette = initial;
  previewedPalette = committedPalette;
  document.documentElement.setAttribute("data-palette", committedPalette);
  updatePaletteActiveUI(committedPalette, committedPalette);

  // Capsule collapse timer (1 second delay on unhover)
  let collapseTimer = null;

  function scheduleCollapse(capsule) {
    clearTimeout(collapseTimer);
    collapseTimer = setTimeout(() => {
      capsule.classList.remove("expanded");
      revertPalette();
    }, 1000);
  }

  function cancelCollapse(capsule) {
    clearTimeout(collapseTimer);
    capsule.classList.add("expanded");
  }

  // Direct dot interactions
  document.querySelectorAll(".palette-dot-btn").forEach(btn => {
    btn.addEventListener("mouseenter", () => {
      const capsule = btn.closest(".theme-control-capsule");
      if (capsule) cancelCollapse(capsule);
      const target = btn.getAttribute("data-palette-target");
      if (target) previewPalette(target);
    });

    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      const capsule = btn.closest(".theme-control-capsule");
      if (capsule) cancelCollapse(capsule);
      const target = btn.getAttribute("data-palette-target");
      if (target) commitPalette(target);
    });

    btn.addEventListener("focus", () => {
      const capsule = btn.closest(".theme-control-capsule");
      if (capsule) cancelCollapse(capsule);
      const target = btn.getAttribute("data-palette-target");
      if (target) previewPalette(target);
    });
  });

  // Track mouse movement across the palette-switcher container for gap/padding hover preview
  document.querySelectorAll(".palette-switcher").forEach(switcher => {
    switcher.addEventListener("mousemove", (e) => {
      const capsule = switcher.closest(".theme-control-capsule");
      if (capsule) cancelCollapse(capsule);

      const clickX = e.clientX;
      let closestBtn = null;
      let minDistance = Infinity;
      switcher.querySelectorAll(".palette-dot-btn").forEach(btn => {
        const rect = btn.getBoundingClientRect();
        const center = rect.left + rect.width / 2;
        const dist = Math.abs(clickX - center);
        if (dist < minDistance) {
          minDistance = dist;
          closestBtn = btn;
        }
      });
      if (closestBtn) {
        const target = closestBtn.getAttribute("data-palette-target");
        if (target) previewPalette(target);
      }
    });

    switcher.addEventListener("click", (e) => {
      const clickX = e.clientX;
      let closestBtn = null;
      let minDistance = Infinity;
      switcher.querySelectorAll(".palette-dot-btn").forEach(btn => {
        const rect = btn.getBoundingClientRect();
        const center = rect.left + rect.width / 2;
        const dist = Math.abs(clickX - center);
        if (dist < minDistance) {
          minDistance = dist;
          closestBtn = btn;
        }
      });
      if (closestBtn) {
        const target = closestBtn.getAttribute("data-palette-target");
        if (target) commitPalette(target);
      }
    });
  });

  // Delegate hover, unhover (with 1s delay), and clicks on .theme-control-capsule
  document.querySelectorAll(".theme-control-capsule").forEach(capsule => {
    capsule.addEventListener("mouseenter", () => {
      cancelCollapse(capsule);
    });

    capsule.addEventListener("mouseleave", () => {
      scheduleCollapse(capsule);
    });

    capsule.addEventListener("focusin", () => {
      cancelCollapse(capsule);
    });

    capsule.addEventListener("focusout", (e) => {
      if (!capsule.contains(e.relatedTarget)) {
        scheduleCollapse(capsule);
      }
    });

    capsule.addEventListener("click", (e) => {
      cancelCollapse(capsule);
      // If the click directly hit a button, the button's own stopPropagation already handled it
      if (e.target.closest(".theme-toggle") || e.target.closest(".palette-dot-btn")) {
        return;
      }
      const paletteSwitcher = capsule.querySelector(".palette-switcher");
      // If the palette switcher is collapsed (width 0 or hidden), any click on the capsule toggles the theme
      const isExpanded = paletteSwitcher && paletteSwitcher.offsetWidth > 10;
      if (!isExpanded) {
        switchTheme(e);
        return;
      }

      const divider = capsule.querySelector(".theme-capsule-divider");
      if (divider) {
        const divRect = divider.getBoundingClientRect();
        if (e.clientX >= divRect.left) {
          // Clicked anywhere on the right side of divider -> trigger theme toggle
          switchTheme(e);
          return;
        }
      }
      // Clicked on the left side -> commit closest palette button
      const clickX = e.clientX;
      let closestBtn = null;
      let minDistance = Infinity;
      capsule.querySelectorAll(".palette-dot-btn").forEach(btn => {
        const rect = btn.getBoundingClientRect();
        const center = rect.left + rect.width / 2;
        const dist = Math.abs(clickX - center);
        if (dist < minDistance) {
          minDistance = dist;
          closestBtn = btn;
        }
      });
      if (closestBtn) {
        const target = closestBtn.getAttribute("data-palette-target");
        if (target) commitPalette(target);
      }
    });
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
  initPalette();
  initTocScrollSpy();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAll);
} else {
  initAll();
}

