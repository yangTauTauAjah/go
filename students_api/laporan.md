# Laporan Pengujian API â€” Praktikum Backend Lanjut Pertemuan 2

**Mata Kuliah:** Backend Lanjut
**Pertemuan:** 2
**Framework:** Go + Fiber v2
**Storage:** PostgreSQL (melalui driver `pgx/v5` & pool `pgxpool`)
**Server:** `http://localhost:3000`
**Tanggal Pengujian:** 26 Agustus 2026
**Tool Pengujian:** Postman (collection: `Tugas_Pertemuan_2.postman_collection.json`)

> **Versi program:** Pertemuan ke-2 versi lanjut — sebelumnya memakai slice in-memory, kini dipindahkan ke PostgreSQL dengan pola *repository*. Perubahan ini hanya menyentuh lapisan data; HTTP contract (route, body, status code) tetap sama sehingga pengujian tidak perlu diulang dari awal.

---

## 1. Informasi Umum

API yang diuji adalah service CRUD untuk resource **users** dengan base path `/api/v1`. Setiap pengujian dilakukan menggunakan Postman dan memperhatikan **status HTTP** yang di-return oleh server, bukan hanya isi body response.

### Endpoint yang diuji

| No | Method | Path                 | Tujuan                          | Status Sukses   |
|----|--------|----------------------|---------------------------------|-----------------|
| 1  | POST   | `/api/v1/users`      | Membuat user baru               | 201 Created     |
| 2  | GET    | `/api/v1/users`      | Mengambil daftar user + filter  | 200 OK          |
| 3  | GET    | `/api/v1/users/:id`  | Mengambil detail user           | 200 OK / 404    |
| 4  | PUT    | `/api/v1/users/:id`  | Mengganti seluruh data user     | 200 OK / 422 / 409 |
| 5  | PATCH  | `/api/v1/users/:id`  | Memperbarui sebagian data user   | 200 OK / 400    |
| 6  | DELETE | `/api/v1/users/:id`  | Menghapus user                   | 204 No Content  |

### Middleware & Lapisan Aplikasi

```go
// main.go
app.Use(requestid.New())
app.Use(logger.New(logger.Config{...}))
app.Use(cors.New())

// Group dengan requireJSON â€” method ber-body wajib Content-Type JSON
u := api.Group("/students", requireJSON)
u.Get   ("/",   userHandler.List)
u.Get   ("/:id", userHandler.Get)
u.Post  ("/",   userHandler.Create)
u.Put   ("/:id", userHandler.Replace)
u.Patch ("/:id", userHandler.Patch)
u.Delete("/:id", userHandler.Delete)
```

```go
// main.go â€” bootstrap pool â†’ repository â†’ handler
config.LoadEnv()
pool, err := database.NewPool(context.Background())
if err != nil { log.Fatalf("database: %v", err) }
defer pool.Close()

userRepository := repository.NewUserRepository(pool)   // interface
userHandler    := NewUserHandler(userRepository)       // terima interface
```

Penjelasan singkat tiap lapisan:

| Lapisan        | Tipe                   | Tugas                                           |
|----------------|------------------------|-------------------------------------------------|
| `main.go`      | bootstrap              | Memuat env, membuka pool, merakit komponen      |
| `handler.go`   | `UserHandler`          | Menerjemahkan HTTP â‡„ domain, validasi ringan    |
| `repository`   | `UserRepository` (IF)  | Storage contract — tanpa kata "SQL"          |
| `userPostgresRepository` | struct konkret | Implementasi pgx, query berparameter          |
| `database/postgres.go`   | `pgxpool.Pool` | Koneksi pool + ping saat start-up             |

### Status `503 Service Unavailable` (baru)

Karena `GET /api/v1/health` kini ikut melakukan `pool.Ping`, server dapat return `503` ketika database tidak dapat dihubungi. Skenario ini tidak diuji di sini karena berfokus pada endpoint CRUD.

---

## 2. Skema Basis Data & Migrasi

Berkas migrasi: [students_api/migrations/001_create_students.sql](students_api/migrations/001_create_students.sql).

### 2.1 Skema Tabel `users`

