import fs from 'fs';
const id = process.argv[2];
const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
const byT = (arr) => { const m = {}; for (const e of arr) (m[e.t] ||= []).push(e); return m; };
const L = byT(doc.loadouts), I = byT(doc.inventory);
console.log('loadouts by t:', Object.entries(L).map(([t,a])=>`${t}:${a.length}`).join(' '));
console.log('inventory by t:', Object.entries(I).map(([t,a])=>`${t}:${a.length}`).join(' '));
// all tracks with t0/t1 and xuid
const trk = doc.tracks.map(t => ({slot:t.slot, team:t.team, xuid:t.xuid, t0:t.points[0]?.[0] ?? t.points[0]?.t, n:t.points.length, endFrame:t.endFrame, p0: JSON.stringify(t.points[0])}));
console.log('track sample point', JSON.stringify(doc.tracks[0].points.slice(0,3)));
for (const t of trk) console.log(`slot ${t.slot} team ${t.team} xuid ${t.xuid} n=${t.n} endFrame=${t.endFrame} p0=${t.p0}`);
