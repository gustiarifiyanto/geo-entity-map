# TECH_DECISIONS.md

Soal tes memberi kebebasan menentukan library, database, API design, state management, map provider, dan styling. File ini menjelaskan **apa yang dipilih dan kenapa**. Cara menjalankan ada di [DOCUMENTATION.md](DOCUMENTATION.md), dan detail lengkapnya di [README.md](README.md).

Prinsip yang dipakai untuk semua keputusan:

1. **Mudah dijalankan tester**: tanpa install database, compiler C, atau API key.
2. **Dependency sesedikit mungkin**: yang bisa ditulis dalam beberapa baris dibuat sendiri.
3. **Backend adalah otoritas**: validasi dan hak akses ditegakkan di server, frontend hanya membantu UX.

## Ringkasan

| Aspek | Pilihan |
|---|---|
| Library | Backend: chi, validator, bcrypt, uuid. Frontend: React Query, react-hook-form + zod, react-leaflet. |
| Database | SQLite (`modernc.org/sqlite`, pure Go) + `database/sql` tanpa ORM |
| API design | REST JSON di `/api`, format response dan kode error yang konsisten, enum dari `GET /api/meta` |
| State management | React Query (server state) + react-hook-form (form) + Context/useState (UI), tanpa Redux |
| Map | Leaflet + `react-leaflet` + tile OpenStreetMap |
| Styling | Tailwind CSS |

## 1. Library yang Digunakan

**Backend (Go)**

| Library | Kenapa |
|---|---|
| `go-chi/chi/v5` | Router ringan di atas `net/http` standar. Mendukung parameter path, sub-router, dan middleware (dipakai untuk cek login/role) tanpa framework besar. |
| `go-playground/validator/v10` | Validasi berbasis tag struct, bisa ditambah aturan custom (enum, rentang koordinat, objek JSON). Pesan error dipetakan ke nama field JSON. |
| `golang.org/x/crypto/bcrypt` | Hash password yang memang dibuat lambat dan sudah memakai salt. Ini paket semi-resmi dari tim Go. |
| `google/uuid` | ID berupa UUID v4 yang dibuat backend. |

**Frontend (React + Vite + TypeScript strict)**

| Library | Kenapa |
|---|---|
| `@tanstack/react-query` | Data dari server: caching, refetch setelah mutasi, dan optimistic update + rollback saat pin di-drag. |
| `react-hook-form` + `zod` | Form dengan error per field. Schema zod menyalin aturan validasi backend, dan error 422 dari server dipetakan ke input yang sama. |
| `leaflet` + `react-leaflet` | Lihat bagian Map. |
| Tailwind CSS | Lihat bagian Styling. |

**Sengaja tidak memakai library** untuk:

- dialog konfirmasi (elemen `<dialog>` bawaan browser) dan toast;
- grafik sensor (SVG sederhana);
- terjemahan ID | EN (kamus sendiri);
- rumus jarak zona (haversine);
- navigasi tab Map | Dashboard (state biasa, tanpa router).

Semuanya cukup beberapa baris, jadi tidak sebanding dengan menambah dependency.

## 2. Database

**Pilihan:** SQLite dengan driver `modernc.org/sqlite`, diakses lewat `database/sql` (stdlib).

- **Tanpa setup.** Database berupa satu file (`backend/data/app.db`) yang dibuat otomatis saat server start, lengkap dengan migrasi dan data contoh. Tester tidak perlu install atau menjalankan server database.
- **Pure Go, tanpa CGO.** `go run` langsung jalan di Windows, macOS, dan Linux. Driver `mattn/go-sqlite3` tidak dipilih karena butuh compiler C.
- **Tanpa ORM.** Query ditulis langsung dan **selalu memakai parameter**, jadi aman dari SQL injection. Skemanya sederhana, sehingga ORM tidak sebanding dengan kompleksitasnya.
- **Integritas data dijaga di database juga:**
  - `foreign_keys` aktif dengan `ON DELETE CASCADE`, jadi menghapus entitas ikut menghapus foto, jadwal, sensor, dan zonanya;
  - `CHECK` constraint untuk role;
  - `UNIQUE` untuk email;
  - WAL + `busy_timeout` supaya baca dan tulis bersamaan tidak saling mengunci.
- **Migrasi aman untuk DB lama.** Kolom baru ditambahkan dengan `ALTER TABLE` setelah dicek, jadi database yang sudah ada tidak perlu dihapus.
- **File foto disimpan di disk** (`UPLOAD_DIR`), bukan di database. Database hanya mencatat metadata-nya.

**Trade-off:** SQLite cocok untuk satu server dan beban kecil–menengah. Untuk banyak instance atau penulisan yang sangat ramai, langkah berikutnya adalah PostgreSQL. Lapisan repository sudah terpisah, jadi perpindahannya terlokalisasi.

## 3. API Design

**Pilihan:** REST + JSON dengan base path `/api`. Backend berlapis `handler → service → repository`: handler tidak pernah menjalankan SQL, dan repository tidak tahu apa pun soal HTTP.

- **Resource yang jelas.** `GET/POST /api/entities`, `GET/PUT/DELETE /api/entities/{id}`.
- **`PATCH /api/entities/{id}/location` terpisah dari `PUT`.** Drag marker hanya mengirim koordinat, jadi tidak bisa menimpa field lain yang sedang diedit orang lain.
- **Format response konsisten:**
  - sukses → `{ "data": ... }`;
  - error → `{ "error": "<kode>", "message": "..." }`;
  - error validasi → **422** dengan `fields` per field (nama field JSON), sehingga frontend bisa menampilkan error tepat di bawah inputnya.
