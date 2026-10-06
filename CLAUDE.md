# CLAUDE.md

Panduan untuk Claude (dan AI agent lain) yang bekerja di repository ini. Baca file ini sampai selesai sebelum melakukan perubahan apa pun.

## Gambaran Proyek

Aplikasi web untuk menampilkan dan mengelola **entitas yang memiliki lokasi geografis** di atas map. Entitas dapat merepresentasikan objek apa pun di dunia nyata (kendaraan, perangkat IoT, fasilitas, dll.) dan memiliki identitas, atribut dasar, status, serta koordinat geografis.

Fitur inti:
- Menampilkan semua entitas sebagai marker di map
- Menambah entitas dengan klik di map
- Melihat detail entitas (popup / side panel)
- Mengubah entitas lewat form, dan mengubah lokasinya dengan drag marker
- Menghapus entitas (dengan konfirmasi)
- Validasi input di **kedua sisi**, frontend dan backend
- Login/register dengan role `admin` (kelola entitas) dan `user` (hanya melihat)

Ini adalah take-home test dengan deadline ketat. **Utamakan MVP yang rapi dan berjalan baik dibanding fitur tambahan.**

## Tech Stack

| Bagian | Pilihan | Catatan |
|---|---|---|
| Bahasa backend | Go | Wajib sesuai soal tes |
| HTTP router | `go-chi/chi/v5` | Ringan, kompatibel dengan `net/http` |
| Database | SQLite | Satu file, tanpa setup |
| Driver SQLite | `modernc.org/sqlite` | Pure Go, **tanpa CGO** — JANGAN ganti ke `mattn/go-sqlite3` |
| Akses DB | `database/sql` | Tanpa ORM; satu tabel tidak membutuhkannya |
| Validasi backend | `go-playground/validator/v10` | Validasi struct berbasis tag |
| Hash password | `golang.org/x/crypto/bcrypt` | Paket semi-resmi Go; JANGAN simpan password dalam bentuk lain |
| Auth | Session di server + cookie HttpOnly | Bukan JWT; session bisa dicabut dengan menghapus baris di DB |
| Frontend | React + Vite + TypeScript | Wajib sesuai soal tes; `strict: true` |
| Map | Leaflet + `react-leaflet` + tile OpenStreetMap | Gratis, tanpa API key |
| Server state | `@tanstack/react-query` | Caching, refetch otomatis setelah mutasi |
| Form | `react-hook-form` + `zod` | Validasi berbasis schema yang typed, error per field |
| Styling | Tailwind CSS | |

Jangan menambah dependency tanpa alasan yang jelas. Jika ada yang ditambahkan, tuliskan alasannya di README.md pada bagian "Alasan Pemilihan Library".

## Struktur Repository

```
/
├── backend/
│   ├── cmd/server/main.go        # entrypoint: config, init DB, migrate, seed, router
│   ├── internal/
│   │   ├── model/                # struct Entity, User, enum (satu-satunya sumber kebenaran)
│   │   ├── handler/              # HTTP handler + middleware auth: decode, validasi, response
│   │   ├── service/              # logika bisnis
│   │   ├── repository/           # query SQL
│   │   ├── validation/           # setup validator + format error
│   │   └── database/             # koneksi, migrasi, seed
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── api/                  # fetch client + fungsi API yang typed
│   │   ├── components/           # komponen UI umum
│   │   ├── features/auth/        # form login/register, hook user saat ini
│   │   ├── features/entities/    # map, marker, form, detail panel, hooks
│   │   ├── schemas/              # schema zod
│   │   └── types/                # tipe TypeScript bersama
│   └── vite.config.ts            # proxy /api -> http://localhost:8080
├── README.md
└── CLAUDE.md
```

Aturan layering (backend): `handler → service → repository`. Handler tidak pernah menjalankan SQL; repository tidak tahu apa pun soal HTTP.

## Model Data

Tabel `entities`:

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | TEXT (UUID v4) | dibuat oleh backend |
| `name` | TEXT | wajib, di-trim, 1–100 karakter |
| `type` | TEXT | wajib, harus salah satu dari type yang diizinkan |
| `status` | TEXT | wajib, harus salah satu dari status yang diizinkan |
| `latitude` | REAL | wajib, -90 ≤ lat ≤ 90 |
| `longitude` | REAL | wajib, -180 ≤ lng ≤ 180 |
| `description` | TEXT | opsional, maks. 500 karakter |
| `attributes` | TEXT (objek JSON) | opsional, jika diisi harus berupa objek JSON yang valid |
| `created_at` | TEXT (RFC3339, UTC) | diisi oleh backend |
| `updated_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Tabel `users`:

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | TEXT (UUID v4) | dibuat oleh backend |
| `email` | TEXT UNIQUE | wajib, di-trim + lowercase, format email valid, maks. 254 karakter |
| `password_hash` | TEXT | hash bcrypt; **tidak pernah** dikirim ke client |
| `role` | TEXT | `user` atau `admin` (CHECK constraint) |
| `created_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Tabel `sessions`:

| Kolom | Tipe | Aturan |
|---|---|---|
| `token_hash` | TEXT PK | SHA-256 (hex) dari token acak 32 byte; token mentah hanya ada di cookie |
| `user_id` | TEXT | FK ke `users.id`, `ON DELETE CASCADE` |
| `expires_at` | TEXT (RFC3339, UTC) | 7 hari setelah login; session kedaluwarsa dianggap tidak ada |
| `created_at` | TEXT (RFC3339, UTC) | diisi oleh backend |
| `last_seen_at` | TEXT (RFC3339, UTC), nullable | request terakhir dengan session ini; diisi saat login, diperbarui middleware **paling sering sekali per menit**. `NULL` untuk session lama dari sebelum kolom ini ada |

