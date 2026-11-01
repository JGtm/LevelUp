// background_tasks_shutdown_test.go — les tâches de fond qui outlivent la requête rejoignent le
// contexte d'arrêt ET le groupe d'attente de la phase scheduler (ADR 0045 D-4, lot L12.3).
package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/config"
)

// La notification de version voit son contexte annulé à l'arrêt, et la phase scheduler (qui attend
// le groupe) ne rend la main qu'une fois la tâche finie : elle ne peut plus écrire dans une base
// déjà fermée.
func TestAppReleaseTask_SousLeContexteEtLeGroupeDArret(t *testing.T) {
	var wg sync.WaitGroup
	boot := newBootTasks(config.DeploymentDesktop, &wg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entree := make(chan struct{})
	sortie := make(chan struct{})
	if !boot.launch(appReleaseTask(ctx, func(c context.Context) {
		close(entree)
		<-c.Done()
		<-sortie // la tâche termine un travail après l'annulation
	})) {
		t.Fatal("app_release non lancée en desktop")
	}
	<-entree

	attente := make(chan struct{})
	go func() { wg.Wait(); close(attente) }()
	select {
	case <-attente:
		t.Fatal("le groupe d'attente rend la main alors que la notification de version tourne : elle n'est pas suivie")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case <-attente:
		t.Fatal("le groupe d'attente rend la main avant la fin du travail de la tâche")
	case <-time.After(50 * time.Millisecond):
	}
	close(sortie)
	select {
	case <-attente:
	case <-time.After(2 * time.Second):
		t.Fatal("la tâche annulée ne rend jamais la main")
	}
}
