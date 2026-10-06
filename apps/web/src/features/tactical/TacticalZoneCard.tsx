/**
 * TacticalZoneCard — la colonne « Zone sélectionnée » du cockpit : le nom en jeu de la zone (servi
 * par `/tactical/{map}/cellule`) et ses coordonnées, sa valeur et son unité, sa sous-ligne par
 * lecture, puis l'intertitre « Rejeu » et la liste des contributions — une mini-tuile chacune
 * (`TacticalRejeuTile`), du plus récent au plus ancien (ordre du serveur), dans une liste à
 * défilement interne : la carte prend la hauteur de la carte du plan.
 *
 * Sans sélection (aucune cellule dans la lecture) : le titre « Zone sélectionnée » et une ligne.
 *
 * OWNERSHIP (ADR 0029) : les contributions sont déjà filtrées côté serveur ; les matchs du périmètre
 * qui ne sont pas au joueur sont comptés en pied de liste, jamais listés.
 */
import { SectionCard } from '@/components/ui/section-card'
import type { CelluleTactique, TacticalCelluleReponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { TacticalRejeuTile } from './TacticalRejeuTile'
import { unitForQuestion, type TacticalQuestion } from './tacticalView.logic'
import { coordonneesDeZone, sousLigneDeZone, titreDeZone, valeurAffichee } from './zone.logic'

interface DetailDeZone {
  data?: TacticalCelluleReponse
  /** Le détail de la zone choisie est en cours de lecture. */
  isPending: boolean
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
}

export function TacticalZoneCard({ t, locale, playerSlug, question, signee, pasM, cellule, detail }: TacticalZoneCardProps) {
  const titre = titreDeZone(t, locale, cellule !== null, detail.data)
  return (
    <div className="h-full min-h-0" data-testid="tactical-zone-card">
      <SectionCard
        title={titre}
        label={t.zoneTitle}
        className="h-full"
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
        <div className="flex min-h-0 flex-1 flex-col gap-1 p-3">
          {cellule ? (
            <>
              <div className="flex flex-wrap items-baseline gap-2">
                <b className="text-[28px] font-semibold leading-[34px] tabular-nums" data-testid="tactical-zone-value">
                  {valeurAffichee(cellule.valeur, signee, locale)}
                </b>
                <span className="text-[13px] text-muted-foreground">{unitForQuestion(t, question)}</span>
              </div>
              <div className="text-xs text-muted-foreground">{sousLigneDeZone(t, question, cellule)}</div>
              <h4 className="mt-2 flex-none border-t border-border pt-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t.zoneReplayHeading}
              </h4>
              <ListeDuRejeu t={t} locale={locale} playerSlug={playerSlug} detail={detail} />
            </>
          ) : (
            <p className="m-0 text-[13px] text-muted-foreground">{t.zoneNone}</p>
          )}
        </div>
      </SectionCard>
    </div>
  )
}

/** ListeDuRejeu — les mini-tuiles des contributions, à défilement interne, et le pied d'ownership. */
function ListeDuRejeu({
  t,
  locale,
  playerSlug,
  detail,
}: {
  t: TacticalText
  locale: Locale
  playerSlug: string
  detail: DetailDeZone
}) {
  if (detail.isPending) return <p className="text-xs text-muted-foreground">{t.zoneContributionsLoading}</p>
  const contributions = detail.data?.contributions ?? []
  const nonOuvrables = detail.data?.matchs_non_ouvrables ?? 0
  return (
    <>
      {contributions.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t.zoneContributionsEmpty}</p>
      ) : (
        <ul className="-mx-3 min-h-0 flex-1 overflow-y-auto" data-testid="tactical-rejeu-list">
          {contributions.map((c, i) => (
            <TacticalRejeuTile
              key={`${c.match_id}-${c.instant_ms}-${i}`}
              t={t}
              locale={locale}
              playerSlug={playerSlug}
              contribution={c}
            />
          ))}
        </ul>
      )}
      {nonOuvrables > 0 && (
        <p className="flex-none text-[11px] text-muted-foreground" data-testid="tactical-zone-not-openable">
          {t.zoneNotOpenable(nonOuvrables)}
        </p>
      )}
    </>
  )
}
