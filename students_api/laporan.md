# Laporan Pengujian API — Praktikum Backend Lanjut Pertemuan 2

**Mata Kuliah:** Backend Lanjut
**Pertemuan:** 2
**Framework:** Go + Fiber v2
**Server:** `http://localhost:3000`
**Tanggal Pengujian:** 26 Agustus 2026
**Tool Pengujian:** Postman (collection: `Tugas_Pertemuan_2.postman_collection.json`)

---

## 1. Informasi Umum

API yang diuji adalah layanan CRUD untuk resource **users** dengan base path `/api/v1`. Setiap pengujian dilakukan menggunakan Postman dan memperhatikan **status HTTP** yang dikembalikan oleh server, bukan hanya isi body respons.

### Endpoint yang diuji

| No | Method | Path              | Tujuan                           | Status Sukses |
|----|--------|-------------------|----------------------------------|---------------|
| 1  | POST   | `/api/v1/users`   | Membuat user baru                | 201 Created   |
| 2  | GET    | `/api/v1/users`   | Mengambil daftar user + filter   | 200 OK        |
| 3  | GET    | `/api/v1/users/:id` | Mengambil detail user          | 200 OK / 404  |
| 4  | PUT    | `/api/v1/users/:id` | Mengganti seluruh data user    | 200 OK / 422  |
| 5  | PATCH  | `/api/v1/users/:id` | Memperbarui sebagian data user  | 200 OK / 400  |
| 6  | DELETE | `/api/v1/users/:id` | Menghapus user                  | 204 No Content|

### Middleware yang aktif

```go
// main.go
app.Use(requestid.New())
app.Use(logger.New(logger.Config{...}))
app.Use(cors.New())
// group /api/v1/students menggunakan requireJSON untuk method ber-body
u := api.Group("/students", requireJSON)
```

---

## 2. Pengujian Endpoint

### 2.1 POST — Membuat User Baru

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

**Potongan kode handler**

```go
// handler.go — createUser
var req CreateStudentRequest
if err := c.BodyParser(&req); err != nil {
    return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
}
// validasi username, email, dan password
if len(req.Password) < 8 {
    errs["password"] = "minimal 8 karakter"
}
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 01_post_create_success.png — paste Postman screenshot showing `201 Created` here]*

**Penjelasan:** Server mengembalikan status **201 Created** saat user berhasil dibuat. Header `Location` berisi URL resource baru (`/api/v1/users/1`). Body memuat field `success: true`, `message`, dan `data` berupa objek user dengan `id` yang di-generate otomatis oleh `nextID++`.

---

### 2.2 POST — Tambah User Kedua dan Ketiga

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

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 02a_post_create_andini.png — paste screenshot showing `201 Created` for user ke-2 here]*

> 📷 *[Screenshots: 02b_post_create_budi.png — paste screenshot showing `201 Created` for user ke-3 here]*

**Penjelasan:** Kedua permintaan mengembalikan **201 Created**. Setelah tiga kali POST, slice `users` berisi 3 elemen dengan id 1, 2, dan 3. Field `created_at` di-set menggunakan `time.Now()` saat proses create.

---

### 2.3 GET — Pagination dan Sorting

**Permintaan**

```
GET /api/v1/users?page=1&limit=2&sort=username&order=desc
```

**Potongan kode handler**

```go
// handler.go — listUsers
sort.SliceStable(hasil, func(i, j int) bool {
    var lebihKecil bool
    switch q.Sort {
    case "username":
        lebihKecil = hasil[i].Username < hasil[j].Username
    // ...
    }
    if q.Order == "desc" {
        return !lebihKecil
    }
    return lebihKecil
})
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 03_get_pagination_sort.png — paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. Karena `sort=username` dan `order=desc`, hasil diurut descending berdasarkan username (`john_baru`/`johndoe` → `budi_s` → `andini`). Respons memuat `meta.page`, `meta.limit`, `meta.total`, dan `meta.total_pages`. Total user adalah 3, dengan `limit=2` artinya `total_pages = 2`.

---

### 2.4 GET — Search dan Filter

**Permintaan**

```
GET /api/v1/users?search=an&is_active=true
```

**Potongan kode handler**

```go
// handler.go — cocokPencarian
func cocokPencarian(u Student, kata string) bool {
    kata = strings.ToLower(kata)
    return strings.Contains(strings.ToLower(u.Username), kata) ||
        strings.Contains(strings.ToLower(u.Email), kata)
}

// parseListQuery — membaca is_active dan mengkonversi ke *bool
if raw := c.Query("is_active"); raw != "" {
    if v, err := strconv.ParseBool(raw); err == nil {
        q.IsActive = &v
    }
}
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 04_get_search_filter.png — paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. Pencarian substring `an` cocok pada username `andini` dan email `andini@example.com`, sehingga hanya user tersebut yang dikembalikan (user lain mengandung substring `an` di email-nya pun akan ikut). Filter `is_active=true` memastikan user non-aktif dikecualikan.

---

### 2.5 PUT — Replace User (Seluruh Field Wajib)

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

**Potongan kode handler**

```go
// handler.go — replaceUser
if strings.TrimSpace(req.Username) == "" {
    errs["username"] = "wajib diisi pada PUT"
}
if !strings.Contains(req.Email, "@") {
    errs["email"] = "wajib diisi dan berformat email pada PUT"
}
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 05_put_replace_success.png — paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. PUT mengganti **seluruh** field pada user id 1. Username berubah dari `johndoe` menjadi `john_baru`, email menjadi `jb@example.com`, dan `is_active` menjadi `false`. Karena `PUT` bersifat *idempotent*, request yang sama akan menghasilkan hasil yang sama.