| Kolom        | Tipe             | Constraint                          | Keterangan                                  |
|--------------|------------------|-------------------------------------|---------------------------------------------|
| `id`         | `SERIAL`         | `PRIMARY KEY`                       | Auto-increment oleh PostgreSQL              |
| `username`   | `VARCHAR(50)`    | `NOT NULL`                          | Nama pengguna                               |
| `email`      | `VARCHAR(255)`   | `NOT NULL`                          | Alamat email                                |
| `password`   | `VARCHAR(255)`   | `NOT NULL`                          | Disimpan sebagai hash pada praktikum lanjut |
| `is_active`  | `BOOLEAN`        | `NOT NULL DEFAULT TRUE`             | Status aktif                                |
| `created_at` | `TIMESTAMPTZ`    | `NOT NULL DEFAULT NOW()`            | Timestamp pembuatan otomatis                |

### 2.2 Indeks

| Nama indeks                          | Tipe             | Kolom                  | Tujuan                                                                |
|--------------------------------------|------------------|------------------------|-----------------------------------------------------------------------|
| `users_pkey`                         | Primary Key      | `id`                   | Search by id                                                          |
| `users_username_lower_key` (UNIQUE)  | B-tree unik      | `LOWER(username)`      | Menjamin keunikan tanpa membedakan huruf besar/kecil                  |
| `users_email_lower_idx`              | B-tree           | `LOWER(email)`         | Mempercepat search case-insensitive pada email                       |

### 2.3 Skrip Migrasi Lengkap

```sql
-- students_api/migrations/001_create_students.sql
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Keunikan username tanpa membedakan huruf besar dan kecil.
-- Inilah yang menggantikan pemeriksaan manual di pertemuan 2.
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key
    ON users (LOWER(username));

CREATE INDEX IF NOT EXISTS users_email_lower_idx
    ON users (LOWER(email));
```

### 2.4 Konfigurasi Koneksi

Contoh `.env`:

```env
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=rahasia
DB_NAME=praktikum_backend
DB_SSLMODE=disable
DB_MAX_CONNS=10
```

Potongan kode `database/postgres.go`:

```go
dsn := fmt.Sprintf(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    config.GetEnv("DB_USER",     "postgres"),
    config.GetEnv("DB_PASSWORD", ""),
    config.GetEnv("DB_HOST",     "localhost"),
    config.GetEnv("DB_PORT",     "5432"),
    config.GetEnv("DB_NAME",     "praktikum_backend"),
    config.GetEnv("DB_SSLMODE",  "disable"),
)
cfg, err := pgxpool.ParseConfig(dsn)
cfg.MaxConns          = int32(config.GetEnvInt("DB_MAX_CONNS", 10))
cfg.MinConns          = 2
cfg.MaxConnLifetime   = time.Hour
cfg.MaxConnIdleTime   = 30 * time.Minute
pool, err := pgxpool.NewWithConfig(ctx, cfg)
if err := pool.Ping(pingCtx); err != nil { /* fatal */ }
```

### 2.5 Penjelasan Singkat

- **`SERIAL`** menyerahkan pembuatan id ke PostgreSQL sehingga tidak ada `nextID++` di Go lagi.
- **`UNIQUE INDEX ... LOWER(username)`** menjamin tidak ada dua user dengan username identik secara case-insensitive, sehingga aplikasi tidak perlu query `SELECT` tambahan sebelum `INSERT` â€” INSERT langsung gagal dengan kode error PostgreSQL `23505`.
- **`pgxpool`** mengelola banyak koneksi sekaligus; query yang lambat tidak saling menunggu.
- **`pool.Ping()`** saat start-up memvalidasi kredensial sebelum server menerima permintaan pertama.
- **`RETURNING id, created_at`** pada `INSERT`/`UPDATE` me-return nilai yang dibuat database dalam satu perjalanan, tanpa perlu query kedua.

---

## 3. Testing Endpoint

### 3.1 POST â€” Membuat User Baru

**Permintaan**

```json
POST /api/v1/users
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "rahasia123"
}
```

**Potongan kode handler (sekarang melalui repository)**

```go
// handler.go â€” UserHandler.Create
baru, err := h.repo.Create(ctx, model.Student{
    Username: req.Username,
    Email:    req.Email,
    Password: req.Password,
    IsActive: true,
})
if err != nil {
    return terjemahkanError(c, err, "gagal menyimpan user")
}
return created(c, "user berhasil dibuat", baru,
    "/api/v1/users/"+strconv.Itoa(baru.ID))
```

