package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

const scanTimeout = 25 * time.Second

func TestManagerScan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	for _, m := range manager.All() {
		m := m

		t.Run(string(m.Name()), func(t *testing.T) {
			t.Parallel()

			if !m.Available() {
				t.Skipf("%s not available on this system", m.Name())
			}

			ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
			defer cancel() //nolint:errcheck

			done := make(chan struct{})
			var pkgs []model.Package
			var err error

			go func() {
				pkgs, err = m.Scan()
				close(done)
			}()

			select {
			case <-ctx.Done():
				t.Fatalf("%s scan timed out", m.Name())
			case <-done:
			}

			if err != nil {
				t.Fatalf("%s scan failed: %v", m.Name(), err)
			}

			if pkgs == nil {
				t.Fatalf("%s returned nil package slice", m.Name())
			}

			t.Logf("%s: scanned %d packages", m.Name(), len(pkgs))

			for i, p := range pkgs {
				if p.Name == "" {
					t.Fatalf("%s: package[%d] has empty name", m.Name(), i)
				}
			}
		})
	}
}

func TestManagerCheckUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	for _, m := range manager.All() {
		m := m

		t.Run(string(m.Name()), func(t *testing.T) {
			t.Parallel()

			if !m.Available() {
				t.Skipf("%s not available on this system", m.Name())
			}

			checker, ok := m.(interface {
				CheckUpdates([]model.Package) map[string]string
			})

			if !ok {
				t.Skipf("%s has no update checker", m.Name())
			}

			pkgs, err := m.Scan()
			if err != nil {
				t.Fatalf("%s scan failed: %v", m.Name(), err)
			}

			if len(pkgs) == 0 {
				t.Skipf("%s has no installed packages", m.Name())
			}

			updates := checker.CheckUpdates(pkgs)

			if updates == nil {
				t.Fatalf("%s update checker returned nil map", m.Name())
			}

			t.Logf("%s: %d updates detected", m.Name(), len(updates))
		})
	}
}
