// Jointure TSV de l instrument C2 x document publie : par vie ti=40, champs de l etat par defaut
// (bloc MPP d524, r19, famille) contre le comportement publie (echantillons, occupants, fin).
import fs from 'fs';
const DOCS = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const films = process.argv.slice(2);
const tally = {}; const lignes = [];
for (const id of films) {
  const L = fs.readFileSync(`${id}.tsv`, 'utf8').trim().split('\n').map(l => l.split('\t'));
  const kfTs = [...new Set(L.filter(r => r[0] === 'KF').map(r => +r[3]))].sort((a, b) => a - b);
  const t0 = kfTs[0];
  const doc = JSON.parse(fs.readFileSync(DOCS + id + '.json', 'utf8'));
  const originUS = t0 + doc.originMs * 1000;
  const vies = {};
  for (const r of L) {
    const k = r[1] + '/' + r[2]; const v = (vies[k] ??= { f: {}, kf: new Set(), dl: 0, dlApres: 0 });
    if (r[0] === 'KF') { v.kf.add(+r[3]); if (!(r[4] in v.f)) v.f[r[4]] = r[5]; }
    else { v.dl++; if (+r[3] > originUS) v.dlApres++; }
  }
  for (const [k, v] of Object.entries(vies)) {
    const [slot, gen] = k.split('/').map(Number);
    const d = (doc.vehicles || []).find(x => x.slot === slot && (x.gen ?? gen) === gen);
    const d524 = v.f['mpp.d524'] || '?';
    const gate = d524.startsWith('14:') ? 1 : 0;
    const premier = Math.min(...v.kf); const avantOrigine = premier < originUS;
    const n = d ? (d.samples || []).length : -1; const rides = d ? (d.rides || []).length : -1;
    const decorL13 = d && n === 1 && d.samples[0].t === d.t0 && d.end === 'film_end' && rides === 0;
    const fam = d ? (d.family || '?') : 'non-publie';
    const cle = `d524=${gate} ${d ? (decorL13 ? 'regleL13=decor' : 'regleL13=non') : 'non-publie'}`;
    tally[cle] = (tally[cle] || 0) + 1;
    lignes.push(`${id} ${k} w32=${(v.f['mpp.w32'] || '').slice(3)} fam=${fam} d524=${d524} r19=${v.f['ds.r19']} bv14=${(v.f['ds.bv14']||'').slice(0,2)} kf=${v.kf.size} avantOrigine=${avantOrigine} dl=${v.dl} dlApresOrigine=${v.dlApres} doc:n=${n} rides=${rides} end=${d ? d.end : '-'} L13=${decorL13 ? 'DECOR' : '-'}`);
  }
}
console.log(lignes.join('\n'));
console.log(JSON.stringify(tally, null, 1));
