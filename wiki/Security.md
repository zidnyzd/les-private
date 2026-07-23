# Keamanan (Security)

## Autentikasi Lapis Ganda
- **Sandi**: Semua kata sandi diamankan menggunakan enkripsi **Bcrypt** (cost standar 10).
- **Session ID**: Token sesi terdiri dari karakter acak (random hex string) berukuran 32 karakter dan disimpan pada cookie dengan atribut `HttpOnly`, `Secure`, dan `SameSite=Lax`.
- **Persistensi Sesi**: Token sesi dan CSRF token kini di-persist secara langsung di database SQLite `sessions`, serta dipulihkan otomatis saat server startup (`loadSessions`), menjaga sesi aktif pengguna tetap tersimpan walau dideploy/restart.

## Pembagian Peran (Role-Based Access Control / RBAC)
Sistem memiliki dua peran (*role*) yang sangat berlawanan secara hak akses:
1. **Guru (`guru`)** — Memiliki akses mutlak (CRUD) ke seluruh data anak, jadwal pertemuan, dan nilai 6 aspek perkembangan anak.
2. **Ortu (`ortu`)** — Hanya memiliki izin baca (*Read-Only*) dan hak aksesnya dibatasi secara teknis HANYA pada data anak yang berelasi langsung dengan akun orang tua tersebut (melalui parameter `parent_id` di database).

Semua permintaan HTTP dicegat terlebih dahulu oleh `authMiddleware(requiredRole string, ...)` sebelum dapat mengeksekusi logika apapun.

## Rate Limiting
Saat ini aplikasi **belum** dilengkapi In-Memory Rate Limiting seperti yang ada pada *Finance Tracker*. Disarankan memasangnya kelak jika pendaftaran ortu mulai rentan terhadap serangan Bot.
