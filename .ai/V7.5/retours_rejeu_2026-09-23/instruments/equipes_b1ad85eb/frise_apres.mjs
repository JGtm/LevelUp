// frise_apres.mjs — la regle PROPOSEE (regle_apres.mjs) sur UN document : places et occupants, puis
// les instants demandes. Usage : node frise_apres.mjs <id8> [affiche_s ...]
import { loadDoc, scoreboard, participants, registry, buildPlayers, displayToFrame, frameToDisplay } from './panneaux_lib.mjs';
import { groupes, panneaux } from './regle_apres.mjs';

const id8 = process.argv[2];
const doc = loadDoc(id8);
const reg = registry(id8);
const st = Number(reg.st);
const toF = (ms) => (Number(ms) - st - doc.originMs) / doc.frameIntervalMs;
const g = groupes(doc, buildPlayers(doc, scoreboard(id8)), participants(id8), { toF });
for (const e of g) {
  console.log(`equipe ${e.key}`);
  e.places.forEach((pl, i) => console.log(`  place ${i} : ${pl.map((o) => `${o.nom}[${o.debut.toFixed(0)}..${Number.isFinite(o.fin) ? o.fin.toFixed(0) : 'fin'}]${o.source === 'api' ? '(equipe API)' : ''}`).join(' -> ')}`));
}
const instants = process.argv.slice(3).map(Number);
for (const s of instants.length ? instants : [0, 134, 384]) {
  const f = displayToFrame(doc, s);
  console.log(`${frameToDisplay(doc, f)} (f${f}) : ${panneaux(g, f).map((c) => `${c.key} ${c.tuiles.length} [${c.tuiles.map((t) => t.name).join(', ')}]`).join('   ')}`);
}
