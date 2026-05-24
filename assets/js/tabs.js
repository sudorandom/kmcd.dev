document.addEventListener("DOMContentLoaded", () => {
  const initTabs = () => {
    const wrappers = document.querySelectorAll(".tabs-wrapper");
    wrappers.forEach((wrapper) => {
      const buttons = wrapper.querySelectorAll(".tab-btn");
      const panels = wrapper.querySelectorAll(".tab-panel-content");

      // Hide all panels initially, then show the active one
      panels.forEach((p) => {
        p.style.display = "none";
      });
      const activeBtn = wrapper.querySelector(".tab-btn.active") || buttons[0];
      if (activeBtn) {
        const targetId = activeBtn.getAttribute("aria-controls");
        const targetPanel = wrapper.querySelector("#" + targetId);
        if (targetPanel) {
          targetPanel.style.display = "block";
        }
      }

      buttons.forEach((btn) => {
        btn.addEventListener("click", () => {
          const targetId = btn.getAttribute("aria-controls");

          // Deactivate all buttons in this wrapper
          buttons.forEach((b) => {
            b.classList.remove("active");
            b.setAttribute("aria-selected", "false");
          });

          // Hide all panels in this wrapper
          panels.forEach((p) => {
            p.style.display = "none";
          });

          // Activate clicked button
          btn.classList.add("active");
          btn.setAttribute("aria-selected", "true");

          // Show target panel
          const targetPanel = wrapper.querySelector("#" + targetId);
          if (targetPanel) {
            targetPanel.style.display = "block";
          }
        });
      });
    });
  };

  initTabs();
});
