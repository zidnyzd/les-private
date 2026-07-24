package main

import (
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var tmplCache = map[string]*template.Template{}

func getTemplate(path string) *template.Template {
	if t, ok := tmplCache[path]; ok {
		return t
	}
	t := template.Must(template.ParseFiles(path))
	tmplCache[path] = t
	return t
}

func getLayout(path string) *template.Template {
	t, err := template.ParseFiles("templates/base.html", path)
	if err != nil {
		log.Println("template error:", err)
		return nil
	}
	return t
}

func getParentLayout(path string) *template.Template {
	t, err := template.ParseFiles("templates/base_parent.html", path)
	if err != nil {
		log.Println("template error:", err)
		return nil
	}
	return t
}

func main() {
	initDB()
	loadSessions()

	// Background: cleanup expired sessions every 30 min
	go func() {
		for range time.Tick(30 * time.Minute) {
			cleanupSessions()
		}
	}()

	// Public routes & Static File Server with Cache-Control
	fs := http.FileServer(http.Dir("static"))
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.StripPrefix("/static/", fs).ServeHTTP(w, r)
	})
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		http.ServeFile(w, r, "static/sw.js")
	})
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/logout", handleLogout)

	// Guru routes
	http.HandleFunc("/", authMiddleware("guru", handleDashboard))
	http.HandleFunc("/students", authMiddleware("guru", handleStudentList))
	http.HandleFunc("/students/new", authMiddleware("guru", handleStudentNew))
	http.HandleFunc("/students/edit", authMiddleware("guru", handleStudentEdit))
	http.HandleFunc("/students/delete", authMiddleware("guru", handleStudentDelete))
	http.HandleFunc("/parents", authMiddleware("guru", handleParentList))
	http.HandleFunc("/parents/reset-password", authMiddleware("guru", handleParentResetPassword))
	http.HandleFunc("/profile", authMiddleware("guru", handleProfile))
	http.HandleFunc("/meetings", authMiddleware("guru", handleMeetingList))
	http.HandleFunc("/meetings/new", authMiddleware("guru", handleMeetingNew))
	http.HandleFunc("/meetings/edit", authMiddleware("guru", handleMeetingEdit))
	http.HandleFunc("/meetings/delete", authMiddleware("guru", handleMeetingDelete))
	http.HandleFunc("/meetings/detail", authMiddleware("guru", handleMeetingDetail))
	http.HandleFunc("/reports", authMiddleware("guru", handleReportList))
	http.HandleFunc("/reports/student", authMiddleware("guru", handleReportStudent))
	http.HandleFunc("/reports/pdf", authMiddleware("guru", handleReportPDF))
	http.HandleFunc("/reports/docx", authMiddleware("guru", handleReportWord))

	// Ortu routes
	http.HandleFunc("/parent", authMiddleware("ortu", handleParentDashboard))
	http.HandleFunc("/parent/meeting/detail", authMiddleware("ortu", handleParentMeetingDetail))
	http.HandleFunc("/parent/report", authMiddleware("ortu", handleParentReport))
	http.HandleFunc("/parent/report/pdf", authMiddleware("ortu", handleParentReportPDF))
	http.HandleFunc("/parent/report/docx", authMiddleware("ortu", handleParentReportWord))
	http.HandleFunc("/parent/profile", authMiddleware("ortu", handleParentProfile))

	// Landing page
	landingTmpl := template.Must(template.ParseFiles("templates/landing.html"))
	http.HandleFunc("/landing", func(w http.ResponseWriter, r *http.Request) {
		landingTmpl.Execute(w, nil)
	})

	// Wrap with security headers + host-based routing
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)

		// Landing page for root domain
		if r.Host == "zira.web.id" && r.URL.Path == "/" {
			landingTmpl.Execute(w, nil)
			return
		}

		http.DefaultServeMux.ServeHTTP(w, r)
	})

	log.Println("Les Private server starting on 127.0.0.1:8082")
	log.Fatal(http.ListenAndServe("127.0.0.1:8082", wrapped))
}

func setSecurityHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")
	h.Set("X-DNS-Prefetch-Control", "off")
	h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://apis.google.com https://static.cloudflareinsights.com; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com https://cdn.jsdelivr.net; img-src 'self' data: https:; connect-src 'self' https://cdn.jsdelivr.net; object-src 'none'; base-uri 'self'")
}

func authMiddleware(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		uid, userRole, ok := getSessionUser(c.Value)
		if !ok {
			// fallback to DB
			var dbUID int
			var role, csrfToken string
			err := db.QueryRow("SELECT user_id, role, csrf_token FROM sessions WHERE token=?", c.Value).Scan(&dbUID, &role, &csrfToken)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			sessionsMu.Lock()
			sessions[c.Value] = sessionInfo{
				userID:    dbUID,
				role:      role,
				csrfToken: csrfToken,
				createdAt: time.Now(),
			}
			sessionsMu.Unlock()
			uid = dbUID
			userRole = role
		}

		if userRole != role {
			// Redirect to correct dashboard
			if userRole == "guru" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/parent", http.StatusSeeOther)
			}
			return
		}

		// Pass user ID & role via headers for handlers
		r.Header.Set("X-User-Id", hex.EncodeToString([]byte{byte(uid >> 24), byte(uid >> 16), byte(uid >> 8), byte(uid)}))
		r.Header.Set("X-User-Role", userRole)

		// CSRF check for POST
		if r.Method == http.MethodPost {
			token := r.FormValue("_csrf")
			expected := getCSRFToken(c.Value)
			if expected == "" || token != expected {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		next(w, r)
	}
}

func cleanupSessions() {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	now := time.Now()
	for tok, s := range sessions {
		if now.Sub(s.createdAt) > sessionMaxAge {
			delete(sessions, tok)
			db.Exec("DELETE FROM sessions WHERE token=?", tok)
		}
	}
}

// Placeholder handlers — will be expanded in later phases
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	user, _ := getUserByID(uid)

	// Today's meetings
	today := time.Now().Format("2006-01-02")
	var todayMeetings []Meeting
	rows, _ := db.Query(`
		SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.status
		FROM meetings m
		JOIN students s ON m.student_id = s.id
		WHERE m.date = ?
		ORDER BY m.start_time`, today)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
			m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))
			todayMeetings = append(todayMeetings, m)
		}
		rows.Close()
	}

	// Counts
	var totalStudents, totalParents, totalMeetings, upcomingCount int
	db.QueryRow("SELECT COUNT(*) FROM students").Scan(&totalStudents)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role='ortu'").Scan(&totalParents)
	db.QueryRow("SELECT COUNT(*) FROM meetings WHERE status='terjadwal'").Scan(&upcomingCount)
	db.QueryRow("SELECT COUNT(*) FROM meetings").Scan(&totalMeetings)

	data := DashboardData{
		UserName:      user.DisplayName,
		TodayMeetings: todayMeetings,
		TotalStudents: totalStudents,
		TotalParents:  totalParents,
		TotalMeetings: totalMeetings,
		UpcomingCount: upcomingCount,
		Page:          "dashboard",
		CSRFToken:     getCSRFToken(getSessionCookie(r)),
	}

	tmpl := getLayout("templates/guru/dashboard.html")
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

// Helpers
func getUserIDFromHeader(r *http.Request) int {
	h := r.Header.Get("X-User-Id")
	if h == "" {
		return 0
	}
	b, err := hex.DecodeString(h)
	if err != nil || len(b) < 4 {
		return 0
	}
	return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
}

func getSessionCookie(r *http.Request) string {
	c, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}

// keep strings import used
var _ = strings.Contains
