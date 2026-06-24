package handlers

import (
	"strings"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/practice"
)

func TestGradeExercise_NormalizesCRLF(t *testing.T) {
	// practice/factorial has multi-line expected output with \n line endings.
	// A Windows user pasting the same output with \r\n should still match.
	expected, ok := practice.ExpectedOutput("practice/factorial")
	if !ok {
		t.Fatal("practice/factorial should have expected output")
	}
	crlf := strings.ReplaceAll(expected, "\n", "\r\n")
	if !gradeExercise("practice/factorial", crlf) {
		t.Fatal("CRLF output should match LF expected after normalization")
	}
}

func TestGradeExercise_NoExpectedFile(t *testing.T) {
	if !gradeExercise("practice/nonexistent", "anything") {
		t.Fatal("missing expected file should pass (no grading)")
	}
}
