/**
 * sessionEmprise.logic.ts — LES MODÈLES PURS des cartes « Frags et usages » de la page Sessions
 * (plan PLAN_SESSIONS_EMPRISE_2026-10-06, D11 et §3 : cartes A à L).
 *
 * Une COLONNE de session (la session affichée, ou la session comparée) porte ses blocs :
 * `SessionColumnBlocks`. Les cartes sont celles de l'Escouade et des Séries temporelles ; ce fichier
 * leur donne ce que la page Sessions sait (l'index de ses matchs, avec score et dominance) et rend
 * `sessionCardsPresence`, LE prédicat de présence par carte : la page le lit pour ses rangées
 * partagées (`_sections.ts`), la colonne pour ses intertitres et ses cartes — un seul endroit,
 * aucune divergence possible entre un intertitre et ce qu'il coiffe.
 *
 * Aucun branchement sur le titre : tout se lit dans les blocs (Halo 5 sans film n'a que la feuille
 * de match : A, B, B' et la barre épaisse des armes spéciales de G).
 *
 * FILM NON LU. Un titre qui lit le film dont certains matchs n'ont pas leur résumé (analyse en cours, ou
 * jamais faite) ne retire pas les cartes qui en dépendent : chaque famille sans aucune carte (Ressources,
 * Prendre, Vies, Équipement) garde son intertitre et UNE carte d'état (`sessionFilmGaps`, ancrée sur la
 * première clé de la famille) — une section absente laissait croire à un bug. Capability absente
 * (`film_unsupported`) : famille absente, comme avant.
 *
 * AUCUN MATCH CONCERNÉ. Une carte tirée de la feuille de match dont aucun match de la session n'est concerné
 * (l'objectif sans match à objectif, la répartition des frags sans frag) reste affichée avec son constat
 * (`sessionScopeEmpties`, `ScopeEmptyCard`) : un bloc qui disparaît se lit comme un bug. Capability absente
 * (`objectiveExpected` faux) : carte absente.
 *
 * Pur : aucun React, aucune couleur, aucune chaîne de langue.
 */
import type { EmpriseMatchIndex, ControlRow, MatchGrid, ResourceFil } from '@/features/squad/emprise/emprise.logic'
import { buildControlRows, buildMatchGrid, buildResourceFil } from '@/features/squad/emprise/emprise.logic'
import { gridHasFilmRows } from '@/features/squad/emprise/empriseContent'
import { buildProductionRows, buildYieldRows, type ProductionRow, type YieldRow } from '@/features/squad/emprise/production.logic'
import { buildSquadToolRows, type SquadToolKindLabels } from '@/features/squad/charts/squadFragTools'
import { objectiveMatches } from '@/features/squad/formes/model/objectives'
import {
  buildObjectiveBalance,
  buildSoloObjectiveSheet,
  objectiveExpected,
  type BalanceFamily,
  type SoloObjectiveSheet,
} from '@/features/squad/objectif/objectif.logic'
import {
  buildEquipmentRows,
  buildLivesModel,
  buildMinePickups,
  type EquipmentRow,
  type LivesModel,
  type MinePickups,
} from '@/features/timeseries/usages/usages.logic'
import { asDominance } from '@/components/charts/outcomeSequence'
import type {
  CoordinationBlock,
  MatchRangeBlock,
  RangeReferenceBlock,
  SessionCompareEntry,
  SessionDetailMatchRow,
  SessionPageResponse,
  SoloEmpriseBlock,
  SquadEmpriseObject,
  SquadFormesBlock,
  TimeseriesLivesNearTeammate,
} from '@/lib/api/types'
import type { ScopeEmptyKind } from '@/components/ui/scope-empty-text'
import { filmGapOfBlock, type FilmGap } from '@/features/_shared/filmGap/filmGap.logic'
import { outcomeCodeToValue } from '@/lib/outcome'

/** Les outils montrés en vue compacte (maquette `makeTools` : les six plus meurtriers). */
export const SESSION_TOOLS_TOP = 6

/** Les blocs d'UNE colonne de session, tels que la réponse de page les sert (aucune requête). */
export interface SessionColumnBlocks {
  entry: SessionCompareEntry | null
  matches: SessionDetailMatchRow[]
  emprise?: SoloEmpriseBlock | null
  lives?: TimeseriesLivesNearTeammate | null
  formes?: SquadFormesBlock | null
  emblemUrl?: string
  coordination?: CoordinationBlock | null
  rangeProfiles?: MatchRangeBlock | null
  /** La période de référence de la portée : UN bloc pour les deux colonnes (filtre de la page). */
  rangeReference?: RangeReferenceBlock | null
}

