package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestStripLeadingConfigCompatArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{name: "no args", args: []string{}, want: []string{}},
		{name: "single leading config", args: []string{"-c", "/tmp/config", "doctor", "-y"}, want: []string{"doctor", "-y"}},
		{name: "multiple leading config flags", args: []string{"-c", "/a", "--config", "/b", "status"}, want: []string{"status"}},
		{name: "non-leading config preserved", args: []string{"update", "-c", "/tmp/config"}, want: []string{"update", "-c", "/tmp/config"}},
		{name: "missing path", args: []string{"--config"}, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripLeadingConfigCompatArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("mismatch\nwant: %#v\n got: %#v", tt.want, got)
			}
		})
	}
}

func TestRunSetupMaintenanceCommandForwardsToSetup(t *testing.T) {
	origResolver := setupDirResolver
	origRunner := setupCommandRunner
	defer func() {
		setupDirResolver = origResolver
		setupCommandRunner = origRunner
	}()

	setupDirResolver = func() (string, error) {
		return "/tmp/inir", nil
	}

	var gotDir string
	var gotArgs []string
	setupCommandRunner = func(dir string, args []string) error {
		gotDir = dir
		gotArgs = append([]string{}, args...)
		return nil
	}

	err := runSetupMaintenanceCommand(&cobra.Command{}, []string{"-c", "/tmp/config", "-y", "--local"}, "doctor", "usage")
	if err != nil {
		t.Fatalf("runSetupMaintenanceCommand returned error: %v", err)
	}

	if gotDir != "/tmp/inir" {
		t.Fatalf("expected setup dir /tmp/inir, got %q", gotDir)
	}

	wantArgs := []string{"doctor", "-y", "--local"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("forwarded args mismatch\nwant: %#v\n got: %#v", wantArgs, gotArgs)
	}
}

func TestRunSetupEntrypointCommandForwardsRawSetupArgs(t *testing.T) {
	origResolver := setupDirResolver
	origRunner := setupCommandRunner
	defer func() {
		setupDirResolver = origResolver
		setupCommandRunner = origRunner
	}()

	setupDirResolver = func() (string, error) {
		return "/tmp/inir", nil
	}

	var gotArgs []string
	setupCommandRunner = func(dir string, args []string) error {
		gotArgs = append([]string{}, args...)
		return nil
	}

	err := runSetupEntrypointCommand(&cobra.Command{}, []string{"-c", "/tmp/config", "install", "-y"})
	if err != nil {
		t.Fatalf("runSetupEntrypointCommand returned error: %v", err)
	}

	wantArgs := []string{"install", "-y"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("forwarded args mismatch\nwant: %#v\n got: %#v", wantArgs, gotArgs)
	}
}
