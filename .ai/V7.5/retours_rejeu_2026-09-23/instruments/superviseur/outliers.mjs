import fs from 'fs';
const ids = process.argv.slice(2);
for (const id of ids) {
  const doc = JSON.parse(fs.readFileSync(`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/${id}.json`, 'utf8'));
  const b = doc.bounds;
  console.log(`== ${id} bounds x[${b.minX},${b.maxX}] y[${b.minY},${b.maxY}] z[${b.minZ},${b.maxZ}] vehicles=${doc.vehicles?.length}`);
  for (const v of (doc.vehicles||[])) {
    const out = v.samples.filter(s => s.x < b.minX - 5 || s.x > b.maxX + 5 || s.y < b.minY - 5 || s.y > b.maxY + 5 || s.z < b.minZ - 10 || s.z > b.maxZ + 10);
    if (out.length) console.log(`  veh ${v.slot} ${v.family} gen=${v.gen}: ${out.length} out-of-bounds samples of ${v.samples.length}: ${out.slice(0,6).map(s=>`t=${s.t}(${s.x},${s.y},${s.z})`).join(' ')}`);
  }
  // tracks: out-of-bounds points
  let nt = 0;
  for (const tr of doc.tracks) {
    const out = tr.points.filter(s => s.x < b.minX - 5 || s.x > b.maxX + 5 || s.y < b.minY - 5 || s.y > b.maxY + 5 || s.z < b.minZ - 10 || s.z > b.maxZ + 10);
    if (out.length) { nt++; if (nt < 12) console.log(`  track slot=${tr.slot} team=${tr.team} xuid=${tr.xuid}: ${out.length}/${tr.points.length} OOB: ${out.slice(0,4).map(s=>`t=${s.t}(${s.x},${s.y},${s.z})`).join(' ')}`); }
  }
  console.log(`  tracks with OOB points: ${nt}`);
}
