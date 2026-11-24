# Règle 3.2 — croix des morts sans type : « S1 ou S2 »

Décision du user (2026-11-15) : livrer dans la 8.2.0, règle « S1 ou S2 ». Source des chiffres :
`mesure_morts_sans_type.md` (99 films, 1 307 joueurs confrontés à la feuille). Deux exécutants : (a) dépôt du
décodeur, (b) web de LevelUp. Chacun RE-VÉRIFIE sur pièces les fichiers et fonctions cités (relus sur la tête
`origin/main` 558af5f du décodeur et sur le worktree `LevelUp-wt-bl-killfx`) avant de coder.

## Règle

Une vie close sans frag est une **mort prouvée** (croix posée, sans type) si :

- **S1** : le document dit que son joueur a lâché la commande du corps (`endReason == "released"`) et que
  `releaseFrame - fin` est dans **[-2, +6] frames** ; ou
- **S2** : un compte à rebours de réapparition du même joueur (`respawnCountdowns[]`, `filmIndex` -> roster -> xuid)
  commence à `from - fin` dans **[-2, +6] frames**.

`fin` = la fin de piste que le web emploie déjà (`trackWindow(track).end` : `endFrame`, sinon le dernier point).
Tolérances mesurées sur les morts certaines (vies revendiquées par un frag), frame de 33 ms :
- S1 : 14 671 lâchers, écart de -2 à +6 frames, aucun au-delà ;
- S2 : 14 783 comptes à moins de 10 frames de la fin, tous de -2 à +6 ; 34 au-delà de 10 frames appartiennent à une
  autre mort du joueur ; aucun entre 7 et 10.
Les deux bornes sont donc les mêmes, en frames ; le web les applique en millisecondes (-66 ms, +200 ms) pour ne pas
dépendre de la cadence du document.

Résultat attendu, mesuré (accord exact par joueur avec la feuille, morts que le tableau ne compte pas retirées) :
plusieurs manches 99,8 % (aujourd'hui 89,1 %), une manche 100 % (98,5 %), BTB 99,3 % (88,1 %). Croix nouvelles sur
l'échantillon : 443, dont 179 sur des morts que le tableau ne compte pas (morts après la décision d'une manche ou
infligées par la fin du match : de vraies morts, croix attendue). Fins de manche (corps supprimé encore commandé,
sans compte) qui reçoivent une croix : 0 sur 653.

---

## (a) Décodeur

### a.1 Lire la suppression d'un corps (grammaire)

Aujourd'hui « supprimé » ne se lit pas : la marche des trames applique les records DEL au monde
(`internal/grammar/frame_infer.go`, `case recDel` -> `br.delier(w, slot)`, `debut_non_prouve.go`) sans rien publier
pour les bipèdes. À ajouter, dans le canal qui lit déjà `unit-control` (`internal/grammar/unit_control_scan.go`,
`controleDesUnites`, reçu par `movementStateScanner` dans `movement_states.go`) :

- garder l'ensemble des slots que `solder` a reconnus bipèdes (`slotEstBipede`), avec la génération du record de
  création (`ID >> 30`, comme `equipement_porte.go` : `vieDObjetPorte.gen`) ;
- à chaque record DEL d'un tel slot et de la même génération (records de la trame : modèle
  `canalDeLEquipementPorte.Trame`, `c.m.recordsDeLaTrame()`, `case recDel`), publier une lecture de suppression.
  Un DEL d'une marche non prouvée (`marcheNonProuvee`) ne délie pas le monde : il ne se publie pas, il se compte.
- type neuf dans `types/grammar_controle.go` :

```go
// BipedDeletion est la suppression d un bipede : le record DEL de son slot dans la vue B.
type BipedDeletion struct {
	Slot        uint32 // identifiant bas du bipede, celui de UnitControlRead et des trajectoires
	TimestampUS uint64 // horodatage du paquet porteur, horloge de BipedPosition
	Chunk, PacketIndex int
}
// UnitControlStats gagne : Deletions (publiees), DeletionsUnproven (DEL d une marche non prouvee, non publies).
```

