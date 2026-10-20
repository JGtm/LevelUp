import fs from 'fs';
const id = process.argv[2]; const slot = +process.argv[3];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const tr = doc.tracks.find(t => t.slot === slot);
console.log('track', slot, 'start', tr.startFrame, 'end', tr.endFrame);
for (const k of ['loadouts','inventory','weaponChanges','pickups','padPickups']) for (const e of (doc[k]||[])) if (e.slot === slot || (k==='padPickups' && e.xuid === tr.xuid)) console.log(k, JSON.stringify(e));
const lab = (w) => { const L = doc.weaponLabels['0x'+w.toUpperCase()] || doc.weaponLabels[w]; return L ? L.en : w; };
const sh = doc.shots.filter(s => s.slot === slot); const byW = {}; for (const s of sh) byW[s.w] = (byW[s.w]||[]).concat(s.t);
for (const [w, ts] of Object.entries(byW)) console.log('tirs', w, doc.weaponLabels[w]?.en, 'n=', ts.length, 'premier', Math.min(...ts), 'dernier', Math.max(...ts));
