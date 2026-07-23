# Changelog

## Versi 1.1.0 (Peningkatan & PWA - Juli 2026)
- **Persistensi Sesi:** Token sesi dan CSRF token kini disimpan di SQLite, menyelesaikan masalah pengguna ter-logout otomatis setiap kali dideploy/restart.
- **Perbaikan Penilaian:** Mengizinkan Guru menyimpan catatan **kegiatan** walau **nilai/score** belum diisi (skema database `assessments` dimigrasikan untuk membolehkan `score` kosong).
- **Responsive Mobile:** Mendesain ulang halaman Siswa, Pertemuan, dan Laporan menjadi format list-card interaktif pada layar mobile (tabel desktop tetap dipertahankan).
- **Lokalisasi Waktu:** Memperbaiki kebocoran format tanggal sistem (`2026-07-20T00:00:00Z`) menjadi format tanggal & waktu Indonesia yang rapi (e.g. `Senin, 20 Juli 2026` dan jam `08.00 - 09.00 WIB`).
- **PWA (Progressive Web App):** Menambahkan dukungan PWA dengan Service Worker, `manifest.json`, cache-busting ikon, dan ikon gambar buku kustom serta memperbaiki peringatan deprecation Chromium.
- **Optimalisasi Dark Mode:** Memperbaiki keterbacaan teks kontras rendah di mode gelap (seperti teks nama wali dan teks sel matriks penilaian).

## Versi 1.0.0 (Awal Pengembangan - Juli 2026)
- **Init:** Proyek dicanangkan melalui `PLAN.md`.
- **Database:** Membangun `users`, `students`, `meetings`, `assessments`, dan `sessions` dengan perlindungan relasi *Constraint*.
- **Keamanan:** Menerapkan Middleware autentikasi tingkat tinggi (`guru` vs `ortu`).
- **Antarmuka:** Menyelesaikan desain *Bootstrap 5* untuk Panel Guru (Manajemen Siswa, Jadwal, Penilaian 6 Aspek).
- **Portal Ortu:** Menyediakan layar khusus untuk pantauan wali murid.
- **Generator Laporan:** Fungsi ekstraksi data ke format `.pdf` dan `.docx` telah dimasukkan ke dalam modul `reports.go`.

*Status saat ini: Aplikasi berjalan stabil pada port `8082`.*
