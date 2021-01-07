package config

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	f, err := os.Open("testdata/test.json")
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()
	actions, err := Parse(f)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := actions.Prepare(); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	t.Log(actions)
}
