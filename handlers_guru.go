package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type StudentPageData struct {
	DashboardData
	Students   []Student
	Search     string
	Student    Student
	EditMode   bool
	Success    string
	Error      string
	ParentName string
}

func handleStudentList(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	search := r.URL.Query().Get("q")

	var students []Student
	var rows *sql.Rows
	var err error

	if search != "" {
		rows, err = db.Query(`
			SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,'')
			FROM students s LEFT JOIN users u ON s.parent_id = u.id
			WHERE s.name LIKE ? OR s.grade LIKE ? ORDER BY s.name`, "%"+search+"%", "%"+search+"%")
	} else {
		rows, err = db.Query(`
			SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,'')
			FROM students s LEFT JOIN users u ON s.parent_id = u.id
			ORDER BY s.name`)
	}
	if err == nil && rows != nil {
		for rows.Next() {
			var s Student
			rows.Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address, &s.ParentID, &s.ParentName)
			students = append(students, s)
		}
		rows.Close()
	}
	if students == nil {
		students = []Student{}
	}

	data := StudentPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "students",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Students: students,
		Search:   search,
	}
	execGuruTemplate(w, "templates/guru/students.html", data)
}

func handleStudentNew(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)

	if r.Method == http.MethodPost {
		r.ParseForm()
		name := r.FormValue("name")
		age, _ := strconv.Atoi(r.FormValue("age"))
		grade := r.FormValue("grade")
		address := r.FormValue("address")
		parentMode := r.FormValue("parent_mode")

		if len(name) < 1 || len(name) > 100 {
			renderStudentForm(w, r, user, Student{}, "Nama siswa harus 1-100 karakter")
			return
		}

		var parentID int

		if parentMode == "new" {
			// Guru buat akun ortu baru
			pUser := r.FormValue("parent_new_username")
			pDisplay := r.FormValue("parent_new_display")
			pPass := r.FormValue("parent_new_password")

			if len(pUser) < 3 || len(pUser) > 20 {
				renderStudentForm(w, r, user, Student{}, "Username ortu harus 3-20 karakter")
				return
			}
			if len(pPass) < 6 || len(pPass) > 72 {
				renderStudentForm(w, r, user, Student{}, "Password ortu harus 6-72 karakter")
				return
			}
			if pDisplay == "" {
				pDisplay = pUser
			}

			// Cek username ortu belum dipake
			var exists int
			db.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", pUser).Scan(&exists)
			if exists > 0 {
				renderStudentForm(w, r, user, Student{}, "Username ortu '"+pUser+"' sudah digunakan")
				return
			}

			hash, _ := bcrypt.GenerateFromPassword([]byte(pPass), bcrypt.DefaultCost)
			res, err := db.Exec("INSERT INTO users (username, display_name, password_hash, role) VALUES (?,?,?,?)",
				pUser, pDisplay, string(hash), "ortu")
			if err != nil {
				log.Printf("[AUDIT] parent create fail err=%v", err)
				renderStudentForm(w, r, user, Student{}, "Gagal membuat akun ortu")
				return
			}
			pid, _ := res.LastInsertId()
			parentID = int(pid)
			log.Printf("[AUDIT] parent create success user=%q by_guru=%s", pUser, user.Username)

		} else {
			// Link ke akun ortu yang sudah ada
			parentUser := r.FormValue("parent_username")
			if parentUser != "" {
				db.QueryRow("SELECT id FROM users WHERE username=? AND role='ortu'", parentUser).Scan(&parentID)
				if parentID == 0 {
					renderStudentForm(w, r, user, Student{}, "Username ortu '"+parentUser+"' tidak ditemukan")
					return
				}
			}
		}

		_, err := db.Exec("INSERT INTO students (name, age, grade, address, parent_id) VALUES (?,?,?,?,?)",
			strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), parentID)
		if err != nil {
			log.Printf("[AUDIT] student create fail err=%v", err)
			renderStudentForm(w, r, user, Student{}, "Gagal menyimpan data")
			return
		}
		log.Printf("[AUDIT] student create success name=%q parent_id=%d", name, parentID)
		http.Redirect(w, r, "/students?added=1", http.StatusSeeOther)
		return
	}
	renderStudentForm(w, r, user, Student{}, "")
}

