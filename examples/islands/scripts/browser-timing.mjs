// Run with Playwright CLI on the 1,000-product demo. Results are in ms.
async (page) => {
  const origin = new URL(page.url()).origin;
  const browser = page.context().browser();
  const hydrated = "[data-shugo-island][data-shugo-hydrated]";
  const samples = 7;
  const profiles = [
    { name: "local", latency: 0, download: -1, upload: -1, cpu: 1 },
    { name: "throttled", latency: 150, download: 1_600_000 / 8, upload: 750_000 / 8, cpu: 4 },
  ];
  const api = await page.request.get(origin + "/_islands/catalog?stock=1");
  const target = "Products: " + (await api.json()).total;

  async function open(context, profile) {
    const tab = await context.newPage();
    const devtools = await context.newCDPSession(tab);
    await devtools.send("Network.enable");
    await devtools.send("Network.emulateNetworkConditions", {
      offline: false,
      latency: profile.latency,
      downloadThroughput: profile.download,
      uploadThroughput: profile.upload,
    });
    await devtools.send("Emulation.setCPUThrottlingRate", { rate: profile.cpu });
    return tab;
  }

  async function load(path, profile) {
    const context = await browser.newContext({ viewport: { width: 1366, height: 768 } });
    try {
      await context.addInitScript(() => {
        const metrics = window.catalogMetrics = { cls: 0, lcp: 0, ready: null };
        new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (!entry.hadRecentInput) metrics.cls += entry.value;
          }
        }).observe({ type: "layout-shift", buffered: true });
        new PerformanceObserver((list) => {
          metrics.lcp = list.getEntries().at(-1).startTime;
        }).observe({ type: "largest-contentful-paint", buffered: true });
        const observer = new MutationObserver(() => {
          if (document.querySelector("[data-shugo-hydrated]")) {
            metrics.ready = performance.now();
            observer.disconnect();
          }
        });
        observer.observe(document, { attributes: true, attributeFilter: ["data-shugo-hydrated"], subtree: true });
      });
      const tab = await open(context, profile);
      const response = await tab.goto(origin + path);
      if (!response.ok()) throw new Error("unexpected navigation status " + response.status());
      if (path === "/catalog") await tab.locator(hydrated).waitFor();
      await tab.waitForFunction(() => performance.getEntriesByName("first-contentful-paint").length > 0);
      return await tab.evaluate(async () => {
        await new Promise((ready) => requestAnimationFrame(() => requestAnimationFrame(ready)));
        const navigation = performance.getEntriesByType("navigation")[0];
        const js = performance.getEntriesByType("resource").filter((entry) => new URL(entry.name).pathname.endsWith(".js"));
        return {
          ttfb: navigation.responseStart,
          fcp: performance.getEntriesByName("first-contentful-paint")[0].startTime,
          lcp: window.catalogMetrics.lcp,
          cls: window.catalogMetrics.cls,
          ready: window.catalogMetrics.ready,
          jsTransfer: js.reduce((total, entry) => total + entry.transferSize, 0),
          jsRequests: js.length,
        };
      });
    } finally {
      await context.close();
    }
  }

  async function interact(profile, js) {
    const context = await browser.newContext({ javaScriptEnabled: js });
    try {
      const tab = await open(context, profile);
      await tab.goto(origin + "/catalog");
      const checkbox = tab.getByLabel("In stock only");
      const button = tab.getByRole("button", { name: "Show results" });
      if (js) await tab.locator(hydrated).waitFor();
      else await checkbox.check(); // measure after the visitor is ready to submit
      let frameMs = null;
      let inputMs = null;
      if (js) {
        await checkbox.evaluate((element, expected) => {
          element.addEventListener("input", () => {
            let maxFrame = 0;
            let last = performance.now();
            const began = last;
            const heading = document.querySelector("#results-heading");
            function next(now) {
              maxFrame = Math.max(maxFrame, now - last);
              last = now;
              if (heading.textContent.trim() === expected) {
                requestAnimationFrame((painted) => {
                  window.catalogSample = { frameMs: Math.max(maxFrame, painted - last), inputMs: painted - began };
                });
              } else {
                requestAnimationFrame(next);
              }
            }
            requestAnimationFrame(next);
          }, { once: true });
        }, target);
      }
      const started = Date.now();
      if (js) {
        await checkbox.check();
      } else {
        await Promise.all([tab.waitForNavigation(), button.click()]);
      }
      await tab.getByText(target, { exact: true }).waitFor();
      const actionMs = Date.now() - started;
      if (js) {
        await tab.waitForFunction(() => window.catalogSample);
        ({ frameMs, inputMs } = await tab.evaluate(() => window.catalogSample));
      }
      return { actionMs, frameMs, inputMs };
    } finally {
      await context.close();
    }
  }

  const result = {};
  for (const profile of profiles) {
    const baseline = [], island = [], inline = [], form = [];
    for (let sample = 0; sample < samples; sample++) {
      if (sample % 2) {
        island.push(await load("/catalog", profile));
        baseline.push(await load("/baseline", profile));
        form.push(await interact(profile, false));
        inline.push(await interact(profile, true));
      } else {
        baseline.push(await load("/baseline", profile));
        island.push(await load("/catalog", profile));
        inline.push(await interact(profile, true));
        form.push(await interact(profile, false));
      }
    }
    result[profile.name] = { profile, baseline, island, inline, form };
  }
  return result;
}
