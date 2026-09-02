/**
 * ReplayMatchRecall — ce que le fil d'Ariane ne dit pas du match : la date, la playlist.
 *
 * POURQUOI (retour utilisateur du 2026-08-27) : arrivé sur le rejeu, on ne sait plus QUEL
 * match on regarde — la page match, elle, le dit en tête. Le rejeu montrait un terrain sans
 * jamais le nommer.
 *
 * CE QU'IL A CESSÉ DE DIRE (2026-09-02). Il rappelait AUSSI le mode et la carte, par un appel
 * à `buildMatchHeadingStr` — la même fonction, sur les mêmes arguments, que celui dont la page
 * tire le libellé du fil d'Ariane. « Slayer sur Streets » s'imprimait donc DEUX fois, à 50 px
 * d'écart. Le fil garde la phrase (c'est sa fonction : dire où l'on est) ; ce composant ne
 * porte plus que ce qu'il ajoutait vraiment. Ce n'est pas un rétrécissement de service, c'est
 * la suppression d'un doublon : la page compacte sa tête, elle ne perd aucune information.
 *
 * ÇA NE COÛTE RIEN : les deux champs sont déjà EN MÉMOIRE. La page monte déjà la vue du
 * match (même clé de cache que la page match) pour son tableau de scores et son fil ; elle
 * n'en lisait simplement pas l'en-tête. Aucune requête de plus, aucune clé i18n de plus non
 * plus — ces libellés sont localisés par le serveur.
 *
 * COMPOSANT DE PRÉSENTATION PUR : ni lecture de store, ni requête, ni règle métier.
 *
 * RIEN TANT QU'ON NE SAIT RIEN : la vue du match peut être en vol quand le rejeu, lui, est
 * déjà là. Dans ce trou, on n'affiche RIEN — pas de squelette : une ligne grise promettrait
 * une information qu'on n'a pas encore. Il vit désormais SUR la ligne du fil d'Ariane, dont
 * la hauteur est tenue par les autres segments : son arrivée tardive ne décale donc plus rien,
 * et la réserve de hauteur que la page portait pour lui (`min-h-6`) a disparu avec le bloc.
 */
import { Badge } from '@/components/ui/badge'

interface ReplayMatchRecallProps {
  startTimeLabel: string | undefined
  playlistLabel: string | undefined
}

export function ReplayMatchRecall({ startTimeLabel, playlistLabel }: ReplayMatchRecallProps) {
  if (!startTimeLabel && !playlistLabel) return null

  return (
    <>
      {/* Le point médian est de la PONCTUATION, pas un libellé : il ne se traduit pas, et il
          ne se lit pas — il n'apparaît que s'il a bien quelque chose à séparer de la feuille
          du fil d'Ariane, qui le précède toujours. */}
      {startTimeLabel && (
        <>
          <span className="select-none text-muted-foreground/50" aria-hidden="true">
            ·
          </span>
          <p className="shrink-0 text-sm text-muted-foreground">{startTimeLabel}</p>
        </>
      )}
      {playlistLabel && (
        <Badge variant="outline" className="shrink-0 text-xs">
          {playlistLabel}
        </Badge>
      )}
    </>
  )
}
