# Arsitektur Aplikasi

Aplikasi menggunakan arsitektur monolitik yang dirender sepenuhnya oleh server (Server-Side Rendering).

## Struktur File
```
/root/les-private/
├── main.go             # Router, Middleware, Setup
├── auth.go             # Logika Login, Register, Cookies
├── db.go               # Koneksi SQLite
├── models.go           # Definisi Struct (Data Penampung)
├── handlers_guru.go    # Kontroler Rute Khusus 'Guru'
├── handlers_ortu.go    # Kontroler Rute Khusus 'Ortu'
├── handlers_profile.go # Pengaturan profil pengguna
├── reports.go          # Penghasil Dokumen PDF Word
└── templates/          # Kumpulan Layout HTML
```

## Mekanisme Akses Lapis Ganda (Middleware)
Karena terdapat pemisahan pengguna (`guru` dan `ortu`), proses autentikasinya dilakukan ketat oleh fungsi:
`authMiddleware(requiredRole string, next http.HandlerFunc)`

**Alur Pengecekan:**
1. Verifikasi **Cookie** (Validasi Token).
2. Pengecekan status **Sesi di Database**.
3. Pengecekan **Role**. Jika user yang *login* adalah `ortu` tapi memaksa mengakses rute `/students` milik Guru, akan ditolak otomatis.