Migrasi kolom: `CREATE TABLE IF NOT EXISTS` tidak menambah kolom ke tabel lama, jadi migrasi wajib mengecek `PRAGMA table_info(sessions)` dan menjalankan `ALTER TABLE ... ADD COLUMN` jika `last_seen_at` belum ada. DB yang sudah ada tidak boleh perlu dihapus.

Tabel `entity_photos`:

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | TEXT (UUID v4) | dibuat oleh backend; juga nama file di disk (`<id>.<ext>`) |
| `entity_id` | TEXT | FK ke `entities.id`, `ON DELETE CASCADE` |
| `content_type` | TEXT | `image/jpeg`, `image/png`, atau `image/webp`, hasil deteksi isi file |
| `size_bytes` | INTEGER | ukuran file |
| `created_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

File foto disimpan di `UPLOAD_DIR` (default `./data/uploads`), **bukan** di database. Nama file selalu dibuat server; nama asli dari user tidak pernah dipakai sebagai path. Menghapus entitas atau foto juga menghapus file-nya di disk.

Tabel `facility_installations` (hanya untuk entitas yang type-nya punya kemampuan `installation`, saat ini `facility` dan `iot_device`; nama tabel tetap karena awalnya hanya untuk fasilitas):

| Kolom | Tipe | Aturan |
|---|---|---|
| `entity_id` | TEXT PK | FK ke `entities.id`, `ON DELETE CASCADE`; satu data pemasangan per entitas |
| `started_on` | TEXT (`YYYY-MM-DD`) | wajib; tanggal mulai pemasangan |
| `target_on` | TEXT (`YYYY-MM-DD`) | wajib; target selesai, ≥ `started_on` |
| `completed_on` | TEXT (`YYYY-MM-DD`), nullable | tanggal selesai sebenarnya; ≥ `started_on` dan tidak boleh di masa depan |
| `updated_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Tanggal disimpan **tanpa jam** karena durasi dihitung per hari. "Hari ini" dihitung di zona waktu `APP_TIMEZONE` (default `Asia/Jakarta`), bukan UTC, supaya status tidak berubah jam 07.00 WIB.

Tabel `sensor_configs` (entitas dengan kemampuan `readings`, saat ini `iot_device`):

| Kolom | Tipe | Aturan |
|---|---|---|
| `entity_id` | TEXT PK | FK ke `entities.id`, `ON DELETE CASCADE`; satu sensor (satu metric) per perangkat |
| `metric` | TEXT | salah satu metric di model (`temperature`, `water_level`, `wind_speed`) |
| `api_key_hash` | TEXT UNIQUE, nullable | SHA-256 (hex) dari API key perangkat; key mentah hanya ditampilkan **sekali** saat dibuat |
| `key_created_at` | TEXT (RFC3339, UTC), nullable | kapan key terakhir dibuat |
| `updated_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Tabel `sensor_readings`:

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | |
| `entity_id` | TEXT | FK ke `entities.id`, `ON DELETE CASCADE`; index `(entity_id, recorded_at)` |
| `metric` | TEXT | metric saat data diterima; data metric lama tidak ditampilkan setelah metric diganti |
| `value` | REAL | dalam rentang metric |
| `recorded_at` | TEXT (RFC3339, UTC) | waktu pengukuran (dari perangkat, atau waktu terima jika tidak dikirim) |
| `received_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Data yang `recorded_at`-nya lebih dari **7 hari** dihapus otomatis.

Tabel `geofences` (zona operasional; entitas dengan kemampuan `geofence`, saat ini `vehicle`):

| Kolom | Tipe | Aturan |
|---|---|---|
| `entity_id` | TEXT PK | FK ke `entities.id`, `ON DELETE CASCADE`; satu zona (lingkaran) per entitas |
| `center_latitude` | REAL | wajib, −90 ≤ lat ≤ 90 |
| `center_longitude` | REAL | wajib, −180 ≤ lng ≤ 180 |
| `radius_m` | REAL | wajib, 100 ≤ radius ≤ 50 000 (meter) |
| `updated_at` | TEXT (RFC3339, UTC) | diisi oleh backend |

Posisi entitas **tidak** dibatasi oleh zonanya: pin boleh berada di luar zona dan hanya ditandai "outside" (keputusan developer, supaya data tetap jujur dan cocok untuk live tracking nanti, di mana kendaraan asli bisa saja keluar zona).

### Enum — satu sumber kebenaran

Nilai yang diizinkan **hanya** didefinisikan di `backend/internal/model`:
- `type`: `vehicle`, `iot_device`, `facility` (akan ada penambahan type nantinya)
- `status`: `active`, `inactive`, `maintenance`
- `role`: `user`, `admin` (tidak diekspos lewat `/api/meta`; register selalu membuat `user`)
- **Kemampuan per type** (`capabilities`): fitur khusus yang dimiliki type tertentu. Saat ini `facility` dan `iot_device` → `installation` (IoT ditambahkan atas keputusan developer); nanti `iot_device` juga → `readings`, dan `vehicle` → `tracking`. Dikirim lewat `GET /api/meta`, jadi frontend memeriksa `meta.capabilities[type]` dan **tidak** meng-hardcode "facility".
- `iot_device` juga punya kemampuan `readings` (data sensor).
- `vehicle` punya kemampuan `geofence` (zona operasional berbentuk lingkaran).
- `metric` (jenis sensor), dikirim lewat `GET /api/meta` sebagai `"metrics": [{ "id", "label", "unit", "min", "max" }]`:
  - `temperature` — Temperature, °C, −50 … 80
  - `water_level` — Water level, cm, 0 … 2000
  - `wind_speed` — Wind speed, m/s, 0 … 100
