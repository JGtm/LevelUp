/**
 * ReplayMatchRecall — le rappel du match sur la page de rejeu : carte, mode, date, playlist.
 *
 * POURQUOI (retour utilisateur du 2026-08-27) : arrivé sur le rejeu, on ne sait plus QUEL
 * match on regarde — la page match, elle, le dit en tête. Le rejeu montrait un terrain sans
 * jamais le nommer.
 *
 * ÇA NE COÛTE RIEN : les quatre champs sont déjà EN MÉMOIRE. La page monte déjà la vue du
 * match (même clé de cache que la page match) pour son tableau de scores et son fil ; elle
 * n'en lisait simplement pas l'en-tête. Aucune requête de plus, aucune clé i18n de plus non
 * plus — ces libellés sont localisés par le serveur.
 *
 * COMPOSANT DE PRÉSENTATION PUR : ni lecture de store, ni requête, ni règle métier. La seule
 * composition de chaîne passe par le helper de la page match, pour que les deux pages disent
 * la même phrase avec les mêmes mots — une troisième copie de cette règle est interdite.
 *
 * RIEN TANT QU'ON NE SAIT RIEN : la vue du match peut être en vol quand le rejeu, lui, est
 * déjà là. Dans ce trou, on n'affiche RIEN — pas de squelette : une ligne grise promettrait
 * une information qu'on n'a pas encore. C'est la PAGE qui réserve la hauteur de la ligne
 * (replay.tsx, revue R1) : l'arrivée de la vraie ligne ne fait alors rien descendre — sans
 * cette réserve, « rien » aurait précisément produit le saut qu'on veut éviter.
 */
import { Badge } from '@/components/ui/badge'
import { buildMatchHeadingStr } from '@/features/match-view/format'
import type { Locale } from '@/lib/i18n/locale'

interface ReplayMatchRecallProps {
  mapUI: string | undefined
  modeUI: string | undefined
  startTimeLabel: string | undefined
  playlistLabel: string | undefined
  locale: Locale
}

export function ReplayMatchRecall({
  mapUI,
  modeUI,
  startTimeLabel,
  playlistLabel,
  locale,
}: ReplayMatchRecallProps) {
  const heading = buildMatchHeadingStr(mapUI, modeUI, locale)
  if (!heading && !startTimeLabel && !playlistLabel) return null

  return (
    <div className="flex items-center gap-2">
      {heading && <p className="text-sm text-muted-foreground">{heading}</p>}
      {/* Le point médian est de la PONCTUATION, pas un libellé : il ne se traduit pas, et il
          ne se lit pas — il n'apparaît que s'il a bien deux choses à séparer. */}
      {heading && startTimeLabel && (
        <span className="select-none text-muted-foreground/50" aria-hidden="true">
          ·
        </span>
      )}
      {startTimeLabel && <p className="text-sm text-muted-foreground">{startTimeLabel}</p>}
      {playlistLabel && (
        <Badge variant="outline" className="text-xs">
          {playlistLabel}
        </Badge>
      )}
    </div>
  )
}
