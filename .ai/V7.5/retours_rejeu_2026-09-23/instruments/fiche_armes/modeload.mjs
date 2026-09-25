// Regularite du « loadout de depart du mode » : la 1re lecture d'une vie (sans prise/ramassage avant)
// egale-t-elle le loadout modal des 1res lectures du match ? (mesure pour la question produit (c))
import fs from 'fs';
const DIR = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const modes = new Map(fs.readFileSync(process.argv[2], 'utf8').trim().split('\n').slice(1).map(l => l.split('\t')));
const random = (p) => /\b(Castle Wars|Fiesta|Husky Raid)\b/.test(p || '');
let R = { fixed: { lives:0, eq:0 }, random: { lives:0, eq:0 } }; const perMode = {};
for (const [id, pair] of modes) {
  let doc; try { doc = JSON.parse(fs.readFileSync(DIR + id + '.json', 'utf8')); } catch { continue; }
  const lo = doc.loadouts || [];
  const firsts = [];
  for (const t of doc.tracks) {
    const s = t.startFrame ?? 0, e = t.endFrame ?? t.points[t.points.length-1].t;
    const mine = lo.filter(l => l.slot === t.slot && l.t >= s && l.t <= e).sort((a,b)=>a.t-b.t);
    if (!mine.length) continue;
    const f = mine[0];
    const touched = (doc.weaponChanges||[]).some(c => c.slot === t.slot && c.t >= s && c.t <= f.t) || (doc.pickups||[]).some(p => p.slot === t.slot && p.kind === 'weapon' && p.t >= s && p.t <= f.t);
    if (touched) continue;
    firsts.push([...f.w].sort().join('+'));
  }
  if (!firsts.length) continue;
  const cnt = {}; for (const k of firsts) cnt[k] = (cnt[k]||0)+1;
  const [modal, n] = Object.entries(cnt).sort((a,b)=>b[1]-a[1])[0];
  const r = random(pair) ? R.random : R.fixed; r.lives += firsts.length; r.eq += n;
  const m = (pair||'?').replace(/ on .*/, ''); perMode[m] ||= { lives:0, eq:0, modal:new Set() }; perMode[m].lives += firsts.length; perMode[m].eq += n; perMode[m].modal.add(modal.split('+').map(w => doc.weaponLabels[w]?.en ?? w).join('+'));
}
console.log(`modes a depart FIXE : ${R.fixed.eq}/${R.fixed.lives} = ${(100*R.fixed.eq/R.fixed.lives).toFixed(1)} % des 1res lectures (sans prise avant) egales au loadout modal du match`);
console.log(`modes a depart ALEATOIRE : ${R.random.eq}/${R.random.lives} = ${(100*R.random.eq/R.random.lives).toFixed(1)} %`);
for (const [m, v] of Object.entries(perMode).sort((a,b)=>b[1].lives-a[1].lives)) console.log(`  ${m.padEnd(26)} ${v.eq}/${v.lives} = ${(100*v.eq/v.lives).toFixed(1)} %  modal: ${[...v.modal].join(' | ')}`);