func renderStudentForm(w http.ResponseWriter, r *http.Request, user User, s Student, errMsg string) {
	data := StudentPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "students",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Student: s,
		Error:   errMsg,
	}
	execGuruTemplate(w, "templates/guru/student_form.html", data)
}

func handleStudentEdit(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	id := r.URL.Query().Get("id")

	if r.Method == http.MethodPost {
		r.ParseForm()
		name := r.FormValue("name")
		age, _ := strconv.Atoi(r.FormValue("age"))
		grade := r.FormValue("grade")
		address := r.FormValue("address")
		parentMode := r.FormValue("parent_mode")
		if len(name) < 1 {
			renderStudentForm(w, r, user, Student{}, "Nama harus diisi")
			return
		}

		var parentID int
		if parentMode == "new" {
			pUser := r.FormValue("parent_new_username")
			pDisplay := r.FormValue("parent_new_display")
			pPass := r.FormValue("parent_new_password")
			if len(pUser) < 3 || len(pUser) > 20 {
				renderStudentForm(w, r, user, Student{}, "Username ortu harus 3-20 karakter")
				return
			}
			if len(pPass) < 6 || len(pPass) > 72 {
				renderStudentForm(w, r, user, Student{}, "Password ortu harus 6-72 karakter")
				return
			}
			if pDisplay == "" {
				pDisplay = pUser
			}
			var exists int
			db.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", pUser).Scan(&exists)
			if exists > 0 {
				renderStudentForm(w, r, user, Student{}, "Username ortu '"+pUser+"' sudah digunakan")
				return
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(pPass), bcrypt.DefaultCost)
			res, err := db.Exec("INSERT INTO users (username, display_name, password_hash, role) VALUES (?,?,?,?)",
				pUser, pDisplay, string(hash), "ortu")
			if err != nil {
				renderStudentForm(w, r, user, Student{}, "Gagal membuat akun ortu")
				return
			}
			pid, _ := res.LastInsertId()
			parentID = int(pid)
			log.Printf("[AUDIT] parent create success user=%q by_guru=%s", pUser, user.Username)
		} else if parentMode == "existing" {
			parentUser := r.FormValue("parent_username")
			if parentUser != "" {
				db.QueryRow("SELECT id FROM users WHERE username=? AND role='ortu'", parentUser).Scan(&parentID)
			}
		}

		if parentID > 0 {
			db.Exec("UPDATE students SET name=?, age=?, grade=?, address=?, parent_id=? WHERE id=?",
				strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), parentID, id)
		} else {
			db.Exec("UPDATE students SET name=?, age=?, grade=?, address=? WHERE id=?",
				strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), id)
		}
		log.Printf("[AUDIT] student update id=%s parent_id=%d", id, parentID)
		http.Redirect(w, r, "/students?updated=1", http.StatusSeeOther)
		return
	}

	var s Student
	err := db.QueryRow(`SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,'')
		FROM students s LEFT JOIN users u ON s.parent_id = u.id WHERE s.id=?`, id).
		Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address, &s.ParentID, &s.ParentName)
	if err != nil {
		http.Redirect(w, r, "/students", http.StatusSeeOther)
		return
	}
	data := StudentPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "students",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Student:  s,
		EditMode: true,
	}
	execGuruTemplate(w, "templates/guru/student_form.html", data)
}