- **Kode status bermakna:**

  | Kode | Arti |
  |---|---|
  | 201 | dibuat |
  | 204 | dihapus |
  | 400 | body rusak |
  | 401 | belum login |
  | 403 | role tidak cukup |
  | 404 | tidak ada |
  | 422 | data tidak valid |
  | 500 | error internal, tanpa membocorkan detail |
- **Enum dari server.** `GET /api/meta` mengirim daftar type, status, metric sensor, dan **kemampuan per type** (misalnya `vehicle` → zona operasional). Frontend tidak meng-hardcode daftar ini, jadi menambah type baru cukup di backend.
- **Validasi ketat.** Field tak dikenal ditolak (`DisallowUnknownFields`), string di-trim, koordinat `NaN`/`Inf` ditolak, dan jenis foto dicek dari isi file, bukan dari nama file.
- **Auth dengan session di server + cookie HttpOnly, bukan JWT.**
  - Session bisa dicabut saat itu juga: logout menghapus barisnya di DB.
  - Token tidak bisa dibaca JavaScript.
  - Di database hanya hash SHA-256 token yang disimpan.
  - Hak akses dicek di middleware backend: `user` hanya melihat, `admin` bisa mengelola.
- **Nilai turunan dihitung server saat diminta, tidak disimpan.** Contohnya status jadwal pemasangan dan jarak/inside zona. Hasilnya selalu sesuai data terbaru dan tidak ada data ganda yang bisa tidak sinkron.

Kontrak lengkap semua endpoint ada di [README.md](README.md) bagian API.

## 4. State Management

**Pilihan:** state dipisah sesuai jenisnya, **tanpa Redux/Zustand**.

| Jenis state | Dikelola oleh | Contoh |
|---|---|---|
| Server state (data dari API) | **React Query** | daftar entitas, detail, foto, statistik dashboard, user yang sedang login |
| Form state | **react-hook-form** + zod | form login, form tambah/edit entitas |
| UI state global yang kecil | **React Context** | bahasa, toast, mode "Pick on map" untuk zona |
| UI state lokal | `useState` | entitas yang dipilih, tab Map/Dashboard, isi pencarian |

Alasannya:

- Hampir semua state di app ini adalah **data server**. React Query sudah menangani cache, loading/error, refetch setelah mutasi, dan polling. Untuk polling: statistik tiap 30 detik, heartbeat online tiap 2 menit, dan data sensor tiap 30 detik.
- **Drag marker memakai optimistic update.** Pin langsung pindah, dan jika backend menolak, cache dikembalikan ke posisi semula disertai toast error.
- Setelah itu, state global yang tersisa sangat sedikit, jadi store terpisah seperti Redux hanya menambah boilerplate.

## 5. Map Provider / Map Library

**Pilihan:** Leaflet + `react-leaflet` dengan tile **OpenStreetMap**.

- **Gratis dan tanpa API key**, jadi tester bisa langsung menjalankan app. Google Maps dan Mapbox butuh API key atau billing. Tile CARTO sempat dicoba, tetapi ternyata sekarang butuh API key (gambarnya berisi watermark), jadi dibatalkan.
- **Ringan dan cukup untuk semua kebutuhan:**
  - marker berwarna sesuai status;
  - marker yang bisa di-drag;
  - klik peta untuk menambah entitas;
  - lingkaran zona operasional (`<Circle>`);
  - animasi terbang ke pin.
- **Jebakan longitude sudah ditangani.** Leaflet bisa menghasilkan longitude di luar −180..180 saat peta digeser melewati batas dunia. Peta memakai `noWrap` + `maxBounds`, dan longitude selalu dinormalisasi sebelum dikirim, jadi tidak ditolak validasi backend.
- **Pengaman drag.** Hanya pin yang sedang dipilih yang bisa di-drag, dan ada tombol Undo, supaya pin tidak tergeser tanpa sengaja saat menggeser peta.

**Trade-off:** tile OSM publik punya kebijakan pemakaian yang wajar. Untuk produksi dengan trafik tinggi sebaiknya memakai penyedia tile berbayar atau server tile sendiri, dan cukup mengganti URL tile.

## 6. Styling Approach

**Pilihan:** Tailwind CSS (utility-first), tanpa component library.

- **Cepat dan konsisten.** Warna, jarak, dan ukuran memakai skala yang sama di semua komponen, tanpa file CSS terpisah per komponen dan tanpa bentrok nama class.
- **Ukuran kecil.** Hanya class yang dipakai yang masuk ke build.
- **Tanpa component library** (MUI, Ant Design, dll.). UI app ini relatif sederhana: form, kartu, dialog, toast, dan bar statistik. Membuatnya sendiri menjaga ukuran bundle dan dependency tetap kecil, dan tampilannya bisa disesuaikan penuh.
- **Pola bersama** seperti `inputClass` dipakai berulang supaya input di semua form terlihat sama.
- **Detail UX:**
  - warna marker dan bar dashboard sama per status;
  - animasi CSS ringan untuk panel, dialog, dan kartu;
  - app bar transparan;
  - atribut aksesibilitas (`aria-*`, `role`) pada dialog, tab, dan pesan error.
