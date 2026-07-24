package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

func handleReportPDF(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("id")
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	var s Student
	err := db.QueryRow("SELECT id, name, age, grade, address FROM students WHERE id=?", studentID).
		Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Load meetings
	var meetings []Meeting
	rows, _ := db.Query(`SELECT id, student_id, '', date, start_time, end_time, status
		FROM meetings WHERE student_id=? AND strftime('%Y-%m', date)=? AND status='selesai'
		ORDER BY date`, studentID, month)
	if rows != nil {
		for rows.Next() {
			var m Meeting
			rows.Scan(&m.ID, &m.StudentID, &m.StudentName, &m.Date, &m.StartTime, &m.EndTime, &m.Status)
			meetings = append(meetings, m)
		}
		rows.Close()
	}

	// Load assessments
	aspectKeys := []string{"pra_membaca", "menulis", "berhitung", "sensory_play", "kreativitas", "brain_game"}
	aspectLabels := []string{"Pra membaca", "Menulis", "Berhitung", "Sensory play", "Kreativitas", "Brain game"}
	assessments := map[int]map[string][]map[string]string{} // meeting_id -> aspect -> [{kegiatan, score}]
	summaryMap := map[string]*AspectSummary{}
	kegiatanLists := map[string][]string{}
	for _, ak := range aspectKeys {
		summaryMap[ak] = &AspectSummary{Key: ak}
		kegiatanLists[ak] = []string{}
	}

	meetingIDs := []int{}
	for _, m := range meetings {
		meetingIDs = append(meetingIDs, m.ID)
	}

	if len(meetingIDs) > 0 {
		for _, mid := range meetingIDs {
			arows, _ := db.Query("SELECT aspect, score, COALESCE(kegiatan,'') FROM assessments WHERE meeting_id=?", mid)
			if arows != nil {
				for arows.Next() {
					var k, v, kg string
					arows.Scan(&k, &v, &kg)
					if assessments[mid] == nil {
						assessments[mid] = map[string][]map[string]string{}
					}
					assessments[mid][k] = append(assessments[mid][k], map[string]string{
						"kegiatan": kg, "score": v,
					})
					
					cleanedKeg := strings.TrimSpace(kg)
					if cleanedKeg != "" && cleanedKeg != "-" {
						kegiatanLists[k] = append(kegiatanLists[k], cleanedKeg)
					}

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
	}

	// Generate PDF (Portrait A4)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, "LAPORAN PENILAIAN PERKEMBANGAN", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Student info
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("Nama: %s", s.Name), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, "Program: Private Tutoring", "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("Bulan: %s", month), "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// Per meeting tables
	for _, m := range meetings {
		// Meeting title
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 8, fmt.Sprintf("Pertemuan (%s)", formatTanggalIndo(m.Date)), "", 1, "C", false, 0, "")
		pdf.Ln(2)

		// Table header: Aspek, Kegiatan, BB, MB, BSH, BSB
		colW := []float64{28, 68, 14, 14, 14, 14} // Total ~152mm
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(180, 198, 231) // B4C6E7 - same as Word
		headers := []string{"Aspek", "Kegiatan", "BB", "MB", "BSH", "BSB"}
		for i, h := range headers {
			pdf.CellFormat(colW[i], 8, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		// Data rows
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetFillColor(255, 255, 255)
		for i, ak := range aspectKeys {
			rawEntries := assessments[m.ID][ak]
			var validEntries []map[string]string
			for _, e := range rawEntries {
				if strings.TrimSpace(e["kegiatan"]) != "" || e["score"] != "" {
					validEntries = append(validEntries, e)
				}
			}
			if len(validEntries) == 0 {
				continue // Skip rendering empty aspects completely
			}
			for j, entry := range validEntries {
				// Aspek (only on first row of aspect)
				if j == 0 {
					pdf.CellFormat(colW[0], 7, aspectLabels[i], "1", 0, "L", false, 0, "")
				} else {
					pdf.CellFormat(colW[0], 7, "", "1", 0, "L", false, 0, "")
				}

				// Kegiatan
				kg := strings.TrimSpace(entry["kegiatan"])
				if kg == "" {
					kg = "-"
				}
				pdf.CellFormat(colW[1], 7, truncateStr(kg, 40), "1", 0, "L", false, 0, "")

				// BB, MB, BSH, BSB
				score := entry["score"]
				checkCol := -1
				switch score {
				case "Belum Berkembang":
					checkCol = 2
				case "Mulai Berkembang":
					checkCol = 3
				case "Berkembang Sesuai Harapan":
					checkCol = 4
				case "Berkembang Sangat Baik":
					checkCol = 5
				}
				for col := 2; col <= 5; col++ {
					if col == checkCol {
						pdf.SetFont("ZapfDingbats", "", 10)
						pdf.CellFormat(colW[col], 7, "3", "1", 0, "C", false, 0, "")
						pdf.SetFont("Helvetica", "", 8)
					} else {
						pdf.CellFormat(colW[col], 7, "", "1", 0, "C", false, 0, "")
					}
				}
				pdf.Ln(-1)
			}
		}
		pdf.Ln(6)
	}

	// Keterangan Stimulasi (Tabel 2 kolom)
	keterangan := []struct{ label, desc string }{
		{"Membaca", "Kegiatan pengenalan huruf atau belajar teknis membaca (suku kata, kata, frasa, kalimat)"},
		{"Berhitung", "Kegiatan pengenalan angka, konsep bilangan, konsep dasar matematika"},
		{"Menulis", "Kegiatan menguatkan otot jari tangan (motoric halus) menggunakan alat tulis"},
		{"Brain Exercise", "Kegiatan untuk menstimulasi kemampuan kognitif dan bahasa"},
		{"Sensory play", "Kegiatan untuk menstimulasi koordinasi mata dengan tangan, panca indera"},
		{"Kreativitas", "Kegiatan untuk mengembangkan imajinasi dan keterampilan seni"},
	}

	skala := []struct{ num, label, desc string }{
		{"1", "Belum Berkembang (BB)", "Belum ingin berkegiatan"},
		{"2", "Mulai Berkembang (MB)", "Mulai ingin berkegiatan tetapi perlu stimulasi lebih lanjut"},
		{"3", "Berkembang Sesuai Harapan (BSH)", "Mampu berkegiatan dan terstimulasi dengan baik"},
		{"4", "Berkembang Sangat Baik (BSB)", "Mahir berkegiatan dan terstimulasi dengan sangat baik"},
	}

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 8, "Keterangan Stimulasi", "", 1, "C", false, 0, "")
	pdf.Ln(2)

	colWidthKet := []float64{35, 145}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(180, 198, 231) // B4C6E7
	pdf.CellFormat(colWidthKet[0], 8, "Aspek", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidthKet[1], 8, "Kegiatan / Keterangan", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, k := range keterangan {
		x := pdf.GetX()
		y := pdf.GetY()
		
		// Set font to bold for aspect label column
		pdf.SetFont("Helvetica", "B", 9)
		// We use MultiCell for description, so we measure its height first or wrap inside border
		pdf.SetXY(x + colWidthKet[0], y)
		pdf.MultiCell(colWidthKet[1], 5, k.desc, "1", "L", false)
		
		y2 := pdf.GetY()
		height := y2 - y
		
		pdf.SetXY(x, y)
		pdf.CellFormat(colWidthKet[0], height, k.label, "1", 0, "L", false, 0, "")
		
		pdf.SetXY(x, y2) // restore position to bottom of multicell row
	}
	pdf.Ln(6)

	// Keterangan Skala (Tabel 3 kolom)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 8, "Keterangan Skala Penilaian", "", 1, "C", false, 0, "")
	pdf.Ln(2)

	colWidthSkala := []float64{15, 60, 105}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(180, 198, 231)
	pdf.CellFormat(colWidthSkala[0], 8, "Skala", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidthSkala[1], 8, "Predikat", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidthSkala[2], 8, "Keterangan", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, sl := range skala {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(colWidthSkala[0], 7, sl.num, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidthSkala[1], 7, sl.label, "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(colWidthSkala[2], 7, sl.desc, "1", 1, "L", false, 0, "")
	}
	pdf.Ln(6)

	// Kesimpulan Stimulasi (Tabel 2 kolom)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 8, "Kesimpulan Stimulasi", "", 1, "C", false, 0, "")
	pdf.Ln(2)

	colWidthKes := []float64{10, 170}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(180, 198, 231)
	pdf.CellFormat(colWidthKes[0], 8, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidthKes[1], 8, "Kesimpulan Perkembangan", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	kIdx := 1
	for idx, ak := range aspectKeys {
		sm := summaryMap[ak]
		if sm == nil || sm.Total == 0 {
			continue
		}
		levels := []struct {
			label string
			count int
		}{
			{"Belum Berkembang", sm.BB},
			{"Mulai Berkembang", sm.MB},
			{"Berkembang Sesuai Harapan", sm.BSH},
			{"Berkembang Sangat Baik", sm.BSB},
		}
		dominant := levels[0]
		for _, l := range levels {
			if l.count > dominant.count {
				dominant = l
			}
		}

		// Formulate rich narrative based on entries
		kegs := kegiatanLists[ak]
		var kegNarrative string
		if len(kegs) > 1 {
			kegNarrative = strings.Join(kegs[:len(kegs)-1], ", ") + ", dan " + kegs[len(kegs)-1]
		} else if len(kegs) == 1 {
			kegNarrative = kegs[0]
		} else {
			kegNarrative = "aktivitas stimulasi harian"
		}

		statusText := dominant.label
		var progNarrative string
		if statusText == "Belum Berkembang" {
			progNarrative = "belum ingin berkegiatan dan masih memerlukan bimbingan serta stimulasi yang lebih intensif."
		} else if statusText == "Mulai Berkembang" {
			progNarrative = "mulai menunjukkan ketertarikan untuk berkegiatan, namun masih memerlukan bantuan dan stimulasi lebih lanjut."
		} else if statusText == "Berkembang Sesuai Harapan" {
			progNarrative = "mampu mengikuti kegiatan dengan baik, aktif berpartisipasi, dan terstimulasi sesuai target perkembangan."
		} else { // Berkembang Sangat Baik
			progNarrative = "sangat mahir dalam berkegiatan, menunjukkan antusiasme yang tinggi, dan terstimulasi dengan sangat maksimal."
		}

		finalText := fmt.Sprintf("Dalam kegiatan %s, %s telah belajar dan mempraktikkan: %s. Secara keseluruhan, %s %s Maka kemampuan %s dalam kegiatan %s dinyatakan %s.",
			aspectLabels[idx], s.Name, kegNarrative, s.Name, progNarrative, s.Name, aspectLabels[idx], statusText)

		x := pdf.GetX()
		y := pdf.GetY()
		
		pdf.SetXY(x + colWidthKes[0], y)
		pdf.MultiCell(colWidthKes[1], 5, finalText, "1", "L", false)
		
		y2 := pdf.GetY()
		height := y2 - y
		
		pdf.SetXY(x, y)
		pdf.CellFormat(colWidthKes[0], height, fmt.Sprintf("%d", kIdx), "1", 0, "C", false, 0, "")
		
		pdf.SetXY(x, y2)
		kIdx++
	}

	// Tanda tangan
	pdf.Ln(10)
	today := time.Now()
	bulanID := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	dateStr := fmt.Sprintf("Nganjuk, %d %s %d", today.Day(), bulanID[today.Month()-1], today.Year())
	pdf.CellFormat(0, 7, dateStr, "", 1, "L", false, 0, "")
	pdf.Ln(15)

	// Signature columns
	var guruName string
	db.QueryRow(`
		SELECT u.display_name FROM students s 
		JOIN users u ON s.teacher_id = u.id 
		WHERE s.id = ?`, studentID).Scan(&guruName)
	if guruName == "" {
		db.QueryRow("SELECT display_name FROM users WHERE role IN ('guru','admin') LIMIT 1").Scan(&guruName)
	}
	if guruName == "" {
		guruName = "Guru Pengajar"
	}

	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(85, 7, "Pengajar", "", 0, "C", false, 0, "")
	pdf.CellFormat(85, 7, "Founder", "", 1, "C", false, 0, "")
	pdf.Ln(20)
	pdf.CellFormat(85, 7, guruName, "", 0, "C", false, 0, "")
	pdf.CellFormat(85, 7, "Admin", "", 1, "C", false, 0, "")

	// Output
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		log.Printf("[AUDIT] pdf generation fail err=%v", err)
		http.Error(w, "PDF generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=laporan_%s_%s.pdf", s.Name, month))
	w.Write(buf.Bytes())
	log.Printf("[AUDIT] report pdf student=%s month=%s", s.Name, month)
}

func handleReportWord(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("id")
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	// Verify student exists
	var s Student
	err := db.QueryRow("SELECT id, name, age, grade, address FROM students WHERE id=?", studentID).
		Scan(&s.ID, &s.Name, &s.Age, &s.Grade, &s.Address)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Get teacher/guru name
	var guruName string
	db.QueryRow(`
		SELECT u.display_name FROM students s 
		JOIN users u ON s.teacher_id = u.id 
		WHERE s.id = ?`, studentID).Scan(&guruName)
	if guruName == "" {
		db.QueryRow("SELECT display_name FROM users WHERE role IN ('guru','admin') LIMIT 1").Scan(&guruName)
	}
	if guruName == "" {
		guruName = "Guru Pengajar"
	}

	// Generate .docx via Python script
	scriptPath := "scripts/generate_docx.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "/root/les-private/scripts/generate_docx.py"
	}
	outPath := fmt.Sprintf("/tmp/laporan_%s_%s.docx", studentID, month)

	cmd := exec.Command("python3", scriptPath, studentID, month, outPath, guruName)
	cmd.Dir = "/root/les-private"
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[AUDIT] docx generation fail student=%s month=%s err=%v output=%s", studentID, month, err, string(output))
		http.Error(w, "Gagal membuat laporan Word", http.StatusInternalServerError)
		return
	}

	// Read and send the file
	data, err := os.ReadFile(outPath)
	if err != nil {
		log.Printf("[AUDIT] docx read fail err=%v", err)
		http.Error(w, "Gagal membaca file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=laporan_%s_%s.docx", s.Name, month))
	w.Write(data)
	log.Printf("[AUDIT] report docx student=%s month=%s size=%d", s.Name, month, len(data))
}

func abbreviateScore(s string) string {
	switch s {
	case "Belum Berkembang":
		return "BB"
	case "Mulai Berkembang":
		return "MB"
	case "Berkembang Sesuai Harapan":
		return "BSH"
	case "Berkembang Sangat Baik":
		return "BSB"
	}
	return "-"
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max] + ".."
	}
	return s
}

func sum(a []float64) float64 {
	var s float64
	for _, v := range a {
		s += v
	}
	return s
}
