package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlagsPrintConfiguredVersion(t *testing.T) {
	const wantVersion = "v0.1.0"

	binary := filepath.Join(t.TempDir(), "unfold")
	build := exec.Command("go", "build", "-ldflags", "-X main.version="+wantVersion, "-o", binary, "..")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	for _, args := range [][]string{{"--version"}, {"-V"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output, err := exec.Command(binary, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("run %q: %v\n%s", args, err, output)
			}
			if got, want := string(output), "unfold version "+wantVersion+"\n"; got != want {
				t.Errorf("version output = %q, want %q", got, want)
			}
		})
	}
}

func TestVersionFlagShorthandsDoNotConflict(t *testing.T) {
	for name, wantShorthand := range map[string]string{
		"debug":   "v",
		"version": "V",
	} {
		flag := rootCmd.PersistentFlags().Lookup(name)
		if flag == nil {
			t.Fatalf("%s flag is not registered", name)
		}
		if got := flag.Shorthand; got != wantShorthand {
			t.Errorf("%s shorthand = %q, want %q", name, got, wantShorthand)
		}
	}
}
