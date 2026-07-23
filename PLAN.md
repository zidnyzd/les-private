# Sistem Jadwal Les Private — Planning Detail

## Tech Stack
- **Backend:** Go 1.25 (net/http, html/template)
- **Database:** SQLite 3 (mattn/go-sqlite3)
- **Frontend:** Bootstrap 5.3, Tabler Icons, ApexCharts (untuk grafik perkembangan)
- **Auth:** bcrypt + session token (HttpOnly, Secure, SameSite)
- **PDF/Word:** `go-pdf` atau `unidoc/unipdf` (PDF), `unioffice` (Word .docx)
- **Deploy:** Cloudflare Tunnel + systemd (sama seperti finance-tracker)
- **CI/CD:** GitHub Actions (build → SCP → restart)

---

## Database Schema

### `users` — akun login (guru & ortu)
| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INTEGER PK | |
| username | TEXT UNIQUE | login ID |
| password_hash | TEXT | bcrypt |
| display_name | TEXT | nama tampilan |
| role | TEXT | `guru` atau `ortu` |
| created_at | DATETIME | |

### `students` — data siswa
| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INTEGER PK | |
| name | TEXT | nama siswa |
| age | INTEGER | usia |
| grade | TEXT | kelas (mis. "TK B", "1 SD") |
| address | TEXT | lokasi rumah |
| parent_id | INTEGER FK → users.id | ortu yang punya akun |
| created_at | DATETIME | |

### `meetings` — pertemuan les
| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INTEGER PK | |
| student_id | INTEGER FK → students.id | |
| date | DATE | tanggal pertemuan |
| start_time | TIME | jam mulai |
| end_time | TIME | jam selesai |
| topic | TEXT | tema belajar (guru isi) |
| notes | TEXT | catatan tambahan (opsional) |
| status | TEXT | `terjadwal` / `selesai` / `batal` |
| created_at | DATETIME | |

### `assessments` — penilaian 6 aspek per pertemuan
| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INTEGER PK | |
| meeting_id | INTEGER FK → meetings.id | |
| aspect | TEXT | nama aspek (6 pilihan) |
| score | TEXT | `masih berkembang` / `berkembang sesuai harapan` / `berkembang dengan baik` |
| created_at | DATETIME | |

**Catatan:** 1 meeting = 6 baris assessment (1 per aspek). Unique constraint: `(meeting_id, aspect)`.

### `sessions` — session login
| Kolom | Tipe | Keterangan |
|-------|------|------------|
| token | TEXT PK | |
| user_id | INTEGER FK → users.id | |
| created_at | DATETIME | |

---

## 6 Aspek (Enum)
Dikode di aplikasi, gak perlu tabel DB:
1. `pra_membaca` — Pra membaca
2. `menulis` — Menulis
3. `berhitung` — Berhitung
4. `sensory_play` — Sensory play
5. `kreativitas` — Kreativitas
6. `brain_game` — Brain game

## 3 Skala Penilaian
- `masih_berkembang` — masih berkembang
- `sesuai_harapan` — berkembang sesuai harapan
- `dengan_baik` — berkembang dengan baik

---

## Endpoint / Routes

### Auth
| Route | Role | Method | Deskripsi |
|-------|------|--------|-----------|
| `/login` | publik | GET/POST | Halaman login |
| `/logout` | login | GET | Logout |
| `/register` | publik | GET/POST | Daftar akun ortu |

### Guru
| Route | Role | Method | Deskripsi |
|-------|------|--------|-----------|
| `/` | guru | GET | Dashboard guru |
| `/students` | guru | GET/POST | Daftar & tambah siswa |
| `/students/new` | guru | GET | Form tambah siswa |
| `/students/{id}` | guru | GET/POST | Detail & edit siswa |
| `/students/{id}/delete` | guru | POST | Hapus siswa |
| `/meetings` | guru | GET | Daftar pertemuan |
| `/meetings/new` | guru | GET/POST | Buat pertemuan |
| `/meetings/{id}` | guru | GET/POST | Detail & isi penilaian |
| `/meetings/{id}/delete` | guru | POST | Hapus pertemuan |
| `/reports` | guru | GET | Rekap perkembangan semua siswa |
| `/reports/{student_id}` | guru | GET | Laporan per siswa |
| `/reports/{student_id}/pdf` | guru | GET | Download PDF bulanan |
| `/reports/{student_id}/docx` | guru | GET | Download Word bulanan |

### Ortu
| Route | Role | Method | Deskripsi |
|-------|------|--------|-----------|
| `/parent` | ortu | GET | Dashboard ortu (progress anak) |
| `/parent/report` | ortu | GET | Laporan bulanan anak |
| `/parent/report/pdf` | ortu | GET | Download PDF |

---

## UI Flow

