# Gestes de production, LevelUp 8.3.0

Liste des gestes à faire en production au déploiement de la 8.3.0. Chaque chantier de la 8.3.0 y ajoute sa section ;
une section ne touche pas aux autres. Rien de ce qui est écrit ici n'est exécuté tant que l'utilisateur n'a pas donné
son accord pour le déploiement. Modèles de forme : `.ai/V8.3/annexes_v8_1_2/gestes_prod_8_2_0.md` et `gestes_prod_8_2_1.md`.

Règles de la 8.2.x, toujours valables (reprises de `gestes_prod_8_2_0.md`) :

- Jamais de copie d'une base locale vers la production. Seuls les rejeux se copient. Les données dérivées se
  réécrivent sur le serveur par les passes `backfill-*`.
- Push sur `main` = déploiement automatique (`CLAUDE.md`) : l'utilisateur est prévenu avant, et le geste est le sien.
- `LU` = `docker compose run --rm --no-deps levelup levelup`, depuis `/opt/levelup`. Une passe qui écrit la base
  partagée exige le serveur arrêté (`docker compose stop levelup`), `--dry-run` compris pour celles qui jouent des
  migrations (ADR 0013).
- Aucune régénération en lot sans accord explicite de l'utilisateur.

## Décodeur de film v1.6.1 (schéma des rejeux 127)

Pièces : plan `.ai/V8.3/PLAN_DECODEUR_V160_2026-11-23.md` (item L.8), procédure et journal de la régénération locale du
2026-11-24 (`.ai/V8.3/annexes_decodeur_v160/l6_regeneration_procedure.md`, `l6_regeneration_reelle_resume.md`), notes de
version (`docs/RELEASE_NOTES.md`, ligne « Update, for the operator »), `docs/RUNBOOK_DEPLOY_CHECKLIST.md`, entrées
du backlog sur l'archive des films du site hébergé.

Ce que la montée change pour le serveur : tous les rejeux passent du schéma 125 au schéma 127 et pèsent environ 40 %
de plus (local, mesuré : 2 149 245 711 puis 3 003 833 741 octets, soit 2,15 Go puis 3,0 Go pour 1 455 rejeux).
Un rejeu à l'ancien schéma est jugé périmé par la 8.3.0 et refait depuis son film (`replaybuild`, `Verdict` : schéma
différent du courant et faits du film absents = « à redécoder »). Le site hébergé n'a plus ses films : sans geste
préalable, ces rejeux restent périmés.

### D1. Contrôles avant (l'utilisateur, ou une session prévenue)

- Espace disque du serveur : prévoir au moins 5 Go libres. Rejeux : environ +0,85 Go (local, mesuré). Dossier
  d'attente : 3,0 Go de plus tant qu'il existe. Dossier de repli de D6 : 2,15 Go de plus. Faits de film : +0,3 Go
  en local (7,2 puis 7,5 Go) ; le serveur n'en garde que s'il a gardé des films : à vérifier avant le déploiement.
  Commande : `df -h /opt/levelup/data` et `du -sh /opt/levelup/data/cache/replays`.
- Sauvegarde restic fraîche (modèle : P4 de `gestes_prod_8_2_0.md`, procédure `scripts/RESTIC_BACKUP.md`).
- Aucune passe `backfill` en cours : `pgrep -f '[b]ackfill'` vide.
- Archive des films : vidée le 2026-11-23 (backlog, entrée « [films/prod] Le calcul des frags archive chaque film... »).
  Contrôle : liste du dossier d'archive vide ou presque.
- Rejeux de la démonstration : une montée de schéma rend tous les rejeux figés de la démonstration périmés ; il faut
  les cuire en local et les envoyer AVANT le push sur `main` (étape permanente « Replay schema bump » de
  `docs/RUNBOOK_DEPLOY_CHECKLIST.md` : `levelup backfill-replay --demo --dry-run` puis `--demo`, `rsync` des seuls
  `.json.gz` de la démonstration). Contrôle : au déploiement, le journal de `seed-demo` (section `rejeux figés`) ne
  liste aucun rejeu `perime`, `recuit` ou `absent`. Si échec : la démonstration garde son rejeu précédent ou ne le
  sert pas, sans plantage (runbook) ; refaire l'envoi et relancer le déploiement de la démonstration.

### D2. Mettre les rejeux au schéma 127 AVANT le premier démarrage de la 8.3.0

Deux voies ; seule la première couvre les matchs dont le film n'existe plus sur le serveur.

**Voie (a), à retenir : copier les rejeux régénérés en local** (régénération du 2026-11-24 : 1 455 rejeux au schéma
127, 1 h 11 ; `l6_regeneration_reelle_resume.md`).

