import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const fmt = (t) => { const s = t * doc.frameIntervalMs / 1000; return `${Math.floor(s/60)}:${(s%60).toFixed(1).padStart(4,'0')}`; };
console.log('roster', JSON.stringify(doc.roster));
console.log('coverage.shots', JSON.stringify(doc.coverage.shots));
console.log('coverage.vehicles', JSON.stringify(doc.coverage.vehicles));
console.log('coverage.seats', JSON.stringify(doc.coverage.seats));
console.log('coverage.teams', JSON.stringify(doc.coverage.teams));
for (const v of doc.vehicles) for (const r of (v.rides||[])) {
  const inRide = doc.shots.filter(s => s.slot === r.slot && s.t >= r.t0 && s.t <= r.t1);
  const byW = {}; for (const s of inRide) byW[s.w]=(byW[s.w]||0)+1;
  console.log(`veh ${v.slot} ${v.family} ride slot=${r.slot} xuid=${r.xuid} ${fmt(r.t0)}-${fmt(r.t1)} src=${r.src} seat=${r.seat} -> shots by rider in ride: ${inRide.length} ${JSON.stringify(byW)}`);
}
// tracks of slot 514/541
for (const tr of doc.tracks.filter(t => [514,541,539].includes(t.slot))) console.log('track', tr.slot, tr.team, tr.xuid, 'n', tr.points.length, 'from', fmt(tr.points[0].t), 'to', fmt(tr.points[tr.points.length-1].t), 'endFrame', tr.endFrame);
