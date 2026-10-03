/**
 * MatchRiposteSection — LA RIPOSTE DU MATCH, DEUX GRAPHES ET AUCUNE PHRASE.
 *
 * « La mort de X a été vengée dans les 5 s, par Y » n'était rendue nulle part sur la page :
 * Nemesis / Souffre-douleur donne un SOLDE de duel (ni fenêtre, ni ordre des événements),
 * et le graphe des assistances un autre sujet.
 *
 * DEUX GRAPHES, UN PAR CAMP, CÔTE À CÔTE (décision D22-2, 2026-09-21 — et PAS de tuiles de
 * face-à-face par camp : sur un match, deux colonnes de chiffres ne se comparaient qu'une
 * ligne à la fois). Chaque graphe : une ligne par joueur, barres DIVERGENTES autour d'un
 * axe central — à gauche ses morts vengées par son camp (événement SUBI, teinte pâle du
 * camp), à droite les ripostes qu'il a portées (événement PORTÉ, teinte pleine).
 *
 * UNE SEULE ÉCHELLE pour les deux graphes ET les deux côtés (`scaleMax`, cf. `_riposte.ts`).
 * Deux bornes côte à côte, et un joueur qui riposte deux fois moins aurait la même barre.
 *
 * AUCUNE PHRASE DE LECTEUR SUR LA CARTE (D22-verbosité, LOI) : ce que la riposte est se lit
 * dans l'infobulle ⓘ du titre, en trois phrases. Le reste, ce sont les comptes écrits au
 * bout des barres et les couples nommés en infobulle.
 *
 * DOM/CSS ET NON ECHARTS, et c'est le mécanisme déjà en service sur le face-à-face des
 * objectifs (`MatchObjectivesSection`) : `BarStackedChart` ne porte pas le divergent (pas
 * de côté négatif), et son rendu canvas n'exposerait ni les comptes ni l'ordre aux tests.
 * Pas un troisième mécanisme.
 *
 * ÉTAT VIDE NOMMÉ (politique D8), jamais une section qui disparaît sans rien dire — sauf
 * le bloc ABSENT, qui signifie « aucune ligne de journal pour ce match » et ne rend rien,
 * exactement comme la porte 1 de `MatchAssistChart`.
 */
import { ChartLegend } from '@/components/charts/ChartLegend'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { Tooltip } from '@/components/ui/tooltip'
import type { MatchRiposteBlock, MatchScoreboardRow } from '@/lib/api/types'
import { formatNumber } from '@/lib/formatters/number'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import type { Locale } from '@/lib/i18n/locale'
import { displayPlayerName } from '@/lib/players/displayName'

import {
  buildRiposteModel,
  riposteFraction,
  splitCouples,
  type RiposteCamp,
  type RiposteCouple,
  type RiposteRow,
} from './_riposte'
import { teamTokenCssVar } from './teamSeriesColor'
import type { MatchViewText } from './i18n'

interface Props {
  /** Bloc du contrat — absent quand le match n'a aucune ligne de journal. */
  block: MatchRiposteBlock | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  locale: Locale
  t: MatchViewText
}

export function MatchRiposteSection({ block, scoreboard, meXUID, locale, t }: Props) {
  // Porte 1 — bloc absent : pas de journal des morts, donc rien à dire et rien de rendu.
  if (!block) return null

  const model = buildRiposteModel(block, meXUID)
  const myTeamSide = scoreboard.find((r) => r.is_me)?.team_side ?? null
  // Porte 2 — le film est là mais aucune mort n'y est lisible, OU il l'est et personne n'a
  // riposté. Deux faits différents, deux libellés : écrire « aucune riposte » sur le premier
  // fabriquerait une observation jamais faite.
  const empty =
    model.measuredDeaths === 0
      ? t.riposteNotUsable
      : model.avengedTotal === 0
        ? t.riposteNoData
        : null

  return (
    <SectionCard
      title={t.riposteTitle}
      label={t.riposteTitle}
      titleAdornment={titleWithInfo(<p>{t.riposteInfo}</p>)}
      footer={
        empty ? undefined : (
          <p className="px-3 pb-3 text-2xs text-muted-foreground">
            {t.riposteFooterFmt(model.avengedTotal, model.measuredDeaths)}
          </p>
        )
      }
    >
      {empty ? (
        <p className="px-3 pb-3 pt-2 text-xs text-muted-foreground">{empty}</p>
      ) : (
        <div className="grid grid-cols-1 gap-5 px-3 pb-2 pt-3 lg:grid-cols-2">
          {model.camps.map((camp) => (
            <CampChart
              key={camp.key}
              camp={camp}
              ally={campAlly(camp, myTeamSide)}
              scaleMax={model.scaleMax}
              scoreboard={scoreboard}
              locale={locale}
              t={t}
            />
          ))}
        </div>
      )}
    </SectionCard>
  )
}

/**
 * « Allié » = le camp du joueur de la page. Camp inconnu, ou page sans `is_me` au tableau
 * des scores : `null` — encre neutre, jamais l'une des deux couleurs d'équipe
 * (même règle que `teamTokenCssVar`).
 */
function campAlly(camp: RiposteCamp, myTeamSide: string | null): boolean | null {
  if (camp.teamSide == null || myTeamSide == null) return null
  return camp.teamSide === myTeamSide
}

