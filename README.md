# Geo Entity Map

Aplikasi web untuk menampilkan dan mengelola **entitas yang memiliki lokasi geografis** (kendaraan, perangkat IoT, fasilitas, dll.) di atas map.

- **Backend:** Go + chi + SQLite (`modernc.org/sqlite`, tanpa CGO)
- **Frontend:** React + Vite + TypeScript (strict) + Leaflet + React Query + react-hook-form/zod + Tailwind CSS

## Fitur

- Semua entitas tampil sebagai pin di map. Warna pin menunjukkan status (legenda di kartu kiri atas).
- **Tambah:** klik area kosong di map → form terbuka dengan lat/lng terisi. Selama form terbuka, klik titik lain atau geser pin hitam untuk mengubah lokasi.
- **Detail:** klik pin → panel detail (type, status, koordinat, deskripsi, attributes, timestamp).
- **Edit:** tombol *Edit* di panel detail → form dengan data entitas.
- **Pindah lokasi:** pilih pin, lalu drag → `PATCH /location` dengan *optimistic update*. Kalau gagal, pin kembali ke posisi semula dan muncul toast error. Setelah berhasil, toast menampilkan tombol **Undo**.
- **Hapus:** tombol *Delete* → dialog konfirmasi.
- **Validasi di kedua sisi** dengan aturan dan pesan yang sama. Error 422 dari backend dipetakan ke field form yang sesuai.
- Daftar type dan status **tidak di-hardcode** di frontend, melainkan diambil dari `GET /api/meta`.

## Menjalankan Secara Lokal

Kebutuhan: **Go 1.26+** dan **Node.js 20+**.

```bash
# Terminal 1 — backend (http://localhost:8080)
cd backend
go run ./cmd/server

# Terminal 2 — frontend (http://localhost:5173)
cd frontend
npm install
npm run dev
```

Buka **http://localhost:5173**.

- Backend membuat file SQLite dan tabel secara otomatis saat start (auto-migrate).
- 6 entitas contoh (Jakarta & Bandung) dimasukkan **hanya jika tabel masih kosong**.
- Vite mem-proxy `/api` ke backend, jadi tidak perlu setup CORS saat development.
- Tidak ada API key yang dibutuhkan (map memakai tile OpenStreetMap).

### Konfigurasi

| Env var | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP backend |
| `DB_PATH` | `./data/app.db` | Lokasi file SQLite (relatif terhadap folder `backend/`) |

Untuk mengulang dari data contoh, hentikan backend lalu hapus folder `backend/data/`.

## Testing

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run lint
```

- **Validasi backend** (`internal/validation`): test table-driven untuk input valid, tiap kasus tidak valid, dan nilai batas (lat `90` vs `90.0001`, lng `180` vs `180.0001`, NaN/Inf, panjang nama 100 vs 101 karakter termasuk multibyte, attributes non-objek).
- **Handler** (`internal/handler`): test HTTP terhadap SQLite sungguhan di folder sementara, mencakup status 200/201/204/400/404/405/422, filter list, update penuh, update lokasi, dan hapus.
- **Database** (`internal/database`): migrasi idempoten, seed hanya saat tabel kosong, constraint koordinat.

## API

Base path `/api`. Request dan response berformat JSON.

| Method | Path | Deskripsi | Sukses |
|---|---|---|---|
| GET | `/api/meta` | Daftar type dan status yang diizinkan | 200 |
| GET | `/api/entities` | List entitas, filter opsional `?type=&status=` | 200 |
| GET | `/api/entities/{id}` | Detail entitas | 200 |
| POST | `/api/entities` | Membuat entitas | 201 |
| PUT | `/api/entities/{id}` | Update penuh | 200 |
| PATCH | `/api/entities/{id}/location` | Update hanya `latitude` + `longitude` | 200 |
| DELETE | `/api/entities/{id}` | Menghapus entitas | 204 |

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
| 404 | `not_found` | Entitas tidak ada, ID bukan UUID, atau route tidak ada |
| 405 | `method_not_allowed` | Method tidak didukung untuk path tersebut |
| 422 | `validation_failed` | Validasi gagal (lihat `fields`) |
| 500 | `internal_error` | Detail internal hanya dicatat di log server, tidak dikirim ke client |

Keputusan tambahan yang tidak diatur di brief awal (disetujui developer):

- Field dengan **tipe salah** (misalnya `"latitude": "north"`) → **422** dengan pesan per field (`"must be a number"`), bukan 400, karena body-nya JSON yang valid. Angka di luar jangkauan float64 (misalnya `1e400`) → `"is out of range"`.
- Filter list yang tidak valid (misalnya `?type=rocket`) → **422**, bukan list kosong.
- `POST` mengembalikan header `Location: /api/entities/{id}`.
- `PUT` adalah update penuh: `description`/`attributes` yang tidak dikirim akan dikosongkan. `created_at` tidak berubah.

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
| `attributes` | opsional, harus **objek** JSON (`null` = kosong) |
| `created_at`, `updated_at` | RFC3339 UTC (presisi detik), diisi backend |

**Enum hanya didefinisikan di `backend/internal/model/entity.go`.** Untuk menambah type atau status cukup menambah konstanta di sana. Tidak perlu migrasi DB maupun perubahan frontend, karena label dan warna punya fallback untuk nilai yang belum dikenal.

## Struktur Repository

```
backend/
  cmd/server/main.go       entrypoint: config, DB, migrate, seed, router, graceful shutdown
  internal/model/          Entity, input, enum (satu-satunya sumber kebenaran)
  internal/validation/     validator + format error per field
  internal/handler/        HTTP: decode, validasi, response, router
  internal/service/        logika bisnis (UUID, timestamp)
  internal/repository/     query SQL berparameter
  internal/database/       koneksi, migrasi, seed
