package main

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestAssessmentSkemaDanSave(t *testing.T) {
	// 1. Setup temporary database
	tempDBPath := "./test_les.db"
	defer os.Remove(tempDBPath)

	testDB, err := sql.Open("sqlite3", tempDBPath)
	if err != nil {
		t.Fatalf("Gagal membuka test db: %v", err)
	}
	defer testDB.Close()

	// 2. Buat tabel meetings dan assessments sesuai skema asli baru
	_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS meetings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id INTEGER NOT NULL,
		date DATE NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'terjadwal'
	)`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel meetings: %v", err)
	}

	_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS assessments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		meeting_id INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
		aspect TEXT NOT NULL,
		score TEXT NOT NULL DEFAULT '' CHECK(score IN ('','Belum Berkembang','Mulai Berkembang','Berkembang Sesuai Harapan','Berkembang Sangat Baik','masih berkembang','berkembang dengan baik')),
		kegiatan TEXT DEFAULT '',
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel assessments: %v", err)
	}

	// Inisialisasi data meeting palsu
	_, err = testDB.Exec(`INSERT INTO meetings (id, student_id, date, start_time, end_time, status) VALUES (1, 99, '2026-07-23', '08:00', '09:00', 'terjadwal')`)
	if err != nil {
		t.Fatalf("Gagal insert dummy meeting: %v", err)
	}

	// 3. Verifikasi simpan multi-row kegiatan per aspek
	_, err = testDB.Exec(`INSERT INTO assessments (meeting_id, aspect, score, kegiatan, sort_order) VALUES (?, ?, ?, ?, ?)`,
		1, "pra_membaca", "Berkembang Sesuai Harapan", "Mengidentifikasi cerita", 0)
	if err != nil {
		t.Errorf("Gagal menyimpan assessment 1: %v", err)
	}

	_, err = testDB.Exec(`INSERT INTO assessments (meeting_id, aspect, score, kegiatan, sort_order) VALUES (?, ?, ?, ?, ?)`,
		1, "pra_membaca", "Mulai Berkembang", "Mengenal vokal", 1)
	if err != nil {
		t.Errorf("Gagal menyimpan assessment 2: %v", err)
	}

	// Hitung jumlah baris multi-row (seharusnya 2 baris)
	var count int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM assessments WHERE meeting_id = 1 AND aspect = 'pra_membaca'`).Scan(&count)
	if err != nil {
		t.Fatalf("Gagal menghitung data assessment: %v", err)
	}
	if count != 2 {
		t.Errorf("Multi-row gagal! Ekspektasi count 2, dapat: %d", count)
	}
}

func TestSessionRestoration(t *testing.T) {
	tempDBPath := "./test_les.db"
	defer os.Remove(tempDBPath)

	testDB, err := sql.Open("sqlite3", tempDBPath)
	if err != nil {
		t.Fatalf("Gagal membuka test db: %v", err)
	}
	defer testDB.Close()

	// Buat tabel sessions sesuai skema baru
	_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL DEFAULT 'ortu',
		csrf_token TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel sessions: %v", err)
	}

	// Simpan data session manual ke DB
	originalToken := "test_token_123"
	originalCSRF := "csrf_val_abc"
	originalRole := "guru"
	originalUID := 42

	_, err = testDB.Exec(`INSERT INTO sessions (token, user_id, role, csrf_token) VALUES (?, ?, ?, ?)`,
		originalToken, originalUID, originalRole, originalCSRF)
	if err != nil {
		t.Fatalf("Gagal insert session ke DB: %v", err)
	}

	// Mock DB global dan global map
	db = testDB
	sessions = make(map[string]sessionInfo)

	// Jalankan loadSessions
	loadSessions()

	// Verifikasi data ter-restore ke in-memory map
	sessionsMu.Lock()
	s, ok := sessions[originalToken]
	sessionsMu.Unlock()

	if !ok {
		t.Fatalf("Session tidak berhasil di-restore ke memori!")
	}
	if s.userID != originalUID {
		t.Errorf("UserID salah: %d", s.userID)
	}
	if s.role != originalRole {
		t.Errorf("Role salah: %q", s.role)
	}
	if s.csrfToken != originalCSRF {
		t.Errorf("CSRF token salah: %q", s.csrfToken)
	}
}
