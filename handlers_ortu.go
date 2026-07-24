package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ParentReportData struct {
	ParentDashboardData
	Month       string
	PrevMonth   string
	NextMonth   string
	HasData     bool
	Summary     []AspectSummary
	ChartLabels template.JS // JSON
	ChartSeries template.JS // JSON
}

func handleParentDashboard(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	user, _ := getUserByID(uid)

	// Find student linked to this parent
	var studentID int
	var studentName string
	var studentAge int
	var studentGrade string
	err := db.QueryRow("SELECT id, name, age, grade FROM students WHERE parent_id=?", uid).Scan(&studentID, &studentName, &studentAge, &studentGrade)
	if err != nil {
		data := ParentDashboardData{
			UserName:  user.DisplayName,
			Page:      "Beranda",
			CSRFToken: getCSRFToken(getSessionCookie(r)),
		}
		tmpl := getParentLayout("templates/ortu/dashboard.html")
		if tmpl == nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base-parent", struct {
			ParentDashboardData
			NoStudent bool
		}{data, true})
		return
	}

	// Get student's recent meetings (all statuses)
	var meetings []Meeting
	rows, _ := db.Query(`SELECT id, student_id, '', date, start_time, end_time, status
		FROM meetings WHERE student_id=?
		ORDER BY date DESC, start_time DESC
		LIMIT 15`, studentID)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
			m.FormattedDate = formatTanggalIndo(m.Date)
			m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))
			meetings = append(meetings, m)
		}
		rows.Close()
	}
	if meetings == nil {
		meetings = []Meeting{}
	}

	data := ParentDashboardData{
		UserName:     user.DisplayName,
		StudentName:  studentName,
		StudentAge:   studentAge,
		StudentGrade: studentGrade,
		Meetings:     meetings,
		Page:         "Beranda",
		CSRFToken:    getCSRFToken(getSessionCookie(r)),
	}
	tmpl := getParentLayout("templates/ortu/dashboard.html")
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base-parent", data)
}

func handleParentReport(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	user, _ := getUserByID(uid)

	// Find student
	var studentID int
	var studentName string
	var studentAge int
	var studentGrade string
	err := db.QueryRow("SELECT id, name, age, grade FROM students WHERE parent_id=?", uid).Scan(&studentID, &studentName, &studentAge, &studentGrade)
	if err != nil {
		http.Redirect(w, r, "/parent", http.StatusSeeOther)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	t, _ := time.Parse("2006-01", month)
	prev := t.AddDate(0, -1, 0).Format("2006-01")
	next := t.AddDate(0, 1, 0).Format("2006-01")

	// Meetings in this month
	var meetings []Meeting
	rows, _ := db.Query(`SELECT id, student_id, '', date, start_time, end_time, status
		FROM meetings WHERE student_id=? AND strftime('%Y-%m', date)=? AND status='selesai'
		ORDER BY date`, studentID, month)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
			m.FormattedDate = formatTanggalIndo(m.Date)
			m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))
			meetings = append(meetings, m)
		}
		rows.Close()
	}
	if meetings == nil {
		meetings = []Meeting{}
	}

	// Build assessment summary
	summaryMap := map[string]*AspectSummary{}
	for _, a := range aspects {
		summaryMap[a.Key] = &AspectSummary{Key: a.Key, Label: a.Label}
	}
	for _, m := range meetings {
		arows, _ := db.Query("SELECT aspect, score FROM assessments WHERE meeting_id=?", m.ID)
		if arows != nil {
			for arows.Next() {
				var k, v string
				arows.Scan(&k, &v)
				if sm, ok := summaryMap[k]; ok {
					switch v {
					case "Belum Berkembang":
						sm.BB++
					case "Mulai Berkembang":
						sm.MB++
					case "Berkembang Sesuai Harapan":
						sm.BSH++
					case "Berkembang Sangat Baik":
						sm.BSB++
					}
					sm.Total++
				}
			}
			arows.Close()
		}
	}

	var summary []AspectSummary
	var chartLabels []string
	var chartData []int
	for _, a := range aspects {
		if sm, ok := summaryMap[a.Key]; ok && sm.Total > 0 {
			sm.BSBPct = float64(sm.BSB) / float64(sm.Total) * 100
			summary = append(summary, *sm)
			chartLabels = append(chartLabels, sm.Label)
			chartData = append(chartData, sm.BSB)
		}
	}
	if summary == nil {
		summary = []AspectSummary{}
	}
	labelsJSON, _ := json.Marshal(chartLabels)
	dataJSON, _ := json.Marshal(chartData)

	data := ParentReportData{
		ParentDashboardData: ParentDashboardData{
			UserName:     user.DisplayName,
			StudentName:  studentName,
			StudentAge:   studentAge,
			StudentGrade: studentGrade,
			Meetings:     meetings,
			Page:         "Laporan",
			CSRFToken:    getCSRFToken(getSessionCookie(r)),
		},
		Month:       month,
		PrevMonth:   prev,
		NextMonth:   next,
		HasData:     len(meetings) > 0,
		Summary:     summary,
		ChartLabels: template.JS(labelsJSON),
		ChartSeries: template.JS(dataJSON),
	}

	tmpl := getParentLayout("templates/ortu/report.html")
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base-parent", data)
}