frontend/src/
  api/                     fetch client (ApiError) + fungsi API yang typed
  components/              ConfirmDialog, toast
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

### Frontend

| Library | Alasan |
|---|---|
| React + Vite + TypeScript | Vite memberi dev server cepat dan proxy `/api`. |
| Leaflet + `react-leaflet` | Library map gratis dan ringan, mendukung marker yang bisa di-drag, dengan API sederhana |
| Tile OpenStreetMap | Gratis dan **tanpa API key**, jadi reviewer bisa langsung menjalankan app. CARTO (sekarang butuh API key) dan Stadia sempat dicoba lalu tidak dipakai. |
| `@tanstack/react-query` | Server state: caching, refetch setelah mutasi, optimistic update + rollback untuk drag marker |
| `react-hook-form` + `zod` + `@hookform/resolvers` | Form performan dengan error per field. Schema zod dibangun dari `/api/meta` dan mengikuti aturan backend. |
| Tailwind CSS | Styling cepat dan konsisten tanpa file CSS terpisah per komponen |

Dialog konfirmasi dan toast dibuat sendiri (elemen `<dialog>` bawaan browser + context React) supaya **tidak menambah dependency**.

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

## Fitur yang Belum Selesai & Keterbatasan

**Belum dikerjakan (nice to have):**

- Filter type/status di UI. Backend dan hook `useEntities(filter)` sudah mendukung, tinggal komponen UI-nya.
- Sidebar daftar entitas yang tersinkron dengan map.
- Clustering marker (butuh dependency tambahan).
- Update realtime (SSE/WebSocket). Saat ini perubahan dari tab atau user lain baru terlihat saat refetch (misalnya saat window kembali difokus).
- **Login/register dengan role admin & user**: direncanakan sebagai pengembangan berikutnya di branch terpisah (user hanya bisa melihat, admin bisa mengelola).

**Keterbatasan yang diketahui:**

- **Tidak ada test otomatis di frontend.** Kualitas dijaga lewat `typecheck` + `lint` dan skrip pengecekan manual. Repository dan service backend tidak punya unit test terpisah, tetapi teruji lewat test handler yang memakai SQLite sungguhan.
- **Undo pindah lokasi** hanya tersedia selama toast tampil (~8 detik). Jika pin yang sama dipindah dua kali, Undo dari toast pertama mengembalikan ke posisi sebelum pemindahan pertama.
- Jika data di-refetch tepat saat pin sedang di-drag, pin bisa melompat ke posisi dari server. Kemungkinannya kecil karena refetch otomatis hanya terjadi saat window kembali difokus.
- **Last write wins:** tidak ada pengecekan konflik/versi saat dua user mengedit entitas yang sama.
- List entitas tidak dipaginasi. Cukup untuk ratusan entitas, belum untuk skala sangat besar.
- Backend tidak menyajikan file hasil build frontend dan tidak mengatur CORS. Untuk produksi, keduanya perlu disajikan dari origin yang sama (reverse proxy) atau CORS perlu ditambahkan.
- **Tile OpenStreetMap** tunduk pada [Tile Usage Policy](https://operations.osmfoundation.org/policies/tiles/) dan tidak ditujukan untuk trafik produksi tinggi. Untuk produksi, gunakan penyedia tile berbayar atau host tile sendiri.
- Vite memberi peringatan ukuran bundle > 500 kB (Leaflet + React dalam satu chunk). Belum dilakukan code splitting.
- Dua perbedaan kecil yang disengaja antara frontend dan backend:
  - JSON `attributes` yang rusak ditangkap frontend sebagai error field `"must be valid JSON"`, sedangkan jika dikirim langsung ke API hasilnya 400 `invalid_json` (sesuai kontrak).
  - `String.prototype.trim()` di JS memangkas beberapa karakter spasi Unicode (misalnya U+FEFF) yang tidak dipangkas `strings.TrimSpace` di Go. Frontend sedikit lebih ketat, jadi tidak menyebabkan error 422 yang tak terduga.
