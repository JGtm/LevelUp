// Balayage du parc : toutes les valeurs de Shot.w ABSENTES de weaponLabels, avec ou sans `v`.
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const agg = {}; let nDocs = 0, nShots = 0, nUnk = 0, schemas = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  nDocs++; schemas[doc.schemaVersion] = (schemas[doc.schemaVersion]||0)+1;
  const labels = doc.weaponLabels || {};
  const famOfSlot = {}; for (const v of (doc.vehicles||[])) famOfSlot[v.slot] = (v.family||'?') + '/' + (v.chassis||'-');
  for (const s of (doc.shots||[])) {
    nShots++;
    if (s.w === undefined) { const k = 'NO_W' + (s.v!==undefined?' +v':''); agg[k] = agg[k] || {n:0, docs:new Set(), fam:{}}; agg[k].n++; agg[k].docs.add(f.slice(0,8)); continue; }
    if (labels[s.w]) continue;
    nUnk++;
    const lo = s.w.slice(10);
    const k = s.w + (s.v!==undefined?' +v':' (a pied)');
    agg[k] = agg[k] || {n:0, docs:new Set(), fam:{}};
    agg[k].n++; agg[k].docs.add(f.slice(0,8));
    if (s.v!==undefined) { const fm = famOfSlot[s.v]||'nonpublie'; agg[k].fam[fm] = (agg[k].fam[fm]||0)+1; }
  }
}
console.log('docs', nDocs, 'schemas', JSON.stringify(schemas), 'shots', nShots, 'w hors weaponLabels', nUnk);
for (const [k, v] of Object.entries(agg).sort((a,b)=>b[1].n-a[1].n)) console.log(k, 'n=', v.n, 'docs=', v.docs.size, [...v.docs].slice(0,8).join(','), 'fam=', JSON.stringify(v.fam));
