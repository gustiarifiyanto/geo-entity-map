# Geo Entity Map

Aplikasi web untuk menampilkan dan mengelola **entitas yang memiliki lokasi geografis** (kendaraan, perangkat IoT, fasilitas, dll.) di atas map.

- **Backend:** Go + chi + SQLite (`modernc.org/sqlite`, tanpa CGO)
- **Frontend:** React + Vite + TypeScript (strict) + Leaflet + React Query + react-hook-form/zod + Tailwind CSS

## Fitur

- **Login/register** dengan dua role:
  - **`admin`** bisa menambah, mengedit, memindah, dan menghapus entitas.
  - **`user`** hanya bisa melihat map dan detail. Register publik selalu membuat role `user`.
  - Hak akses ditegakkan di backend (401/403), frontend hanya menyembunyikan kontrol.
- **Dashboard** (tab *Map | Dashboard* di bar atas):
  - **Semua role:** total entitas, jumlah per status (warna sama dengan pin), dan jumlah per type.
  - **Admin saja:** jumlah user terdaftar, *Online (last 5 min)*, dan *With an active session* (ketiganya hanya role `user`, admin tidak dihitung), plus jumlah per role. Diperbarui otomatis tiap 30 detik.
  - Pindah tab tidak menghilangkan pin yang dipilih, form yang terbuka, atau posisi map.
- **Logout** lewat ikon di bar atas, selalu dengan dialog konfirmasi (semua role).
- **Tampilan:** bar atas transparan dengan efek blur di atas map, dan animasi halus (tab yang bergeser, panel dan dialog yang muncul perlahan, bar dashboard yang tumbuh). Semua animasi hanya memakai CSS dan otomatis mati jika sistem operasi diatur untuk mengurangi gerakan (*reduce motion*).
- Semua entitas tampil sebagai pin di map. Warna pin menunjukkan status (legenda di kartu kiri atas).
- **Tambah:** klik area kosong di map → form terbuka dengan lat/lng terisi. Selama form terbuka, klik titik lain atau geser pin hitam untuk mengubah lokasi.
- **Detail:** klik pin → panel detail (type, status, koordinat, foto, deskripsi, attributes, timestamp).
- **Edit:** tombol *Edit* di panel detail → form dengan data entitas.
- **Foto entitas:**
  - Semua role melihat galeri di panel detail. Klik foto untuk tampilan besar, dan pindah foto dengan tombol ‹ › atau panah keyboard.
  - Admin menambah atau menghapus foto di form *New entity* / *Edit entity*. Perubahan foto baru diproses saat *Create* / *Save changes*, dan *Cancel* membatalkan semuanya.
  - Maksimal 5 foto per entitas, 5 MB per foto, format JPEG, PNG, atau WebP.
- **Jadwal pemasangan** untuk fasilitas dan perangkat IoT:
  - Admin mengisi tanggal mulai, target, dan selesai (opsional) di bagian *Installation* pada form New/Edit. Bagian ini hanya muncul untuk type yang mendukungnya, berdasarkan `capabilities` dari `GET /api/meta`.
  - Status dihitung server: *Scheduled*, *In progress*, *Overdue*, *Completed on time*, *Completed late*, plus jumlah hari berjalan dan hari terlambat.
  - Panel detail menampilkan badge status, progress bar (bagian merah = lewat target), dan tanggal. Dashboard menampilkan jumlah per status dan daftar yang paling terlambat.
- **Attributes** diisi lewat baris *nama → nilai* (bukan JSON mentah). Nilai seperti `5000` atau `true` tersimpan sebagai angka/boolean, sisanya sebagai teks. Attributes lama yang berisi data bertingkat otomatis diedit dalam mode JSON supaya tidak rusak.
- **Pindah lokasi:** pilih pin, lalu drag → `PATCH /location` dengan *optimistic update*. Kalau gagal, pin kembali ke posisi semula dan muncul toast error. Setelah berhasil, toast menampilkan tombol **Undo**.
- **Hapus:** tombol *Delete* → dialog konfirmasi.
- **Validasi di kedua sisi** dengan aturan dan pesan yang sama. Error 422 dari backend dipetakan ke field form yang sesuai.
- Daftar type dan status **tidak di-hardcode** di frontend, melainkan diambil dari `GET /api/meta`.

## Menjalankan Secara Lokal

Kebutuhan: **Go 1.26+** dan **Node.js 20+**.

```bash
# Terminal 1 — backend (http://localhost:8080), dengan akun admin pertama
cd backend
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=ganti-password-ini go run ./cmd/server

# Terminal 2 — frontend (http://localhost:5173)
cd frontend
npm install
npm run dev
```

Di Windows PowerShell, set env di baris yang sama dengan perintahnya:

```powershell
$env:ADMIN_EMAIL="admin@example.com"; $env:ADMIN_PASSWORD="ganti-password-ini"; go run ./cmd/server
```

