# Annexe - Classification Des Constats Post-v6.2.1

## But

Cette annexe classe les constats de revue en trois groupes:

1. certainement nouveaux depuis v6.2.1,
2. probablement aggraves depuis v6.2.1,
3. anciens mais toujours toxiques.

Reference temporelle:

- `v6.2.1` = 2026-03-29
- `v6.3.0` = 2026-04-06

Le but n'est pas de refaire la revue, mais d'eviter l'amalgame entre regression recente, dette revelee par un nouveau chantier, et passif ancien encore dangereux.

## 1. Certainement Nouveaux Depuis v6.2.1

### N1 - Contrat ambigu de persistance UI v6.4

Pourquoi ici:

- La semantique localStorage dans `data_loader.py` est introduite le 2026-04-07.
- Le module `browser_storage` utilise dans le runtime actuel a ete fortement remanie le 2026-04-07.

Constat:

- Le code parle de localStorage navigateur, mais la logique Python active persiste dans un fichier global serveur.
- Le vrai frontend localStorage existe, mais n'est pas la piece visible du parcours principal audite.

Impact:

- Risque d'ecrasement croise entre sessions/navigateurs/utilisateurs.
- Contrat trompeur pour la maintenance future.

### N2 - Migration legacy des prefs non atomique

Pourquoi ici:

- `_resolve_prefs_path()` et sa migration `.streamlit -> data/players` datent du 2026-04-07.

Constat:

- La copie legacy est suivie d'une suppression de l'ancien fichier sans strategie atomique ni rollback.

Impact:

- Perte potentielle des preferences en cas d'erreur I/O pendant la migration.

### N3 - Watcher media Linux sans guard process-level

Pourquoi ici:

- Le watcher Linux est introduit le 2026-04-07.
- Le guard `_PERIODIC_STARTED` existait deja, mais ne couvre que le polling legacy.

Constat:

- En mode Linux, le code peut recreer observer + scan initial au rerun.

Impact:

- Duplication de threads/observers, contention I/O, comportement non deterministe.

### N4 - Healthcheck auto-repair avec statut global incoherent

Pourquoi ici:

- Le module healthcheck et son auto-repair sont introduits le 2026-04-07.

Constat:

- Les checks peuvent passer en `repaired`, mais le statut agregat n'est pas recompose apres modification.

Impact:

- Signal d'exploitation ambigu entre detail des checks et statut JSON global.

### N5 - Duplication de logique destructive dans le deploiement

Pourquoi ici:

- `deploy.sh` et la duplication dans le workflow sont introduits apres v6.2.1, les 2026-04-02 a 2026-04-06.

Constat:

- `git fetch/reset/clean` existent a la fois dans le workflow et dans le script VPS.

Impact:

- Dette de maintenance infra et risque de divergence future.

### N6 - Parsing post-deploy trop permissif sur les cas UNKNOWN

Pourquoi ici:

- Le parsing JSON du healthcheck dans `deploy.sh` est nouveau avec le healthcheck post-deploy.

Constat:

- Des erreurs stderr ou un crash du script peuvent etre aplatis en statut peu exploitable.

Impact:

- Observabilite infra degradee au pire moment: juste apres deploy.

## 2. Probablement Aggraves Depuis v6.2.1

### A1 - Dependance metadata -> shared devenue structurelle

Pourquoi ici:

- L'ordre `shared -> metadata` dans le runner est plus ancien que v6.2.1.
- En revanche, l'ajout des colonnes FR de `mv_player_matches` et le fallback `NULL AS ..._fr` datent d'apres v6.2.1.

Constat:

- Une faiblesse ancienne d'ordre de migration est devenue un probleme concret avec les vues i18n v6.3/v6.4.

Impact:

- Vues degradees silencieusement au lieu d'un ordre de migration deterministic.

### A2 - Obsolescence de la documentation agentique amplifiee par v6

Pourquoi ici:

- Les references `shared_matches.duckdb` dans `CLAUDE.md` sont anterieures au tag v6.2.1.
- Depuis v6.3/v6.4, l'ecart entre la doc et la realite s'est accentue.

Constat:

- La doc active pousse encore un modele de stockage qui n'est plus la source reelle.

Impact:

- Reintroduction possible de chemins legacy ou d'hypotheses fausses par les agents et contributeurs.

### A3 - Dette de taille/complexite des modules critiques

Pourquoi ici:

- Certains modules etaient deja lourds avant v6.2.1.
- Les chantiers v6.3/v6.4 les ont encore charges ou les maintiennent a la limite.

Modules les plus sensibles:

- `src/data/sync/_engine_connections.py`
- `src/utils/healthcheck_db.py`
- `src/ui/pages/media_tab.py`
- `src/ui/pages/settings.py`

Impact:

- Cout de correction plus eleve, tentation de workaround local, surface de regression accrue.

### A4 - Observabilite media plus importante depuis l'ajout du watcher

Pourquoi ici:

- `_index_with_retry()` est plus ancien que v6.2.1.
- Avec le watcher Linux et l'indexation plus reactive, le manque de log sur les succes apres retry devient plus penalant.

Impact:

- Diagnostic plus difficile des erreurs transitoires et des recuperations silencieuses.

## 3. Anciens Mais Toujours Toxiques

### T1 - Guard des migrations shared valide trop tot

Pourquoi ici:

- Le marquage `_SHARED_MIGRATIONS_DONE.add(db_key)` avant succes effectif date du 2026-03-18, donc avant v6.2.1.

Constat:

- Une base peut etre consideree comme migree pour tout le process alors qu'une migration critique a echoue.

Impact:

- Faux succes process-level, absence de retry naturel, etat partiellement migre.

### T2 - Ordre de base du runner non modele par dependance

Pourquoi ici:

- L'ordre `shared`, puis `shared_pve`, puis `player`, puis `metadata` est anterieur a v6.2.1.

Constat:

- Le runner ne porte pas de graphe de dependances de migrations; il repose sur un ordre fixe historique.

Impact:

- Toute nouvelle dependance inter-bases risque de se traduire en fallback ou patch correctif local.

### T3 - Retry media minimaliste

Pourquoi ici:

- `_index_with_retry()` date du 2026-03-07, donc avant v6.2.1.

Constat:

- Il n'expose pas clairement le chemin de recuperation apres echec intermediaire.

Impact:

- Dette d'observabilite ancienne, devenue plus visible avec les derniers chantiers media.

### T4 - Documentation active partiellement figee dans le modele pre-v6

Pourquoi ici:

- Le probleme n'est pas limite a un oubli ponctuel recent; c'est une dette documentaire qui traverse plusieurs iterations.

Impact:

- Les corrections purement code ne suffiront pas si les documents prescriptifs ne sont pas nettoyes.

## Lecture Recommandee

- Les items `N*` sont a traiter comme regressions ou defauts de conception recents.
- Les items `A*` sont souvent les plus piegeux: ils semblent nouveaux, mais revelent une faiblesse plus ancienne qui vient de changer d'echelle.
- Les items `T*` ne doivent pas etre repousses indefiniment sous pretexte qu'ils preexistent a v6.2.1; ils restent des multiplicateurs de risque.

## Ce Que Je N'ai Pas Vu En Plus A Ce Stade

Apres cette passe supplementaire, je ne vois pas d'autre finding majeur du meme niveau de severite que ceux deja listes.

Il peut rester:

- des micro-dettes de style,
- des incoherences de logs secondaires,
- des points preexistants hors delta,

mais rien que je considererais aujourd'hui comme omission critique par rapport a la revue et au plan.