Condition : la régénération locale est refaite si le décodeur ou le schéma change avant la release (la v1.6.1 est celle
qu'exige aujourd'hui `apps/go-api/go.mod`). Aucune base ne se copie : le dossier des rejeux seul (`.json.gz` et
`.meta`, ni `rasters/`, ni `.derived.json`, comme en 8.2.1).

1. Qui : l'utilisateur, ou une session après l'avoir prévenu. Quand : avant le déploiement ; le serveur peut rester
   actif, on ne touche pas au dossier servi à cette étape.
2. Poser les fichiers dans un dossier d'attente SOUS `data/`, à l'abri du `git clean` du déploiement (leçon de la
   8.2.0) : `/opt/levelup/data/replays-staging-v8.3.0/halo_infinite/`. Transfert : 3,0 Go (1 455 rejeux), contre
   2,15 Go avant. Modèle de commande : le `rsync` de `docs/RUNBOOK_DEPLOY_CHECKLIST.md`. Contrôle : 1 455 `.json.gz`
   et 1 455 `.meta`, taille totale identique au local, et `grep -l '"schemaVersion":127' .../*.meta | wc -l` égal
   à 1 455. Si échec : relancer le transfert (rsync reprend) ; ne rien poser dans le dossier servi.
3. Bascule : `bash data/replays-staging-v8.3.0/BASCULE.sh` (simulation), puis `... --apply` (copie par temporaire et
   renommage, contrôle de taille, comme à la 8.2.1 : 1 455 rejeux copiés attendus, 0 erreur). Les rejeux servis sans
   équivalent régénéré ne sont pas touchés. Si échec : le script s'arrête à la première erreur ; relancer après
   correction (il est idempotent).
   **À vérifier avant le déploiement** : `BASCULE.sh` n'est pas versionné (il vit sur le serveur) ; le recréer depuis
   celui de la 8.2.1 en changeant le dossier d'attente et le schéma attendu (127).

