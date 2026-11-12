// Probe for the app-runner review round 2026-11-13T01-31-14+0800_130f89.
// Reproduces: (A) a title-only related link is persisted, (B) the command-center
// peek then renders an openable button for the empty URL, (C) dropping a row on
// "+ 新建分类" does nothing although README documents it, (D) how many save calls
// a double click on the sheet save button produces.
const {chromium} = require('playwright');
const fs = require('node:fs/promises');
const path = require('node:path');
const {spawn} = require('node:child_process');
const assert = require('node:assert/strict');
const root = path.resolve(__dirname, '..', '..', '..');
const output = path.join(root, 'selfcheck', 'review');
const port = 9392;

async function fixture(library, name, title, category) {
  const dir = path.join(library, name);
  await fs.mkdir(path.join(dir, 'content'), {recursive: true});
  await fs.writeFile(path.join(dir, 'content', 'app.exe'), 'probe entry');
  const lines = ['title = "' + title + '"', 'type = "GUI"', 'entry = "app.exe"'];
  if (category) lines.push('category = "' + category + '"');
  await fs.writeFile(path.join(dir, 'app.toml'), lines.join('\n') + '\n');
  return dir;
}

async function main() {
  const library = await fs.mkdtemp(path.join(output, 'probe-library-'));
  const alpha = await fixture(library, 'alpha', 'Alpha Editor');
  await fixture(library, 'beta', 'Beta Terminal');
  const binary = path.join(root, 'selfcheck', 'test-bin', 'app-runner-ui-test.exe');
  const cache = path.join(library, '.cache');
  const server = spawn(binary, ['--app-dir', library], {
    cwd: root, windowsHide: true,
    env: {...process.env, LOCALAPPDATA: cache, XDG_CACHE_HOME: cache, WAILS_SERVER_HOST: '127.0.0.1', WAILS_SERVER_PORT: String(port)},
  });
  let serverOutput = '';
  server.stdout.on('data', chunk => serverOutput += chunk);
  server.stderr.on('data', chunk => serverOutput += chunk);
  let browser;
  const results = {};
  try {
    for (let i = 0; i < 120; i++) {
      try { const response = await fetch('http://127.0.0.1:' + port); if (response.ok) break; } catch {}
      if (server.exitCode !== null) throw new Error(serverOutput);
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    browser = await chromium.launch({headless: true, channel: 'msedge'});
    const page = await browser.newPage({viewport: {width: 920, height: 640}});
    await page.goto('http://127.0.0.1:' + port);
    await page.locator('#cmd-results .result').first().waitFor();

    /* (A) save a related link that only has a title */
    await page.locator('#cmd-search').fill('alpha');
    await page.locator('#cmd-peek [data-edit]').click();
    await page.locator('#sheet-layer.open').waitFor();
    await page.locator('[data-tab=links]').click();
    await page.locator('#add-link').click();
    await page.locator('#sheet-body [data-link="0"][data-key=title]').fill('只有名字');
    await page.locator('#sheet-save').click();
    await page.locator('#sheet-layer.open').waitFor({state: 'hidden'});
    const disk = await fs.readFile(path.join(alpha, 'app.toml'), 'utf8');
    results.persistedToml = disk.trim().split('\n').filter(line => line.startsWith('title') || line.startsWith('url'));
    results.linkCount = (disk.match(/\[\[related_links\]\]/g) || []).length;

    /* (B) the peek renders an open button for the empty URL */
    await page.locator('#cmd-peek').waitFor();
    results.peekLinkCount = await page.locator('#cmd-peek .peek-link').count();
    results.peekLinkUrls = await page.locator('#cmd-peek .peek-link').evaluateAll(nodes => nodes.map(node => node.dataset.openUrl));

    /* (C) dropping a row on "+ 新建分类" */
    await page.locator('#cmd-search').fill('打开应用库');
    await page.locator('.result').filter({hasText: '打开应用库'}).first().dblclick();
    await page.locator('#lib-table tr[data-id]').first().waitFor();
    const before = await fs.readFile(path.join(alpha, 'app.toml'), 'utf8');
    await page.locator('#lib-table tr[data-id]').filter({hasText: 'Alpha'}).dragTo(page.locator('#new-category'));
    await page.waitForTimeout(800);
    results.newCategoryDialogOpen = await page.locator('#category-dialog[open]').count();
    results.newCategoryCount = await page.locator('#lib-filters [data-filter^="category:"]').count();
    results.profileUnchanged = (await fs.readFile(path.join(alpha, 'app.toml'), 'utf8')) === before;

    /* (D) count service calls for a double click on the sheet save button */
    await page.locator('#lib-table tr[data-id]').filter({hasText: 'Alpha'}).locator('[data-act=edit]').click();
    await page.locator('#sheet-layer.open').waitFor();
    await page.locator('#edit-description').fill('double click probe');
    await page.waitForTimeout(600);
    const posts = [];
    await page.route('**/*', async route => {
      if (route.request().method() === 'POST') {
        const body = route.request().postData() || '';
        const match = /"name"\s*:\s*"([^"]+)"/.exec(body) || /(Save|CheckEntry)/.exec(body);
        posts.push(match ? match[1] || match[0] : body.slice(0, 80));
      }
      await route.continue();
    });
    await page.locator('#sheet-save').dblclick();
    await page.waitForTimeout(1500);
    await page.unroute('**/*');
    results.saveRequestCount = posts.length;
    results.savePosts = posts;
    results.sheetAfterDoubleClick = await page.locator('#sheet-layer.open').count();
    results.toastText = await page.locator('#toast').textContent().catch(() => '');
  } catch (error) {
    results.error = String(error && error.message || error).split('\n')[0];
  } finally {
    await browser?.close();
    server.kill();
  }
  console.log(JSON.stringify(results, null, 2));
}
main().catch(error => { console.error(error); process.exit(1); });
