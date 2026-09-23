Nama: **Habib Anwash**
NIM: **434241033**
Repo URL: https://github.com/yangTauTauAjah/go

## **1. Testing Endpoint**

### ** 1. GET — /students (list, admin)**

#### Request

```
GET /api/v1/students/
Authorization: Bearer <tokenAdmin>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 2. GET — /students (list, staff)**

#### Request

```
GET /api/v1/students/
Authorization: Bearer <tokenStaff>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 3. GET — /students (list, user)**

#### Request

```
GET /api/v1/students/
Authorization: Bearer <tokenUser>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 4. GET — /students/:id diri sendiri (admin/staff/user)**

#### Request

```
GET /api/v1/students/1
Authorization: Bearer <tokenAdmin>
```

(Untuk peran staff gunakan id=2, untuk peran user gunakan id=3.)

#### Screenshot pengujian

[[image placeholder]]

### ** 5. GET — /students/:id milik orang lain (admin)**

#### Request

```
GET /api/v1/students/2
Authorization: Bearer <tokenAdmin>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 6. GET — /students/:id milik orang lain (staff)**

#### Request

```
GET /api/v1/students/3
Authorization: Bearer <tokenStaff>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 7. GET — /students/:id milik orang lain (user)**

#### Request

```
GET /api/v1/students/1
Authorization: Bearer <tokenUser>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 8. PUT — /students/:id milik orang lain (admin)**

#### Request

```
PUT /api/v1/students/2
Authorization: Bearer <tokenAdmin>
Content-Type: application/json

{
  "username": "staff_updated",
  "email": "staff_new@unair.ac.id",
  "is_active": true
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 9. PUT — /students/:id milik orang lain (staff)**

#### Request

```
PUT /api/v1/students/1
Authorization: Bearer <tokenStaff>
Content-Type: application/json

