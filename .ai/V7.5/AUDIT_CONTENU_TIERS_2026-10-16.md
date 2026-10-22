# Audit du contenu tiers suivi dans le dépôt — 2026-10-16

Contexte : passage MIT vers AGPL-3.0 (code) et CC BY 4.0 (docs). La licence ne couvre que ce
que l'auteur a écrit ; ce document recense ce qui, dans l'arbre suivi, appartient à Microsoft ou
à des joueurs. `NOTICE` §4 le déclare hors licence. Rien n'a été supprimé : retirer du contenu
d'un dépôt public est une décision de l'utilisateur, et seule une purge de l'historique
retire ce qui a déjà été publié.

## Inventaire (arbre au 2026-10-16)

| Catégorie | Emplacement | Volume | Nature |
|---|---|---|---|
| Sortie de décompilation du jeu (RETIRÉE le 2026-10-16) | copie hors dépôt : `Downloads/LevelUp_decompilation_Ghidra`, motif ignoré par `.gitignore` | 273 fichiers `.c` | Dérivé direct du binaire Microsoft : retiré de l'arbre courant, reste dans l'historique |
| Captures mémoire du jeu et extraits de recherche (RETIRÉS le 2026-10-22) | `.ai/V7.5/dumps/` (78 fichiers suivis retirés avec `tools/ce/*.bin`, dont `ce_run2_cliffhanger.bin.gz` 8 Mo et `ce_capture_delta.csv` 19 Mo) ; copie hors dépôt : `Downloads/LevelUp_captures_memoire_jeu` (95 fichiers, SHA-256 vérifiés) ; motifs ignorés par `.gitignore` | ~76 Mo | Contenu mémoire du jeu en cours d'exécution, extraits du jeu et sorties d'outils. Restent, nommés dans `NOTICE` §4 comme fixtures de test : 3 `.mvar` (`mapvar/`), `callout_zones_ridgeline_clipped.json`, `carte_validee_v1.png` ; et les 12 scripts `tools/ce/filmdec_*.lua` (notre code) |
| Audio du jeu | `static/sounds/halo_infinite` | 182 wav, 83 Mo | Servi par l'application ; GARDÉ (décision du 2026-10-16 : le rejeu en a besoin) |
| Images du jeu | `static/{medals,maps,ranks,commendations,weapons-assets,vehicles-assets,abilities-assets,prestige-assets}`, `apps/web/public/titles/halo_5/spartan/{emblems,nameplates}*`, `data/titles/*/reference/map_backgrounds`, `data/cache/career-rank-image` | plus de 2 000 png/webp | Servies par l'application ; certaines viennent de l'API de contenu officielle |
| Référentiel prébuilt | `data/titles/halo_infinite/warehouse/metadata-prebuilt.zip` | 5 Mo | Données de l'API |
| Films de matchs réels (fixtures) | `**/testdata/**` (film, replay, killsource, objectifs), `apps/web/src/features/match-replay/test/fixtures/go`, `.ai/V7.5/replay2d/registre_film/lotBP` | ~100 fichiers `.bin(.gz)` | Contenu Microsoft + gamertags de joueurs réels |

## Risques, du plus au moins exposé

1. **Décompilation et captures mémoire** : aucune tolérance de type « contenu de fan » ne les
   couvre ; une demande de retrait par Microsoft est plausible. Les faits qu'elles ont permis de
   découvrir sont libres et déjà écrits dans les docs de recherche ; les fichiers eux-mêmes ne
   sont pas nécessaires à la compilation ni à l'exécution.
