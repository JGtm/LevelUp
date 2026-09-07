/**
 * ReplaySettingsToggle — LES DEUX COMMANDES du tiroir de reglages, et la marque « (i) » de
 * reserve de mesure qui les accompagne parfois.
 *
 * EXTRAIT DE `ReplaySettingsDrawer.tsx` LE 2026-08-18 (lot R2-V) : le tiroir a franchi le seuil
 * de taille du depot (CLAUDE.md n°5) en recevant la bascule de la trainee et la portee de la
 * carte de chaleur. La decoupe tombe sur une frontiere nette — ce fichier ne connait AUCUN
 * reglage du rejeu, il ne sait que dessiner un interrupteur ; le tiroir, lui, ne sait plus
 * comment un interrupteur se dessine.
 *
 * DEUX COMMANDES, PARCE QU'IL Y A DEUX QUESTIONS (2026-08-29). Le tiroir posait les deux avec
 * le MÊME bouton, et c'est ce que l'utilisateur a relevé : « pour les réglages je préfère un
 * toggle plutôt que des boutons comme aujourd'hui ».
 *
 *  - `SettingsToggle` — UN OUI/NON indépendant (un calque, un effet, une catégorie de son, la
 *    lecture automatique). C'est un vrai interrupteur : libellé à gauche, bascule à droite,
 *    `role="switch"` + `aria-checked`, sur le modèle de `ThemeToggle` (components/shell). Une
 *    ligne par réglage — un interrupteur se lit sur son rail, pas dans une grille où deux
 *    voisins allumés se confondent avec un groupe de choix ;
 *  - `SettingsChoice` — UN PARMI N, exclusif (lecture et portée de la carte de chaleur,
 *    couleur des points). Il GARDE le bouton pressé d'avant, et ce n'est pas un oubli : un
 *    interrupteur promet qu'on peut tout éteindre, ce qui n'a pas de sens pour un choix dont
 *    exactement une option est vraie. La forme dit la nature de la question.
 *
 * `SettingsToggle` sert une quinzaine de fois, `SettingsChoice` six : un seul rendu chacun
 * plutôt que vingt copies presque identiques (CLAUDE.md n°6).
 */
import { Button } from '@/components/ui/button'

/**
 * InfoMark — la marque « (i) » d'une RÉSERVE DE MESURE, en SVG et jamais en caractère : le
 * « i cerclé » typographique dépend de la police installée et se rend en carré vide sur les
 * machines qui ne l'ont pas. Elle ne prend pas le focus et n'ouvre rien — tout ce qu'elle a
 * à dire tient dans son infobulle, et le lecteur d'écran lit la même phrase.
 *
 * EXPORTÉE depuis le 2026-08-24 : la marque se pose sur le TITRE de la section du tiroir
 * (demande utilisateur), plus à côté d'une bascule — le tiroir la compose lui-même.
 */
export function InfoMark({ text }: { text: string }) {
  return (
    <span
      role="img"
      aria-label={text}
      title={text}
      className="inline-flex shrink-0 items-center text-muted-foreground"
    >
      <svg
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.1"
        strokeLinecap="round"
        aria-hidden="true"
      >
        <circle cx="6" cy="6" r="4.6" />
        <path d="M6 5.4v3" />
        <path d="M6 3.6v.2" />
      </svg>
    </span>
  )
}

/**
 * SettingsToggle — UN RÉGLAGE OUI/NON : son libellé, et l'interrupteur qui dit son état.
 *
 * `role="switch"` ET NON UN BOUTON PRESSÉ, et la différence n'est pas cosmétique : un lecteur
 * d'écran annonce « interrupteur, activé », c'est-à-dire l'état d'un réglage, là où
 * `aria-pressed` annonce un bouton qu'on a enfoncé. C'est bien un réglage.
 *
 * LE RAIL EST DÉCORATIF (`aria-hidden`) : l'état est déjà porté par `aria-checked` sur le
 * bouton, et le doubler ferait entendre deux fois la même chose.
 */
export function SettingsToggle({
  label, pressed, onToggle, hint,
}: {
  label: string
  pressed: boolean
  onToggle: () => void
  hint?: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={pressed}
      onClick={onToggle}
      title={hint}
      className="flex h-7 w-full items-center justify-between gap-2 rounded px-2 text-left text-xs transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
    >
      <span className="truncate">{label}</span>
      <ToggleTrack on={pressed} />
    </button>
  )
}

