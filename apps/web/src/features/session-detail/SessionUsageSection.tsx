/**
 * SessionUsageSection — LES TROIS BLOCS « usages d'équipement, armes spéciales et objectifs »
 * de la page Sessions (chantier session-usage S3, GO utilisateur 2026-09-05 sur la
 * voie de repli : grammaire du handoff §1 + modèle de la vue match livrée).
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.usage`, attaché
 * par le service Go — aucune query nouvelle, aucun appel de plus). Le contexte
 * Solo/Escouade est résolu EN AMONT par le serveur : le bloc porte `squad_players`
 * et des lignes `squad` par grandeur — vides en solo. À l'écran, l'escouade ajoute
 * une ligne par coéquipier dans les grilles et découpe la piste du lobby par
 * joueur ; le solo garde moi + les agrégats.
 *
 * TROIS CARTES (le gabarit `SectionCard` de la vue match) :
 *   1. « Usages d'équipement » — grille alignée des cadences (ValueGrid), jauges de
 *      parts avec parité, bandes de régularité match par match ;
 *   2. « Contrôle des armes spéciales » — jauges par famille d'arme puis leur TOTAL,
 *      PISTE DU LOBBY découpée par joueur (hachuré = eux, anonyme), bande de
 *      régularité ; ramassages non attribués et bonus ANONYMES en pied ;
 *   3. « Objectifs par rôle et par famille » — jauges par rôle, grille famille ×
 *      rôle, et (escouade) grille joueur × rôle. Son scope est INDÉPENDANT de la
 *      couverture des films.
 *
 * « Matchs mesurés N/M » est TOUJOURS visible (bandeau de titre). Bloc indisponible →
 * carte d'état vide avec la raison, jamais un crash ni un bloc fantôme.
 *
 * CE QUI N'EST PLUS ÉCRIT SOUS LES GRAPHES (revue de lisibilité 2026-09-09,
 * `.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md`) : la lecture des barres, le trait
 * de parité, les cases de régularité et le corollaire du §4 (épisode actif ≠ bonus
 * ramassé, jamais additionnés) vivent dans UNE aide par carte, portée par le libellé du
 * titre (`cardTitleAdornment`). Le pied ne garde que des CHIFFRES — un chiffre ne se
 * survole pas. Ne pas re-déverser de méthode dans le corps : c'est exactement le pavé
 * qui a été retiré.
 *
 * Aucun calcul ici : projections dans `usageLogic.ts` / `usageGrids.ts`, formes
 * dans `SessionUsageForms.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { tokenCssVar } from '@/lib/accessibility'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'
import type { SessionUsageBlock, SessionUsageMetric } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { UsageGaugeGrid, UsageLobbyTrack, UsageRegularityBand } from './SessionUsageForms'
import { USAGE_TEXT, powerupLabel, roleLabel, type UsageText } from './usageI18n'
import {
  buildCadenceGrid,
  buildObjectiveFamilyGrid,
  buildSquadRoleGrid,
  usagePlayerInk,
  type UsageGridInks,
} from './usageGrids'
import {
  USAGE_METRIC_TOKENS,
  buildGaugeRow,
  buildLobbyTrack,
  buildRegularityBand,
  equipmentMetrics,
  formatUsageRate,
  metricKind,
  metricLabel,
  padMetric,
  roleToken,
  sortRoles,
  teamOfLobbyParityPct,
  usageAvailability,
  type UsageGaugeRowModel,
} from './usageLogic'

interface Props {
  /** Le bloc `usage` de la réponse — absent (vieux serveur) : rien ne se rend. */
  usage: SessionUsageBlock | null | undefined
  /** Libellé du joueur de la page (slug de la route — l'identité des lignes « moi »). */
  meLabel: string
  /**
   * Colonne divisée (drawer de comparaison ouvert) : les DEUX colonnes passent en
   * compact, jamais une seule — deux rendus différents côte à côte ne se comparent pas.
   *
   * Ce que le compact retire, et pourquoi : la bande de régularité, la piste du lobby et
   * les grilles famille × rôle. Toutes trois sont des formes LARGES qui ne feraient que
   * défiler dans une demi-colonne, et aucune n'est ce qu'on vient comparer — on vient
   * comparer des PARTS et des CADENCES, qui restent. Rien n'est retiré du calcul : le
   * même bloc déplié en pleine largeur les montre toutes.
   */
  compact?: boolean
}