Buka **http://localhost:5173**, lalu login dengan email dan password admin di atas, atau buat akun `user` lewat tab *Register*.

- Saat start, backend membuat akun admin **hanya jika email tersebut belum terdaftar**. Log menampilkan `admin account created` atau `admin account already exists`.
- Mengganti `ADMIN_PASSWORD` **tidak** mengubah password admin yang sudah ada. Email yang sudah terdaftar sebagai `user` juga **tidak** dinaikkan menjadi admin.
- Tanpa `ADMIN_EMAIL`/`ADMIN_PASSWORD`, server tetap jalan dengan peringatan di log, tetapi tidak ada yang bisa mengubah entitas. Jika hanya salah satu diisi, atau nilainya tidak lolos aturan register, server menolak start.

- Backend membuat file SQLite dan tabel secara otomatis saat start (auto-migrate).
- 6 entitas contoh (Jakarta & Bandung) dimasukkan **hanya jika tabel masih kosong**.
- Vite mem-proxy `/api` ke backend, jadi tidak perlu setup CORS saat development.
- Tidak ada API key yang dibutuhkan (map memakai tile OpenStreetMap).

### Konfigurasi

| Env var | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP backend |
| `DB_PATH` | `./data/app.db` | Lokasi file SQLite (relatif terhadap folder `backend/`) |
| `ADMIN_EMAIL` | – | Email admin pertama (lihat di atas) |
| `ADMIN_PASSWORD` | – | Password admin pertama, 8–72 byte. **Jangan di-commit.** |
| `COOKIE_SECURE` | `false` | Set `true` jika app disajikan lewat HTTPS, supaya cookie session hanya dikirim lewat HTTPS |

| `UPLOAD_DIR` | `./data/uploads` | Folder file foto, satu subfolder per entitas (relatif terhadap folder `backend/`) |
| `APP_TIMEZONE` | `Asia/Jakarta` | Zona waktu IANA untuk menentukan "hari ini" pada status pemasangan. Nilai yang salah membuat server menolak start. |

Untuk mengulang dari data contoh, hentikan backend lalu hapus folder `backend/data/`. Semua akun, session, dan foto ikut terhapus.

