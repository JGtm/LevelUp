// Tirs publies par joueur (filmIndex) : les index >= 16 recoivent-ils des tirs ?
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
let tot = {lo:0, hi:0, docs:0};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const roster = doc.roster || [];
  if (!roster.some(r => r.filmIndex >= 16)) continue;
  tot.docs++;
  const xuidIdx = {}; for (const r of roster) xuidIdx[r.xuid] = r.filmIndex;
  const slotX = {}; for (const tr of doc.tracks) slotX[tr.slot + ':' + tr.points[0].t] = tr.xuid;
  // piste d'un tir : slot + instant -> trace qui couvre
  const bySlot = {}; for (const tr of doc.tracks) (bySlot[tr.slot] ||= []).push(tr);
  const cnt = {};
  for (const s of doc.shots) {
    if (s.v !== undefined) continue;
    const tr = (bySlot[s.slot]||[]).find(tr => s.t >= tr.points[0].t && s.t <= tr.points[tr.points.length-1].t) || (bySlot[s.slot]||[])[0];
    const idx = tr ? xuidIdx[tr.xuid] : undefined;
    const k = idx === undefined ? 'inconnu' : idx;
    cnt[k] = (cnt[k]||0)+1;
  }
  const lo = Object.entries(cnt).filter(([k])=>k!=='inconnu' && +k<16).reduce((a,[,n])=>a+n,0);
  const hi = Object.entries(cnt).filter(([k])=>k!=='inconnu' && +k>=16).reduce((a,[,n])=>a+n,0);
  tot.lo += lo; tot.hi += hi;
  console.log(f.slice(0,8), 'roster', roster.length, 'max idx', Math.max(...roster.map(r=>r.filmIndex)), 'tirs idx<16:', lo, 'idx>=16:', hi, 'par idx', JSON.stringify(cnt));
}
console.log('TOTAL', JSON.stringify(tot));
