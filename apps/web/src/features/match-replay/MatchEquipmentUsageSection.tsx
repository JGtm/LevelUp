/**
 * MatchEquipmentUsageSection — LE BILAN D'ÉQUIPEMENT D'UN MATCH, par joueur et par équipe.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. Le rejeu 2D montre chaque geste d'équipement à l'IMAGE où
 * il a lieu : un mur posé à 3:12, une traction de grappin à 5:40. Personne ne peut en tirer
 * « qui a posé six murs » en regardant six instants séparés de trois minutes. Cette section les
 * COMPTE — et c'est tout ce qu'elle ajoute : aucune donnée nouvelle, aucun appel de plus.
 *
 * DEUX VUES, PLUS UN TABLEAU (2026-09-03, retours utilisateur sur l'onglet Chronologie). Le
 * tableau à deux niveaux d'en-tête donnait la bonne mesure dans la mauvaise forme : une grille
 * de chiffres où l'œil ne trouve ni le geste dominant ni le camp qui s'en est servi. À sa place,
 * dans la même carte :
 * PLUS DE COLONNE DE GRENADES (2026-09-13, cadrage utilisateur) : les lancers restent mesurés
 * et dessinés par le rejeu mais ne sont pas un équipement.
 * PLUS DE GROUPE « ÉTATS ACTIFS » NON PLUS (2026-09-19, décision 6 du plan d'ajustements
 * pré-v7.5) : « actif » et « utilisé » disaient la même chose — un épisode de camouflage ou de
 * surbouclier alimente DÉJÀ le côté « utilisé » de la colonne d'équipement du power-up. Restent
 * les trois issues d'une pile : Utilisé / Gardé / Lâché.
 * PLUS DE REPLI « VOIR PLUS » (même jour) : tout ce que la donnée justifie s'affiche.
 *
 * DEUX CARTES SUR LA MÊME RANGÉE depuis le 2026-09-21 (lot D du plan d'ajustements
 * supplémentaires pré-v7.5) : les deux vues ci-dessous ont chacune sa `SectionCard`, côte à
 * côte en `lg:grid-cols-2`. Leurs rendus internes n'ont PAS bougé — une maquette est en cours
 * pour leur forme, et ce lot ne touche que le chrome.
 *
 *   1. « Usages par joueur » — la grille partagée `components/charts/ValueGrid` :
 *      lignes = joueurs dans l'ordre du roster, camp par camp, filet entre les deux camps ;
 *      colonnes = grandeurs, CHACUNE AVEC SON ÉCHELLE (un mur se compare à un mur) ;
 *   2. « Part de chaque équipe » — depuis le 2026-09-21 (D20, proposition 5.A de la maquette),
 *      une PISTE ÉPAISSE par famille, LES MÊMES familles et le même ordre que les colonnes
 *      ci-dessus, toutes sur une seule échelle d'usages : segment gauche mon camp, droit le
 *      leur, compte écrit dedans, total en bout de ligne, pourcentage en infobulle.
 * Les colonnes restent DÉCIDÉES PAR LA DONNÉE (`usageColumnGroups`) : aucune liste en dur.
 *
 * ELLE VIT DANS `match-replay/` ET NON DANS `match-view/`, à la différence de la courbe de score
 * qui la précède dans l'onglet. La raison est le VOCABULAIRE : chaque libellé qu'elle écrit —
 * familles de pose, familles d'état actif, types de grenade, socles de bonus — appartient au
 * dictionnaire du rejeu (`REPLAY_TEXT`) et aux tables du document. Le poser dans `match-view`
 * aurait forcé soit un import du dictionnaire voisin, soit une seconde table de noms qui
 * divergerait au premier ajout du manifeste du titre. Le sens de l'import est déjà établi :
 * `MatchScoreCurveChart` lit `match-replay/queries` depuis `match-view`.
 *
 * DOUBLE PORTE, comme la courbe de score : pas d'artefact (le cas de la quasi-totalité des
 * matchs) OU aucune grandeur mesurée -> RIEN. Pas de cadre vide, pas de « bientôt disponible ».
 * La MÊME clé de cache que la courbe (`useMatchReplay`, gaté par `header.replay_available`) :
 * les deux blocs de l'onglet partagent un seul téléchargement.
 *
 * CE QUE L'ÉCRAN DIT DE SA PROPRE MESURE, et il doit le dire :
 *   - les gestes que le film mesure sans en nommer l'auteur ou l'origine ne comptent dans
 *     aucune des deux vues : ils se disent en UNE PHRASE dans l'infobulle du TITRE de la carte.
 *     AUCUN TEXTE DE PIED (2026-09-14, décision utilisateur) — le pied de carte a porté
 *     successivement le paragraphe répulseur/propulseur (retiré le 2026-09-13), la ligne des
 *     socles de bonus vidés, les dénominateurs de couverture et les deux réserves ; il ne porte
 *     plus rien du tout. Les réserves qui survivent vivent au survol : celle du répulseur et du
 *     propulseur dans `groupEquipmentHint`, celle des gestes hors vues sur le titre. Les socles
 *     de bonus vidés se lisent dans le bloc « Contrôle des armes spéciales », juste en dessous
 *     dans l'onglet « Contrôle » ;
 *
 * COULEURS. Dans la grille, les familles de geste prennent la table d'encres de
 * `equipmentUsageChart` (jetons sémantiques, jamais un hex) ; dans la vue des parts la couleur
 * dit LE CAMP et rien d'autre. Les camps prennent `teamTokenCssVar` — les jetons `team-ally` /
 * `team-enemy` que les réglages d'accessibilité surchargent, et NON la cascade d'identité de
 * `teamColor.ts` (cf. l'en-tête de `match-view/teamSeriesColor.ts` pour la frontière).
 *
 * Aucun calcul ici : tout vient de `equipmentUsageLogic` (les mesures),
 * `equipmentUsageColumns` (les colonnes et leurs noms) et `equipmentUsageChart` (la projection).
 */