## Testing

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run lint
```

- **Validasi backend** (`internal/validation`): test table-driven untuk input valid, tiap kasus tidak valid, dan nilai batas (lat `90` vs `90.0001`, lng `180` vs `180.0001`, NaN/Inf, panjang nama 100 vs 101 karakter termasuk multibyte, attributes non-objek).
- **Validasi auth**: email 254 vs 255 karakter, password 7 vs 8 karakter dan 72 vs 73 byte (termasuk huruf multibyte), password tidak di-trim, login hanya mengecek field wajib.
- **Handler** (`internal/handler`): test HTTP terhadap SQLite sungguhan di folder sementara, mencakup status 200/201/204/400/404/405/422, filter list, update penuh, update lokasi, dan hapus.
- **Auth** (`internal/handler/auth_test.go`):
  - register (role selalu `user`, field `role` ditolak), email duplikat beda huruf besar/kecil (422);
  - login gagal dengan pesan yang sama untuk email tidak terdaftar maupun password salah (401);
  - session kedaluwarsa, logout, login ulang mengganti session lama;
  - matriks hak akses semua route × belum login / `user` / `admin`, dan cek auth berjalan sebelum validasi body;
  - bentuk response `/api/admin/stats`, termasuk role yang jumlahnya 0.
- **Service auth** (`internal/service/auth_test.go`, dengan jam palsu):
  - `last_seen_at` tidak ditulis di detik ke-30 dan ke-59, lalu ditulis tepat di menit ke-1;
  - session lama dengan `last_seen_at` `NULL` tetap bisa dipakai;
  - statistik: user dengan 2 session dihitung sekali, session kedaluwarsa tidak dihitung, user yang diam 6 menit tidak online, user tepat di batas 5 menit masih online.
- **Foto** (`internal/handler/photo_test.go`):
  - JPEG/PNG/WebP diterima walaupun nama file-nya menyesatkan; file HTML bernama `.jpg` dan GIF ditolak;
  - file kosong, tepat 5 MB vs 5 MB + 1 byte, foto ke-6, field yang salah, dua foto sekaligus, body bukan multipart;
  - entitas/foto tidak ada (404), header `nosniff` dan cache saat file diambil;
  - hapus foto dan hapus entitas ikut menghapus file di disk; upload yang ditolak tidak meninggalkan file;
  - hak akses belum login / `user` / `admin`.
- **Pemasangan** (`internal/service/installation_test.go` dengan tanggal tetap, `internal/handler/installation_test.go`):
  - status di setiap batas: mulai besok → *scheduled*; mulai hari ini / target hari ini → *in_progress*; target kemarin → *overdue* 1 hari; selesai tepat di target atau lebih awal → *on time*; sehari setelah target → *late*;
  - "hari ini" mengikuti `APP_TIMEZONE` (18.00 UTC tanggal 5 = tanggal 6 di Jakarta);
  - validasi tanggal: format, tanggal yang tidak ada (`2026-02-30`), urutan, selesai di masa depan;
  - type tanpa kemampuan → 400, entitas tidak ada → 404, urutan daftar, ganti type menyembunyikan jadwal, hapus entitas ikut menghapus jadwal, hak akses tiap role.
- **Penyimpanan file** (`internal/storage`): tulis lewat file sementara lalu rename, nama berbahaya (`""`, `..`, `../x`, `a/b`) ditolak tanpa menghapus apa pun.
- **Database** (`internal/database`): migrasi idempoten, seed hanya saat tabel kosong, constraint koordinat, CHECK role, email unik, hapus user ikut menghapus session, migrasi menambah `last_seen_at` ke tabel `sessions` lama tanpa kehilangan data.

## API

Base path `/api`. Request dan response berformat JSON.

| Method | Path | Deskripsi | Sukses |
|---|---|---|---|
| GET | `/api/meta` | Daftar type, status, dan kemampuan per type (`capabilities`) | 200 |
| GET | `/api/entities` | List entitas, filter opsional `?type=&status=` | 200 |
| GET | `/api/entities/{id}` | Detail entitas | 200 |
| POST | `/api/entities` | Membuat entitas | 201 |
| PUT | `/api/entities/{id}` | Update penuh | 200 |
| PATCH | `/api/entities/{id}/location` | Update hanya `latitude` + `longitude` | 200 |
| DELETE | `/api/entities/{id}` | Menghapus entitas | 204 |
| POST | `/api/auth/register` | Buat akun `user` `{email, password}` lalu langsung login | 201 |
| POST | `/api/auth/login` | Login `{email, password}` | 200 |
| POST | `/api/auth/logout` | Hapus session (tetap 204 walau belum login) | 204 |
| GET | `/api/auth/me` | User yang sedang login | 200 |
| GET | `/api/admin/stats` | Statistik user untuk dashboard admin | 200 |
| GET | `/api/entities/{id}/photos` | Daftar foto entitas, terlama dulu | 200 |
| POST | `/api/entities/{id}/photos` | Upload satu foto (`multipart/form-data`, field `photo`) | 201 |
| GET | `/api/photos/{photoId}` | File gambar (bukan JSON) | 200 |
| DELETE | `/api/photos/{photoId}` | Menghapus foto | 204 |
| GET | `/api/entities/{id}/installation` | Jadwal pemasangan; `{ "data": null }` jika belum diisi | 200 |
| PUT | `/api/entities/{id}/installation` | Isi/ubah jadwal `{ started_on, target_on, completed_on }` | 200 |
| DELETE | `/api/entities/{id}/installation` | Hapus jadwal | 204 |
| GET | `/api/installations` | Semua jadwal (dashboard), paling terlambat dulu | 200 |

Objek user: `{ "id", "email", "role", "created_at" }`. Hash password tidak pernah dikirim.

**Hak akses:**

| Endpoint | Belum login | `user` | `admin` |
|---|---|---|---|
| `/api/auth/*` | ✅ | ✅ | ✅ |
| `GET /api/meta`, `GET /api/entities[/{id}]` | 401 | ✅ | ✅ |
| `POST`, `PUT`, `PATCH .../location`, `DELETE` pada `/api/entities` | 401 | 403 | ✅ |
| `GET /api/admin/stats` | 401 | 403 | ✅ |
| `GET .../photos`, `GET /api/photos/{photoId}` | 401 | ✅ | ✅ |
| `POST .../photos`, `DELETE /api/photos/{photoId}` | 401 | 403 | ✅ |
| `GET .../installation`, `GET /api/installations` | 401 | ✅ | ✅ |
| `PUT`, `DELETE .../installation` | 401 | 403 | ✅ |

**Session:** login dan register memasang cookie `session` (`HttpOnly`, `SameSite=Lax`, berlaku 7 hari). Isinya token acak 32 byte; database hanya menyimpan hash SHA-256 token tersebut, sehingga file DB yang bocor tidak bisa dipakai untuk login.

**Statistik admin** (`GET /api/admin/stats`):

```json
{ "data": {
    "users": { "total": 12, "by_role": { "user": 10, "admin": 2 },
               "with_active_session": 5, "online": 2 },
    "online_window_minutes": 5 } }
```

- `total`, `with_active_session`, dan `online` **hanya menghitung role `user`**. Admin tidak ikut dihitung karena merekalah yang membaca dashboard; jumlah admin tetap terlihat di `by_role`.
- Yang dihitung adalah **user**, bukan session: satu user yang login di dua browser dihitung sekali.
- `by_role` selalu berisi semua role, termasuk yang jumlahnya 0.
- `with_active_session`: user yang punya session belum kedaluwarsa (belum logout dan belum lewat 7 hari).
- `online`: user dengan session aktif yang dipakai dalam `online_window_minutes` terakhir. Setiap request yang sudah login memperbarui `sessions.last_seen_at`, tetapi paling sering sekali per menit, supaya kebanyakan request hanya membaca DB. Selama tab browser terlihat, frontend memanggil `GET /api/auth/me` tiap 2 menit (*heartbeat*), jadi user yang membuka map tapi diam tetap terhitung online.
- Statistik entitas dihitung di frontend dari `GET /api/entities`, jadi tidak perlu endpoint tambahan.

**Foto:**

```json
{ "data": { "id": "…", "entity_id": "…", "url": "/api/photos/…",
            "content_type": "image/png", "size_bytes": 24512,
            "created_at": "2026-10-06T07:24:46Z" } }
```

- Jenis file ditentukan dari **isi file** (`http.DetectContentType`), bukan dari nama file atau header kiriman client. File HTML yang diberi nama `.jpg` ditolak.
- File disimpan di `UPLOAD_DIR/<entityId>/<photoId>.<ext>`. Namanya selalu dibuat server, dan setiap nama dicek supaya path tidak bisa keluar dari folder upload.
- Batas 5 foto dicek dalam satu query bersama insert, jadi dua upload bersamaan tidak bisa melewatinya.
- `GET /api/photos/{id}` mengirim `X-Content-Type-Options: nosniff` (browser tidak menebak tipe lain) dan `Cache-Control: private, max-age=86400` (foto tidak pernah diubah, hanya dihapus).
- Bentuk objek entitas tidak berubah; foto diambil lewat endpoint sendiri.

**Kemampuan per type & jadwal pemasangan:**

`GET /api/meta` mengirim `"capabilities": { "iot_device": ["installation"], "facility": ["installation"] }`. Daftar ini hanya ada di `backend/internal/model/installation.go`, dan frontend menampilkan fitur khusus berdasarkan daftar itu, tanpa menulis nama type sendiri. Perangkat IoT ditambahkan dengan mengubah satu baris di file tersebut.

```json
{ "data": { "entity_id": "…", "started_on": "2026-09-20", "target_on": "2026-10-01",
            "completed_on": null, "status": "overdue",
            "planned_days": 11, "elapsed_days": 16, "days_late": 5,
            "updated_at": "2026-10-06T08:11:44Z" } }
```

| Status | Kapan |
|---|---|
| `scheduled` | tanggal mulai setelah hari ini |
| `in_progress` | belum selesai, hari ini ≤ target |
| `overdue` | belum selesai, hari ini > target |
| `completed_on_time` | selesai ≤ target |
| `completed_late` | selesai > target |

- Status dan jumlah hari **dihitung saat diminta**, tidak disimpan, jadi selalu sesuai dengan tanggal hari ini.
- Tanggal disimpan sebagai `YYYY-MM-DD` tanpa jam. "Hari ini" dihitung di `APP_TIMEZONE` (default `Asia/Jakarta`), supaya status berganti tengah malam WIB, bukan jam 07.00. Data zona waktu disertakan di binary (`time/tzdata`), jadi tetap jalan di Windows.
- Endpoint jadwal untuk entitas yang type-nya tidak mendukung (misalnya kendaraan) → **400 `invalid_request`**. Kalau type entitas diganti, jadwalnya disimpan tetapi disembunyikan.

Format response:

```jsonc
// sukses
{ "data": { /* entity */ } }          // atau { "data": [ ... ] }

