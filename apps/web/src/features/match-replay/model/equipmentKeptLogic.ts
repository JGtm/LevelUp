/**
 * equipmentKeptLogic.ts — LA TROISIÈME ISSUE (« gardé sans l'utiliser ») et la reconnaissance
 * rang -> famille de `equipmentChanges` qui la nourrit.
 *
 * Extrait de `equipmentUsageLogic.ts` le 2026-09-09 (scission obligatoire n°2,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09.md, étape E5 — 582 L, seuil 500, CLAUDE.md n°5). Même
 * découpe de principe que `equipmentUsageColumns.ts` extrait le 2026-08-25 : d'un côté ce que le
 * pont slot -> joueur -> équipe mesure (le fichier voisin), de l'autre ce que la TROISIÈME ISSUE
 * ajoute par-dessus (E2, décision utilisateur 2026-09-09, amende P1).
 *
 * `kept` se DÉRIVE, il n'est JAMAIS lu d'un canal direct : `keptOf(taken, utilisé, lâché)` =
 * `max(0, taken - utilisé - lâché)` par famille de `KEPT_FAMILIES`, où `taken` vient de
 * `equipmentChanges` (kind `taken`), lu par `readEquipmentChanges`.
 *
 * # LE CÔTÉ « UTILISÉ » A DEUX LECTURES (lot 5.7, aligné sur le Go `us6`)
 *
 * CORRIGÉ LE 2026-09-10 (lot 5.7, jumeau web du lot 5.5 Go —
 * `internal/games/halo_infinite/film/replay/usage_summary_outcomes.go` / `usage_summary_families.go`). Ce fichier lisait « utilisé » sur les
 * POSES `deployed` pour TOUTES les familles : c'était juste pour le MUR (qui engendre une pièce
 * distincte, ses panneaux) et FAUX pour tout le reste — une pose `deployed` sur un objet PORTÉ
 * ne mesure pas un déploiement, elle mesure un LÂCHER VOLONTAIRE À MI-VIE (l'objet qui tombe
 * parce que son porteur en ramasse un autre) ; le Go a mesuré ZÉRO consommation de charge
 * couverte par une pose de la même famille du même joueur à moins de 2 s, hors mur (84 %).
 * L'écart était CONNU ; ce lot le referme côté web.
 *
 * La règle, désormais commune aux deux dépôts :
 *   - les deux BONUS (`camo`, `overshield`) : le compte d'ÉPISODES d'état actif ;
 *   - le MUR, seul déployable qui ENGENDRE UNE PIÈCE (`KEPT_FAMILIES_WITH_SPAWNED_PIECE`) : ses
 *     poses `deployed` ;
 *   - tout autre déployable (capteur, traqueur, écran occultant, champ de réparation, balise du
 *     translocateur) : ses CONSOMMATIONS de charge (`equipmentChanges` kind `spent`, tally
 *     `spent`, jointes sur le rang PRÉCÉDENT `from`).
 * `usageUsedOf` est le SEUL endroit qui traduit entre ces trois vocabulaires — jumeau du
 * `usageUsedOf` Go (`usage_summary_outcomes.go`), même nom, même rôle, deux dépôts.
 *
 * # LA JOINTURE RANG -> FAMILLE SE FAIT SUR LA FAMILLE PUBLIÉE (schéma 51, lot 4.3)
 *
 * Elle se faisait sur la RACINE DU LIBELLÉ (`EQUIPMENT_CHANGE_FAMILY_STEMS`, un pis-aller écrit
 * avant que le document publie la table du manifeste). Le document publie désormais
 * `abilityLabels[rank].family` (type généré `Label.family`, cf. `apps/web/src/lib/api/
 * generated.ts`) : la reconstruction par racine a DISPARU d'ici (lot 5.7), avec elle la seconde
 * copie de la table que la règle CLAUDE.md n°6 plafonnait — même bascule que le Go
 * (`equipmentOutcomeStems` supprimée côté Go le 2026-09-10).
 *
 * Le vocabulaire du manifeste est celui des POSES/SOCLES pour les deux bonus (`powerup_camo`,
 * `powerup_overshield`) : le pont `EPISODE_FAMILY_OF_POWERUP` (gameChangers.ts, décision D5) les
 * ramène au vocabulaire d'ÉPISODE (`camo`, `overshield`) qu'emploie `KEPT_FAMILIES` — jamais une
 * seconde table, la même que le bilan « game changers » utilise déjà dans l'autre sens.
 *
 * UN LABEL SANS `family` (artefact ANTÉRIEUR au schéma 51, ou table du film incomplète) EST UN
 * REPLI EXPLICITE, JAMAIS UNE DEVINETTE : la prise rejoint la RÉSERVE « sans famille connue »
 * (`unnamedTaken`), exactement comme un rang totalement absent de `abilityLabels`. Ce n'est PAS
 * ce que fait le Go (qui traite un label sans famille comme une exclusion produit silencieuse,
 * `usage_summary_outcomes.go` : « les deux se taisent ») — divergence VOULUE côté web pour ne
 * jamais faire disparaître une prise sans le dire (décision du lot 5.7).
 */
