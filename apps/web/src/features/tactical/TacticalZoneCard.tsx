/**
 * TacticalZoneCard — la colonne « Zone sélectionnée » du cockpit, en trois parties : l'en-tête compact (le
 * nom en jeu de la zone servi par `/tactical/{map}/cellule`, ses coordonnées, « valeur unité · matchs »
 * sur une ligne), la liste des contributions — une mini-tuile chacune (`TacticalRejeuTile`), du plus
 * récent au plus ancien (ordre du serveur) — qui prend toute la place restante et défile dans son cadre,
 * et EN BAS un bloc DISTINCT de hauteur fixe : le bilan de la zone sans tuile choisie, le détail de la
 * mort avec une tuile choisie (même hauteur, aucun saut ; un contenu long défile dans le bloc).
 *
 * Sans sélection (aucune cellule dans la lecture) : le titre « Zone sélectionnée » et une ligne.
 *
 * OWNERSHIP (ADR 0029) : les contributions sont déjà filtrées côté serveur ; les matchs du périmètre
 * qui ne sont pas au joueur ne sont jamais listés (le serveur les compte, la carte ne les dit pas).
 */
import { SectionCard } from '@/components/ui/section-card'
import type { CelluleTactique, TacticalBilanZone, TacticalCelluleReponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { TacticalMortDetail } from './TacticalMortDetail'
import { TacticalRejeuTile } from './TacticalRejeuTile'
import { TacticalZoneBilan } from './TacticalZoneBilan'
import { misEnAvant } from './tueurs.logic'
import { unitForQuestion, type TacticalQuestion } from './tacticalView.logic'
import type { VueDeMort } from './vueDeMort'
import { coordonneesDeZone, sousLigneDeZone, titreDeZone, valeurAffichee } from './zone.logic'

interface DetailDeZone {
  data?: TacticalCelluleReponse
  /** Le détail de la zone choisie est en cours de lecture. */
  isPending: boolean
  /** La lecture du détail a échoué : la liste dit l'échec, jamais « aucun match ouvrable ». */
  isError: boolean
}

interface TacticalZoneCardProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  /** La lecture À LAQUELLE la cellule répond (unité, sous-ligne). */
  question: TacticalQuestion
  /** La lecture est signée (« victoires − défaites », « solde ») : la valeur porte son signe. */
  signee: boolean
  /** Le pas de la grille servie, en mètres (coordonnées de la zone). */
  pasM: number
  /** La cellule choisie, `null` sans sélection. */
  cellule: CelluleTactique | null
  detail: DetailDeZone
  /** L'index de la contribution mise en avant (tuile ou point du plan survolé), `null` sinon. */
  survol?: number | null
  onSurvol?: (index: number | null) => void
  /** L'index de la contribution choisie (tuile bordée), `null` sinon. */
  choisie?: number | null
  onChoisir?: (index: number) => void
  /** Le bilan de la zone (équipe à la mort, rose, riposte), servi avec le détail ; absent sans mesure. */
  bilan?: TacticalBilanZone
  /** La vue de la mort choisie ; `null` = vue de la zone (bilan au-dessus de la liste). */
  vue?: VueDeMort | null
}

