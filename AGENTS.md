# AGENTS.md

Panduan untuk AI agent ada di [CLAUDE.md](CLAUDE.md). Baca file itu sampai selesai sebelum melakukan perubahan apa pun: tech stack, kontrak API, aturan validasi, konvensi kode, dan batasan untuk agent semuanya ada di sana.

Ringkasan singkat:

- **Backend:** Go + chi + SQLite (`modernc.org/sqlite`), di folder `backend/`. Jalankan dengan `go run ./cmd/server`, test dengan `go test ./...`.
- **Frontend:** React + Vite + TypeScript, di folder `frontend/`. Jalankan dengan `npm run dev`, cek dengan `npm run typecheck && npm run lint`.
- **Jangan** menambah dependency, mengubah kontrak API, atau menulis secret ke repository tanpa persetujuan developer.
- Tulis test bersamaan dengan setiap perubahan backend. Tugas belum selesai sebelum build dan test lulus.