import { REPLAY_NO_ABILITY_RANK } from '@/lib/api/types'

import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import { bump } from './equipmentGestures'
import { droppedFamilyOf, EPISODE_FAMILY_OF_POWERUP } from './gameChangers'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { EquipmentUsageTally } from './equipmentUsageLogic'

/**
 * LES FAMILLES DU BILAN « SERVI OU GÂCHÉ » (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) : les
 * six déployables (mêmes clés que `deployed`/`dropped`, tirées de `PLACEMENT_RENDER`) et les
 * deux power-ups (mêmes clés que `episodes` — `camo`/`overshield`, PAS `powerup_*`, qui nomme
 * le MÊME objet côté pose : cf. `droppedFamilyOf`). Le répulseur, le grappin et le propulseur
 * n'y figurent PAS : leur activation n'est mesurée par aucun canal ici mobilisé (P4).
 */
export const KEPT_FAMILIES: readonly string[] = [
  'wall',
  'sensor',
  'translocator_beacon',
  'shroud_screen',
  'threat_seeker',
  'repair_field',
  EQUIP_FAMILY_CAMO,
  EQUIP_FAMILY_OVERSHIELD,
]

/** Vrai pour les deux familles dont le côté « utilisé » vient des ÉPISODES, pas des poses. */
export function isEpisodeMeasuredFamily(family: string): boolean {
  return family === EQUIP_FAMILY_CAMO || family === EQUIP_FAMILY_OVERSHIELD
}

/**
 * KEPT_FAMILIES_WITH_SPAWNED_PIECE — LES FAMILLES DONT LE DÉPLOIEMENT ENGENDRE UNE PIÈCE
 * DISTINCTE sur le terrain, et donc les SEULES dont le côté « utilisé » se lit sur les POSES
 * (`deployed`) plutôt que sur les CONSOMMATIONS (`spent`).
 *
 * Transcription du MANIFESTE du titre (`config/titles/halo_infinite/mappings/
 * replay_labels.toml`, `kind = "deployed"`) — UNE DONNÉE ÉCRITE, pas une déduction — et JUMELLE
 * de `usageFamiliesWithSpawnedPiece` côté Go (`internal/games/halo_infinite/film/replay/
 * usage_summary_families.go`) : même décision produit, deux dépôts (rapport E0 du 2026-09-10,
 * lots 5.5/5.7). Le manifeste n'en désigne aujourd'hui qu'UNE : le MUR (ses deux panneaux,
 * `WALL_PANEL_IDS`). Le web ne peut pas relire le manifeste lui-même depuis ce module PUR — même
 * raison que côté Go (fonction pure du document déjà servi) — d'où la transcription, figée par
 * le test dédié (1 élément). Toute famille ajoutée côté manifeste est une décision CONSCIENTE à
 * prendre ici ET côté Go, jamais un défaut silencieux.
 */
const KEPT_FAMILIES_WITH_SPAWNED_PIECE: ReadonlySet<string> = new Set(['wall'])

