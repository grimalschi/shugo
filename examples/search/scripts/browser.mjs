// Run with Playwright CLI on an open page of this example.
async (page) => {
  const origin = new URL(page.url()).origin;
  const products = "/products?q=" + encodeURIComponent("cups") + "&category=ceramics&price_to=200&available=1";
  const articles = "/articles?q=" + encodeURIComponent("cup") + "&category=guides&price_from=5&available=1";
  const hydrated = "[data-shugo-island][data-shugo-hydrated]";
  const rows = "[data-hit-id]";
  const errors = [];
  const fetches = [];
  let documents = 0;

  function check(condition, description) {
    if (!condition) throw new Error(description);
  }
  async function expect(tab, expected) {
    await tab.waitForFunction(({ rows, expected }) =>
      JSON.stringify(Array.from(document.querySelectorAll(rows), (row) => row.dataset.hitId)) === JSON.stringify(expected),
    { rows, expected });
  }
  async function endpoint(tab) {
    const props = await tab.locator("[data-shugo-props]").textContent();
    return JSON.parse(props).initial.endpoint;
  }

  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "warning" || /hydrat/i.test(message.text())) errors.push(message.text());
  });
  page.on("request", (request) => {
    if (request.isNavigationRequest() && request.frame() === page.mainFrame()) documents++;
    const { pathname } = new URL(request.url());
    if (pathname.startsWith("/_islands/search/")) fetches.push(pathname);
  });

  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const js = (url) => url.pathname.startsWith("/_assets/island-") && url.pathname.endsWith(".js");
  await page.route(js, async (route) => { await gate; await route.continue(); });
  try {
    await page.goto(origin + products, { waitUntil: "commit" });
    await expect(page, ["p1"]);
    check(await page.locator(hydrated).count() === 0, "HTML already contains the filtered product before hydration");
    await page.evaluate((rows) => { window.ssrFirstRow = document.querySelector(rows); }, rows);
    release();
    await page.locator(hydrated).waitFor();
  } finally {
    release();
    await page.unrouteAll({ behavior: "wait" });
  }
  check(await page.evaluate((rows) => window.ssrFirstRow === document.querySelector(rows), rows),
    "hydration preserves the original server-rendered node");
  check(fetches.length === 0, "hydration does not requery the initial result");

  const productEndpoint = await endpoint(page);
  const initialDocuments = documents;
  await page.getByRole("searchbox").fill("");
  await page.locator("input[name=price_to]").fill("400");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page, ["p1", "p3"]);
  const submitted = page.url();
  check(new URL(submitted).searchParams.get("price_to") === "400", "submitted range is reflected in the address");
  await page.goBack();
  await expect(page, ["p1"]);
  check(page.url() === origin + products, "Back restores the initial filter URL");
  await page.goForward();
  await expect(page, ["p1", "p3"]);
  check(page.url() === submitted, "Forward restores the next filters");
  check(documents === initialDocuments && fetches.length === 3 && fetches.every((path) => path === productEndpoint),
    "all three changes query the product binding in the same document");

  await page.goto(origin + articles);
  await expect(page, ["a1"]);
  await page.locator(hydrated).waitFor();
  const articleEndpoint = await endpoint(page);
  check(articleEndpoint !== productEndpoint && fetches.length === 3, "article hydration uses its own SSR binding");
  await page.getByRole("searchbox").fill("flax");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page, []);
  check(fetches.at(-1) === articleEndpoint, "article controls fetch only the article binding");

  const browser = page.context().browser();
  const context = await browser.newContext({ javaScriptEnabled: false });
  try {
    const noJS = await context.newPage();
    await noJS.goto(origin + articles);
    await expect(noJS, ["a1"]);
    check(await noJS.locator(hydrated).count() === 0, "the article link works without client code");
    await noJS.getByRole("searchbox").fill("teapot");
    const [response] = await Promise.all([
      noJS.waitForNavigation(),
      noJS.getByRole("button", { name: "Search" }).click(),
    ]);
    await expect(noJS, []);
    check(response.request().method() === "GET" && new URL(noJS.url()).pathname === "/articles",
      "the article form submits a native GET on its own route");
  } finally {
    await context.close();
  }
  check(errors.length === 0, JSON.stringify(errors));
  return { productEndpoint, articleEndpoint, clientRequests: fetches.length, errors };
}
