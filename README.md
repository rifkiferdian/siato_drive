# Cloud Drive

Fondasi aplikasi Cloud Drive modular-monolith dengan Gin, MySQL 8, server-side HTML, private local storage, dan portal publik multi-organisasi.

## Prasyarat

- Go 1.21 atau lebih baru
- MySQL 8.0.23 atau lebih baru
- Tailwind CSS v4 CLI (hanya saat membangun CSS)

Node.js dan Docker tidak dibutuhkan saat runtime.

## Menjalankan lokal

1. Buat database dan user MySQL:

   ```sql
   CREATE DATABASE cloud_drive CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   CREATE USER 'cloud_drive'@'localhost' IDENTIFIED BY 'ganti-password-ini';
   GRANT ALL PRIVILEGES ON cloud_drive.* TO 'cloud_drive'@'localhost';
   ```
2. Salin `.env.example` menjadi `.env`, lalu ganti credential dan secret development.
3. Jalankan migrasi: `go run ./cmd/migrate`.
4. Buat super admin: `go run ./cmd/seed`. Password minimal 6 karakter, mengandung huruf besar dan angka, serta dibaca dari environment.
5. Jalankan server: `go run ./cmd/server`, kemudian buka `http://localhost:8080`.

### Development dengan hot reload

Air sudah dikonfigurasi melalui `.air.toml`. Jalankan dari PowerShell:

```powershell
air -c .air.toml
```

Jika folder Go bin belum masuk `PATH`, gunakan:

```powershell
& "$env:USERPROFILE\go\bin\air.exe" -c .air.toml
```

Air memantau file Go, HTML, CSS, dan JavaScript, lalu membangun dan menjalankan ulang server secara otomatis. Hentikan dengan `Ctrl+C`.

### Aset frontend lokal

SweetAlert2 dikunci pada versi yang tercatat di `package.json` dan disajikan secara lokal dari `web/static/vendor`; aplikasi production tidak mengambil aset dari CDN. Untuk memperbarui salinan vendor setelah `npm install`, jalankan `npm run vendor:sweetalert` (gunakan `npm.cmd` pada PowerShell yang membatasi script `.ps1`).

Jika Tailwind CLI tersedia, CSS dapat dibangun dengan `tailwindcss -i assets/css/input.css -o web/static/css/app.css --minify`. CSS siap-pakai disertakan agar bootstrap development tidak bergantung pada CLI.

## Yang telah berfungsi

- Konfigurasi `.env`, pool `database/sql` dengan driver MySQL, graceful HTTP shutdown, health check, dan security headers.
- Migration awal satu database dengan prefix tabel per modul, FK, index, unique active sibling name, tenant-aware drive model, serta trigger pencegah folder cycle.
- Seed role/permission serta super admin berbasis environment dengan Argon2id.
- Login/logout, server-side session yang hanya menyimpan hash token, cookie HttpOnly/SameSite/Secure configurable, CSRF, permission middleware, dan audit login.
- Abstraksi storage dan local private provider: streaming temporary upload, SHA-256, atomic move, serta perlindungan path traversal.
- Portal `/public` dan `/public/:organizationSlug`: browsing, search, filter, sort, serta public download yang memvalidasi organisasi, public drive, status file/node, dan storage object.
- Dashboard SSR responsif serta route dashboard Drive Saya/Bersama/Publik/Sampah yang diproteksi session.
- Drive Saya end-to-end: browsing folder, breadcrumb, pencarian, sorting, membuat folder, multi-upload dari UI, streaming file ke private storage, checksum SHA-256, download terproteksi, rename, dan hapus ke Sampah.
- Drive Publik pengelola end-to-end: organisasi default, membership, folder, upload streaming, preview/download terproteksi, rename, hapus ke Sampah, serta publikasi otomatis ke portal anonim `/public` dan `/public/:organizationSlug`.
- Drive Bersama end-to-end: Shared Drive organisasi, membership dan role drive, folder, upload streaming, preview/download, rename inline, hapus ke Sampah, pencarian, sorting, serta tampilan List/Grid untuk kolaborasi internal.
- Sampah terintegrasi untuk Drive Saya, Drive Bersama, dan Drive Publik: pencarian/filter asal drive, restore ke lokasi semula, serta hapus permanen metadata dan objek private storage dengan konfirmasi SweetAlert2.
- Nginx dan systemd example.

## Belum selesai

Fase lanjutan masih memerlukan copy/move, favorite, ACL resource, sharing, quota/resumable upload, admin CRUD/reporting, PDF.js, malware scanner integration, login rate limiting persisten, pembersihan Sampah terjadwal, serta integration/security test dengan MySQL terisolasi.

## Backup dan restore

Metadata: `mysqldump --single-transaction --routines --triggers cloud_drive > cloud_drive.sql`; restore dengan `mysql cloud_drive < cloud_drive.sql`. Backup direktori `storage/objects` pada snapshot yang konsisten dengan database. Uji restore berkala dan lindungi backup dengan enkripsi serta akses terbatas.

## Validasi

Jalankan `go test ./...`, `go vet ./...`, dan `go build ./cmd/server`. Migration perlu diuji pada MySQL 8 nyata sebelum deployment jika server MySQL belum tersedia di environment pengembangan.
