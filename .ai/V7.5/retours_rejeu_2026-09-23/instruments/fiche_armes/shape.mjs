import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const desc = (v) => Array.isArray(v) ? `array[${v.length}]` + (v.length && typeof v[0]==='object' ? ' keys=' + JSON.stringify(Object.keys(v[0])) : '') : (v && typeof v === 'object' ? 'object keys=' + JSON.stringify(Object.keys(v)).slice(0,400) : JSON.stringify(v)?.slice(0,120));
for (const [k,v] of Object.entries(doc)) console.log(k.padEnd(28), desc(v));