// 422 validasi — key memakai nama field JSON
{ "error": "validation_failed", "fields": { "latitude": "must be between -90 and 90" } }

// error lain
{ "error": "not_found", "message": "entity not found" }
```

| Status | Kode | Kapan |
|---|---|---|
| 400 | `invalid_json` | Body rusak/kosong, ada field yang tidak dikenal, lebih dari satu objek JSON, atau > 1 MiB |
| 400 | `invalid_request` | Upload foto: body bukan `multipart/form-data`, field selain `photo`, atau lebih dari satu foto |
| 401 | `unauthorized` | Belum login, atau session tidak valid / kedaluwarsa |
| 401 | `invalid_credentials` | Email atau password salah (pesan sama untuk keduanya, supaya tidak membocorkan email yang terdaftar) |
| 403 | `forbidden` | Sudah login, tapi bukan admin |
| 404 | `not_found` | Entitas tidak ada, ID bukan UUID, atau route tidak ada |
| 405 | `method_not_allowed` | Method tidak didukung untuk path tersebut |
| 422 | `validation_failed` | Validasi gagal (lihat `fields`) |
| 500 | `internal_error` | Detail internal hanya dicatat di log server, tidak dikirim ke client |

Keputusan tambahan yang tidak diatur di brief awal (disetujui developer):

- Field dengan **tipe salah** (misalnya `"latitude": "north"`) → **422** dengan pesan per field (`"must be a number"`), bukan 400, karena body-nya JSON yang valid. Angka di luar jangkauan float64 (misalnya `1e400`) → `"is out of range"`.
- Filter list yang tidak valid (misalnya `?type=rocket`) → **422**, bukan list kosong.
- `POST` mengembalikan header `Location: /api/entities/{id}`.
- `PUT` adalah update penuh: `description`/`attributes` yang tidak dikirim akan dikosongkan. `created_at` tidak berubah.
- Email yang sudah terdaftar saat register → **422** `fields.email: "is already registered"` (bukan 409), supaya langsung tampil di bawah input.
- Upload foto yang tidak valid → **422** di `fields.photo`. Body upload yang bukan `multipart/form-data` yang valid → **400 `invalid_request`** (kode baru, karena `invalid_json` tidak tepat untuk upload file).
- Auth dicek **sebelum** body divalidasi: request tanpa hak akses selalu mendapat 401/403, tidak pernah 422.

## Model Data & Aturan Validasi

| Field | Aturan |
|---|---|
| `id` | UUID v4, dibuat backend |
| `name` | wajib, di-trim, 1–100 karakter (dihitung per karakter Unicode, bukan byte) |
| `type` | wajib, salah satu dari `GET /api/meta` |
| `status` | wajib, salah satu dari `GET /api/meta` |
| `latitude` | wajib, −90 … 90, bukan NaN/Inf |
| `longitude` | wajib, −180 … 180, bukan NaN/Inf |
| `description` | opsional, di-trim, maks. 500 karakter |
| `attributes` | opsional, harus **objek** JSON (`null` = kosong). Di form diisi sebagai baris nama → nilai; nama wajib dan tidak boleh dobel |

**Jadwal pemasangan:** `started_on` dan `target_on` wajib, format `YYYY-MM-DD` persis dan tanggalnya benar-benar ada; `target_on` ≥ `started_on`; `completed_on` opsional, ≥ `started_on`, dan tidak boleh setelah hari ini.
| `created_at`, `updated_at` | RFC3339 UTC (presisi detik), diisi backend |

**User:**

| Field | Aturan |
|---|---|
| `email` | wajib, di-trim + lowercase, format email valid, maks. 254 karakter, unik |
| `password` (register) | wajib, 8–72 **byte** (batas bcrypt; huruf seperti `é` dihitung 2 byte), **tidak di-trim** |
| `password` (login) | cukup wajib diisi; password yang terlalu pendek dianggap salah (401), bukan 422 |
| `role` | `user` atau `admin` (enum di `backend/internal/model/user.go` + CHECK di SQL) |

**Enum hanya didefinisikan di `backend/internal/model/`.** Untuk menambah type atau status cukup menambah konstanta di sana. Tidak perlu migrasi DB maupun perubahan frontend, karena label dan warna punya fallback untuk nilai yang belum dikenal.

## Struktur Repository

```
backend/
  cmd/server/main.go       entrypoint: config, DB, migrate, seed, router, graceful shutdown
  internal/model/          Entity, User, input, enum (satu-satunya sumber kebenaran)
  internal/validation/     validator + format error per field
  internal/handler/        HTTP: decode, validasi, response, router, middleware auth (401/403)
  internal/service/        logika bisnis (UUID, timestamp, bcrypt, token session)
  internal/repository/     query SQL berparameter (entities, users, sessions)
  internal/database/       koneksi, migrasi (termasuk penambahan kolom untuk DB lama), seed
  internal/storage/        file foto di disk (satu folder per entitas, nama selalu dicek)