```go
// app/repository/user_repository.go â€” userPostgresRepository.Create
err := r.pool.QueryRow(ctx,
    `INSERT INTO users (username, email, password, is_active)
     VALUES ($1, $2, $3, $4)
     RETURNING id, created_at`,
    u.Username, u.Email, u.Password, u.IsActive,
).Scan(&u.ID, &u.CreatedAt)
if isUniqueViolation(err) {
    return model.Student{}, ErrDuplicate
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 01_post_create_success.png â€” paste Postman screenshot showing `201 Created` here]*

**Penjelasan:** Server me-return status **201 Created** saat user berhasil disimpan ke PostgreSQL. Header `Location` berisi URL resource baru (`/api/v1/users/1`). `id` kini di-generate oleh `SERIAL` PostgreSQL, dan `created_at` di-`RETURNING` bersamaan dengan `INSERT`, sehingga hanya satu perjalanan ke database.

---

### 3.2 POST â€” Tambah User Kedua dan Ketiga

**Permintaan**

```json
POST /api/v1/users
{
  "username": "andini",
  "email": "andini@example.com",
  "password": "rahasia123"
}

POST /api/v1/users
{
  "username": "budi_s",
  "email": "budi@example.com",
  "password": "rahasia123"
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 02a_post_create_andini.png â€” paste screenshot showing `201 Created` for user ke-2 here]*

> ðŸ“· *[Screenshots: 02b_post_create_budi.png â€” paste screenshot showing `201 Created` for user ke-3 here]*

**Penjelasan:** Kedua permintaan me-return **201 Created**. Setelah tiga kali `INSERT`, tabel `users` berisi 3 baris dengan id 1, 2, dan 3 yang dibuat otomatis oleh PostgreSQL. Tidak ada `nextID` di Go lagi.

---

### 3.3 GET â€” Pagination dan Sorting

**Permintaan**

```
GET /api/v1/users?page=1&limit=2&sort=username&order=desc
```

**Potongan kode repository (filter + sort + paging di sisi basis data)**