### Guru
1. **Login** → dashboard
2. **Dashboard** — ringkasan: pertemuan hari ini, jumlah siswa, aspek menonjol
3. **Siswa** — daftar siswa, tambah/edit/hapus
4. **Pertemuan** — buat pertemuan baru (pilih siswa + tanggal + jam)
5. **Detail pertemuan** — isi tema belajar + nilai 6 aspek
6. **Laporan** — pilih siswa + bulan → lihat rekap + download PDF/Word

### Ortu
1. **Login** → dashboard ortu
2. **Dashboard** — lihat progress anak (grafik 6 aspek, histori pertemuan)
3. **Laporan** — lihat laporan bulanan + download PDF

---

## Fitur Laporan

### Konten Laporan Bulanan
- Header: nama les, nama siswa, usia, kelas, bulan
- Tabel pertemuan: tanggal, tema belajar
- Tabel penilaian: per pertemuan × 6 aspek
- Rekap aspek menonjol: aspek mana yang paling sering "berkembang dengan baik"
- Grafik perkembangan (opsional di web view)
- Footer: tanda tangan guru

### Format
- **PDF** — pake library `go-pdf` atau `unipdf`
- **Word (.docx)** — pake `unioffice`

---

## Security (sama seperti finance-tracker)
- bcrypt password hashing
- Session token random 16-byte, HttpOnly + Secure + SameSite cookies
- CSRF token di semua form POST
- Rate limiting (login 5/min, register 3/hour)
- Security headers (HSTS, CSP, X-Frame-Options, dll)
- Input validation
- Audit logging
- Role-based middleware (guru/ortu)

---

## Development Phases

### Phase 1: Foundation (MVP)
- [ ] DB schema & migrations
- [ ] Auth (login/register/logout, session)
- [ ] Role-based middleware (guru/ortu)
- [ ] Base template (layout, sidebar, navbar)
- [ ] Dashboard guru (kosong)

### Phase 2: Manajemen Siswa
- [ ] CRUD siswa
- [ ] Form tambah/edit siswa
- [ ] Daftar siswa dengan search

### Phase 3: Manajemen Pertemuan
- [ ] Buat pertemuan (pilih siswa, tanggal, jam)
- [ ] List pertemuan (filter by siswa/tanggal)
- [ ] Edit/hapus pertemuan
- [ ] Status pertemuan (terjadwal/selesai/batal)

### Phase 4: Penilaian
- [ ] Form penilaian 6 aspek di detail pertemuan
- [ ] Skala 3 level (radio button atau dropdown)
- [ ] Edit penilaian
- [ ] Validasi: 1 aspek per pertemuan

### Phase 5: Laporan
- [ ] Rekap aspek menonjol per siswa
- [ ] Grafik perkembangan (ApexCharts)
- [ ] Laporan bulanan web view
- [ ] Export PDF
- [ ] Export Word

### Phase 6: Dashboard Ortu
- [ ] Dashboard ortu (progress anak)
- [ ] Histori pertemuan anak
- [ ] Laporan bulanan (view + PDF)
- [ ] Link ortu ke siswa saat register

### Phase 7: Polish
- [ ] Dark mode
- [ ] Mobile responsive
- [ ] Landing page
- [ ] Security hardening
- [ ] Deploy & CI/CD

---

## Struktur Project

```
les-private/
├── main.go              # Entry point, routes, middleware
├── db.go                # DB init, migrations, helpers
├── models.go            # Struct definitions
├── auth.go              # Login, register, session
├── handlers_guru.go     # Handler guru (siswa, pertemuan, penilaian)
├── handlers_ortu.go     # Handler ortu
├── reports.go           # Laporan & export PDF/Word
├── go.mod / go.sum
├── .github/workflows/
│   └── build-deploy.yml
├── templates/
│   ├── base.html
│   ├── login.html
│   ├── register.html
│   ├── guru/
│   │   ├── dashboard.html
│   │   ├── students.html
│   │   ├── student_form.html
│   │   ├── student_detail.html
│   │   ├── meetings.html
│   │   ├── meeting_form.html
│   │   ├── meeting_detail.html  (isi penilaian)
│   │   └── report.html
│   ├── ortu/
│   │   ├── dashboard.html
│   │   └── report.html
│   └── landing.html
└── les.db               # SQLite (gitignored)
```

## Estimasi Waktu
- Phase 1-3 (MVP): ~3-5 hari kerja
- Phase 4-5 (Penilaian + Laporan): ~2-3 hari
- Phase 6-7 (Ortu + Polish): ~2-3 hari
- **Total: ~7-11 hari kerja**

---

## Status
- ✅ Requirements confirmed
- ✅ Planning detail ready
- ⏳ Menunggu konfirmasi untuk mulai development