/**
 * LA TEINTE PÂLE DU CAMP — l'événement SUBI. `color-mix` sur l'encre du camp, jamais une
 * seconde couleur : c'est le MÊME jeton, à 40 % ; une teinte choisie à part dériverait du
 * camp dès que l'utilisateur change sa palette d'accessibilité. Même mécanisme que
 * `ReviewBadge` (le seul autre endroit où une encre de jeton est éclaircie).
 */
function paleColor(color: string): string {
  return `color-mix(in oklab, ${color} 40%, transparent)`
}

/** Un camp : son nom, ses lignes, et la légende de ses deux teintes. */
function CampChart({
  camp,
  ally,
  scaleMax,
  scoreboard,
  locale,
  t,
}: {
  camp: RiposteCamp
  ally: boolean | null
  scaleMax: number
  scoreboard: MatchScoreboardRow[]
  locale: Locale
  t: MatchViewText
}) {
  const color = teamTokenCssVar(ally)
  const label = resolveTeamLabel(
    scoreboard.filter((r) => (r.team_side ?? '') === (camp.teamSide ?? '')),
    camp.teamSide,
    t,
  )
  return (
    <section aria-label={label} data-testid={`riposte-camp-${camp.key}`}>
      <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider" style={{ color }}>
        {label}
      </h4>
      <div className="mb-1 flex justify-between text-3xs text-muted-foreground">
        <span>{t.riposteSideAvenged}</span>
        <span>{t.riposteSideDid}</span>
      </div>
      <div className="space-y-1.5">
        {camp.rows.map((row) => (
          <PlayerRow
            key={row.xuid}
            row={row}
            color={color}
            scaleMax={scaleMax}
            locale={locale}
            t={t}
          />
        ))}
      </div>
      <ChartLegend
        className="pt-2"
        ariaLabel={label}
        items={[
          { key: 'avenged', label: t.riposteLegendAvenged, color: paleColor(color) },
          { key: 'did', label: t.riposteLegendDid, color },
        ]}
      />
    </section>
  )
}

/** Une ligne : compte + barre à gauche, le nom au centre, barre + compte à droite. */
function PlayerRow({
  row,
  color,
  scaleMax,
  locale,
  t,
}: {
  row: RiposteRow
  color: string
  scaleMax: number
  locale: Locale
  t: MatchViewText
}) {
  const name = displayPlayerName(row.gamertag, row.xuid)
  return (
    <div className="grid grid-cols-[1fr_minmax(0,96px)_1fr] items-center gap-2">
      <Side
        align="end"
        count={row.deathsAvenged}
        fraction={riposteFraction(row.deathsAvenged, scaleMax)}
        color={paleColor(color)}
        testId={`riposte-bar-avenged-${row.xuid}`}
        ariaLabel={`${name} — ${t.riposteSideAvenged} : ${row.deathsAvenged}`}
        couples={row.avengedBy}
        coupleFmt={t.riposteAvengedByFmt}
        locale={locale}
        t={t}
      />
      <span className="truncate text-center text-2xs" title={name}>
        {name}
      </span>
      <Side
        align="start"
        count={row.ripostes}
        fraction={riposteFraction(row.ripostes, scaleMax)}
        color={color}
        testId={`riposte-bar-did-${row.xuid}`}
        ariaLabel={`${name} — ${t.riposteSideDid} : ${row.ripostes}`}
        couples={row.avengedFor}
        coupleFmt={t.riposteAvengedForFmt}
        locale={locale}
        t={t}
      />
    </div>
  )
}

/**
 * Un côté de l'axe : la barre part du CENTRE (le côté gauche colle la sienne à droite de sa
 * cellule) et le compte s'écrit à son bout.
 */
function Side({
  align,
  count,
  fraction,
  color,
  testId,
  ariaLabel,
  couples,
  coupleFmt,
  locale,
  t,
}: {
  align: 'start' | 'end'
  count: number
  fraction: number
  color: string
  testId: string
  ariaLabel: string
  couples: RiposteCouple[]
  coupleFmt: (name: string, s: string) => string
  locale: Locale
  t: MatchViewText
}) {
  const value = <span className="text-2xs tabular-nums text-muted-foreground">{count}</span>
  const bar = (
    <Tooltip
      content={<CoupleList couples={couples} fmt={coupleFmt} locale={locale} t={t} />}
      className={`min-w-0 flex-1${align === 'end' ? ' justify-end' : ''}`}
    >
      <div
        className="h-[14px] rounded-sm"
        tabIndex={0}
        role="img"
        aria-label={ariaLabel}
        data-testid={testId}
        style={{ backgroundColor: color, width: `${fraction * 100}%` }}
      />
    </Tooltip>
  )
  return (
    <div className={`flex items-center gap-1.5${align === 'end' ? ' justify-end' : ''}`}>
      {align === 'end' ? value : bar}
      {align === 'end' ? bar : value}
    </div>
  )
}

/** Les couples nommés d'une barre, cinq lignes puis le reste en nombre. */
function CoupleList({
  couples,
  fmt,
  locale,
  t,
}: {
  couples: RiposteCouple[]
  fmt: (name: string, s: string) => string
  locale: Locale
  t: MatchViewText
}) {
  const { shown, rest } = splitCouples(couples)
  if (shown.length === 0) return null
  return (
    <ul className="space-y-0.5">
      {shown.map((c, i) => (
        <li key={`${c.name}-${c.delaiMs}-${i}`}>
          {fmt(displayPlayerName(c.name, null), formatNumber(c.delaiMs / 1000, locale, 1))}
        </li>
      ))}
      {rest > 0 ? <li>{t.riposteMoreFmt(rest)}</li> : null}
    </ul>
  )
}
