/**
 * TacticalMapTile — UNE carte de la grille : son fond, son nom, son nombre de matchs et sa
 * barre victoires / défaites.
 *
 * DEUX ÉTATS, ET UN SEUL EST CLIQUABLE :
 *   - carte OUVRABLE : bouton de sélection. Ce que le clic fait est dit par son nom
 *     accessible (« Sélectionner <carte> ») et son état (`aria-pressed`) ; la sélection
 *     est visible (encadré) et vit dans l'URL, donc partageable.
 *   - carte SOUS LE PLANCHER : bouton DÉSACTIVÉ (`disabled` + `aria-disabled`), désaturé,
 *     portant sa raison EN CLAIR (« N matchs sur 10 requis »). Elle reste affichée — le
 *     joueur doit voir qu'il y a joué — mais rien ne laisse croire qu'elle s'ouvrira.
 *
 * La désaturation passe par des utilitaires SANS couleur (`opacity`, `grayscale`) : aucun
 * token n'est détourné pour dire « indisponible ».
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundUrl } from './queries'
import { barreResultats, estOuvrable, nomCarte } from './tacticalLogic'

interface TacticalMapTileProps {
  carte: TacticalMapCard
  plancher: number
  playerSlug: string
  locale: Locale
  t: TacticalText
  selectionnee: boolean
  onSelect: (mapId: string) => void
}

export function TacticalMapTile({
  carte,
  plancher,
  playerSlug,
  locale,
  t,
  selectionnee,
  onSelect,
}: TacticalMapTileProps) {
  const ouvrable = estOuvrable(carte)
  const nom = nomCarte(carte, locale)
  const parts = barreResultats(carte)
  const fond = useTacticalMapBackgroundUrl(playerSlug, carte.map_id)

  const cadre = selectionnee ? 'border-2 border-primary' : 'border border-border'
  const attenuation = ouvrable ? '' : ' opacity-60 grayscale'

  return (
    <button
      type="button"
      disabled={!ouvrable}
      aria-disabled={!ouvrable}
      aria-pressed={ouvrable ? selectionnee : undefined}
      aria-label={ouvrable ? t.select(nom) : undefined}
      onClick={ouvrable ? () => onSelect(carte.map_id) : undefined}
      data-testid={`tactical-map-${carte.map_id}`}
      className={`flex flex-col overflow-hidden rounded-lg bg-card text-left ${cadre}${attenuation}${
        ouvrable ? ' hover:border-primary' : ' cursor-not-allowed'
      }`}
    >
      <span className="relative block aspect-video w-full overflow-hidden bg-muted">
        {fond && (
          <img
            src={fond}
            alt=""
            aria-hidden
            className="h-full w-full object-cover"
            data-testid={`tactical-map-fond-${carte.map_id}`}
          />
        )}
      </span>

      <span className="flex flex-col gap-1 p-3">
        <span className="text-sm font-medium text-foreground">{nom}</span>
        <span className="text-xs text-muted-foreground">{t.matches(carte.matchs)}</span>

        <span
          role="img"
          aria-label={t.recordLabel(carte.victoires, carte.defaites, carte.matchs)}
          className="mt-1 flex h-1.5 w-full overflow-hidden rounded-full bg-muted"
        >
          <span
            className="h-full"
            style={{
              width: `${parts.victoires * 100}%`,
              backgroundColor: tokenCssVar('outcome-win'),
            }}
          />
          <span
            className="h-full"
            style={{
              width: `${parts.defaites * 100}%`,
              backgroundColor: tokenCssVar('outcome-loss'),
            }}
          />
        </span>
        <span className="text-xs text-muted-foreground">
          {t.record(carte.victoires, carte.defaites)}
        </span>

        {!ouvrable && (
          <span
            className="text-xs text-muted-foreground"
            data-testid={`tactical-map-plancher-${carte.map_id}`}
          >
            {t.floorReason(carte.matchs, plancher)}
          </span>
        )}
        {ouvrable && selectionnee && <span className="sr-only">{t.selected}</span>}
      </span>
    </button>
  )
}