/** Vrai si le déploiement de cette famille engendre une pièce distincte (le mur, et lui seul). */
export function isFamilyWithSpawnedPiece(family: string): boolean {
  return KEPT_FAMILIES_WITH_SPAWNED_PIECE.has(family)
}

/**
 * usageUsedOf — le côté « utilisé » d'une famille du bilan, pour UN compteur. TROIS canaux, et
 * l'appelant ne voit pas la différence : c'est le SEUL endroit qui traduit (cf. l'en-tête).
 * Jumeau du `usageUsedOf` Go (`usage_summary_outcomes.go`) — même nom, même rôle.
 */
export function usageUsedOf(tally: EquipmentUsageTally, family: string): number {
  if (isEpisodeMeasuredFamily(family)) return tally.episodes[family]?.count ?? 0
  if (isFamilyWithSpawnedPiece(family)) return tally.deployed[family] ?? 0
  return tally.spent[family] ?? 0
}

/**
 * familyHasAnyTrace — ce compteur porte-t-il UNE TRACE MESURÉE de cette famille ?
 *
 * LE CRITÈRE D'ENTRÉE D'UNE FAMILLE DANS LA COLONNE FUSIONNÉE (`EquipmentUsageColumns.equipment`,
 * `columnsOf`). Il est DÉLIBÉRÉMENT PLUS LARGE que les trois issues : une pose compte aussi, même
 * quand elle ne vaut plus « utilisé » depuis le lot 5.7 (sur un objet porté, une pose `deployed`
 * mesure un lâcher volontaire à mi-vie). Une famille dont le film a mesuré QUELQUE CHOSE mérite
 * sa colonne ; c'est le REMPLISSAGE de la barre, lui, qui départage les issues.
 *
 * CORRECTION C2 (revue de la vague 5, 2026-09-10) : le côté « utilisé » manquait à cette liste.
 * Elle testait `deployed` / `dropped` / `kept` et rejouait ainsi sa propre définition d'usage —
 * celle d'avant le lot 5.7. Un capteur pris puis CONSOMMÉ (ni posé, ni lâché, donc rien à garder)
 * n'ouvrait plus aucune colonne, et `equipmentUsageColumns.ts` retire le GROUPE ENTIER quand la
 * liste est vide : la vue match n'affichait plus rien de la seule famille employée. Passer par
 * `usageUsedOf` raccroche la liste à la règle canonique — une seule écriture, ici comme ailleurs.
 *
 * `dropped` se lit dans le vocabulaire des POSES (`powerup_camo`), d'où le pont `droppedFamilyOf`
 * — même lecture que la dérivation du gardé ci-dessous. `deployed`, lui, n'a jamais de membre
 * pour les deux bonus (aucun déploiement de power-up n'est mesuré) : le terme y est simplement
 * toujours nul, il n'a pas besoin d'être écarté.
 */
export function familyHasAnyTrace(tally: EquipmentUsageTally, family: string): boolean {
  return (
    usageUsedOf(tally, family) > 0 ||
    (tally.deployed[family] ?? 0) > 0 ||
    (tally.dropped[droppedFamilyOf(family)] ?? 0) > 0 ||
    (tally.kept[family] ?? 0) > 0
  )
}

/**
 * keptFamilyOf — normalise une famille BRUTE publiée par le document (vocabulaire de POSE,
 * `powerup_camo`) vers le vocabulaire `KEPT_FAMILIES` (`camo`) via le pont D5
 * (`EPISODE_FAMILY_OF_POWERUP`), puis filtre au PÉRIMÈTRE du bilan. `null` = famille CONNUE mais
 * hors bilan (grappin, propulseur, répulseur — P4) : une exclusion PRODUIT, jamais une mesure
 * manquante — c'est à l'appelant de distinguer ce cas d'un `family` absent (cf. `unnamedTaken`).
 */
function keptFamilyOf(rawFamily: string): string | null {
  const bridged = Object.prototype.hasOwnProperty.call(EPISODE_FAMILY_OF_POWERUP, rawFamily)
    ? EPISODE_FAMILY_OF_POWERUP[rawFamily as keyof typeof EPISODE_FAMILY_OF_POWERUP]
    : rawFamily
  return KEPT_FAMILIES.includes(bridged) ? bridged : null
}