/** Le titre d'une vue à l'intérieur d'une carte (même gabarit que la vue match). */
function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

/**
 * Le bandeau de titre d'une carte : le libellé PORTEUR DE SON AIDE, puis « Matchs
 * mesurés N/M ».
 *
 * L'AIDE EST SUR LE LIBELLÉ, PAS SUR UNE ICÔNE ⓘ : convention du dépôt depuis V73-L2
 * 2.4c (`lib/table/columnMeta`), et c'est elle qui rend le pied de carte inutile —
 * lecture des barres, trait de parité, cases de régularité et réserves de mesure y sont
 * réunies au lieu d'occuper trois paragraphes sous les graphes (D5).
 */
function cardTitleAdornment(measured: string, hint: string) {
  return (label: string) => (
    <span className="flex items-baseline gap-2">
      <HeaderLabelTooltip text={hint} focusable>
        <span>{label}</span>
      </HeaderLabelTooltip>
      <span className="text-3xs font-medium normal-case text-muted-foreground tabular-nums">
        {measured}
      </span>
    </span>
  )
}

/** Les encres partagées des grilles : colonnes par famille/rôle, lignes par joueur. */
function useGridInks(): UsageGridInks {
  return useMemo(
    () => ({
      columnColor: (key: string) => tokenCssVar(USAGE_METRIC_TOKENS[metricKind(key)]),
      rowAccent: (kind, squadIndex) =>
        kind === 'aggregate' ? undefined : usagePlayerInk(kind, squadIndex),
    }),
    [],
  )
}

export function SessionUsageSection({ usage, meLabel, compact = false }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = USAGE_TEXT[locale]
  const availability = usageAvailability(usage, t)

  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard
          title={t.blockUnavailableTitle}
          label={t.blockUnavailableTitle}
          titleAdornment={
            usage.available
              ? cardTitleAdornment(
                  t.measuredFmt(usage.matches_measured, usage.matches_total),
                  t.cardHintEquipment,
                )
              : undefined
          }
        >
          <p className="px-3 pb-3 pt-3 text-sm text-muted-foreground">{availability.message}</p>
        </SectionCard>
        {/* Les objectifs vivent HORS films : ils s'affichent même sans match mesuré. */}
        {usage.available && (
          <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
        )}
      </>
    )
  }

  return (
    <>
      <EquipmentCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
      <PadControlCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
      <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} compact={compact} />
    </>
  )
}

interface CardProps {
  usage: SessionUsageBlock
  meLabel: string
  t: UsageText
  locale: Locale
  /** Colonne divisée : les formes larges sont retirées (cf. `Props.compact`). */
  compact: boolean
}

/** Les jauges d'une liste de grandeurs, contre les trois parités du bloc. */
function metricGaugeRows(
  metrics: SessionUsageMetric[],
  usage: SessionUsageBlock,
  t: UsageText,
  locale: Locale,
): UsageGaugeRowModel[] {
  const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
  return metrics.map((m) =>
    buildGaugeRow({
      key: m.key,
      label: metricLabel(m.key, t),
      shares: m,
      teamParityPct: usage.team_parity_pct,
      lobbyParityPct: usage.lobby_parity_pct,
      teamOfLobbyParityPct: teamOfLobby,
      t,
      locale,
    }),
  )
}

/**
 * La légende de comptage d'une bande : les matchs au-dessus de la parité D'ÉQUIPE.
 *
 * SEULEMENT L'ÉQUIPE, et c'est la même parité que celle qui TEINTE les cases
 * (`buildRegularityBand` compare à `team_parity_pct`). Le compte « lobby » qui suivait
 * était vrai mais orphelin : aucune case de la bande ne le représentait, et il doublait
 * la longueur d'une légende posée à droite de chaque ligne (D5).
 */
function bandAboveCaption(m: SessionUsageMetric, measured: number, t: UsageText): string {
  if (m.matches_above_team_parity == null) return t.bandAboveFmt(t.notMeasured)
  return t.bandAboveFmt(`${m.matches_above_team_parity}/${measured}`)
}

