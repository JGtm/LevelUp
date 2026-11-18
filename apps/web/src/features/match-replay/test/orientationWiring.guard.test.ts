/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LE CANVAS DONNE L'ORIENTATION ET LE PLAN DES MURS À CEUX QUI EN ONT BESOIN.
 *
 * POURQUOI UN TEST DE TEXTE (même raison que `playerFocusWiring.guard`) : le canvas n'a pas de test
 * de rendu. Retirer l'une de ces liaisons laisserait tous les tests de calque verts :
 *  - sans le choix retenu au cadrage, la carte ne tourne plus, alors que la commande répond ;
 *  - sans l'orientation EFFECTIVE (celle que `useReplayView` résout, Auto comprise) à Suivre, la carte reste
 *    entière après une rotation jusqu'au prochain déplacement ;
 *  - UN SEUL POINT DE RÉSOLUTION : la règle Auto n'est appliquée que par `useAutoTurned` (`lib/replay`, gardé là-bas),
 *    que le rejeu appelle par `useResolvedTurns` (appelé par `useReplayView`), et le canvas ne lit le choix retenu
 *    qu'une fois, pour le lui donner ;
 *  - un plan cuit quand son interrupteur est éteint recuit un calque invisible à chaque zoom.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire, nomDe, sourcesDeLaFeature } from './featureFiles'

const canvas = () => lire(fichierNomme('ReplayCanvas.tsx'))

describe('garde-rail : orientation et plan des murs câblés dans le canvas', () => {
  it('le cadrage prend le choix retenu pour la carte (null = Auto)', () => {
    expect(canvas()).toContain('turns: orientation.retained')
    expect(canvas()).toMatch(/useReplayView\(\{[^}]*turns: orientation\.retained/)
  })

  it('Suivre recadre au changement d’orientation, sur l’orientation EFFECTIVE du cadrage', () => {
    expect(canvas()).toMatch(/const \{[^}]*\bturns\b[^}]*\} = useReplayView\(/)
    expect(canvas()).toMatch(/useReplayPlayerFocus\(\{[^}]*\bturns \}\)/)
  })

  it('le canvas ne lit le choix retenu qu’une fois, et le cadrage rend l’orientation effective', () => {
    expect(canvas().match(/orientation\.retained/g)).toHaveLength(1)
    expect(lire(fichierNomme('useReplayView.ts'))).toMatch(/useResolvedTurns\(scopeKey, choice, bounds, width, freeHeight\)/)
  })

  it('l’état de l’Auto est rangé avec le match affiché : le canvas donne sa portée au cadrage', () => {
    expect(canvas()).toMatch(/useReplayView\(\{[^}]*scopeKey: matchId/)
  })

  it('la règle Auto ne s’applique pas dans la feature : le cadrage passe par useResolvedTurns, qui passe par useAutoTurned', () => {
    const appelants = sourcesDeLaFeature()
      .filter((f) => /\bautoTurns\(/.test(lire(f)))
      .map(nomDe)
    expect(appelants).toEqual([])
    expect(lire(fichierNomme('useResolvedTurns.ts'))).toContain('useAutoTurned(')
  })

  it('le panneau de la vue reçoit l’orientation', () => {
    expect(canvas()).toMatch(/useReplayViewMenu\(\{\s*settings, orientation,/)
  })

  it('la variante du fond posé est celle RÉELLEMENT chargée, pas celle qui est demandée', () => {
    const page = lire(fichierNomme('ReplayPage.tsx'))
    expect(page).toMatch(/const \{ image: mapImage, variante \} = useReplayMapImage\(/)
    expect(page).toMatch(/buildMapBackgroundLayer\(\{[^}]*\bimage: mapImage, variante,/)
    expect(lire(fichierNomme('mapBackgroundLayerLogic.ts'))).toMatch(/plan: normalizeWallPlan\(background\.plan\),\s*variante,/)
    expect(canvas()).toContain("shown: mapImage?.variante === 'plan'")
  })

  it('le plan des murs n’est cuit que quand son interrupteur est allumé et que le fond n’est pas déjà le plan', () => {
    expect(canvas()).toContain("const wallPlanOn = isWallPlanDrawn(showWallPlan, mapImage?.variante ?? 'rendu')")
    expect(canvas()).toContain('plan: wallPlanOn ? (mapImage?.plan ?? null) : null')
  })

  it('la probabilité de réapparition est rognée à l’image plan, comme les zones nommées, et seulement s’il y a une vue plan', () => {
    const c = canvas()
    expect(c).toContain('const planDuFond = mapImage?.zonesClip ?? null')
    expect(c).toMatch(/useReplayRespawnForecast\(\{[^}]*clipMask: usePlanClipMask\(planDuFond\)/s)
    expect(c).toContain('plan: planDuFond }')
  })
})