/**
 * Le rail et sa pastille — la seule partie DESSINÉE de l'interrupteur.
 *
 * TOKENS SÉMANTIQUES UNIQUEMENT (CLAUDE.md n°12) : `primary` pour l'état actif, `muted` pour
 * l'état éteint. Aucune valeur hex, aucune classe de couleur Tailwind — le tiroir se pose sur
 * la carte, et il doit suivre le thème comme le reste de la page.
 */
function ToggleTrack({ on }: { on: boolean }) {
  return (
    <span
      aria-hidden
      className={`flex h-4 w-7 shrink-0 items-center rounded-full border px-[2px] transition-colors ${
        on ? 'border-primary bg-primary' : 'border-border bg-muted'
      }`}
    >
      {/* LA COURSE EST CALCULÉE, PAS APPROCHÉE : le rail fait 28 px de bord à bord, moins
          2 px de bordure et 4 px de retrait interne, soit 22 px utiles ; la pastille en
          occupe 12, il lui en reste donc 10 à parcourir. Une valeur de l'échelle (`translate-x-3`,
          12 px) la ferait déborder du rail de 2 px à droite. */}
      <span
        className={`h-3 w-3 rounded-full transition-transform duration-150 ${
          on ? 'translate-x-[10px] bg-primary-foreground' : 'translate-x-0 bg-muted-foreground'
        }`}
      />
    </span>
  )
}

/**
 * SettingsChoice — UNE OPTION D'UN CHOIX EXCLUSIF (lecture de la carte de chaleur, portée,
 * couleur des points). Exactement le rendu que TOUTES les commandes du tiroir avaient avant le
 * 2026-08-29 : un bouton plein quand il est retenu, fantôme sinon.
 *
 * IL RESTE UN BOUTON `aria-pressed`, ET C'EST LE POINT : ces options sont mutuellement
 * exclusives, l'une est toujours vraie. Un interrupteur y promettrait un « tout éteint » que
 * le réglage n'accepte pas — et sa pastille éteinte se lirait « désactivé » sur une option qui
 * n'attend qu'un clic pour devenir la lecture courante.
 */
/**
 * SettingsSegments — UN CHOIX EXCLUSIF SUR UNE SEULE LIGNE.
 *
 * POURQUOI IL EXISTE (2026-09-02, retour utilisateur sur « Ce que la chaleur mesure »). Un
 * choix exclusif rendu en `SettingsChoice` empilés coûte une ligne par option, précédée d'un
 * paragraphe gris qui fait office d'étiquette. Deux axes — ce qu'on mesure, sur quelle durée —
 * y prenaient huit lignes, et surtout : des options empilées RESSEMBLENT à des interrupteurs
 * indépendants. On croit pouvoir en allumer deux.
 *
 * Le segmenté dit la contrainte par sa forme : un rail, des cases jointives, une seule
 * enfoncée. Étiquette à gauche, choix à droite — la même grammaire que `SettingsToggle`, dont
 * il est le frère à N états.
 *
 * `role="radiogroup"` et non un groupe de boutons pressés : c'est un choix parmi N, et les
 * technologies d'assistance doivent l'annoncer comme tel.
 */
export function SettingsSegments<T extends string>({
  label, value, options, onSelect,
}: {
  label: string
  value: T
  options: readonly { value: T; label: string; hint?: string }[]
  onSelect: (value: T) => void
}) {
  return (
    <div className="flex items-center justify-between gap-2 py-0.5">
      <span className="shrink-0 text-xs text-muted-foreground">{label}</span>
      <div role="radiogroup" aria-label={label} className="flex rounded-md border border-border p-px">
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={value === o.value}
            title={o.hint}
            onClick={() => onSelect(o.value)}
            className={`cursor-pointer rounded-[5px] px-2 py-1 text-[11px] font-medium transition-colors ${
              value === o.value
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            }`}
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  )
}

export function SettingsChoice({
  label, pressed, onToggle, hint,
}: {
  label: string
  pressed: boolean
  onToggle: () => void
  hint?: string
}) {
  return (
    <Button
      type="button"
      variant={pressed ? 'default' : 'ghost'}
      size="sm"
      onClick={onToggle}
      className="h-7 justify-start px-2 text-xs"
      title={hint}
      aria-pressed={pressed}
    >
      {label}
    </Button>
  )
}