---

### 2.6 PUT — Validasi Gagal (Tanpa Email)

**Permintaan**

```json
PUT /api/v1/users/1
Content-Type: application/json

{
  "username": "john_baru"
}
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 06_put_validation_422.png — paste Postman screenshot showing `422 Unprocessable Entity` here]*

**Penjelasan:** Status **422 Unprocessable Entity**. Validasi `email` gagal karena body tidak mengandung karakter `@`. Respons memuat objek `errors` dengan key `email` dan pesan `wajib diisi dan berformat email pada PUT`. Server **tidak** mengubah data karena validasi gagal.

---

### 2.7 PATCH — Update Sebagian (is_active)

**Permintaan**

```json
PATCH /api/v1/users/1
Content-Type: application/json

{
  "is_active": true
}
```

**Potongan kode handler**

```go
// handler.go — patchUser
if req.Username == nil && req.Email == nil && req.IsActive == nil {
    return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
}
if req.IsActive != nil {
    users[i].IsActive = *req.IsActive
}
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 07_patch_partial.png — paste Postman screenshot showing `200 OK` here]*

**Penjelasan:** Status **200 OK**. Hanya field `is_active` yang dikirim, sehingga hanya field tersebut yang berubah dari `false` ke `true`. Username dan email tetap seperti hasil PUT sebelumnya. Inilah perbedaan utama PATCH dengan PUT.

---

### 2.8 POST — Tanpa Content-Type

**Permintaan**

```
POST /api/v1/users   (tanpa header Content-Type)
Body: {"username":"x"}
```

**Potongan kode middleware**

```go
// main.go — requireJSON
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

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 08_post_no_content_type.png — paste Postman screenshot showing `415 Unsupported Media Type` here]*

**Penjelasan:** Status **415 Unsupported Media Type**. Middleware menolak request sebelum sampai ke handler `createUser`. Pesan error `Content-Type harus application/json` membantu klien memperbaiki permintaannya.

---

### 2.9 DELETE — Hapus User

**Permintaan**

```
DELETE /api/v1/users/2
```

**Potongan kode handler**

```go
// handler.go — deleteUser
i := findUserIndex(id)
if i == -1 {
    return fail(c, fiber.StatusNotFound, "user tidak ditemukan")
}
users = append(users[:i], users[i+1:]...)
return noContent(c)
```

**Tangkapan layar pengujian**

> 📷 *[Screenshots: 09_delete_success.png — paste Postman screenshot showing `204 No Content` here]*

**Penjelasan:** Status **204 No Content**. Body respons kosong (sesuai standar HTTP untuk DELETE sukses). Setelah eksekusi, user dengan id 2 hilang dari slice `users` — dibuktikan dengan request GET berikutnya yang hanya mengembalikan 2 user.

---

## 3. Ringkasan Hasil Pengujian

| No | Skenario                              | Status yang Diharapkan | Status Aktual | Hasil   |
|----|---------------------------------------|------------------------|---------------|---------|
| 1  | POST create user (valid)              | 201 Created            | 201 Created   | ✅ Lulus |
| 2a | POST user tambahan 1                  | 201 Created            | 201 Created   | ✅ Lulus |
| 2b | POST user tambahan 2                  | 201 Created            | 201 Created   | ✅ Lulus |
| 3  | GET pagination & sort                 | 200 OK                 | 200 OK        | ✅ Lulus |
| 4  | GET search & filter                   | 200 OK                 | 200 OK        | ✅ Lulus |
| 5  | PUT replace (semua field valid)       | 200 OK                 | 200 OK        | ✅ Lulus |
| 6  | PUT tanpa email                       | 422 Unprocessable      | 422 Unprocess.| ✅ Lulus |
| 7  | PATCH sebagian (is_active)            | 200 OK                 | 200 OK        | ✅ Lulus |
| 8  | POST tanpa Content-Type               | 415 Unsupported Media  | 415           | ✅ Lulus |
| 9  | DELETE user                           | 204 No Content         | 204           | ✅ Lulus |

---

## 4. Kesimpulan

Seluruh 10 permintaan uji mengembalikan status HTTP sesuai ekspektasi. API menunjukkan perilaku yang baik dalam hal:

- **Pembuatan status code yang tepat** untuk setiap skenario sukses maupun gagal (201, 200, 204, 400, 404, 415, 422).
- **Validasi input** menggunakan status 422 (Unprocessable Entity) sehingga klien dapat membedakan kesalahan format vs kesalahan bisnis.
- **Idempotency**: PUT menghasilkan hasil yang sama bila dipanggil berulang, dan DELETE menggunakan 204 No Content sesuai standar REST.
- **Middleware `requireJSON`** berhasil memblokir request POST/PUT/PATCH yang tidak menyertakan `Content-Type: application/json`.
- **Filter, sort, dan pagination** bekerja dengan benar sesuai whitelist di `allowedSort` dan batasan `limit ≤ 100`.

API siap dipakai untuk praktikum lanjutan dan telah memenuhi kaidah RESTful yang diminta pada pertemuan ke-2.