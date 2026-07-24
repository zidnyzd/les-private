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
	Parents    []User
	Teachers   []User
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

	if user.Role == "guru" {
		if search != "" {
			rows, err = db.Query(`
				SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,'')
				FROM students s LEFT JOIN users u ON s.parent_id = u.id
				WHERE (s.name LIKE ? OR s.grade LIKE ?) AND (s.teacher_id = ? OR s.teacher_id IS NULL OR s.teacher_id = 0) ORDER BY s.name`, "%"+search+"%", "%"+search+"%", user.ID)
		} else {
			rows, err = db.Query(`
				SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,'')
				FROM students s LEFT JOIN users u ON s.parent_id = u.id
				WHERE s.teacher_id = ? OR s.teacher_id IS NULL OR s.teacher_id = 0 ORDER BY s.name`, user.ID)
		}
	} else {
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
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "students",
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

		if parentMode == "none" {
			parentID = 0
		} else if parentMode == "new" {
			// Guru buat akun ortu baru
			pUser := r.FormValue("parent_new_username")
			pDisplay := r.FormValue("parent_new_display")
			pPass := r.FormValue("parent_new_password")

			if pUser == "" && pPass == "" {
				parentID = 0
			} else {
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
			}
		} else if parentMode == "existing" {
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

		teacherID, _ := strconv.Atoi(r.FormValue("teacher_id"))
		if teacherID == 0 {
			teacherID = user.ID
		}

		_, err := db.Exec("INSERT INTO students (name, age, grade, address, parent_id, teacher_id) VALUES (?,?,?,?,?,?)",
			strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), parentID, teacherID)
		if err != nil {
			log.Printf("[AUDIT] student create fail err=%v", err)
			renderStudentForm(w, r, user, Student{}, "Gagal menyimpan data")
			return
		}
		log.Printf("[AUDIT] student create success name=%q parent_id=%d teacher_id=%d", name, parentID, teacherID)
		http.Redirect(w, r, "/students?added=1", http.StatusSeeOther)
		return
	}
	renderStudentForm(w, r, user, Student{}, "")
}

func renderStudentForm(w http.ResponseWriter, r *http.Request, user User, s Student, errMsg string) {
	var parents []User
	rows, err := db.Query("SELECT id, username, display_name, role FROM users WHERE role='ortu' ORDER BY display_name ASC")
	if err == nil && rows != nil {
		for rows.Next() {
			var p User
			rows.Scan(&p.ID, &p.Username, &p.DisplayName, &p.Role)
			parents = append(parents, p)
		}
		rows.Close()
	}
	if parents == nil {
		parents = []User{}
	}

	var teachers []User
	trows, err := db.Query("SELECT id, username, display_name, role FROM users WHERE role IN ('guru','admin') ORDER BY display_name ASC")
	if err == nil && trows != nil {
		for trows.Next() {
			var t User
			trows.Scan(&t.ID, &t.Username, &t.DisplayName, &t.Role)
			teachers = append(teachers, t)
		}
		trows.Close()
	}
	if teachers == nil {
		teachers = []User{}
	}

	if s.TeacherID == 0 {
		s.TeacherID = user.ID
	}

	data := StudentPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "students",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Student:  s,
		Parents:  parents,
		Teachers: teachers,
		EditMode: s.ID > 0,
		Error:    errMsg,
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
		if parentMode == "none" {
			parentID = 0
		} else if parentMode == "new" {
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

		teacherID, _ := strconv.Atoi(r.FormValue("teacher_id"))
		if teacherID == 0 {
			teacherID = user.ID
		}

		if parentMode == "none" {
			db.Exec("UPDATE students SET name=?, age=?, grade=?, address=?, parent_id=NULL, teacher_id=? WHERE id=?",
				strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), teacherID, id)
		} else if parentID > 0 {
			db.Exec("UPDATE students SET name=?, age=?, grade=?, address=?, parent_id=?, teacher_id=? WHERE id=?",
				strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), parentID, teacherID, id)
		} else {
			db.Exec("UPDATE students SET name=?, age=?, grade=?, address=?, teacher_id=? WHERE id=?",
				strings.TrimSpace(name), age, strings.TrimSpace(grade), strings.TrimSpace(address), teacherID, id)
		}
		log.Printf("[AUDIT] student update id=%s parent_id=%d teacher_id=%d", id, parentID, teacherID)
		http.Redirect(w, r, "/students?updated=1", http.StatusSeeOther)
		return
	}

	var s Student
	err := db.QueryRow(`SELECT s.id, s.name, s.age, s.grade, s.address, COALESCE(s.parent_id,0), COALESCE(u.display_name,''), COALESCE(u.username,''), COALESCE(s.teacher_id,0)
		FROM students s LEFT JOIN users u ON s.parent_id = u.id WHERE s.id=?`, id).
		Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address, &s.ParentID, &s.ParentName, &s.ParentUsername, &s.TeacherID)
	if err != nil {
		http.Redirect(w, r, "/students", http.StatusSeeOther)
		return
	}

	renderStudentForm(w, r, user, s, "")
}