import { useCallback, useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { ValueGrid } from '@/components/charts/ValueGrid'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'

import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import { StackedTrack, type StackedTrackSegment } from '@/components/charts/StackedTrack'

import {
  buildUsageFamilyBars,
  buildUsageGrid,
  usageGroupColor,
  type UsageFamilyBars,
} from './model/equipmentUsageChart'
import { uniqueUsageGroups, usageColumnGroups } from './model/equipmentUsageColumns'
import { buildEquipmentUsage, tallyTotal, type EquipmentUsage } from './model/equipmentUsageLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import type { ReplayText } from './i18n/i18nContract'
import { useMatchReplay } from '../../lib/replay/queries'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — le même gate que le lien rejeu et que la courbe de score. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  locale: ReplayLocale
}

export function MatchEquipmentUsageSection({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  locale,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const u = t.equipmentUsage
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const usage = useMemo(() => (data ? buildEquipmentUsage(data, board) : null), [data, board])
  // PLUS DE REPLI DEPUIS LE 2026-09-19 (plan d'ajustements pré-v7.5, lot 2) : toutes les
  // colonnes que la donnée justifie s'affichent au chargement. Le bouton « Voir plus (N) /
  // Replier » et la partition « game changers » qui l'alimentait ont été retirés — la même
  // décision que le bloc voisin (« Contrôle des armes spéciales ») avait déjà prise le
  // 2026-09-13 : une carte qui cache sa mesure par défaut ne se lit pas.
  const { groups, familles } = useUsageGroups(usage, t)
  const reserve = useMemo(() => usageReserve(usage), [usage])
  const meRow = useMemo(() => board.find((r) => r.is_me), [board])
  const meSide = meRow?.team_side ?? null

  const teamLabel = useCallback(
    (side: string | null) =>
      resolveTeamLabel(
        side ? board.filter((r) => (r.team_side ?? '') === side) : [],
        side,
        t,
      ),
    [board, t],
  )
  // « Allié » = du côté du joueur de la page. Sans `is_me` au tableau des scores, ou pour un
  // joueur que le film a vu vivre sans ligne de scoreboard, le camp est INCONNU (null) : encre
  // neutre, jamais l'une des deux couleurs d'équipe (même règle que `ReplayTeamHeader`).
  const teamAccent = useCallback(
    (side: string | null) =>
      teamTokenCssVar(side == null || meSide == null ? null : side === meSide),
    [meSide],
  )

  const grid = useMemo(
    () =>
      buildUsageGrid({
        teams: usage?.byTeam ?? [],
        groups,
        meXUID: meRow?.xuid ?? null,
        teamLabel,
        teamAccent,
        tipFmt: t.equipmentUsage.gridTipFmt,
      }),
    [usage, groups, meRow, teamLabel, teamAccent, t],
  )
  const barres = useMemo(
    () =>
      buildUsageFamilyBars({
        teams: usage?.byTeam ?? [],
        groups,
        allySide: meSide,
        teamLabel,
        teamAccent,
      }),
    [usage, groups, meSide, teamLabel, teamAccent],
  )

  // Double porte : pas d'artefact, ou rien de mesuré -> rien du tout.
  if (!usage?.hasData) return null

  // DEUX CARTES SUR LA MÊME RANGÉE depuis le 2026-09-21 (lot D) : « par joueur » et « part de
  // chaque équipe » répondaient à deux questions dans une seule carte, l'une sous l'autre, et
  // la page s'allongeait d'autant. Les RENDUS INTERNES sont inchangés (une maquette est en
  // cours pour leur forme) : seul le chrome se dédouble. La RÉSERVE est sur les deux titres —
  // elle vaut pour les deux vues, et une carte qui ne la porterait pas mentirait par omission.
  const reserveTip = reserve > 0 ? <p>{u.coverageReserveFmt(reserve)}</p> : null
  const titreDeCarte = titleWithInfo(reserveTip)

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {groups.length > 0 && (
        <SectionCard
          title={u.viewByPlayer}
          label={u.viewByPlayer}
          titleAdornment={titreDeCarte}
        >
          <div className="px-3 pb-3 pt-3">
            <ValueGrid model={grid} />
            <ChartLegend
              className="pt-2"
              items={familles.map((g) => ({
                key: g.key,
                label: g.label,
                color: usageGroupColor(g.key),
              }))}
            />
          </div>
        </SectionCard>
      )}
      {barres.rows.length > 0 && (
        <SectionCard
          title={u.viewTeamShare}
          label={u.viewTeamShare}
          titleAdornment={titreDeCarte}
        >
          <div className="px-3 pb-3 pt-3">
            <UsageFamilyTracks model={barres} t={t} />
          </div>
        </SectionCard>
      )}
    </div>
  )
}