/** sessionColumnBlocks — les blocs de la session affichée (`current`) ou comparée (`compare`). */
export function sessionColumnBlocks(data: SessionPageResponse, side: 'current' | 'compare'): SessionColumnBlocks {
  const common = { emblemUrl: data.player_emblem_url || undefined, rangeReference: data.range_reference }
  if (side === 'current') {
    return {
      ...common,
      entry: data.current_session ?? null,
      matches: data.matches ?? [],
      emprise: data.emprise,
      lives: data.lives_near_teammate,
      formes: data.formes_retenues,
      coordination: data.coordination,
      rangeProfiles: data.range_profiles,
    }
  }
  return {
    ...common,
    entry: data.compare_session ?? null,
    matches: data.compare_matches ?? [],
    emprise: data.compare_emprise,
    lives: data.compare_lives_near_teammate,
    formes: data.compare_formes_retenues,
    coordination: data.compare_coordination,
    rangeProfiles: data.compare_range_profiles,
  }
}

/**
 * sessionMatchIndex — l'index des matchs de la session : heure, carte, mode, résultat, score et
 * drapeau de dominance (D15) — la bande de résultats et l'encoche du fil (D), les en-têtes de la
 * grille (E).
 */
export function sessionMatchIndex(rows: SessionDetailMatchRow[]): EmpriseMatchIndex {
  return new Map(
    rows.map((r) => [
      r.match_id,
      {
        matchId: r.match_id,
        startTime: r.start_time ?? '',
        map: r.map_name ?? '',
        mode: r.mode_ui ?? '',
        outcome: outcomeCodeToValue(r.outcome),
        score: r.score_label || null,
        dominance: asDominance(r.dominance_flag),
      },
    ]),
  )
}

/** Les modèles des cartes de l'Emprise, de « Mes vies » et de l'Objectif d'une colonne. */
export interface SessionEmpriseModels {
  controlRows: ControlRow[]
  fil: ResourceFil | null
  grid: MatchGrid | null
  mine: MinePickups | null
  production: ProductionRow[]
  yieldRows: YieldRow[]
  lives: LivesModel | null
  equipment: EquipmentRow[]
  /** Le bloc d'objectif, seulement quand la session a au moins un match à objectif. */
  objective: SquadFormesBlock | null
  balance: BalanceFamily[]
  soloSheet: SoloObjectiveSheet | null
  /** Le titre mesure l'objectif mais aucun match de la session n'est à objectif : la carte dit le constat. */
  objectiveEmpty: boolean
}

/** buildSessionEmpriseModels — tous les modèles d'une colonne, depuis ses blocs (aucune requête). */
export function buildSessionEmpriseModels(
  col: SessionColumnBlocks,
  nameOf: (o: SquadEmpriseObject) => string,
): SessionEmpriseModels {
  const block = col.emprise ?? null
  const formes = col.formes ?? null
  const objective = formes && objectiveMatches(formes).length > 0 ? formes : null
  const index = sessionMatchIndex(col.matches)
  return {
    controlRows: block ? buildControlRows(block) : [],
    fil: block ? buildResourceFil(block, index) : null,
    grid: block ? buildMatchGrid(block, index) : null,
    mine: block ? buildMinePickups(block, nameOf) : null,
    production: block ? buildProductionRows(block) : [],
    yieldRows: block ? buildYieldRows(block) : [],
    lives: buildLivesModel(col.lives),
    equipment: block ? buildEquipmentRows(block) : [],
    objective,
    balance: objective ? buildObjectiveBalance(objective) : [],
    soloSheet: objective ? buildSoloObjectiveSheet(objective) : null,
    objectiveEmpty: objective == null && objectiveExpected(formes),
  }
}

/** Les cartes A à L (et B', D14), une clé chacune. */
export interface SessionCardsPresence {
  frag_donut: boolean
  tools: boolean
  weapon_accuracy: boolean
  control: boolean
  fil: boolean
  grid: boolean
  mine: boolean
  production: boolean
  yield: boolean
  lives: boolean
  objective_balance: boolean
  objective_sheet: boolean
  equipment: boolean
}

/** sessionFilmGap — le film des matchs de la colonne n'est pas lu pour au moins un match (`filmGapOfBlock`). */
export function sessionFilmGap(col: Pick<SessionColumnBlocks, 'emprise'>): FilmGap | null {
  return filmGapOfBlock(col.emprise)
}

/**
 * Les familles de cartes tirées du film : si AUCUNE de leurs cartes ne se dessine alors que le film n'est
 * pas lu, la première porte l'état (`FilmGapCard`) sous l'intertitre de la famille. Objectif (feuille de
 * match) et frags (feuille de match) n'en font pas partie : leur absence dit « aucun match concerné ».
 */
const FILM_FAMILIES: ReadonlyArray<readonly (keyof SessionCardsPresence)[]> = [
  ['control', 'fil', 'grid', 'mine'],
  ['production', 'yield'],
  ['lives'],
  ['equipment'],
]

/** sessionFilmGaps — les cartes d'ancrage qui portent un état « film non lu » au lieu de leur contenu. */
export function sessionFilmGaps(
  col: Pick<SessionColumnBlocks, 'emprise'>,
  drawn: SessionCardsPresence,
): Partial<Record<keyof SessionCardsPresence, FilmGap>> {
  const gap = sessionFilmGap(col)
  const out: Partial<Record<keyof SessionCardsPresence, FilmGap>> = {}
  if (!gap) return out
  for (const family of FILM_FAMILIES) {
    if (!family.some((key) => drawn[key])) out[family[0]] = gap
  }
  return out
}

