package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveUploadKeepsSpecialNamesInsideUploadRoot(t *testing.T) {
	dir := t.TempDir()
	s := &TicketService{uploadDir: dir}
	rel, err := s.SaveUpload("entry/../../../../outside.txt", []byte("secret"))
	if err != nil { t.Fatal(err) }
	abs := filepath.Join(dir, rel)
	root, err := filepath.Abs(dir)
	if err != nil { t.Fatal(err) }
	resolved, err := filepath.Abs(abs)
	if err != nil { t.Fatal(err) }
	relative, err := filepath.Rel(root, resolved)
	if err != nil { t.Fatal(err) }
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) { t.Fatalf("attachment escaped upload root: %s", resolved) }
	if _, err := os.Stat(resolved); err != nil { t.Fatal(err) }
}

func TestSaveUploadKeepsOrdinaryAttachmentReadable(t *testing.T) {
	dir := t.TempDir()
	s := &TicketService{uploadDir: dir}
	want := []byte("normal attachment")
	rel, err := s.SaveUpload("report.txt", want)
	if err != nil { t.Fatal(err) }
	got, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil { t.Fatal(err) }
	if string(got) != string(want) { t.Fatalf("saved content = %q, want %q", got, want) }
}
