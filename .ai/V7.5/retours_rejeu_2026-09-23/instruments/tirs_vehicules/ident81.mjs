import fs from 'fs';
const id = process.argv[2] || '81c02726';
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const idn = doc.identity;
console.log('identity keys', Object.keys(idn));
console.log('coverage', JSON.stringify(idn.coverage));
console.log('players sample', JSON.stringify(idn.players?.slice?.(0,3) ?? idn.players).slice(0,600));
const bs = idn.bipedSlots;
console.log('bipedSlots type', Array.isArray(bs) ? 'array '+bs.length : typeof bs);
const arr = Array.isArray(bs) ? bs : Object.entries(bs).map(([k,v])=>({k,...v}));
console.log('bipedSlots sample', JSON.stringify(arr.slice(0,3)).slice(0,800));
for (const s of arr) { const j = JSON.stringify(s); if (j.includes('514') || j.includes('541')) console.log('  match', j.slice(0,600)); }
console.log('coverage.bridge', JSON.stringify(doc.coverage.bridge));
console.log('tracks of GMONEY:');
for (const tr of doc.tracks) if (tr.xuid === '2533274810173379') console.log('  slot', tr.slot, 'team', tr.team, 'n', tr.points.length, 'first t', tr.points[0].t, 'last t', tr.points[tr.points.length-1].t, 'endFrame', tr.endFrame);
