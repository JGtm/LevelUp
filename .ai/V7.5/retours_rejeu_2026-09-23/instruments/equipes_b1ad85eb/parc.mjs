// parc.mjs — L'INVARIANT « a tout instant, tuiles par equipe <= taille d'equipe du mode » (et
// « >= presents selon l'API ») mesure sur les 111 documents, en REJOUANT la regle d'affichage du
// web (panneaux_lib.mjs). Un document a la fois en memoire.
//
// Echantillonnage : une image par seconde (10 frames) dans la fenetre de jeu
// [ (t0FilmMs - originMs)/pas , (duration*1000 - originMs)/pas ].
// Capacite : taille nominale de la liste de lecture (BTB 12, sinon 4) ; controle = max simultane de
// l'API par equipe (doit etre <= capacite).
// Presence API d'un participant : [first_joined_time, last_leave_time) ramene sur l'axe du film
// (abs = start_time + originMs + f * pas), last_leave NULL = jusqu'a la fin.
//
// Usage : node parc.mjs [regle]   regle = web (defaut, la regle d'aujourd'hui) | apres (cf. regle_apres.mjs)
import fs from 'fs';
import {
  REPO, SPD, loadDoc, scoreboard, participants, registry, buildPlayers, buildSeats,
  groupSeatsByTeam, panneaux, stripBotSuffix,
} from './panneaux_lib.mjs';

const regle = process.argv[2] ?? 'web';
const apres = regle === 'apres' ? await import('./regle_apres.mjs') : null;

const ids = fs.readdirSync(`${REPO}/data/cache/replays/halo_infinite`)
  .filter((f) => /^[0-9a-f]{8}\.json$/.test(f)).map((f) => f.slice(0, 8)).sort();

// Taille d'equipe du mode, MESUREE sur les 111 matchs (present_at_beginning + present_at_completion
// par equipe) : BTB 12, Squad Battle 8, Quick Play / Ranked Arena / Ranked Slayer 4.
const capaciteNominale = (reg) => (/big team/i.test(reg?.playlist_name ?? '') ? 12 : /squad battle/i.test(reg?.playlist_name ?? '') ? 8 : 4);

const lignes = [];
const agg = {
  regle, docs: 0, secondes: 0,
  docsSurCapacite: 0, secSur: 0, docsSousAPI: 0, secSous: 0, docsSousAPI30: 0,
  fantomesSecIllegitimes: 0, docsFantomesIllegitimes: 0, fantomesSecLegitimes: 0,
  botsRoster: 0, botsSansEquipe: 0, entreesSansEquipe: 0, docsBotsSansEquipe: 0,
  horsTable: 0, docsHorsTable: 0, pistesAnonymes: 0, docsPistesAnonymes: 0,
  sansPresence: 0, docsSansPresence: 0, presencesCloses: 0, apparies: 0, reprisesEcrites: 0,
  indexPartages: 0, docsIndexPartages: 0, apiMaxSurCapacite: 0, docsSiegesSurCapacite: 0,
  participantsAPIHorsScoreboard: 0, docsParticipantsHorsScoreboard: 0,
};

