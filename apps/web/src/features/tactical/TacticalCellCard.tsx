/**
 * TacticalCellCard — la carte « Cellule sélectionnée » de la vue d'analyse (item 5.6,
 * posée par le lot M1 — Tactique S.1, « voir dans le rejeu » ; horloge EXACTE, lot M1b du
 * 2026-09-08, décision utilisateur ferme « corriger le décalage »).
 *
 * DEUX BLOCS : la valeur agrégée du raster (toujours servie avec la cellule), PUIS les
 * CONTRIBUTIONS individuelles (`useTacticalCellule`, lancée uniquement quand une cellule
 * est sélectionnée) — chacune un lien vers le rejeu du match, à l'instant concerné.
 *
 * OWNERSHIP (ADR 0029) : les contributions reçues sont déjà filtrées côté serveur — cette
 * carte ne fait qu'afficher `matchs_non_ouvrables` en pied de liste (0 → rien, jamais un
 * zéro qui inviterait à le lire comme une absence de restriction).
 *
 * LE LIEN EST UN `<Link>` DU ROUTEUR, JAMAIS UN `<a href>` : un `<a href>` natif provoque
 * une navigation DOCUMENT — l'application entière se recharge, et le retour arrière la
 * recharge une deuxième fois. C'est le « ça recharge la page » constaté par l'utilisateur
 * (reproduit le 2026-09-13 : 1 évènement `load` et 2 `framenavigated` sur un clic de lien).
 *
 * LE LIEN PORTE `?t=<instant_ms>&clock=<c.clock>`, JAMAIS UNE FRAME PRÉ-CALCULÉE ICI. Cette
 * carte n'a pas l'artefact de rejeu sous la main (il n'est pas toujours cuit) et ne peut donc
 * PAS savoir si l'instant est déjà exact (`clock: "film"`, questions `temps`/`routes`) ou
 * doit être recalé par un décalage publié seulement quand le document est chargé (`clock:
 * "match"`, questions `morts`/`kills`/`gagne`/`isole`) : c'est la ROUTE du rejeu qui
 * convertit, une fois le document ouvert (`lib/replay/replayLogic.resolveTacticalReplayInstant`).
 * `instantToFrame` (tacticalView.logic.ts) est mort depuis ce lot — cette conversion a
 * remplacé sa seule utilisation, et il a été retiré avec ses tests.
 */
import { Link } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { CelluleTactique, TacticalContribution } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import { useOutcomeMapping } from '@/lib/i18n/fieldMappings'
import type { Locale } from '@/lib/i18n/locale'
import { outcomeTokenFromCanonical } from '@/lib/outcome-color'
import { formatClock } from '@/lib/replay/replayLogic'
import { useTitleSlug } from '@/lib/title-routing'

import type { TacticalText } from './i18n'
import {
  questionSansCellule,
  unitForQuestion,
  type TacticalQuestion,
} from './tacticalView.logic'

export interface TacticalCellCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  question: TacticalQuestion
  cellule: CelluleTactique | null
  /** Contributions de la cellule sélectionnée. `null` = aucune cellule choisie (la requête
   *  n'est alors pas lancée, cf. `useTacticalCellule`) OU chargement en cours. */
  contributions: readonly TacticalContribution[] | null
  contributionsLoading: boolean
  matchsNonOuvrables: number
}