{
  "username": "admin_updated",
  "email": "admin_new@unair.ac.id",
  "is_active": true
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 10. PUT — /students/:id milik orang lain (user)**

#### Request

```
PUT /api/v1/students/1
Authorization: Bearer <tokenUser>
Content-Type: application/json

{
  "username": "user_updated",
  "email": "user_new@unair.ac.id",
  "is_active": true
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 11. DELETE — /students/:id milik orang lain (admin)**

#### Request

```
DELETE /api/v1/students/2
Authorization: Bearer <tokenAdmin>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 12. DELETE — /students/:id milik orang lain (staff)**

#### Request

```
DELETE /api/v1/students/1
Authorization: Bearer <tokenStaff>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 13. DELETE — /students/:id milik orang lain (user)**

#### Request

```
DELETE /api/v1/students/1
Authorization: Bearer <tokenUser>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 14. DELETE — /students/:id diri sendiri (admin)**

#### Request

```
DELETE /api/v1/students/1
Authorization: Bearer <tokenAdmin>
```

(Untuk peran staff gunakan id=2, untuk peran user gunakan id=3.)

#### Screenshot pengujian

[[image placeholder]]

### ** 15. PATCH — /students/:id/role milik orang lain (admin)**

#### Request

```
PATCH /api/v1/students/3/role
Authorization: Bearer <tokenAdmin>
Content-Type: application/json

{
  "role": "staff"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 16. PATCH — /students/:id/role milik orang lain (staff)**

#### Request

```
PATCH /api/v1/students/1/role
Authorization: Bearer <tokenStaff>
Content-Type: application/json

{
  "role": "admin"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 17. PATCH — /students/:id/role milik orang lain (user)**

#### Request

```
PATCH /api/v1/students/1/role
Authorization: Bearer <tokenUser>
Content-Type: application/json

{
  "role": "admin"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 18. PATCH — /students/:id/role diri sendiri (admin)**

#### Request

```
PATCH /api/v1/students/1/role
Authorization: Bearer <tokenAdmin>
Content-Type: application/json

{
  "role": "staff"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 19. PATCH — /students/:id/role diri sendiri (staff)**

#### Request

```
PATCH /api/v1/students/2/role
Authorization: Bearer <tokenStaff>
Content-Type: application/json

{
  "role": "admin"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 20. PATCH — /students/:id/role diri sendiri (user)**

#### Request

```
PATCH /api/v1/students/3/role
Authorization: Bearer <tokenUser>
Content-Type: application/json

{
  "role": "admin"
}
```

#### Screenshot pengujian

[[image placeholder]]

### ** 21. Tanpa Authorization header — GET list (semua peran)**

#### Request

```
GET /api/v1/students/
```

(Tidak ada header Authorization; response yang diharapkan adalah 401.)

#### Screenshot pengujian

[[image placeholder]]

### ** 22. Tanpa Authorization header — PUT (semua peran)**

#### Request

```
PUT /api/v1/students/1
Content-Type: application/json

{
  "username": "x",
  "email": "x@example.com",
  "is_active": true
}
```

(Tidak ada header Authorization; response yang diharapkan adalah 401.)

#### Screenshot pengujian

[[image placeholder]]

### ** 23. Tanpa Authorization header — DELETE (semua peran)**

#### Request

```
DELETE /api/v1/students/1
```

(Tidak ada header Authorization; response yang diharapkan adalah 401.)

#### Screenshot pengujian

[[image placeholder]]

### ** 24. Tanpa Authorization header — PATCH role (semua peran)**

#### Request

```
PATCH /api/v1/students/1/role
Content-Type: application/json

{
  "role": "staff"
}
```

(Tidak ada header Authorization; response yang diharapkan adalah 401.)

#### Screenshot pengujian

[[image placeholder]]

### ** 25. GET — /students dengan token lama setelah role berubah (user lama)**

#### Request

```
GET /api/v1/students/
Authorization: Bearer <tokenUserLama>
```

#### Screenshot pengujian

[[image placeholder]]

### ** 26. GET — /students setelah login ulang (user baru)**

#### Request

```
GET /api/v1/students/
Authorization: Bearer <tokenUserBaru>
```

#### Screenshot pengujian

[[image placeholder]]

## **2. Test Summary**


| No  | Skenario                                               | Peran            | Expected | Actual | Hasil  |
| --- | ------------------------------------------------------ | ---------------- | -------- | ------ | ------ |
| R1  | GET /students (list)                                   | admin            | 200      | 200    | ✅Pass |
| R1  | GET /students (list)                                   | staff            | 200      | 200    | ✅Pass |
| R1  | GET /students (list)                                   | user             | 403      | 403    | ✅Pass |
| R2  | GET /students/:id diri sendiri                         | admin/staff/user | 200      | 200    | ✅Pass |
| R3  | GET /students/:id milik orang lain                     | admin            | 200      | 200    | ✅Pass |
| R3  | GET /students/:id milik orang lain                     | staff            | 200      | 200    | ✅Pass |
| R3  | GET /students/:id milik orang lain                     | user             | 403      | 403    | ✅Pass |
| R4  | PUT /students/:id milik orang lain                     | admin            | 200      | 200    | ✅Pass |
| R4  | PUT /students/:id milik orang lain                     | staff            | 403      | 403    | ✅Pass |
| R4  | PUT /students/:id milik orang lain                     | user             | 403      | 403    | ✅Pass |
| R5  | DELETE /students/:id milik orang lain                  | admin            | 204      | 204    | ✅Pass |
| R5  | DELETE /students/:id milik orang lain                  | staff            | 403      | 403    | ✅Pass |
| R5  | DELETE /students/:id milik orang lain                  | user             | 403      | 403    | ✅Pass |
| R6  | DELETE /students/:id diri sendiri                      | admin/staff/user | 403      | 403    | ✅Pass |
| R7  | PATCH /students/:id/role milik orang lain              | admin            | 200      | 200    | ✅Pass |
| R7  | PATCH /students/:id/role milik orang lain              | staff            | 403      | 403    | ✅Pass |
| R7  | PATCH /students/:id/role milik orang lain              | user             | 403      | 403    | ✅Pass |
| R8  | PATCH /students/:id/role diri sendiri                  | admin            | 422      | 422    | ✅Pass |
| R8  | PATCH /students/:id/role diri sendiri                  | staff            | 403      | 403    | ✅Pass |
| R8  | PATCH /students/:id/role diri sendiri                  | user             | 403      | 403    | ✅Pass |
| R9  | Tanpa Authorization header                             | semua peran      | 401      | 401    | ✅Pass |
| R10 | GET dengan token lama setelah role berubah             | user (lama)      | 403      | 403    | ✅Pass |
| R10 | GET setelah login ulang (token baru membawa role baru) | user (baru)      | 200      | 200    | ✅Pass |

---

## **3. Additional Discussion **

### **3.1 Why not to authorize from middleware?**

Di `middleware/authz.go`, `RequirePermission` bekerja dengan **satu informasi saja**: peran dari token. Ia tidak tahu id baris data yang akan disentuh, karena id itu baru tersedia *setelah* Fiber memparsing path parameter `/students/:id` dan route handler mulai berjalan. Akibatnya, keputusan "user ini boleh menyentuh user id=N karena N == current.UserID" hanya bisa dibuat **setelah** id itu diketahui, yaitu di dalam service handler. Pada kode saya, pemeriksaannya duduk di `app/service/authz_rules.go`:

````go
// filepath: students_api/app/service/authz_rules.go
func CanAccessStudent(
    current model.AuthUser,
    targetID int,
    perms *helper.PermissionSet,
    anyPermission string,
) bool {
    if current.UserID == targetID {
        return true
    }
    return perms.Can(current.Role, anyPermission)
}
````

Dipanggil dari `app/service/user_service.go::Get`:

````go
// filepath: students_api/app/service/user_service.go
// ...existing code...
user, ok := helper.CurrentUser(c)
if !ok {
    return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
}
id, valid := helper.ParamID(c)
if !valid {
    return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
}
if !CanAccessStudent(current, id, s.perms, "user:read:any") {
    return helper.Fail(c, fiber.StatusForbidden,
        "tidak berhak mengakses data user lain")
}
user, err := s.repo.FindByID(ctx, id)
// ...existing code...
````

Kalau aturan "milik sendiri ATAU punya permission `:any`" itu dipaksakan ke middleware, middleware harus mengekstrak `c.Params("id")` sendiri dan menduplikasi logika pada **setiap** handler `/:id` (Get, Replace, Patch, Delete, AssignRole). Setiap titik duplikasi adalah potensi drift — handler `Get` mungkin memeriksa `current.UserID == id`, handler `Delete` kelupaan, dan celah keamanan terbuka tanpa terlihat.

Middleware yang **bisa** dipasang di route, dan memang sudah dipasang, adalah `RequirePermission(perms, "student:list")` pada `students.Get("/", …)`. Keputusan itu **tidak** bergantung pada id — "siapa yang boleh list?" adalah peran, bukan id. Itulah batasan yang jelas: **yang bergantung pada data → service; yang hanya bergantung pada peran → middleware**.

### **3.2 Pemeriksaan akses route & service**

Risiko konkret: **inkonsistensi akibat drift permission string**. Route di `route/route.go` lulus permission seperti `"student:list"`, `"student:update:any"`, `"student:delete"`, `"role:assign"` ke `RequirePermission`. Service handler di `user_service.go` melewati `"user:read:any"` (lihat baris di atas) ke `CanAccessStudent`. Bila di kemudian hari seseorang mengubah `student:list` di route tetapi lupa memperbarui konstanta `user:read:any` di service (atau sebaliknya), akan ada dua jalur masuk: satu route yang langsung menerima (200), satu service yang menolak (403), untuk permintaan yang kelihatannya identik. Penyerang yang menemukan rute yang luput akan mendapat akses; pengguna sah yang kena rute yang salah akan terkunci.

Risiko ini berkurang karena saya memakai *typed wrapper* di `middleware/authz.go` dan *named variable* di service, sehingga string permission **hanya ditulis sekali per lokasi**. Saya mitigasi lebih jauh dengan tiga hal:

1. **Test untuk matrix peran**. `api_test.http` dan folder "RBAC" di Postman collection menjalankan semua kombinasi peran × endpoint; baris mana pun yang menjawab di luar sel tabel §5.1 akan langsung kelihatan.
2. **Test untuk service-side check** (`CanAccessStudent`) — saat ini masih berupa penggunaan langsung; kontribusi berikutnya yang layak adalah membungkus pemanggilannya sebagai `assertCanAccess(...)` agar semua handler `/:id` melewati satu fungsi yang sama, dan rename permission di satu tempat.
3. **Daftar permission ada di `config/app.go`** sebagai konstanta — rename dilakukan di sana, lalu `go build` mengurai seluruh pemakaian dan menolak bila ada ketikgalan.

### **3.3 RBAC masih memadai?**

Terkait concern  "dosen wali hanya boleh melihat mahasiswa bimbingannya", ini tidak memadai. RBAC menjawab pertanyaan *"peran apa yang boleh melakukan aksi X"*. Kebutuhan dosen wali menjawab pertanyaan berbeda: *"aksi X boleh dilakukan pada baris **Y** bila relasi wali–mahasiswa mengandung (dosen_id, mahasiswa_id) = (current, Y)"*. Itu adalah **row-level access control (ReBAC / row-level security)**.

Wujud minimal yang perlu ditambahkan:

1. **Tabel relasi baru** (mis. `dosen_wali(user_id INT NOT NULL, student_id INT NOT NULL, PRIMARY KEY(user_id, student_id))`) — menggantikan relasi owner_id sederhana dengan relasi banyak-ke-banyak.
2. **Predikat akses** di service: `CanAccessStudent(current, targetID, perms)` tumbuh menjadi `CanAccessStudent(current, targetID, perms, repo)` yang melakukan satu query `SELECT 1 FROM dosen_wali WHERE user_id=$1 AND student_id=$2` sebelum memutuskan mengizinkan akses. Predikat dievaluasi oleh `RequirePermission` **tidak** mungkin (middleware tidak punya akses ke DB), sehingga peran rule (RBAC) tetap dipasang di route sebagai pagar pertama, dan row-level rule dipasang di service sebagai pagar kedua.
3. **Daftar permission diperluas**: `student:read:any` (admin), `student:read:bimbingannya` (dosen). Permission string tetap satu sumber kebenaran di `config/app.go`.
4. **Resource server / PDP** bila aturan makin banyak: ketika peran × relasi × resource sudah saling silang, pindahkan keputusan ke luar handler (mis. OPA, Casbin, atau tabel `policy_rules`) sehingga service tidak perlu menulis ulang `if` bertingkat setiap ada relasi baru.

Ringkasnya: RBAC adalah pagar kasar di layer route. Kebutuhan dosen wali membutuhkan pagar halus di layer service yang **tahu data per baris**, dan idealnya dipisahkan lagi bila kombinasi aturan makin banyak.
