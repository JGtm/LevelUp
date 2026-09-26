# Revue adversariale de la vague 5 — 2026-09-10

Un relecteur en contexte frais (lentilles L1 anti-ART, L4 données, L6 couverture), une ronde,
règles du skill `adversarial-review`. Diff relu : `c9096ae53...6d6ded5ba` (lots 5.1 collecteur
de sync + garde de schéma + compteur d'alarme, 5.5 « utilisé » sur les consommations (us6),
5.6 palette unanimité + bruit i48, 5.7 « gardé / utilisé » web). Le lot 5.3 (hygiène XS,
mécanique) est couvert par la delivery-checklist seule.

## Constats

- **P1 — `internal/analysis/sessionusage/usage_outcomes.go:83-92`** : `equipmentUsedOf` lisait
  encore les poses pour toutes les familles ; sous us6 la page Sessions/Synthèse et la vue match
  disaient deux vérités pour le même match (`taken=3, spent=2, dropped=1` → utilisé 0 côté
  session, 2 côté match), `metricValue` perdait des objets et `metricKeys` pouvait faire sortir
  la famille ; `PlayerRow.SpentByFamily` chargée sans lecteur ; doc inversée. Quatre tests
  épinglaient l'ancienne règle. **Corrigé en ronde de corrections (`wt/corrections-v5`).**
- **P1 — `apps/web/src/features/match-replay/model/equipmentUsageLogic.ts:438`** : le filtre
  des colonnes d'équipement ignorait `spent` ; un capteur pris puis consommé (gardé 0, ni pose ni
  lâcher) vidait `columns.equipment` et le groupe disparaissait de la vue match. Mutation
  prouvée (`expect(columns.equipment).toEqual(['sensor'])` rouge). **Corrigé en ronde de
  corrections.**
- **P2 — `equipmentKeptLogic.ts:147`** : `equipmentChangeFamilyOf` sans appelant de production,
  épinglée par 7 tests verts (code mort avec tests). **Corrigé en ronde de corrections** (un
  seul chemin de lecture de la famille).

## Conditions vérifiées qui tiennent : 14

Écritures `match_lives` inchangées (BatchBuilder, aucune allowlist) ; aucune vie de bot en base
(test explicite) ; axe de temps des participants du collecteur = formule canonique du lecteur
de faits (règle n° 8) ; `bid(N.0)` filtrés du roster numérique ; projection de bots unique +
ratchet ; empreinte du décodeur inchangée à raison ; compteur d'alarme mordant (mutation) ;
garde-rail des familles à pièce engendrée lisant réellement le manifeste (mutation) ; mur sur
ses panneaux, bonus inchangés, `kept ≥ 0` des deux côtés ; périmètre du bilan clos (8 familles
des deux côtés) ; 5.6 sans bump (omitempty, openapi additif, golden +1 ligne, mutation du
plafond de rang) ; multi-titre, couleurs, i18n, seuils.

## Bilan

P0 = 0, P1 = 2, P2 = 1 — tous corrigés dans la ronde de corrections, relus en ronde 2 sur les
seules corrections. Puis `make gate-push`, push de `feat/v75` = CI de vague.
