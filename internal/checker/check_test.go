package checker_test

import (
	"testing"

	"github.com/tamnd/bunpy/v1/internal/checker"
)

func issues(t *testing.T, src string) []checker.Issue {
	t.Helper()
	return checker.Check("test.py", src)
}

func hasCode(issues []checker.Issue, code string) bool {
	for _, i := range issues {
		if i.Code == code {
			return true
		}
	}
	return false
}

func TestCheckBareExcept(t *testing.T) {
	if !hasCode(issues(t, "try:\n    pass\nexcept:\n    pass\n"), "E001") {
		t.Error("expected E001 for bare except")
	}
}

func TestCheckExceptWithType(t *testing.T) {
	if hasCode(issues(t, "try:\n    pass\nexcept Exception:\n    pass\n"), "E001") {
		t.Error("E001 should not fire for typed except")
	}
}

func TestCheckLongLine(t *testing.T) {
	long := "x = " + string(make([]byte, 120)) + "\n"
	if !hasCode(issues(t, long), "W002") {
		t.Error("expected W002 for long line")
	}
}

func TestCheckShortLine(t *testing.T) {
	if hasCode(issues(t, "x = 1\n"), "W002") {
		t.Error("W002 should not fire for short line")
	}
}

func TestCheckTrailingWhitespace(t *testing.T) {
	if !hasCode(issues(t, "x = 1   \n"), "W001") {
		t.Error("expected W001 for trailing whitespace")
	}
}

func TestCheckNoneComparison(t *testing.T) {
	if !hasCode(issues(t, "if x == None:\n    pass\n"), "E003") {
		t.Error("expected E003 for == None")
	}
}

func TestCheckBoolComparison(t *testing.T) {
	if !hasCode(issues(t, "if x == True:\n    pass\n"), "E004") {
		t.Error("expected E004 for == True")
	}
}

func TestCheckPrint2(t *testing.T) {
	if !hasCode(issues(t, `print "hello"`+"\n"), "E002") {
		t.Error("expected E002 for print statement")
	}
}

func TestCheckPrint3(t *testing.T) {
	if hasCode(issues(t, `print("hello")`+"\n"), "E002") {
		t.Error("E002 should not fire for print() call")
	}
}

func TestCheckClean(t *testing.T) {
	src := "x = 1\ny = x + 2\n"
	got := issues(t, src)
	if len(got) != 0 {
		t.Errorf("expected 0 issues for clean file, got: %v", got)
	}
}

func TestCheckUnusedImport(t *testing.T) {
	src := "import os\nx = 1\n"
	if !hasCode(issues(t, src), "E005") {
		t.Error("expected E005 for unused import")
	}
}

func TestCheckUsedImport(t *testing.T) {
	src := "import os\nx = os.path.join('a', 'b')\n"
	if hasCode(issues(t, src), "E005") {
		t.Error("E005 should not fire for used import")
	}
}
