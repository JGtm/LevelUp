// bits.mjs — relit sweep_out.txt et imprime, pour chaque echantillon aberrant, la CHAINE DE BITS
// X|Y|Z aux largeurs de la carte (catalogue). But : voir si les « positions » aberrantes sont un
// MEME motif de bits lu sur des cartes/axes differents (donc pas une coordonnee monde).
import fs from 'fs';
import path from 'path';
const REPO = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration';
const HERE = path.dirname(new URL(import.meta.url).pathname).replace(/^\/([A-Za-z]:)/, '$1');
const cat = JSON.parse(fs.readFileSync(`${REPO}/data/titles/halo_infinite/reference/map_quant_bounds.json`, 'utf8'));
const norm = (s) => (s || '').toLowerCase().split(/\s+/).filter(Boolean).join(' ').replace(/ - ranked$/, '').replace(/ heavies$/, '');
const lines = fs.readFileSync(path.join(HERE, 'sweep_out.txt'), 'utf8').split('\n');
const seen = new Set();
const out = [];
for (const l of lines) {
  const f = l.split('\t');
  if (f.length < 9 || !/^(track|veh)-/.test(f[2])) continue;
  if (f[2] === 'veh-oob') continue; // doublons des excursions / chutes reelles
  const m = f[8].match(/^q=(\d+)\/(\d+)\/(\d+)$/);
  if (!m) continue;
  const e = cat.maps[norm(f[1])];
  const q = [+m[1], +m[2], +m[3]];
  const key = f[0] + f[3] + f[5];
  if (seen.has(key)) continue;
  seen.add(key);
  const b = q.map((v, i) => v.toString(2).padStart(e.axisWidths[i], '0'));
  out.push({ id: f[0], map: f[1], kind: f[2], slot: f[3], t: f[5], pos: f[7], w: e.axisWidths.join('/'), bits: b.join('|'), flat: b.join('') });
}
for (const o of out) console.log(`${o.id} ${o.map.padEnd(18)} ${o.kind.padEnd(12)} ${o.slot} ${o.t.padEnd(7)} w=${o.w} ${o.bits}   ${o.pos}`);
// plus longue sous-chaine commune (>= 16 bits) entre paires de cartes DIFFERENTES de largeurs
function lcs(a, b) {
  let best = '', dp = new Array(b.length + 1).fill(0);
  for (let i = 1; i <= a.length; i++) {
    let prev = 0;
    for (let j = 1; j <= b.length; j++) {
      const tmp = dp[j];
      dp[j] = a[i - 1] === b[j - 1] ? prev + 1 : 0;
      if (dp[j] > best.length) best = a.slice(i - dp[j], i);
      prev = tmp;
    }
  }
  return best;
}
console.log('\n# sous-chaines communes >= 24 bits entre echantillons de DOCUMENTS differents');
const pairs = [];
for (let i = 0; i < out.length; i++) for (let j = i + 1; j < out.length; j++) {
  if (out[i].id === out[j].id) continue;
  const s = lcs(out[i].flat, out[j].flat);
  if (s.length >= 24) pairs.push([s.length, `${out[i].id}/${out[i].map}/${out[i].slot}@${out[i].t}(w${out[i].w}) ~ ${out[j].id}/${out[j].map}/${out[j].slot}@${out[j].t}(w${out[j].w}) : ${s} @${out[i].flat.indexOf(s)}/${out[j].flat.indexOf(s)}`]);
}
pairs.sort((a, b) => b[0] - a[0]);
for (const p of pairs.slice(0, 80)) console.log(p[0], p[1]);
console.log('paires >= 24 bits :', pairs.length, 'sur', out.length * (out.length - 1) / 2);
