/**
 * TacticalMapsColumn — la colonne « Cartes jouées » de l'écran unique de l'onglet Tactique.
 *
 * Un champ de recherche en tête (au fil de la frappe, sans casse ni accents, sur le nom affiché
 * et le nom canonique), la liste des cartes OUVRABLES de la plus jouée à la moins jouée — rangée
 * défilante au-dessus du plan sous 1 400 px de large ; au-delà, colonne À LA HAUTEUR DE LA CARTE DU
 * PLAN, comme « Zone sélectionnée » (étirée sur la rangée, `contain: size` pour ne pas la peser),
 * à défilement interne — puis le repli des cartes sous le plancher. La logique vit dans
 * `cockpit.logic` (`colonneDesCartes`) ; ce composant rend ce qu'elle décide.
 *
 * `etat` remplace le corps de la colonne quand la page n'a pas de cartes à montrer (composition
 * impossible, échec, attente, aucune carte) : la colonne garde sa place et dit pourquoi.
 */
import { useMemo, useState, type ReactNode } from 'react'

import { Input } from '@/components/ui/input'
import { SectionCard } from '@/components/ui/section-card'
import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { colonneDesCartes, type ColonneDesCartes } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { TacticalMapTile } from './TacticalMapTile'
import { classeRelecture } from './tacticalLecture.logic'
import { nomCarte } from './tacticalLogic'

interface TacticalMapsColumnProps {
  cartes: readonly TacticalMapCard[]
  plancher: number
  /** La carte affichée par le plan (sélectionnée ou choisie d'office) ; '' = aucune. */
  carteActive: string
  playerSlug: string
  locale: Locale
  t: TacticalText
  onSelect: (mapId: string) => void
  /** Les cartes affichées répondent à l'ANCIEN filtre (relecture en cours). */
  enRelecture: boolean
  /** Corps de remplacement quand la page n'a pas de cartes à montrer. */
  etat?: ReactNode
}

export function TacticalMapsColumn({
  cartes,
  plancher,
  carteActive,
  playerSlug,
  locale,
  t,
  onSelect,
  enRelecture,
  etat,
}: TacticalMapsColumnProps) {
  const [recherche, setRecherche] = useState('')
  const colonne = useMemo(() => colonneDesCartes(cartes, recherche, locale), [cartes, recherche, locale])

  return (
    <div className="min-w-0 min-[1400px]:self-stretch min-[1400px]:[contain:size]" data-testid="tactical-maps-column">
      <SectionCard title={t.mapsTitle} label={t.mapsTitle} className="h-full">
        {etat ?? (
          <>
            <div className="flex-none px-2 pt-2 pb-0.5">
              <Input
                type="search"
                value={recherche}
                onChange={(e) => setRecherche(e.target.value)}
                placeholder={t.mapsSearchPlaceholder}
                aria-label={t.mapsSearchLabel}
                autoComplete="off"
                className="h-7 px-2 text-xs"
              />
            </div>
            {enRelecture && (
              <p role="status" className="px-2.5 pt-1.5 text-xs text-muted-foreground" data-testid="tactical-grille-updating">
                {t.analysisUpdating}
              </p>
            )}
            <div
              className={`flex min-h-0 flex-row gap-1.5 overflow-x-auto overflow-y-hidden p-1.5 min-[1400px]:flex-1 min-[1400px]:flex-col min-[1400px]:overflow-x-hidden min-[1400px]:overflow-y-auto ${classeRelecture(enRelecture)}`}
              aria-busy={enRelecture}
              data-testid="tactical-grille"
            >
              {colonne.ouvrables.map((c) => (
                <TacticalMapTile
                  key={c.map_id}
                  carte={c}
                  playerSlug={playerSlug}
                  locale={locale}
                  t={t}
                  selectionnee={c.map_id === carteActive}
                  onSelect={onSelect}
                />
              ))}
              {colonne.listeVide && (
                <p className="self-center px-1.5 py-1 text-[11px] leading-[14px] text-muted-foreground">
                  {colonne.listeVide === 'aucune_correspondance' ? t.mapsNoMatch : t.mapsNoneOpenable}
                </p>
              )}
            </div>
            {colonne.totalSousPlancher > 0 && (
              <RepliSousPlancher colonne={colonne} plancher={plancher} locale={locale} t={t} />
            )}
          </>
        )}
      </SectionCard>
    </div>
  )
}

/**
 * RepliSousPlancher — les cartes sous le plancher, une ligne « nom · n sur plancher » chacune,
 * résumé « N cartes sous le plancher » (« x sur N » pendant une recherche).
 *
 * L'ouverture d'office suit la recherche (`repliOuvert`) ; entre deux changements, l'utilisateur
 * ouvre et ferme le repli librement : la clé remonte l'élément quand l'ouverture d'office bascule.
 */
function RepliSousPlancher({
  colonne,
  plancher,
  locale,
  t,
}: {
  colonne: ColonneDesCartes
  plancher: number
  locale: Locale
  t: TacticalText
}) {
  return (
    <details
      key={String(colonne.repliOuvert)}
      open={colonne.repliOuvert}
      className="mt-1.5 flex-none border-t border-border px-2.5 pt-1.5 pb-2 text-xs"
      data-testid="tactical-maps-floor"
    >
      <summary className="cursor-pointer text-muted-foreground">
        {colonne.rechercheActive
          ? t.floorFoldFiltered(colonne.sousPlancher.length, colonne.totalSousPlancher)
          : t.floorFold(colonne.totalSousPlancher)}
      </summary>
      <ul className="mt-1.5 max-h-[180px] overflow-y-auto pr-0.5">
        {colonne.sousPlancher.map((c) => (
          <li
            key={c.map_id}
            className="flex justify-between gap-2 border-b border-border text-[11.5px] leading-[19px] last:border-b-0"
            data-testid={`tactical-map-plancher-${c.map_id}`}
          >
            <span className="min-w-0 truncate">{nomCarte(c, locale)}</span>
            <span className="flex-none tabular-nums text-muted-foreground">{t.floorCount(c.matchs, plancher)}</span>
          </li>
        ))}
      </ul>
    </details>
  )
}