func handleStudentDelete(w http.ResponseWriter, r *http.Request) {
	if getUserIDFromHeader(r) == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("id")
	if id != "" {
		db.Exec("DELETE FROM students WHERE id=?", id)
		log.Printf("[AUDIT] student delete id=%s", id)
	}
	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

func getUser(r *http.Request) (int, User) {
	uid := getUserIDFromHeader(r)
	u, _ := getUserByID(uid)
	return uid, u
}

func execGuruTemplate(w http.ResponseWriter, tmplPath string, data interface{}) {
	tmpl := getLayout(tmplPath)
	if tmpl == nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	err := tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("[TEMPLATE ERROR] %s: %v", tmplPath, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// Helper format tanggal Indonesia
func formatTanggalIndo(dateStr string) string {
	if len(dateStr) >= 10 {
		dateStr = dateStr[:10]
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	months := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	days := []string{
		"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu",
	}
	return fmt.Sprintf("%s, %d %s %d", days[t.Weekday()], t.Day(), months[t.Month()-1], t.Year())
}

func formatWaktuIndo(timeStr string) string {
	return strings.ReplaceAll(timeStr, ":", ".")
}

// ====== MEETING HANDLERS ======

type MeetingPageData struct {
	DashboardData
	Meetings      []Meeting
	Students      []Student
	Meeting       Meeting
	EditMode      bool
	Error         string
	Success       string
	FilterStudent string
	FilterDate    string
}

func handleMeetingList(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	fs := r.URL.Query().Get("student")
	fd := r.URL.Query().Get("date")

	// Build query
	q := `SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.topic, m.notes, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id`
	var args []interface{}
	var conditions []string

	if fs != "" {
		conditions = append(conditions, "m.student_id=?")
		args = append(args, fs)
	}
	if fd != "" {
		conditions = append(conditions, "m.date=?")
		args = append(args, fd)
	}
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY m.date DESC, m.start_time DESC"

	var meetings []Meeting
	rows, err := db.Query(q, args...)
	if err == nil && rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Topic, &m.Notes, &m.Status)
			m.FormattedDate = formatTanggalIndo(m.Date)
			m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))
			meetings = append(meetings, m)
		}
		rows.Close()
	}
	if meetings == nil {
		meetings = []Meeting{}
	}

	// All students for filter dropdown
	students := getAllStudents()

	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "meetings",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Meetings:      meetings,
		Students:      students,
		FilterStudent: fs,
		FilterDate:    fd,
	}
	execGuruTemplate(w, "templates/guru/meetings.html", data)
}

func handleMeetingNew(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)

	if r.Method == http.MethodPost {
		r.ParseForm()
		studentID := r.FormValue("student_id")
		date := r.FormValue("date")
		startTime := r.FormValue("start_time")
		endTime := r.FormValue("end_time")

		if studentID == "" || date == "" || startTime == "" || endTime == "" {
			renderMeetingForm(w, r, user, Meeting{}, "Semua field harus diisi")
			return
		}
		_, err := db.Exec("INSERT INTO meetings (student_id, date, start_time, end_time, status) VALUES (?,?,?,?,'terjadwal')",
			studentID, date, startTime, endTime)
		if err != nil {
			log.Printf("[AUDIT] meeting create fail err=%v", err)
			renderMeetingForm(w, r, user, Meeting{}, "Gagal menyimpan pertemuan")
			return
		}
		log.Printf("[AUDIT] meeting create success student=%s date=%s", studentID, date)
		http.Redirect(w, r, "/meetings?added=1", http.StatusSeeOther)
		return
	}

	renderMeetingForm(w, r, user, Meeting{}, "")
}

func renderMeetingForm(w http.ResponseWriter, r *http.Request, user User, m Meeting, errMsg string) {
	students := getAllStudents()
	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "meetings",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Students: students,
		Meeting:  m,
		Error:    errMsg,
	}
	execGuruTemplate(w, "templates/guru/meeting_form.html", data)
}

func handleMeetingEdit(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	id := r.URL.Query().Get("id")

	if r.Method == http.MethodPost {
		r.ParseForm()
		studentID := r.FormValue("student_id")
		date := r.FormValue("date")
		startTime := r.FormValue("start_time")
		endTime := r.FormValue("end_time")
		topic := r.FormValue("topic")
		notes := r.FormValue("notes")
		status := r.FormValue("status")

		db.Exec("UPDATE meetings SET student_id=?, date=?, start_time=?, end_time=?, topic=?, notes=?, status=? WHERE id=?",
			studentID, date, startTime, endTime, topic, notes, status, id)
		log.Printf("[AUDIT] meeting update id=%s", id)
		http.Redirect(w, r, "/meetings?updated=1", http.StatusSeeOther)
		return
	}

	var m Meeting
	err := db.QueryRow(`SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.topic, m.notes, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id WHERE m.id=?`, id).
		Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Topic, &m.Notes, &m.Status)
	if err != nil {
		http.Redirect(w, r, "/meetings", http.StatusSeeOther)
		return
	}
	students := getAllStudents()
	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "meetings",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Students: students,
		Meeting:  m,
		EditMode: true,
	}
	execGuruTemplate(w, "templates/guru/meeting_form.html", data)
}

