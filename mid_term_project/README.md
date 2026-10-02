# SIAKAD Mini — Backend API (UTS PBE)

RESTful API untuk sistem akademik mini menggunakan **Go + Fiber + PostgreSQL**.
Mendukung dua peran: **admin** (kelola data mahasiswa) dan **mahasiswa** (lihat
mata kuliah, ambil KRS dengan batas SKS otomatis berdasarkan IPK).

---

## 1. Stack

| Komponen | Versi |
|----------|-------|
| Go | 1.26.6 |
| Fiber | v2.52 |
| pgx | v5 |
| JWT | golang-jwt/jwt v5 |
| Validator | go-playground/validator v10 |
| Logger | lumberjack (rotasi harian) |
| Password | bcrypt (cost 12) |

---

## 2. Struktur Proyek

```
siakad/
├── main.go                  # bootstrap Fiber + graceful shutdown
├── go.mod
├── cmd/seed/main.go         # seeder (migrations + admin/mahasiswa/courses)
├── config/                  # env, logger, app
├── database/                # pgx pool
├── migrations/              # 001_users, 002_students, 003_courses, 004_enrollments
├── app/
│   ├── model/               # DTO & entity
│   ├── repository/          # akses DB (pgx)
│   └── service/             # business logic
├── helper/                  # response, error, security, jwt, context, validator
├── middleware/              # auth, role, recover, helmet, ratelimit
└── route/route.go           # registrasi endpoint
```

---

## 3. Cara Menjalankan

### Prasyarat
- Go 1.26+
- PostgreSQL 14+ berjalan di `localhost:5432`
- Database `siakad` sudah dibuat (`CREATE DATABASE siakad;`)

### Langkah
```powershell
# 1. Salin env
Copy-Item .env.example .env
# (edit DB_PASSWORD dan JWT_SECRET)

# 3. Build & jalankan seeder (membuat tabel + data awal)
go run ./cmd/seed

# 4. Jalankan server
go run .
```

Seeder bersifat **idempotent**: jika data sudah ada, hanya data yang belum ada
yang akan di-insert.

### Akun Default
| Role   | Email                | Password   |
|--------|---------------------|------------|
| Admin  | `admin@siakad.test` | `admin123` |
| Mahasiswa | `<nim>@siakad.test` | `<nim>` (12 digit) |

> Contoh: mahasiswa dengan NIM `202410000001` login dengan password
> `202410000001`.

---

## 4. Endpoint

Base URL: `http://localhost:3000/api/v1`

| # | Method | Path | Auth | Role | Status Code |
|---|--------|------|------|------|-------------|
| 1 | POST | `/auth/login` | – | – | 200 / 401 / 422 / 429 |
| 2 | GET  | `/auth/me`    | ✔ | – | 200 / 401 |
| 3 | GET  | `/students`   | ✔ | admin | 200 / 401 / 403 |
| 4 | POST | `/students`   | ✔ | admin | 201 / 422 / 403 |
| 5 | GET  | `/students/:id` | ✔ | – | 200 / 403 / 404 |
| 6 | PUT  | `/students/:id` | ✔ | admin | 200 / 404 / 422 / 403 |
| 7 | DELETE | `/students/:id` | ✔ | admin | 204 / 404 / 403 |
| 8 | GET  | `/courses` | ✔ | – | 200 / 401 |
| 9 | POST | `/enrollments` | ✔ | mahasiswa | 201 / 409 / 422 / 403 |
| 10 | DELETE | `/enrollments/:id` | ✔ | mahasiswa | 204 / 403 / 404 |

`GET /health` tersedia tanpa token untuk cek server & koneksi DB.

---

## 5. Aturan Bisnis

### 5.1 Batas SKS berdasarkan IPK
| IPK Terakhir | Maksimal SKS |
|--------------|--------------|
| ≥ 3.00 | 24 |
| ≥ 2.50 | 21 |
| < 2.50 | 18 |

Pengecekan dilakukan **setelah transaksi enroll** agar data konsisten.
Jika melampaui batas, transaksi akan di-rollback dan dikembalikan respons 422.

### 5.2 Enrollment
- Validasi **kuota** mata kuliah (atomic dengan `SELECT ... FOR UPDATE`)
- Validasi **duplikat** (`UNIQUE(student_id, course_id, tahun_akademik)`)
- Validasi **batas SKS** mahasiswa

### 5.3 NIM
- 12 digit numerik
- Digunakan sebagai password awal

### 5.4 Tahun Akademik
- Format regex `^\d{4}/\d{4}(Ganjil|Genap)$`, contoh: `2026/2027Ganjil`

---

## 6. Response Format

#### Sukses — Object Tunggal
```json
{
  "success": true,
  "message": "...",
  "data": { ... }
}
```

#### Sukses — List (Paginated)
```json
{
  "success": true,
  "message": "...",
  "data": [ ... ],
  "meta": {
    "current_page": 1,
    "per_page": 10,
    "total": 25,
    "last_page": 3
  }
}
```

#### Gagal
```json
{
  "success": false,
  "code": "VALIDATION",
  "message": "...",
  "fields": { "field_name": "pesan" },
  "request_id": "uuid"
}
```

| Code | HTTP |
|------|------|
| `VALIDATION` | 422 |
| `BAD_REQUEST` | 400 |
| `UNAUTHORIZED` | 401 |
| `FORBIDDEN` | 403 |
| `NOT_FOUND` | 404 |
| `CONFLICT` | 409 |
| `RATE_LIMIT` | 429 |
| `INTERNAL` | 500 |

---

## 7. Keamanan

| Fitur | Implementasi |
|---|---|
| Password | bcrypt cost 12 |
| Token | JWT HS256 (issuer + expiry divalidasi) |
| Rate limit | 5 percobaan login/menit per IP (429 + `Retry-After`) |
| CORS | whitelist via `ALLOWED_ORIGINS` |
| Helmet | header keamanan standar |
| Validation | `go-playground/validator` + custom (`tahunakademik`, `ipkrange`) |
| Logging | JSON access log + lumberjack rotasi 10MB / 5 backup / 14 hari |

---

## 8. Pengujian Cepat dengan VS Code REST Client

Buka `api_test.http` di VS Code + ekstensi **REST Client**, lalu jalankan
permintaan `POST /auth/login` terlebih dahulu untuk mendapatkan token.

Bearer: request body `POST /auth/login` terlebih dahulu untuk mendapatkan token.