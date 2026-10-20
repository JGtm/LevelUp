// Pour chaque chassis du parc : vies, docs, echantillons, episodes, tirs publies (w),
// et le VOISIN (slot-1 / slot+1) de meme fenetre, avec sa famille.
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const agg = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const vs = doc.vehicles || [];
  const shotsByV = {};
  for (const s of doc.shots) if (s.v !== undefined) { (shotsByV[s.v] ||= {}); shotsByV[s.v][s.w] = (shotsByV[s.v][s.w]||0)+1; }
  for (const v of vs) {
    const k = v.chassis || 'none';
    const a = (agg[k] ||= { fam: v.family || '?', lives: 0, docs: new Set(), samples: 0, noSamples: 0, rides: 0, rideFrames: 0, seats: {}, srcs: {}, w: {}, neigh: {} });
    a.lives++; a.docs.add(f.slice(0,8)); a.samples += (v.samples?.length||0); if (!v.samples?.length) a.noSamples++;
    for (const r of (v.rides||[])) { a.rides++; a.rideFrames += r.t1 - r.t0; a.seats[r.seat ?? 'nul'] = (a.seats[r.seat ?? 'nul']||0)+1; a.srcs[r.src] = (a.srcs[r.src]||0)+1; }
    for (const [w, n] of Object.entries(shotsByV[v.slot] || {})) a.w[w] = (a.w[w]||0)+n;
    for (const o of vs) {
      if (o === v) continue;
      const d = o.slot - v.slot;
      if (Math.abs(d) !== 1) continue;
      const overlap = Math.min(o.t1 ?? 1e9, v.t1 ?? 1e9) - Math.max(o.t0, v.t0);
      if (overlap <= 0) continue;
      const key = (d > 0 ? '+1:' : '-1:') + (o.family || '?') + '/' + o.chassis;
      a.neigh[key] = (a.neigh[key]||0)+1;
    }
  }
}
for (const [k, a] of Object.entries(agg).sort((x,y)=>y[1].lives-x[1].lives)) {
  console.log(`${k} fam=${a.fam} vies=${a.lives} docs=${a.docs.size} ech=${a.samples} viesSansEch=${a.noSamples} episodes=${a.rides} (${(a.rideFrames/10).toFixed(0)}s) sieges=${JSON.stringify(a.seats)} src=${JSON.stringify(a.srcs)} tirs=${JSON.stringify(a.w)} voisins=${JSON.stringify(a.neigh)}`);
}