- `MarcheDesTrames` gagne `BipedDeletions []types.BipedDeletion` (rempli à côté de `m.UnitControls` dans
  `marcherPourLaCuisson`), puis `replay.FilmInputs` (`film_inputs.go`, à côté de `UnitControls`) et
  `LecturesDePresence` (`lectures_de_presence.go`, champ `Deletions`) ; façade : alias `decfilm.BipedDeletion`, et
  `ScanUnitControls` (`decfilm/presence.go`) rend aussi les suppressions (signature à étendre, ou une seconde
  fonction `ScanBipedDeletions` ; l'exécutant choisit et le dit).
- Faits : `encodeControlesDeBipede` / `decodeControlesDeBipede` (`replay/filmfacts_presence.go`) portent les
  suppressions ; `SchemaDesFaits` 26 -> 27 (`filmfacts_fichier.go`), entrée de chronique des faits.
- Révision : `grammar.Rev` monte (`internal/grammar/rev.go`), entrée dans `internal/grammar/rev_chronique.go` ;
  goldens de révision et empreintes refigés comme au commit 89c2559.
- Garde-rail existant à respecter : aucun nouveau balayage bit à bit (`internal/depottest/balayages_test.go`) — la
  lecture se branche sur les records que la marche décode déjà.

### a.2 Publier la fin de chaque piste (assemblage)

Fichier neuf `replay/tracks_fin_par_commande.go`, appelé à la fin de `poserLesPistes` (`replay/build_pistes.go`),
après `poserLesTraces` (les fins de piste sont alors définitives, `tracks_fin_de_vie.go` compris). Entrées :
`a.doc.Tracks`, `a.opt.Presences.Controls`, `a.opt.Presences.Deletions`, `a.origin`, `a.step` (frames par
`frameOf`, comme `document_horloges.go`).

Pour chaque piste, par slot et dans l'ordre des débuts (règle de la mesure, `bl1115_morts_research_test.go`) :
1. fenêtre de lecture : de `max(début - margeDeNaissance, fin de la piste précédente du slot + 1)` jusqu'au début de
   la piste suivante du même slot (exclu) ; `margeDeNaissance = 5 s` (borne de la sonde : la création du corps
   précède son premier point publié) ;
2. la première lecture commandée (`Controller >= 0`) fixe le joueur du corps ;
3. le premier des événements suivants clôt la vie :
   - lecture `Controller == -1` -> `endReason = "released"`, `releaseFrame` = sa frame ;
   - suppression du slot -> `endReason = "deleted"` ;
   - lecture d'un autre joueur aux commandes du slot -> `"deleted"` (corps supprimé puis recréé, suppression non
     lue), compté à part ;
4. aucune lecture commandée, ou aucun des trois événements -> champs absents.

Le décodeur ne juge pas : il ne compare pas `releaseFrame` à la fin de piste (la tolérance est côté web).

Champs, dans `Track` (`replay/document_tracks.go`), optionnels :

```go
// EndReason dit comment le corps de la vie a fini, lu dans le film : "released", son joueur lache la commande
// (`unit-control` sans joueur : une mort) ; "deleted", le corps est supprime encore commande (fin de manche,
// remise en place, depart). Vide : aucune lecture ne le dit.
EndReason string `json:"endReason,omitempty"`
// ReleaseFrame est la frame du lacher, sur l axe du document ; present pour "released" seulement. Pointeur : une
// frame nulle ou negative est une lecture.
ReleaseFrame *int `json:"releaseFrame,omitempty"`
```

Couverture, dans `TrackCoverage` (`replay/tracks_publication.go`) :

```go
LifeEnds *LifeEndCoverage `json:"lifeEnds,omitempty"`
type LifeEndCoverage struct {
	Released int `json:"released"`  // pistes closes par un lacher
	Deleted  int `json:"deleted"`   // closes par une suppression lue
	Retaken  int `json:"retaken"`   // closes par la prise du slot par un autre joueur (suppression non lue)
	Unread   int `json:"unread"`    // aucune lecture commandee dans la fenetre
	Open     int `json:"open"`      // un joueur aux commandes, aucun evenement de fin (survivant, fin du film)
	DeletionsUnproven int `json:"deletionsUnproven"` // DEL d une marche non prouvee, non publies
}
```

Calque et schéma :
- `layers.go` : la ligne `"tracks"` reste `grammar.Rev` (la fin se lit dans la grammaire) ; son commentaire cite la
  nouvelle source (`FilmInputs.UnitControls + BipedDeletions`).
- `SchemaVersion` (`replay/document.go`) : la version suivante de la tête au moment de l'écriture (la branche des
  manches de la phase C occupe déjà 122), avec son entrée dans `document_chronicle.go` (champs, sources, règle de
  lecture). La montée sert la régénération unique de la phase C (D-3(b)) ; aucune régénération pour ce lot seul.