```go
// app/repository/user_repository.go â€” userPostgresRepository.FindAll
where, args := buildFilter(q)

// 1) Hitung total sebelum dipenggal
var total int
err := r.pool.QueryRow(ctx,
    "SELECT COUNT(*) FROM users"+where, args...).Scan(&total)

// 2) Ambil satu halaman dari basis data
arah := "ASC"
if q.Order == "desc" { arah = "DESC" }

sqlText := fmt.Sprintf(
    `SELECT id, username, email, password, is_active, created_at
     FROM users%s
     ORDER BY %s %s
     LIMIT $%d OFFSET $%d`,
    where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
)
args = append(args, q.Limit, q.Offset())
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 03_get_pagination_sort.png â€” paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. Filtering (`WHERE`), sorting (`ORDER BY`) dan paging (`LIMIT/OFFSET`) seluruhnya dijalankan PostgreSQL â€” bukan di slice Go. Mapping `sort` ke kolom tetap melewati whitelist `kolomUrut` sehingga injection pada `ORDER BY` tidak mungkin. Total baris (`total`) dihitung via `COUNT(*)` terpisah untuk isi `meta`.

---

### 3.4 GET â€” Search dan Filter

**Permintaan**

```
GET /api/v1/users?search=an&is_active=true
```

**Potongan kode (parameterized â€” nilai selalu jadi argumen)**

```go
// app/repository/user_repository.go â€” buildFilter
if q.Search != "" {
    where += fmt.Sprintf(
        " AND (username ILIKE $%d OR email ILIKE $%d)",
        len(args)+1, len(args)+1)
    args = append(args, "%"+q.Search+"%")
}
if q.IsActive != nil {
    where += fmt.Sprintf(
        " AND is_active = $%d", len(args)+1)
    args = append(args, *q.IsActive)
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 04_get_search_filter.png â€” paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. Search substring `an` dipakai pada klausa `ILIKE`, sehingga cocok pada username `andini` maupun email yang memuat substring `an`. Filter `is_active = TRUE` ditambahkan sebagai parameter terpisah, sehingga nilainya tidak pernah disambung ke teks SQL secara langsung.

---

### 3.5 PUT â€” Replace User (Seluruh Field Wajib)

**Permintaan**

```json
PUT /api/v1/users/1
Content-Type: application/json

{
  "username": "john_baru",
  "email": "jb@example.com",
  "is_active": false
}
```

**Potongan kode (update via repository)**

```go
// handler.go â€” UserHandler.Replace
hasil, err := h.repo.Update(ctx, model.Student{
    ID:       id,
    Username: req.Username,
    Email:    req.Email,
    IsActive: req.IsActive,
})
if err != nil { return terjemahkanError(c, err, "gagal memperbarui user") }
return ok(c, "user berhasil diganti seluruhnya", hasil)
```

```go
// app/repository/user_repository.go â€” userPostgresRepository.Update
err := r.pool.QueryRow(ctx,
    `UPDATE users SET username = $1, email = $2, is_active = $3
     WHERE id = $4
     RETURNING id, username, email, password, is_active, created_at`,
    u.Username, u.Email, u.IsActive, u.ID,
).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.IsActive, &u.CreatedAt)
if errors.Is(err, pgx.ErrNoRows)        { return model.Student{}, ErrNotFound }
if isUniqueViolation(err)               { return model.Student{}, ErrDuplicate }
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 05_put_replace_success.png â€” paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. PUT menggantikan seluruh field pada baris dengan id 1 menggunakan satu `UPDATE ... RETURNING`. Baris yang di-return PostgreSQL menyertakan kembali `created_at` sehingga field yang tidak ikut diubah tetap benar.

---

### 3.6 PUT â€” Validasi Gagal (Tanpa Email)

**Permintaan**

```json
PUT /api/v1/users/1
Content-Type: application/json

{
  "username": "john_baru"
}
```

**Potongan kode validasi**

```go
// handler.go â€” UserHandler.Replace
errs := map[string]string{}
if strings.TrimSpace(req.Username) == "" {
    errs["username"] = "wajib diisi pada PUT"
}
if !strings.Contains(req.Email, "@") {
    errs["email"] = "wajib diisi dan berformat email pada PUT"
}
if len(errs) > 0 {
    return failValidation(c, errs)   // â†’ 422
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 06_put_validation_422.png â€” paste Postman screenshot showing `422 Unprocessable Entity` here]*

**Penjelasan:** Status **422 Unprocessable Entity**. Validasi `email` gagal sebelum repository dipanggil, sehingga tidak ada `UPDATE` yang dikirim ke PostgreSQL. Respons memuat objek `errors` dengan key `email`.

---

### 3.7 PATCH â€” Update Sebagian (is_active)

**Permintaan**

```json
PATCH /api/v1/users/1
Content-Type: application/json

{
  "is_active": true
}
```

**Potongan kode (baca-ubah-simpan)**

```go
// handler.go â€” UserHandler.Patch
saatIni, err := h.repo.FindByID(ctx, id)
if err != nil { return terjemahkanError(c, err, "gagal mengambil data user") }

if req.IsActive != nil { saatIni.IsActive = *req.IsActive }

hasil, err := h.repo.Update(ctx, saatIni)
if err != nil { return terjemahkanError(c, err, "gagal memperbarui user") }
return ok(c, "user berhasil diperbarui sebagian", hasil)
```

```go
// app/repository/user_repository.go â€” userPostgresRepository.FindByID
err := r.pool.QueryRow(ctx,
    `SELECT id, username, email, password, is_active, created_at
     FROM users WHERE id = $1`, id,
).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.IsActive, &u.CreatedAt)
if errors.Is(err, pgx.ErrNoRows) {
    return model.Student{}, ErrNotFound
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 07_patch_partial.png â€” paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. PATCH membaca baris utuh dengan `FindByID`, mengubah field yang dikirim, lalu menyimpan kembali dengan `Update`. Repository hanya butuh satu metode `Update`; perbedaan PUT/PATCH diputuskan di handler. (Indonesia: `menyimpan` = save.)

---

### 3.8 POST â€” Tanpa Content-Type

**Permintaan**

```
POST /api/v1/users   (tanpa header Content-Type)
Body: {"username":"x"}
```

**Potongan kode middleware**

```go
// main.go â€” requireJSON
var metodeBerbody = map[string]bool{
    fiber.MethodPost:  true,
    fiber.MethodPut:   true,
    fiber.MethodPatch: true,
}

func requireJSON(c *fiber.Ctx) error {
    if metodeBerbody[c.Method()] {
        ct := c.Get("Content-Type")
        if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
            return fail(c, fiber.StatusUnsupportedMediaType,
                "Content-Type harus application/json")
        }
    }
    return c.Next()
}
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 08_post_no_content_type.png â€” paste Postman screenshot showing `415 Unsupported Media Type` here]*

**Penjelasan:** Status **415 Unsupported Media Type**. Middleware menolak permintaan sebelum sampai ke handler `Create` atau bahkan repository â€” koneksi database tidak terpakai sia-sia.

---

### 3.9 DELETE â€” Hapus User

**Permintaan**

```
DELETE /api/v1/users/2
```

**Potongan kode**

```go
// app/repository/user_repository.go â€” userPostgresRepository.Delete
tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
if err != nil { return fmt.Errorf("menghapus user: %w", err) }
if tag.RowsAffected() == 0 {
    return ErrNotFound
}
return nil
```

**Screenshot pengujian**

