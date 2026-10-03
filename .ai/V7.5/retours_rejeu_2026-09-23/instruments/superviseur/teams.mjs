import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const fmt = (t) => { const s = t * doc.frameIntervalMs / 1000; return `${Math.floor(s/60)}:${(s%60).toFixed(1).padStart(4,'0')}`; };
console.log('originMs', doc.originMs, 't0FilmMs', doc.t0FilmMs, 'frames', doc.frameCount);
console.log('roster', JSON.stringify(doc.roster, null, 0));
console.log('coverage.teams', JSON.stringify(doc.coverage.teams));
console.log('coverage.seats', JSON.stringify(doc.coverage.seats));
console.log('coverage.tracks', JSON.stringify(doc.coverage.tracks));
console.log('identity.coverage', JSON.stringify(doc.identity?.coverage));
console.log('identity.players', JSON.stringify(doc.identity?.players)?.slice(0, 3000));
console.log('identity.bipedSlots', JSON.stringify(doc.identity?.bipedSlots)?.slice(0, 3000));
const bySlot = {};
for (const tr of doc.tracks) {
  const p0 = tr.points[0], p1 = tr.points[tr.points.length-1];
  console.log(`track slot=${tr.slot} team=${tr.team} xuid=${tr.xuid ?? '-'} n=${tr.points.length} ${fmt(p0.t)}..${fmt(p1.t)} endFrame=${tr.endFrame ?? ''}`);
}
