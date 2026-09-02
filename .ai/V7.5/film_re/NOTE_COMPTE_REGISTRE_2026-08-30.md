# Le compte du registre est tranche : 50 blocs, pas 118

Date : 2026-08-30. Lot 3 du plan « percer la trame » (`../PLAN_PERCER_TRAME_FILM_2026-08-30.md`).
Mesure sur les 1 367 `chunk_00` du cache ; correctif de production dans `parseRegistry`
(`apps/go-api/internal/analysis/filmdec/registry.go`) ; instrument :
`lot3_registre_compte_research_test.go` (garde `LOT3_CORPUS`).

## Le verdict, chiffre

1. **Le « 118 blocs d'archetype » du dossier etait `len(fichier)/taille_bloc`** — la taille
   inflatee de `chunk_00` (1 973 120 o) divisee par celle d'un bloc (16 640 o). Le lot D avait
   raison. Le garde-rail G2, tenu pour l'arbitre, ne verifiait QUE les slots nommes : execute
   sur les trois temoins du lot D, il est tombe ROUGE sur `00162144` (slot fantome « B » au
   bloc 71, 1 068 slots, empreinte inconnue) — il n'avait jamais vu ce film.
2. **La fin structurelle du registre** — premier bloc violant « suite de slots nommes, puis un
   slot de terminaison dont seul `flags` peut etre non nul, puis des zeros » — **tombe sur le
   bloc portant la section d'identification pour 1 362 films sur 1 362** (critere C1, 100 %).
3. **Builds de reference (`HI_1_12_0`/`HI_1_13_0`) : 50 blocs, 49 porteurs, 1 067 slots, sur
   1 253 films sur 1 253** (critere C3, 100 %). Builds anterieurs (`HI_1_4_1` a `HI_1_11_0`) :
   **49 blocs**, slots 1 029 a 1 034 selon le build (le « 116 blocs » de `06dfe6d9` etait le
   meme artefact ; ses 1 031 slots et son empreinte `0x5827362c37d2adb3` sont confirmes).
4. **L'ancien parse ramassait des slots fantomes dans le corps du fichier sur 397 films
   (29 %), 541 slots en tout** — 100 % au-dela du bloc d'identification (critere C2'). Ma
   prediction d'ouverture (« moins de 1 % des films ») est REFUTEE et publiee telle quelle :
   l'ancrage sur 3 temoins sous-estimait d'un facteur 29. Consequence passee : sur ces 397
   films, l'alerte « empreinte du registre INCONNUE » se declenchait a tort.
5. **Decouverte au passage** : le slot de terminaison d'un bloc porte un u32 non nul en
   `flags` (0x01/0x02) sur ~40 des 50 blocs — coherent avec le decalage R7-e (« le jeu lit le
   niveau un cran plus loin ») : c'est le niveau du dernier composant sous la lecture decalee.

## Le correctif

`parseRegistry` s'arrete a la fin structurelle (helper `registryBlockTail`). Effets mesures :

- `00162144` retrouve l'empreinte connue (1 067 slots, plus d'alerte) ; G2 passe sur les
  trois temoins du lot D : `50 blocs, 49 porteurs, 1067 lignes`.
- **Goldens killsource** (garde-fou du plan) : les 98 kills publies, l'ACCORD 85/DESACCORD 0,
  les 30 ancres Theater, la ligne discriminante et le controle negatif sont INCHANGES.
  Quatre compteurs de diagnostic bougent de ±1 (candidats consultes 118→119 sur `fccc61cd`,
  un candidat de plus correctement rejete ; scores de calibration ±1) : les boucles d'essai
  n'iterent plus sur 68 archetypes vides. Goldens regeneres dans le meme commit.
- `TestRegistryFingerprintDomain` mis a la nouvelle grammaire : bruit dans le bourrage d'un
  bloc = fin du registre (plus un bruit ignore) ; flags de terminaison exempte.

## Ce que cela change — conclusions anterieures a revoir

- **L'inventaire des composants (325 noms, 1 067 couples) est CONFIRME corpus entier** : les
  conclusions du type « aucun composant ne porte X » TIENNENT. Rien a rouvrir cote lunette.
- Toute phrase « 118 archetypes » ou « 116 contre 118 » se lit desormais « 50 » / « 49 contre
  50 » (commentaires de code corriges dans le meme commit ; les notes historiques du dossier
  restent datees et ne sont pas reecrites).
- Tout diagnostic passe qui s'appuyait sur « empreinte inconnue ⇒ grammaire differente » pour
  un film PARTICULIER est suspect si ce film portait un slot fantome (397 films) ; la
  comparaison PAR BUILD reste valable (les empreintes des builds anciens sont re-mesurees et
  distinctes de la reference).
- `keyframe_world.go` : la borne semantique kfArchMax=50 (COUNT de l'exe) et
  `len(reg.Archetypes)` coincident desormais — fermeture arithmetique a deux sources.
