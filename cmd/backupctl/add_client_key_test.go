package main

import "testing"

func TestBuildAuthorizedKeysLine(t *testing.T) {
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... backupctl-client"
	line, err := buildAuthorizedKeysLine(pubKey)
	if err != nil {
		t.Fatalf("buildAuthorizedKeysLine: %v", err)
	}
	if line != authorizedKeysOptions+" "+pubKey {
		t.Errorf("line = %q", line)
	}
}

func TestBuildAuthorizedKeysLine_Invalid(t *testing.T) {
	cases := []string{"", "not-a-key", "  "}
	for _, c := range cases {
		if _, err := buildAuthorizedKeysLine(c); err == nil {
			t.Errorf("buildAuthorizedKeysLine(%q) expected an error", c)
		}
	}
}

func TestResolvePubKeyArg_LiteralKey(t *testing.T) {
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... test"
	got, err := resolvePubKeyArg(key)
	if err != nil {
		t.Fatalf("resolvePubKeyArg: %v", err)
	}
	if got != key {
		t.Errorf("got = %q, want %q", got, key)
	}
}

func TestResolvePubKeyArg_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/id_ed25519.pub"
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... test\n"
	writeFile(t, path, key)

	got, err := resolvePubKeyArg(path)
	if err != nil {
		t.Fatalf("resolvePubKeyArg: %v", err)
	}
	if got != key {
		t.Errorf("got = %q, want %q", got, key)
	}
}

func TestResolvePubKeyArg_Empty(t *testing.T) {
	if _, err := resolvePubKeyArg(""); err == nil {
		t.Fatal("expected an error for an empty -pubkey")
	}
}
