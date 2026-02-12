(function() {
  function initCompare() {
    const sliders = document.querySelectorAll('.compare-slider');
    if (sliders.length === 0) return;

    sliders.forEach(slider => {
      const wrapper = slider.closest('.compare-wrapper');
      const before = wrapper.querySelector('.compare-before');
      const beforeImg = before.querySelector('img');

      // Reset to 50 explicitly to prevent browsers from remembering state
      slider.value = 50;

      // Set the width of the before image to match the container width
      function syncWidth() {
        beforeImg.style.width = `${wrapper.offsetWidth}px`;
      }

      function updateSlider() {
        before.style.width = `${slider.value}%`;
      }

      slider.addEventListener('input', updateSlider);

      // Sync on image load (especially for SVGs without explicit dimensions)
      beforeImg.addEventListener('load', syncWidth);

      // Use ResizeObserver for more reliable width syncing
      const ro = new ResizeObserver(() => {
        syncWidth();
      });
      ro.observe(wrapper);

      // Initial sync
      syncWidth();
      updateSlider();
    });
  }

  // Run on DOM content loaded and window load (to ensure all assets/styles are ready)
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initCompare);
  } else {
    initCompare();
  }
  window.addEventListener('load', initCompare);
})();
