// Exercise the real browser binary through the booba bridge under Node.
// This checks application behavior, not browser rendering or phone scanning.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import { setTimeout as delay } from 'node:timers/promises';
vm.runInThisContext(await readFile('web/wasm_exec.js', 'utf8'));
const go = new Go();
let exitCode;
go.exit = code => { exitCode = code; };
const { instance } = await WebAssembly.instantiate(await readFile('web/demos/make.wasm'), go.importObject);
const timeout = setTimeout(() => { console.error('QR generator timed out'); process.exit(1); }, 15000);
const running = go.run(instance);
async function expectOutput(text) {
  let output = '';
  for (let n = 0; n < 100 && !output.includes(text); n++) {
    await delay(25);
    output += bubbletea_read();
  }
  assert.ok(output.includes(text), `expected terminal output: ${text}`);
  return output;
}
try {
  for (let n = 0; typeof globalThis.bubbletea_read !== 'function' && n < 100; n++) await delay(20);
  assert.equal(typeof globalThis.bubbletea_read, 'function');
  bubbletea_resize(100, 40);
  await expectOutput('Make a QR code');
  // Negotiate the same program-status support as the browser action bar.
  bubbletea_write('\x1b]7501;?\x1b\\');
  const advertised = await expectOutput(':msg=');
  const encoded = advertised.match(/:msg=([A-Za-z0-9+/=]+)/)?.[1];
  assert.ok(encoded, 'action status payload');
  const description = Buffer.from(encoded, 'base64').toString('utf8');
  for (const label of ['Generate', 'Clear', 'Graphics', 'Quit']) {
    assert.ok(description.includes(label), `advertised control: ${label}`);
  }
  bubbletea_write('\x15'); // Ctrl+U
  await expectOutput('Your QR code will appear here.');
  bubbletea_write('\r');
  await expectOutput('Cannot generate:');
  bubbletea_write('\x1b[200~https://example.org/東京?q=☕\x1b[201~');
  await expectOutput('Press Enter to generate.');
  bubbletea_write('\r');
  await expectOutput('Ready to scan');
  bubbletea_resize(20, 10);
  await expectOutput('Enlarge');
  bubbletea_resize(100, 40);
  await expectOutput('Ready to scan');
  bubbletea_write('\x03');
  await running;
  assert.equal(exitCode, 0);
  console.log('QR generator: startup, paste, validation, generation, resize, and quit passed');
} finally {
  clearTimeout(timeout);
}
process.exit(0);