export function TacticalCellCard({
  t,
  locale,
  playerSlug,
  question,
  cellule,
  contributions,
  contributionsLoading,
  matchsNonOuvrables,
}: TacticalCellCardProps) {
  const unite = unitForQuestion(t, question)
  const numFmt = new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 2 })

  return (
    <SectionCard title={t.cellTitle} label={t.cellTitle}>
      <div className="flex flex-col gap-3 p-3" data-testid="tactical-cell-card">
        {questionSansCellule(question) ? (
          // « Mes routes de spawn » empile des trajets : il n'y a pas de grandeur par
          // cellule à détailler. Le dire vaut mieux qu'inviter à un clic sans réponse.
          <EmptyStateNotice
            title={t.cellPlaceholderRoutes}
            description={t.cellPlaceholderRoutesDescription}
          />
        ) : !cellule ? (
          <EmptyStateNotice title={t.cellPlaceholder} description={t.cellPlaceholderDescription} />
        ) : (
          <>
            <div className="flex flex-col gap-1">
              <p className="flex items-baseline gap-2" data-testid="tactical-cell-value">
                <span className="text-2xl font-semibold text-foreground">
                  {numFmt.format(cellule.valeur)}
                </span>
                <span className="text-sm text-muted-foreground">{unite}</span>
              </p>
              {cellule.matchs > 0 && (
                <p className="text-xs text-muted-foreground">{t.cellMatches(cellule.matchs)}</p>
              )}
            </div>
            <TacticalCellContributions
              t={t}
              locale={locale}
              playerSlug={playerSlug}
              contributions={contributions}
              loading={contributionsLoading}
              matchsNonOuvrables={matchsNonOuvrables}
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}

interface TacticalCellContributionsProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  contributions: readonly TacticalContribution[] | null
  loading: boolean
  matchsNonOuvrables: number
}

function TacticalCellContributions({
  t,
  locale,
  playerSlug,
  contributions,
  loading,
  matchsNonOuvrables,
}: TacticalCellContributionsProps) {
  const titleSlug = useTitleSlug()
  const dateFmt = new Intl.DateTimeFormat(intlLocale(locale), { dateStyle: 'medium' })

  return (
    <div className="flex flex-col gap-2 border-t border-border pt-2" data-testid="tactical-cell-contributions">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {t.cellContributionsTitle}
      </h3>
      {loading && <p className="text-xs text-muted-foreground">{t.cellContributionsLoading}</p>}
      {!loading && (contributions === null || contributions.length === 0) && (
        <p className="text-xs text-muted-foreground">{t.cellContributionsEmpty}</p>
      )}
      {!loading && contributions !== null && contributions.length > 0 && (
        <ul className="flex flex-col gap-1">
          {contributions.map((c, i) => (
            <TacticalCellContributionRow
              key={`${c.match_id}-${c.instant_ms}-${i}`}
              t={t}
              contribution={c}
              date={dateFmt.format(new Date(c.match_started_at))}
              titleSlug={titleSlug}
              playerSlug={playerSlug}
            />
          ))}
        </ul>
      )}
      {matchsNonOuvrables > 0 && (
        <p className="text-2xs text-muted-foreground" data-testid="tactical-cell-not-openable">
          {t.cellFooterNotOpenable(matchsNonOuvrables)}
        </p>
      )}
    </div>
  )
}

interface TacticalCellContributionRowProps {
  t: TacticalText
  contribution: TacticalContribution
  date: string
  titleSlug: string
  playerSlug: string
}

/**
 * UNE ligne de la liste : la date du match, SON ISSUE (mot et couleur du titre), et le lien
 * « ouvrir à mm:ss » — la forme de la maquette 034b1915.
 *
 * L'ISSUE VIENT DU SERVEUR SOUS SA FORME CANONIQUE (`resultat` : win/loss/tie/dnf) : le mot
 * est celui d'`outcomes.toml` (`useOutcomeMapping`), la couleur un jeton sémantique. Une
 * issue inconnue n'affiche RIEN plutôt qu'un mot par défaut, qui se lirait comme une défaite.
 */
function TacticalCellContributionRow({
  t,
  contribution: c,
  date,
  titleSlug,
  playerSlug,
}: TacticalCellContributionRowProps) {
  const instant = formatClock(c.instant_ms)
  const issue = useOutcomeMapping(c.resultat ?? '')
  const jeton = outcomeTokenFromCanonical(c.resultat ?? '')

  return (
    <li className="flex items-baseline gap-2 border-b border-border py-1 last:border-b-0">
      <span className="text-xs text-muted-foreground">{date}</span>
      {issue && jeton && (
        <span className="text-xs font-medium" style={{ color: tokenCssVar(jeton) }}>
          {issue.label}
        </span>
      )}
      <Link
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay"
        params={{ titleSlug, playerSlug, matchId: c.match_id }}
        // `t` est une CHAÎNE dans le schéma de la route (pas un nombre) : `FullSearchSchema`
        // fusionne tous les schémas de recherche du dépôt, et un champ numérique y casse
        // des lecteurs sans rapport qui supposent chaque valeur déjà une chaîne
        // (`HelpPage.tsx`/`SettingsPage.tsx`, `new URLSearchParams(location.search)`).
        // `c.clock` est un `string` côté contrat généré (Go publie `"match"`/`"film"`
        // par construction, domain.TacticalClockMatch/Film) : la route revalide au
        // moment de le lire (`z.enum`), ce cast n'écarte donc aucune garde réelle.
        search={{ t: String(c.instant_ms), clock: c.clock as 'match' | 'film' }}
        aria-label={t.cellContributionLabel(date, instant)}
        className="ml-auto whitespace-nowrap text-xs text-primary hover:underline"
        data-testid="tactical-cell-contribution-link"
      >
        {t.cellContributionOpen(instant)}
      </Link>
    </li>
  )
}
