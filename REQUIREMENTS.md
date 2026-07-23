# Sistem Jadwal Les Private — Requirements

## Sumber
Dari user (911), 17 Juli 2026

## Inti Sistem
Jadwal les private — guru les ngelola jadwal & penilaian siswa.

## Alur Utama
1. Guru buat pertemuan manual (tanggal, jam, siswa)
2. Di tiap pertemuan, guru isi:
   - **Tema belajar** (bebas, guru yang nentuin)
   - **Penilaian 6 aspek** (skala 3 level, tidak disingkat):
     - `masih berkembang`
     - `berkembang sesuai harapan`
     - `berkembang dengan baik`
3. Sistem catat & bisa lihat aspek mana yang paling menonjol per siswa

## 6 Aspek Penilaian
1. Pra membaca
2. Menulis
3. Berhitung
4. Sensory play
5. Kreativitas
6. Brain game

## User & Akses
- **1 guru** aja (single-teacher)
- Akses: **guru + ortu** (dua-duanya bisa login)
  - Guru: kelola semua (siswa, jadwal, penilaian)
  - Ortu: lihat progress anak mereka saja

## Platform
- Web aja

## Data Disimpan
- **Data guru** — nama, akun login
- **Data siswa** — nama, usia, kelas, lokasi rumah

## Jadwal
- **Manual** — guru bikin pertemuan manual tiap kali

## Notifikasi
- ❌ Tidak perlu

## Laporan
- ✅ Perlu — laporan perkembangan **bulanan** per siswa
- Format: **PDF atau Word** (bisa di-download)

## Output Berguna
- Histori pertemuan per siswa
- Rekap aspek mana yang kuat/lemah → gambaran perkembangan anak

## Status
- ✅ Lengkap — siap untuk planning detail
