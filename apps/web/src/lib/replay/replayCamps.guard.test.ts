/**
 * Garde-rail — UNE SECTION SANS ÉQUIPE N'EXISTE PAS SUR LA PAGE REJEU (décision du 2026-10-06 ;
 * CLAUDE.md n°6 et anti-patterns « Compatibility guard forever », « Factorisation abandonnée »).
 *
 * POURQUOI. L'équipe d'un joueur du rejeu est le désignateur du FILM (`ReplayPlayer.team`, posé
 * par `buildPlayers`), et tous les regroupements de la page en sortent par `replayCamps.ts`. Ce
 * qui a fait apparaître une troisième colonne « Sans équipe » est une RELECTURE de la feuille de
 * match pour décider de l'appartenance (`campsParCote`, clé `s:<team_side>`, groupes
 * `side == null`) : un repli de ce genre se réécrit de bonne foi — d'où ce test, qui refuse la
 * FORME et pas seulement la valeur.
 *
 * TROIS INTERDITS, sur le code (commentaires ôtés) de `features/match-replay` et `lib/replay`
 * (tests exclus) :
 *   (a) lire `.team_side` hors du helper de libellé (`replayCamps.ts`), sauf dans les fichiers
 *       de l'allowlist ci-dessous — chacun JUSTIFIÉ, DATÉ et COMPTÉ (le compte est un cliquet :
 *       une lecture de plus échoue, une de moins aussi, pour que la liste dise toujours vrai) ;
 *   (b) le retour de « Sans équipe » / « No team » dans le CODE (chaîne ou texte JSX, casse
 *       ignorée), des clés `teamUnknown` / `viewpointNoTeam`, ou de la clé de rendu
 *       `sans-equipe` — ici, et sur la route de la page Rejeu ;
 *   (c) depuis le 2026-10-06, l'ALLÉGEANCE (allié / adverse) lue ailleurs que dans le film :
 *       une table d'identité interrogée pour son drapeau `ally` (`….get(xuid)?.ally`), ou le
 *       lecteur de feuille de la page Match (`allyOfTeamId`). Le seul calcul d'allégeance du
 *       rejeu est `filmAllegiance.ts` — ici aussi sur la route.
 */
