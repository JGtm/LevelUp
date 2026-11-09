import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
for (const t of doc.tracks) if ([534,536,539,530,519].includes(t.slot)) console.log(t.slot, JSON.stringify(Object.keys(t)), 'startFrame', t.startFrame, 'endFrame', t.endFrame, 'p0', JSON.stringify(t.points[0]), 'pN', JSON.stringify(t.points[t.points.length-1]));
