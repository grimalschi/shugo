// Run with Playwright CLI run-code --filename, on an open demo page.
async (page) => {
  const start = new URL(page.url());
  const origin = start.origin;
  const catalog = start.pathname === "/" ? "/catalog" : start.pathname;
  const endpoint = "/_islands/catalog";
  const hydrated = "[data-shugo-island][data-shugo-hydrated]";
  const busy = "[data-catalog-results] [aria-busy=false]";
  const cards = "[data-product-id]";
  const checks = [];
  const errors = [];
  let requests = 0;
  let navigations = 0;

  function assert(condition, description) {
    if (!condition) throw new Error(description);
  }
  function same(actual, expected, description) {
    assert(JSON.stringify(actual) === JSON.stringify(expected),
      description + "\nactual " + JSON.stringify(actual) + "\nexpected " + JSON.stringify(expected));
  }
  async function results(tab, search, settle = true) {
    const response = await tab.request.get(origin + endpoint + search);
    assert(response.ok(), "catalog API returned " + response.status());
    const expected = await response.json();
    const ids = expected.hits.map(({ id }) => id);
    if (settle) await tab.waitForFunction(({ ids, busy, cards }) =>
      document.querySelector(busy) && JSON.stringify(Array.from(document.querySelectorAll(cards),
        (card) => card.dataset.productId)) === JSON.stringify(ids), { ids, busy, cards });
    same(await tab.locator(cards).evaluateAll((found) => found.map((card) => card.dataset.productId)),
      ids, "SSR/DOM card IDs match the API for " + search);
    same(await tab.getByRole("heading", { level: 2 }).textContent(),
      "Products: " + expected.total, "result count matches the API");
    return expected;
  }

  async function controls(tab, filters) {
    const values = await tab.locator("form.filters").evaluate((form) => {
      const data = new FormData(form);
      const price = (name) => data.get(name) === "" ? null : Number(data.get(name));
      return {
        q: data.get("q"),
        category: data.getAll("category"),
        size: data.getAll("size"),
        stock: data.has("stock"),
        priceFrom: price("price_from"),
        priceTo: price("price_to"),
        sort: data.get("sort"),
      };
    });
    const { q, category, size, stock, priceFrom, priceTo, sort } = filters;
    same(values, { q, category, size, stock, priceFrom, priceTo, sort },
      "visible filter controls match the normalized URL state");
  }

  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "warning" || /hydrat/i.test(message.text())) errors.push(message.text());
  });
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === endpoint) requests++;
    if (request.isNavigationRequest() && request.frame() === page.mainFrame()) navigations++;
  });

  await page.goto(origin + catalog);
  await page.locator(hydrated).waitFor();
  const all = await results(page, "");
  same(requests, 0, "hydration uses the SSR snapshot without fetching it again");
  same(all.hits.length, 24, "seed at least 24 products");
  checks.push("default SSR, one hydration and zero initial API requests");

  const deep = "?q=" + encodeURIComponent("cup") + "&stock=1&sort=cheap&page=2";
  let releaseModule;
  const waitForSSR = new Promise((resolve) => { releaseModule = resolve; });
  const moduleRoute = (url) => url.pathname.startsWith("/_assets/island-") && url.pathname.endsWith(".js");
  await page.route(moduleRoute, async (route) => {
    await waitForSSR;
    await route.continue();
  });
  let direct;
  try {
    direct = await page.goto(origin + catalog + deep, { waitUntil: "commit" });
    await page.locator(cards).first().waitFor();
    same(await page.locator(hydrated).count(), 0, "server-rendered cards arrive before the module");
    await page.evaluate((cards) => {
      window.ssrHeading = document.querySelector("#results-heading");
      window.ssrCard = document.querySelector(cards);
    }, cards);
    releaseModule();
    await page.locator(hydrated).waitFor();
  } finally {
    releaseModule();
    await page.unrouteAll({ behavior: "wait" });
  }
  assert(await page.evaluate((cards) =>
    window.ssrHeading === document.querySelector("#results-heading") &&
    window.ssrCard === document.querySelector(cards), cards),
    "strict hydration preserves the exact heading and first card from server HTML");
  const open = await results(page, deep);
  assert(open.page === 2, "seed at least 1,000 products for the deep-link check");
  await controls(page, open.filters);
  same((await direct.allHeaders())["x-robots-tag"], "noindex", "filtered HTML robots tag");
  same(requests, 0, "filtered deep-link hydration does not duplicate the API request");
  checks.push("query-specific SSR, page two and hydration preserving the server DOM with zero API requests");

  const before = navigations;
  const search = page.getByRole("searchbox");
  const submit = page.getByRole("button", { name: "Show results" });
  const urlSearch = () => new URL(page.url()).search;
  await search.fill("cup");
  await page.getByLabel("Sort by").selectOption("expensive");
  const sorted = await results(page, urlSearch());
  assert(sorted.page === 1 && sorted.hits.every((hit, n, hits) => n === 0 || hit.price <= hits[n - 1].price),
    "sort applies and resets page");

  const categoryInputs = page.getByRole("group", { name: "Categories" }).getByRole("checkbox");
  await categoryInputs.nth(0).check();
  await results(page, urlSearch());
  await categoryInputs.nth(1).check();
  same(new URL(page.url()).searchParams.getAll("category").length, 2, "category is repeated in the URL");
  const size = page.getByRole("group", { name: "Sizes" }).getByRole("checkbox").first();
  await size.check();
  await results(page, urlSearch());
  await page.getByLabel("Minimum price").fill("10");
  await page.getByLabel("Maximum price").fill("3000");
  await submit.click();
  const filtersURL = urlSearch();
  const combined = await results(page, filtersURL);
  same(combined.filters.priceTo, 3000, "number range applies");
  await page.getByRole("link", { name: "Reset" }).click();
  await results(page, "");
  same(urlSearch(), "", "reset removes query string");
  await page.evaluate(() => history.back());
  await page.waitForURL((url) => url.search === filtersURL);
  await results(page, filtersURL);
  await controls(page, combined.filters);
  await page.evaluate(() => history.forward());
  await page.waitForURL((url) => url.search === "");
  await results(page, "");
  await controls(page, all.filters);
  checks.push("repeated facets, stock, price, sorting, reset and Back/Forward with control values restored");

  await page.getByRole("link", { name: "Next" }).click();
  same((await results(page, urlSearch())).page, 2, "paging preserves catalog state");
  same(new URL(page.url()).pathname, catalog, "paging keeps the catalog's own route");
  assert(await page.locator("#results-heading").evaluate((heading) => heading === document.activeElement),
    "paging moves keyboard focus to results");
  await page.getByRole("link", { name: "Previous" }).click();
  await results(page, "");
  same(page.url(), origin + catalog, "returning to the first page drops the empty query");
  same(navigations, before, "all interactive actions stayed in one document");
  checks.push("keyboard focus and client-side pagination");

  let firstStarted;
  const started = new Promise((resolve) => { firstStarted = resolve; });
  let releaseFirst;
  const slow = new Promise((resolve) => { releaseFirst = resolve; });
  const apiRoute = (url) => url.pathname === endpoint;
  await page.route(apiRoute, async (route) => {
    const q = new URL(route.request().url()).searchParams.get("q");
    if (q === "cup") {
      firstStarted();
      await slow;
    }
    await route.continue().catch(() => {});
  });
  await search.fill("cup");
  await submit.click();
  await started;
  await page.locator("[data-catalog-results] [aria-busy=true]").waitFor();
  same(await page.locator(cards).count(), 24, "keeps existing cards during a slow fetch");
  await search.fill("tray");
  await submit.click();
  const current = urlSearch();
  await results(page, current);
  releaseFirst();
  await page.unrouteAll({ behavior: "wait" });
  await results(page, current);
  checks.push("delayed response cannot replace the latest results");

  await page.route(apiRoute, (route) => route.fulfill({ status: 503, body: "temporary error" }), { times: 1 });
  await search.fill("thisproductdoesnotexist");
  await submit.click();
  await page.getByRole("alert").waitFor();
  assert(await page.locator(cards).count() > 0, "failed fetch preserves previously loaded cards");
  await page.getByRole("button", { name: "Retry" }).click();
  await results(page, urlSearch());
  await page.getByText("No products match these filters.").waitFor();
  checks.push("retry after HTTP failure, keeps cards and renders empty state");
  same(navigations, before, "race and failure checks also stayed in the same document");

  const browser = page.context().browser();
  const noJS = await browser.newContext({ javaScriptEnabled: false });
  try {
    const tab = await noJS.newPage();
    await tab.goto(origin + catalog + deep);
    await results(tab, deep, false);
    await controls(tab, open.filters);
    same(await tab.locator(hydrated).count(), 0, "no JS context stays unhydrated");
    await Promise.all([
      tab.waitForNavigation(),
      tab.getByRole("link", { name: "Previous" }).click(),
    ]);
    same(new URL(tab.url()).pathname, catalog, "native pagination keeps the catalog's own route");
    const previous = await results(tab, new URL(tab.url()).search, false);
    same(previous.page, 1, "native pagination opens the previous page");
    await tab.getByRole("searchbox").fill("mug");
    const [submitted] = await Promise.all([
      tab.waitForNavigation(),
      tab.getByRole("button", { name: "Show results" }).click(),
    ]);
    same(submitted.request().method(), "GET", "native form submits a document GET");
    const url = new URL(tab.url());
    same(url.searchParams.get("q"), "mug", "native form carries the edited parameter");
    const raw = await results(tab, url.search, false);
    await controls(tab, raw.filters);
    same(await tab.locator(hydrated).count(), 0, "native GET results are rendered without client JS");
  } finally {
    await noJS.close();
  }
  checks.push("browser with JavaScript disabled can deep-link and submit a native GET form");

  const samples = '"< & > 🏺 </script><span data-shugo-test-injection>';
  await page.goto(origin + catalog + "?q=" + encodeURIComponent(samples));
  await page.locator(hydrated).waitFor();
  same(await page.getByRole("searchbox").inputValue(), samples, "SSR preserves HTML-sensitive search text");
  same(await page.locator("[data-shugo-test-injection]").count(), 0, "props remain inert JSON");
  checks.push("HTML-sensitive query stays text during SSR and hydration");

  await page.goto(origin + "/multiple");
  await page.waitForFunction((selector) => document.querySelectorAll(selector).length === 2, hydrated);
  const buttons = page.getByRole("button", { name: /island: increment/ });
  await buttons.first().click();
  same(await buttons.allTextContents(), ["Count 1", "Count 10"], "each island has its own state");
  same(await page.locator("script[type=module]").count(), 1, "repeated islands share one module tag");
  await page.goto(origin + "/");
  same(await page.locator("script[type=module]").count(), 0, "ordinary Shugo page receives no module");
  same(errors, [], "browser logged no JavaScript errors or hydration warnings");
  checks.push("two independent instances, one module, and no JS on the ordinary page");
  return { checks, requests, navigations, errors };
}
