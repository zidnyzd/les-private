package main

type User struct {
	ID          int
	Username    string
	DisplayName string
	Role        string // "guru" atau "ortu"
}

type Student struct {
	ID         int
	Name       string
	Age        int
	Grade      string
	Address    string
	ParentID   int
	ParentName string
}

type Meeting struct {
	ID            int
	StudentID     int
	StudentName   string
	Date          string
	StartTime     string
	EndTime       string
	Topic         string
	Notes         string
	Status        string // terjadwal, selesai, batal
	FormattedDate string
	FormattedTime string
}

type Assessment struct {
	ID         int
	MeetingID  int
	Aspect     string
	AspectLabel string
	Score      string
}

// AssessmentMatrix — per pertemuan: 6 aspek + skor
type AssessmentMatrix struct {
	MeetingID int
	Date      string
	Topic     string
	Items    []AssessmentItem
}

type AssessmentItem struct {
	Aspect      string
	AspectLabel string
	Kegiatan    string
	Score       string // kosong = belum dinilai
}

// LoginData untuk template login
type LoginData struct {
	Error string
}

// RegisterData untuk template register
type RegisterData struct {
	Error string
}

// DashboardData untuk dashboard guru
type DashboardData struct {
	UserName        string
	TodayMeetings   []Meeting
	TotalStudents   int
	TotalParents    int
	TotalMeetings   int
	UpcomingCount   int
	Page            string
	CSRFToken       string
}

// ParentDashboardData untuk dashboard ortu
type ParentDashboardData struct {
	UserName     string
	StudentName  string
	StudentAge   int
	StudentGrade string
	Meetings     []Meeting
	Page         string
	CSRFToken    string
}
