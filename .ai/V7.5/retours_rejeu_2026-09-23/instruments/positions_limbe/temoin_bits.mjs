// temoin_bits.mjs — TEMOIN du partage de motifs de bits. Les echantillons ABERRANTS de documents
// differents partagent >= 24 bits contigus a la MEME position (bits_out.txt). Est-ce le cas
// d'echantillons NORMAUX tires au hasard dans les memes documents ? Si oui, le partage ne prouve
// rien (effet de la quantification) ; si non, il est propre aux aberrants.
import fs from 'fs';
import path from 'path';
const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration';
const DIR = `${REPO}/data/cache/replays/halo_infinite`;
const HERE = path.dirname(new URL(import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
const cat = JSON.parse(fs.readFileSync(`${REPO}/data/titles/halo_infinite/reference/map_quant_bounds.json`, 'utf8'));
const norm = (s) => (s || '').toLowerCase().split(/\s+/).filter(Boolean).join(' ').replace(/ - ranked$/, '').replace(/ heavies$/, '');
const maps = {};
for (const line of fs.readFileSync(path.join(HERE, 'maps.tsv'), 'utf8').split('\n').slice(1)) {
  const [id8, map] = line.split('\t');
  if (id8) maps[id8] = map;
}
// memes documents que les aberrants
const aberr = fs.readFileSync(path.join(HERE, 'bits_out.txt'), 'utf8').split('\n').filter((l) => /^[0-9a-f]{8} /.test(l));
const docs = [...new Set(aberr.map((l) => l.slice(0, 8)))];
let seed = 777;
const rnd = () => { seed = (seed * 1103515245 + 12345) & 0x7fffffff; return seed / 0x7fffffff; };
function bitsOf(e, p) {
  const v = [p.x, p.y, p.z];
  return v.map((val, ax) => {
    const w = e.axisWidths[ax];
    const step = (e.max[ax] - e.min[ax]) / 2 ** w;
    const q = Math.max(0, Math.min(2 ** w - 1, Math.round((val - e.min[ax]) / step - 0.5)));
    return q.toString(2).padStart(w, '0');
  }).join('');
}
const samples = [];
for (const id of docs) {
  const doc = JSON.parse(fs.readFileSync(`${DIR}/${id}.json`, 'utf8'));
  const e = cat.maps[norm(maps[id])];
  const pool = [];
  for (const tr of doc.tracks) for (const p of tr.points) pool.push(p);
  for (const v of doc.vehicles || []) for (const s of (v.samples || [])) pool.push(s);
  // echantillons normaux : 3 par document, dans les bornes du document
  const B = doc.bounds;
  let n = 0, guard = 0;
  while (n < 10 && guard++ < 1000) {
    const p = pool[Math.floor(rnd() * pool.length)];
    if (p.x < B.minX || p.x > B.maxX || p.y < B.minY || p.y > B.maxY || p.z < B.minZ || p.z > B.maxZ) continue;
    samples.push({ id, flat: bitsOf(e, p) });
    n++;
  }
}
// meme mesure que bits.mjs : plus longue sous-chaine commune a la MEME position (offset egal)
function lcsSameOffset(a, b) {
  let best = 0, run = 0;
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) { if (a[i] === b[i]) { run++; if (run > best) best = run; } else run = 0; }
  return best;
}
let pairs = 0, hits = 0;
for (let i = 0; i < samples.length; i++) for (let j = i + 1; j < samples.length; j++) {
  if (samples[i].id === samples[j].id) continue;
  pairs++;
  if (lcsSameOffset(samples[i].flat, samples[j].flat) >= 24) hits++;
}
console.log(`TEMOIN normaux : ${samples.length} echantillons, ${pairs} paires inter-documents, ${hits} paires partageant >= 24 bits a la meme position`);
// meme mesure sur les aberrants (offset egal)
const ab = aberr.map((l) => ({ id: l.slice(0, 8), flat: l.split(/\s+/).find((x) => /^[01|]+$/.test(x) && x.includes('|')).replace(/\|/g, '') }));
let pa = 0, ha = 0;
const touched = new Set();
for (let i = 0; i < ab.length; i++) for (let j = i + 1; j < ab.length; j++) {
  if (ab[i].id === ab[j].id) continue;
  pa++;
  if (lcsSameOffset(ab[i].flat, ab[j].flat) >= 24) { ha++; touched.add(i); touched.add(j); }
}
console.log(`ABERRANTS : ${ab.length} echantillons, ${pa} paires inter-documents, ${ha} paires partageant >= 24 bits a la meme position ; echantillons concernes ${touched.size}/${ab.length}`);