frontend/src/
  api/                     fetch client (ApiError) + fungsi API yang typed
  components/              AppBar (tab Map | Dashboard), ConfirmDialog, toast, style form bersama
  features/auth/           halaman login/register, hooks user saat ini + heartbeat
  features/dashboard/      ringkasan entitas, statistik user admin
  features/photos/         galeri view-only, tampilan foto besar, pemilih foto di form (draft)
  features/installations/  bagian Installation di form, ringkasan status di panel detail, label/warna status
  features/entities/       map, marker, form, panel detail, hooks React Query, helper geo
  schemas/                 schema zod (mengikuti aturan backend)
  types/                   tipe bersama
```

Layering backend: `handler → service → repository`. Handler tidak menjalankan SQL, dan repository tidak tahu soal HTTP.

## Alasan Pemilihan Library

### Backend

| Library | Alasan |
|---|---|
| `go-chi/chi/v5` | Router ringan yang kompatibel dengan `net/http`, mendukung parameter path dan sub-router tanpa framework berat |
| `modernc.org/sqlite` | Driver SQLite **pure Go (tanpa CGO)**, jadi `go run` langsung jalan di Windows/macOS/Linux tanpa compiler C |
| `database/sql` (stdlib) | Cuma satu tabel, ORM tidak sebanding dengan kompleksitasnya. Semua query berparameter. |
| `go-playground/validator/v10` | Validasi berbasis tag struct dengan dukungan aturan custom (enum, rentang koordinat, objek JSON) |
| `google/uuid` | Membuat dan memvalidasi UUID v4 untuk `id`. Tidak tercantum di tech stack awal, ditambahkan karena dibutuhkan untuk ID. |
| `golang.org/x/crypto/bcrypt` | Hash password yang lambat secara sengaja dan sudah memakai salt. Paket semi-resmi tim Go; sebelumnya sudah ada sebagai dependency tidak langsung. |

**Session di server + cookie HttpOnly, bukan JWT.** Session bisa dicabut seketika (logout menghapus barisnya di DB), token tidak bisa dibaca JavaScript, dan tidak perlu library tambahan. Token dan hash-nya dibuat dengan `crypto/rand` dan `crypto/sha256` dari stdlib.

### Frontend

| Library | Alasan |
|---|---|
| React + Vite + TypeScript | Vite memberi dev server cepat dan proxy `/api`. |
| Leaflet + `react-leaflet` | Library map gratis dan ringan, mendukung marker yang bisa di-drag, dengan API sederhana |
| Tile OpenStreetMap | Gratis dan **tanpa API key**, jadi reviewer bisa langsung menjalankan app. CARTO (sekarang butuh API key) dan Stadia sempat dicoba lalu tidak dipakai. |
| `@tanstack/react-query` | Server state: caching, refetch setelah mutasi, optimistic update + rollback untuk drag marker |
| `react-hook-form` + `zod` + `@hookform/resolvers` | Form performan dengan error per field. Schema zod dibangun dari `/api/meta` dan mengikuti aturan backend. |
| Tailwind CSS | Styling cepat dan konsisten tanpa file CSS terpisah per komponen |

Dialog konfirmasi, toast, halaman login, dan bar di dashboard dibuat sendiri (elemen `<dialog>` bawaan browser + context React + react-hook-form + `div` dengan Tailwind) supaya **tidak menambah dependency**. Library chart tidak diperlukan untuk beberapa bar horizontal. Upload foto memakai `multipart` dan `http.DetectContentType` dari stdlib Go, dan `FormData` + `<input type="file">` bawaan browser. Tidak ada router: halaman login tampil saat belum login, dan tab Map | Dashboard cukup berupa state biasa.

## Workflow AI

Saya mengerjakan proyek ini bersama **Claude Code** (Anthropic) sebagai *pair programmer*. AI membantu menulis kode, tetapi saya yang mengatur alur kerja, mereview semua kode, mengambil keputusan desain akhir, dan berperan sebagai QA.

1. **`CLAUDE.md` sebagai sumber kebenaran.** Saya menyusun `CLAUDE.md` dengan bantuan AI: brief, tech stack, kontrak API, aturan validasi, dan aturan untuk agent (tidak menambah dependency atau mengubah kontrak tanpa persetujuan saya, test ditulis bersamaan dengan perubahan backend, dll.). File ini dibaca AI sebelum menulis kode apa pun.
2. **Rencana bertahap.** Saya memecah pekerjaan menjadi 9 langkah, satu commit kecil per langkah dengan gaya conventional commits: setup → model/DB → validasi → API → data layer frontend → map → form → drag/hapus → README. Saya mereview diff setiap langkah sebelum commit.
3. **Keputusan desain di tangan saya.** Setiap kali ada penyimpangan dari `CLAUDE.md` atau pilihan desain, AI wajib berhenti dan bertanya. Beberapa keputusan yang saya ambil:
   - **Basemap:** tetap memakai tile OpenStreetMap supaya reviewer bisa langsung menjalankan app tanpa API key.
   - **Pengaman drag:** hanya pin yang sedang dipilih yang bisa di-drag, ditambah tombol Undo di toast.
   - **422 vs 400:** field dengan tipe salah dikembalikan sebagai 422 per field, bukan 400, karena body-nya tetap JSON yang valid.
   - Hal lain yang saya setujui: versi Go dinaikkan ke 1.26 karena dependency menuntutnya, dan perilaku form saat klik map.
4. **Verifikasi sebelum dinyatakan selesai.** Setiap langkah ditutup dengan `gofmt`/`go vet`/`go test` atau `typecheck`/`lint`/`build`. Selain itu ada smoke test API dengan curl lewat proxy Vite, dan pengecekan schema zod serta helper geo/error-mapping dengan skrip Node terhadap kasus yang sama dengan test backend.
5. **QA manual oleh saya.** Saya mencoba UI langsung di browser. Dari situ saya menemukan masalah nyata: marker bisa **tergeser tanpa sengaja** saat menggeser map. Setelah diperbaiki dengan pengaman drag + Undo, saya mengetes ulang: hanya pin terpilih yang bisa di-drag, pin terkunci saat form edit terbuka, Undo mengembalikan posisi (tetap setelah refresh), dan pin kembali ke posisi semula disertai toast error saat backend mati.
6. **Kesalahan AI yang saya tangkap dan perbaiki:**
   - asersi test yang salah membaca `null` pada `json.RawMessage`;
   - klaim bahwa tile CARTO gratis tanpa API key: curl mengembalikan 200, tapi gambarnya ternyata watermark "API KEY REQUIRED". Saya menemukannya dari screenshot di browser, lalu perubahan itu langsung di-revert.
7. **Fitur auth setelah MVP, di branch `feat/auth`.** Supaya MVP di `main` tetap utuh, auth dikerjakan di branch terpisah. Kontraknya (endpoint, hak akses, tabel, kode error) saya review dan setujui di `CLAUDE.md` sebelum ada kode. Saat QA saya menemukan masalah UX yang lalu diperbaiki:
   - pesan *"is required"* sudah muncul sebelum form disubmit, karena validasi berjalan saat input kehilangan fokus. Sekarang validasi berjalan saat submit, di form login/register maupun form entitas;
   - tidak bisa memastikan password sudah terketik benar, jadi saya minta tombol *Show/Hide* password;
   - pesan error saat backend mati atau env admin hanya terisi sebagian kurang jelas, sehingga diperjelas.
8. **Dashboard di branch `feat/dashboard`.** Saya memutuskan dashboard admin dan user digabung dalam satu branch karena keduanya mengubah area UI yang sama. Untuk "user yang sedang login", AI menawarkan dua arti: punya session aktif (sederhana) atau benar-benar aktif dalam 5 menit terakhir (butuh kolom dan heartbeat). Saya memilih yang kedua setelah memastikan perubahannya tidak besar, dan keduanya ditampilkan dengan label yang jujur. Sebelum dipakai di data saya, migrasi kolom baru diuji pada salinan database lokal.
9. **Foto entitas dan editor attributes.** Saat QA saya mengubah beberapa keputusan UX:
   - upload foto dipindah dari panel detail ke form *New/Edit entity*, dan saya memilih perubahan foto ikut diproses saat *Save* (bukan langsung), supaya *Cancel* tetap membatalkan semuanya;
   - kolom attributes yang awalnya berupa JSON mentah saya minta diganti dengan baris nama → nilai, karena JSON mudah salah ketik bagi user biasa;
   - saya juga menemukan kolom nilai yang "kegepeng" di editor attributes, lalu diperbaiki.
10. **Jadwal pemasangan.** Saya meminta fitur durasi pemasangan untuk mengontrol kualitas pemasangan fasilitas. Saya menyetujui tanggal tanpa jam, status yang dihitung server, dan zona waktu Jakarta. Setelah fiturnya jadi, saya memutuskan perangkat IoT juga perlu jadwal pemasangan. Karena fitur khusus per type sejak awal dirancang lewat `capabilities` di `/api/meta`, perubahan itu cukup satu baris di backend, dan UI langsung mengikuti.

## Fitur yang Belum Selesai & Keterbatasan

**Belum dikerjakan (nice to have):**

- Filter type/status di UI. Backend dan hook `useEntities(filter)` sudah mendukung, tinggal komponen UI-nya.
- Sidebar daftar entitas yang tersinkron dengan map.
- Clustering marker (butuh dependency tambahan).
- Update realtime (SSE/WebSocket). Saat ini perubahan dari tab atau user lain baru terlihat saat refetch (misalnya saat window kembali difokus).
- **Lupa password / ganti password.**
- **Kelola user** (daftar user, menaikkan user menjadi admin, menghapus akun). Admin tambahan saat ini hanya bisa dibuat lewat `ADMIN_EMAIL`/`ADMIN_PASSWORD` dengan email baru.
- Daftar nama user yang sedang online, serta grafik/riwayat aktivitas di dashboard.
- Untuk foto: resize/thumbnail otomatis, crop, mengatur urutan foto, dan keterangan (caption) per foto.
- Untuk jadwal pemasangan: riwayat perubahan tanggal, tahapan/milestone, penanggung jawab/kontraktor, dan notifikasi saat terlambat.
- Direncanakan (urutan sudah disepakati): data sensor IoT (endpoint perangkat + API key, data dummy) → live tracking kendaraan (simulator + SSE). Pilihan bahasa Indonesia/English juga ditunda.
- **Rate limiting login.** Belum ada pembatasan percobaan login berulang.

**Keterbatasan yang diketahui:**

- **Tidak ada test otomatis di frontend.** Kualitas dijaga lewat `typecheck` + `lint` dan skrip pengecekan manual. Repository dan service entitas tidak punya unit test terpisah, tetapi teruji lewat test handler yang memakai SQLite sungguhan. Service auth punya unit test sendiri untuk logika yang bergantung pada waktu.
- **Status online bukan realtime.** Dihitung dari aktivitas terakhir (batas 5 menit), bukan koneksi terbuka. User yang baru menutup browser masih terhitung online sampai ±5 menit; tab browser yang tersembunyi tidak mengirim heartbeat, jadi user-nya bisa terhitung offline walaupun app masih terbuka.
- *With an active session* ikut menghitung user yang menutup browser tanpa logout, sampai session-nya kedaluwarsa (7 hari).
- **Undo pindah lokasi** hanya tersedia selama toast tampil (~8 detik). Jika pin yang sama dipindah dua kali, Undo dari toast pertama mengembalikan ke posisi sebelum pemindahan pertama.
- Jika data di-refetch tepat saat pin sedang di-drag, pin bisa melompat ke posisi dari server. Kemungkinannya kecil karena refetch otomatis hanya terjadi saat window kembali difokus.
- **Last write wins:** tidak ada pengecekan konflik/versi saat dua admin mengedit entitas yang sama.
- **Perlindungan CSRF** hanya mengandalkan `SameSite=Lax` pada cookie dan tidak adanya CORS (request lintas situs dengan body JSON diblokir browser). Belum ada token CSRF terpisah.
- **Session tidak diperpanjang otomatis.** Setelah 7 hari user harus login ulang walaupun aktif. Session kedaluwarsa dibersihkan dari DB setiap ada login.
- Format email dicek oleh dua library berbeda (`validator` di Go, `z.email()` di frontend). Untuk email normal hasilnya sama, tetapi format yang sangat tidak lazim bisa diterima di satu sisi dan ditolak di sisi lain. Backend tetap jadi penentu.
- List entitas tidak dipaginasi. Cukup untuk ratusan entitas, belum untuk skala sangat besar.
- **Foto tidak di-resize.** Thumbnail di galeri memuat file aslinya (sampai 5 MB), jadi bisa terasa lambat di koneksi pelan.
- **Isi foto hanya dicek dari tanda tangan awal file** (seperti `http.DetectContentType`), bukan di-decode penuh. File dengan header gambar yang benar tapi isinya rusak tetap diterima; browser hanya gagal menampilkannya.
- **File foto dan baris DB tidak dalam satu transaksi.** Kalau server mati tepat di antara menyimpan file dan menulis ke DB, bisa tersisa file yatim di `UPLOAD_DIR` (tidak terlihat di app, hanya memakan ruang disk).
- **Perubahan foto di form diproses setelah entitas tersimpan.** Kalau upload sebagian gagal, entitas tetap tersimpan dan muncul toast berisi jumlah yang gagal; foto yang gagal perlu ditambahkan ulang.
- **Validasi "tidak boleh di masa depan" di frontend memakai tanggal browser**, sedangkan backend memakai `APP_TIMEZONE`. Kalau zona waktu browser berbeda jauh, frontend bisa lolos tapi backend menolak (atau sebaliknya). Backend tetap jadi penentu, dan entitasnya tetap tersimpan dengan toast berisi error jadwal.
- **Nama tabel `facility_installations`** tetap dipakai walaupun sekarang juga untuk perangkat IoT, supaya tidak perlu migrasi ganti nama.
- **Angka di attributes:** teks yang persis seperti angka (misalnya `1234`) selalu disimpan sebagai angka. Untuk memaksa disimpan sebagai teks, perlu mode JSON.
- Backend tidak menyajikan file hasil build frontend dan tidak mengatur CORS. Untuk produksi, keduanya perlu disajikan dari origin yang sama (reverse proxy) atau CORS perlu ditambahkan.
- **Tile OpenStreetMap** tunduk pada [Tile Usage Policy](https://operations.osmfoundation.org/policies/tiles/) dan tidak ditujukan untuk trafik produksi tinggi. Untuk produksi, gunakan penyedia tile berbayar atau host tile sendiri.
- Vite memberi peringatan ukuran bundle > 500 kB (Leaflet + React dalam satu chunk). Belum dilakukan code splitting.
- Dua perbedaan kecil yang disengaja antara frontend dan backend:
  - JSON `attributes` yang rusak (hanya mungkin di mode JSON untuk data bertingkat) ditangkap frontend sebagai error field `"must be valid JSON"`, sedangkan jika dikirim langsung ke API hasilnya 400 `invalid_json` (sesuai kontrak).
  - `String.prototype.trim()` di JS memangkas beberapa karakter spasi Unicode (misalnya U+FEFF) yang tidak dipangkas `strings.TrimSpace` di Go. Frontend sedikit lebih ketat, jadi tidak menyebabkan error 422 yang tak terduga.
