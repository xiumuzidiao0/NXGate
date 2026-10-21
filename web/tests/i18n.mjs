import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
// Load the plain dictionary without changing the browser project's module mode.
const dictionarySource = readFileSync(new URL('../src/i18n.js', import.meta.url), 'utf8');
const { I18N } = await import(`data:text/javascript;base64,${Buffer.from(dictionarySource).toString('base64')}`);
assert.deepEqual(Object.keys(I18N.en).sort(), Object.keys(I18N.zh).sort(), 'English and Chinese dictionary keys must match');
for (const [language, dictionary] of Object.entries(I18N)) {
  for (const [key, value] of Object.entries(dictionary)) assert.ok(typeof value === 'string' && value.trim(), `${language}.${key} must not be empty`);
}
const html = readFileSync(new URL('../dist/index.html', import.meta.url), 'utf8');
for (const [, key] of html.matchAll(/data-i18n(?:-placeholder|-title)?="([^"]+)"/g)) {
  assert.ok(key in I18N.en, `Missing HTML translation: ${key}`);
}
assert.equal(I18N.en['pe.auth_none'], 'No Authentication');
assert.ok(I18N.en['sb.bind_outbound_help'].includes('does not guarantee a residential IP'));
console.log(`i18n dictionary parity, non-empty values and HTML references passed (${Object.keys(I18N.en).length} keys).`);
