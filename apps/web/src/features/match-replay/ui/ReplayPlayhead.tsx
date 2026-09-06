/**
 * ReplayPlayhead — LE TRAIT DE LECTURE de la frise.
 *
 * EXTRAIT DE `ReplayTimelineTracks.tsx` LE 2026-09-06, à sa naissance : le composant hôte
 * franchissait les 500 lignes (`max-lines`, règle n° 5 du dépôt), et la règle est d'extraire
 * plutôt que de relever le plafond. La découpe tombe sur une frontière nette — l'hôte compose
 * les rangées, ce fichier ne s'occupe que de la verticale qui les traverse.
 */
import { CURSOR_RATIO_VAR } from '../hooks/useReplayPlayback'
import { trackLeftVar } from '../model/replayTimelineTracksLogic'
import { TRACKS_COLUMN_LEFT } from './replayTimelineGrid'

/**
 * LE TRAIT DE LECTURE — un seul trait vertical, à l'aplomb du curseur, qui traverse toutes les
 * pistes (demande utilisateur du 2026-09-06).
 *
 * CE QU'IL RÉPARE : une marque de kill et l'instant lu se comparaient à l'œil, sur vingt à
 * soixante pixels de hauteur, en descendant jusqu'au curseur puis en remontant. Le trait fait de
 * cette comparaison une coïncidence — la marque est traversée, ou elle ne l'est pas.
 *
 * IL PARTAGE LA FORMULE DES MARQUES, et c'est la seule chose qui le rend juste : `trackLeftVar`
 * rend la position du ratio courant dans la géométrie du curseur natif, exactement comme
 * `trackLeft` la rend pour un ratio connu au rendu. Un trait posé à `left: var(--played)` — le
 * pourcentage brut du remplissage — dériverait jusqu'à 8 px du kill qu'il désigne, et le ferait
 * le plus près des deux bouts, là où l'on cherche justement le premier et le dernier frag. Le
 * garde-rail `timelineGeometry.guard.test.ts` interdit de recopier la formule ici.
 *
 * IL PARTAGE AUSSI LEUR ANCRE, et pas seulement leur formule : le point rendu par la formule est
 * le CENTRE des trois objets — la pastille du curseur (que le navigateur y centre lui-même), la
 * marque de kill (centrée par sa demi-largeur depuis la décision du 2026-09-06, cf. `MarkTrack`)
 * et ce trait. Il coupe donc chaque marque en son milieu au lieu d'en longer le bord, et c'est ce
 * qui rend la coïncidence lisible : deux objets alignés sur des ancres différentes se seraient
 * frôlés de un à deux pixels à chaque frag, sans que rien ne dise lequel des deux mentait.
 *
 * LUI, IL N'A PAS BESOIN D'ÊTRE TRANSLATÉ pour cela : il fait un pixel de large, sa gauche et son
 * centre sont le même endroit. La translation vit là où une largeur la rend nécessaire.
 *
 * CE QU'IL NE TRAVERSE PAS. La colonne des libellés : il commence au bord gauche de la colonne
 * des pistes (`TRACKS_COLUMN_LEFT`) — un nom barré d'un trait ne se lit plus. Et la pastille du
 * curseur : son conteneur est la grille des PISTES, qui s'arrête une rangée au-dessus du
 * transport (cf. l'en-tête de `ReplayTimelineTracks.tsx`, § des deux grilles). Le trait meurt
 * donc au-dessus du curseur sans avoir à connaître la hauteur de quoi que ce soit — c'est
 * `inset-y-0` qui le dit, pas un nombre de pixels qu'une rangée de plus rendrait faux.
 *
 * IL NE CAPTE PAS LE POINTEUR ET N'A PAS DE NOM ACCESSIBLE : la frise reste saisissable au pixel
 * près sous lui (règle de toutes les pistes), et il ne dit rien que le champ n'expose déjà —
 * sa position EST la valeur du curseur.
 *
 * SON ENCRE EST CELLE DU CURSEUR (`--foreground` : la pastille et le remplissage), atténuée.
 * C'est le même objet qui se prolonge vers le haut, pas un repère de plus — à pleine encre il
 * pèserait plus lourd que les marques qu'il sert à lire. Couleur STRUCTURELLE et non sémantique,
 * exception assumée du skill `color-tokens` : les deux encres disponibles ici disent un CAMP
 * (`team-ally`, `team-enemy`), et le trait n'en désigne aucun.
 */
export function ReplayPlayhead() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-y-0 right-0"
      style={{ left: TRACKS_COLUMN_LEFT }}
    >
      <span
        className="absolute inset-y-0 w-px bg-foreground/40"
        style={{ left: trackLeftVar(CURSOR_RATIO_VAR) }}
      />
    </div>
  )
}
