// Tirs d'arme PERSONNELLE poses sur un vehicule (champ v) : par famille, et selon que le roster
// depasse 16 index (repliement du FilmIndex sur 4 bits) ou non.
import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const agg = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const big = (doc.roster||[]).some(r => r.filmIndex >= 16);
  const fam = {}; for (const v of doc.vehicles || []) fam[v.slot] = v.family || ('?' + v.chassis);
  for (const s of doc.shots) {
    if (s.v === undefined) continue;
    const perso = (doc.weaponLabels||{})[s.w] !== undefined;
    if (!perso) continue;
    const k = fam[s.v] + (big ? ' [roster>16]' : ' [roster<=16]');
    const a = (agg[k] ||= { n: 0, docs: new Set() }); a.n++; a.docs.add(f.slice(0,8));
  }
}
for (const [k, a] of Object.entries(agg).sort((x,y)=>y[1].n-x[1].n)) console.log(k, a.n, [...a.docs].join(','));
