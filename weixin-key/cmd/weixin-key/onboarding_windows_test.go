//go:build windows

package main

import (
	"path/filepath"
	"testing"

	"weixin-key/internal/config"
)

func TestInitAccountCLIEmitsOneProofAndPreservesExisting(t *testing.T) {
	source := onboardingSource(t)
	cfg := filepath.Join(t.TempDir(), "config.json")
	args := []string{"init-account", "--db-root", source, "--config", cfg, "--account", "wxid_fixture"}
	code, stdout, stderr := runCmd(t, args...)
	var proof config.MetadataInitProof
	decodeSingleJSON(t, stdout, &proof)
	if code != exitOK || stderr != "" || !proof.Applied || !proof.MetadataVerified || !proof.PrivateFile {
		t.Fatal("CLI initialization did not complete", stderr)
	}
	code, stdout, stderr = runCmd(t, args...)
	decodeSingleJSON(t, stdout, &proof)
	if code != exitError || stderr == "" || proof.Applied {
		t.Fatal("CLI retry replaced existing config")
	}
}
