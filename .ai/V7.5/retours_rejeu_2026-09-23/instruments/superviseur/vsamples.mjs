import fs from 'fs';
const [id, slotArg, fromArg, toArg] = process.argv.slice(2);
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const from = +fromArg || 0, to = +toArg || 1e9;
for (const v of doc.vehicles.filter(v => String(v.slot) === slotArg)) {
  console.log(`veh ${v.slot} ${v.family} t0=${v.t0} t1=${v.t1} end=${v.end} samples=${v.samples.length}`);
  let prev = null;
  for (const s of v.samples) {
    if (s.t < from || s.t > to) { prev = s; continue; }
    const d = prev ? Math.hypot(s.x - prev.x, s.y - prev.y) : 0;
    console.log(`  t=${s.t} (${(s.t/10).toFixed(1)}s) x=${s.x} y=${s.y} z=${s.z} h=${s.h ?? ''} step=${d.toFixed(2)}m dt=${prev? s.t-prev.t:''}`);
    prev = s;
  }
}