func handleMeetingDelete(w http.ResponseWriter, r *http.Request) {
	if getUserIDFromHeader(r) == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("id")
	if id != "" {
		db.Exec("DELETE FROM meetings WHERE id=?", id)
		log.Printf("[AUDIT] meeting delete id=%s", id)
	}
	http.Redirect(w, r, "/meetings", http.StatusSeeOther)
}

// Helper: get all students (id + name) for dropdown
func getAllStudents() []Student {
	var students []Student
	rows, err := db.Query("SELECT id, name FROM students ORDER BY name")
	if err == nil && rows != nil {
		for rows.Next() {
			var s Student
			rows.Scan(&s.ID, &s.Name)
			students = append(students, s)
		}
		rows.Close()
	}
	if students == nil {
		students = []Student{}
	}
	return students
}

// ====== ASSESSMENT HANDLERS ======

type AssessmentFormItem struct {
	Key      string
	Label    string
	Kegiatan string
	Score    string
}

type MeetingDetailPageData struct {
	DashboardData
	Meeting   Meeting
	Aspects   []AssessmentFormItem
	ScoreList []string
	Success   string
	Error     string
}

func handleMeetingDetail(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Redirect(w, r, "/meetings", http.StatusSeeOther)
		return
	}

	// Load meeting
	var m Meeting
	err := db.QueryRow(`SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.topic, m.notes, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id WHERE m.id=?`, id).
		Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Topic, &m.Notes, &m.Status)
	if err != nil {
		http.Redirect(w, r, "/meetings", http.StatusSeeOther)
		return
	}
	m.FormattedDate = formatTanggalIndo(m.Date)
	m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))

	if r.Method == http.MethodPost {
		r.ParseForm()
		// Update meeting topic & notes
		topic := r.FormValue("topic")
		notes := r.FormValue("notes")
		status := r.FormValue("status")
		if _, err := db.Exec("UPDATE meetings SET topic=?, notes=?, status=? WHERE id=?", topic, notes, status, id); err != nil {
			log.Printf("[ERROR] update meeting id=%s: %v", id, err)
		}

		// Save assessments (6 aspects)
		for _, a := range aspects {
			score := r.FormValue("aspect_" + a.Key)
			kegiatan := r.FormValue("kegiatan_" + a.Key)
			if score != "" || kegiatan != "" {
				if _, err := db.Exec("INSERT OR REPLACE INTO assessments (meeting_id, aspect, score, kegiatan) VALUES (?,?,?,?)",
					id, a.Key, score, kegiatan); err != nil {
					log.Printf("[ERROR] save assessment meeting=%s aspect=%s: %v", id, a.Key, err)
				}
			}
		}
		log.Printf("[AUDIT] assessment save meeting=%s topic=%q", id, topic)
		http.Redirect(w, r, "/meetings/detail?id="+id+"&saved=1", http.StatusSeeOther)
		return
	}

	// Load existing assessments
	scoreMap := map[string]string{}
	kegMap := map[string]string{}
	rows, _ := db.Query("SELECT aspect, score, COALESCE(kegiatan,'') FROM assessments WHERE meeting_id=?", id)
	if rows != nil {
		for rows.Next() {
			var k, v, kg string
			rows.Scan(&k, &v, &kg)
			scoreMap[k] = v
			kegMap[k] = kg
		}
		rows.Close()
	}

	var items []AssessmentFormItem
	for _, a := range aspects {
		items = append(items, AssessmentFormItem{
			Key:      a.Key,
			Label:    a.Label,
			Kegiatan: kegMap[a.Key],
			Score:    scoreMap[a.Key],
		})
	}

	data := MeetingDetailPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "meetings",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Meeting:   m,
		Aspects:   items,
		ScoreList: scoreLabels,
		Success:   r.URL.Query().Get("saved"),
	}
	execGuruTemplate(w, "templates/guru/meeting_detail.html", data)
}

// ====== REPORT HANDLERS ======

type ReportStudentItem struct {
	ID   int
	Name string
	Age  int
	TotalMeetings int
}



type ReportPageData struct {
	DashboardData
	Students []ReportStudentItem
}

type AspectSummary struct {
	Key        string
	Label      string
	BB         int
	MB         int
	BSH        int
	BSB        int
	Total      int
	BSBPct     float64
}

