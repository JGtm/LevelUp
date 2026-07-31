import { describe, expect, it } from 'vitest'

import { familyOf } from './shotEffects'

describe('familyOf', () => {
  it('classe les armes du catalogue par nature de décharge', () => {
    expect(familyOf('S7 Sniper')).toBe('ballistic')
    expect(familyOf('Pulse Carbine')).toBe('plasma')
    expect(familyOf('Sentinel Beam')).toBe('light')
    expect(familyOf('Shock Rifle')).toBe('shock')
    expect(familyOf('M41 SPNKr')).toBe('explosive')
    expect(familyOf('Energy Sword')).toBe('melee')
    expect(familyOf('Needler')).toBe('needles')
  })

  it('ne rapproche JAMAIS une arme inconnue d’une famille voisine', () => {
    // Un rendu emprunté affirmerait une arme qu'on ignore — même faute qu'un visuel par défaut.
    expect(familyOf('Arme Inventée 9000')).toBe('plain')
  })

  it('sans libellé, ne suppose aucune famille', () => {
    expect(familyOf(undefined)).toBe('plain')
    expect(familyOf('')).toBe('plain')
  })

  it('couvre les 22 armes que l’artefact du film de référence nomme', () => {
    // Si le catalogue d'armes s'enrichit sans que ce fichier suive, les nouvelles tombent en
    // `plain` : c'est le comportement voulu, mais ce test rappelle le contrat.
    const duFilm = [
      'S7 Sniper', 'Skewer', 'Cindershot', 'Heatwave', 'BR75', 'Pulse Carbine', 'MA40 AR',
      'Energy Sword', 'M41 SPNKr', 'Mangler', 'CQS48 Bulldog', 'Disruptor', 'Shock Rifle',
      'MLRS-2 Hydra', 'Needler', 'Mk51 Sidekick', 'VK78 Commando', 'Sentinel Beam',
      'Gravity Hammer', 'Plasma Pistol', 'Ravager', 'Stalker Rifle',
    ]
    const sansFamille = duFilm.filter((w) => familyOf(w) === 'plain')
    expect(sansFamille).toEqual([])
  })
})
