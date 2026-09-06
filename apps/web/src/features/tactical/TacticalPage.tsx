/**
 * TacticalPage — L'ÉCRAN D'ENTRÉE de l'onglet Tactique : la grille des cartes JOUÉES.
 *
 * Ce que la page décide : RIEN. Le tri, le verdict de plancher, la barre et la couverture
 * viennent de `tacticalLogic` (pur, testé seul) ; les libellés du manifeste `tactical.toml`.
 *
 * LE CLIC SÉLECTIONNE, IL N'OUVRE PAS ENCORE (décision de la phase 4, consignée au plan) :
 * la vue d'analyse d'une carte est la phase 5, GELÉE jusqu'au lot D de l'audit du rejeu, et
 * sa route n'existe pas. Un `Link` vers une route inexistante ne compilerait pas ; un
 * `navigate` vers une chaîne construite mènerait à un 404 — un lien MORT. La carte choisie
 * est donc écrite dans l'URL (`?carte=<map_id>`), ce qui est à la fois honnête (le bouton
 * SÉLECTIONNE, son nom accessible le dit, et la sélection se voit), durable (elle survit au
 * retour navigateur et se partage) et exactement l'état que la phase 5 consommera.
 *
 * L'onglet n'a AUCUN filtre propre : les filtres globaux de l'omnibar s'appliquent comme
 * partout ailleurs (cf. `queries.useTacticalMaps`).
 */
import { useMemo } from 'react'
import { useNavigate, useParams, useSearch } from '@tanstack/react-router'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionCard } from '@/components/ui/section-card'
import { useAppShellStore } from '@/stores/appShellStore'

import { getTacticalText } from './i18n'
import { TacticalMapTile } from './TacticalMapTile'
import { useTacticalMaps } from './queries'
import { couvertureGrille, trierCartes } from './tacticalLogic'

export function TacticalPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug?: string }
  const search = useSearch({ strict: false }) as { carte?: string }
  const navigate = useNavigate()
  const locale = useAppShellStore((s) => s.locale)
  const t = getTacticalText(locale)

  const slug = playerSlug ?? ''
  const { data, isLoading, isError } = useTacticalMaps(slug)

  const cartes = useMemo(() => trierCartes(data?.cartes ?? []), [data])
  const couverture = useMemo(() => couvertureGrille(cartes), [cartes])
  const plancher = data?.plancher_matchs ?? 0

  const selectionner = (mapId: string) => {
    void navigate({
      to: '.',
      search: (prev: Record<string, unknown>) => ({ ...prev, carte: mapId }),
      replace: true,
    })
  }

  return (
    <SectionCard
      title={t.mapsTitle}
      label={t.mapsLabel}
      footer={
        cartes.length > 0 ? (
          <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
            <p data-testid="tactical-couverture">
              {t.coverage(couverture.cartes, couverture.matchs)}
            </p>
            {plancher > 0 && <p className="mt-1">{t.floorNote(plancher)}</p>}
          </div>
        ) : undefined
      }
    >
      <div className="p-3">
        {isLoading && <p className="text-sm text-muted-foreground">{t.loading}</p>}
        {!isLoading && isError && (
          <p className="text-sm text-muted-foreground" data-testid="tactical-erreur">
            {t.error}
          </p>
        )}
        {!isLoading && !isError && cartes.length === 0 && (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        )}
        {!isLoading && !isError && cartes.length > 0 && (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {cartes.map((carte) => (
              <TacticalMapTile
                key={carte.map_id}
                carte={carte}
                plancher={plancher}
                playerSlug={slug}
                locale={locale}
                t={t}
                selectionnee={search.carte === carte.map_id}
                onSelect={selectionner}
              />
            ))}
          </div>
        )}
      </div>
    </SectionCard>
  )
}
