// Episodes d'occupation poses sur une vie ENFANT (tourelle, 0 echantillon) : ecart entre la position
// DESSINEE (naissance de la tourelle) et le chassis porteur (slot+1/+2, meme fenetre), a mi-episode.
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const posAt = (samples, t) => { let i = samples.findIndex(s => s.t >= t); if (i === -1) i = samples.length - 1; if (i === 0) return samples[0]; const a = samples[i-1], b = samples[i]; if (b.t === a.t) return b; const f = Math.min(1, Math.max(0, (t - a.t)/(b.t - a.t))); return { x: a.x + (b.x-a.x)*f, y: a.y + (b.y-a.y)*f }; };
const agg = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const bySlot = {}; for (const v of doc.vehicles || []) (bySlot[v.slot] ||= []).push(v);
  for (const v of doc.vehicles || []) {
    if (v.samples?.length || !(v.rides||[]).length) continue;
    for (const r of v.rides) {
      const tm = Math.round((r.t0 + r.t1) / 2);
      let parent = null; for (const d of [1, 2]) { parent = (bySlot[v.slot + d] || []).find(o => o.samples?.length && tm >= o.t0 && tm <= (o.t1max ?? 1e9)); if (parent) break; }
      const k = v.chassis + ' -> ' + (parent ? parent.family + '/' + parent.chassis : 'aucun porteur');
      const a = (agg[k] ||= { ep: 0, sec: 0, d: [] });
      a.ep++; a.sec += (r.t1 - r.t0) / 10;
      if (parent && v.spawn) { const p = posAt(parent.samples, tm); a.d.push(Math.hypot(p.x - v.spawn.x, p.y - v.spawn.y)); }
    }
  }
}
for (const [k, a] of Object.entries(agg).sort((x,y)=>y[1].sec-x[1].sec)) { const d = a.d.sort((x,y)=>x-y); const q = (p) => d.length ? d[Math.floor(p*(d.length-1))].toFixed(1) : '-'; console.log(k, 'episodes', a.ep, 'duree', a.sec.toFixed(0)+'s', 'ecart dessin/porteur med', q(0.5), 'p90', q(0.9)); }
