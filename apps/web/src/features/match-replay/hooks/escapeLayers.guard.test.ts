/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : TOUTE COUCHE DU REJEU QUI SE FERME À ÉCHAP SE SIGNALE AU MODE PLEIN ÉCRAN.
 *
 * POURQUOI. En superposition seule, Échap ferme d'abord ce qui est ouvert le plus à l'intérieur
 * (panneau de la vue, fiche de joueur, média agrandi, menu de vitesse), le mode ensuite. Le mode ne connaît pas ces couches :
 * il voit seulement, dans le DOM, une racine porteuse de `data-replay-escape-layer`
 * (`REPLAY_ESCAPE_LAYER_ATTR`, cf. useReplayFullscreen). Une couche neuve qui écoute Échap sans
 * porter l'attribut ferait sortir du mode ET se fermerait d'une seule frappe — la faute ne se
 * verrait qu'en plein écran, au clavier.
 *
 * LA RÈGLE : un fichier source du rejeu qui ferme une couche à Échap — par `useDismissable`, le seul
 * écouteur d'Échap permis dans un `.tsx` du rejeu (`features/dismissableLayers.guard.test.ts`), ou par
 * un test direct de la touche Échap (`===`, `!==`, `==`, `!=`, guillemets simples ou doubles, dans
 * un sens ou dans l'autre) — écrit aussi l'attribut en clair sur sa racine. Le mode lui-même lit
 * Échap : il est la seule exemption, nommée, car il n'est pas une couche.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, sourcesDeLaFeature } from '../test/featureFiles'
import { REPLAY_ESCAPE_LAYER_ATTR } from './useReplayFullscreen'

const COMPARAISON = '(?:===|!==|==|!=)'
const ECHAP = `['"]Escape['"]`
/** Un test de la touche (`x.key <op> 'Escape'` ou `'Escape' <op> x.key`) ou un appel de `useDismissable`. */
const ECOUTE_ECHAP = new RegExp(
  `\\bkey\\s*${COMPARAISON}\\s*${ECHAP}|${ECHAP}\\s*${COMPARAISON}\\s*[\\w.?]*\\bkey\\b|\\buseDismissable\\(`,
)
const SIGNALEE = `${REPLAY_ESCAPE_LAYER_ATTR}=""`

/** Le mode plein écran : il lit Échap pour sortir du mode, et n'est pas une couche. */
const LE_MODE = 'hooks/useReplayFullscreen.ts'

function couchesQuiEcoutentEchap(): string[] {
  return sourcesDeLaFeature().filter((f) => cheminCourt(f) !== LE_MODE && ECOUTE_ECHAP.test(lire(f)))
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

  it('auto-preuve : le motif voit les quatre comparaisons, dans les deux sens et sous les deux guillemets, pas une autre touche', () => {
    const vu = (code: string): boolean => ECOUTE_ECHAP.test(code)
    expect(vu("if (e.key === 'Escape') close()")).toBe(true)
    expect(vu('if (e.key === "Escape") close()')).toBe(true)
    expect(vu("if (e.key !== 'Escape') return")).toBe(true)
    expect(vu('if (event.key !== "Escape") return')).toBe(true)
    expect(vu("if (e.key == 'Escape') close()")).toBe(true)
    expect(vu("if (e.key != 'Escape') return")).toBe(true)
    expect(vu("if ('Escape' === e.key) close()")).toBe(true)
    expect(vu("if ('Escape' !== event.key) return")).toBe(true)
    expect(vu('useDismissable(open, onClose)')).toBe(true)
    expect(vu("if (e.key === 'Enter') return")).toBe(false)
    expect(vu("const s = 'Escape'")).toBe(false)
  })

  it('le mode plein écran est la seule exemption : il lit Échap et n’est pas une couche', () => {
    const lisentEchap = sourcesDeLaFeature()
      .filter((f) => ECOUTE_ECHAP.test(lire(f)))
      .map(cheminCourt)
    expect(lisentEchap).toContain(LE_MODE)
    expect(couchesQuiEcoutentEchap().map(cheminCourt)).not.toContain(LE_MODE)
  })

  it('et il y en a bien — panneau de la vue, fiche de joueur, média agrandi, menus de vitesse et de filtre — sans quoi ce garde ne garderait rien', () => {
    expect(couchesQuiEcoutentEchap().map(cheminCourt).sort()).toEqual(
      expect.arrayContaining([
        'settings/ReplayViewPanel.tsx',
        'ui/ReplayFeedFilterMenu.tsx',
        'ui/ReplayMediaLightbox.tsx',
        'ui/ReplayPlayerPanel.tsx',
        'ui/ReplaySpeedMenu.tsx',
      ]),
    )
  })
})
