/**
 * Les classes des pilules de réglage des vues 2D (étiquette atténuée + valeurs, ou bouton seul), partagées par
 * le bandeau du plan de l'onglet Tactique et par « Occupation du terrain ».
 */
export const PILULE =
  'inline-flex h-7 items-stretch overflow-hidden rounded-md border border-input bg-background text-[11.5px] font-normal'
export const ETIQUETTE = 'flex items-center whitespace-nowrap border-r border-input bg-muted px-1.5 text-muted-foreground'
/** La valeur d'une pilule : retenue = fond primaire. */
export const VALEUR =
  'px-[7px] text-foreground aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground'
/** Un réglage en bouton seul (la forme de « Zones nommées »). */
export const BOUTON_REGLAGE =
  'inline-flex h-7 items-center justify-center rounded-md border border-input bg-background px-2.5 text-[11.5px] font-normal leading-none text-muted-foreground aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground'
