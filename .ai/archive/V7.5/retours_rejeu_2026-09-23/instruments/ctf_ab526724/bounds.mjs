import fs from 'fs';
for (const id of process.argv.slice(2)) {
  const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
  let mn = {x:1e9,y:1e9,z:1e9}, mx = {x:-1e9,y:-1e9,z:-1e9}; let n=0;
  const zs = [];
  for (const tr of doc.tracks) for (const p of tr.points) { n++; for (const k of ['x','y','z']) { if (p[k] == null) continue; mn[k]=Math.min(mn[k],p[k]); mx[k]=Math.max(mx[k],p[k]); } if (p.z!=null) zs.push(p.z); }
  zs.sort((a,b)=>a-b);
  const q = (f) => zs[Math.floor(f*(zs.length-1))];
  console.log(id, 'bounds', JSON.stringify(doc.bounds));
  console.log(' tracks points', n, 'x', mn.x.toFixed(2), mx.x.toFixed(2), 'y', mn.y.toFixed(2), mx.y.toFixed(2), 'z', mn.z.toFixed(2), mx.z.toFixed(2), 'z q01/q50/q99', q(0.01)?.toFixed(2), q(0.5)?.toFixed(2), q(0.99)?.toFixed(2));
  const pads = (doc.weaponPads||[]).map(p=>`(${p.x.toFixed(1)},${p.y.toFixed(1)},${p.z.toFixed(1)} ${p.weapon})`).join(' ');
  console.log(' weaponPads', pads);
  // points near vehicle area (y < -120)
  let near = 0; for (const tr of doc.tracks) for (const p of tr.points) if (p.y < -118) near++;
  console.log(' track points with y < -118 :', near);
}
