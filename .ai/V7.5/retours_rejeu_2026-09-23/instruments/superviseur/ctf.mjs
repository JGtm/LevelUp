import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const keys = Object.keys(doc);
console.log('top keys', keys.join(','));
for (const k of ['objectives','flagCarries','objectiveObjects','zoneStates','flagReturnZone','scoreTimeline','neutralDeaths','rounds','victory']) {
  const v = doc[k];
  if (v === undefined) { console.log(k, 'ABSENT'); continue; }
  console.log(k, Array.isArray(v) ? `array[${v.length}] ${JSON.stringify(v.slice(0,3)).slice(0,600)}` : JSON.stringify(v).slice(0, 800));
}
for (const k of ['score','flagCarries','objectives','objectiveObjects','zones','vehicles','teams','seats']) console.log('coverage.'+k, JSON.stringify(doc.coverage[k]));
console.log('layers', JSON.stringify(doc.layers).slice(0, 2000));
console.log('vehicles', JSON.stringify(doc.vehicles.map(v=>({slot:v.slot,family:v.family,chassis:v.chassis,t0:v.t0,t1:v.t1,end:v.end,spawn:v.spawn,n:v.samples.length,rides:v.rides?.length}))));