export function TacticalZoneCard({
  t,
  locale,
  playerSlug,
  question,
  signee,
  pasM,
  cellule,
  detail,
  survol = null,
  onSurvol,
  choisie = null,
  onChoisir,
  bilan,
  vue = null,
}: TacticalZoneCardProps) {
  const titre = titreDeZone(t, cellule !== null, detail.data)
  const morts = bilan?.morts && (question === 'morts' || question === 'isole') ? bilan.morts : undefined
  const aUnBloc = cellule !== null && (vue !== null || morts !== undefined)
  return (
    <div className="flex h-full min-h-0 flex-col gap-2.5" data-testid="tactical-zone-card">
      <SectionCard
        title={titre}
        label={t.zoneTitle}
        className="min-h-0 flex-1"
        titleAdornment={(libelle) => (
          <span className="flex items-baseline gap-2">
            <span className="min-w-0 truncate font-semibold" title={libelle} data-testid="tactical-zone-title">
              {libelle}
            </span>
            {cellule && (
              <span className="ml-auto flex-none whitespace-nowrap text-[11px] font-normal tabular-nums text-muted-foreground">
                {coordonneesDeZone(t, cellule.col, cellule.lig, pasM, locale)}
              </span>
            )}
          </span>
        )}
      >
        <div className="flex min-h-0 flex-1 flex-col gap-2 p-3">
          {cellule ? (
            <>
              <div className="flex flex-none items-baseline gap-1.5 border-b border-border pb-2">
                <b className="text-[22px] font-semibold leading-7 tabular-nums" data-testid="tactical-zone-value">
                  {valeurAffichee(cellule.valeur, signee, locale)}
                </b>
                <span className="text-xs text-muted-foreground">
                  {unitForQuestion(t, question)} · {sousLigneDeZone(t, question, cellule)}
                </span>
              </div>
              <h4 className="m-0 flex-none text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                {t.zoneReplayHeading}
              </h4>
              <ListeDuRejeu t={t} locale={locale} playerSlug={playerSlug} detail={detail} survol={survol}
                onSurvol={onSurvol}
                choisie={choisie}
                onChoisir={onChoisir}
              />
            </>
          ) : (
            <p className="m-0 text-[13px] text-muted-foreground">{t.zoneNone}</p>
          )}
        </div>
      </SectionCard>
      {aUnBloc && (
        // LE BLOC DU BAS : une carte séparée, de HAUTEUR FIXE, réservée à deux usages exclusifs — le bilan
        // de la zone (aucune tuile choisie) ou le détail de la mort (tuile choisie). Même hauteur dans les
        // deux états : rien ne saute. Un contenu plus long défile dans le bloc.
        <div
          className="h-[196px] flex-none overflow-y-auto rounded-lg border border-border bg-card p-3"
          data-testid="tactical-zone-bloc"
        >
          {vue ? (
            <TacticalMortDetail t={t} locale={locale} playerSlug={playerSlug} vue={vue} />
          ) : (
            morts && <TacticalZoneBilan t={t} locale={locale} morts={morts} />
          )}
        </div>
      )}
    </div>
  )
}

/** ListeDuRejeu — les mini-tuiles des contributions, à défilement interne. */
function ListeDuRejeu({
  t,
  locale,
  playerSlug,
  detail,
  survol,
  onSurvol,
  choisie,
  onChoisir,
}: {
  t: TacticalText
  locale: Locale
  playerSlug: string
  detail: DetailDeZone
  survol: number | null
  onSurvol?: (index: number | null) => void
  choisie: number | null
  onChoisir?: (index: number) => void
}) {
  if (detail.isError) {
    return (
      <p className="text-xs text-muted-foreground" data-testid="tactical-zone-erreur">
        {t.zoneError}
      </p>
    )
  }
  if (detail.isPending) return <p className="text-xs text-muted-foreground">{t.zoneContributionsLoading}</p>
  const contributions = detail.data?.contributions ?? []
  if (contributions.length === 0) return <p className="text-xs text-muted-foreground">{t.zoneContributionsEmpty}</p>
  return (
    <ul
      role="listbox"
      aria-label={t.zoneEventsLabel}
      className="-mx-3 flex min-h-28 flex-1 flex-col gap-1.5 overflow-y-auto px-3 py-0.5"
      data-testid="tactical-rejeu-list"
    >
      {contributions.map((c, i) => (
        <TacticalRejeuTile
          key={`${c.match_id}-${c.instant_ms}-${i}`}
          t={t}
          locale={locale}
          playerSlug={playerSlug}
          contribution={c}
          actif={misEnAvant(survol, choisie) === i}
          choisi={choisie === i}
          onChoisir={() => onChoisir?.(i)}
          onSurvol={(survole) => onSurvol?.(survole ? i : null)}
        />
      ))}
    </ul>
  )
}
