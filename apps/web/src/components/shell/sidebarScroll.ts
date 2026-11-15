/**
 * Défilement des listes de sections de la barre latérale (dépliée et repliée) : barre de défilement fine,
 * teintée comme les bordures, placée à gauche. `direction: rtl` sur le conteneur qui défile y range la barre ;
 * le contenu repasse en `direction: ltr` (`SIDEBAR_SCROLL_CONTENT_CLASS`), sans quoi le texte et l'alignement
 * des entrées s'inversent.
 */
export const SIDEBAR_SCROLL_CLASS = '[scrollbar-width:thin] [scrollbar-color:var(--border)_transparent] [direction:rtl]'

/** Classe de l'enfant direct du conteneur `SIDEBAR_SCROLL_CLASS` : rétablit le sens de lecture. */
export const SIDEBAR_SCROLL_CONTENT_CLASS = '[direction:ltr]'
