/**
 * CollapsedItemsToggle — le bouton « Voir plus (N) / Replier » du repli « game changers ».
 *
 * FOYER CANONIQUE (règle ≤2 copies, CLAUDE.md n°6 ; plan `.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md`,
 * lot H, décision H-D5). Deux copies locales identiques existaient depuis les lots G1/G2
 * (`MatchEquipmentUsageSection.CollapsedColumnsToggle`, `MatchPadControlSection.CollapsedWeaponsToggle`)
 * — le plan les acceptait explicitement jusqu'à une 3e copie ; le lot H commande leur migration
 * ICI, qu'une 3e copie apparaisse ou non ailleurs (H2, tranché).
 *
 * AGNOSTIQUE DE FEATURE PAR CONSTRUCTION : aucun import de dictionnaire i18n. Chaque appelant
 * lui passe ses propres libellés déjà résolus dans sa langue (manifeste TOML ou dictionnaire
 * local — H-D5) — ce composant ne sait pas dans quelle feature il vit, donc ne viole jamais la
 * frontière `lint-cross-feature-imports` ni la frontière inversée `components/` → `features/`.
 *
 * N=0 -> AUCUN BOUTON, posé ICI (pas chez l'appelant) : la règle de la grammaire du repli
 * (« rien ne ment ») veut qu'un repli vide ne s'affiche jamais comme un repli — un appelant qui
 * oublierait la garde `count > 0` ne peut plus rendre de bouton fantôme.
 */
export interface CollapsedItemsToggleProps {
  /** Nombre d'éléments masqués derrière le repli. <= 0 -> le composant ne rend rien. */
  count: number
  /** État déplié/replié, posé par l'appelant (jamais persisté — décision D3 du plan). */
  expanded: boolean
  onToggle: () => void
  /** Libellé replié, déjà formaté par l'appelant avec le compte (ex. « Voir plus (3) »). */
  showLabelFmt: (count: number) => string
  /** Libellé déplié (ex. « Replier »). */
  hideLabel: string
  /** Infobulle (title) : la promesse du repli — rien n'est supprimé, tout reste compté. */
  hint: string
  className?: string
}

const DEFAULT_CLASS_NAME =
  'text-xs font-normal text-muted-foreground underline-offset-2 hover:text-foreground hover:underline'

export function CollapsedItemsToggle({
  count,
  expanded,
  onToggle,
  showLabelFmt,
  hideLabel,
  hint,
  className,
}: CollapsedItemsToggleProps) {
  if (count <= 0) return null
  return (
    <button
      type="button"
      className={className ?? DEFAULT_CLASS_NAME}
      title={hint}
      onClick={onToggle}
    >
      {expanded ? hideLabel : showLabelFmt(count)}
    </button>
  )
}
