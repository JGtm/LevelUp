import fs from 'fs';
const ids = process.argv.slice(2);
for (const id of ids) {
  const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
  console.log('==', id);
  for (const k of ['decoder','fallbacks','deathsPaths','filmMajorVersion','originResolved','t0Film','tracks','seats','shots']) {
    if (doc.coverage[k] !== undefined) console.log('coverage.'+k, JSON.stringify(doc.coverage[k]).slice(0, 2500));
  }
}
