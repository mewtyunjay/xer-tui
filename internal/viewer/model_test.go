package viewer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestJumpToMatchWrapsAcrossRows(t *testing.T) {
	data := mustLoadViewerData(t)
	m := NewModel(data, t.TempDir())
	m.tableIndex = findTableIndex(data.Tables, "TASK")

	m.applyRowQuery("concrete", false)

	if got, want := m.matchedRows, []int{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("matched rows = %v, want %v", got, want)
	}
	if got, want := m.selectedRow, 1; got != want {
		t.Fatalf("selected row = %d, want %d", got, want)
	}

	m.jumpToMatch(1)
	if got, want := m.selectedRow, 2; got != want {
		t.Fatalf("selected row after next = %d, want %d", got, want)
	}
	if got, want := m.status, "match 2/2"; got != want {
		t.Fatalf("status after next = %q, want %q", got, want)
	}

	m.jumpToMatch(1)
	if got, want := m.selectedRow, 1; got != want {
		t.Fatalf("selected row after wrap next = %d, want %d", got, want)
	}

	m.jumpToMatch(-1)
	if got, want := m.selectedRow, 2; got != want {
		t.Fatalf("selected row after wrap prev = %d, want %d", got, want)
	}
	if got, want := m.status, "match 2/2"; got != want {
		t.Fatalf("status after prev = %q, want %q", got, want)
	}
}

func TestRowFilterRestrictsVisibleRows(t *testing.T) {
	data := mustLoadViewerData(t)
	m := NewModel(data, t.TempDir())
	m.tableIndex = findTableIndex(data.Tables, "TASK")

	m.applyRowQuery("concrete", true)

	if !m.rowFilterActive {
		t.Fatalf("row filter should be active")
	}
	if got, want := m.visibleRowCount(), 2; got != want {
		t.Fatalf("visible row count = %d, want %d", got, want)
	}
	if got, want := m.filteredRows, []int{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("filtered rows = %v, want %v", got, want)
	}
	if got, want := m.selectedActualRow(), 1; got != want {
		t.Fatalf("selected actual row = %d, want %d", got, want)
	}

	m.moveRow(1)
	if got, want := m.selectedActualRow(), 2; got != want {
		t.Fatalf("selected actual row after move = %d, want %d", got, want)
	}

	m.jumpToMatch(1)
	if got, want := m.selectedRow, 0; got != want {
		t.Fatalf("selected visible row after wrap = %d, want %d", got, want)
	}

	m.clearFilters()
	if m.rowFilterActive {
		t.Fatalf("row filter should be cleared")
	}
	if got, want := m.selectedRow, 1; got != want {
		t.Fatalf("selected row after clearing filter = %d, want %d", got, want)
	}
}

func TestBrowserListsAndOpensXERFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestXER(t, filepath.Join(dir, "alpha.xer"), "Alpha")
	writeTestXER(t, filepath.Join(dir, "beta.xer"), "Beta")
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	m := NewModel(nil, dir)
	if got, want := m.mode, screenBrowser; got != want {
		t.Fatalf("mode = %q, want %q", got, want)
	}
	if got, want := len(m.files), 2; got != want {
		t.Fatalf("files len = %d, want %d", got, want)
	}
	if got, want := filepath.Base(m.files[0]), "alpha.xer"; got != want {
		t.Fatalf("first file = %q, want %q", got, want)
	}

	m.fileIndex = 1
	if err := m.openSelectedFile(); err != nil {
		t.Fatalf("open selected file: %v", err)
	}

	if got, want := m.mode, screenViewer; got != want {
		t.Fatalf("mode after open = %q, want %q", got, want)
	}
	if m.data == nil {
		t.Fatalf("data should be loaded")
	}
	if got, want := m.data.Name, "beta.xer"; got != want {
		t.Fatalf("opened file = %q, want %q", got, want)
	}
}

func mustLoadViewerData(t *testing.T) *FileData {
	t.Helper()

	const sample = "" +
		"ERMHDR\t19.12\t2026-04-02\txv-test\talice\tAlice\tSandbox\tProject Management\tUSD\n" +
		"%T\tTASK\n" +
		"%F\ttask_id\ttask_name\tstatus_code\n" +
		"%R\t1\tExcavate\tTK_NotStart\n" +
		"%R\t2\tPour Concrete\tTK_Active\n" +
		"%R\t3\tFinish Concrete\tTK_Active\n" +
		"%R\t4\tBackfill\tTK_NotStart\n"

	data, err := Load(strings.NewReader(sample), "sample.xer")
	if err != nil {
		t.Fatalf("load sample: %v", err)
	}
	return data
}

func findTableIndex(tables []TableData, name string) int {
	for i, table := range tables {
		if table.Name == name {
			return i
		}
	}
	return -1
}

func writeTestXER(t *testing.T, path, projectName string) {
	t.Helper()

	content := "" +
		"ERMHDR\t19.12\t2026-04-02\txv-test\talice\tAlice\tSandbox\tProject Management\tUSD\n" +
		"%T\tPROJECT\n" +
		"%F\tproj_id\tproj_short_name\n" +
		"%R\t1\t" + projectName + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