import { describe, expect, it } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à
// node:fs ni aux types node dans le tsconfig applicatif.
const PERIMETRE = import.meta.glob(['/src/features/match-replay/**/*.{ts,tsx}', '/src/lib/replay/**/*.{ts,tsx}'], {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>
const ROUTE = import.meta.glob('/src/routes/**/replay.tsx', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const HELPER = '/src/lib/replay/replayCamps.ts'

/**
 * (a) L'ALLOWLIST — les lectures de `team_side` qui ne font que NOMMER un camp. Depuis que
 * l'encre et l'allégeance viennent du film (2026-10-06, `filmAllegiance.ts`), il n'en reste
 * qu'une famille : le helper de libellé lui-même. Les huit entrées de l'encre (tables de l'onglet
 * Arsenal, objectifs, bandeau, écran de fin, fil, opposition du capteur, calque de score) en sont
 * sorties une à une, le compte à zéro. Ajouter une ligne ici demande une raison écrite et datée,
 * et une lecture qui NOMME : ni un regroupement, ni une encre n'y ont leur place.
 */
const ALLOWLIST: Readonly<Record<string, { lectures: number; raison: string }>> = {
  [HELPER]: {
    lectures: 2,
    raison: '2026-10-06 — LE helper de libellé : nomme un camp du film (côté majoritaire de ses membres, lignes de son côté).',
  },
}

/** Le code seul : blocs et lignes de commentaire ôtés (un `//` dans une URL citée reste). */
function code(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:'"`\\])\/\/.*$/gm, '$1')
}

/**
 * (a) Une LECTURE de `team_side`, sous ses quatre formes :
 *   - l'accès pointé : `r.team_side`, `p.board?.team_side` ;
 *   - l'accès entre crochets sur une VALEUR : `r['team_side']` (un type indexé
 *     `MatchScoreboardRow['team_side']` commence par une majuscule et n'en est pas un) ;
 *   - la déstructuration d'une variable : `const { team_side } = r`,
 *     `for (const { xuid, team_side } of rows)` ;
 *   - la déstructuration d'un paramètre, à n'importe quel rang : `({ team_side }) => …`,
 *     `(n, { team_side }) => …`, `function f({ team_side }: Row) { … }`.
 * N'en sont pas : une clé d'objet littéral (`{ team_side: 't0' }`, même passé en argument), un
 * type (`Pick<…, 'team_side'>`, annotation `r: { team_side: string }`). Limite assumée : un appel
 * détourné du helper lui-même (`campSideOf`) pour décider d'une appartenance échappe au grep — ce
 * sont les tests des surfaces (entrées muettes que la feuille connaît) qui le rattrapent.
 */
const LECTURES = [
  /\.team_side\b/g,
  /(?<!\b[A-Z][\w$]*)\[\s*['"`]team_side['"`]\s*\]/g,
  /\b(?:const|let|var)\s*\{[^{}]*\bteam_side\b[^{}]*\}\s*(?:=(?!=)|of\b)/g,
  /[(,]\s*\{[^{}]*\bteam_side\b[^{}]*\}(?:\s*:[^,()]*)?\s*(?:,[^()]*)?\)\s*(?::[^=;{]*)?(?:=>|\{)/g,
]
/** (b) Le retour de la section « sans équipe » dans le code : son libellé, ses clés. */
const SANS_EQUIPE = [/sans équipe|no team/i, /\bteamUnknown\b/, /\bviewpointNoTeam\b/, /sans-equipe/]
/**
 * (c) L'allégeance lue hors du film : le drapeau `ally` d'une table d'identité (une lecture
 * `get(…)` suivie de `.ally`, chaîne optionnelle comprise), et le lecteur de feuille de la page
 * Match. Les propriétés `ally` STRUCTURELLES (le côté allié d'un bandeau, d'un score, d'une
 * table de sons) ne sont pas des lectures d'identité et ne sont pas prises.
 */
const ALLEGEANCE_HORS_FILM = [/\.get\([^()]*\)\s*\??\.\s*ally\b/, /\ballyOfTeamId\b/]

function lectures(source: string): number {
  const c = code(source)
  return LECTURES.reduce((n, motif) => n + (c.match(motif) ?? []).length, 0)
}

const sourcesDuPerimetre = () =>
  Object.entries(PERIMETRE).filter(([chemin]) => !/\.test\.tsx?$/.test(chemin))

describe('garde-rail : l’appartenance d’un joueur du rejeu vient du film seul', () => {
  it('le périmètre est bien lu — sans quoi ce garde ne garderait rien', () => {
    const chemins = sourcesDuPerimetre().map(([c]) => c)
    expect(chemins).toContain(HELPER)
    expect(chemins).toContain('/src/features/match-replay/model/seatLogic.ts')
    // 271 sources hors tests au 2026-10-06 : un balayage qui n'en verrait qu'une poignée serait
    // vert et inerte.
    expect(chemins.length).toBeGreaterThan(200)
    expect(Object.keys(ROUTE).some((c) => c.endsWith('/matches/$matchId/replay.tsx'))).toBe(true)
  })

  it('(a) aucune lecture de `team_side` hors du helper de libellé et de l’allowlist comptée', () => {
    const fautifs = sourcesDuPerimetre()
      .map(([chemin, source]) => [chemin, lectures(source)] as const)
      .filter(([chemin, n]) => n !== (ALLOWLIST[chemin]?.lectures ?? 0))
      .map(([chemin, n]) => `${chemin} : ${n} lecture(s), ${ALLOWLIST[chemin]?.lectures ?? 0} admise(s)`)
    expect(
      fautifs,
      'lire `team_side` décide d’une appartenance que seul le film donne — passer par `replayCamps.ts` ' +
        '(campLabel pour nommer), ou justifier et dater une entrée de l’allowlist',
    ).toEqual([])
  })

  it('(a) chaque entrée de l’allowlist existe et porte sa raison datée', () => {
    const chemins = new Set(sourcesDuPerimetre().map(([c]) => c))
    for (const [chemin, { raison }] of Object.entries(ALLOWLIST)) {
      expect(chemins.has(chemin), chemin).toBe(true)
      expect(raison, chemin).toMatch(/^\d{4}-\d{2}-\d{2} — .{20,}/)
    }
  })

  it('(c) aucune allégeance lue hors du film : ni `….get(xuid)?.ally`, ni `allyOfTeamId`', () => {
    const fautifs = [...sourcesDuPerimetre(), ...Object.entries(ROUTE)]
      .filter(([, source]) => ALLEGEANCE_HORS_FILM.some((motif) => motif.test(code(source))))
      .map(([chemin]) => chemin)
    expect(
      fautifs,
      'l’allégeance du rejeu vient du film (`filmAllegiance.ts`) — la table d’identité ne sert qu’aux noms',
    ).toEqual([])
  })

  it('(b) ni « Sans équipe », ni « No team », ni `teamUnknown`, `viewpointNoTeam` ou `sans-equipe`', () => {
    const fautifs = [...sourcesDuPerimetre(), ...Object.entries(ROUTE)]
      .filter(([, source]) => SANS_EQUIPE.some((motif) => motif.test(code(source))))
      .map(([chemin]) => chemin)
    expect(fautifs, 'une section « sans équipe » n’existe pas sur la page Rejeu').toEqual([])
  })
})

describe('garde-rail : contre-épreuves des détecteurs', () => {
  it('(a) une lecture de `team_side` est comptée, une clé, un type ou un commentaire ne l’est pas', () => {
    expect(lectures('const s = p.board?.team_side ?? null')).toBe(1)
    expect(lectures('rows.filter((r) => r.team_side === side)')).toBe(1)
    expect(lectures("const s = row['team_side']")).toBe(1)
    expect(lectures('const { team_side } = p.board ?? {}')).toBe(1)
    expect(lectures('const { xuid, team_side: side } = row')).toBe(1)
    expect(lectures('for (const { xuid, team_side } of scoreboard) parXuid.set(xuid, team_side)')).toBe(1)
    expect(lectures('rows.map(({ team_side }) => team_side)')).toBe(1)
    expect(lectures('scoreboard.reduce((n, { team_side }) => n + (team_side ? 1 : 0), 0)')).toBe(1)
    expect(lectures("function f({ team_side }: Row): string { return team_side ?? '' }")).toBe(1)
    expect(lectures('rows.map((r) => ({ id: r.xuid, side: r.team_side }))')).toBe(1)
    expect(lectures("const r = { team_side: 't0' }")).toBe(0)
    expect(lectures("const rows = [{ xuid: 'a', team_side: 't0' }]")).toBe(0)
    expect(lectures("rows.map((r) => ({ ...r, team_side: 't0' }))")).toBe(0)
    expect(lectures('out.push({ xuid, team_side: side })')).toBe(0)
    expect(lectures("type S = MatchScoreboardRow['team_side']")).toBe(0)
    expect(lectures('const r: { team_side: string | null } = row')).toBe(0)
    expect(lectures('const f = (r: { team_side: string }) => r')).toBe(0)
    expect(lectures("type R = Pick<MatchScoreboardRow, 'team_side'>")).toBe(0)
    expect(lectures('// le côté (`board.team_side`) ne fait que nommer\nconst x = 1')).toBe(0)
    expect(lectures('/** le côté `r.team_side` */\nconst x = 1')).toBe(0)
  })

  it('(b) le libellé (chaîne ou texte JSX) et les clés sont pris, la prose des commentaires non', () => {
    const pris = (s: string) => SANS_EQUIPE.some((motif) => motif.test(code(s)))
    expect(pris("teamUnknown: 'Sans équipe',")).toBe(true)
    expect(pris("viewpointNoTeam: 'No team',")).toBe(true)
    expect(pris('const label = t.teamUnknown')).toBe(true)
    expect(pris("key: seg.side ?? 'sans-equipe',")).toBe(true)
    expect(pris('return <span>Sans équipe</span>')).toBe(true)
    expect(pris('const fmt = (n: number) => `No team (${n})`')).toBe(true)
    expect(pris('// jamais une section « sans équipe »\nconst x = 1')).toBe(false)
    expect(pris("const label = campLabel(camp, rows, t)")).toBe(false)
  })

  it('(c) la lecture d’allégeance dans une table d’identité est prise, l’allégeance du film et les `ally` structurels non', () => {
    const pris = (s: string) => ALLEGEANCE_HORS_FILM.some((motif) => motif.test(code(s)))
    expect(pris('const isAlly = (xuid: string) => xuidMeta?.get(xuid)?.ally ?? false')).toBe(true)
    expect(pris('if (identity.get(k.xuid)?.ally !== true) continue')).toBe(true)
    expect(pris('const a = meta.get(x).ally')).toBe(true)
    expect(pris('allyOf: (teamId) => allyOfTeamId(board, xuidMeta, teamId)')).toBe(true)
    expect(pris('const ally = allegiance.ofXuid(xuid)')).toBe(false)
    expect(pris('readHillHold(doc, reading.ally.teamId, reading.enemy.teamId, frame)')).toBe(false)
    expect(pris("if (side === 'ally') return entry.ally")).toBe(false)
    expect(pris('// jamais xuidMeta.get(xuid)?.ally\nconst x = 1')).toBe(false)
  })
})