func handleParentReportPDF(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	var studentID int
	db.QueryRow("SELECT id FROM students WHERE parent_id=?", uid).Scan(&studentID)
	if studentID == 0 {
		http.Error(w, "No student linked", http.StatusNotFound)
		return
	}
	// Rewrite query params for student ID
	q := r.URL.Query()
	q.Set("id", fmt.Sprintf("%d", studentID))
	r.URL.RawQuery = q.Encode()
	handleReportPDF(w, r)
}

func handleParentReportWord(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	var studentID int
	db.QueryRow("SELECT id FROM students WHERE parent_id=?", uid).Scan(&studentID)
	if studentID == 0 {
		http.Error(w, "No student linked", http.StatusNotFound)
		return
	}
	q := r.URL.Query()
	q.Set("id", fmt.Sprintf("%d", studentID))
	r.URL.RawQuery = q.Encode()
	handleReportWord(w, r)
}

type ParentMeetingDetailData struct {
	ParentDashboardData
	Meeting Meeting
	Aspects []struct {
		Key      string
		Label    string
		Kegiatan string
		Score    string
	}
}

func handleParentMeetingDetail(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	user, _ := getUserByID(uid)
	idStr := r.URL.Query().Get("id")
	meetingID, _ := strconv.Atoi(idStr)

	// Fetch meeting & verify that this meeting belongs to parent's student
	var m Meeting
	err := db.QueryRow(`
		SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.status
		FROM meetings m
		JOIN students s ON m.student_id = s.id
		WHERE m.id=? AND s.parent_id=?`, meetingID, uid).
		Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)

	if err != nil {
		http.Redirect(w, r, "/parent", http.StatusSeeOther)
		return
	}

	m.FormattedDate = formatTanggalIndo(m.Date)
	m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))

	// Get assessments (6 aspects)
	type aspectItem struct {
		Key      string
		Label    string
		Kegiatan string
		Score    string
	}
	var aspectsList []aspectItem

	for _, a := range aspects {
		var kegiatan, score string
		db.QueryRow("SELECT COALESCE(kegiatan,''), COALESCE(score,'') FROM assessments WHERE meeting_id=? AND aspect=?", m.ID, a.Key).
			Scan(&kegiatan, &score)
		aspectsList = append(aspectsList, aspectItem{
			Key:      a.Key,
			Label:    a.Label,
			Kegiatan: kegiatan,
			Score:    score,
		})
	}

	data := struct {
		ParentDashboardData
		Meeting Meeting
		Aspects []aspectItem
	}{
		ParentDashboardData: ParentDashboardData{
			UserName:  user.DisplayName,
			Page:      "Beranda",
			CSRFToken: getCSRFToken(getSessionCookie(r)),
		},
		Meeting: m,
		Aspects: aspectsList,
	}

	tmpl := getParentLayout("templates/ortu/meeting_detail.html")
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base-parent", data)
}

func handleParentProfile(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	user, _ := getUserByID(uid)

	if r.Method == http.MethodPost {
		r.ParseForm()
		action := r.FormValue("action")

		if action == "name" {
			display := r.FormValue("display")
			if len(display) < 1 || len(display) > 50 {
				http.Redirect(w, r, "/parent/profile?err=nama", http.StatusSeeOther)
				return
			}
			db.Exec("UPDATE users SET display_name=? WHERE id=?", display, uid)
			log.Printf("[AUDIT] parent profile update name uid=%d", uid)
			http.Redirect(w, r, "/parent/profile?ok=1", http.StatusSeeOther)
			return
		}

		if action == "password" {
			oldPass := r.FormValue("old_password")
			newPass := r.FormValue("new_password")
			confirm := r.FormValue("confirm_password")

			if len(newPass) < 6 || len(newPass) > 72 {
				http.Redirect(w, r, "/parent/profile?err=sandi", http.StatusSeeOther)
				return
			}
			if newPass != confirm {
				http.Redirect(w, r, "/parent/profile?err=konfirmasi", http.StatusSeeOther)
				return
			}

			var hash string
			db.QueryRow("SELECT password_hash FROM users WHERE id=?", uid).Scan(&hash)
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPass)); err != nil {
				http.Redirect(w, r, "/parent/profile?err=old", http.StatusSeeOther)
				return
			}

			newHash, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			db.Exec("UPDATE users SET password_hash=? WHERE id=?", string(newHash), uid)
			log.Printf("[AUDIT] parent profile update password uid=%d", uid)
			http.Redirect(w, r, "/parent/profile?ok=1", http.StatusSeeOther)
			return
		}
	}

	tmpl := getParentLayout("templates/ortu/profile.html")
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	errMsg := ""
	success := ""
	if r.URL.Query().Get("ok") == "1" {
		success = "Profil berhasil diperbarui"
	}
	if r.URL.Query().Get("err") == "nama" {
		errMsg = "Nama tampilan harus 1-50 karakter"
	} else if r.URL.Query().Get("err") == "sandi" {
		errMsg = "Kata sandi baru harus 6-72 karakter"
	} else if r.URL.Query().Get("err") == "konfirmasi" {
		errMsg = "Konfirmasi sandi tidak cocok"
	} else if r.URL.Query().Get("err") == "old" {
		errMsg = "Kata sandi lama salah"
	}

	tmpl.ExecuteTemplate(w, "base-parent", struct {
		ParentDashboardData
		User    User
		Success string
		Error   string
	}{
		ParentDashboardData: ParentDashboardData{
			UserName:  user.DisplayName,
			Page:      "profile",
			CSRFToken: getCSRFToken(getSessionCookie(r)),
		},
		User:    user,
		Success: success,
		Error:   errMsg,
	})
}
