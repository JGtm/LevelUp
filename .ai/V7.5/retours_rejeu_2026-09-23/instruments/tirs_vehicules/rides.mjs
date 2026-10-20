// Episodes d'un document : vehicule, occupant (filmIndex), bornes, tirs publies pendant l'episode.
import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const idx = {}; for (const r of doc.roster) idx[r.xuid] = r.filmIndex + ':' + r.name;
console.log(id, 'roster', doc.roster.length, 'origin', doc.originMs, 'frames', doc.frameCount, 'cov.shots', JSON.stringify(doc.coverage.shots));
const cv = doc.coverage.vehicles; console.log('cov.veh shots', cv.shots, 'noRide', cv.shotsNoRide, 'amb', cv.shotsAmbiguous, 'unplaced', cv.shotsUnplaced, 'vw', cv.shotsVehicleWeapon, 'rides', cv.rides, 'read', cv.ridesRead, 'prox', cv.ridesProximity);
for (const v of doc.vehicles) for (const r of (v.rides||[])) {
  const sh = doc.shots.filter(s => s.slot === r.slot && s.t >= r.t0 && s.t <= r.t1);
  const byW = {}; for (const s of sh) byW[s.w + (s.v!==undefined?'+v':'')] = (byW[s.w + (s.v!==undefined?'+v':'')]||0)+1;
  console.log(`  veh ${v.slot} ${v.family||'?'}/${v.chassis} | occ slot ${r.slot} ${idx[r.xuid]||r.xuid} | t ${r.t0}-${r.t1} (${((r.t1-r.t0)/10).toFixed(1)}s) src=${r.src} seat=${r.seat} | tirs ${sh.length} ${JSON.stringify(byW)}`);
}
