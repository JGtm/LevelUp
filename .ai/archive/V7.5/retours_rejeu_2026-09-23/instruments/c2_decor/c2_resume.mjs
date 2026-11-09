// Resume par vie (slot/gen) : pour chaque champ KF, valeur si constante sinon *N ; puis DL.
import fs from 'fs';
const [file, ...champsDemandes] = process.argv.slice(2);
const L = fs.readFileSync(file, 'utf8').trim().split('\n').map(l => l.split('\t'));
const kf = {}, dl = {};
for (const r of L) {
  const k = r[1] + '/' + r[2];
  if (r[0] === 'KF') { (kf[k] ??= { ts: new Set(), f: {} }); kf[k].ts.add(r[3]); (kf[k].f[r[4]] ??= new Set()).add(r[5]); }
  else { (dl[k] ??= []).push(r.slice(3)); }
}
const champs = new Set(); for (const k in kf) for (const c in kf[k].f) champs.add(c);
const liste = champsDemandes.length ? champsDemandes : [...champs].sort();
for (const k of Object.keys(kf).sort()) {
  const e = kf[k];
  const parts = liste.map(c => { const s = e.f[c]; if (!s) return `${c}=_`; return s.size === 1 ? `${c}=${[...s][0]}` : `${c}=*${s.size}`; });
  const d = dl[k] || [];
  const types = {}; const masks = {};
  for (const x of d) { types[x[1]] = (types[x[1]] || 0) + 1; masks[x[2]] = (masks[x[2]] || 0) + 1; }
  console.log(`${k} kf=${e.ts.size} DL=${d.length} ${JSON.stringify(types)}\n   ${parts.join(' ')}`);
  if (d.length && d.length < 12) console.log('   DLmasks', JSON.stringify(masks), 'ts', d.map(x => x[0]).slice(0, 5).join(','));
}