Point d'ordre à trancher : les notes de version et le plan demandent les rejeux au schéma 127 AVANT le premier
démarrage de la 8.3.0, alors que la bascule de la 8.2.x se faisait APRÈS le démarrage. Option retenue par la
consigne : serveur arrêté, bascule `--apply`, puis démarrage de la 8.3.0 (le dossier servi est au 127 dès le premier
démarrage ; un ancien binaire arrêté ne le lit pas). Ce que fait le premier démarrage devant des rejeux à l'ancien schéma sans
film est ÉTABLI (vérifié le 2026-11-24 par lecture du code, non rejoué sur une copie) : rien. La 8.3.0 les sert tels quels (`apps/go-api/internal/service/replay_service.go`, `GetReplay` : aucune comparaison de schéma à la lecture, désérialisation tolérante, calques récents absents) ; la régénération à la demande est réservée au bureau et au développement (`apps/go-api/internal/config/deployment.go`, capacité `GenerateReplays`), donc sur le site hébergé rien n'est remis en file ni retéléchargé ; seul un administrateur voit le badge « à régénérer » (`apps/web/src/features/match-replay/ui/ReplaySchemaBadge.tsx`) ; `docs/RELEASE_NOTES.md` annonce que les anciens rejeux restent lisibles. Non bloquant ; ces rejeux n'auront jamais les calques neufs (entrée du backlog « Rejeux du site hébergé à l'ancien schéma, sans film »).

**Voie (b), régénérer sur le serveur : impossible pour les matchs sans film.** L'archive des films du site hébergé a
été vidée le 2026-11-23, et une tâche planifiée en retire chaque heure les films de plus de 24 h (backlog). Le serveur
a rempli son disque au rythme d'environ 20 Go de films par semaine, avec 5 425 films de frags en retard au relevé :
ces films servent au calcul des frags, rien ne garantit qu'un film soit présent pour un match de rejeu donné. À la
8.2.1, environ 2 080 rejeux servis n'avaient déjà pas d'équivalent régénéré faute de film (`gestes_prod_8_2_1.md`).
Cette voie ne couvre au mieux que les matchs dont le film est encore là : ne pas compter dessus pour le parc.

### D3. Passes dérivées à rejouer SUR le serveur, après la bascule (serveur arrêté, une fenêtre)

Elles écrivent dans la base partagée du serveur : elles ne se copient pas. Ordre de la procédure locale
(`l6_regeneration_procedure.md`), durées mesurées en local le 2026-11-24 ; le serveur a moins de cœurs, compter plus
(non mesuré : à vérifier avant le déploiement) :

```bash
cd /opt/levelup && docker compose stop levelup
LU() { docker compose run --rm --no-deps levelup levelup "$@"; }
LU backfill-usage-summary            # 3 min 50 en local ; sans --force
LU backfill-pad-tiers --force        # 3 min 47
LU backfill-killsource --workers 1   # 2 min 33 en local (1 454 films sautés) ; voir la note ci-dessous
LU backfill-vehicle-takes --force    # 3 min 43
LU backfill-flag-grabs-net --force   # 3 min 34
LU tactical-rasters --backfill       # 4 min 34 ; seconde passe : 0 écrit
LU backfill-kickoff --dry-run        # d'abord, lecture seule
LU backfill-kickoff                  # 7,8 s en local ; range la révision du coup d'envoi
```

- `backfill-killsource` décode des films : sur le serveur, sans film, il est sans objet (décision de la 8.2.1 :
  « sans objet en prod, fait en local »). À vérifier avant le déploiement : si on l'omet, quel état les journaux
  de morts du serveur gardent-ils après la montée.
- Contrôle de chaque passe : le bilan imprimé (0 échec), puis la passe relancée dit tout « à jour ». Durée cumulée
  en local : environ 20 min ; prévoir une fenêtre de 40 min sur le serveur.
- Si une passe échoue : la relancer (elles reprennent où elles se sont arrêtées) ; ne pas enchaîner la suivante.
- Rattrapage du coup d'envoi : par la commande ci-dessus, sinon au démarrage quand la révision `kickoff_revision`
  du `sync_meta` diffère de celle du binaire (ADR 0050). Lancée avant, elle évite l'attente au démarrage. Contrôle :
  une seconde simulation donne 0 changement.
- Recalcul des notes : automatique au premier démarrage, joueur par joueur (ADR 0047) ; rien à lancer. Contrôle :
  journal `MatchRecomputer: terminé` pour chaque joueur, et un second redémarrage qui ne relance rien. Ne pas
  redémarrer en boucle pendant le recalcul.
- Redémarrage : `docker compose up -d levelup`, puis `/health` répond 200.

### D4. Nombre de fils du décodeur sur le serveur : 1, aucun réglage

Le nombre est dérivé du profil de déploiement (ADR 0044) : `decoderThreadsOf` rend 4 pour `desktop` et `dev`, 1 pour
`hosted` et `demo` (`apps/go-api/internal/config/deployment.go`, relu). Rien à poser. Au-delà de 1 fil seulement, le
décodeur peut employer brièvement quelques goroutines de plus que demandé (commentaire de `decoderThreadsLocal`).

Contrôle : aucun journal n'imprime ce nombre (procédure locale). Le contrôle indirect est que le serveur tourne bien
au profil hébergé. **À vérifier avant le déploiement** : l'endroit exact où se lit le profil (journal de démarrage ou
réponse de `/bootstrap`).

### D5. Purge horaire des films du serveur : elle reste en place

