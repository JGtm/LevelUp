// frise.mjs — pour UN document : a chaque seconde ou le compte affiche differe de l'API, qui manque
// et qui est en trop. Usage : node frise.mjs <id8> [pasEnSecondes]
import {
  loadDoc, scoreboard, participants, registry, buildPlayers, buildSeats, groupSeatsByTeam, panneaux,
  frameToDisplay, trackWindow,
} from './panneaux_lib.mjs';

const id8 = process.argv[2];
const pasS = Number(process.argv[3] ?? 1);
const doc = loadDoc(id8);
const reg = registry(id8);
const parts = participants(id8);
const st = Number(reg.st);
const pas = doc.frameIntervalMs;
const toF = (ms) => (Number(ms) - st - doc.originMs) / pas;
const players = buildPlayers(doc, scoreboard(id8));
const groups = groupSeatsByTeam(buildSeats(players, doc));
const f0 = doc.t0FilmMs != null ? Math.round((doc.t0FilmMs - doc.originMs) / pas) : 0;
const fEnd = Math.min(doc.frameCount - 1, Math.floor((Number(reg.duration_seconds) * 1000 - doc.originMs) / pas));
console.log(`${id8} ${reg.playlist_name} ${reg.pair_name} f0=${f0} fEnd=${fEnd} frames=${doc.frameCount}`);
console.log('participants API (frame film) :');
for (const p of parts) {
  const cle = p.xuid.startsWith('bid(') ? `bot:${p.gamertag} [bot]` : p.xuid;
  const pl = players.find((x) => x.xuid === cle);
  const env = pl && pl.lives.length ? `${Math.min(...pl.lives.map((l) => trackWindow(l).start))}..${Math.max(...pl.lives.map((l) => trackWindow(l).end))} (${pl.lives.length} vies)` : 'AUCUNE VIE';
  const r = (doc.roster ?? []).find((e) => (e.xuid || (e.bot ? `bot:${e.name}` : '')) === cle);
  console.log(`  t${p.team_id} ${p.gamertag.padEnd(18)} api[${p.fj == null ? '-' : toF(p.fj).toFixed(0)}..${p.ll == null ? 'fin' : toF(p.ll).toFixed(0)}] deb=${p.present_at_beginning} fin=${p.present_at_completion} vies=${env} roster=${r ? `idx${r.filmIndex}/siege${r.seat}/team${r.team ?? '?'}` : 'ABSENT'}`);
}
const apiAt = (team, f) => parts.filter((p) => Number(p.team_id) === team && (p.fj == null || toF(p.fj) <= f) && (p.ll == null || f < toF(p.ll)));
let prev = '';
for (let f = f0; f <= fEnd; f += 10 * pasS) {
  const cols = panneaux(groups, f);
  const parts2 = cols.map((c) => {
    const m = /^f(\d+)$/.exec(c.key);
    const api = m ? apiAt(Number(m[1]), f) : [];
    return `${c.key}:${c.tuiles.length}t/${api.length}api [${c.tuiles.map((t) => t.name + (t.kind === 'parti' ? '(parti)' : t.fantome ? '(fantome)' : '')).join(',')}]`;
  }).join('  ');
  if (parts2 !== prev) console.log(`${frameToDisplay(doc, f)} f${f} ${parts2}`);
  prev = parts2;
}
