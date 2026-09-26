// Les vies de chassis ENFANTS (tourelles) : naissance, echantillons, parent (slot+1), episodes.
import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const bySlot = {}; for (const v of doc.vehicles) (bySlot[v.slot] ||= []).push(v);
for (const v of doc.vehicles) {
  if (v.family) continue;
  const parent = (bySlot[v.slot + 1] || []).find(o => o.t0 <= v.t1 && v.t0 <= (o.t1 ?? 1e9));
  const parent2 = (bySlot[v.slot + 2] || []).find(o => o.t0 <= v.t1 && v.t0 <= (o.t1 ?? 1e9));
  console.log(`${v.slot}/${v.chassis} t=${v.t0}-${v.t1} spawn=${JSON.stringify(v.spawn)} ech=${v.samples?.length||0} episodes=${(v.rides||[]).length} | +1=${parent ? parent.family+'/'+parent.chassis+' ech='+parent.samples.length : '-'} +2=${parent2 ? parent2.family+'/'+parent2.chassis : '-'}`);
}
