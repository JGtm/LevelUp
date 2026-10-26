// Projectiles publies : naissent-ils sur un vehicule occupe (Ghost) pendant un episode ?
import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
console.log('projectiles', doc.projectiles.length, 'exemple', JSON.stringify(doc.projectiles[0]).slice(0, 300));
const posAt = (v, t) => { let best = null, bd = 1e9; for (const s of v.samples) { const d = Math.abs(s.t - t); if (d < bd) { bd = d; best = s; } } return best ? { ...best, dt: bd } : null; };
let near = 0;
for (const p of doc.projectiles) {
  const t0 = p.t0; const p0 = p.p[0];
  const x = p0[1], y = p0[2];
  let hit = [];
  for (const v of doc.vehicles) {
    const s = posAt(v, t0); if (!s || s.dt > 5) continue;
    const d = Math.hypot(s.x - x, s.y - y);
    if (d < 6) { const rides = (v.rides||[]).filter(r => t0 >= r.t0 && t0 <= r.t1).map(r => r.slot); hit.push(`${v.family}/${v.slot} d=${d.toFixed(1)} ride=${JSON.stringify(rides)}`); }
  }
  if (hit.length) near++;
  if (hit.length) console.log('proj t0', t0, 'p0', JSON.stringify(p0), 'n', p.p.length, '->', hit.join(' | '));
}
console.log('projectiles nes a < 6 m d un vehicule :', near);
