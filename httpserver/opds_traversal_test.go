package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func withTempUploadDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	prev := uploadDir
	uploadDir = filepath.Join(root, "uploads")
	if err := os.MkdirAll(filepath.Join(uploadDir, "config"), 0o755); err != nil {
		t.Fatalf("create upload dir: %v", err)
	}
	t.Cleanup(func() { uploadDir = prev })
	return root
}

func createBooksDB(t *testing.T, key, format string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(uploadDir, "config", "books.db"))
	if err != nil {
		t.Fatalf("open books.db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE books (
		key TEXT PRIMARY KEY,
		name TEXT,
		author TEXT,
		description TEXT,
		cover TEXT,
		format TEXT,
		publisher TEXT,
		size INTEGER,
		page INTEGER
	)`)
	if err != nil {
		t.Fatalf("create books table: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO books (key, name, author, description, cover, format, publisher, size, page)
		 VALUES (?, ?, '', '', '', ?, '', 1, 1)`,
		key, "Traversal", format,
	)
	if err != nil {
		t.Fatalf("insert book: %v", err)
	}
}

func TestOPDSDownloadRejectsTraversalKey(t *testing.T) {
	root := withTempUploadDir(t)
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("outside upload dir"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	createBooksDB(t, "../../secret", "txt")

	rec := httptest.NewRecorder()
	handleOPDSDownload(rec, httptest.NewRequest(http.MethodGet, "/opds/download/x", nil), "../../secret")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestOPDSCoverRejectsTraversalKey(t *testing.T) {
	root := withTempUploadDir(t)
	secret := filepath.Join(root, "secret.jpg")
	if err := os.WriteFile(secret, []byte("outside cover"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	createBooksDB(t, "../../secret", "txt")

	rec := httptest.NewRecorder()
	handleOPDSCover(rec, httptest.NewRequest(http.MethodGet, "/opds/cover/x", nil), "../../secret")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestOPDSDownloadServesBookInsideUploadDir(t *testing.T) {
	withTempUploadDir(t)
	if err := os.MkdirAll(filepath.Join(uploadDir, "book"), 0o755); err != nil {
		t.Fatalf("create book dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, "book", "book1.txt"), []byte("inside upload dir"), 0o644); err != nil {
		t.Fatalf("write book file: %v", err)
	}
	createBooksDB(t, "book1", "txt")

	rec := httptest.NewRecorder()
	handleOPDSDownload(rec, httptest.NewRequest(http.MethodGet, "/opds/download/book1", nil), "book1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "inside upload dir" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "inside upload dir")
	}
}

func TestOPDSCoverServesCoverInsideUploadDir(t *testing.T) {
	withTempUploadDir(t)
	if err := os.MkdirAll(filepath.Join(uploadDir, "cover"), 0o755); err != nil {
		t.Fatalf("create cover dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, "cover", "book1.jpg"), []byte("inside cover"), 0o644); err != nil {
		t.Fatalf("write cover file: %v", err)
	}
	createBooksDB(t, "book1", "txt")

	rec := httptest.NewRecorder()
	handleOPDSCover(rec, httptest.NewRequest(http.MethodGet, "/opds/cover/book1", nil), "book1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "inside cover" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "inside cover")
	}
}
