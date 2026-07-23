# Changelog

## Versi 1.0.0 (Awal Pengembangan - Juli 2026)
- **Init:** Proyek dicanangkan melalui `PLAN.md`.
- **Database:** Membangun `users`, `students`, `meetings`, `assessments`, dan `sessions` dengan perlindungan relasi *Constraint*.
- **Keamanan:** Menerapkan Middleware autentikasi tingkat tinggi (`guru` vs `ortu`).
- **Antarmuka:** Menyelesaikan desain *Bootstrap 5* untuk Panel Guru (Manajemen Siswa, Jadwal, Penilaian 6 Aspek).
- **Portal Ortu:** Menyediakan layar khusus untuk pantauan wali murid.
- **Generator Laporan:** Fungsi ekstraksi data ke format `.pdf` dan `.docx` telah dimasukkan ke dalam modul `reports.go`.

*Status saat ini: Aplikasi berjalan stabil pada port `8082`.*