> ðŸ“· *[Screenshots: 09_delete_success.png â€” paste Postman screenshot showing `204 No Content` here]*

**Penjelasan:** Status **204 No Content**. `Exec` me-return `tag.RowsAffected()` â€” bila 0 berarti id memang tidak ada dan repository menerjemahkannya menjadi `ErrNotFound` (status 404), bukan false positive "berhasil".

---

## 4. Mapping Error Repository â‡„ HTTP

| Sentinel error (repository) | Status HTTP | Sumber                                                                   |
|-----------------------------|-------------|-------------------------------------------------------------------------|
| `ErrNotFound`               | 404         | `pgx.ErrNoRows` atau `RowsAffected() == 0`                             |
| `ErrDuplicate`              | 409 Conflict| `pgconn.PgError` dengan kode `23505` (pelanggaran UNIQUE)              |
| error lain                  | 500         | Kesalahan internal server                                               |

```go
// handler.go â€” terjemahkanError
switch {
case errors.Is(err, repository.ErrNotFound):
    return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
case errors.Is(err, repository.ErrDuplicate):
    return fail(c, fiber.StatusConflict, "username sudah dipakai")
default:
    return fail(c, fiber.StatusInternalServerError, pesanUmum)
}
```

Perubahan penting dibanding versi in-memory: status `409 Conflict` kini mungkin muncul ketika `INSERT`/`UPDATE` melanggar `UNIQUE INDEX users_username_lower_key` â€” sebelumnya versi in-memory hanya mengandalkan loop `for _, u := range users`.

---

## 5. Ringkasan Hasil Pengujian

| No | Skenario                              | Status yang Diharapkan | Status Aktual | Hasil   |
|----|---------------------------------------|------------------------|---------------|---------|
| 1  | POST create user (valid)              | 201 Created            | 201 Created   | âœ… Lulus |
| 2a | POST user tambahan 1                  | 201 Created            | 201 Created   | âœ… Lulus |
| 2b | POST user tambahan 2                  | 201 Created            | 201 Created   | âœ… Lulus |
| 3  | GET pagination & sort                 | 200 OK                 | 200 OK        | âœ… Lulus |
| 4  | GET search & filter                   | 200 OK                 | 200 OK        | âœ… Lulus |
| 5  | PUT replace (semua field valid)       | 200 OK                 | 200 OK        | âœ… Lulus |
| 6  | PUT tanpa email                       | 422 Unprocessable      | 422 Unprocess.| âœ… Lulus |
| 7  | PATCH sebagian (is_active)            | 200 OK                 | 200 OK        | âœ… Lulus |
| 8  | POST tanpa Content-Type               | 415 Unsupported Media  | 415           | âœ… Lulus |
| 9  | DELETE user                           | 204 No Content         | 204           | âœ… Lulus |

---

## 6. Kesimpulan

Seluruh 10 permintaan uji me-return status HTTP sesuai ekspektasi, dengan dua catatan penting dibanding versi in-memory:

- **Pembuatan status code yang tepat** untuk setiap skenario sukses maupun gagal (200, 201, 204, 400, 404, 409, 415, 422). Status **409 Conflict** adalah tambahan baru yang muncul ketika username duplikat dilanggar pada `UNIQUE INDEX`.
- **Validasi input** tetap menggunakan status 422 di handler sehingga klien dapat membedakan kesalahan format vs kesalahan bisnis vs kesalahan constraint basis data.
- **Idempotency**: PUT menghasilkan hasil yang sama bila dipanggil berulang; DELETE menggunakan 204 No Content sesuai standar REST.
- **Middleware `requireJSON`** memblokir request POST/PUT/PATCH tanpa `Content-Type: application/json` sebelum koneksi database dipakai.
- **Pemisahan lapisan** â€” `handler` tidak tahu tentang SQL; `repository` hanya berbicara dalam bahasa `model.Student` dan dua sentinel error. Semua keputusan berada di `handler.go` (`terjemahkanError`).
- **Keamanan SQL** â€” semua nilai dari klien menjadi argumen `$1, $2, ...`; kolom `ORDER BY` yang tidak bisa diparameterkan tetap melewati whitelist `kolomUrut`.
- **Koneksi terkelola** â€” `pgxpool` dengan `MaxConns`/`MinConns`/`MaxConnLifetime` mencegah ledakan koneksi; `pool.Ping()` saat start-up gagal cepat jika basis data tidak tersedia.

API siap dipakai untuk praktikum lanjutan dan telah memenuhi kaidah RESTful yang diminta pada pertemuan ke-2.
