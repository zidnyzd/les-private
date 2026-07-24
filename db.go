package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)
var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./les.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		fmt.Println("Error opening db:", err)
		return
	}

	// Users (admin, guru & ortu)
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'ortu' CHECK(role IN ('admin','guru','ortu')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Students
	db.Exec(`CREATE TABLE IF NOT EXISTS students (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		age INTEGER,
		grade TEXT DEFAULT '',
		address TEXT DEFAULT '',
		parent_id INTEGER REFERENCES users(id),
		teacher_id INTEGER REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_students_parent ON students(parent_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_students_teacher ON students(teacher_id)")

	// Meetings (pertemuan)
	db.Exec(`CREATE TABLE IF NOT EXISTS meetings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id INTEGER NOT NULL REFERENCES students(id),
		teacher_id INTEGER REFERENCES users(id),
		date DATE NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'terjadwal' CHECK(status IN ('terjadwal','selesai','batal')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_meetings_student ON meetings(student_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_meetings_teacher ON meetings(teacher_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_meetings_date ON meetings(date)")

	// Assessments (penilaian 6 aspek per pertemuan)
	db.Exec(`CREATE TABLE IF NOT EXISTS assessments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		meeting_id INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
		aspect TEXT NOT NULL,
		score TEXT NOT NULL DEFAULT '' CHECK(score IN ('','Belum Berkembang','Mulai Berkembang','Berkembang Sesuai Harapan','Berkembang Sangat Baik','masih berkembang','berkembang dengan baik')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		kegiatan TEXT DEFAULT '',
		UNIQUE(meeting_id, aspect)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_assessments_meeting ON assessments(meeting_id)")

	// Sessions
	db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		role TEXT NOT NULL DEFAULT 'ortu',
		csrf_token TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)")

	// Migrations
	db.Exec("ALTER TABLE assessments ADD COLUMN kegiatan TEXT DEFAULT ''")
	db.Exec("ALTER TABLE students ADD COLUMN teacher_id INTEGER REFERENCES users(id)")
	db.Exec("ALTER TABLE meetings ADD COLUMN teacher_id INTEGER REFERENCES users(id)")

	// Auto-assign existing data without teacher_id to primary teacher/admin
	db.Exec(`UPDATE students SET teacher_id = COALESCE((SELECT id FROM users WHERE role IN ('guru','admin') LIMIT 1), 0) WHERE teacher_id IS NULL OR teacher_id = 0`)
	db.Exec(`UPDATE meetings SET teacher_id = COALESCE((SELECT id FROM users WHERE role IN ('guru','admin') LIMIT 1), 0) WHERE teacher_id IS NULL OR teacher_id = 0`)

	// Seed admin guru from env vars (idempotent)
	if os.Getenv("ADMIN_USERNAME") != "" {
		seedAdmin(os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD"), os.Getenv("ADMIN_DISPLAY"))
	}
}

// seedAdmin bootstraps a guru account from env vars. Idempotent — skips if username exists.
func seedAdmin(username, password, display string) {
	if username == "" || password == "" {
		return
	}
	var c int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", username).Scan(&c)
	if c > 0 {
		// Update existing seeded account to 'admin' role if it was 'guru'
		db.Exec("UPDATE users SET role='admin' WHERE username=? AND role='guru'", username)
		return
	}
	if display == "" {
		display = username
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	db.Exec("INSERT INTO users (username, display_name, password_hash, role) VALUES (?,?,?,?)",
		username, display, string(hash), "admin")
	log.Println("[AUDIT] admin seeded:", username)
}

// 6 Aspek Penilaian
var aspects = []struct {
	Key   string
	Label string
}{
	{"pra_membaca", "Pra membaca"},
	{"menulis", "Menulis"},
	{"berhitung", "Berhitung"},
	{"sensory_play", "Sensory play"},
	{"kreativitas", "Kreativitas"},
	{"brain_game", "Brain game"},
}

// 4 Skala Penilaian
var scoreLabels = []string{
	"Belum Berkembang",
	"Mulai Berkembang",
	"Berkembang Sesuai Harapan",
	"Berkembang Sangat Baik",
}

// Short labels for display (PDF/table)
var scoreShortLabels = map[string]string{
	"Belum Berkembang":          "BB",
	"Mulai Berkembang":          "MB",
	"Berkembang Sesuai Harapan": "BSH",
	"Berkembang Sangat Baik":    "BSB",
}