- `FORMAT.md` et `GLOSSARY.md` (« fin de vie », `endReason`) mis à jour dans le même commit.

### a.3 `replay-equiv`

Attendu : seules l'étape des pistes et la couverture changent, et seulement par `tracks[].endReason`,
`tracks[].releaseFrame`, `coverage.tracks.lifeEnds`, plus la révision de grammaire dans `layers` et
`coverage.decoder`, et `schemaVersion`. Toutes les autres étapes : digests identiques. Preuve : sur les 20 films,
document neuf moins ces champs = document d'avant, octet pour octet. Puis refigement (`go run ./cmd/replay-equiv
-update`, côté LevelUp, par la session de la montée).

### a.4 Tests (aucun vrai xuid ; identifiants fictifs de `testdata/IDENTITES.tsv` si une fixture en porte)

- Grammaire : un DEL d'un slot bipède produit une suppression ; un DEL d'un autre archétype, rien ; un DEL d'une
  génération ancienne, rien ; un DEL de marche non prouvée, compté et non publié.
- Assemblage (fonction pure, entrées synthétiques) : lâcher à la fin -> `released` et sa frame ; suppression
  encore commandé -> `deleted` ; slot repris par un autre joueur -> `deleted` et `retaken` ; aucune lecture ->
  absent ; le lâcher de la vie précédente du slot n'est pas pris par la suivante ; une lecture après le début de la
  vie suivante du slot n'est pas prise ; lâcher à frame négative -> pointeur non nul.
- Golden : la fixture de présences (`golden_presences_test.go`, `presences_<short8>.bin.gz`) gagne les suppressions ;
  `TestGoldenPresencesPorteDesXuidsFictifs` reste vert. Témoin sur un film réel du parc (hors dépôt) : les comptes
  de `lifeEnds` de `d9781168` (Oddball à deux manches) et d'un BTB, avant et après, dans le compte rendu.

### a.5 Crédit à Den

- Code : en-tête de `tracks_fin_par_commande.go` et commentaire du canal de suppression : « Méthode reprise de
  libfilmshell (Dend, OpenSpartan, licence MIT, `src/owners.rs` et `src/motion.rs`) : une vie finit par une mort quand
  son joueur lâche la commande du corps, par la suppression du corps quand une manche la termine. » Aucun plan, lot,
  date de mesure ni chemin `C:\t` dans les commentaires.
- `NOTICE.md` : une ligne dans la section libfilmshell (méthode reprise, aucun code copié).
- Note de version du tag : « La fin de chaque vie se lit dans le film (`tracks[].endReason`, `releaseFrame`) :
  lâcher de commande ou suppression du corps, méthode de libfilmshell (Dend, MIT). » et la liste des champs ajoutés
  (règle des notes du décodeur : dire tout changement de sortie).

---

## (b) Web (LevelUp)

Dépendance : le tag du décodeur qui porte a.2 est monté (étape 3.3, par la session de la montée), puis
`make generate-types` : `ReplayTrack` (`lib/api/types.ts`, `components['schemas']['Track']`) gagne `endReason?` et
`releaseFrame?`, le document gagne `respawnCountdowns?` s'il ne l'a pas encore. `normalizeReplayDocument`
(`lib/replay/replayNormalize.ts`) : `respawnCountdowns ?? []`, rien d'autre.

### b.1 La preuve, fonction pure

Fichier neuf `features/match-replay/model/lifeEndProof.ts` :

```ts
/** Lâcher de commande : écart à la fin de piste, en ms. Mesuré sur 14 671 morts fraggées du parc, frame de 33 ms :
 * de -2 à +6 frames, aucun au-delà. */
export const UNIT_RELEASE_TOL_MS = { before: 66, after: 200 } as const
/** Début du compte à rebours de réapparition : écart à la fin de piste, en ms. Mesuré sur 14 783 morts fraggées
 * du parc : de -2 à +6 frames ; au-delà de 10 frames, le compte est celui d'une autre mort. */
export const RESPAWN_COUNTDOWN_TOL_MS = { before: 66, after: 200 } as const

