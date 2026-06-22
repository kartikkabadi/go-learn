package practice

import "testing"

func TestExpectedOutput(t *testing.T) {
	got, ok := ExpectedOutput("practice/hello")
	if !ok {
		t.Fatal("expected hello to have an expected file")
	}
	if got != "Hello, world!" {
		t.Fatalf("hello: got %q", got)
	}
	if _, ok := ExpectedOutput("practice/nonexistent"); ok {
		t.Fatal("nonexistent should not have an expected file")
	}
}
