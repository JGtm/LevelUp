import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const chassisFam = {};
const tagChassis = {};
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  const bySlot = {}; for (const v of (doc.vehicles||[])) { bySlot[v.slot] = v; const k = v.chassis||'none'; chassisFam[k] = chassisFam[k] || {fam: new Set(), docs: new Set(), n:0}; chassisFam[k].fam.add(v.family||'?'); chassisFam[k].docs.add(f.slice(0,8)); chassisFam[k].n++; }
  for (const s of doc.shots) { if (s.v === undefined) continue; if (!(typeof s.w === 'string' && s.w.endsWith('00000000'))) continue; const v = bySlot[s.v]; const k = s.w + ' chassis=' + (v?.chassis||'none') + ' fam=' + (v?.family||'?'); tagChassis[k] = (tagChassis[k]||0)+1; }
}
console.log('chassis -> families');
for (const [k, v] of Object.entries(chassisFam).sort((a,b)=>b[1].n-a[1].n)) console.log(' ', k, [...v.fam].join('/'), 'lives=', v.n, 'docs=', v.docs.size);
console.log('vehicle-weapon tag -> chassis');
for (const [k, n] of Object.entries(tagChassis).sort((a,b)=>b[1]-a[1])) console.log(' ', k, n);
