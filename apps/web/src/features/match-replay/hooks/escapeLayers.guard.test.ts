/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : TOUTE COUCHE DU REJEU QUI SE FERME À ÉCHAP SE SIGNALE AU MODE PLEIN ÉCRAN.
 *
 * POURQUOI. En superposition seule, Échap ferme d'abord ce qui est ouvert le plus à l'intérieur
 * (tiroir, média agrandi, menu de vitesse), le mode ensuite. Le mode ne connaît pas ces couches :
 * il voit seulement, dans le DOM, une racine porteuse de `data-replay-escape-layer`
 * (`REPLAY_ESCAPE_LAYER_ATTR`, cf. useReplayFullscreen). Une couche neuve qui écoute Échap sans
 * porter l'attribut ferait sortir du mode ET se fermerait d'une seule frappe — la faute ne se
 * verrait qu'en plein écran, au clavier.
 *
 * LA RÈGLE : un fichier source du rejeu qui teste `key === 'Escape'` écrit aussi l'attribut en
 * clair sur sa racine. Le mode lui-même teste `!==` : il n'est pas une couche.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, sourcesDeLaFeature } from '../test/featureFiles'
import { REPLAY_ESCAPE_LAYER_ATTR } from './useReplayFullscreen'

const ECOUTE_ECHAP = /\.key === 'Escape'/
const SIGNALEE = `${REPLAY_ESCAPE_LAYER_ATTR}=""`

function couchesQuiEcoutentEchap(): string[] {
  return sourcesDeLaFeature().filter((f) => ECOUTE_ECHAP.test(lire(f)))
}

describe('garde-rail : les couches fermées à Échap se signalent au mode plein écran', () => {
  it('chaque écouteur d’Échap du rejeu pose l’attribut sur sa couche', () => {
    const muettes = couchesQuiEcoutentEchap()
      .filter((f) => !lire(f).includes(SIGNALEE))
      .map(cheminCourt)
    expect(
      muettes,
      `ces couches se ferment à Échap sans le dire au mode plein écran : [${muettes.join(', ')}]. ` +
        `Poser ${SIGNALEE} sur leur racine (cf. hooks/useReplayFullscreen.ts).`,
    ).toEqual([])
  })

  it('et il y en a bien — tiroir, média agrandi, menu de vitesse — sans quoi ce garde ne garderait rien', () => {
    expect(couchesQuiEcoutentEchap().map(cheminCourt).sort()).toEqual(
      expect.arrayContaining([
        'settings/ReplaySettingsDrawer.tsx',
        'ui/ReplayMediaLightbox.tsx',
        'ui/ReplaySpeedMenu.tsx',
      ]),
    )
  })
})
