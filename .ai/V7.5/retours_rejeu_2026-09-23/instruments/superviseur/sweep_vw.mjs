import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const byTag = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const famOfSlot = {}; for (const v of (doc.vehicles||[])) famOfSlot[v.slot] = v.family || '?';
  for (const s of doc.shots) {
    if (s.v === undefined) continue;
    const vw = typeof s.w === 'string' && s.w.endsWith('00000000');
    const key = (vw ? s.w : 'playerWeapon') + ' @' + famOfSlot[s.v];
    byTag[key] = byTag[key] || { n: 0, docs: new Set(), withH: 0 };
    byTag[key].n++; byTag[key].docs.add(f.slice(0,8)); if (s.h !== undefined) byTag[key].withH++;
  }
}
for (const [k, v] of Object.entries(byTag).sort((a,b)=>b[1].n-a[1].n)) console.log(k, 'n=', v.n, 'withHeading=', v.withH, 'docs=', [...v.docs].join(','));
