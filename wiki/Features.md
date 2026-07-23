# Fitur Hak Akses (Role-Based)

## Panel GURU
Guru adalah pemegang kendali utama (*Super-Admin* dari sistem pendidikan).
- **Manajemen Siswa:** Menambahkan anak dan menautkannya ke akun *Ortu* yang ada.
- **Jadwal Les:** Membuat tiket pertemuan.
- **Penilaian Aspek:** Saat pertemuan berstatus `selesai`, Guru bisa menginput Nilai, Catatan, dan Tema Belajar.
- **Laporan Keseluruhan:** Membaca grafik kinerja anak dan mencetak Rapor (*PDF/Docx*).

## Panel ORTU
Hanya sebagai pemantau (Read-Only).
- **Dashboard Personal:** Ortu hanya melihat data anak-anak milik mereka saja.
- **Riwayat Belajar:** Melihat komentar Guru pada pertemuan yang lalu.
- **Unduh Rapor:** Ortu dapat mendownload sendiri Rapor resmi anak ke dalam format PDF/Word.

## Register Terbuka
Siapapun bisa mendaftar lewat halaman `/register`, namun peran bawaannya (Default) otomatis diset sebagai **`ortu`**. Akun Ortu yang baru terdaftar belum bisa melihat apa-apa hingga Guru membuatkan profil "Student" dan menautkannya ke akun tersebut.