for (const id8 of ids) {
  const doc = loadDoc(id8);
  const reg = registry(id8);
  const board = scoreboard(id8);
  const parts = participants(id8);
  agg.docs++;
  const pas = doc.frameIntervalMs;
  const st = Number(reg.st);
  const f0 = doc.t0FilmMs != null ? Math.round((doc.t0FilmMs - doc.originMs) / pas) : 0;
  const fEnd = Math.min(doc.frameCount - 1, Math.floor((Number(reg.duration_seconds) * 1000 - doc.originMs) / pas));
  const cap = capaciteNominale(reg);
  const toF = (ms) => (ms == null ? null : (Number(ms) - st - doc.originMs) / pas);
  // --- Compteurs de publication (document)
  const roster = doc.roster ?? [];
  const bots = roster.filter((e) => e.bot);
  const botsSansEquipe = bots.filter((e) => e.team === undefined || e.team === null).length;
  const sansEquipe = roster.filter((e) => e.team === undefined || e.team === null).length;
  const parIndex = new Map();
  for (const e of roster) parIndex.set(e.filmIndex, (parIndex.get(e.filmIndex) ?? 0) + 1);
  const indexPartages = [...parIndex.values()].filter((n) => n > 1).length;
  const ht = doc.identity?.coverage?.bipedSlot?.non_resolu_par_cause?.index_hors_table ?? 0;
  const anonymes = (doc.tracks ?? []).filter((t) => !t.xuid && !t.bot).length;
  const seatsCov = doc.coverage?.seats ?? {};
  const horsBoard = parts.length - board.length;
  agg.botsRoster += bots.length; agg.botsSansEquipe += botsSansEquipe; agg.entreesSansEquipe += sansEquipe;
  if (botsSansEquipe) agg.docsBotsSansEquipe++;
  agg.horsTable += ht; if (ht) agg.docsHorsTable++;
  agg.pistesAnonymes += anonymes; if (anonymes) agg.docsPistesAnonymes++;
  agg.sansPresence += seatsCov.sansPresence ?? 0; if (seatsCov.sansPresence) agg.docsSansPresence++;
  agg.presencesCloses += seatsCov.presencesCloses ?? 0;
  agg.apparies += seatsCov.apparies ?? 0; agg.reprisesEcrites += seatsCov.reprisesEcrites ?? 0;
  agg.indexPartages += indexPartages; if (indexPartages) agg.docsIndexPartages++;
  agg.participantsAPIHorsScoreboard += horsBoard; if (horsBoard) agg.docsParticipantsHorsScoreboard++;

  // --- La participation API, par cle d'identite du rejeu (xuid, ou bot:<nom> via le gamertag)
  const apiParCle = new Map();
  const apiParEquipe = new Map();
  for (const p of parts) {
    const cle = p.xuid.startsWith('bid(') ? `bot:${p.gamertag} [bot]` : p.xuid;
    const a = p.fj == null ? -Infinity : toF(p.fj);
    const b = p.ll == null ? Infinity : toF(p.ll);
    apiParCle.set(cle, { a, b, left: p.left_in_progress === 'true', team: p.team_id == null ? null : Number(p.team_id) });
    if (p.team_id == null) continue;
    const t = Number(p.team_id);
    if (!apiParEquipe.has(t)) apiParEquipe.set(t, []);
    apiParEquipe.get(t).push([a, b]);
  }
  const apiAt = (team, f) => (apiParEquipe.get(team) ?? []).filter(([a, b]) => a <= f && f < b).length;

  // --- Rendu
  const players = buildPlayers(doc, board);
  const groups = apres ? apres.groupes(doc, players, parts, { st, toF }) : groupSeatsByTeam(buildSeats(players, doc));
  const siegesSurCap = !apres && groups.some((g) => g.seats.length > cap);
  if (siegesSurCap) agg.docsSiegesSurCapacite++;
  let secSur = 0; let secSous = 0; let fantIll = 0; let fantLeg = 0; let maxT = 0; let apiMax = 0;
  let pireSur = null; let pireSous = null; let sousContinu = 0; let sousContinuMax = 0;
  for (let f = f0; f <= fEnd; f += 10) {
    agg.secondes++;
    const cols = apres ? apres.panneaux(groups, f) : panneaux(groups, f);
    let sousIci = false;
    for (const col of cols) {
      const tuiles = col.tuiles.length;
      const presents = col.tuiles.filter((t) => t.kind === 'present').length;
      for (const t of col.tuiles) {
        if (!t.fantome) continue;
        const api = apiParCle.get(t.xuid);
        // ILLEGITIME : l'API dit que ce joueur a QUITTE, et l'instant lu est apres son depart API
        // (ou le joueur est inconnu de l'API). LEGITIME : il est reste jusqu'au bout (mort sans
        // reapparaitre avant la fin : « mourir n'est pas partir »).
        if (!api || (api.left && f >= api.b)) fantIll++; else fantLeg++;
      }
      maxT = Math.max(maxT, tuiles);
      if (tuiles > cap) { secSur++; if (!pireSur || tuiles > pireSur.n) pireSur = { f, n: tuiles, col: col.key }; }
      const m = /^f(\d+)$/.exec(col.key);
      if (m) {
        const api = apiAt(Number(m[1]), f);
        apiMax = Math.max(apiMax, api);
        if (presents < api) {
          secSous++; sousIci = true;
          if (!pireSous || api - presents > pireSous.d) pireSous = { f, d: api - presents, col: col.key, presents, api };
        }
      }
    }
    sousContinu = sousIci ? sousContinu + 1 : 0;
    sousContinuMax = Math.max(sousContinuMax, sousContinu);
  }
  agg.secSur += secSur; agg.secSous += secSous; agg.fantomesSecIllegitimes += fantIll; agg.fantomesSecLegitimes += fantLeg;
  if (secSur) agg.docsSurCapacite++;
  if (secSous) agg.docsSousAPI++;
  if (sousContinuMax > 30) agg.docsSousAPI30++;
  if (fantIll) agg.docsFantomesIllegitimes++;
  if (apiMax > cap) agg.apiMaxSurCapacite++;
  lignes.push([id8, reg.playlist_name, cap, groups.length, apres ? '' : groups.map((g) => g.seats.length).join('/'),
    maxT, secSur, secSous, sousContinuMax, fantIll, fantLeg, bots.length, botsSansEquipe, sansEquipe, indexPartages, ht, anonymes,
    seatsCov.sansPresence ?? '', seatsCov.presencesCloses ?? '', seatsCov.apparies ?? '', seatsCov.arrivants ?? '', horsBoard,
    pireSur ? `${pireSur.col}@${pireSur.f}:${pireSur.n}` : '', pireSous ? `${pireSous.col}@${pireSous.f}:${pireSous.presents}<${pireSous.api}` : '', apiMax].join('\t'));
}
const entete = ['id8', 'playlist', 'cap', 'groupes', 'siegesParGroupe', 'maxTuiles', 'secSur', 'secSousAPI', 'sousContinuMax',
  'fantomeSecIllegitime', 'fantomeSecLegitime', 'bots', 'botsSansEquipe', 'entreesSansEquipe', 'indexPartages', 'horsTable',
  'pistesAnonymes', 'sansPresence', 'presencesCloses', 'apparies', 'arrivants', 'participantsHorsScoreboard', 'pireSur', 'pireSous',
  'apiMax'].join('\t');
fs.writeFileSync(`${SPD}/parc_${regle}.tsv`, [entete, ...lignes].join('\n') + '\n');
console.log(JSON.stringify(agg, null, 1));
