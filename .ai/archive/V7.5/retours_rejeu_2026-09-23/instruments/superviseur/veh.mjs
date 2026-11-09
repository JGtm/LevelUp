import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const fmt = (t) => { const s = t * doc.frameIntervalMs / 1000; return `${Math.floor(s/60)}:${String((s%60).toFixed(1)).padStart(4,'0')}`; };
console.log('originMs', doc.originMs, 't0FilmMs', doc.t0FilmMs, 'frameInterval', doc.frameIntervalMs, 'durationMs', doc.durationMs);
for (const v of doc.vehicles) {
  console.log(`veh slot=${v.slot} gen=${v.gen} chassis=${v.chassis} family=${v.family} t0=${v.t0}(${fmt(v.t0)}) t1=${v.t1}(${v.t1!=null?fmt(v.t1):''}) t1max=${v.t1max} end=${v.end} spawn=${JSON.stringify(v.spawn)} samples=${v.samples?.length} rides=${JSON.stringify(v.rides)?.slice(0,300)}`);
  if (v.samples) console.log('  first samples', JSON.stringify(v.samples.slice(0,6)), '... last', JSON.stringify(v.samples.slice(-3)));
}
const shotKeys = {};
for (const s of doc.shots) for (const k of Object.keys(s)) shotKeys[k] = (shotKeys[k]||0)+1;
console.log('shot keys', shotKeys);
const byW = {};
for (const s of doc.shots) { const k = s.w; byW[k] = (byW[k]||0)+1; }
console.log('shots by w', Object.entries(byW).sort((a,b)=>b[1]-a[1]).map(([k,n])=>`${k}:${n}${doc.weaponLabels[k]?' ['+(doc.weaponLabels[k].en||doc.weaponLabels[k].key||JSON.stringify(doc.weaponLabels[k]).slice(0,40))+']':''}`).join('\n'));
