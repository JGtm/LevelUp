import fs from 'fs';
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const files = fs.readdirSync(dir).filter(f => f.endsWith('.json') && !f.includes('derived'));
const agg = {};
let rows = [];
for (const f of files) {
  let doc; try { doc = JSON.parse(fs.readFileSync(dir + f, 'utf8')); } catch { continue; }
  if (!doc.vehicles || !doc.vehicles.length) continue;
  const cv = doc.coverage?.vehicles || {};
  const ridden = {};
  let rideFrames = 0, ridesFilm = 0, ridesProx = 0;
  for (const v of doc.vehicles) for (const r of (v.rides || [])) { ridden[v.family || '?'] = (ridden[v.family||'?']||0) + (r.t1 - r.t0); rideFrames += r.t1 - r.t0; if (r.src === 'film') ridesFilm++; else ridesProx++; }
  const vshots = doc.shots.filter(s => s.v !== undefined);
  const vshotsByW = {};
  for (const s of vshots) vshotsByW[s.w] = (vshotsByW[s.w]||0)+1;
  const vehWeap = vshots.filter(s => typeof s.w === 'string' && s.w.endsWith('00000000')).length;
  rows.push({ id: f.slice(0,8), schema: doc.schemaVersion, fams: [...new Set(doc.vehicles.map(v=>v.family||'?'))].join('/'), ridden: Object.entries(ridden).map(([k,v])=>`${k}:${(v/10).toFixed(0)}s`).join(' '), ridesFilm, ridesProx, vshots: vshots.length, vehWeap, noRide: cv.shotsNoRide, noSlot: doc.coverage?.shots?.noSlot, covShots: cv.shots, covVW: cv.shotsVehicleWeapon });
}
rows.sort((a,b)=> (b.noRide||0)-(a.noRide||0));
console.log('docs with vehicles:', rows.length, 'of', files.length);
console.log('id\tschema\tfamilies\tridden\tridesFilm\tridesProx\tvehShotsPublished\tvehWeaponShots\tshotsNoRide\tshotsNoSlot');
for (const r of rows) console.log([r.id, r.schema, r.fams, r.ridden, r.ridesFilm, r.ridesProx, r.vshots, r.vehWeap, r.noRide, r.noSlot].join('\t'));
