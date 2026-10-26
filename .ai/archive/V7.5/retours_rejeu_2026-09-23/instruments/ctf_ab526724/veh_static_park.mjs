// Parc : vies de vehicule jamais deplacees (<= 1 echantillon ou deplacement nul) et jamais occupees,
// et leur distance a l'emprise publiee (doc.bounds) — pour mesurer une regle « decor hors emprise ».
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const out = [];
let tot = 0, staticN = 0;
const byFam = {};
for (const f of files) {
  let d; try { d = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const b = d.bounds; if (!b) continue;
  for (const v of d.vehicles || []) {
    tot++;
    const s = v.samples || [];
    let path = 0; for (let i = 1; i < s.length; i++) path += Math.hypot(s[i].x - s[i-1].x, s[i].y - s[i-1].y);
    const rides = (v.rides || []).length;
    const p = v.spawn || s[0]; if (!p) continue;
    const dx = Math.max(b.minX - p.x, 0, p.x - b.maxX), dy = Math.max(b.minY - p.y, 0, p.y - b.maxY);
    const outM = Math.hypot(dx, dy);
    const dz = p.z != null ? Math.max(b.minZ - p.z, 0, p.z - b.maxZ) : null;
    const isStatic = path < 0.5 && rides === 0;
    if (isStatic) staticN++;
    const k = `${v.family || '?'} static=${isStatic} ${outM > 3 ? 'HORS' : 'dans'}`;
    byFam[k] = (byFam[k] || 0) + 1;
    if (isStatic && v.family) out.push(`${f.slice(0,8)} slot=${v.slot} ${v.family} n=${s.length} path=${path.toFixed(1)} rides=${rides} horsEmpriseXY=${outM.toFixed(1)}m dz=${dz?.toFixed(1)}`);
  }
  d = null;
}
console.log('vies', tot, 'statiques(non deplacees, non occupees)', staticN);
console.log(Object.entries(byFam).sort((a,b)=>b[1]-a[1]).map(([k,n])=>`${n}\t${k}`).join('\n'));
console.log('--- vies statiques de famille resolue :');
console.log(out.join('\n'));
