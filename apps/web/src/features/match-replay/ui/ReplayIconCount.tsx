/**
 * ReplayIconCount — le CHIFFRE d'un compte (charges d'un équipement, stock d'un type de
 * grenade), en surimpression sur le coin bas droit d'une vignette.
 *
 * UN SEUL RENDU POUR TOUS LES COMPTES de la rangée d'inventaire : la cellule de capacité et
 * la puce de grenade le posent tous deux, jamais une copie de son style.
 *
 * Posé hors du flux (`absolute`) : le parent est la boîte `relative` de la vignette, et le
 * chiffre n'ajoute aucune largeur à la rangée. Une pastille de fond tient le chiffre lisible
 * sur n'importe quelle vignette, en thème clair comme sombre (jetons `background` et
 * `foreground`, qui basculent ensemble). Il laisse passer le pointeur : l'infobulle est celle
 * de la vignette dessous.
 *
 * `fade` est l'estompage propre de SA lecture. Absent, le chiffre hérite de l'opacité de la
 * boîte qui le porte (la boîte de grenades pâlit déjà tout entière avec son âge).
 *
 * `outsetPx` sort le chiffre de la vignette vers la droite, de ce nombre de pixels. La boîte qui
 * le porte doit réserver cette place, sans quoi il serait coupé.
 */
export function ReplayIconCount({
  count,
  fade,
  outsetPx,
}: {
  count: number
  fade?: number
  outsetPx?: number
}) {
  return (
    <span
      aria-hidden="true"
      className="pointer-events-none absolute bottom-0 right-0 rounded-sm bg-background/75 px-[2px] font-mono text-[9px] font-semibold leading-none tabular-nums text-foreground"
      style={{ opacity: fade, right: outsetPx === undefined ? undefined : -outsetPx }}
    >
      {count}
    </span>
  )
}
