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
 *   1. « Nombre de gestes par joueur » — la grille partagée `components/charts/ValueGrid` :
 *      lignes = joueurs dans l'ordre du roster, camp par camp, filet entre les deux camps ;
 *      colonnes = grandeurs, CHACUNE AVEC SON ÉCHELLE (un mur se compare à un mur) ;
 *   2. « Part de chaque équipe » — une barre 100 % par famille de geste, le
 *      compte brut ET le pourcentage écrits dans le segment.
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
 * COULEURS. Les familles de geste prennent la table d'encres de `equipmentUsageChart` (jetons
 * sémantiques, jamais un hex) ; les camps prennent `teamTokenCssVar` — les jetons `team-ally` /
 * `team-enemy` que les réglages d'accessibilité surchargent, et NON la cascade d'identité de
 * `teamColor.ts` (cf. l'en-tête de `match-view/teamSeriesColor.ts` pour la frontière).
 *
 * Aucun calcul ici : tout vient de `equipmentUsageLogic` (les mesures),
 * `equipmentUsageColumns` (les colonnes et leurs noms) et `equipmentUsageChart` (la projection).
 */
import { useCallback, useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { Tooltip } from '@/components/ui/tooltip'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import {
  buildUsageGrid,
  buildUsageShares,
  usageGroupColor,
  type UsageShareRow,
} from './model/equipmentUsageChart'
import {
  uniqueUsageGroups,
  usageColumnGroups,
  type UsageColumnGroup,
} from './model/equipmentUsageColumns'
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
  const shares = useMemo(
    () => buildUsageShares({ teams: usage?.byTeam ?? [], groups: familles, teamLabel, teamAccent }),
    [usage, familles, teamLabel, teamAccent],
  )

  // Double porte : pas d'artefact, ou rien de mesuré -> rien du tout.
  if (!usage?.hasData) return null

  return (
    <SectionCard
      title={t.equipmentUsage.title}
      label={t.equipmentUsage.title}
      titleAdornment={(label) => (
        // LA RÉSERVE EST AU SURVOL DU TITRE, et nulle part ailleurs : sans réserve à dire,
        // `HeaderLabelTooltip` rend le libellé nu (aucun nœud superflu).
        <HeaderLabelTooltip
          text={reserve > 0 ? u.coverageReserveFmt(reserve) : undefined}
          focusable
        >
          <span>{label}</span>
        </HeaderLabelTooltip>
      )}
    >
      <UsageViews grid={grid} groups={groups} familles={familles} shares={shares} t={t} />
    </SectionCard>
  )
}

/**
 * useUsageGroups — les colonnes réellement rendues : TOUTES celles que la donnée justifie,
 * dans l'ordre écrit de `usageColumnGroups` (plus de repli depuis le 2026-09-19).
 *
 * `familles` sert la légende et la vue 2, qui raisonnent PAR FAMILLE DE GESTE : un groupe n'y a
 * qu'une occurrence (cf. `uniqueUsageGroups`).
 */
function useUsageGroups(usage: EquipmentUsage | null, t: ReplayText) {
  const groups = useMemo(() => (usage ? usageColumnGroups(usage, t) : []), [usage, t])
  const familles = useMemo(() => uniqueUsageGroups(groups), [groups])
  return { groups, familles }
}

/**
 * UsageViews — le corps de la carte : les deux vues empilées, chacune gardée par son contenu.
 *
 * Aucune grandeur mesurée : les deux vues n'ont rien à dessiner et la carte entière est déjà
 * fermée en amont par la double porte du composant.
 */
function UsageViews({
  grid,
  groups,
  familles,
  shares,
  t,
}: {
  grid: ReturnType<typeof buildUsageGrid>
  groups: UsageColumnGroup[]
  familles: UsageColumnGroup[]
  shares: UsageShareRow[]
  t: ReplayText
}) {
  return (
    <div className="space-y-5 px-3 pb-3 pt-3">
      {groups.length > 0 && (
        <section aria-label={t.equipmentUsage.viewByPlayer}>
          <ViewTitle>{t.equipmentUsage.viewByPlayer}</ViewTitle>
          <ValueGrid model={grid} />
          <ChartLegend
            className="pt-2"
            items={familles.map((g) => ({
              key: g.key,
              label: g.label,
              color: usageGroupColor(g.key),
            }))}
          />
        </section>
      )}
      {shares.length > 0 && (
        <section aria-label={t.equipmentUsage.viewTeamShare}>
          <ViewTitle>{t.equipmentUsage.viewTeamShare}</ViewTitle>
          <UsageTeamShares rows={shares} t={t} />
        </section>
      )}
    </div>
  )
}

/** Le titre d'une vue à l'intérieur de la carte : les deux vues répondent à deux questions. */
function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

/**
 * UsageTeamShares — la vue 2 : une barre 100 % par famille de geste, un segment par camp.
 *
 * LE COMPTE BRUT ET LE POURCENTAGE SONT ÉCRITS DANS LE SEGMENT (demande utilisateur du
 * 2026-09-03) : une part sans son compte laisse croire qu'un 4-1 et un 40-10 racontent la même
 * partie. À gauche, le nom de la famille, son total, et sa pastille — la même encre que la
 * colonne correspondante de la vue 1.
 */
function UsageTeamShares({ rows, t }: { rows: UsageShareRow[]; t: ReplayText }) {
  return (
    // PLUS DE DÉFILEMENT HORIZONTAL (2026-09-19, lot 2) : la vue tenait derrière un
    // `overflow-x-auto` et une largeur plancher de 420 px, donc une barre de défilement sous la
    // carte dès qu'elle était à l'étroit. Les deux colonnes (nom de famille, rail) se
    // répartissent maintenant la largeur disponible, et le nom se tronque plutôt que de pousser
    // le rail hors du cadre (`min-w-0` autorise la grille à passer sous la taille du contenu).
    <div className="min-w-0">
      <div className="space-y-2.5">
        {rows.map((row) => {
          const segTotal = row.segments.reduce((a, s) => a + s.count, 0)
          return (
          <div
            key={row.key}
            className="grid grid-cols-[minmax(0,158px)_minmax(0,1fr)] items-center gap-3.5"
          >
            <div className="flex items-center justify-end gap-2 text-xs">
              <HeaderLabelTooltip text={row.hint} focusable>
                <span className="truncate text-right">{row.label}</span>
              </HeaderLabelTooltip>
              <span className="text-muted-foreground tabular-nums">{row.total}</span>
              <span
                className="h-2.5 w-2.5 flex-none"
                style={{ backgroundColor: row.color }}
                aria-hidden="true"
              />
            </div>
            <div className="flex h-[22px]">
              {/* La largeur est portée par l'ITEM du flex, en `calc(%)` — jamais un flexGrow
                  sur le contenu d'un Tooltip : son wrapper garde flex-grow 0 et les segments
                  se dimensionneraient à leur texte, pas à leurs comptes (revue adversariale
                  2026-09-05 ; pattern : MatchPadControlSection). Et comme là-bas, le libellé
                  ne s'écrit que si le segment est assez large pour le porter — l'infobulle
                  garde toujours la valeur exacte. */}
              {row.segments.map((seg) => {
                const tip = t.equipmentUsage.shareTipFmt(seg.label, row.label, seg.count, row.total)
                const fraction = segTotal > 0 ? seg.count / segTotal : 0
                // `text-white` : le libellé est posé SUR l'aplat du camp, quelle que soit la
                // palette réglée — ce n'est pas une couleur sémantique mais le contraste d'un
                // texte dans un aplat (même usage que `MatchNemesisCards`).
                return (
                  <div
                    key={seg.side ?? 'sans-equipe'}
                    className="mr-[2px] h-full last:mr-0"
                    style={{ width: `calc(${fraction * 100}% - 2px)` }}
                  >
                    <Tooltip content={tip} className="h-full w-full">
                      <div
                        className="flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap px-1 text-3xs font-semibold text-white"
                        style={{ backgroundColor: seg.accent }}
                        tabIndex={0}
                        role="img"
                        aria-label={tip}
                      >
                        {fraction >= 0.11 ? `${seg.count} · ${seg.percent} %` : ''}
                      </div>
                    </Tooltip>
                  </div>
                )
              })}
            </div>
          </div>
          )
        })}
      </div>
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
