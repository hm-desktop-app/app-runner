const {chromium} = require('playwright');
const fs = require('node:fs/promises');
const path = require('node:path');
const {spawn} = require('node:child_process');
const root = path.resolve(__dirname, '..', '..', '..');
const output = path.join(root, 'selfcheck', 'review');
const port = 9393;
async function main() {
  const library = await fs.mkdtemp(path.join(output, 'probe2-library-'));
  const alpha = path.join(library, 'alpha');
  await fs.mkdir(path.join(alpha, 'content'), {recursive: true});
  await fs.writeFile(path.join(alpha, 'content', 'app.exe'), 'probe entry');
  await fs.writeFile(path.join(alpha, 'app.toml'), 'title = "Alpha Editor"\ntype = "GUI"\nentry = "app.exe"\n');
  const cache = path.join(library, '.cache');
  const server = spawn(path.join(root, 'selfcheck', 'test-bin', 'app-runner-ui-test.exe'), ['--app-dir', library], {
    cwd: root, windowsHide: true,
    env: {...process.env, LOCALAPPDATA: cache, XDG_CACHE_HOME: cache, WAILS_SERVER_HOST: '127.0.0.1', WAILS_SERVER_PORT: String(port)},
  });
  let out = '';
  server.stdout.on('data', c => out += c);
  server.stderr.on('data', c => out += c);
  const results = {};
  let browser;
  try {
    for (let i = 0; i < 120; i++) { try { const r = await fetch('http://127.0.0.1:' + port); if (r.ok) break; } catch {} if (server.exitCode !== null) throw new Error(out); await new Promise(r => setTimeout(r, 100)); }
    browser = await chromium.launch({headless: true, channel: 'msedge'});
    const page = await browser.newPage({viewport: {width: 920, height: 640}});
    await page.goto('http://127.0.0.1:' + port);
    await page.locator('#cmd-results .result').first().waitFor();
    await page.locator('#cmd-search').fill('alpha');
    await page.locator('#cmd-peek [data-edit]').click();
    await page.locator('#sheet-layer.open').waitFor();
    await page.waitForTimeout(300);
    results.beforeIds = await page.evaluate(() => ({title: document.querySelector('#edit-title')?.value, description: document.querySelector('#edit-description')?.value}));
    await page.locator('#edit-description').fill('DESC-PROBE');
    results.afterFill = await page.evaluate(() => ({title: document.querySelector('#edit-title')?.value, description: document.querySelector('#edit-description')?.value, state: document.querySelector('#sheet-state')?.textContent}));
    await page.locator('#sheet-save').click();
    await page.locator('#sheet-layer.open').waitFor({state: 'hidden'});
    await page.waitForTimeout(300);
    results.disk = await fs.readFile(path.join(alpha, 'app.toml'), 'utf8');
  } catch (error) { results.error = String(error && error.message || error).split('\n')[0]; }
  finally { await browser?.close(); server.kill(); }
  console.log(JSON.stringify(results, null, 2));
}
main().catch(e => { console.error(e); process.exit(1); });