/** Libellés NEUTRES : ils servent à COMPTER les outils, jamais à les afficher. */
const COUNT_ONLY_TOOL_LABELS: SquadToolKindLabels = {
  melee: 'melee',
  grenade: 'grenade',
  assassination: 'assassination',
  ground_pound: 'ground_pound',
  shoulder_bash: 'shoulder_bash',
  explosive_object: 'explosive_object',
  environment: 'environment',
  unattributed: 'unattributed',
}

/** Le nom du joueur de la page, tel que les outils le portent (sinon l'Emprise, sinon la repli). */
export function sessionPlayerName(col: SessionColumnBlocks, fallback: string): string {
  return col.entry?.weapon_tools?.players?.[0] || col.emprise?.players?.[0]?.gamertag || fallback
}

/**
 * sessionCardsPresence — LA présence de chaque carte d'une colonne. Une carte n'existe que si elle
 * dessine quelque chose (S10) — ou porte l'état « film non lu » de sa famille (`sessionFilmGaps`) :
 *   - A : des frags par classe ; B : au moins un outil NOMMÉ (« Non attribué » seul ne se dessine
 *     pas en vue compacte, la carte ne s'ouvre donc nulle part) ; B' : la précision par arme native ;
 *   - C, D : des prises de ressource ; E : au moins une ligne lue au film ; F : au moins une prise de
 *     l’équipe ; G, H : leurs lignes ; I : au moins une vie rangée ; J, K : un match à objectif ;
 *   - L : au moins une famille d'équipement tenue.
 */
export function sessionCardsPresence(col: SessionColumnBlocks, m: SessionEmpriseModels): SessionCardsPresence {
  const drawn = drawnCards(col, m)
  const gaps = sessionFilmGaps(col, drawn)
  const empties = sessionScopeEmpties(col, m, drawn)
  return Object.fromEntries(
    (Object.keys(drawn) as (keyof SessionCardsPresence)[]).map((key) => [
      key,
      drawn[key] || gaps[key] != null || empties[key] != null,
    ]),
  ) as unknown as SessionCardsPresence
}

/**
 * sessionScopeEmpties — les cartes qui restent affichées faute de match concerné, avec ce qui manque :
 * la répartition des frags et les outils de destruction d'une session sans frag, l'objectif d'une session sans match à objectif (carte
 * d'ancrage `objective_balance`, sous l'intertitre « Objectif »). Une colonne sans session (`entry` nul)
 * n'a rien à dire.
 */
export function sessionScopeEmpties(
  col: Pick<SessionColumnBlocks, 'entry'>,
  m: Pick<SessionEmpriseModels, 'objectiveEmpty'>,
  drawn: SessionCardsPresence,
): Partial<Record<keyof SessionCardsPresence, ScopeEmptyKind>> {
  const out: Partial<Record<keyof SessionCardsPresence, ScopeEmptyKind>> = {}
  if (col.entry != null && !drawn.frag_donut) {
    out.frag_donut = 'frags'
    // Sans frag, « Outils de destruction » n'a rien non plus : sa carte reste avec l'état vide de son graphe.
    if (!drawn.tools) out.tools = 'frags'
  }
  if (m.objectiveEmpty) out.objective_balance = 'objective'
  return out
}

/** Les cartes qui DESSINENT quelque chose, sans les cartes d'état. */
export function drawnCards(col: SessionColumnBlocks, m: SessionEmpriseModels): SessionCardsPresence {
  const entry = col.entry
  const classes = entry?.frag_distribution?.classes ?? []
  const tools = buildSquadToolRows(entry?.weapon_tools, { locale: 'fr', labels: COUNT_ONLY_TOOL_LABELS, top: SESSION_TOOLS_TOP })
  return {
    frag_donut: (entry?.frag_distribution?.total_kills ?? 0) > 0 && classes.some((c) => c.kills > 0),
    tools: tools != null,
    weapon_accuracy: (entry?.weapon_accuracy ?? []).length > 0,
    control: m.controlRows.length > 0,
    fil: m.fil != null && m.controlRows.length > 0,
    grid: gridHasFilmRows(m.grid),
    mine: m.mine != null,
    production: m.production.length > 0,
    yield: m.yieldRows.length > 0,
    lives: m.lives != null,
    objective_balance: m.balance.length > 0,
    objective_sheet: m.soloSheet != null,
    equipment: m.equipment.length > 0,
  }
}

/** Le nom d'un objet ne change ni les lignes ni leur nombre : pour la seule présence, sa clé suffit. */
const KEY_AS_NAME = (o: SquadEmpriseObject) => o.key ?? ''

/** sessionColumnPresence — la présence des cartes d'une colonne, depuis ses seuls blocs. */
export function sessionColumnPresence(col: SessionColumnBlocks): SessionCardsPresence {
  return sessionCardsPresence(col, buildSessionEmpriseModels(col, KEY_AS_NAME))
}
