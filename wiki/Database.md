# Skema Database (SQLite)

Menggunakan mekanisme *Foreign Key* dan *ON DELETE CASCADE*.

## 1. Tabel `users`
| Kolom | Tipe | Keterangan |
|---|---|---|
| `username` | TEXT | Unik, untuk login |
| `password_hash` | TEXT | Enkripsi Bcrypt |
| `role` | TEXT | Dibatasi: `guru` atau `ortu` |

## 2. Tabel `students`
Setiap anak dihubungkan ke ID akun milik orang tuanya.
| Kolom | Tipe | Keterangan |
|---|---|---|
| `name` | TEXT | Nama anak |
| `parent_id` | INTEGER | Relasi ke `users(id)` (Wali) |

## 3. Tabel `meetings`
Jadwal pertemuan mengajar.
| Kolom | Tipe | Keterangan |
|---|---|---|
| `student_id` | INTEGER | Anak yang diajar |
| `date`, `start_time` | DATE | Waktu mengajar |
| `status` | TEXT | `terjadwal`, `selesai`, `batal` |

## 4. Tabel `assessments`
Skor untuk tiap aspek yang diukur pada satu *Meeting* yang sama.
*Aturan ketat:* Terdapat constraint `UNIQUE(meeting_id, aspect)` untuk mencegah input ganda. Kolom `score` berstatus `DEFAULT ''` (memperbolehkan simpan data kegiatan walau score belum diisi).

## 5. Tabel `sessions`
Token sesi aktif pengguna untuk autentikasi & CSRF.
| Kolom | Tipe | Keterangan |
|---|---|---|
| `token` | TEXT | Primary Key (32 karakter acak hex) |
| `user_id` | INTEGER | Relasi ke `users(id)` |
| `role` | TEXT | Role pengguna (`guru` / `ortu`) |
| `csrf_token` | TEXT | Token validasi permintaan POST CSRF |
| `created_at` | DATETIME | Waktu sesi dibuat |
