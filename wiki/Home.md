# Sistem Jadwal dan Rapor Les Private

Aplikasi manajemen operasional Les Private yang dikhususkan untuk sistem **Satu Guru (Single-Teacher)** dan interaksi langsung dengan **Orang Tua / Wali Murid**. Dibangun untuk mendigitalkan penjadwalan serta memberikan pemantauan perkembangan anak (*Rapor Aspek*) secara *real-time*.

## Teknologi Utama
- **Backend:** Go 1.25 (Pustaka `net/http` murni tanpa framework)
- **Database:** SQLite3 (Berjalan pada mode WAL)
- **Frontend:** Server-Side Rendering HTML (`html/template`) dengan Bootstrap 5
- **Fitur Spesial:** Ekspor Rapor otomatis ke format **.PDF** dan **.DOCX**

## Konsep Penilaian 6 Aspek
Guru menilai progres anak di setiap pertemuan berdasarkan 6 kategori utama:
1. Pra membaca
2. Menulis
3. Berhitung
4. Sensory play
5. Kreativitas
6. Brain game

*Skala Penilaian: "Masih berkembang", "Berkembang sesuai harapan", "Berkembang dengan baik".*

## Indeks Wiki
- [Arsitektur Sistem](Architecture.md)
- [Struktur Database](Database.md)
- [Fitur Akses](Features.md)
- [Changelog](Changelog.md)