L'unité `levelup-purge-films.timer` (script `/usr/local/bin/levelup-purge-films.sh`) retire chaque heure les films
archivés depuis plus de 24 h (backlog). Elle reste tant que le correctif n'est pas livré : « le calcul des frags
respecte le réglage d'archivage », avec un site hébergé qui n'archive plus par défaut (capacité du profil de
déploiement, ADR 0044) et une application Windows qui continue d'archiver. Condition de retrait : correctif livré ET
déployé, puis contrôle que le disque ne regrossit plus. Geste de retrait (l'utilisateur, non exécuté ici) :
`systemctl disable --now levelup-purge-films.timer`. Si le disque regrossit après : `systemctl enable --now
levelup-purge-films.timer`. L'état du correctif à la clôture de la 8.3.0 est à vérifier avant le déploiement.

### D6. Retour arrière

- Avant la bascule, copier (pas déplacer) le dossier des rejeux servi, schéma 125, vers un dossier de repli sous
  `data/` ; le garder jusqu'au contrôle de D7. Retours arrière certains : ce dossier de repli et la sauvegarde restic.
- Une 8.2.x ne lit pas des rejeux au schéma 127 : non prouvé. La 8.2.1 écrit seulement qu'une 8.2.0 ne connaît pas
  le schéma 124 (`gestes_prod_8_2_1.md`). À vérifier avant le déploiement : essai d'une 8.2.x sur une copie de rejeu
  au schéma 127, ou lecture de `replaybuild`. En attendant, un retour arrière du binaire impose aussi de remettre
  le dossier de repli.
- Les passes dérivées sont append-only : un retour arrière ne les efface pas ; l'ancien binaire lit ses vues
  `_latest` (ADR 0026). Les révisions de notes et de coup d'envoi rangées en base peuvent faire rejouer ces recalculs
  au prochain démarrage d'un autre binaire (comparaison sur « différent », ADR 0047) : le dire avant de revenir en
  arrière (même remarque que G1 de `gestes_prod_8_2_0.md`).

### D7. Contrôles après, puis nettoyage

1. Trois rejeux ouverts sur le site : une arène, une Grande bataille, un Drapeau. Chacun s'ouvre et se lit sans
   message d'erreur.
2. Page d'administration des rejeux (panneau de régénération, `apps/web/src/components/replay-regen/`) : aucun rejeu
   signalé « à régénérer » pour les matchs copiés. Libellé exact à vérifier avant le déploiement.
3. Journaux du serveur sans erreur de schéma : `grep -riE 'FATAL|panic|schema' /opt/levelup/data/logs/*.log`, en
   lisant chaque ligne trouvée (le mot « schema » peut figurer dans des lignes normales).
4. `grep -l '"schemaVersion":127' /opt/levelup/data/cache/replays/halo_infinite/*.meta | wc -l` au moins égal à 1 455.
5. Seulement ensuite : supprimer le dossier d'attente (`rm -rf /opt/levelup/data/replays-staging-v8.3.0`), puis, sur
   un second accord de l'utilisateur, le dossier de repli de D6.

Plafond de mémoire des cuissons (`-plafond-gio`, réglage de l'hôte, pas du décodeur) : à 1 Gio le pic du plus gros film
tombe à 1 161 Mio pour +1,3 % de temps (mesuré sur ce seul film, `.ai/V8.3/annexes_decodeur_v160/profil2.md`, section 6.2) ;
la régénération réelle du 2026-11-24 a donné un pic d'un enfant de 1,64 Gio pour un plafond de 3 Gio, sans aucun arrêt.
À dimensionner d'après la mémoire du serveur.

Points « à vérifier avant le déploiement » de cette section : faits de film du serveur (D1), recréation de
`BASCULE.sh` (D2, le comportement du premier démarrage devant des rejeux périmés est établi, voir plus haut), durée des passes et rôle de
`backfill-killsource` sur le serveur (D3), lecture du profil hébergé (D4), état du correctif d'archivage (D5),
comportement d'une 8.2.x devant le schéma 127 (D6), libellé du panneau d'administration (D7).

## Archivage des films et démarrage du serveur (backlog du 24 novembre, piste serveur)

Écrit le 2026-11-24 à l'intégration de la piste. Rien n'est exécuté. Complète D5 : la purge horaire des films
ne se retire qu'après le contrôle A3.

### A1. Avant de déployer : lire deux clés du fichier de réglages du serveur

Dans `app_settings.json` du serveur, lire `archive_films` (présente ou absente, sa valeur) et
`replay_retention_months`.

| `archive_films` | `replay_retention_months` | Effet de la 8.3.0 |
|---|---|---|
| absente | 0 | rien ne change ; le site n'archive plus les films (défaut du profil hébergé) |
| absente | plus de 0 | la durée de conservation est ramenée à 0 (illimitée) avec un seul avertissement au journal, puis écrite ainsi dans le fichier à la première modification d'un réglage |
| `true` écrite | toute valeur | le nouveau défaut ne s'applique pas : le site continue d'archiver tant que la clé n'est pas passée à `false` |
| `false` | 0 | état visé du site hébergé |

Décision à prendre dans le cas « absente, plus de 0 » : accepter la conservation illimitée des rejeux, ou
écrire `archive_films: true` en connaissance de cause (le site archive alors de nouveau les films).

### A2. Si la clé vaut `true` : la passer à `false`

Par `PATCH /settings`, après avoir vérifié que `replay_retention_months` vaut 0 (sinon la demande est refusée).

### A3. Après le déploiement : contrôler, puis retirer la purge horaire

- Le dossier d'archive des films ne grossit plus ; le compteur `killsource_films_non_archives` monte.
- Alors seulement : retirer la tâche planifiée de purge horaire des films (minuterie systemd du serveur ; son
  nom se lit par `systemctl list-timers`).

### A4. Démarrage : deux lignes à lire au journal

`pool: créé vide…` puis, à la fin de la résolution des comptes, le bilan avec `resolve_ms`. Un bilan « aucun
compte résolu » demande d'authentifier un joueur. Aucun geste. Le gain de temps au démarrage n'a été mesuré que
sur un banc simulé : relever le temps réel jusqu'à « serveur prêt » au premier déploiement.