- `installation_status` (dihitung, tidak disimpan): `scheduled` (mulai > hari ini), `in_progress` (belum selesai, hari ini ≤ target), `overdue` (belum selesai, hari ini > target), `completed_on_time` (selesai ≤ target), `completed_late` (selesai > target)

Frontend **tidak boleh** meng-hardcode daftar ini untuk dropdown. Frontend mengambilnya dari `GET /api/meta`. Menambah type baru cukup dengan mengubah daftar konstanta di Go. (Label/warna marker di frontend boleh punya fallback untuk nilai yang tidak dikenal.)

## Kontrak API

Base path: `/api`. Request dan response dalam format JSON.

| Method | Path | Deskripsi | Sukses |
|---|---|---|---|
| GET | `/api/meta` | Daftar type dan status yang diizinkan | 200 |
| GET | `/api/entities` | List entitas, filter opsional `?type=&status=` | 200 |
| GET | `/api/entities/{id}` | Detail entitas | 200 |
| POST | `/api/entities` | Membuat entitas | 201 |
| PUT | `/api/entities/{id}` | Update penuh (semua field yang bisa diubah) | 200 |
| PATCH | `/api/entities/{id}/location` | Update hanya `latitude` + `longitude` (drag marker) | 200 |
| DELETE | `/api/entities/{id}` | Menghapus entitas | 204 |
| GET | `/api/entities/{id}/photos` | Daftar foto entitas, terlama dulu | 200 |
| POST | `/api/entities/{id}/photos` | Upload satu foto (`multipart/form-data`, field `photo`) | 201 |
| GET | `/api/photos/{photoId}` | File gambar (bukan JSON) | 200 |
| DELETE | `/api/photos/{photoId}` | Menghapus foto | 204 |
| GET | `/api/entities/{id}/installation` | Data pemasangan; `{ "data": null }` jika belum diisi | 200 |
| PUT | `/api/entities/{id}/installation` | Isi/ubah data pemasangan `{ "started_on", "target_on", "completed_on" }` | 200 |
| DELETE | `/api/entities/{id}/installation` | Menghapus data pemasangan | 204 |
| GET | `/api/installations` | Semua data pemasangan (untuk dashboard), terlambat dulu | 200 |
| GET | `/api/entities/{id}/sensor` | Konfigurasi sensor; `{ "data": null }` jika belum diisi | 200 |
| PUT | `/api/entities/{id}/sensor` | Pilih metric `{ "metric" }` | 200 |
| DELETE | `/api/entities/{id}/sensor` | Hapus konfigurasi + API key (data lama tetap sampai kedaluwarsa) | 204 |
| POST | `/api/entities/{id}/sensor/key` | Buat (atau ganti) API key perangkat; key lama langsung tidak berlaku | 201 |
| GET | `/api/entities/{id}/readings?hours=24` | Data sensor `hours` jam terakhir (1–168, default 24), terlama dulu | 200 |
| POST | `/api/devices/{id}/readings` | **Dipanggil oleh perangkat**, bukan browser: `Authorization: Bearer <api key>`, body `{ "value", "recorded_at"? }` | 201 |

Objek sensor: `{ "entity_id", "metric", "has_api_key", "key_created_at", "updated_at" }` (hash key tidak pernah dikirim). Respons `POST .../sensor/key`: `{ "data": { "api_key": "gem_…", "key_created_at" } }`; ini **satu-satunya** saat key mentah terlihat.

Respons readings: `{ "data": { "metric": { …spec }, "readings": [ { "value", "recorded_at" } ], "latest": { … } | null } }`, hanya data dengan metric yang sedang aktif.

Endpoint perangkat:
- Key salah/tidak ada, atau key milik perangkat lain → **401 `unauthorized`** dengan pesan yang sama ("invalid device key"), tanpa membedakan penyebabnya.
- `value` wajib, berupa angka, dalam rentang metric → 422 `fields.value` (`"is required"`, `"must be between -50 and 80"`).
- `recorded_at` opsional, RFC3339, tidak boleh lebih dari 5 menit ke depan dan tidak lebih tua dari 7 hari → 422 `fields.recorded_at`.
- Endpoint sensor/readings untuk type tanpa kemampuan `readings` → 400 `invalid_request`; key dibuat sebelum metric dipilih → 400 `invalid_request` "choose the sensor metric first".

Zona operasional:

| Method | Path | Deskripsi | Sukses |
|---|---|---|---|
| GET | `/api/entities/{id}/geofence` | Zona entitas; `{ "data": null }` jika belum diisi | 200 |
| PUT | `/api/entities/{id}/geofence` | Isi/ubah zona `{ "center_latitude", "center_longitude", "radius_m" }` | 200 |
| DELETE | `/api/entities/{id}/geofence` | Hapus zona | 204 |
| GET | `/api/geofences` | Semua zona (untuk lingkaran di map dan dashboard), yang di luar zona dulu | 200 |

Objek geofence:

```json
{
  "entity_id": "…",
  "center_latitude": -6.2088,
  "center_longitude": 106.8456,
  "radius_m": 5000,
  "distance_m": 1234.5,
  "inside": true,
  "updated_at": "2026-10-06T07:00:00Z"
}
```