/**
 * useUsageGroups — les colonnes réellement rendues : TOUTES celles que la donnée justifie,
 * dans l'ordre écrit de `usageColumnGroups` (plus de repli depuis le 2026-09-19).
 *
 * `familles` sert LA SEULE LÉGENDE DE LA GRILLE depuis le 2026-09-21 (lot K) : elle raisonne
 * par FAMILLE DE GESTE, une occurrence par groupe (cf. `uniqueUsageGroups`). La vue 2, elle,
 * est passée aux COLONNES (`groups`) — c'est tout l'objet de la proposition 5.A.
 */
function useUsageGroups(usage: EquipmentUsage | null, t: ReplayText) {
  const groups = useMemo(() => (usage ? usageColumnGroups(usage, t) : []), [usage, t])
  const familles = useMemo(() => uniqueUsageGroups(groups), [groups])
  return { groups, familles }
}

/**
 * UsageFamilyTracks — la vue 2 depuis le 2026-09-21 (D20, proposition 5.A de la maquette).
 *
 * UNE PISTE ÉPAISSE PAR FAMILLE, TOUTES SUR LA MÊME ÉCHELLE. Jusqu'ici la vue rendait une
 * barre 100 % par GROUPE de colonnes : « Grappin » et « Équipement » faisaient la même
 * longueur en valant 24 et 38 gestes, et la seconde mêlait murs, capteurs, propulseurs,
 * surbouclier et camouflage — le lecteur qui venait de lire « quatre murs » dans la grille de
 * gauche ne retrouvait AUCUNE de ses colonnes à droite. Les lignes sont maintenant les
 * colonnes de la grille, même liste et même ordre, et leur longueur est le nombre de gestes
 * rapporté à la borne commune (le plus gros total) : le volume et le rapport de force se
 * lisent dans la MÊME marque, sans pourcentage à traduire.
 *
 * LE COMPTE EST ÉCRIT DANS LE SEGMENT quand il tient, le total en bout de ligne, et le
 * POURCENTAGE de la famille reste en infobulle — il ne se lit plus dans la longueur, qui dit
 * désormais le volume (cf. `shareTipFmt`).
 *
 * La piste est `components/charts/StackedTrack`, celle de l'encart cible : la part de la
 * borne qu'aucun camp n'emploie reste en fond neutre, et c'est elle qui donne l'échelle à
 * l'œil. Aucune pastille de famille ici — l'encre de famille est celle de la GRILLE, et sur
 * cette vue la couleur dit le CAMP (jetons `team-ally` / `team-enemy`, `teamTokenCssVar`).
 */
