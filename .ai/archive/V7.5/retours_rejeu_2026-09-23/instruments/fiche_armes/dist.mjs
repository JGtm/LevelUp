import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(DIR).filter(f => /^[0-9a-f]{8}\.json$/.test(f)).sort();
const q = (a, p) => { const s = [...a].sort((x,y)=>x-y); return s.length ? s[Math.min(s.length-1, Math.floor(p*s.length))] : NaN; };
const hole = [], short = [], wait = [], ahead = [];
for (const f of files) {
  const doc = JSON.parse(fs.readFileSync(DIR + f, 'utf8'));
  const fi = doc.frameIntervalMs || 100;
  const inv = doc.inventory || [], lo = doc.loadouts || [];
  const kts = [...new Set([...inv.map(e=>e.t), ...lo.map(e=>e.t)])].sort((a,b)=>a-b);
  if (kts.length < 2) continue;
  const grid = new Set(kts);
  for (let i = 1; i < kts.length; i++) { const d = kts[i]-kts[i-1]; if (d > 300) { const n = Math.round(d/200); for (let k = 1; k < n; k++) grid.add(Math.round(kts[i-1] + k*d/n)); } }
  for (let t = kts[0]-200; t >= 0; t -= 200) grid.add(t);
  for (let t = kts[kts.length-1]+200; t < doc.frameCount; t += 200) grid.add(t);
  const g = [...grid];
  for (const t of doc.tracks) {
    const s = t.startFrame ?? 0, e = t.endFrame ?? t.points[t.points.length-1].t;
    const mine = lo.filter(l => l.slot === t.slot && l.t >= s && l.t <= e).map(l=>l.t);
    const dur = (e - s) * fi / 1000;
    if (mine.length) { const w = (Math.min(...mine) - s) * fi / 1000; wait.push(w); if (w > 0) ahead.push(w); continue; }
    if (g.some(k => k > s && k < e)) hole.push(dur); else short.push(dur);
  }
}
const fmt = (a) => `n=${a.length} p10=${q(a,.1).toFixed(1)} med=${q(a,.5).toFixed(1)} p90=${q(a,.9).toFixed(1)} max=${q(a,.9999).toFixed(1)}`;
console.log('vies sans loadout malgre une image-cle (duree de vie, s) :', fmt(hole));
console.log('vies sans loadout finies avant toute image-cle (duree de vie, s) :', fmt(short));
console.log('vies avec loadout : attente naissance -> 1re lecture (s) :', fmt(wait));
