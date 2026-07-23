# ZiRa Les Private

Sistem manajemen jadwal dan rapor Bimbingan Belajar (Les Private) berbasis web. Dibuat khusus dengan arsitektur **Single-Teacher** yang menghubungkan langsung progres belajar siswa dengan Orang Tua / Wali Murid secara *Real-Time*.

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-Proprietary-red)

---

## 📚 Gambaran Umum Fitur

Aplikasi ini secara terpusat mengatur seluruh pertemuan belajar dan memonitor perkembangan kognitif siswa dalam satu pintu.

### ✅ Panel Khusus Guru (Admin)
- **Manajemen Jadwal:** Buat jadwal, tentukan durasi, lalu tandai saat pertemuan selesai.
- **Rapor 6 Aspek:** Penilaian perkembangan kognitif siswa setelah sesi les berakhir (Pra membaca, Menulis, Berhitung, Sensory play, Kreativitas, Brain game).
- **Generator Dokumen:** Cetak profil atau rapot perkembangan anak dengan format `.PDF` dan `.DOCX` secara otomatis.
- **Sistem Keamanan Tinggi:** Middleware RBAC yang sangat ketat menolak segala akses dari non-guru.

### ✅ Portal Orang Tua (Wali Murid)
- **Akses Read-Only:** Login untuk memantau nilai dan catatan *Tema Belajar* dari Guru khusus untuk anak milik mereka.
- **Unduh Rapor Bebas:** Wali dapat mengekstrak langsung rapor anak di rumah tanpa harus meminta ke guru.

---

## Stack Aplikasi

| Komponen | Alat yang Digunakan |
|---|---|
| **Bahasa Pemrograman** | Go (`net/http`) |
| **Database** | SQLite 3 (WAL Mode) |
| **Render UI** | Server-Side Rendering (SSR) |
| **Frontend Styling** | Bootstrap 5, Poppins Font |

---

## Hak Milik (Proprietary)

**Seluruh Hak Cipta Dilindungi.**

Kode sumber, desain antarmuka, logika basis data, dan entitas dari aplikasi *Les Private* ini adalah hak milik tertutup dan eksklusif. Penyalinan, pendistribusian, modifikasi, atau rekayasa balik (reverse-engineering) sebagian maupun keseluruhan komponen ini **DILARANG KERAS** tanpa izin eksplisit tertulis dari pemilik.

Aplikasi dirancang oleh [ZidStore](https://zidstore.net).