function UsageFamilyTracks({ model, t }: { model: UsageFamilyBars; t: ReplayText }) {
  const u = t.equipmentUsage
  return (
    <div className="min-w-0 space-y-2">
      {model.rows.map((row) => (
        <div
          key={row.key}
          className="grid grid-cols-[minmax(0,140px)_minmax(0,1fr)_auto] items-center gap-3"
          data-testid={`usage-ligne-${row.key}`}
        >
          <HeaderLabelTooltip text={row.hint} focusable>
            <span className="block truncate text-right text-xs">{row.label}</span>
          </HeaderLabelTooltip>
          <StackedTrack
            segments={row.segments.map<StackedTrackSegment>((seg) => {
              // La MÊME phrase au survol et pour le lecteur d'écran : le segment ne porte que
              // son compte quand il est large, l'exacte mesure se dit ici.
              const tip = u.shareTipFmt(seg.label, row.label, seg.count, row.total, seg.percent)
              return {
                key: seg.side ?? 'sans-equipe',
                widthPct: seg.widthPct,
                color: seg.accent,
                label: String(seg.count),
                tooltip: tip,
                ariaLabel: tip,
              }
            })}
            ariaLabel={row.label}
            testId={`usage-famille-${row.key}`}
          />
          <span className="text-2xs tabular-nums text-muted-foreground">{row.total}</span>
        </div>
      ))}
      <ChartLegend
        className="pt-1"
        items={model.legend.map((team) => ({
          key: team.side ?? 'sans-equipe',
          label: team.label,
          color: team.accent,
        }))}
      />
    </div>
  )
}

/**
 * unknownOriginPlacements — LES POSES D'ORIGINE INCONNUE : la somme des poses `famille/unknown`
 * de `coverage.placements.byFamilyOrigin` — ~5 % du parc mesurés par E0 (681/11 438).
 * `origin: 'unknown'` n'est ni un déploiement ni un lâcher (schéma antérieur au 10, ou pose sans
 * poseur mesuré) : elle n'est comptée dans AUCUNE des deux vues, et pourtant elle a eu lieu.
 */
function unknownOriginPlacements(cov: EquipmentUsage['coverage']): number {
  let total = 0
  for (const [key, n] of Object.entries(cov.placementsByFamilyOrigin)) {
    if (key.endsWith('/unknown')) total += n
  }
  return total
}

/**
 * usageReserve — CE QUE LES DEUX VUES NE COMPTENT PAS, en un seul nombre.
 *
 * Deux mesures, une seule réserve : les gestes sans propriétaire (slot n'appartenant à aucun
 * joueur, ou poseur non mesuré) et les poses d'origine inconnue. Elles ont eu lieu, le film ne
 * les rattache à personne, elles ne peuvent donc entrer ni dans la grille par joueur ni dans la
 * part par équipe. La réserve NE SE CACHE PAS (décision utilisateur 2026-09-09) : depuis le
 * 2026-09-14 elle se dit dans l'INFOBULLE DU TITRE, en une phrase, et non plus en pied de carte
 * (l'utilisateur ne veut aucun texte de pied sous ce bloc).
 *
 * `unnamedTaken` (objets pris dont le rang n'a pas de famille connue) n'y entre PAS : décision
 * utilisateur du 2026-09-09, ces objets ne s'affichent pas dans l'interface — le compteur reste
 * publié par la logique pour l'outillage d'investigation.
 */
function usageReserve(usage: EquipmentUsage | null): number {
  if (!usage) return 0
  return tallyTotal(usage.unattributed) + unknownOriginPlacements(usage.coverage)
}
