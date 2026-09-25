// Pendant un episode, quels canaux du document parlent encore de l'occupant (slot) ?
import fs from 'fs';
const [id, slotS, t0S, t1S] = process.argv.slice(2);
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const slot = +slotS, t0 = +t0S, t1 = +t1S;
for (const [k, v] of Object.entries(doc)) {
  if (!Array.isArray(v) || !v.length || typeof v[0] !== 'object') continue;
  const hits = v.filter(e => (e.slot === slot || e.owner === slot) && ((e.t ?? e.t0) >= t0) && ((e.t ?? e.t0) <= t1));
  if (hits.length) console.log(k, hits.length, JSON.stringify(hits.slice(0, 4)).slice(0, 400));
}
const tr = doc.tracks.find(t => t.slot === slot);
const pts = tr.points.filter(p => p.t >= t0 && p.t <= t1);
console.log('points de trace pendant l episode', pts.length, 'premier/dernier', JSON.stringify(pts[0]), JSON.stringify(pts[pts.length-1]));