type StudentReportData struct {
	DashboardData
	Student     Student
	Month       string
	PrevMonth   string
	NextMonth   string
	Meetings    []Meeting
	Matrix      []AssessmentMatrix
	Summary     []AspectSummary
	ChartLabels template.JS // JSON
	ChartSeries template.JS // JSON
	HasData     bool
}

func handleReportList(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)

	var students []ReportStudentItem
	rows, _ := db.Query(`
		SELECT s.id, s.name, s.age,
			(SELECT COUNT(*) FROM meetings WHERE student_id=s.id) as total
		FROM students s ORDER BY s.name`)
	if rows != nil {
		for rows.Next() {
			var s ReportStudentItem
			rows.Scan(&s.ID, &s.Name, &s.Age, &s.TotalMeetings)
			students = append(students, s)
		}
		rows.Close()
	}
	if students == nil {
		students = []ReportStudentItem{}
	}

	data := ReportPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "reports",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Students: students,
	}
	execGuruTemplate(w, "templates/guru/reports.html", data)
}

func handleReportStudent(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	studentID := r.URL.Query().Get("id")
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if studentID == "" {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	// Load student
	var s Student
	err := db.QueryRow("SELECT id, name, age, grade, address FROM students WHERE id=?", studentID).
		Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address)
	if err != nil {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	// Prev/Next month
	t, _ := time.Parse("2006-01", month)
	prev := t.AddDate(0, -1, 0).Format("2006-01")
	next := t.AddDate(0, 1, 0).Format("2006-01")

	// Meetings in this month
	var meetings []Meeting
	rows, _ := db.Query(`SELECT id, student_id, '', date, start_time, end_time, topic, notes, status
		FROM meetings WHERE student_id=? AND strftime('%Y-%m', date)=?
		ORDER BY date`, studentID, month)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Topic, &m.Notes, &m.Status)
			if m.Status == "selesai" {
				meetings = append(meetings, m)
			}
		}
		rows.Close()
	}
	if meetings == nil {
		meetings = []Meeting{}
	}

	// Build assessment matrix per meeting
	var matrix []AssessmentMatrix
	summaryMap := map[string]*AspectSummary{}
	for _, a := range aspects {
		summaryMap[a.Key] = &AspectSummary{Key: a.Key, Label: a.Label}
	}

	for _, m := range meetings {
		var items []AssessmentItem
		arows, _ := db.Query("SELECT aspect, score, COALESCE(kegiatan,'') FROM assessments WHERE meeting_id=?", m.ID)
		if arows != nil {
			for arows.Next() {
				var k, v, kg string
				arows.Scan(&k, &v, &kg)
				var label string
				for _, a := range aspects {
					if a.Key == k {
						label = a.Label
						break
					}
				}
				items = append(items, AssessmentItem{Aspect: k, AspectLabel: label, Kegiatan: kg, Score: v})

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
		matrix = append(matrix, AssessmentMatrix{MeetingID: m.ID, Date: m.Date, Topic: m.Topic, Items: items})
	}
	if matrix == nil {
		matrix = []AssessmentMatrix{}
	}

	// Build summary sorted by aspect order
	var summary []AspectSummary
	for _, a := range aspects {
		if sm, ok := summaryMap[a.Key]; ok {
			if sm.Total > 0 {
				sm.BSBPct = float64(sm.BSB) / float64(sm.Total) * 100
			}
			summary = append(summary, *sm)
		}
	}
	if summary == nil {
		summary = []AspectSummary{}
	}

	// Chart data (JSON)
	var chartLabels []string
	var chartData []int
	for _, sm := range summary {
		chartLabels = append(chartLabels, sm.Label)
		chartData = append(chartData, sm.BSB)
	}
	labelsJSON, _ := json.Marshal(chartLabels)
	dataJSON, _ := json.Marshal(chartData)

	data := StudentReportData{
		DashboardData: DashboardData{UserName: user.DisplayName, Page: "reports",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Student:     s,
		Month:       month,
		PrevMonth:   prev,
		NextMonth:   next,
		Meetings:    meetings,
		Matrix:      matrix,
		Summary:     summary,
		ChartLabels: template.JS(labelsJSON),
		ChartSeries: template.JS(dataJSON),
		HasData:     len(meetings) > 0,
	}
	execGuruTemplate(w, "templates/guru/report_student.html", data)
}
