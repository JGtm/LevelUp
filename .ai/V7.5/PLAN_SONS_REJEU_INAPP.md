# PLAN — Sons du rejeu 2D : variation RANGED et distance, IN-APP

> Branche : `feat/sons-rejeu-inapp` (a creer depuis `feat/extraction-sons-armes`).
> Ouvert le 2026-08-16. Contrat d'execution : skill `plan-execution`.
> Execution CONFIEE A UN AGENT ; le pilote relit chaque gate.

## Decisions utilisateur (fermes — ne pas rouvrir)

1. Les sons extraits restent PURS : aucune variation cuite dans les `.wav`.
2. La variation RANGED (volume/hauteur par lecture) et la distance s'appliquent COTE APP.
3. Reglages sur la PAGE D'ADMIN uniquement — pas des reglages utilisateur.
4. SIMPLICITE de calibrage exigee : 3 boutons maximum, valeurs par defaut = valeurs du jeu.

## Ce qui existe deja (ne rien reecrire)

- `apps/go-api/cmd/weapon-sounds` : parseur de banks. Le paquet RANGED est deja LOCALISE et
  VALIDE (`proprietes.go`, `lirePaquetProps(d, suite, 2)`) — ses valeurs ne sont juste pas
  exportees. Fourchette = [val - min, val + max] par propriete (volume dB, pitch centiemes).
- Les courbes de fondu des `Blend` sont decodees (`conteneurs_autres.go`) — inutile pour ce
  plan (la distance in-app est un effet applicatif), mais disponible plus tard.
- Sons retenus par vote : `Desktop/Halo Infinite - Sons armes/` + votes JSON. La LIVRAISON
  des fichiers retenus attend la fin du re-vote (33 coups) — le plan prepare le chemin et
  le format, pas le contenu final.

## Reglages (page admin, section « Sons du rejeu »)

	variation   : curseur 0-100 % (defaut 100 = fourchettes du jeu telles quelles ; 0 = off)
	distance    : curseur 0-100 % (defaut 0 = son pur ; attenuation de gain + passe-bas)
	(pas d'autre bouton)

Stockage : meme mecanisme que les autres reglages d'app (`app_settings.json` / endpoint
admin existant — DECOUVRIR le pattern en place et s'y conformer, ne pas en inventer un).

## Etapes

### Etape 1 — Decouverte (rien coder avant)

Gate : compte rendu ecrit au journal de CE fichier.

- [ ] Ou vit le rejeu 2D cote web (composants, lecteur audio existant ou absent)
- [ ] Pattern exact des reglages admin existants (stockage, endpoint, page, i18n FR/EN)
- [ ] Ou ranger sons + manifeste pour l'app (convention assets existante,
      `static/weapons-assets/...` ou autre — suivre l'existant)

### Etape 2 — Export des fourchettes RANGED (Go, cmd/weapon-sounds)

Gate : `go build ./...` + `go vet` verts ; nouvelle sortie JSON documentee dans l'en-tete
de main.go. NE PAS lancer le module de 7,24 Go : laisser la commande ecrite au journal,
le pilote l'executera (contrainte memoire du chantier sons).

- [ ] Lire les valeurs du paquet RANGED (le lecteur existe, exporter min/max par propriete)
- [ ] Les faire remonter dans le rapport par arme : fourchette volume (dB) et hauteur
      (centiemes) par (mode, perspective) — agregation : fourchette de la couche dominante
- [ ] Champ `variation` dans le manifeste destine a l'app (schema ecrit, valeurs a venir)

### Etape 3 — Lecteur cote web (WebAudio)

Gate : `make check-types` + `make test-web` verts ; test unitaire du calcul de variation.

- [ ] Module de lecture des sons d'armes du rejeu 2D (ou extension du lecteur existant) :
      par lecture, tirage uniforme dans [min, max] x (variation/100) applique en gain
      (GainNode) et hauteur (playbackRate = 2^(cents/1200))
- [ ] Distance : chaine GainNode + BiquadFilter passe-bas, mappee sur le curseur
      (0 % = neutre absolu — AUCUN noeud dans le chemin du signal a 0)
- [ ] Fallback sans manifeste de variation : lecture pure (aucune erreur, aucun silence)

### Etape 4 — Reglages admin

Gate : page admin affiche la section, valeurs persistees et relues ; typecheck + lint verts.

- [ ] Deux curseurs, strings FR/EN via i18n.ts (parite typee), tokens de couleur semantiques
- [ ] Endpoint conforme au pattern decouvert a l'etape 1

### Etape 5 — Cloture

- [ ] Journal de ce plan + thought_log + delivery-checklist
- [ ] Commit(s) prefixes `feat(sons-rejeu):`, PAS de push sur main

## Hors perimetre (ne pas toucher)

- Regeneration des sons, votes, artefact de tri (chantier sons-armes)
- RTPC de couche, delais d'action (statues au plan sons-armes)
- Tout reglage expose aux utilisateurs finaux

## Livraison des sons (regle utilisateur du 2026-08-16)

**La livraison est UNIQUE, FINALE, et c'est un REMPLACEMENT EN MIROIR** : le dossier
cible est vide puis reecrit avec exactement les fichiers votes + le manifeste — « pour
pas avoir de redondance ou d'elements obsoletes ». Un fichier absent du manifeste n'a pas
le droit d'exister dans le dossier cible. Decision utilisateur : « les sons sont
definitifs a ce stade [...] il n'y aura pas plusieurs livraisons de prevues. Il faut
considerer notre travail ici comme final. » Toute reprise future (arme nouvelle, erreur)
REJOUE la recette documentee — `RECETTE_SONS_ARMES.md` — et se conclut par une nouvelle
livraison miroir ; on n'amende jamais le dossier livre a la main.

Contenu de la premiere livraison (fige, cf. handoff sons-armes section 7) : les 46 votes
de `votes-sons-armes(4).json`, avec les roles multiples confirmes — Ravageur (bb31841b =
tir 3 coups pour le rejeu, reconstitue be684013 = coup unique, c15c9e77 = montee en
charge) et Rayon de sentinelle (503433748 = rejeu, reconstitue = tir continu). Le
manifeste app encode ces roles : `rejeu` / `conserve`.

## Journal

### 2026-08-16 — Ouverture

Piste consignee au passage (chantier sons-armes, pas ici) : l'utilisateur se souvient d'un
« pan... clic » sur la Carabine Vestige — symptome possible du delai d'action non prouve.
A verifier a l'oreille sur les rendus regeneres avant d'instruire.
