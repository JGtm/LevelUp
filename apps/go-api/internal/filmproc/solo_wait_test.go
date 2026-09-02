package filmproc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSoloWaitRefuseApresLaBorne(t *testing.T) {
	root := t.TempDir()
	held, err := AcquireSolo(root, "test", "a")
	if err != nil {
		t.Fatalf("premier verrou : %v", err)
	}
	defer held.Release()
	debut := time.Now()
	_, err = AcquireSoloWait(context.Background(), root, "test", "b", 700*time.Millisecond)
	if !errors.Is(err, ErrDecodeBusy) {
		t.Fatalf("attendu ErrDecodeBusy apres la borne, obtenu %v", err)
	}
	if time.Since(debut) < 700*time.Millisecond {
		t.Fatalf("a refuse apres %s : il n'a pas attendu la borne", time.Since(debut))
	}
}

func TestSoloWaitPasseQuandLeVerrouEstRendu(t *testing.T) {
	root := t.TempDir()
	held, err := AcquireSolo(root, "test", "a")
	if err != nil {
		t.Fatalf("premier verrou : %v", err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		held.Release()
	}()
	l, err := AcquireSoloWait(context.Background(), root, "test", "b", 5*time.Second)
	if err != nil {
		t.Fatalf("attendu le verrou une fois rendu, obtenu %v", err)
	}
	l.Release()
}

func TestSoloWaitRespecteLAnnulation(t *testing.T) {
	root := t.TempDir()
	held, err := AcquireSolo(root, "test", "a")
	if err != nil {
		t.Fatalf("premier verrou : %v", err)
	}
	defer held.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = AcquireSoloWait(ctx, root, "test", "b", time.Minute)
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrDecodeBusy) {
		t.Fatalf("attendu une erreur portant l'annulation ET le refus, obtenu %v", err)
	}
}
