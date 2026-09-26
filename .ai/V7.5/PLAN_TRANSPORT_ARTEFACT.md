# Plan — Transport de l'artefact et choix du LIEU de construction

> Ecrit le 2026-08-14. Ferme le dernier maillon de la piste F : aujourd'hui l'ouvrier prend
> un travail, le decode et rend un COMPTE RENDU, mais l'ARTEFACT (~2 Mo) reste chez lui.
> Ajoute le reglage « ou se construit un rejeu » : LOCAL en dev, OUVRIER DISTANT en prod
> (decision utilisateur du 2026-08-14 : « pour le dev c'est en local mais pour la prod j'ai
> decide que ce sera sur un autre VPS »).
> Execution sous `plan-execution`, branche `feat/v75`, commits par etape, PAS de push.

## Le sens du transport — leve d'ambiguite (question utilisateur)

« C'est l'app qui va le recuperer, non ? » — OUI, et c'est compatible avec le schema du
master plan (§1), qui dit « renvoie l'artefact ». Les deux disent la meme chose vue des deux
bouts, et le point qui compte est celui-ci :

**L'OUVRIER N'A AUCUN PORT ENTRANT.** C'est lui qui initie TOUTES les connexions (il tire le
travail, il pousse le resultat) ; l'app expose les routes et RANGE le fichier. Donc :
l'app « recupere » au sens ou elle est le seul point d'entree et le seul lieu de stockage —
mais le paquet part de l'ouvrier. Aucune des deux machines n'a besoin de joindre l'autre a
l'improviste, et l'ouvrier reste installable derriere n'importe quelle box.

## Etape 1 — Le transport lui-meme

- [x] 1.1 `POST /api/v1/internal/build-queue/artifact` (meme jeton d'ouvrier, meme
      middleware que `claim`/`complete`/`heartbeat`) : corps = l'artefact. **Verifications
      NON NEGOCIABLES avant d'ecrire quoi que ce soit sur le disque** :
      - le job existe, est `running`, et est bien CLAIME PAR CET OUVRIER (sinon 409, comme
        `complete` le fait deja) ;
      - **taille bornee** (un plafond explicite ; ~2 Mo mesures, prevoir large mais fini) —
        un corps non borne est une porte ouverte sur le disque du VPS, dont le plan rappelle
        qu'il est « sous tension » ;
      - **le contenu est un artefact VALIDE** : il se deserialise en `replay.ReplayDocument`,
        son `matchId` est celui du job, son `schemaVersion` est celui qu'on attend. Un
        fichier qui echoue = 400, rien n'est ecrit.
- [x] 1.2 Ecriture ATOMIQUE a la place canonique (`PathResolver.ReplayArtifactPath`) :
      fichier temporaire puis renommage — jamais d'ecriture en place (un artefact a moitie
      ecrit serait servi tel quel par le service de lecture).
- [x] 1.3 Le `complete` actuel devient le POINT FINAL : un job n'est `succeeded` que si son
      artefact est arrive ET valide. Ordre a trancher et a ECRIRE : artefact d'abord, puis
      `complete` (recommande — le compte rendu ne ment jamais sur la presence du fichier).
