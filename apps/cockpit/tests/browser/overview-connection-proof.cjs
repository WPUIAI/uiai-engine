const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const { readFileSync, writeFileSync } = require("node:fs");
const { join } = require("node:path");
const { chromium } = require("playwright");

const base = process.env.COCKPIT_PREVIEW_URL || "http://127.0.0.1:4173";
const output = join(process.cwd(), "visual-proof");
const keys = ["uiai.engine.url", "uiai.scope.project_root", "uiai.scope.continuity_id", "uiai.response.profile"];
const proof = { schema: "uiai.cockpit_consumer_proof.v1", commit_sha: process.env.GITHUB_SHA, fixture: true, status: "running", captures: [], checks: [] };
const storage = (page) => page.evaluate((names) => Object.fromEntries(names.map((key) => [key, localStorage.getItem(key)])), keys);

async function capture(page, name, width) {
  const metrics = await page.evaluate(() => ({ viewport: innerWidth, scroll: document.documentElement.scrollWidth }));
  await page.screenshot({ path: join(output, name), fullPage: true });
  proof.captures.push({ file: name, viewport_width: width, metrics, sha256: createHash("sha256").update(readFileSync(join(output, name))).digest("hex") });
  assert.equal(metrics.viewport, width, "the CSS viewport must match the requested width");
  assert.ok(metrics.scroll <= width + 1, `horizontal overflow: ${metrics.scroll}px in ${width}px viewport`);
}

async function run(browser, width) {
  let mode = "empty";
  const context = await browser.newContext({ viewport: { width, height: 1200 } });
  const page = await context.newPage();
  await page.addInitScript(() => {
    localStorage.setItem("uiai.engine.url", "http://127.0.0.1:7456");
    localStorage.setItem("uiai.scope.project_root", "/fixture/original");
    localStorage.setItem("uiai.scope.continuity_id", "fixture-original");
    localStorage.setItem("uiai.response.profile", "agent_compact");
  });
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.origin === base) return route.continue();
    const headers = { "Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Content-Type", "Access-Control-Allow-Methods": "GET, OPTIONS" };
    if (route.request().method() === "OPTIONS") return route.fulfill({ status: 204, headers });
    if (route.request().method() !== "GET") return route.abort("failed");
    const json = (body) => route.fulfill({ contentType: "application/json", headers, body: JSON.stringify(body) });
    if (url.pathname === "/health") {
      if (url.hostname === "localhost") return json({ status: "healthy", service: "not-uiai" });
      if (url.hostname === "unavailable.invalid" || mode === "offline") return route.abort("failed");
      return json({ status: "healthy", service: "uiai-engine" });
    }
    if (url.pathname === "/api/health/browser") {
      if (mode === "offline" || mode === "partial") return route.abort("failed");
      return json({ status: "healthy", browser_state: "ready", active_pages: 0, max_pages: 4 });
    }
    if (url.pathname === "/api/session/") {
      if (mode === "offline") return route.abort("failed");
      return json({ sessions: mode === "empty" ? [] : [{ id: "fixture-only", title: "Fixture browser session", url: "https://example.org/", width: 1280, height: 800 }], count: mode === "empty" ? 0 : 1, max: 4 });
    }
    // No real provider or browser-worker requests, mutations, scopes or sessions.
    return route.abort("failed");
  });

  try {
    await page.goto(`${base}/settings`);
    const before = await storage(page);
    await page.getByLabel("Engine URL", { exact: true }).fill("http://unavailable.invalid:7456");
    await page.getByLabel("Project root", { exact: true }).fill("/fixture/draft");
    await page.getByLabel("Response profile", { exact: true }).selectOption("evidence_grade");
    await page.getByRole("button", { name: "Test connection", exact: true }).click();
    await page.locator(".connection-result.error").filter({ hasText: "Engine unavailable" }).waitFor();
    assert.deepEqual(await storage(page), before, "a failed test must preserve the saved connection and unrelated drafts");
    await capture(page, `settings-failed-test-${width}.png`, width);

    await page.getByLabel("Engine URL", { exact: true }).fill("http://healthy.invalid:7456");
    await page.getByRole("button", { name: "Test connection", exact: true }).click();
    await page.locator(".connection-result:not(.error)").filter({ hasText: "healthy" }).waitFor();
    assert.deepEqual(await storage(page), before, "a successful test must not activate the candidate");
    await page.getByRole("button", { name: "Save changes", exact: true }).click();
    const saved = await storage(page);
    assert.equal(saved["uiai.engine.url"], "http://healthy.invalid:7456");
    assert.equal(saved["uiai.scope.project_root"], "/fixture/draft");
    assert.equal(saved["uiai.response.profile"], "evidence_grade");
    proof.checks.push({ width, check: "failed_and_successful_tests_read_only_explicit_save_activates", passed: true });

    await page.goto(base);
    await page.getByText("No live browser sessions", { exact: true }).waitFor();
    await capture(page, `overview-confirmed-empty-${width}.png`, width);
    mode = "offline";
    await page.getByRole("button", { name: "↻ Refresh", exact: true }).click();
    await page.getByRole("heading", { name: "Browser activity unavailable", exact: true }).waitFor();
    assert.equal(await page.getByText("No live browser sessions", { exact: true }).count(), 0);
    assert.equal(await page.getByText("All clear", { exact: true }).count(), 0);
    assert.equal(await page.locator(".screen-header .badge.success").count(), 0);
    await capture(page, `overview-unavailable-${width}.png`, width);

    mode = "live";
    await page.getByRole("button", { name: "↻ Refresh", exact: true }).click();
    await page.getByText("Fixture browser session", { exact: true }).waitFor();
    assert.equal(await page.getByRole("heading", { name: "Browser activity unavailable", exact: true }).count(), 0);
    mode = "partial";
    await page.getByRole("button", { name: "↻ Refresh", exact: true }).click();
    await page.locator('[role="alert"]').filter({ hasText: "Browser health:" }).waitFor();
    await page.getByText("Fixture browser session", { exact: true }).waitFor();
    await capture(page, `overview-partial-failure-${width}.png`, width);
    proof.checks.push({ width, check: "confirmed_empty_failure_recovery_and_independent_partial_failure", passed: true });
  } catch (error) {
    await page.screenshot({ path: join(output, `consumer-failure-${width}.png`), fullPage: true });
    throw error;
  } finally {
    await context.close();
  }
}

(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const width of [1440, 375]) await run(browser, width);
    proof.status = "passed";
  } catch (error) {
    proof.status = "failed";
    proof.error = error instanceof Error ? error.message : String(error);
    throw error;
  } finally {
    writeFileSync(join(output, "overview-connection-consumer-proof.json"), JSON.stringify(proof, null, 2));
    await browser.close();
  }
})().catch((error) => { console.error(error); process.exitCode = 1; });