2. **Fixtures de films** : nécessaires aux tests du décodeur ; contiennent des gamertags de
   joueurs tiers dans un dépôt public (angle RGPD, en plus du droit d'auteur).
3. **Assets servis** (images, sons) : l'application en a besoin ; c'est l'usage « fan content »
   des règles de Microsoft pour le contenu de jeu (non commercial, mention de non-affiliation,
   déjà au pied de page). Le volume audio (83 Mo) est le plus lourd à justifier.

## Options (aucune appliquée)

- **A. Minimale** : garder tel quel, `NOTICE` §4 déclare la situation et offre le retrait sur
  demande. Ne règle ni le risque 1 ni le risque 2.
- **B. Recommandée** : retirer de l'arbre courant les `.c` Ghidra et les captures mémoire
  (`git rm`, aucune conséquence sur le build), pseudonymiser ou anonymiser les gamertags des
  fixtures quand c'est possible. Les docs de recherche en prose restent.
- **C. Complète** : B + purge de l'historique (`git filter-repo`), ce qui réécrit tous les
  hashs publiés, casse les branches et worktrees ouverts, et demande un push forcé. À ne faire
  qu'à froid, après sauvegarde, et jamais sans l'accord explicite de l'utilisateur.

## Statut au 2026-10-22 : gamertags tiers des fixtures de films anonymisés

Décision de l'utilisateur (item Notion du 2026-10-22) : anonymiser les gamertags des joueurs tiers
dans les fixtures de films si c'est faisable sans casser les tests. C'est fait, sur la branche
`chore/anonymisation-gamertags`.

- **Périmètre parcouru** : 351 fichiers suivis (266 sous `**/testdata/**`, 9 sous
  `apps/web/src/features/match-replay/test/fixtures/go`, 76 sous
  `.ai/V7.5/replay2d/registre_film/lotBP`). Le lot BP ne contient aucun octet de film ni aucun nom
  (journaux et TSV de recherche) ; les mini-bobines d'objectifs (`corpus_objectifs`) non plus.
- **Avant** : 67 fichiers portaient 5 573 occurrences de 151 gamertags tiers distincts (37 fichiers
  d'octets de film, 8 faits figés `inputs_*.bin.gz`, 8 documents web, 13 golden texte, 1 graine de
  fuzz). **Après** : 0. Restent les joueurs maison (JGtm, Chocoboflor, Madina97294 dans ces films) et
  les bots.
- **Méthode** : pseudonymes de MÊME longueur (espaces conservés à leur place), cherchés en UTF-16
  petit- et gros-boutiste aux huit décalages de bit et en UTF-8 aligné ; chunks zlib décompressés,
  réécrits, recompressés. Outil : `apps/go-api/cmd/anonymise-gamertags` (porte de test, le ratchet
  des octets de film interdisant `compress/zlib` au code de production sous `cmd/`). Table de
  correspondance (213 noms) HORS dépôt : `Downloads/LevelUp_correspondance_gamertags.txt`. Copie
  des 351 fichiers d'origine, vérifiée par empreintes : `Downloads/LevelUp_fixtures_films_originales`.
- **Preuve** : sur les 11 films versionnés, 13 balayages du décodeur (tirs, grenades, armes, inventaire,
  morts, index, projectiles, table des joueurs, temps forts, véhicules, objectifs, statborg,
  killsource) rendent une sortie identique à celle de l'original aux noms près (143/143) ; la
  régénération Go des 8 documents web depuis les faits anonymisés rend, à l'octet, le JSON réécrit
  par l'outil ; les deux empreintes qui bougent (morts de la mini-bobine, temps forts v41) sont celles
  de l'original noms substitués ; les golden killsource des 4 films du cache passent sur une copie
  anonymisée de ces films. Aucune grammaire, aucun profil, aucun artefact de démo touché.
- **Coût permanent** : toute régénération d'une fixture depuis le cache remet les noms réels (rejouer
  l'outil avant de commiter) ; les gates locaux adossés au cache (`TestGoldenFilms` killsource,
  `TestGoldenInputsFidelite`) se jouent sur une copie anonymisée des films.
- **Reste hors de ce lot** : les XUID (identifiants numériques, présents partout) ; les noms cités
  dans la prose `.ai/` (au moins 36 fichiers, 6 043 occurrences de noms de la table) ;
  `apps/go-api/tests/fixtures/golden_values/match_view_slayer.json` (réponse d'API, hors films) ;
  l'historique git, qui garde les fixtures d'avant (option C).

## Autres constats

- Auteurs : le dépôt contient aussi 47 commits d'un bot Copilot et 17 de dependabot (code généré
  ou trivial). Des commits d'une adresse non personnelle ont été repérés ; leur nettoyage
  (réécriture d'historique) est une décision distincte, hors de ce lot.
- Dépendances : 161 modules Go et 121 paquets npm de production examinés (licences de leurs
  fichiers). Aucun GPL/AGPL/SSPL. Un seul copyleft faible : `hashicorp/golang-lru/v2` et un
  paquet npm en MPL-2.0, compatibles AGPL-3.0.
- Tag de licence choisi : `AGPL-3.0-only` (pas `-or-later`) pour garder la main sur les versions
  futures de la licence.