- `distance_m` = jarak dari pusat zona ke posisi entitas **saat ini** (rumus haversine, jari-jari bumi 6 371 008.8 m), dibulatkan 0.1 m. `inside` = `distance_m ≤ radius_m`. Keduanya **dihitung saat diminta**, tidak disimpan, jadi selalu sesuai posisi terakhir (termasuk setelah drag).
- Pengecekan zona ada di satu fungsi di backend supaya live tracking nanti memakai fungsi yang sama dengan sumber posisi yang berbeda.
- Endpoint geofence untuk type tanpa kemampuan `geofence` → 400 `invalid_request` "this entity type has no operating zone".

**Simulator data dummy:** jika `SIMULATE_SENSORS` bukan `false` (default aktif), backend mengirim satu nilai per menit untuk setiap perangkat yang sudah punya metric, lewat **service dan validasi yang sama** dengan endpoint perangkat. Nilainya naik-turun secara wajar di dalam rentang metric. Perangkat yang belum punya data 24 jam terakhir diisi riwayat 24 jam (per 15 menit) supaya grafik langsung terisi. Simulator juga menghapus data > 7 hari.

`GET /api/meta` kini juga mengirim `"capabilities": { "iot_device": ["installation"], "facility": ["installation"] }` (field tambahan, field lama tidak berubah).

Objek installation:

```json
{
  "entity_id": "…",
  "started_on": "2026-10-01",
  "target_on": "2026-10-20",
  "completed_on": null,
  "status": "in_progress",
  "planned_days": 19,
  "elapsed_days": 5,
  "days_late": 0,
  "updated_at": "2026-10-06T07:00:00Z"
}
```

- `planned_days` = target − mulai. `elapsed_days` = (selesai, atau hari ini jika belum selesai) − mulai, minimal 0. `days_late` = berapa hari melewati target (0 jika tidak terlambat).
- Endpoint installation untuk entitas yang type-nya **tidak** punya kemampuan `installation` → **400 `invalid_request`** "this entity type has no installation data".

Objek foto: `{ "id", "entity_id", "url", "content_type", "size_bytes", "created_at" }`, dengan `url` = `/api/photos/{id}`. Bentuk objek entitas **tidak berubah**.

`GET /api/photos/{photoId}` mengirim header `Content-Type` dari DB, `X-Content-Type-Options: nosniff`, dan `Cache-Control: private, max-age=86400` (foto tidak pernah diubah, hanya dihapus).

### Auth

| Method | Path | Body | Sukses | Catatan |
|---|---|---|---|---|
| POST | `/api/auth/register` | `{ "email", "password" }` | 201 `{ "data": user }` | role selalu `user`; langsung login (cookie dipasang) |
| POST | `/api/auth/login` | `{ "email", "password" }` | 200 `{ "data": user }` | cookie dipasang |
| POST | `/api/auth/logout` | – | 204 | session dihapus, cookie dikosongkan; tetap 204 walau belum login |
| GET | `/api/auth/me` | – | 200 `{ "data": user }` | 401 jika belum login |

Objek user: `{ "id", "email", "role", "created_at" }`.

