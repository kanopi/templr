async function initWasm() {
  const go = new Go(); // from wasm_exec.js
  const res = await WebAssembly.instantiateStreaming(fetch('templr.wasm'), go.importObject);
  go.run(res.instance);
}

function $(id){ return document.getElementById(id); }

async function readZipToMap(file) {
  const JSZip = await (await import('https://cdn.jsdelivr.net/npm/jszip@3.10.1/dist/jszip.min.js')).default;
  const data = await file.arrayBuffer();
  const zip = await JSZip.loadAsync(data);
  const out = {};
  const promises = [];
  zip.forEach((path, entry) => {
    if (!entry.dir) {
      promises.push(entry.async('string').then(s => { out[path] = s; }));
    }
  });
  await Promise.all(promises);
  return out;
}

async function render() {
  const template = $('template').value;
  const values = $('values').value;
  const helpers = $('helpers').value;
  const strict = $('strict').checked;
  const defaultMissing = $('dm').value || '<no value>';
  const injectGuard = $('injectGuard').checked;
  const guardMarker = $('guardMarker').value || '#templr generated';

  let files = {};
  const z = $('zip').files[0];
  if (z) {
    try { files = await readZipToMap(z); }
    catch (e) { $('output').value = `zip read error: ${e.message || e}`; return; }
  }

  const payload = JSON.stringify({
    template, values, helpers, defaultMissing, strict, files, injectGuard, guardMarker
  });
  const res = JSON.parse(window.templrRender(payload));
  $('output').value = res.error ? `ERROR: ${res.error}` : res.output;
}

function loadSample() {
  $('template').value = `# Demo
Name: {{ safe .name "anon" }}
City: {{ .city }}
From Files: {{ .Files.Get "hello.txt" | default "n/a" }}
`;
  $('values').value = `name: templr`;
  $('helpers').value = `{{- define "banner" -}}BANNER{{- end -}}`;
  $('dm').value = 'N/A';
  $('strict').checked = false;
  $('injectGuard').checked = true;
  $('guardMarker').value = '#templr generated';
}

window.addEventListener('DOMContentLoaded', async () => {
  await initWasm();
  $('render').addEventListener('click', render);
  $('sample').addEventListener('click', loadSample);
});
