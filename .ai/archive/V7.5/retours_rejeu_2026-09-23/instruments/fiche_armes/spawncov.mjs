import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const rows = [];
let tot = { lives:0, restated:0, wcDecoded:0, eqSpawned:0, docs:0 };
for (const f of files) {
  const doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8'));
  const wc = doc.coverage?.weaponChanges || {}, ec = doc.coverage?.equipmentChanges || {};
  const lives = doc.tracks.length;
  // premieres emissions weaponChanges par vie : ecart a la naissance
  const start = new Map(doc.tracks.map(t => [t.slot, t.startFrame ?? 0]));
  let firstNear = 0;
  const seen = new Set();
  for (const c of (doc.weaponChanges||[]).slice().sort((a,b)=>a.t-b.t)) { if (seen.has(c.slot)) continue; seen.add(c.slot); const s = start.get(c.slot); if (s !== undefined && c.t - s <= 10) firstNear++; }
  rows.push({ id: f.slice(0,8), lives, wcDec: wc.decoded, restated: wc.restated, taken: wc.taken, eqSpawned: ec.spawned, eqTaken: ec.taken, firstWcNearBirth: firstNear });
  tot.lives += lives; tot.restated += wc.restated||0; tot.wcDecoded += wc.decoded||0; tot.eqSpawned += ec.spawned||0; tot.docs++;
}
console.log(JSON.stringify(tot));
rows.sort((a,b)=>(b.eqSpawned||0)-(a.eqSpawned||0));
for (const r of rows.slice(0,15)) console.log(JSON.stringify(r));
console.log('premieres emissions d arme publiees <=1 s de la naissance (toutes vies):', rows.reduce((s,r)=>s+r.firstWcNearBirth,0));