type ParentListItem struct {
	User
	ChildrenNames string
	ChildrenCount int
}

type ParentPageData struct {
	DashboardData
	Parents  []ParentListItem
	Search   string
	Success  string
	Error    string
}

func handleParentList(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	search := r.URL.Query().Get("q")
	success := r.URL.Query().Get("success")
	errMsg := r.URL.Query().Get("error")

	if success == "reset" {
		success = "Password orang tua berhasil di-reset!"
	}

	var parents []ParentListItem
	var rows *sql.Rows
	var err error

	if search != "" {
		rows, err = db.Query(`
			SELECT u.id, u.username, u.display_name, u.role,
			       COALESCE(GROUP_CONCAT(s.name, ', '),''),
			       COUNT(s.id)
			FROM users u
			LEFT JOIN students s ON u.id = s.parent_id
			WHERE u.role = 'ortu' AND (u.username LIKE ? OR u.display_name LIKE ? OR s.name LIKE ?)
			GROUP BY u.id
			ORDER BY u.display_name ASC`, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	} else {
		rows, err = db.Query(`
			SELECT u.id, u.username, u.display_name, u.role,
			       COALESCE(GROUP_CONCAT(s.name, ', '),''),
			       COUNT(s.id)
			FROM users u
			LEFT JOIN students s ON u.id = s.parent_id
			WHERE u.role = 'ortu'
			GROUP BY u.id
			ORDER BY u.display_name ASC`)
	}

	if err == nil && rows != nil {
		for rows.Next() {
			var p ParentListItem
			rows.Scan(&p.ID, &p.Username, &p.DisplayName, &p.Role, &p.ChildrenNames, &p.ChildrenCount)
			parents = append(parents, p)
		}
		rows.Close()
	}
	if parents == nil {
		parents = []ParentListItem{}
	}

	data := ParentPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "parents",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Parents: parents,
		Search:  search,
		Success: success,
		Error:   errMsg,
	}
	execGuruTemplate(w, "templates/guru/parents.html", data)
}

func handleParentResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/parents", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	parentID := r.FormValue("parent_id")
	newPassword := r.FormValue("new_password")

	if parentID == "" || len(newPassword) < 6 {
		http.Redirect(w, r, "/parents?error=Password+minimal+6+karakter", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Redirect(w, r, "/parents?error=Gagal+reset+password", http.StatusSeeOther)
		return
	}

	_, err = db.Exec("UPDATE users SET password_hash=? WHERE id=? AND role='ortu'", string(hash), parentID)
	if err != nil {
		http.Redirect(w, r, "/parents?error=Gagal+update+database", http.StatusSeeOther)
		return
	}

	log.Printf("[AUDIT] parent password reset parent_id=%s by_guru", parentID)
	http.Redirect(w, r, "/parents?success=reset", http.StatusSeeOther)
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
	q := `SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id`
	var args []interface{}
	var conditions []string

	if user.Role == "guru" {
		conditions = append(conditions, "(m.teacher_id=? OR s.teacher_id=? OR m.teacher_id IS NULL OR m.teacher_id=0)")
		args = append(args, user.ID, user.ID)
	}
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

	// All students for filter dropdown
	students := getAllStudents(user)

	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "meetings",
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
	students := getAllStudents(user)
	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "meetings",
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
		status := r.FormValue("status")

		db.Exec("UPDATE meetings SET student_id=?, date=?, start_time=?, end_time=?, status=? WHERE id=?",
			studentID, date, startTime, endTime, status, id)
		log.Printf("[AUDIT] meeting update id=%s", id)
		http.Redirect(w, r, "/meetings?updated=1", http.StatusSeeOther)
		return
	}

	var m Meeting
	err := db.QueryRow(`SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id WHERE m.id=?`, id).
		Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
	if err != nil {
		http.Redirect(w, r, "/meetings", http.StatusSeeOther)
		return
	}
	students := getAllStudents(user)
	data := MeetingPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "meetings",
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
func getAllStudents(user User) []Student {
	var students []Student
	var rows *sql.Rows
	var err error
	if user.Role == "guru" {
		rows, err = db.Query("SELECT id, name FROM students WHERE teacher_id = ? OR teacher_id IS NULL OR teacher_id = 0 ORDER BY name", user.ID)
	} else {
		rows, err = db.Query("SELECT id, name FROM students ORDER BY name")
	}
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

type AssessmentSubItem struct {
	ID       int
	Kegiatan string
	Score    string
	Index    int
}

type AssessmentFormItem struct {
	Key   string
	Label string
	Items []AssessmentSubItem
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
	err := db.QueryRow(`SELECT m.id, m.student_id, s.name, m.date, m.start_time, m.end_time, m.status
		FROM meetings m JOIN students s ON m.student_id = s.id WHERE m.id=?`, id).
		Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
	if err != nil {
		http.Redirect(w, r, "/meetings", http.StatusSeeOther)
		return
	}
	m.FormattedDate = formatTanggalIndo(m.Date)
	m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))

	if r.Method == http.MethodPost {
		r.ParseForm()
		status := r.FormValue("status")
		if _, err := db.Exec("UPDATE meetings SET status=? WHERE id=?", status, id); err != nil {
			log.Printf("[ERROR] update meeting id=%s: %v", id, err)
		}

		// Clear existing assessments for this meeting before saving multi-row items
		db.Exec("DELETE FROM assessments WHERE meeting_id=?", id)

		// Save assessments (supporting multi-row per aspect)
		for _, a := range aspects {
			kegs := r.Form["kegiatan_" + a.Key]
			for idx, kg := range kegs {
				score := r.FormValue(fmt.Sprintf("aspect_%s_%d", a.Key, idx))
				kgTrim := strings.TrimSpace(kg)
				if kgTrim != "" || score != "" {
					if _, err := db.Exec("INSERT INTO assessments (meeting_id, aspect, score, kegiatan, sort_order) VALUES (?,?,?,?,?)",
						id, a.Key, score, kgTrim, idx); err != nil {
						log.Printf("[ERROR] save assessment meeting=%s aspect=%s idx=%d: %v", id, a.Key, idx, err)
					}
				}
			}
		}
		log.Printf("[AUDIT] assessment save meeting=%s", id)
		http.Redirect(w, r, "/meetings/detail?id="+id+"&saved=1", http.StatusSeeOther)
		return
	}

	// Load existing assessments grouping by aspect
	itemMap := map[string][]AssessmentSubItem{}
	rows, err := db.Query("SELECT id, aspect, score, COALESCE(kegiatan,'') FROM assessments WHERE meeting_id=? ORDER BY sort_order ASC, id ASC", id)
	if err == nil && rows != nil {
		for rows.Next() {
			var sub AssessmentSubItem
			var aspect string
			rows.Scan(&sub.ID, &aspect, &sub.Score, &sub.Kegiatan)
			itemMap[aspect] = append(itemMap[aspect], sub)
		}
		rows.Close()
	}

	var items []AssessmentFormItem
	for _, a := range aspects {
		subs := itemMap[a.Key]
		if len(subs) == 0 {
			subs = []AssessmentSubItem{{Index: 0}}
		} else {
			for i := range subs {
				subs[i].Index = i
			}
		}
		items = append(items, AssessmentFormItem{
			Key:   a.Key,
			Label: a.Label,
			Items: subs,
		})
	}

	data := MeetingDetailPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "meetings",
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
	var rows *sql.Rows
	if user.Role == "guru" {
		rows, _ = db.Query(`
			SELECT s.id, s.name, s.age,
				(SELECT COUNT(*) FROM meetings WHERE student_id=s.id) as total
			FROM students s WHERE s.teacher_id = ? OR s.teacher_id IS NULL OR s.teacher_id = 0 ORDER BY s.name`, user.ID)
	} else {
		rows, _ = db.Query(`
			SELECT s.id, s.name, s.age,
				(SELECT COUNT(*) FROM meetings WHERE student_id=s.id) as total
			FROM students s ORDER BY s.name`)
	}
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
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "reports",
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
	rows, _ := db.Query(`SELECT id, student_id, '', date, start_time, end_time, status
		FROM meetings WHERE student_id=? AND strftime('%Y-%m', date)=?
		ORDER BY date, start_time`, studentID, month)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
			if m.Status == "selesai" {
				m.FormattedDate = formatTanggalIndo(m.Date)
				m.FormattedTime = fmt.Sprintf("%s - %s WIB", formatWaktuIndo(m.StartTime), formatWaktuIndo(m.EndTime))
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
		matrix = append(matrix, AssessmentMatrix{MeetingID: m.ID, Date: formatTanggalIndo(m.Date), Time: m.FormattedTime, Items: items})
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
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "reports",
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

// ====== TEACHER MANAGEMENT HANDLERS (ADMIN ONLY) ======

type TeacherItem struct {
	ID            int
	Username      string
	DisplayName   string
	TotalStudents int
	CreatedAt     string
}

type TeacherPageData struct {
	DashboardData
	Teachers []TeacherItem
	Teacher  TeacherItem
	EditMode bool
	Success  string
	Error    string
}

func handleTeacherList(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)

	var teachers []TeacherItem
	rows, err := db.Query(`
		SELECT u.id, u.username, u.display_name, strftime('%d-%m-%Y', u.created_at),
			(SELECT COUNT(*) FROM students WHERE teacher_id = u.id) as total
		FROM users u WHERE u.role = 'guru' ORDER BY u.display_name ASC`)
	if err == nil && rows != nil {
		for rows.Next() {
			var t TeacherItem
			rows.Scan(&t.ID, &t.Username, &t.DisplayName, &t.CreatedAt, &t.TotalStudents)
			teachers = append(teachers, t)
		}
		rows.Close()
	}
	if teachers == nil {
		teachers = []TeacherItem{}
	}

	data := TeacherPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "teachers",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Teachers: teachers,
		Success:  r.URL.Query().Get("success"),
	}
	execGuruTemplate(w, "templates/guru/teachers.html", data)
}