/** Bloc 1 — usages d'équipement : cadences, parts, régularité. */
function EquipmentCard({ usage, meLabel, t, locale, compact }: CardProps) {
  const inks = useGridInks()
  const metrics = useMemo(() => equipmentMetrics(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const cadenceGrid = useMemo(
    () => buildCadenceGrid({ metrics, squadPlayers, meLabel, t, locale, ...inks }),
    [metrics, squadPlayers, meLabel, t, locale, inks],
  )
  const gaugeRows = useMemo(() => metricGaugeRows(metrics, usage, t, locale), [metrics, usage, t, locale])
  if (metrics.length === 0) return null

  return (
    <SectionCard
      title={t.blockEquipment}
      label={t.blockEquipment}
      titleAdornment={cardTitleAdornment(
        t.measuredFmt(usage.matches_measured, usage.matches_total),
        t.cardHintEquipment,
      )}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {cadenceGrid && (
          <section aria-label={t.viewCadences}>
            <ViewTitle>{t.viewCadences}</ViewTitle>
            <ValueGrid model={cadenceGrid} />
          </section>
        )}
        {/* Pas de ViewTitle ici : la grille de jauges porte son propre en-tête, qui NOMME
            le dénominateur rendu (« Ma part dans mon équipe ») — un titre de vue au-dessus
            répéterait la même chose en moins précis. L'`aria-label` garde le nom de la vue. */}
        <section aria-label={t.viewShares}>
          <UsageGaugeGrid rows={gaugeRows} t={t} />
        </section>
        {!compact && (
        <section aria-label={t.viewRegularity}>
          <ViewTitle>{t.viewRegularity}</ViewTitle>
          <div className="space-y-1.5">
            {metrics.map((m) => (
              <UsageRegularityBand
                key={m.key}
                label={metricLabel(m.key, t)}
                cells={buildRegularityBand(m.per_match, usage.team_parity_pct, t, locale)}
                caption={bandAboveCaption(m, usage.matches_measured, t)}
              />
            ))}
          </div>
        </section>
        )}
      </div>
    </SectionCard>
  )
}

/** Bloc 2 — armes spéciales : familles nommées, leur total, bonus anonymes en pied. */
function PadControlCard({ usage, meLabel, t, locale, compact }: CardProps) {
  const pad = useMemo(() => padMetric(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  // LES FAMILLES D'ABORD, LE TOTAL ENSUITE (D6). `pad_pickups` est exactement la somme
  // des familles : posé en tête et à leur niveau, il se lisait comme une grandeur de
  // plus. En pied et marqué `isTotal`, il redevient ce qu'il est — un total.
  const gaugeRows = useMemo(() => {
    const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
    const rows = (usage.pad_families ?? []).map((fam) =>
      buildGaugeRow({
        key: `family-${fam.family_key}`,
        // Le NOM vient du serveur (catalogue d'armes du titre, résolu à la requête).
        // Absent = famille hors catalogue : la CLÉ s'affiche, jamais un nom approchant
        // (même règle que le catalogue du rejeu — D7).
        label: fam.family_label || t.padFamilyFmt(fam.family_key),
        shares: fam,
        teamParityPct: usage.team_parity_pct,
        lobbyParityPct: usage.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
    if (pad) {
      rows.push(
        ...metricGaugeRows([pad], usage, t, locale).map((row) => ({ ...row, isTotal: true })),
      )
    }
    return rows
  }, [pad, usage, t, locale])
  const track = useMemo(
    () =>
      pad
        ? buildLobbyTrack({ meLabel, shares: pad, squadPlayers, squadShares: pad.squad, t, locale })
        : null,
    [pad, meLabel, squadPlayers, t, locale],
  )
  const powerups = usage.powerup_pickups ?? []
  if (pad == null && gaugeRows.length === 0 && powerups.length === 0) return null

  return (
    <SectionCard
      title={t.blockPadControl}
      label={t.blockPadControl}
      titleAdornment={cardTitleAdornment(
        t.measuredFmt(usage.matches_measured, usage.matches_total),
        t.cardHintPadControl,
      )}
      footer={<PadControlFootnotes usage={usage} pad={pad} t={t} locale={locale} />}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {gaugeRows.length > 0 && (
          <section aria-label={t.viewShares}>
            <UsageGaugeGrid rows={gaugeRows} t={t} />
          </section>
        )}
        {track && !compact && (
          <section aria-label={t.viewLobbyTrack}>
            <ViewTitle>{t.viewLobbyTrack}</ViewTitle>
            <UsageLobbyTrack segments={track} label={t.viewLobbyTrack} />
          </section>
        )}
        {pad?.per_match != null && pad.per_match.length > 0 && !compact && (
          <section aria-label={t.viewRegularity}>
            <ViewTitle>{t.viewRegularity}</ViewTitle>
            <UsageRegularityBand
              label={metricLabel(pad.key, t)}
              cells={buildRegularityBand(pad.per_match, usage.team_parity_pct, t, locale)}
              caption={bandAboveCaption(pad, usage.matches_measured, t)}
            />
          </section>
        )}
      </div>
    </SectionCard>
  )
}

/**
 * Le pied du bloc 2 : ce que les formes ne PEUVENT PAS porter — les ramassages que la
 * mesure n'attribue à personne, et les bonus qui ne sont attribuables à personne.
 *
 * LA RÈGLE DE TRI EST « CHIFFRE OU MÉTHODE » (D5) : ce qui est un CHIFFRE reste au pied
 * — un chiffre ne se survole pas ; ce qui était de la MÉTHODE (le corollaire « épisode
 * actif ≠ bonus ramassé ») est parti dans l'aide d'en-tête. Le paragraphe de quatre
 * lignes devient au plus trois lignes de comptes. Rien à dire (session entièrement
 * attribuée, sans bonus, sans cadence) → pas de pied du tout.
 */
function PadControlFootnotes({
  usage,
  pad,
  t,
  locale,
}: {
  usage: SessionUsageBlock
  pad: SessionUsageMetric | null
  t: UsageText
  locale: Locale
}) {
  const powerups = usage.powerup_pickups ?? []
  const unnamed = usage.pad_unnamed_total ?? 0
  if (unnamed <= 0 && powerups.length === 0 && pad == null) return undefined
  const detail = powerups
    .map((p) => t.powerupDetailFmt(powerupLabel(p.family_key, t), p.occupations))
    .join(' · ')
  return (
    <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
      {unnamed > 0 && <p>{t.padUnnamedFmt(unnamed)}</p>}
      {powerups.length > 0 && <p>{`${t.powerupLine} : ${detail}`}</p>}
      {pad != null && (
        <p>
          {t.padCadenceFmt(
            formatUsageRate(pad.player_per_10min, locale),
            formatUsageRate(pad.team_per_10min, locale),
            formatUsageRate(pad.lobby_per_10min, locale),
          )}
        </p>
      )}
    </div>
  )
}

/** Bloc 3 — objectifs par rôle et famille (scope indépendant des films). */
function ObjectivesCard({ usage, meLabel, t, locale, compact }: CardProps) {
  const inks = useGridInks()
  const obj = usage.objectives
  const roles = useMemo(() => sortRoles(obj?.roles), [obj])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const roleInks = useMemo(
    () => ({ ...inks, columnColor: (key: string) => tokenCssVar(roleToken(key)) }),
    [inks],
  )
  const gaugeRows = useMemo(() => {
    if (obj == null) return []
    const teamOfLobby = teamOfLobbyParityPct(obj.team_size_avg, obj.lobby_size_avg)
    return roles.map((r) =>
      buildGaugeRow({
        key: r.role,
        label: roleLabel(r.role, t),
        shares: r,
        isDuration: r.is_duration === true,
        teamParityPct: obj.team_parity_pct,
        lobbyParityPct: obj.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
  }, [obj, roles, t, locale])
  const familyGrid = useMemo(
    () => buildObjectiveFamilyGrid({ families: obj?.families ?? [], t, locale, ...roleInks }),
    [obj, t, locale, roleInks],
  )
  const squadGrid = useMemo(
    () => buildSquadRoleGrid({ roles, squadPlayers, meLabel, t, locale, ...roleInks }),
    [roles, squadPlayers, meLabel, t, locale, roleInks],
  )
  if (obj == null || obj.matches_with_objectives <= 0 || roles.length === 0) return null

  return (
    <SectionCard
      title={t.blockObjectives}
      label={t.blockObjectives}
      titleAdornment={cardTitleAdornment(
        t.objectivesScopeFmt(obj.matches_with_objectives, usage.matches_total),
        t.cardHintObjectives,
      )}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        <section aria-label={t.viewRoles}>
          <UsageGaugeGrid rows={gaugeRows} t={t} />
        </section>
        {familyGrid && !compact && (
          <section aria-label={t.viewFamilies}>
            <ViewTitle>{t.viewFamilies}</ViewTitle>
            <ValueGrid model={familyGrid} />
          </section>
        )}
        {squadGrid && !compact && (
          <section aria-label={t.viewSquadRoles}>
            <ViewTitle>{t.viewSquadRoles}</ViewTitle>
            <ValueGrid model={squadGrid} />
          </section>
        )}
      </div>
    </SectionCard>
  )
}