/**
 * equipmentChangeFamilyOf — LA SEULE LECTURE de `abilityLabels` du bilan : la famille CANONIQUE
 * (vocabulaire `KEPT_FAMILIES`) que nomme le rang d'un `equipmentChanges`, et si ce rang est
 * NOMMÉ dans CE film.
 *
 * LES DEUX « NON » NE DISENT PAS LA MÊME CHOSE, et c'est tout l'intérêt de `named` :
 *  - `named: false` — le rang n'a AUCUN label, ou son label n'a pas de `family` (artefact
 *    antérieur au schéma 51, ou table du film incomplète). C'est une MESURE MANQUANTE : la prise
 *    rejoint la réserve `unnamedTaken`, jamais une famille devinée par une racine de libellé.
 *  - `named: true, family: null` — rang labellisé, famille CONNUE mais HORS BILAN (grappin,
 *    propulseur, répulseur — P4). C'est une EXCLUSION PRODUIT : elle se tait.
 *
 * CORRIGÉ LE 2026-09-10 (constat C3 de la revue de la vague 5). Cette fonction rendait un simple
 * `string | null` qui confondait les deux « non », si bien que son unique appelant possible
 * ([deriveKeptFromTaken]) relisait `abilityLabels` lui-même — DEUX fois — pour les distinguer :
 * la fonction n'avait plus aucun appelant de production tout en restant épinglée par ses tests
 * (anti-pattern n°1, « dead code museum » avec des tests verts qui entretiennent l'illusion). Le
 * second retour lui rend son rôle, et il n'existe désormais qu'UN SEUL chemin de lecture de la
 * famille — même forme que le jumeau Go, `equipmentOutcomeFamilyOf(labels, rank) (family, named)`
 * (`usage_summary_outcomes.go`).
 */
export function equipmentChangeFamilyOf(
  labels: ReplayDocumentReady['abilityLabels'],
  rank: number,
): { family: string | null; named: boolean } {
  const rawFamily = labels?.[String(rank)]?.family
  if (!rawFamily) return { family: null, named: false }
  return { family: keptFamilyOf(rawFamily), named: true }
}

/**
 * Un changement d'équipement retenu par le bilan : une PRISE (`taken`) ou une CONSOMMATION
 * (`spent`) de charge, avec sa famille canonique (vocabulaire `KEPT_FAMILIES`), le slot du porteur
 * et l'image où le film l'écrit (`t`).
 */
export interface EquipmentChange {
  kind: 'taken' | 'spent'
  family: string
  slot: number
  t: number
}

/**
 * readEquipmentChanges — LA SEULE LECTURE DATÉE de `equipmentChanges` : chaque prise et chaque
 * consommation que le bilan retient, dans l'ordre du document, plus le compte des objets PRIS dont
 * le rang n'a AUCUNE famille dans CE film (réserve du MATCH, `EquipmentUsage.unnamedTaken` — jamais
 * un porteur mis en cause pour une non-mesure). Le total du match et la lecture à l'instant lisent
 * ce même résultat.
 *
 * Une prise dont la table ou la famille manque (artefact ancien) rejoint la réserve — repli
 * EXPLICITE (lot 5.7), jamais une famille devinée par une racine de libellé. Une prise labellisée
 * mais hors bilan (grappin, propulseur, répulseur) est une exclusion PRODUIT (P4) : elle se tait.
 * Une consommation sur chaîne de compteur trouée (`gap`) n'est pas une identité fiable (même garde
 * que le Go, `Gap > 0` — `SpentUnreliableFrom`) ; son rang est sur `from`, jamais sur `r` (vide sur
 * un `spent`) ; celle du mur se lit sur SES poses, jamais ici (cf. `usageUsedOf`).
 */