export type LifeEndProof = 'release' | 'countdown' | null
export function lifeEndProof(track, doc, countdownsByXuid): LifeEndProof
```

- S1 d'abord : `endReason === 'released'` et `releaseFrame` présent et écart dans la tolérance -> `'release'`.
- Sinon S2 : un compte du xuid de la piste dont `from` tombe dans la tolérance -> `'countdown'`.
- Sinon `null`.
- `countdownsByXuid` se construit une fois par document : `filmIndex` -> xuid par `doc.roster` ; un `filmIndex` que
  plusieurs entrées portent, une entrée sans xuid ou de bot : ses comptes ne servent pas (compté).
- Les deux constantes sont distinctes (deux canaux, deux sources) même si leurs valeurs sont égales.
- Conversion frames -> ms par les helpers existants (`frameToMs` / `msToFrames`), jamais par 33 en dur.

### b.2 Où elle s'applique

- `model/killFeedLogic.ts` : `lifeEndsOf` porte la piste (ou sa preuve) dans `LifeEnd` ; `neutralDeath` reçoit la
  preuve ; `ReplayDeath` gagne `proven: boolean` (preuve non nulle). Les deux chemins (`alignFeedByOrigin`,
  `alignFeedToTracks`) passent par `lifeEndsOf` : une seule pose. `attachDeathKinds` inchangé (le type reste celui du
  document).
- `model/livesMarks.ts` `deathFacts` : `e.death.kind === '' && !e.death.proven` -> compté dans `unplacedDeaths` comme
  aujourd'hui ; sinon croix (une mort typée garde sa croix, preuve ou non). Commentaires de `deathFacts` et de
  `killFeedLogic` réécrits : le contrat (« une fin de vie sans frag est une mort quand le film la prouve : lâcher de
  commande ou compte à rebours ; une fin de manche supprime le corps sans l'un ni l'autre »), pas l'histoire.
- La ligne du fil reste neutre (sans pictogramme) : seule la croix change.

### b.3 Cas

- Fin de match : inchangée. Une vie close dans les 3 dernières secondes (`SURVIVOR_GRACE_MS`) n'est pas une mort
  neutre ; une vie close avant, prouvée, a sa croix (la fin du match inflige de vraies morts).
- Joueur sans `filmIndex` dans le roster : S2 impossible, S1 seul ; sans lâcher lu, pas de croix, compté.
- Bot : pistes sans xuid, hors de `lifeEndsOf` comme aujourd'hui.
- Document d'ancien schéma (sans `endReason` ni `respawnCountdowns`) : aucune preuve, comportement d'aujourd'hui.
  Document intermédiaire (schéma 120-121 : comptes sans `endReason`) : S2 seul, et c'est voulu.

### b.4 Tests vitest

- `lifeEndProof.test.ts` : lâcher à -2, 0, +6 frames -> `release` ; à +7 -> null ; `deleted` sans compte -> null ;
  `deleted` avec compte à +3 -> `countdown` ; compte à -2 et +6 -> `countdown`, à -3 et +7 -> null ; compte d'un autre
  joueur -> null ; `filmIndex` porté par deux entrées -> null ; joueur absent du roster, lâcher dans la tolérance ->
  `release` ; ancien document -> null ; intervalle de frames différent de 33 ms : la tolérance suit les ms.
- `killFeedLogic.test.ts` : une mort neutre porte `proven` ; une fin de manche (corps supprimé, aucun compte) donne
  `proven: false`.
- `livesMarks.test.ts` : mort sans type prouvée -> croix et `unplacedDeaths` inchangé pour elle ; non prouvée ->
  comptée ; typée -> croix comme aujourd'hui ; bot -> aucune croix ; ancien document -> sorties d'aujourd'hui.
- Porte : `npx vitest run src/features/match-replay`, `make check-types`.

---

## (c) Ce qui reste sans croix

- Fins de manche et remises en place : corps supprimé encore commandé, sans compte (653 sur l'échantillon). Voulu.
- Lâchers de fin de match sans compte à rebours (21) : la fin du match lâche les vies sans tuer. Voulu.
- Vraies morts dont le lâcher et le compte suivent la fin de piste de plus de 6 frames (56, toutes non comptées par
  le tableau) : les positions du corps s'arrêtent avant la mort ; aucune des deux tolérances ne les prend.
- Vies dont ni le corps ni le compte ne sont lus (33 sur l'échantillon, dont 3 morts que le tableau compte, surtout en BTB) : rien ne les prouve.
- Morts des 3 dernières secondes (survivants pour le web), bots, joueurs sans xuid : hors population, inchangé.
- Documents non régénérés : jusqu'à la régénération unique de la phase C.
