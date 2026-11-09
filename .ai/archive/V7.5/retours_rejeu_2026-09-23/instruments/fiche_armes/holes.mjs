import fs from 'fs';
const o = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
let cls = { prefix:0, suffix:0, middle:0, interleaved:0, allmissing:0 };
const byDoc = {};
for (const h of o.holedDetail) {
  const p = h.presentSlots, m = h.missing;
  let c;
  if (p.length === 0) c = 'allmissing';
  else {
    const pmin = Math.min(...p), pmax = Math.max(...p);
    const below = m.filter(s => s < pmin).length, above = m.filter(s => s > pmax).length, inside = m.length - below - above;
    if (inside === 0 && above === 0) c = 'prefix';
    else if (inside === 0 && below === 0) c = 'suffix';
    else if (below === 0 && above === 0) c = 'middle';
    else c = 'interleaved';
  }
  cls[c]++;
  (byDoc[h.id] ||= []).push(`${h.t}:${c}:${h.present}/${h.alive}`);
}
console.log('classes des images-cles trouees', cls);
const ratio = o.holedDetail.map(h => h.missing.length / h.alive);
const bins = {}; for (const r of ratio) { const b = r >= 0.999 ? '100%' : r >= 0.5 ? '50-99%' : r >= 0.25 ? '25-49%' : '<25%'; bins[b] = (bins[b]||0)+1; }
console.log('part des vies manquantes par image-cle trouee', bins);
for (const [id, l] of Object.entries(byDoc).slice(0, 40)) console.log(id, l.join(' '));
