import fs from 'fs';
const ids = process.argv.slice(2);
for (const id of ids) {
  const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
  console.log('==', id, 'schema', doc.schemaVersion, 'frames', doc.frameCount, 'originMs', doc.originMs, 't0FilmMs', doc.t0FilmMs);
  console.log('coverage keys', Object.keys(doc.coverage).join(','));
  for (const k of ['bridge','deaths','verdict','score','flagCarries','objectives','objectiveObjects','teams','identity','killFeed','chronicle','origin','neutralDeaths']) {
    if (doc.coverage[k] !== undefined) console.log('coverage.'+k, JSON.stringify(doc.coverage[k]).slice(0, 1500));
  }
}
