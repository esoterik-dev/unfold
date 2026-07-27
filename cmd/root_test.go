package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionFlagsPrintConfiguredVersion(t *testing.T) {
	const wantVersion = "v0.1.0"

	originalVersion := Version
	Version = wantVersion
	t.Cleanup(func() { Version = originalVersion })

	for _, args := range [][]string{{"--version"}, {"-V"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			versionFlag := rootCmd.PersistentFlags().Lookup("version")
			if versionFlag == nil {
				t.Fatal("version flag is not registered")
			}
			if err := versionFlag.Value.Set("false"); err != nil {
				t.Fatalf("reset version flag: %v", err)
			}
			versionFlag.Changed = false

			var output bytes.Buffer
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)
			rootCmd.SetArgs(args)
			t.Cleanup(func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
				rootCmd.SetArgs(nil)
				_ = versionFlag.Value.Set("false")
				versionFlag.Changed = false
			})

			if err := Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", args, err)
			}
			if got, want := output.String(), "unfold version "+wantVersion+"\n"; got != want {
				t.Errorf("version output = %q, want %q", got, want)
			}
		})
	}
}