func handleTeacherNew(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)

	if r.Method == http.MethodPost {
		r.ParseForm()
		tUser := strings.TrimSpace(r.FormValue("username"))
		tDisplay := strings.TrimSpace(r.FormValue("display_name"))
		tPass := r.FormValue("password")

		if len(tUser) < 3 || len(tUser) > 20 {
			renderTeacherForm(w, r, user, TeacherItem{}, "Username guru harus 3-20 karakter")
			return
		}
		if len(tPass) < 6 || len(tPass) > 72 {
			renderTeacherForm(w, r, user, TeacherItem{}, "Password guru harus 6-72 karakter")
			return
		}
		if tDisplay == "" {
			tDisplay = tUser
		}

		var exists int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", tUser).Scan(&exists)
		if exists > 0 {
			renderTeacherForm(w, r, user, TeacherItem{}, "Username '"+tUser+"' sudah digunakan")
			return
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte(tPass), bcrypt.DefaultCost)
		_, err := db.Exec("INSERT INTO users (username, display_name, password_hash, role) VALUES (?,?,?,?)",
			tUser, tDisplay, string(hash), "guru")
		if err != nil {
			log.Printf("[AUDIT] teacher create fail err=%v", err)
			renderTeacherForm(w, r, user, TeacherItem{}, "Gagal membuat akun guru")
			return
		}

		log.Printf("[AUDIT] teacher create success user=%q by_admin=%s", tUser, user.Username)
		http.Redirect(w, r, "/teachers?success=Akun+guru+berhasil+dibuat", http.StatusSeeOther)
		return
	}

	renderTeacherForm(w, r, user, TeacherItem{}, "")
}

