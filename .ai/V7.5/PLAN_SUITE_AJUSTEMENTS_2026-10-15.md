# Plan — suite des ajustements du 14/10 (2026-10-15)

Le plan d'origine, clos, est archivé : `.ai/V7.5/PLAN_AJUSTEMENTS_2026-10-14.md`. Ce plan porte les
lots ouverts le 15/10 à la suite des retours du user. Contrat : skill `plan-execution`.
Superviseur : session levelup-66.

## Décisions du user (15/10)

- La faille se situe à sa POSE, lue dans le film, pas par la téléportation.
- Kill refusé, équipe des bots, vies sans nom : enquête puis correction (accord « ok »).
- Firefight : le mode n'est pas géré ; on ne fabrique ni ne sert plus son rejeu.
- Armes au sol : armes de départ lâchées masquées par défaut, option pour les réafficher.
- Lot de correction des morts servies deux fois : accord « oui ».

## Lots

- [x] Armes au sol (`feat/armes-sol-depart`, fusionné ; 31/90 sur 0d9a9af9, 87/221 sur 1b2d9e08).
- [x] Firefight exclu de la cuisson et du service (`fix/rejeu-sans-firefight`, fusionné 424fa1393).
- [x] Nettoyage du cache Firefight (16/10 : edc5daf6 et son raster supprimés ; 775 matchs Firefight au registre).
- [x] Faille située à sa pose (`feat/faille-pose`, session levelup-80) : objet ti=38 né à la pose
  (123/123), repli porteur compté, position par échange retirée, schéma 98 ; fin de la faille
  bornée par la suppression lue de l'objet (en cours).
- [x] K-A doublons d'identité des temps forts et morts fantômes de la passe 2 (`fix/ks-a-doublons`).
- [x] K-D morts servies deux fois (`fix/ks-d-morts-doublees`, contient K-A) : siège de bot repris
  (le bot tient l'indice jusqu'à la fin de sa dernière déclaration close — VALIDÉ par le user le 15/10 : le début d'une déclaration n'est qu'une borne, 7 morts de bot confirmées par l'API),
  appariement du crédit par écart croissant, fiches de kill au kill voisin, filet « une mort d'humain
  = un événement death de l'API ». Relecture adversariale en cours.
- [x] K-D relecture : P1 DELETE sur `killer_victim_pairs` qui porte 3 index ART hérités (idx_kv_*) → échange sans DELETE + migration qui retire les index (validé par le user le 15/10) ; P2 preuves `Known` et ordre des écritures ; WARN rétabli — corrections en cours.
- [x] K-B équipe des bots par build (`fix/ks-b-equipe-bots`) : relecture rendue (P2 « hors
  grammaire » au roster, test de garde sans preuve, `EquipeLue` morte) — corrections en cours.
- [x] K-C vies nommées (`fix/ks-c-vies-nommees`).
- [x] Intégration groupée (feat/integ-film d046f1b84, CI verte ; révisions grammar-2026-10-16, profile-2026-10-16, killsource-2026-10-16, faits 16, schéma 98, placement-2026-10-16, highlight-credit-2026-10-16 ; bcb6d393 12 → 15 joueurs ; fin de faille lue 1 675/1 722 poses ; 0 panique sur 2 676 films) K-D (+K-A), K-B, K-C, faille-pose : renuméroter les révisions (collisions
  grammar-2026-10-15.3, killsource-2026-10-15.2, faits 16) ; RÉGÉNÉRER `bots_*.json`, références
  d'assemblage et fixtures web (bcb6d393 figé avec l'équipe fausse) ; garde-rail « chaque
  `bots_*.json` porte `killsource.Rev` ».
- [x] Rattrapage et recuisson (16/10, 07:12-09:17, serveur arrêté, sauvegarde `data/backups/avant_recuisson_98_2026-10-16/`) : 2 696 lignes fantômes retirées sur 86 matchs, couples recalculés sur 537 matchs, index hérités retirés, 237 artefacts au schéma 98, projections, killsource crédit + films 0 erreur (34db6b33 passe), rasters ; air relancé. ACCORD DU USER le 16/10 : : `repair-highlight-phantoms` (86 matchs) et
  `--couples` (477 matchs), recuisson complète au dernier schéma, `backfill-killsource --credit-only`
  puis `backfill-killsource`, purge Firefight.

## Découvertes

- (intégration) 3f048bf3 (HI_1_13_0) : une vie de bot de 6 images perd son nom et une autre change de bot (deux bots successifs sur l indice 8) — à instruire (règle de fin de déclaration).
- (intégration) Vies sans identité dans les rejeux cuits : 2 376 sur 43 films (5 de build inconnu, 38 de HI_1_9 à HI_1_13), déjà présentes sur feat/v75 (2 379) — contraire à la règle du user, à instruire.
- (intégration) feat/v75 portait un rouge : `TestCheminsAiCitesDansLeCodeExistent` (53 chemins .ai/ cassés par l archivage), corrigé.

- 47d20b5d (film ancien, version 31) : pont d'identité vide, 298 vies sans nom (décision du user du
  15/10 : films d'avant la 1.8 abandonnés, voir le registre).
- 7aa132d1 : pistes parasites sur des slots hors plage des bipèdes, non prouvé.
- HI_1_8_0 : octets 0x21/0x22 donnent l'équipe de la feuille sur 4 bots / 4 (échantillon trop petit).
- Digests de référence de `replay-equiv` périmés (20/20 diffèrent dès la référence).
- `replaybuild.botSuccessions` indexe les bots par bid seul.