- [x] 1.4 `cmd/replay-worker` : envoie l'artefact puis appelle `complete` ; en cas d'echec
      d'envoi, ne marque RIEN et laisse le bail expirer (le job repart en file, mecanique
      deja livree). Supprime ses morceaux de film locaux apres coup (le master plan §1 le
      demande : l'ouvrier ne conserve rien).

Gate 1 : test de bout en bout etendu (mise en file -> claim -> ARTEFACT -> complete ->
l'artefact est lisible par le service de lecture, a l'octet identique a celui envoye) ;
tests de refus : job d'un autre ouvrier, artefact trop gros, JSON invalide, mauvais matchId.

## Etape 2 — Le reglage « OU se construit un rejeu »

- [x] 2.1 UN reglage, trois valeurs, dans `AppSettings` (patron du scheduler : relu a chaque
      cycle, editable depuis l'admin sans redemarrage) :
      - `local` — le serveur construit lui-meme, en processus (ce qui existe deja : etape
        post-sync + job admin). DEFAUT EN DEV.
      - `worker` — le serveur MET EN FILE et ne decode jamais. DEFAUT EN PROD (decision
        utilisateur ; le master plan l'exige : « le VPS web ne decode JAMAIS »).
      - `off` — ni l'un ni l'autre (aucune construction ; le rejeu se contente de ce qui
        existe deja).
- [x] 2.2 UN SEUL POINT DE DECISION dans le code (pas un `if` recopie a trois endroits) :
      une fonction qui, pour un match donne, dit « je construis / je mets en file / je ne
      fais rien ». Les trois appelants actuels (etape post-sync, job admin, CLI) passent par
      elle. Le CLI de backfill garde son comportement direct — c'est un outil d'operateur,
      pas un chemin de service : le dire en commentaire.
- [x] 2.3 Le garde EXISTANT reste : en production, l'etape post-sync locale ne s'installe
      pas (`replay_local_gate` / wiring non-production). Le reglage `local` en prod doit
      donc etre REFUSE explicitement, avec un message clair a l'admin, plutot que silencieux.
- [x] 2.4 UI admin : le reglage visible et modifiable a cote de la fenetre de retention,
      i18n FR+EN, avec l'etat de la file et des ouvriers deja livre juste a cote.

Gate 2 : tests des trois valeurs (le bon chemin est pris, et un seul) ; test du refus
`local` en production ; typecheck purge + eslint + vitest ; `vitest run src/lib/query/` si
une query key est ajoutee (garde-fou de classement, il a deja rougi).

## Etape 3 — Boucler la boucle (ce qui rend la chaine utile)

- [x] 3.1 Brancher la MISE EN FILE au fil de l'eau : en mode `worker`, l'etape post-sync
      n'appelle plus le constructeur mais `EnqueueReplayBuild` (deja ecrit, deja porteur des
      URL pre-signees). C'etait le report explicite du lot precedent, il devient faisable
      maintenant que le transport existe.
- [x] 3.2 Fenetre de retention respectee des la mise en file (ne pas enfiler ce que la purge
      effacera).
- [x] 3.3 Journal et compteurs : artefacts recus, refuses (par motif), octets transportes —
      visibles dans le monitoring existant, sans nouvelle page.

Gate 3 : une preuve locale complete — un cycle qui enfile, un ouvrier lance a la main qui
construit, et l'artefact qui apparait dans le rejeu de l'app sans intervention.

## Hors perimetre (a dire maintenant)

- Deploiement du 2e VPS, provisioning, service systemd, secrets d'infra : ce plan livre le
  BINAIRE et le PROTOCOLE, pas l'installation. Elle viendra avec l'activation prod, et elle
  a sa propre place au runbook.
- Activation du rejeu en production (le garde local n'est PAS touche par ce plan).
- Plusieurs ouvriers en parallele : la mecanique le permet (bail + claim atomique) mais rien
  n'est mesure a plusieurs — ne pas le promettre, le noter au registre.
- Chiffrement/compression du transport : l'artefact est du JSON de 2 Mo sur HTTPS ; si le
  volume devient un sujet, la compression HTTP standard suffit. Ne pas inventer de format.

## Ce qui peut faire echouer ce plan

1. Le VPS web n'a pas la place pour recevoir : le plan rappelle un disque « sous tension »
   (plafond de cache 5 Go, zero swap). La fenetre de retention et la purge existent — les
   VERIFIER avant d'ouvrir le robinet, pas apres.
2. Un artefact valide au sens JSON mais construit avec un decodeur plus ancien : c'est le
   role de `schemaVersion` (deja la cle de reprise du backfill). Le refuser, pas le ranger.
3. Deux ouvriers qui rendent le meme match : le claim atomique l'empeche deja ; le test doit
   le montrer, pas le supposer.

## Journal d'execution

### Etape 1 — close le 2026-08-14 (commit « transport »)

Gate 1 PASSE. Preuve de bout en bout sur les VRAIES routes montees
(`internal/api/wire/build_queue_transport_e2e_cgo_test.go`) : file -> prise -> refus du
succes anticipe -> depot -> **artefact a l'octet identique** sur disque -> lu par
`service.NewReplayService` -> compte rendu accepte. Refus testes (chacun verifiant qu'aucun
fichier n'est ecrit) : job d'un autre ouvrier, job inconnu, identite absente, artefact trop
gros (413), JSON invalide, mauvais matchId, schema perime, artefact sans trajectoire.
Contrats HTTP : `internal/api/handlers/build_worker_artifact_test.go`.

DECISIONS PRISES EN COURS D'ETAPE, hors lettre du plan :

1. **Le plafond de corps devait etre pose explicitement**, pas seulement « prevu large » :
   Huma applique un defaut de 1 Mio et aurait refuse TOUS les artefacts (~2 Mo). D'ou
   `humacore.MaxBody` + `domain.MaxBuildArtifactBytes` (16 Mio) — et un second controle dans
   le handler, assume redondant : un plafond ne doit pas dependre d'un seul point de cablage.
2. **`replaybuild.writeArtifact` bascule sur la meme ecriture atomique** que le depot. Le
   plan ne demandait l'atomicite que pour la route, mais le constructeur local ecrit LE MEME
   fichier a la MEME place : deux facons d'ecrire auraient fini par diverger.
3. **L'ouvrier ne supprime ses morceaux de film que si `--work` designe un dossier a lui.**
   Son defaut est le cache film du depot — archive IRREMPLACABLE (les films expirent cote
   serveur, 29,3 % du corpus deja perdu). Un ouvrier ne detruit jamais l'archive de la
   machine qui l'heberge. Item 1.4 tenu, mais garde.

### Etapes 2 et 3 — closes le 2026-08-14 (commit « placement »)

Gates 2 et 3 PASSES. Point de decision unique : `replaybuild.DecidePlacement` (11 combinaisons
sous test). Trois appelants de SERVICE le consultent — fil de l'eau post-sync, action admin
`/replay-build/run`, PATCH /settings ; le CLI de backfill garde son chemin direct, dit en
commentaire au fichier. Refus de `local` en production : 400 avec motif au PATCH, et 409 nomme
a l'action admin. UI : le reglage vit a cote de la fenetre de retention (onglet Analyse),
FR + EN. Fil de l'eau `worker` : la fenetre de retention et l'idempotence s'appliquent AVANT la
file. Compteurs expvar : artefacts recus, octets transportes, refuses par motif
(`not_claimed` / `invalid` / `write`), succes annonce sans artefact, jobs enfiles au fil de l'eau.

DECISIONS PRISES EN COURS D'ETAPE, hors lettre du plan :

1. **`worker` sans jeton d'ouvrier configure degrade en `off`.** Enfiler quand personne ne
   viendra vider la file resoudrait un manifeste Halo par match a chaque cycle, pour rien, et
   ferait grossir la base de monitoring. Consequence VOULUE : la production, qui n'a pas encore
   de jeton, reste exactement dans son etat actuel — ce lot n'active rien en prod, conformement
   au perimetre. Le jour ou le 2e VPS existe, poser le jeton suffit.
2. **Un chemin de sync sans file cablee degrade en `off`, jamais en construction locale.** Le
   repli « silencieusement local » ferait decoder le VPS web — exactement ce que la regle
   interdit. Mieux vaut ne rien construire, en le journalisant.
3. **Les trois sites de wiring perdent leur `if !IsProduction()`** au profit d'une fabrique
   unique `replayartifacts.NewHook(cfg, settings, enqueue)` : le hook s'installe toujours, c'est
   LUI qui decide. C'etait la seule facon de tenir « un seul point de decision » sans laisser
   trois copies de la regle dans le cablage.

### Gate final — passe le 2026-08-14 (commit « preuve ouvrier »)

`go test -tags=integration -p 1 -run TestOuvrierReel ./internal/api/wire/` — PASS en 93,8 s.
Le BINAIRE `cmd/replay-worker` est compile puis lance en `--once` contre les vraies routes :
job enfile, pris par HTTP, 28 morceaux tires d'un faux CDN (morceaux du cache film servis en
zlib, comme Azure), film decode (module `ridgeline`), artefact de **2 195 683 octets /
99 trajectoires / 4 985 frames** pousse, range cote serveur **a l'octet identique**, lu par le
service de rejeu, job `succeeded`, morceaux de film supprimes par l'ouvrier.

Isolation : l'ouvrier travaille sur un depot A LUI (copie des references, 1,5 Mo) et un dossier
temporaire ; le depot de l'utilisateur est seulement LU (son cache film est une archive
irremplacable, ses artefacts ne sont pas touches). Le test SAUTE la ou le film temoin n'est pas
en cache (CI, poste neuf).

Ressources mesurees : pic memoire de l'ouvrier **121 Mo**, un seul decodage a la fois.
