package main

import "testing"

func TestParseArgsRecognizesMigrationMode(t *testing.T) {
	migrateOnly, err := parseArgs([]string{"-migrate"})
	if err != nil || !migrateOnly {
		t.Fatalf("parseArgs(-migrate) = %v, %v; want true, nil", migrateOnly, err)
	}
	migrateOnly, err = parseArgs(nil)
	if err != nil || migrateOnly {
		t.Fatalf("parseArgs(nil) = %v, %v; want false, nil", migrateOnly, err)
	}
}
