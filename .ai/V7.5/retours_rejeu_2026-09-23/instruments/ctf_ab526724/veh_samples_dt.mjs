import fs from 'fs';
const [id, ...slots] = process.argv.slice(2);
const d = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
for (const v of d.vehicles.filter(v => slots.includes(String(v.slot)))) {
  const ts = (v.samples||[]).map(s => s.t);
  const dts = ts.slice(1).map((t,i)=>t-ts[i]);
  dts.sort((a,b)=>a-b);
  console.log(id, v.slot, v.family, 't0', v.t0, 't1', v.t1, 't1max', v.t1max, 'n', ts.length, 'first', ts.slice(0,5).join(','), 'last', ts.slice(-3).join(','), 'dt median', dts[Math.floor(dts.length/2)], 'dt max', dts[dts.length-1], 'spawn', JSON.stringify(v.spawn));
}