func renderTeacherForm(w http.ResponseWriter, r *http.Request, user User, t TeacherItem, errMsg string) {
	data := TeacherPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "teachers",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Teacher: t,
		Error:   errMsg,
	}
	execGuruTemplate(w, "templates/guru/teacher_form.html", data)
}

func handleTeacherEdit(w http.ResponseWriter, r *http.Request) {
	_, user := getUser(r)
	id := r.URL.Query().Get("id")

	if r.Method == http.MethodPost {
		r.ParseForm()
		tDisplay := strings.TrimSpace(r.FormValue("display_name"))
		newPass := r.FormValue("password")

		if tDisplay == "" {
			renderTeacherForm(w, r, user, TeacherItem{ID: parseID(id)}, "Nama tampilan harus diisi")
			return
		}

		db.Exec("UPDATE users SET display_name=? WHERE id=? AND role='guru'", tDisplay, id)

		if newPass != "" {
			if len(newPass) < 6 || len(newPass) > 72 {
				renderTeacherForm(w, r, user, TeacherItem{ID: parseID(id), DisplayName: tDisplay}, "Password baru harus 6-72 karakter")
				return
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			db.Exec("UPDATE users SET password_hash=? WHERE id=? AND role='guru'", string(hash), id)
		}

		log.Printf("[AUDIT] teacher update id=%s by_admin=%s", id, user.Username)
		http.Redirect(w, r, "/teachers?success=Akun+guru+berhasil+diubah", http.StatusSeeOther)
		return
	}

	var t TeacherItem
	err := db.QueryRow("SELECT id, username, display_name FROM users WHERE id=? AND role='guru'", id).
		Scan(&t.ID, &t.Username, &t.DisplayName)
	if err != nil {
		http.Redirect(w, r, "/teachers", http.StatusSeeOther)
		return
	}

	data := TeacherPageData{
		DashboardData: DashboardData{UserName: user.DisplayName, UserRole: user.Role, Page: "teachers",
			CSRFToken: getCSRFToken(getSessionCookie(r))},
		Teacher:  t,
		EditMode: true,
	}
	execGuruTemplate(w, "templates/guru/teacher_form.html", data)
}

func handleTeacherDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		db.Exec("DELETE FROM users WHERE id=? AND role='guru'", id)
		log.Printf("[AUDIT] teacher delete id=%s", id)
	}
	http.Redirect(w, r, "/teachers?success=Akun+guru+berhasil+dihapus", http.StatusSeeOther)
}

func parseID(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
