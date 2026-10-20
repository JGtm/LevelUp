import fs from 'fs';
for (const id of process.argv.slice(2)) {
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
console.log('==', id, 'originMs', doc.originMs, 't0FilmMs', doc.t0FilmMs, 'frames', doc.frameCount, 'durationMs', doc.durationMs);
console.log('identity.coverage', JSON.stringify(doc.identity?.coverage));
console.log('identity.statborgSlots', JSON.stringify(doc.identity?.statborgSlots)?.slice(0, 1500));
console.log('coverage.originResolved', JSON.stringify(doc.coverage.originResolved), 't0Film', JSON.stringify(doc.coverage.t0Film));
console.log('coverage.score', JSON.stringify(doc.coverage.score));
console.log('coverage.objectives', JSON.stringify(doc.coverage.objectives));
console.log('coverage.verdict', JSON.stringify(doc.coverage.verdict));
console.log('roster', JSON.stringify(doc.roster));
}
