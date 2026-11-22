/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, « à la 3e copie, centraliser » + garde-rail) : LES COTES D'UNE
 * FICHE VIVENT DANS `model/cardGabarit.ts`, et nulle part ailleurs.
 *
 * POURQUOI. Sept constantes de cotes vivaient dans quatre fichiers de `ui/` (`WEAPON_CELL_W`,
 * `AMMO_CELL_W`, `GRENADES_BOX_W`, `SCORE_CELL_W`, `COUNT_CELL_W`, `GRENADE_ICON_PX`,
 * `HUD_ICON_PX`, `WATERMARK_PX`). Le 2026-09-06 (plan fiches compactes, I1) elles sont
 * devenues les champs d'un objet `CardGabarit` — et la seule façon pour qu'un gabarit reste
 * l'unique foyer de ses cotes, c'est qu'aucune cote ne REVIENNE en constante locale dans un
 * composant : le jour où elle revient, elle ne vaut plus que pour ce composant.
 *
 * TROIS INTERDITS, sur les composants de la fiche :
 *   1. une déclaration `const X_CELL_W = …`, `X_BOX_W`, `X_ICON_PX`, `X_PX` — la forme exacte
 *      des constantes migrées ;
 *   2. un littéral de densité (`'compacte'` / `'normale'`) — les composants reçoivent des
 *      nombres et des booléens, jamais un mot (la forme que `10fe3228a` a supprimée) ;
 *   3. un import depuis `../settings/` — la densité n'est pas un réglage, sans drapeau.
 *
 * Et la contre-épreuve : le foyer canonique déclare bien les gabarits qui existent, et aucun gabarit
 * normal (sans lecteur) n'y revient.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, fichierNomme, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

/** Les composants de la fiche : tout ce qui rend une cote du gabarit. */
const COMPOSANTS_DE_LA_FICHE = new Set([
  'ReplayPlayerCard.tsx',
  'ReplayCardFx.tsx',
  'ReplayWeaponsRow.tsx',
  'ReplayInventoryRow.tsx',
  'ReplayCountersBadge.tsx',
  'ReplayAbilityCell.tsx',
  'ReplayVitality.tsx',
  'ReplayObjectiveMark.tsx',
  // La ligne de la colonne des équipes (gabarit « ligne ») : ses cotes viennent de `GABARIT_LIGNE`.
  'ReplayRosterRow.tsx',
  'ReplayRosterCells.tsx',
  'ReplayRosterColumn.tsx',
  'ReplayRosterRowDetail.tsx',
  // La fiche compacte de la Grande équipe : ses cotes viennent de `GABARIT_FICHE_COMPACTE`.
  'ReplayCompactCard.tsx',
  // La rangée des équipes du Cinéma et du Plein cadre : ses cotes viennent de `COTES_RANGEE` et `COTES_RANGEE_GRANDE_EQUIPE`.
  'ReplayTeamsRow.tsx',
])

const COTE_LOCALE = /\bconst \w+_(CELL_W|BOX_W|ICON_PX|PX) =/
const MOT_DE_DENSITE = /['"](compacte|normale)['"]/
/**
 * Sur toute la feature, la forme d'un LITTÉRAL TypeScript (guillemets simples, la convention du
 * dépôt) : deux commentaires citent une « explosion "normale" » entre guillemets doubles
 * (`vehiclesPaint.ts`, `vehiclesLayer.ts`) — de la prose, pas un discriminant.
 */
const LITTERAL_DE_DENSITE = /'(compacte|normale)'/
const IMPORT_REGLAGES = /from '\.\.\/settings\//

function composants(): string[] {
  return sourcesDeLaFeature().filter((f) => COMPOSANTS_DE_LA_FICHE.has(nomDe(f)))
}

describe('garde-rail : les cotes de la fiche vivent dans model/cardGabarit.ts', () => {
  it('les quatorze composants de la fiche, de la ligne et de la rangée existent tous — sans quoi ce garde ne garderait rien', () => {
    expect(new Set(composants().map(nomDe))).toEqual(COMPOSANTS_DE_LA_FICHE)
  })

  it('aucune constante de cote locale dans un composant de la fiche', () => {
    const fautifs = composants().filter((f) => COTE_LOCALE.test(lire(f))).map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('aucun mot de densité dans un composant de la fiche : des nombres et des booléens', () => {
    const fautifs = composants().filter((f) => MOT_DE_DENSITE.test(lire(f))).map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('aucun import des réglages dans un composant de la fiche : la densité n’est pas un réglage', () => {
    const fautifs = composants().filter((f) => IMPORT_REGLAGES.test(lire(f))).map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('le mot de densité n’est admis nulle part dans le rejeu', () => {
    const fautifs = sourcesDeLaFeature()
      .filter((f) => LITTERAL_DE_DENSITE.test(lire(f)))
      .map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('et le foyer canonique déclare bien les trois gabarits, les cotes du bloc « À l’instant » et celles de la rangée, sans gabarit normal', () => {
    const src = lire(fichierNomme('cardGabarit.ts'))
    expect(src).not.toMatch(/GABARIT_NORMAL/)
    expect(src).toMatch(/export const GABARIT_COMPACT: CardGabarit/)
    expect(src).toMatch(/export const COTES_BLOC_INSTANT = /)
    expect(src).toMatch(/export const GABARIT_LIGNE: LigneGabarit/)
    expect(src).toMatch(/export const GABARIT_FICHE_COMPACTE: FicheCompacteGabarit/)
    expect(src).toMatch(/export const COTES_RANGEE = /)
    expect(src).toMatch(/export const COTES_RANGEE_GRANDE_EQUIPE = /)
  })
})
