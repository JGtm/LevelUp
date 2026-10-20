// Tirs poses sur une vie de chassis ENFANT sans echantillon : ou tombent-ils par rapport au chassis
// porteur (slot+1 / slot+2, meme fenetre) ?
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const posAt = (samples, t) => { if (!samples?.length) return null; let i = samples.findIndex(s => s.t >= t); if (i === -1) i = samples.length - 1; if (i === 0) return samples[0]; const a = samples[i-1], b = samples[i]; if (b.t === a.t) return b; const f = (t - a.t)/(b.t - a.t); return { x: a.x + (b.x-a.x)*Math.min(1,Math.max(0,f)), y: a.y + (b.y-a.y)*Math.min(1,Math.max(0,f)) }; };
const agg = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const bySlot = {}; for (const v of doc.vehicles || []) (bySlot[v.slot] ||= []).push(v);
  for (const s of doc.shots) {
    if (s.v === undefined) continue;
    const v = (bySlot[s.v] || []).find(o => s.t >= o.t0 - 5 && s.t <= (o.t1max ?? 1e9) + 5) || (bySlot[s.v]||[])[0];
    if (!v || v.samples?.length) continue;
    let parent = null;
    for (const d of [1, 2]) { parent = (bySlot[s.v + d] || []).find(o => o.samples?.length && s.t >= o.t0 && s.t <= (o.t1max ?? 1e9)); if (parent) break; }
    const k = (v.chassis||'?') + ' -> parent ' + (parent ? parent.family + '/' + parent.chassis : 'aucun');
    const a = (agg[k] ||= { n: 0, d: [] });
    a.n++;
    if (parent) { const p = posAt(parent.samples, s.t); if (p) a.d.push(Math.hypot(p.x - s.x, p.y - s.y)); }
  }
}
for (const [k, a] of Object.entries(agg)) { const d = a.d.sort((x,y)=>x-y); const q = (p) => d.length ? d[Math.floor(p*(d.length-1))].toFixed(1) : '-'; console.log(k, 'tirs', a.n, 'ecart au porteur (m) med', q(0.5), 'p90', q(0.9), 'max', q(1)); }
