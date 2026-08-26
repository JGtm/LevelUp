# PLAN — LOT BUILDALL : le post-sync ne decode plus dans le processus du serveur

> Ecrit le 2026-08-26 (lot D, suite du lot RUNNER). Cible unique :
> `internal/sync/replayartifacts.buildAll`. **PLAN SEUL — aucun code. STOP validation.**

## Le defaut, et pourquoi il est au registre

`buildAll` enchaine jusqu'a `maxPerCycle = 5` films a travers `replaybuild.BuildMatch`, **dans le
processus du serveur**, **sans sentinelle memoire**. C'est la forme EXACTE du sinistre du
2026-08-20 : quatre petits films cuits, effondrement sur le CINQUIEME. Ce qui limite le risque
aujourd'hui — l'etape n'existe qu'en environnement NON-PRODUCTION (« le VPS web ne decode
JAMAIS ») et elle est best-effort — borne le NOMBRE de films, jamais le PIC de l'un d'eux.

## Piste A — DELEGUER a l'enfant `backfill-replay` existant

Un film = un processus deja blinde, deja teste, deja en production. C'est la piste a instruire
en premier. Voici ce que la lecture sur pieces en dit.

**CE QUE L'ENFANT EXIGE** (`cmd/levelup/cmd_backfill_replay_child.go`) : la racine du depot, le
slug, les noms de carte, la racine du cache, un plafond memoire. Il ouvre la base **en LECTURE**
pour prendre SES faits de match et **relache le handle AVANT de decoder** — precisement pour ne
pas etre le lecteur qui grossit avec le corpus. Il rend un CODE de sortie, jamais une erreur.

**PIEGE N°1 — CONFIRME, ET LE DEPOT LE DOCUMENTE DEJA.** Si l'enfant ecrit l'artefact, c'est
`writeArtifactBytes` **de son processus** qui s'execute. Or le puits de notification
(`SetArtifactStoredSink`) n'est cable **qu'au boot du serveur**, et `artifact_events.go` en tire
la consequence noir sur blanc : « le CLI `levelup backfill-replay` ne le cable pas, donc un
backfill de masse ne produit AUCUN message ». **Deleguer tel quel supprimerait donc la
notification Discord des rejeux post-sync**, sans rien casser de visible — le pire des defauts.

*Remede a instruire* : le PARENT (le serveur) publie l'evenement **apres** le succes de son
enfant. Il lui manque `Bytes` / `Tracks` / `SchemaVersion` : l'enfant les lui rend par une
**ligne de protocole**, exactement comme il rend deja son pic memoire (`EmitPeak` / `parsePeak`,
`internal/filmproc/runner.go`). Aucune mecanique neuve — le patron existe et est teste.
*Garde* : `archlint/no_second_artifact_sink_test.go` interdit un second cablage du puits, ce qui
CONFIRME que la publication doit rester cote serveur et nulle part ailleurs.

**OBSTACLE N°2 — DECISIF, ET IL PEUT FAIRE TOMBER LA PISTE.** `filmproc.NewRunner` re-execute
`os.Executable()` : le patron suppose que le parent et l'enfant sont **le meme binaire**. Ici ils
ne le sont pas — le parent est `cmd/server`, l'enfant vit dans `cmd/levelup`. Deleguer exige donc
(a) que `NewRunner` accepte un executable EXPLICITE, et (b) qu'un binaire `levelup` soit
**garanti present sur l'hote**. A trancher au plan : d'ou vient ce chemin (configuration ? a cote
du binaire du serveur ?) et que fait-on quand il manque — degradation propre vers la piste B, ou
refus de l'etape ? **Lancer `go run` depuis le serveur est exclu** : compiler dans le chemin d'un
service est une regression en soi.

**CE QUI N'EST PAS UN PROBLEME, verifie** : le garde anti-regression de `writeArtifactBytes`
(refus d'ecraser un artefact plus recent) part avec l'ecriture, donc reste au bon endroit ; et la
DECISION de reconstruire (`etatArtefact`, artefact appauvri ou perime) reste cote serveur, avant
la delegation — c'est bien la qu'elle doit vivre.

## Piste B — REPLI : sentinelle SOUPLE in-process

Si la delegation coince (obstacle n°2 non levable), on borne sans changer de processus.

**JAMAIS TUER LE PROCESSUS SERVEUR** — il tient des handles d'ecriture DuckDB, et une mort
brutale au milieu d'une transaction est exactement ce que la doctrine anti-corruption interdit
(ADR 0013/0019/0030). La sentinelle de `filmproc` est utilisable **telle quelle** : son callback
ne decide rien, c'est l'appelant qui choisit sa doctrine. Ici le callback **leve un drapeau**, lu
**entre deux films** : les films restants du cycle sont SAUTES (re-enfiles au cycle suivant),
journalises, et le serveur continue.

**LA LIMITE EST A ECRIRE, PAS A MASQUER** : un drapeau lu entre deux films protege contre
l'ACCUMULATION — le vrai mecanisme des trois sinistres — mais **PAS contre un film-bombe isole**,
puisqu'on ne peut pas interrompre un decodage en vol sans tuer le processus. La piste B est donc
strictement plus faible que la piste A, et le plan doit le dire au lieu de la presenter comme
equivalente.

## Ce que devient `maxPerCycle = 5`

- **Piste A retenue** : le cap perd sa fonction de garde-fou memoire (chaque film est isole) et
  ne garde que celle de borner la DUREE d'un cycle. Il reste, inchange, et son commentaire est
  reecrit pour dire ce qu'il borne vraiment — sinon la prochaine lecture lui pretera une
  protection qu'il n'a plus.
- **Piste B retenue** : le cap devient la seule borne du NOMBRE, la sentinelle souple bornant
  l'accumulation. Il reste a 5 ; le baisser sans mesure serait un reglage au doigt mouille.

Dans les deux cas : **le cap ne monte pas dans ce lot**. Le relever demanderait une mesure du
cout d'un cycle, qui n'est pas le sujet.

## Ordre, gates, sortie

1. Trancher l'obstacle n°2 **sur pieces** (existence et provenance du binaire `levelup`) — c'est
   lui qui decide A ou B. Verdict ecrit avant toute ligne de code.
2. Implementer la piste retenue. La publication de l'evenement reste **cote serveur** dans les
   deux cas.
3. Tests : la notification survit a la delegation (piste A) ou au saut de film (piste B) ; un
   film saute est COMPTE et journalise, jamais silencieux ; le cycle suivant le reprend.
4. Gates NUS : `go vet`, `go test ./internal/sync/... ./internal/replaybuild/... ./internal/archlint/...`,
   et le garde-rail `no_unbounded_film_loop_test.go` dont l'entree de `buildAll` doit passer de
   **DETTE** a un regime sain — c'est le critere de fin du lot.

**RISQUE PRINCIPAL, ET IL EST DIT D'AVANCE** : ce lot touche le pipeline post-sync, chemin de
production locale. La regle de non-regression est que le fil de l'eau continue de produire ses
artefacts ET ses notifications ; un lot qui bornerait la memoire en perdant la notification
aurait echange un defaut invisible contre un autre.
