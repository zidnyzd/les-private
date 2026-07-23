# Development & Gotchas

Panduan internal bagi pengembang (*Developer*) yang ingin memodifikasi kerangka kerja proyek ini.

## Menjalankan Server Lokal
```bash
go build -o les-app .
./les-app
```
*Port default:* `8082`.

## Hal Penting (Gotchas)
1. **Sesi Kehilangan Profil (Blank Page):** Seperti umumnya aplikasi SSR dengan bahasa Go, *Template Layout Utama* sering membutuhkan Data Profil yang disuplai secara paksa dari *Backend*. Jika Anda membuat halaman baru, pastikan Struct Backend mengirim parameter wajib seperti `DisplayName` dan `Role` agar bilah menu kiri (*Sidebar*) berhasil di-render tanpa memicu *Template Panic Error*.
2. **Direktori `scripts/`**: Tidak seperti proyek lain yang memakai aset `static/` (seperti CSS/JS), proyek ini mengompilasi logika bisnis ekspor laporannya ke *PDF/Word* secara bawaan. Folder `scripts/` atau direktori pembantu harus selalu ikut di-_deploy_ bersama dengan _Binary_.
3. **Data Anak Tidak Muncul:** Pada halaman *Ortu*, jika data anak kosong, itu bukan sebuah _bug_. Guru/Admin memang harus membuat profil murid tersebut dan secara manual mengisi kolom `parent_id` agar tertaut dengan akun Ortu terkait secara Database.
