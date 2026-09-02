/**
 * ReplayZoomControl — LA COMMANDE DE CADRAGE, en surimpression dans un angle de la carte.
 *
 * # OÙ, ET POURQUOI LÀ (demande utilisateur du 2026-09-02)
 *
 * « Le zoom peut être un élément en surimpression dans un angle au niveau de la map. » Elle ne
 * coûte donc AUCUNE place dans la mise en page — ce qui compte sur une page dont le budget
 * vertical est la ressource rare. Coin BAS-DROIT : la légende de la carte de chaleur tient le
 * coin bas-gauche, et deux surimpressions au même endroit se marcheraient dessus.
 *
 * # UNE CROIX, PAS UN GLISSER
 *
 * La demande laissait le choix (« soit le déplacement à la souris soit une croix
 * directionnelle »). La croix gagne pour deux raisons, l'une d'usage et l'autre de coût.
 * D'usage : l'horloge tourne, et glisser pendant que l'action bouge revient à poursuivre le jeu
 * à la souris. De coût : un glisser change le cadrage à chaque mouvement de pointeur, donc
 * recuit les quatre calques statiques à chaque image — la croix va par pas discrets, une
 * cuisson par clic.
 *
 * # LA CROIX SE DÉSACTIVE SEULE À 1x
 *
 * Personne n'écrit la règle « pas de déplacement quand on voit tout » : à 1x la fenêtre vaut la
 * scène, donc il n'existe qu'une position légale (cf. `clampCenter`). `canPan` ne fait que
 * rapporter cette propriété à l'écran.
 */
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayZoom } from './useReplayZoom'

export function ReplayZoomControl({
  zoom,
  locale,
}: {
  zoom: ReplayZoom
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <div
      className="absolute bottom-2 right-3 flex flex-col items-center gap-1 rounded-md border border-border bg-card/85 p-1"
      role="group"
      aria-label={t.zoomGroup}
    >
      <div className="flex items-center gap-0.5">
        <Key onClick={zoom.zoomOut} disabled={!zoom.canZoomOut} label={t.zoomOut}>
          −
        </Key>
        {/* LE NIVEAU EST UN TEXTE, PAS UNE JAUGE : « 2x » se lit d'un coup d'oeil et se dit à
            voix haute. C'est aussi ce qui rend les paliers discrets défendables — une valeur
            continue n'aurait rien à afficher ici. */}
        <span className="min-w-[2.1rem] text-center font-mono text-[10.5px] tabular-nums text-muted-foreground">
          {t.zoomLevelFmt(zoom.level)}
        </span>
        <Key onClick={zoom.zoomIn} disabled={!zoom.canZoomIn} label={t.zoomIn}>
          +
        </Key>
      </div>
      {/* LA CROIX, en grille 3x3 : les cases vides des coins gardent l'alignement sans porter de
          commande. `dy` positif va vers le HAUT de la carte — le monde a son Y vers le haut, la
          toile l'inverse, et c'est `worldToCanvas` qui porte cette inversion, pas la commande. */}
      <div className="grid grid-cols-3 gap-0.5">
        <span />
        <Key onClick={() => zoom.panStep(0, 1)} disabled={!zoom.canPan} label={t.panUp}>
          ▲
        </Key>
        <span />
        <Key onClick={() => zoom.panStep(-1, 0)} disabled={!zoom.canPan} label={t.panLeft}>
          ◀
        </Key>
        <Key onClick={zoom.reset} disabled={!zoom.canZoomOut && !zoom.canPan} label={t.zoomReset}>
          ⌂
        </Key>
        <Key onClick={() => zoom.panStep(1, 0)} disabled={!zoom.canPan} label={t.panRight}>
          ▶
        </Key>
        <span />
        <Key onClick={() => zoom.panStep(0, -1)} disabled={!zoom.canPan} label={t.panDown}>
          ▼
        </Key>
        <span />
      </div>
    </div>
  )
}

/**
 * Une touche de la commande. Le NOM ACCESSIBLE est porté par `aria-label` et `title` : les
 * glyphes de la croix sont des formes, pas des mots, et un lecteur d'écran n'a rien à en dire.
 */
function Key({
  onClick,
  disabled,
  label,
  children,
}: {
  onClick: () => void
  disabled?: boolean
  label: string
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      className={`inline-flex h-5 w-5 items-center justify-center rounded text-[9px] leading-none transition-colors ${
        disabled
          ? 'cursor-not-allowed text-muted-foreground/35'
          : 'cursor-pointer text-muted-foreground hover:bg-accent hover:text-accent-foreground'
      }`}
    >
      {children}
    </button>
  )
}
