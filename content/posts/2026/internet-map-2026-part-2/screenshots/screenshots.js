import { chromium } from 'playwright';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.resolve(__dirname, '..');
const baseUrl = (process.env.URL || 'https://map.kmcd.dev').replace(/\/$/, '');

async function waitForMapReady(page, expectedView = '3d', settleMs = 4500) {
  await page.waitForFunction(
    (view) =>
      !document.body.classList.contains('is-loading') &&
      document.body.dataset.mapView === view &&
      document.getElementById('view-status')?.hidden,
    expectedView,
    { timeout: 60000 }
  );
  await page.waitForSelector(expectedView === '3d' ? '#globe-map canvas' : '#leaflet-map canvas', {
    timeout: 60000,
  });
  await page.waitForTimeout(settleMs);
}

async function createDarkPage(browser, width = 1280, height = 760) {
  const context = await browser.newContext({
    viewport: { width, height },
    deviceScaleFactor: 2,
  });
  await context.addInitScript(() => {
    localStorage.setItem('submarine-map-theme', 'dark');
  });
  const page = await context.newPage();
  return { context, page };
}

const onlyTargets = new Set(process.argv.slice(2));
function shouldRun(name) {
  return onlyTargets.size === 0 || onlyTargets.has(name);
}

async function main() {
  const browser = await chromium.launch({
    headless: true,
    args: ['--use-gl=angle', '--use-angle=metal', '--enable-webgl', '--ignore-gpu-blocklist'],
  });

  try {
    // 1. Cover image (clean ambient 3D globe at 1200x630 @ 2x)
    if (shouldRun('cover')) {
      console.log('Capturing cover.png...');
      const { context, page } = await createDarkPage(browser, 1200, 630);
      await page.goto(`${baseUrl}/?mode=ambient&view=3d&lat=30&lng=-30&altitude=1.85&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 5000);
      await page.screenshot({ path: path.join(outDir, 'cover.png'), type: 'png' });
      await context.close();
    }

    // 2. Globe overview (full UI)
    if (shouldRun('globe-overview')) {
      console.log('Capturing globe-overview.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=28&lng=-32&altitude=1.9&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 5000);
      await page.screenshot({ path: path.join(outDir, 'globe-overview.png'), type: 'png' });
      await context.close();
    }

    // 3. 3D Globe vs 2D projection comparison
    if (shouldRun('view-compare')) {
      console.log('Capturing view-globe.png and view-2d.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=25&lng=-20&altitude=1.95&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.screenshot({ path: path.join(outDir, 'view-globe.png'), type: 'png' });

      await page.goto(`${baseUrl}/?view=2d&lat=20&lng=-10&z=2.1&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '2d', 4500);
      await page.screenshot({ path: path.join(outDir, 'view-2d.png'), type: 'png' });
      await context.close();
    }

    // 4. Surface routes vs 3D cable arches comparison (identical camera angle)
    if (shouldRun('cables-compare')) {
      console.log('Capturing cables-arches.png and cables-surface.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?mode=ambient&view=3d&lat=26&lng=-38&altitude=1.75&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.screenshot({ path: path.join(outDir, 'cables-arches.png'), type: 'png' });

      await page.evaluate(() => {
        document.querySelector('.cable-mode-btn[data-mode="surface"]')?.click();
      });
      await page.waitForTimeout(2500);
      await page.screenshot({ path: path.join(outDir, 'cables-surface.png'), type: 'png' });
      await context.close();
    }

    // 5. Hub closeup showing BGP density, blue peering vs purple IP-only cities, and city labels
    if (shouldRun('hub-closeup')) {
      console.log('Capturing hub-closeup.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=49.0&lng=4.5&altitude=0.28&autoRotate=0&alerts=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 5000);
      await page.screenshot({ path: path.join(outDir, 'hub-closeup.png'), type: 'png' });
      await context.close();
    }

    // 6. Historical cable timeline view (filtered for decommissioned cables)
    if (shouldRun('historical-timeline')) {
      console.log('Capturing historical-timeline.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?year=2026&autoRotate=0&background=1&cables=1&pulses=1&alerts=0&peering=0&ips=0&ipv6=0&q=decomm&lat=39.1067&lng=6.1832&altitude=1.7000`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.screenshot({ path: path.join(outDir, 'historical-timeline.png'), type: 'png' });
      await context.close();
    }

    // 7. Cloudflare outage alerts
    if (shouldRun('cloudflare-alerts')) {
      console.log('Capturing cloudflare-alerts.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=35&lng=35&altitude=1.7&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);

      // Click the first outage item to focus the camera and open the alert detail panel
      const clickedItem = await page.evaluate(() => {
        const items = [...document.querySelectorAll('#outages-panel-content .outages-panel-item')];
        const ongoing = items.find((el) => !el.classList.contains('is-resolved')) || items[0];
        if (ongoing) {
          ongoing.click();
          return true;
        }
        document.getElementById('btn-show-outages-panel')?.click();
        return false;
      });
      console.log('Outage item clicked:', clickedItem);
      await page.waitForTimeout(3500);
      await page.screenshot({ path: path.join(outDir, 'cloudflare-alerts.png'), type: 'png' });
      await context.close();
    }

    // 8. Kiosk mode
    if (shouldRun('kiosk-plash')) {
      console.log('Capturing kiosk-plash.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?mode=ambient&view=3d&lat=22&lng=15&altitude=1.95&autoRotate=1&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.screenshot({ path: path.join(outDir, 'kiosk-plash.png'), type: 'png' });
      await context.close();
    }

    // 9. Layers menu
    if (shouldRun('layers-menu')) {
      console.log('Capturing layers-menu.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=35&lng=15&altitude=1.9&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.click('#btn-layers');
      await page.waitForTimeout(600);
      await page.screenshot({
        path: path.join(outDir, 'layers-menu.png'),
        type: 'png',
        clip: { x: 0, y: 0, width: 420, height: 445 },
      });
      await context.close();
    }

    // 10. Search ('France')
    if (shouldRun('search-france')) {
      console.log('Capturing search-france.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=45.5&lng=2.5&altitude=1.15&autoRotate=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.click('#btn-filter');
      await page.fill('#filter-input', 'France');
      await page.press('#filter-input', 'Enter');
      await page.waitForTimeout(2500);
      await page.screenshot({ path: path.join(outDir, 'search-france.png'), type: 'png' });
      await context.close();
    }

    // 11. Statistics panel
    if (shouldRun('statistics-panel')) {
      console.log('Capturing statistics-panel.png...');
      const { context, page } = await createDarkPage(browser, 1280, 760);
      await page.goto(`${baseUrl}/?view=3d&lat=28&lng=-25&altitude=1.85&autoRotate=0&alerts=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 4500);
      await page.evaluate(() => {
        document.getElementById('btn-show-stats-panel')?.click();
      });
      await page.waitForTimeout(1500);
      await page.screenshot({ path: path.join(outDir, 'statistics-panel.png'), type: 'png' });
      await context.close();
    }

    // 12. Smooth fly-in animation (fly-in.webp)
    if (shouldRun('fly-in')) {
      const totalFrames = 240;
      const zoomStart = 15;
      const zoomFrames = 200;
      const startAlt = 1.85;
      const endAlt = 0.065;
      console.log(`Capturing ${totalFrames} high-quality fly-in.webp frames...`);
      const { execFileSync } = await import('child_process');
      const fs = await import('fs');
      const tmpDir = path.join('/tmp', 'kmcd-fly-in-frames');
      fs.rmSync(tmpDir, { recursive: true, force: true });
      fs.mkdirSync(tmpDir, { recursive: true });

      const width = 1280;
      const height = 760;
      const context = await browser.newContext({
        viewport: { width, height },
        deviceScaleFactor: 1.5,
      });
      await context.addInitScript(() => {
        localStorage.setItem('submarine-map-theme', 'dark');
      });
      const page = await context.newPage();
      await page.goto(`${baseUrl}/?view=3d&lat=50.5&lng=4.5&altitude=${startAlt}&autoRotate=0&alerts=0&year=2026`, {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
      });
      await waitForMapReady(page, '3d', 5000);

      // Ensure no city or cable is hovered
      await page.evaluate(() => {
        document.getElementById('globe-map')?.dispatchEvent(new PointerEvent('pointerleave'));
      });

      const easeInOut = (t) => 0.5 - 0.5 * Math.cos(t * Math.PI);
      const altAt = (t) => startAlt * Math.pow(endAlt / startAlt, easeInOut(t));
      const webpFrames = [];

      for (let i = 0; i < totalFrames; i++) {
        if (i >= zoomStart && i < zoomStart + zoomFrames) {
          const k = i - zoomStart;
          const altPrev = altAt(k / zoomFrames);
          const altNext = altAt((k + 1) / zoomFrames);
          // Exact OrbitControls inverse for dolly scale = Math.pow(0.95, 0.5 * Math.sqrt(alt) * |deltaY| * 0.01)
          const deltaY =
            (-100 * Math.log((1 + altNext) / (1 + altPrev))) /
            (0.5 * Math.sqrt(altPrev) * Math.log(0.95));

          await page.evaluate(
            ({ dy, cx, cy }) => {
              const canvas = document.querySelector('#globe-map canvas');
              canvas?.dispatchEvent(
                new WheelEvent('wheel', {
                  deltaY: dy,
                  clientX: cx,
                  clientY: cy,
                  deltaMode: 0,
                  bubbles: true,
                  cancelable: true,
                })
              );
            },
            { dy: deltaY, cx: width / 2, cy: height / 2 }
          );
          // Wait 110ms so both scheduleCityRecluster(80) and VIEWPORT_PATCH_DELAY_MS(100) complete
          await page.waitForTimeout(110);
        } else {
          await page.waitForTimeout(50);
        }

        const pngPath = path.join(tmpDir, `frame-${String(i).padStart(3, '0')}.png`);
        const webpPath = path.join(tmpDir, `frame-${String(i).padStart(3, '0')}.webp`);
        await page.screenshot({ path: pngPath, type: 'png' });
        execFileSync('/opt/homebrew/bin/cwebp', ['-q', '92', pngPath, '-o', webpPath], { stdio: 'ignore' });
        webpFrames.push(webpPath);
      }

      const bounceFrames = [...webpFrames, ...webpFrames.slice(1, -1).reverse()];
      const args = ['-loop', '0'];
      bounceFrames.forEach((f, idx) => {
        const isTurnaround = idx === totalFrames - 1;
        const isLoopEnd = idx === bounceFrames.length - 1;
        const delay = isTurnaround || isLoopEnd ? '700' : '33';
        args.push('-d', delay, '-lossy', '-q', '92', f);
      });
      args.push('-o', path.join(outDir, 'fly-in.webp'));
      execFileSync('/opt/homebrew/bin/img2webp', args, { stdio: 'inherit' });
      fs.rmSync(tmpDir, { recursive: true, force: true });
      await context.close();
    }

    console.log('All requested screenshots generated successfully!');
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
