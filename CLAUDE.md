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
│   │   ├── model/                # struct Entity, enum (satu-satunya sumber kebenaran)
│   │   ├── handler/              # HTTP handler: decode, validasi, response
│   │   ├── service/              # logika bisnis
│   │   ├── repository/           # query SQL
│   │   ├── validation/           # setup validator + format error
│   │   └── database/             # koneksi, migrasi, seed
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── api/                  # fetch client + fungsi API yang typed
│   │   ├── components/           # komponen UI umum
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

### Enum — satu sumber kebenaran

Nilai yang diizinkan **hanya** didefinisikan di `backend/internal/model`:
- `type`: `vehicle`, `iot_device`, `facility` (akan ada penambahan type nantinya)
- `status`: `active`, `inactive`, `maintenance`

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
- 404 `not_found` — entitas tidak ada / id tidak valid
- 500 `internal_error` — jangan pernah membocorkan detail internal

Key di dalam `fields` memakai nama field JSON (snake_case), supaya frontend bisa langsung memetakan error ke input form.

## Aturan Validasi

Validasi wajib ada di **kedua sisi** dengan **aturan yang sama**. Backend adalah otoritas; validasi frontend untuk kebutuhan UX.

Backend:
- Decode dengan `DisallowUnknownFields`
- Trim string sebelum divalidasi
- Tolak koordinat `NaN`/`Inf`
- Validasi `type`/`status` terhadap enum di model
- `attributes` harus berupa objek JSON (bukan array/string/angka)
- Jangan pernah berasumsi frontend sudah melakukan validasi

Frontend:
- Schema zod di `src/schemas/` mengikuti aturan backend
- Tampilkan error per field langsung di bawah input
- Petakan `fields` dari response 422 backend ke field form yang sesuai

## Perilaku Map

- Klik area kosong di map → buka form tambah dengan lat/lng terisi otomatis
- Klik marker → tampilkan detail (popup atau side panel) dengan aksi Edit / Hapus
- Drag marker → `PATCH /location`
  - Optimistic update; jika gagal, kembalikan marker ke posisi semula dan tampilkan toast error
- Hapus wajib melalui dialog konfirmasi
- Warna marker mencerminkan status

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

Konfigurasi lewat environment variable dengan nilai default: `PORT=8080`, `DB_PATH=./data/app.db`.

## Testing

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run lint
```

Ekspektasi minimal:
- Unit test validasi backend (input valid, tiap kasus tidak valid, nilai batas seperti lat = 90 / 90.0001)
- Test handler untuk status code (201, 422, 404, 204)

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
- [ ] Filter berdasarkan type / status
- [ ] Sidebar daftar entitas yang tersinkron dengan map
- [ ] Clustering marker
- [ ] Update realtime (SSE/WebSocket)

## Aturan untuk AI Agent

- Ikuti keputusan di file ini. Jika sebuah perubahan mengharuskan menyimpang darinya, berhenti dan tanyakan ke developer terlebih dahulu.
- Jangan menambah dependency, mengganti driver DB, atau mengubah kontrak API tanpa persetujuan.
- Nilai enum hanya didefinisikan di model backend.
- Tulis atau perbarui test bersamaan dengan setiap perubahan backend.
- Jangan menandai sebuah task selesai kecuali build berhasil dan test lulus.
- Jangan menulis secret atau kredensial ke dalam repository.
- Developer me-review semua kode yang dihasilkan dan mengambil keputusan desain akhir. Setiap fitur yang belum selesai atau diketahui bermasalah wajib dilaporkan agar bisa didokumentasikan di README.md.
