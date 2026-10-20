// repro_b1ad85eb.mjs — ce que les deux panneaux d'equipe affichent a 0:00, 2:14 et 6:24.
import {
  loadDoc, scoreboard, buildPlayers, buildSeats, groupSeatsByTeam, panneaux,
  displayToFrame, frameToDisplay, trackWindow, playerLabel,
} from './panneaux_lib.mjs';

const id8 = process.argv[2] ?? 'b1ad85eb';
const doc = loadDoc(id8);
const board = scoreboard(id8);
console.log(`== ${id8} scoreboard servi (${board.length} lignes) :`,
  board.map((r) => `${r.gamertag}/${r.team_side}${r.is_bot ? '/bot' : ''}`).join(', '));
const players = buildPlayers(doc, board);
console.log('\n-- joueurs (buildPlayers) : cle | nom affiche | board | vies | enveloppe');
for (const p of players) {
  const env = p.lives.length ? `${Math.min(...p.lives.map((l) => trackWindow(l).start))}..${Math.max(...p.lives.map((l) => trackWindow(l).end))}` : '-';
  console.log(`  ${p.xuid.padEnd(24)} | ${playerLabel(p).padEnd(16)} | ${p.board ? p.board.team_side : 'AUCUN'} | ${p.lives.length} | ${env}`);
}
const seats = buildSeats(players, doc);
const groups = groupSeatsByTeam(seats);
console.log('\n-- sieges (buildSeats) par groupe (groupSeatsByTeam)');
for (const g of groups) {
  console.log(`  groupe ${g.key} side=${g.side}`);
  for (const s of g.seats) {
    console.log(`    siege ${s.seat} : ${s.occupants.map((o) => `${playerLabel(o.player)} [${o.fromFrame}..${o.toFrame}]`).join(' -> ')}`);
  }
}
const instants = [
  ['depart lecture (preambule)', displayToFrame(doc, -1)],
  ['0:00', displayToFrame(doc, 0)],
  ['2:14', displayToFrame(doc, 134)],
  ['6:24', displayToFrame(doc, 384)],
];
for (const [lbl, f] of instants) {
  console.log(`\n== ${lbl} -> frame ${f} (affiche ${frameToDisplay(doc, f)})`);
  for (const col of panneaux(groups, f)) {
    const nom = col.side === 't0' ? 'Eagle' : col.side === 't1' ? 'Cobra' : col.side;
    console.log(`  ${nom} (${col.key}) : ${col.tuiles.length} tuiles`);
    for (const t of col.tuiles) {
      const etat = t.kind === 'parti' ? 'PARTI (tuile « A quitte »)' : t.fantome ? `PRESENT-FANTOME (presence finie a ${t.o.toFrame} = ${frameToDisplay(doc, t.o.toFrame)}, aucun successeur sur ce siege)` : 'present';
      console.log(`     siege ${String(t.seat).padStart(2)} ${t.name.padEnd(16)} ${etat}`);
    }
  }
}
