# DOCUMENTATION.md

Dokumentasi singkat untuk tester. Detail lengkap (kontrak API, aturan validasi, keterbatasan) ada di [README.md](README.md). Alasan pilihan database, API design, state management, map, dan styling dijelaskan di [TECH_DECISIONS.md](TECH_DECISIONS.md).

## 1. Cara Menjalankan Program

### Kebutuhan

- **Go 1.26+** → cek dengan `go version`
- **Node.js 20+** → cek dengan `node -v`
- Git

Tidak perlu install database, compiler C, atau API key. SQLite dibuat otomatis dan peta memakai OpenStreetMap.

### Langkah

**1. Clone repository**

```bash
git clone https://github.com/gustiarifiyanto/geo-entity-map.git
cd geo-entity-map
```

**2. Jalankan backend** (terminal 1, http://localhost:8080)

macOS / Linux / Git Bash:

```bash
cd backend
DEMO_ACCOUNTS=true go run ./cmd/server
```

Windows PowerShell:

```powershell
cd backend
$env:DEMO_ACCOUNTS="true"; go run ./cmd/server
```

Saat pertama kali jalan, backend membuat database `backend/data/app.db`, membuat tabel, dan mengisi 6 entitas contoh.

**3. Jalankan frontend** (terminal 2, http://localhost:5173)

```bash
cd frontend
npm install
npm run dev
```

**4. Buka http://localhost:5173 dan login**

Dengan `DEMO_ACCOUNTS=true`, halaman login menampilkan dua kartu akun demo:

| Kartu | Email | Bisa apa |
|---|---|---|
| **Admin** | `admin@demo.local` | Menambah (klik peta), mengedit, memindah (drag pin), dan menghapus entitas, plus melihat statistik user di Dashboard |
| **User** | `user@demo.local` | Hanya melihat peta, detail, dan dashboard |

Klik salah satu kartu. Email dan password akan terisi otomatis, lalu tekan **Masuk / Log in**.

- Password demo dibuat acak setiap kali server start, jadi akun demo hanya aktif selama server berjalan dengan `DEMO_ACCOUNTS=true`.
- Untuk membuat akun sendiri, pakai tab **Daftar / Register**. Akun baru selalu ber-role `user`.

### Opsional

**Admin dengan email sendiri, tanpa mode demo:**

```bash
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=password-minimal-8 go run ./cmd/server
```

PowerShell:

```powershell
$env:ADMIN_EMAIL="admin@example.com"; $env:ADMIN_PASSWORD="password-minimal-8"; go run ./cmd/server
```

**Menjalankan test:**

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run lint
```

**Mengulang dari data awal:** hentikan backend, hapus folder `backend/data/`, lalu jalankan lagi.

**Kalau ada masalah:**

| Gejala | Solusi |
|---|---|
| Frontend menampilkan error koneksi / 502 | Backend belum jalan. Jalankan langkah 2 dulu. |
| Kartu akun demo tidak muncul | Backend dijalankan tanpa `DEMO_ACCOUNTS=true`. Restart backend dengan env itu. |
| Port 8080 atau 5173 sudah dipakai | Tutup aplikasi lain yang memakai port tersebut. Untuk backend, `PORT` bisa diganti, tapi proxy di `frontend/vite.config.ts` juga harus ikut diubah. |
| `go.mod requires go >= 1.26.0` | Update Go ke versi 1.26 atau lebih baru. |

## 2. Alasan Pemilihan Library

Prinsipnya: sesedikit mungkin dependency, dan tidak ada yang butuh setup atau API key.

### Backend (Go)

| Library | Alasan |
|---|---|
| `go-chi/chi/v5` | Router ringan yang memakai `net/http` standar. Mendukung parameter path, sub-router, dan middleware (dipakai untuk cek login/role). |
| `modernc.org/sqlite` | SQLite **pure Go tanpa CGO**, jadi `go run` langsung jalan di Windows, macOS, dan Linux tanpa compiler C. Database berupa satu file, tanpa server. |
| `database/sql` (stdlib) | Query SQL ditulis langsung dan selalu memakai parameter. ORM tidak sebanding dengan kebutuhan proyek ini. |
| `go-playground/validator/v10` | Validasi berbasis tag struct, dengan aturan custom untuk enum, rentang koordinat, dan objek JSON. |
| `golang.org/x/crypto/bcrypt` | Hash password yang memang dibuat lambat dan sudah memakai salt. Ini paket semi-resmi dari tim Go. |
| `google/uuid` | Membuat dan memvalidasi UUID v4 untuk ID. |

Login memakai **session di server + cookie HttpOnly**, bukan JWT. Alasannya: session bisa dicabut saat itu juga (logout menghapus barisnya di DB), token tidak bisa dibaca JavaScript, dan tidak perlu library tambahan.

### Frontend (React)

| Library | Alasan |
|---|---|
| React + Vite + TypeScript (`strict`) | Sesuai soal. Vite memberi dev server cepat dan proxy `/api`, jadi tidak perlu setup CORS. |
| Leaflet + `react-leaflet` + OpenStreetMap | Peta gratis dan ringan, marker bisa di-drag, dan **tanpa API key**, sehingga tester bisa langsung menjalankan app. |
| `@tanstack/react-query` | Mengelola data dari server: caching, refetch otomatis setelah perubahan, serta optimistic update + rollback saat pin di-drag. |
| `react-hook-form` + `zod` | Form dengan error per field. Schema zod mengikuti aturan validasi backend yang sama. |
| Tailwind CSS | Styling cepat dan konsisten tanpa file CSS terpisah. |

Beberapa bagian sengaja dibuat sendiri supaya tidak menambah dependency:

- dialog konfirmasi, toast, dan grafik sensor (SVG);
- terjemahan ID | EN (kamus sendiri);
- rumus jarak zona (haversine);
- navigasi tab Map | Dashboard (tanpa router).

## 3. Workflow Penggunaan Agentic AI

Saya mengerjakan proyek ini bersama **Claude Code** (Anthropic) sebagai *pair programmer*. AI menulis sebagian besar kode. Saya yang mengatur alur kerja, mereview semua perubahan, mengambil keputusan desain, dan mengetes aplikasi (QA).

**Sejauh mana AI dipakai:**

| Bagian | Peran AI | Peran saya |
|---|---|---|
| Perencanaan & kontrak | Menyusun draf `CLAUDE.md` (tech stack, kontrak API, aturan validasi) | Mereview, mengubah, dan menyetujui sebelum ada kode |
| Kode backend & frontend | Menulis kode dan test | Mereview diff setiap langkah, lalu commit sendiri |
| Keputusan desain | Menawarkan pilihan beserta untung-ruginya, dan wajib berhenti untuk bertanya jika harus menyimpang dari `CLAUDE.md` | Memilih (contoh di bawah) |
| Verifikasi | Menjalankan `go vet`, `go test`, typecheck, lint, build, dan smoke test API | QA manual di browser, lalu melaporkan bug |
| Dokumentasi | Menulis draf README dan dokumen ini | Mereview dan menyetujui isinya |

**Alur kerja:**

1. **`CLAUDE.md` sebagai sumber kebenaran.** File ini berisi brief, kontrak API, aturan validasi, dan aturan untuk AI, misalnya:
   - tidak boleh menambah dependency atau mengubah kontrak tanpa persetujuan;
   - test wajib ditulis bersamaan dengan perubahan backend;
   - tidak boleh menulis secret ke repository.

   AI membaca file ini sebelum menulis kode apa pun. [AGENTS.md](AGENTS.md) mengarahkan AI agent lain ke file yang sama.
2. **Bertahap dan kecil.** MVP dipecah menjadi beberapa langkah: setup → model/DB → validasi → API → data layer → map → form → drag/hapus → README. Setiap langkah menjadi satu commit bergaya conventional commits. Fitur setelah MVP dikerjakan di branch terpisah (`feat/auth`, `feat/dashboard`), dan kontraknya ditulis di `CLAUDE.md` lebih dulu sebelum kodenya.
3. **Keputusan tetap di tangan developer.** Contohnya:
   - tetap memakai tile OpenStreetMap;
   - pengaman drag: hanya pin terpilih yang bisa di-drag, plus tombol Undo;
   - admin tidak ikut dihitung di statistik user;
   - pin di luar zona hanya ditandai, tidak ditolak;
   - foto disimpan saat *Save*, bukan langsung saat dipilih;
   - akun demo memakai password acak per start supaya tidak ada kredensial di repository.
4. **Tidak ada yang dianggap selesai sebelum terbukti.** Setiap langkah ditutup dengan build dan test yang lulus. Untuk logika penting (pembaruan `last_seen_at` maksimal sekali per menit, zona waktu "hari ini", kepemilikan API key perangkat, batas radius zona), test diuji dengan sengaja merusak kodenya untuk memastikan test benar-benar gagal.
5. **QA manual menemukan masalah yang tidak terlihat dari test.** Contohnya:
   - marker tergeser tanpa sengaja saat menggeser peta;
   - pesan "is required" muncul sebelum form dikirim;
   - input yang "kegepeng";
   - klaim AI bahwa tile CARTO gratis: HTTP 200, tetapi isinya gambar watermark "API KEY REQUIRED", jadi perubahan itu dibatalkan.

   Semua temuan diperbaiki lalu dites ulang.

Ringkasnya, AI mempercepat penulisan kode dan test, sedangkan arah, keputusan, dan pengecekan kualitas tetap dipegang developer. Cerita lengkap per fitur ada di bagian **Workflow AI** di [README.md](README.md).
