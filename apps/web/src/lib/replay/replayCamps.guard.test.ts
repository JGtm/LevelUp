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
 * DEUX INTERDITS, sur le code (commentaires ôtés) de `features/match-replay` et `lib/replay`
 * (tests exclus) :
 *   (a) lire `.team_side` hors du helper de libellé (`replayCamps.ts`), sauf dans les fichiers
 *       de l'allowlist ci-dessous — chacun JUSTIFIÉ, DATÉ et COMPTÉ (le compte est un cliquet :
 *       une lecture de plus échoue, une de moins aussi, pour que la liste dise toujours vrai) ;
 *   (b) le retour de « Sans équipe » / « No team » dans le CODE (chaîne ou texte JSX, casse
 *       ignorée), des clés `teamUnknown` / `viewpointNoTeam`, ou de la clé de rendu
 *       `sans-equipe` — ici, et sur la route de la page Rejeu.
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
 * (a) L'ALLOWLIST — les lectures de `team_side` qui NE décident PAS de l'appartenance d'un joueur
 * à un camp de la page, revues le 2026-10-06 (lot « Rejeu : aucune section sans équipe »). Toutes
 * relèvent de la famille ENCRE / ALLÉGEANCE (allié ou adverse du joueur regardé, issue, score),
 * que la feuille porte et que ce lot n'a pas refondue (point 6 du brief). Ajouter une ligne ici
 * demande une raison écrite et datée ; un fichier de regroupement n'y a pas sa place.
 */
const ALLOWLIST: Readonly<Record<string, { lectures: number; raison: string }>> = {
  [HELPER]: {
    lectures: 2,
    raison: '2026-10-06 — LE helper de libellé : nomme un camp du film (côté majoritaire de ses membres, lignes de son côté).',
  },
  '/src/features/match-replay/MatchEquipmentUsageSection.tsx': {
    lectures: 1,
    raison: '2026-10-06 — encre allié / adverse des tables : le côté du joueur de la page (`is_me`) dit lequel des camps du film est le sien.',
  },
  '/src/features/match-replay/MatchPadControlSection.tsx': {
    lectures: 1,
    raison: '2026-10-06 — même encre allié / adverse, bloc du contrôle des socles.',
  },
  '/src/features/match-replay/model/matchSides.ts': {
    lectures: 3,
    raison: '2026-10-06 — camp allié et camp par xuid des calques d’objectifs (drapeau, zones, crâne) : allégeance, pas appartenance à une section.',
  },
  '/src/features/match-replay/model/scoreBannerLogic.ts': {
    lectures: 1,
    raison: '2026-10-06 — les deux camps du bandeau de score et leur allégeance.',
  },
  '/src/features/match-replay/model/victoryLogic.ts': {
    lectures: 5,
    raison: '2026-10-06 — écran de fin : l’API ne publie l’issue que du côté de la feuille du joueur de la page.',
  },
  '/src/features/match-replay/ui/ReplayKillFeed.tsx': {
    lectures: 1,
    raison: '2026-10-06 — camp du tueur dans le fil, qui vient de la base comme le fil lui-même (encre, dominance aux frags).',
  },
  '/src/lib/replay/rosterLogic.ts': {
    lectures: 1,
    raison: '2026-10-06 — `sideResolver` : opposition du capteur de menaces (découverte D2 du plan, laissée sur la feuille).',
  },
  '/src/lib/replay/scoreTimeline.ts': {
    lectures: 1,
    raison: '2026-10-06 — `allyOfTeamId` : allégeance d’un camp du calque de score.',
  },
}

/** Le code seul : blocs et lignes de commentaire ôtés (un `//` dans une URL citée reste). */
function code(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:'"`\\])\/\/.*$/gm, '$1')
}

/** (a) Une LECTURE de `team_side` : un accès de propriété, pas une clé d'objet ni un type. */
const LECTURE = /\.team_side\b/g
/** (b) Le retour de la section « sans équipe » dans le code : son libellé, ses clés. */
const SANS_EQUIPE = [/sans équipe|no team/i, /\bteamUnknown\b/, /\bviewpointNoTeam\b/, /sans-equipe/]

function lectures(source: string): number {
  return (code(source).match(LECTURE) ?? []).length
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
    expect(lectures("const r = { team_side: 't0' }")).toBe(0)
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
})
