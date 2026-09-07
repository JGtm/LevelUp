import { describe, expect, it } from 'vitest'

import { familyOf } from './shotEffects'

describe('familyOf', () => {
  // LE CATALOGUE D'ARMES N'EST PLUS ICI (lot 3.2) : le document publie la famille de
  // rendu (`weaponLabels[id].fx`), résolue hors ligne depuis les mappings du titre. Ce
  // fichier ne sait plus ce qu'est un Ravager, et c'est le but — un catalogue Halo dans
  // du code de rendu ne connaît qu'un jeu, et qu'un état de ce jeu.
  //
  // La COUVERTURE des 22 armes du film de référence, elle, est vérifiée là où elle vit
  // désormais : `replay_labels.toml` + le golden d'assemblage Go (colonne `fx`).
  it('accepte les sept familles que le rendu sait dessiner', () => {
    expect(familyOf('ballistic')).toBe('ballistic')
    expect(familyOf('plasma')).toBe('plasma')
    expect(familyOf('light')).toBe('light')
    expect(familyOf('shock')).toBe('shock')
    expect(familyOf('explosive')).toBe('explosive')
    expect(familyOf('melee')).toBe('melee')
    expect(familyOf('needles')).toBe('needles')
  })

  it('ne rapproche JAMAIS une famille inconnue d’une famille voisine', () => {
    // Un rendu emprunté affirmerait une arme qu'on ignore — même faute qu'un visuel par
    // défaut. Le cas est RÉEL : un artefact plus récent que l'app peut nommer une famille
    // que ce fichier ne dessine pas encore.
    expect(familyOf('gravitic')).toBe('plain')
    expect(familyOf('Ballistic')).toBe('plain')
  })

  it('sans famille publiée, ne suppose rien', () => {
    // C'est le cas d'une arme hors catalogue du titre : le document ne lui donne ni nom
    // ni effet, et l'écran la dessine sobrement plutôt que comme sa voisine.
    expect(familyOf(undefined)).toBe('plain')
    expect(familyOf('')).toBe('plain')
  })

  it('`plain` n’est pas une famille publiable : c’est l’absence de famille', () => {
    // Le loader Go refuse `plain` dans replay_labels.toml (liste fermée) ; ici le pendant
    // côté rendu — la valeur ne peut pas venir du document, et si elle venait elle
    // donnerait le même trait neutre.
    expect(familyOf('plain')).toBe('plain')
  })
})