export function readEquipmentChanges(
  doc: Pick<ReplayDocumentReady, 'equipmentChanges' | 'abilityLabels'>,
): { changes: EquipmentChange[]; unnamedTaken: number } {
  const changes: EquipmentChange[] = []
  let unnamedTaken = 0
  for (const c of doc.equipmentChanges) {
    if (c.kind === 'taken') {
      const { family, named } = equipmentChangeFamilyOf(doc.abilityLabels, c.r)
      if (!named) {
        if (c.r !== REPLAY_NO_ABILITY_RANK) unnamedTaken += 1
        continue
      }
      if (!family) continue
      changes.push({ kind: 'taken', family, slot: c.slot, t: c.t })
      continue
    }
    if (c.kind === 'spent') {
      if (c.gap) continue
      const { family } = equipmentChangeFamilyOf(doc.abilityLabels, c.from)
      if (!family || isFamilyWithSpawnedPiece(family)) continue
      changes.push({ kind: 'spent', family, slot: c.slot, t: c.t })
    }
  }
  return { changes, unnamedTaken }
}

/**
 * applyEquipmentChange — verse un changement dans le compteur de son propriétaire : une
 * consommation sur `tally.spent`, une prise sur `taken` (le scratch par famille de CE compteur,
 * jamais posé sur `EquipmentUsageTally` : ce n'est pas une grandeur affichée, seulement l'entrée de
 * `keptOf`).
 */
export function applyEquipmentChange(
  tally: EquipmentUsageTally,
  taken: Record<string, number>,
  c: EquipmentChange,
): void {
  bump(c.kind === 'spent' ? tally.spent : taken, c.family)
}

/**
 * keptOf — LE GARDÉ d'une famille : `max(0, pris - utilisé - lâché)`. L'écart résiduel mesuré par E0.4
 * (2,45 % toutes familles, médiane 0,00 %) est absorbé par le clamp à zéro plutôt qu'affiché comme un
 * gardé négatif. Seule écriture de cette règle : le total du match et la lecture à l'instant la lisent.
 */
export function keptOf(taken: number, used: number, dropped: number): number {
  return Math.max(0, taken - used - dropped)
}

/**
 * deriveKeptFromTaken — LA TROISIÈME ISSUE, en une passe sur `readEquipmentChanges` : verse les
 * prises (`taken`) ET les consommations (`spent`) par famille CANONIQUE et par compteur (via
 * `tallyOfSlotAt`, le propriétaire de la VIE qui occupe le slot à l'instant du geste), pose
 * `spent` sur le tally (canal EXPOSÉ, jumeau de `SpentByFamily` côté Go — la colonne « utilisé »
 * de la vue l'emploie aussi, via `usageUsedOf`), puis dérive
 * `tally.kept[famille] = keptOf(pris, usageUsedOf(famille), lâché)` une fois `episodes` /
 * `deployed` / `dropped` / `spent` déjà posés pour ce compteur (règle validée par E0.4).
 *
 * Rend le compte des objets PRIS dont le rang n'a AUCUNE famille dans CE film.
 */
export function deriveKeptFromTaken(
  doc: Pick<ReplayDocumentReady, 'equipmentChanges' | 'abilityLabels'>,
  tallyOfSlotAt: (slot: number, t0: number) => EquipmentUsageTally,
): number {
  const { changes, unnamedTaken } = readEquipmentChanges(doc)
  const takenByTally = new Map<EquipmentUsageTally, Record<string, number>>()
  for (const c of changes) {
    const t = tallyOfSlotAt(c.slot, c.t)
    const taken = takenByTally.get(t) ?? {}
    applyEquipmentChange(t, taken, c)
    takenByTally.set(t, taken)
  }
  // LE GARDÉ SE DÉRIVE APRÈS COUP, une fois `deployed`/`dropped`/`spent`/`episodes` posés pour
  // CE compteur : `taken` sans usage ni lâcher pour la même famille.
  for (const [t, taken] of takenByTally) {
    for (const family of KEPT_FAMILIES) {
      const takenN = taken[family] ?? 0
      if (takenN === 0) continue
      t.kept[family] = keptOf(takenN, usageUsedOf(t, family), t.dropped[droppedFamilyOf(family)] ?? 0)
    }
  }
  return unnamedTaken
}
