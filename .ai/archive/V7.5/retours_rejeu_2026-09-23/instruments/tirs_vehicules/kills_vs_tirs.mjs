// Pour chaque frag de classe VEHICULE du parc (kill-feed, vue _latest) : le document publie-t-il
// un tir du tueur dans les 2 s qui precedent ? Ventile par tag de degat (-> vehicule).
import fs from 'fs';
const SP = process.argv[2];
const dir = 'C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite/';
const lignes = fs.readFileSync(SP + '/sondes/tirs_vehicules/kills_vehicules.tsv', 'utf8').trim().split('\n').slice(1).filter(l => !l.startsWith('('));
const NOM = { f712c64a: 'Ghost (veh_cv_ghost)', deffdc6b: 'Wraith (233c877d)', '7c4f8a28': 'Wraith (233c877d)', '382cafaf': 'Warthog LAAG (dd7f9102)',
  '674a7d69': 'Falcon lance-grenades/gauss (1a043c29)', a859230a: 'Falcon lance-grenades/gauss (1a043c29)', fa4fad21: 'Banshee (000026ed)', '53b1e6c5': 'Banshee (000026ed)', a21bf18a: 'Banshee (000026ed)',
  '5a4450e4': 'Rockethog (bcfb852f)', '00426796': 'Gungoose (000025aa)', '28907150': 'Pelican/Falcon (0000254b)', d2ffec3f: '? (20737e7f)', '77a61ef5': 'Falcon LMG (f4c45d71)',
  '3c7560e0': 'Wraith tourelle plasma (001b33fc)', '366522a6': 'Chopper/Banshee/Ghost partage', '19bd6810': 'Scorpion canon', '0bece71e': 'Scorpion canon', ab06203c: 'Wasp (b65b3b4a)', '3b3b3d40': 'Tourelle auto bannie/Wasp', '003f582d': 'Tourelle UNSC (003f00c7)' };
const parDoc = {};
for (const l of lignes) { const [m, t, x, g, tag] = l.split('\t'); (parDoc[m] ||= []).push({ t: +t, x, g, tag }); }
const agg = {};
for (const [m, kills] of Object.entries(parDoc)) {
  const f = fs.readdirSync(dir).find(n => n.startsWith(m) && n.endsWith('.json') && !n.includes('derived'));
  if (!f) continue;
  const doc = JSON.parse(fs.readFileSync(dir + f, 'utf8'));
  const labels = doc.weaponLabels || {};
  const slotsDe = {}; for (const tr of doc.tracks) (slotsDe[tr.xuid] ||= new Set()).add(tr.slot);
  const ridesDe = {}; for (const v of doc.vehicles || []) for (const r of v.rides || []) (ridesDe[r.xuid] ||= []).push({ ...r, fam: v.family || '?', ch: v.chassis });
  for (const k of kills) {
    const fr = Math.round((k.t - doc.originMs) / doc.frameIntervalMs);
    const slots = slotsDe[k.x] || new Set();
    const tirs = doc.shots.filter(s => slots.has(s.slot) && s.t >= fr - 20 && s.t <= fr + 2);
    const tirsVeh = tirs.filter(s => s.v !== undefined || !labels[s.w]);
    const ride = (ridesDe[k.x] || []).find(r => fr >= r.t0 - 5 && fr <= r.t1 + 5);
    const nom = NOM[k.tag] || k.tag;
    const a = (agg[nom] ||= { kills: 0, avecTirVeh: 0, avecTirQuelconque: 0, avecEpisode: 0, docs: new Set() });
    a.kills++; a.docs.add(m); if (tirsVeh.length) a.avecTirVeh++; if (tirs.length) a.avecTirQuelconque++; if (ride) a.avecEpisode++;
  }
}
console.log('vehicule/arme (tag de degat)\tfrags\tdocs\tfrag couvert par un episode du tueur\tfrag precede d un tir d ARME DE VEHICULE publie (2 s)\tfrag precede d un tir quelconque publie');
for (const [k, a] of Object.entries(agg).sort((x, y) => y[1].kills - x[1].kills)) console.log(`${k}\t${a.kills}\t${a.docs.size}\t${a.avecEpisode}\t${a.avecTirVeh}\t${a.avecTirQuelconque}`);
