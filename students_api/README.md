# Dokumentasi API

REST API berbasis **Go + Fiber v2** untuk resource `students`. Base URL: `http://localhost:3000`. Data disimpan in-memory.

---

## Menjalankan

```bash
cd tugas2
go mod tidy
go run .
```

---

## Endpoint

| Method | Path                        | Deskripsi                       |
|--------|-----------------------------|---------------------------------|
| GET    | `/`                         | Hello World (cek koneksi)       |
| GET    | `/api/v1/health`            | Health check + timestamp        |
| POST   | `/api/v1/students`          | Buat student baru               |
| GET    | `/api/v1/students`          | List student + filter           |
| GET    | `/api/v1/students/:id`      | Detail student                  |
| PUT    | `/api/v1/students/:id`      | Ganti seluruh field             |
| PATCH  | `/api/v1/students/:id`      | Perbarui sebagian field         |
| DELETE | `/api/v1/students/:id`      | Hapus student                   |

---

## Konvensi Response

Semua respons (kecuali DELETE 204) memakai struktur:

```json
{
  "success": true,
  "message": "string",
  "data":    {},
  "meta":    {},   // hanya di endpoint list
  "errors":  {}    // hanya di error 422
}
```

Method ber-body (POST/PUT/PATCH) **wajib** mengirim header `Content-Type: application/json`, jika tidak akan mendapat `415`.

---

## 1. Hello World

**Request**

```http
GET /
```

**Response** — `200 OK`

```
Hello, World!
```

---

## 2. Health Check

**Request**

```http
GET /api/v1/health
```

**Response** — `200 OK`

```json
{
  "success": true,
  "message": "server berjalan",
  "data": { "timestamp": "2026-08-26T10:15:30.123456789+07:00" }
}
```

---

## 3. Buat Student

**Request**

```http
POST /api/v1/students
Content-Type: application/json

{
  "username": "habib",
  "email":    "habib@example.com",
  "password": "rahasia123"
}
```

**Required params:** `username` (non-empty), `email` (mengandung `@`), `password` (min. 8 karakter). `username` harus unik (case-insensitive).

**Response** — `201 Created`

```json
{
  "success": true,
  "message": "user berhasil dibuat",
  "data": {
    "id": 1,
    "username": "habib",
    "email": "habib@example.com",
    "password": "rahasia123",
    "is_active": true,
    "created_at": "2026-08-26T10:15:30.123456789+07:00"
  }
}
```

Header `Location: /api/v1/students/1` ikut dikembalikan.

---

## 4. List Student

**Request**

```http
GET /api/v1/students?page=1&limit=5&search=habib&sort=username&order=asc&is_active=true
```

**Query params (semua opsional):**

| Param      | Tipe   | Default | Keterangan                                                                  |
|------------|--------|---------|-----------------------------------------------------------------------------|
| `page`     | int    | `1`     | Halaman, min `1`                                                            |
| `limit`    | int    | `10`    | Item per halaman, `1`–`100`                                                 |
| `search`   | string | `""`    | Substring pada `username` atau `email` (case-insensitive)                   |
| `sort`     | string | `id`    | `id` \| `username` \| `email` \| `created_at`. Lainnya → `id`               |
| `order`    | string | `asc`   | `asc` \| `desc`                                                             |
| `is_active`| bool   | -       | `true` / `false` untuk filter status                                        |

**Response** — `200 OK`

```json
{
  "success": true,
  "message": "daftar user berhasil diambil",
  "data": [
    { "id": 1, "username": "habib", "email": "habib@example.com", "is_active": true }
  ],
  "meta": { "page": 1, "limit": 5, "total": 1, "total_pages": 1 }
}
```

---

## 5. Detail Student

**Request**

```http
GET /api/v1/students/1
```

**Path param:** `id` (int, ≥ 1).

**Response** — `200 OK`

```json
{
  "success": true,
  "message": "user ditemukan",
  "data": {
    "id": 1, "username": "habib", "email": "habib@example.com",
    "is_active": true, "created_at": "2026-08-26T10:15:30.123456789+07:00"
  }
}
```

---

## 6. Ganti Student (PUT)

**Request**

```http
PUT /api/v1/students/1
Content-Type: application/json

{
  "username":  "habib_baru",
  "email":     "habib_baru@example.com",
  "is_active": false
}
```

**Required params:** `username` (non-empty), `email` (mengandung `@`). `is_active` opsional.

**Response** — `200 OK`

```json
{
  "success": true,
  "message": "user berhasil diganti seluruhnya",
  "data": { "id": 1, "username": "habib_baru", "email": "habib_baru@example.com", "is_active": false }
}
```

---

## 7. Perbarui Sebagian (PATCH)

**Request**

```http
PATCH /api/v1/students/1
Content-Type: application/json

{ "is_active": true }
```

**Body:** minimal satu dari `username`, `email`, `is_active`. `username` dan `email` jika dikirim harus valid.

**Response** — `200 OK`

```json
{
  "success": true,
  "message": "user berhasil diperbarui sebagian",
  "data": { "id": 1, "username": "habib_baru", "email": "habib_baru@example.com", "is_active": true }
}
```

---

## 8. Hapus Student

**Request**

```http
DELETE /api/v1/students/1
```

**Response** — `204 No Content` (tanpa body).

---

## Kode Status

| Kode | Makna                                                  |
|------|--------------------------------------------------------|
| 200  | Sukses (GET/PUT/PATCH)                                 |
| 201  | Resource baru dibuat (POST)                            |
| 204  | Sukses tanpa body (DELETE)                             |
| 400  | Body invalid / `id` invalid / PATCH tanpa field        |
| 404  | Resource / endpoint tidak ditemukan                    |
| 415  | `Content-Type` bukan `application/json`                |
| 422  | Validasi field gagal — lihat `errors` di body          |

---

## Pengujian

- **Postman**: import `Praktikum_Pertemuan_2.postman_collection.json` (lengkap) atau `Tugas_Pertemuan_2.postman_collection.json` (latihan curl).
- **REST Client (VS Code)**: buka `api_test.http`.

Variabel koleksi: `baseUrl` (`http://localhost:3000`), `apiPrefix` (`/api/v1/students`), `contentType` (`application/json`).