Cookie `session`: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Max-Age` 7 hari. `Secure` diatur oleh env `COOKIE_SECURE` (default `false` untuk localhost).

### Hak akses

Otorisasi **wajib** dilakukan di middleware backend. Frontend hanya menyembunyikan kontrol, bukan pengaman.

| Endpoint | Belum login | `user` | `admin` |
|---|---|---|---|
| `/api/auth/*` | ✅ | ✅ | ✅ |
| `GET /api/meta`, `GET /api/entities`, `GET /api/entities/{id}` | 401 | ✅ | ✅ |
| `POST`, `PUT`, `PATCH .../location`, `DELETE` pada `/api/entities` | 401 | 403 | ✅ |
| `GET /api/admin/stats` | 401 | 403 | ✅ |
| `GET /api/entities/{id}/photos`, `GET /api/photos/{photoId}` | 401 | ✅ | ✅ |
| `POST /api/entities/{id}/photos`, `DELETE /api/photos/{photoId}` | 401 | 403 | ✅ |
| `GET /api/entities/{id}/installation`, `GET /api/installations` | 401 | ✅ | ✅ |
| `PUT`, `DELETE /api/entities/{id}/installation` | 401 | 403 | ✅ |
| `GET /api/entities/{id}/sensor`, `GET /api/entities/{id}/readings` | 401 | ✅ | ✅ |
| `PUT`, `DELETE /api/entities/{id}/sensor`, `POST .../sensor/key` | 401 | 403 | ✅ |
| `POST /api/devices/{id}/readings` | API key perangkat (bukan session) | – | – |
| `GET /api/entities/{id}/geofence`, `GET /api/geofences` | 401 | ✅ | ✅ |
| `PUT`, `DELETE /api/entities/{id}/geofence` | 401 | 403 | ✅ |

### Statistik admin

`GET /api/admin/stats` → 200:

```json
{
  "data": {
    "users": {
      "total": 12,
      "by_role": { "user": 10, "admin": 2 },
      "with_active_session": 5,
      "online": 2,
      "active_users": [
        { "id": "…", "email": "budi@example.com", "last_seen_at": "2026-10-06T07:00:00Z", "online": true }
      ]
    },
    "online_window_minutes": 5
  }
}
```

- `total`, `with_active_session`, dan `online` **hanya menghitung role `user`**. Admin adalah pembaca dashboard, jadi tidak ikut dihitung supaya angkanya tidak ambigu (keputusan developer).
- `total`: user ber-role `user` yang terdaftar.
- `by_role`: berisi **setiap** role di `model.Roles` (termasuk `admin`), termasuk yang jumlahnya 0.
- `with_active_session`: jumlah **user** (bukan session) yang punya minimal satu session belum kedaluwarsa.
- `online`: jumlah user dengan session belum kedaluwarsa yang `last_seen_at`-nya dalam `online_window_minutes` terakhir.
- `active_users`: siapa saja di balik `with_active_session` (hanya role `user`), diurutkan dari yang terakhir aktif, maksimal 50. `last_seen_at` adalah aktivitas terbaru dari semua session-nya (`null` untuk session lama tanpa catatan), `online` memakai aturan 5 menit yang sama. Hanya email yang dikirim; hash password dan token tidak pernah ikut (keputusan developer, menggantikan "daftar nama user yang online" yang sebelumnya di luar scope).
- Batas online (5 menit) adalah konstanta di backend dan dikirim di response, jadi frontend tidak meng-hardcode angkanya.
- Statistik entitas **tidak** ada di endpoint ini; frontend menghitungnya dari `GET /api/entities` yang sudah boleh diakses kedua role.

### Format response

Sukses (tunggal): `{ "data": { ...entity } }`
Sukses (list): `{ "data": [ ... ] }`

Error validasi — **422**:
```json
{
  "error": "validation_failed",
  "fields": {
    "latitude": "must be between -90 and 90",
    "name": "is required"
  }
}
```

Error lainnya: `{ "error": "<kode>", "message": "<pesan yang mudah dibaca>" }`
- 400 `invalid_json` — body rusak atau ada field yang tidak dikenal
- 401 `unauthorized` — belum login, atau session tidak valid / kedaluwarsa
- 401 `invalid_credentials` — email atau password salah (pesan dibuat umum, tidak membedakan email tidak terdaftar vs password salah)
- 403 `forbidden` — sudah login tapi role tidak cukup
- 404 `not_found` — entitas tidak ada / id tidak valid
- 500 `internal_error` — jangan pernah membocorkan detail internal

Email yang sudah terdaftar saat register → **422** dengan `fields.email: "is already registered"`.

Upload foto yang tidak valid → **422** di `fields.photo`: `"is required"`, `"must be a JPEG, PNG or WebP image"`, `"must be at most 5 MB"`, atau `"this entity already has the maximum of 5 photos"`. Body upload yang bukan `multipart/form-data` yang valid → **400 `invalid_request`** (kode baru, karena `invalid_json` tidak tepat untuk upload file). Entitas atau foto yang tidak ada → 404 `not_found`.

Key di dalam `fields` memakai nama field JSON (snake_case), supaya frontend bisa langsung memetakan error ke input form.

## Aturan Validasi

Validasi wajib ada di **kedua sisi** dengan **aturan yang sama**. Backend adalah otoritas; validasi frontend untuk kebutuhan UX.

Backend:
- Decode dengan `DisallowUnknownFields`
- Trim string sebelum divalidasi
- Tolak koordinat `NaN`/`Inf`
- Validasi `type`/`status` terhadap enum di model
- `attributes` harus berupa objek JSON (bukan array/string/angka)
- `email`: trim + lowercase, format email valid, maks. 254 karakter
- `password` saat register: 8–72 karakter (72 = batas byte bcrypt; hitung dalam byte); **tidak di-trim**
- `password` saat login: cukup wajib diisi (aturan panjang tidak dicek supaya tidak membocorkan info)
- Installation: `started_on` dan `target_on` wajib, format `YYYY-MM-DD` dan tanggal yang benar-benar ada (`2026-02-30` ditolak); `target_on` ≥ `started_on`; `completed_on` opsional, ≥ `started_on`, dan tidak boleh setelah hari ini. Pesan 422: `"is required"`, `"must be a date (YYYY-MM-DD)"`, `"must be on or after the start date"`, `"cannot be in the future"`
- Geofence: `center_latitude`/`center_longitude` wajib dengan aturan yang sama seperti koordinat entitas (rentang, bukan NaN/Inf); `radius_m` wajib, angka, 100–50 000. Pesan 422: `"is required"`, `"must be between -90 and 90"`, `"must be between 100 and 50000"`
- Foto: jenis file ditentukan dari **isi file** (`http.DetectContentType`), bukan dari ekstensi atau header `Content-Type` kiriman client; maks. 5 MB (`MaxBytesReader`); maks. 5 foto per entitas
- Jangan pernah berasumsi frontend sudah melakukan validasi

Frontend:
- Schema zod di `src/schemas/` mengikuti aturan backend
- Tampilkan error per field langsung di bawah input
- Petakan `fields` dari response 422 backend ke field form yang sesuai

## Perilaku Dashboard

- Header punya tab **Map | Dashboard** (state biasa, tanpa router). Pindah tab tidak menghapus pilihan atau form di map.
- Kedua role melihat **ringkasan entitas**: total, jumlah per status (warna sama dengan marker), dan jumlah per type. Status/type yang jumlahnya 0 tetap ditampilkan, diambil dari `GET /api/meta`.
- Hanya `admin` yang melihat **statistik user** dari `GET /api/admin/stats`. Frontend tidak memanggil endpoint ini untuk role `user`.
- Statistik user di-refetch tiap 30 detik selama tab Dashboard terbuka dan tab browser terlihat.
- **Heartbeat:** selama tab browser terlihat, frontend memanggil `GET /api/auth/me` tiap 2 menit supaya user yang membuka app tapi diam tetap terhitung online.
- Label di UI harus jujur (UI berbahasa Inggris): "Online (last 5 min)" dan "With an active session", bukan "logged in now".
- Daftar entitas di dashboard (semua entitas di kartu Total, entitas yang dilacak pemasangannya, entitas di luar zona) **bisa diklik**: pindah ke tab Map, map terbang ke pin, dan panel detail terbuka. Jika sedang ada form tambah/edit, form itu tidak diganti (tidak ada data yang hilang); map tetap terbang ke pin dan muncul pemberitahuan (keputusan developer).

## Perilaku Pemasangan Fasilitas

- Form New/Edit entity menampilkan bagian **Installation** hanya jika type yang dipilih punya kemampuan `installation` (dari `/api/meta`). Sama seperti foto, perubahan diproses saat Create/Save dan dibatalkan oleh Cancel. Jika type diganti ke type tanpa kemampuan itu, bagian tersebut disembunyikan dan tidak dikirim.
- Panel detail (semua role) menampilkan badge status, tanggal, dan progress (`elapsed_days` / `planned_days`), plus "terlambat N hari" jika ada.
- Dashboard (semua role): jumlah per status pemasangan dan daftar entitas (fasilitas maupun perangkat IoT) yang `overdue`.

## Perilaku Sensor IoT

- Form New/Edit menampilkan bagian **Sensor** hanya untuk type dengan kemampuan `readings` (dari `/api/meta`). Admin memilih metric (dari `meta.metrics`); disimpan saat Create/Save seperti bagian lain.
- Di form Edit, admin bisa menekan **Generate API key** (langsung dikirim, karena key harus ditampilkan sekali dan disalin). Key ditampilkan dengan tombol salin dan peringatan bahwa key tidak akan ditampilkan lagi. Generate ulang mematikan key lama.
- Panel detail (semua role): nilai terakhir + satuan + "N min ago", grafik garis 24 jam (SVG, tanpa library), refetch tiap 30 detik selama panel terbuka.

## Perilaku Zona Operasional

- Form New/Edit menampilkan bagian **Operating zone** hanya untuk type dengan kemampuan `geofence` (dari `/api/meta`). Isinya checkbox "Limit to an operating zone", radius, dan pusat zona. Pusat default = posisi entitas di form; bisa diisi angka, tombol "Use pin position", atau tombol **"Pick on map"** lalu klik map (klik map berikutnya mengisi pusat zona, bukan membuat entitas baru). Disimpan saat Create/Save, Cancel membatalkan.
- Map menggambar **lingkaran zona** untuk setiap entitas yang punya zona: garis putus-putus abu-abu jika di dalam, merah jika di luar. Saat form terbuka, lingkaran mengikuti isian form (pratinjau) sebelum disimpan.
- Drag pin ke luar zona **tetap diizinkan**; setelah berhasil, toast peringatan "… is now outside its operating zone".
- Panel detail (semua role): radius, dan "Inside (1.2 km from center)" atau badge merah "Outside zone by 800 m".
- Dashboard (semua role): jumlah entitas di luar zona dan daftarnya.

## Bahasa (ID | EN)

- UI tersedia dalam **English** dan **Bahasa Indonesia**. Pilihan ada di bar atas dan di halaman login (tombol `ID | EN`), disimpan di `localStorage` (dibungkus try/catch), default mengikuti bahasa browser (`id*` → ID, selain itu EN). `<html lang>` ikut diganti.
- **Tanpa library**: kamus di `frontend/src/i18n/` (`en.ts` sebagai acuan, `id.ts` wajib punya key yang sama, dicek TypeScript). Teks UI tidak boleh ditulis langsung di komponen; ambil dari kamus.
- **Backend dan kontrak API tidak berubah.** Pesan validasi tetap fragmen bahasa Inggris (`"is required"`, `"must be between -90 and 90"`); frontend menerjemahkannya saat ditampilkan dengan tabel pola yang dikenal. Pesan yang tidak dikenal ditampilkan apa adanya (bahasa Inggris), jadi tidak ada error yang hilang.
- Nilai dari backend (type, status, role, metric, status pemasangan) diterjemahkan jika ada di kamus; nilai baru yang belum ada memakai label cadangan seperti sekarang.
- Tanggal dan angka diformat dengan locale bahasa yang dipilih (`id-ID` / `en-US`).
- Data milik user (nama entitas, deskripsi, attributes) **tidak** diterjemahkan.

## Perilaku Map

- Klik area kosong di map → buka form tambah dengan lat/lng terisi otomatis
- Klik marker → tampilkan detail (popup atau side panel) dengan aksi Edit / Hapus
- Drag marker → `PATCH /location`
  - Optimistic update; jika gagal, kembalikan marker ke posisi semula dan tampilkan toast error
- Hapus wajib melalui dialog konfirmasi
- Warna marker mencerminkan status
- **Kartu pencarian** (di bawah legenda): cari nama (tanpa beda huruf besar/kecil dan aksen) dan filter status. Hasilnya ditampilkan sebagai daftar yang bisa diklik (peta terbang ke pin + panel detail terbuka, Enter membuka hasil pertama, Escape menghapus pencarian), dan selama pencarian/filter aktif **hanya pin yang cocok yang tampil di peta** (keputusan developer). Pin yang sedang dipilih tetap tampil. Disaring di frontend dari list yang sudah dimuat; tidak ada perubahan API.

**Jebakan yang perlu diwaspadai:** Leaflet bisa mengembalikan longitude di luar rentang -180..180 ketika map digeser melewati batas dunia. Selalu normalisasi longitude (atau batasi dengan `maxBounds` / `noWrap`) sebelum dikirim ke backend, jika tidak, validasi backend akan menolaknya.

## Menjalankan Secara Lokal

Kebutuhan: Go 1.26+, Node.js 20+.

```bash
# backend (http://localhost:8080)
cd backend
go run ./cmd/server

# frontend (http://localhost:5173)
cd frontend
npm install
npm run dev
```

- Backend membuat file SQLite dan tabel secara otomatis saat start (auto-migrate).
- Seed data hanya dimasukkan jika tabel masih kosong.
- Vite mem-proxy `/api` ke backend, jadi tidak perlu setup CORS saat development.

Konfigurasi lewat environment variable dengan nilai default: `PORT=8080`, `DB_PATH=./data/app.db`, `COOKIE_SECURE=false`, `UPLOAD_DIR=./data/uploads`, `APP_TIMEZONE=Asia/Jakarta`, `SIMULATE_SENSORS=true`.

Admin pertama: `ADMIN_EMAIL` + `ADMIN_PASSWORD`. Saat start, jika email tersebut belum terdaftar, backend membuat user ber-role `admin`. Jika env tidak diisi, server tetap jalan dan mencatat peringatan di log. Nilai ini **tidak boleh** di-commit ke repository.

## Testing

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run lint
```

Ekspektasi minimal:
- Unit test validasi backend (input valid, tiap kasus tidak valid, nilai batas seperti lat = 90 / 90.0001)
- Test handler untuk status code (201, 422, 404, 204)
- Test auth: register/login/logout/me, email duplikat (422), kredensial salah (401), session kedaluwarsa (401), serta hak akses tiap role (401/403/sukses)
- Test dashboard: migrasi menambah `last_seen_at` ke tabel `sessions` lama tanpa kehilangan data, `last_seen_at` diperbarui maksimal sekali per menit, perhitungan `stats` (user dengan beberapa session dihitung sekali, session kedaluwarsa dan di luar batas online tidak dihitung), hak akses `/api/admin/stats` (401/403/200)
- Test foto: upload JPEG/PNG/WebP (201), file teks yang diberi nama `.jpg` (422), 5 MB vs 5 MB + 1 byte, foto ke-6 (422), entitas tidak ada (404), file ikut terhapus saat foto/entitas dihapus, header `nosniff` saat file diambil, hak akses tiap role
- Test installation: perhitungan status di setiap batas (mulai besok → scheduled; target hari ini → in_progress; target kemarin → overdue; selesai tepat di target → completed_on_time; sehari setelah target → completed_late) dengan jam palsu, validasi tanggal (format, tanggal tidak ada, urutan, masa depan), type tanpa kemampuan (400), entitas tidak ada (404), hapus entitas ikut menghapus data pemasangan, hak akses tiap role
- Test sensor: endpoint perangkat dengan key benar (201), key salah/kosong/milik perangkat lain (401, pesan sama), key lama setelah generate ulang (401), nilai di batas rentang (−50 vs −50.1), `recorded_at` di masa depan/terlalu lama (422), key sebelum metric dipilih (400), type tanpa kemampuan (400), readings hanya metric aktif dan urut waktu, penghapusan data > 7 hari, simulator menulis lewat validasi yang sama dan tetap di dalam rentang, hak akses tiap role
- Test geofence: rumus jarak dengan titik yang diketahui (Monas → Bundaran HI ≈ 2.2 km, toleransi kecil), batas `inside` (jarak = radius → inside, sedikit lebih → outside), radius 100 vs 99.9 dan 50 000 vs 50 000.1, koordinat pusat di batas, `inside` berubah setelah entitas dipindah (PATCH location), type tanpa kemampuan (400), entitas tidak ada (404), hapus entitas ikut menghapus zona, urutan daftar (di luar zona dulu), hak akses tiap role

## Konvensi Kode

Go:
- Bersih dari `gofmt` / `go vet`
- Kembalikan error, jangan panic; bungkus dengan konteks (`fmt.Errorf("...: %w", err)`)
- Gunakan `context.Context` dari request di pemanggilan repository
- Hanya SQL dengan parameter — jangan pernah menyusun query dengan konkatenasi string

TypeScript / React:
- `strict: true`, tanpa `any`
- Server state dikelola React Query, bukan di state komponen
- Logika map ditempatkan di `features/entities/`; komponen dibuat kecil-kecil

Git:
- Commit kecil dan bermakna dengan gaya conventional (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`)
- Satu perubahan logis per commit

## Scope

MVP (wajib):
- [x] Backend CRUD + PATCH location + `/api/meta`
- [x] Validasi backend + test
- [x] Map dengan marker
- [x] Tambah via klik map, edit via form, pindah via drag, hapus dengan konfirmasi
- [x] Tampilan detail
- [x] Validasi frontend + pemetaan error dari backend
- [x] README (cara menjalankan, alasan pemilihan library, workflow AI, fitur yang belum selesai)

Nice to have (hanya setelah MVP selesai):
- [ ] Filter berdasarkan type / status (status + cari nama sudah ada di kartu pencarian peta; filter type belum)
- [ ] Sidebar daftar entitas yang tersinkron dengan map
- [ ] Clustering marker
- [ ] Update realtime (SSE/WebSocket)

Auth (branch `feat/auth`, setelah MVP):
- [x] Backend: tabel users/sessions, register/login/logout/me, seed admin dari env
- [x] Backend: middleware 401/403 + test
- [x] Frontend: tampilan login/register, logout, sembunyikan kontrol kelola untuk role `user`
- [x] README: kontrak auth, env admin, alasan bcrypt, keterbatasan

Di luar scope auth (catat sebagai keterbatasan): lupa/ganti password, kelola user (promote ke admin), rate limiting login.

Dashboard (branch `feat/dashboard`, setelah auth):
- [x] Backend: kolom `last_seen_at` + migrasi untuk DB lama, update di middleware (maks. sekali per menit) + test
- [x] Backend: `GET /api/admin/stats` (admin saja) + test
- [x] Frontend: tab Map | Dashboard, ringkasan entitas untuk kedua role
- [x] Frontend: statistik user untuk admin, heartbeat `/auth/me`
- [x] README: endpoint stats, arti "online", keterbatasan

Di luar scope dashboard (catat sebagai keterbatasan): status online realtime (WebSocket), grafik/riwayat aktivitas. (Daftar akun dengan sesi aktif sudah ditambahkan atas permintaan developer.)

Foto entitas (lanjutan di branch `feat/dashboard`, keputusan developer):
- [x] Backend: tabel `entity_photos`, penyimpanan file di `UPLOAD_DIR`, 4 endpoint foto + test
- [x] Frontend: galeri view-only di panel detail (semua role) + lihat foto besar; tambah/hapus foto di form New/Edit entity (admin), diproses saat Create/Save dan dibatalkan oleh Cancel (keputusan developer)
- [x] README: endpoint foto, `UPLOAD_DIR`, keterbatasan

Di luar scope foto (catat sebagai keterbatasan): resize/thumbnail otomatis, crop, urutan foto yang bisa diatur, keterangan (caption) per foto.

Durasi pemasangan fasilitas (branch `feat/dashboard`):
- [x] Backend: `capabilities` di `/api/meta`, tabel `facility_installations`, 4 endpoint + perhitungan status di `APP_TIMEZONE` + test
- [x] Frontend: bagian Installation di form New/Edit (hanya type dengan kemampuan `installation`), tampilan status di panel detail
- [x] Frontend: ringkasan status pemasangan + daftar overdue di dashboard
- [x] README: endpoint, arti status, zona waktu, keterbatasan

Di luar scope pemasangan (catat sebagai keterbatasan): riwayat perubahan tanggal, tahapan/milestone pemasangan, penanggung jawab/kontraktor, notifikasi saat terlambat.

Data sensor IoT (branch `feat/dashboard`; keputusan developer: simulator di dalam backend, satu metric per perangkat, simpan 7 hari, simulator tiap 1 menit):
- [x] Backend: metrics di `/api/meta`, tabel `sensor_configs` + `sensor_readings`, endpoint sensor/key/readings, endpoint perangkat dengan API key + test
- [x] Backend: simulator + pembersihan data lama + test
- [x] Frontend: bagian Sensor di form (metric + Generate API key), nilai terakhir + grafik 24 jam di panel detail
- [x] README: cara menghubungkan alat sungguhan (contoh `curl`), metric, simulator, keterbatasan

Di luar scope sensor (catat sebagai keterbatasan): beberapa metric per perangkat, ambang batas/alarm, notifikasi, rate limiting endpoint perangkat, MQTT/protokol IoT lain, kalibrasi.

Zona operasional kendaraan (branch `feat/dashboard`; keputusan developer: dikerjakan sebelum live tracking, lingkaran, pin di luar zona hanya ditandai, pusat bisa dipilih di map, radius 100 m – 50 km):
- [x] Backend: kemampuan `geofence`, tabel `geofences`, 4 endpoint, perhitungan jarak/inside + test
- [x] Frontend: bagian Operating zone di form (termasuk "Pick on map"), lingkaran zona di map + pratinjau saat form terbuka, toast saat drag ke luar zona
- [x] Frontend: status zona di panel detail + ringkasan di luar zona di dashboard
- [x] README: endpoint, perilaku, keterbatasan

Di luar scope zona (catat sebagai keterbatasan): zona poligon, beberapa zona per kendaraan, jadwal zona per jam, riwayat keluar-masuk zona, notifikasi.

Pilihan bahasa ID | EN (branch `feat/dashboard`, keputusan developer; frontend saja):
- [x] Kamus `en`/`id` + provider + tombol ID | EN (bar atas dan halaman login), disimpan di browser
- [x] Semua teks UI, label nilai backend, tanggal/angka mengikuti bahasa
- [x] Terjemahan pesan error backend/zod dengan tabel pola + fallback apa adanya
- [x] README: cara kerja, cara menambah teks/bahasa, keterbatasan

Di luar scope bahasa (catat sebagai keterbatasan): bahasa lain, terjemahan dari backend (Accept-Language), terjemahan data milik user.

Rencana berikutnya (belum dikerjakan, ditunda oleh developer): live tracking kendaraan (simulator + SSE, posisi di memori, kendaraan yang dilacak tidak bisa di-drag), memakai fungsi pengecekan zona yang sama untuk menandai/mencatat saat kendaraan keluar zona. Kemampuan per type dikirim lewat `GET /api/meta` supaya frontend tidak meng-hardcode type.

## Aturan untuk AI Agent

- Ikuti keputusan di file ini. Jika sebuah perubahan mengharuskan menyimpang darinya, berhenti dan tanyakan ke developer terlebih dahulu.
- Jangan menambah dependency, mengganti driver DB, atau mengubah kontrak API tanpa persetujuan.
- Nilai enum hanya didefinisikan di model backend.
- Tulis atau perbarui test bersamaan dengan setiap perubahan backend.
- Jangan menandai sebuah task selesai kecuali build berhasil dan test lulus.
- Jangan menulis secret atau kredensial ke dalam repository.
- Developer me-review semua kode yang dihasilkan dan mengambil keputusan desain akhir. Setiap fitur yang belum selesai atau diketahui bermasalah wajib dilaporkan agar bisa didokumentasikan di README.md.
