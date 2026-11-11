//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"students_api/app/model"
	"students_api/app/repository"
)

// ---------------------------------------------------------------------------
// Test untuk Bagian D.2.
//
// Dua test ini menguji SATU sifat yang tidak terlihat dari kode HTTP saja:
// konsistennya state KRS ketika sebuah langkah gagal.
//
// - TestSwapEndpoint_TargetFull_KeepsSourceEnrollment:
//   POST /enrollments/swap ke kelas tujuan yang sudah PENUH harus gagal
//   dengan 409 COURSE_FULL, dan mahasiswa harus TETAP terdaftar di kelas
//   asal. enrolled_count kedua kelas tidak boleh berubah, dan tidak boleh
//   ada event enrollment.swapped di outbox.
//
// - TestDropEndpoint_RemovesEnrollmentAndEmitsDroppedEvent:
//   DELETE /courses/:id/enroll harus mengembalikan 204, menghapus baris
//   enrollment, mengurangi enrolled_count satu, dan menghasilkan TEPAT SATU
//   event enrollment.dropped di outbox.
//
// Tanpa transaction, swap yang gagal akan meninggalkan mahasiswa tanpa kelas
// sama sekali: ia keluar dari kelas asal, tetapi tidak berhasil masuk kelas
// tujuan. Itulah sebabnya kedua test ini memakai PostgreSQL sungguhan.
// ---------------------------------------------------------------------------

// swapResult merangkum jawaban POST /enrollments/swap.
type swapResult struct {
	status int
	code   string
}

// swap mengirim POST /enrollments/swap atas nama student dan mengembalikan
// status HTTP beserta kode error pada body (string kosong bila sukses).
func (a api) swap(t *testing.T, student model.Student, fromCourseID, toCourseID int) swapResult {
	t.Helper()

	token, err := a.jwt.GenerateAccess(model.Student{
		ID: student.ID, Username: student.Username, Role: "user",
	})
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return swapResult{}
	}

	payload, err := json.Marshal(model.SwapRequest{
		FromCourseID: fromCourseID, ToCourseID: toCourseID,
	})
	if err != nil {
		t.Errorf("menyusun body swap: %v", err)
		return swapResult{}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap",
		strings.NewReader(string(payload)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request swap: %v", err)
		return swapResult{}
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return swapResult{status: resp.StatusCode, code: body.Code}
}

// drop mengirim DELETE /courses/:id/enroll atas nama student dan
// mengembalikan status HTTP.
func (a api) drop(t *testing.T, student model.Student, courseID int) int {
	t.Helper()

	token, err := a.jwt.GenerateAccess(model.Student{
		ID: student.ID, Username: student.Username, Role: "user",
	})
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0
	}

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request drop: %v", err)
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// enrolledCount membaca enrolled_count terkini sebuah kelas.
func enrolledCount(t *testing.T, courseID int) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT enrolled_count FROM courses WHERE id = $1`, courseID).Scan(&n); err != nil {
		t.Fatalf("membaca enrolled_count kelas %d: %v", courseID, err)
	}
	return n
}

// isEnrolled menjawab "apakah student masih terdaftar di kelas ini?".
func isEnrolled(t *testing.T, courseID, userID int) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM enrollments WHERE course_id = $1 AND user_id = $2)`,
		courseID, userID).Scan(&exists); err != nil {
		t.Fatalf("memeriksa pendaftaran kelas %d: %v", courseID, err)
	}
	return exists
}

// TestSwapEndpoint_TargetFull_KeepsSourceEnrollment membuktikan bahwa swap
// bersifat ATOMIK: kelas tujuan penuh membuat seluruh langkah dibatalkan.
//
// Empat hal diperiksa sekaligus, karena keempatnya adalah satu janji:
//  1. status 409 dengan kode COURSE_FULL;
//  2. mahasiswa TETAP terdaftar di kelas asal;
//  3. enrolled_count kedua kelas tidak berubah;
//  4. tidak ada event enrollment.swapped di outbox.
//
// Mutasi yang harus membuat test ini gagal: TxManager.runOnce melakukan
// COMMIT meski fn mengembalikan error.
func TestSwapEndpoint_TargetFull_KeepsSourceEnrollment(t *testing.T) {
	a := newAPI(t)

	from := seedCourse(t, 3, 40)
	to := seedCourse(t, 3, 1)
	student := seedStudent(t)
	other := seedStudent(t)

	ctx := context.Background()
	repo := repository.NewCourseRepository(pool)

	// Kelas tujuan dibuat PENUH oleh mahasiswa lain, sehingga swap student
	// pasti ditolak. Kuota 1 dan satu peserta sudah ada: IncrementEnrolled
	// tidak akan menemukan baris yang boleh diubah.
	if err := repo.InsertEnrollment(ctx, to.ID, other.ID); err != nil {
		t.Fatalf("seed pendaftaran kelas tujuan: %v", err)
	}
	if _, err := repo.IncrementEnrolled(ctx, to.ID); err != nil {
		t.Fatalf("seed kuota kelas tujuan: %v", err)
	}

	// Mahasiswa yang berpindah kelas harus terdaftar di kelas asal.
	if status, _ := a.enroll(t, student, from.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran kelas asal: status = %d, seharusnya 201", status)
	}

	fromBefore := enrolledCount(t, from.ID)
	toBefore := enrolledCount(t, to.ID)

	res := a.swap(t, student, from.ID, to.ID)
	if res.status != http.StatusConflict || res.code != "COURSE_FULL" {
		t.Fatalf("swap ke kelas penuh: dapat %d %s, seharusnya 409 COURSE_FULL",
			res.status, res.code)
	}

	if !isEnrolled(t, from.ID, student.ID) {
		t.Errorf("mahasiswa tidak lagi terdaftar di kelas asal; swap yang gagal harus membatalkan seluruh langkah")
	}
	if got := enrolledCount(t, from.ID); got != fromBefore {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap %d", got, fromBefore)
	}
	if got := enrolledCount(t, to.ID); got != toBefore {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap %d", got, toBefore)
	}
	if got := countRows(t,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`,
		to.ID, student.ID); got != 0 {
		t.Errorf("mahasiswa terdaftar di kelas tujuan = %d, seharusnya 0", got)
	}
	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentSwapped)
	if events != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0", events)
	}
}

// TestDropEndpoint_RemovesEnrollmentAndEmitsDroppedEvent membuktikan bahwa
// keluar dari kelas menghapus pendaftaran, mengurangi jumlah peserta satu,
// dan mencatat TEPAT SATU event enrollment.dropped.
//
// Mutasi yang harus membuat test ini gagal: DecrementEnrolled tidak
// mengurangi enrolled_count.
func TestDropEndpoint_RemovesEnrollmentAndEmitsDroppedEvent(t *testing.T) {
	a := newAPI(t)

	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran: status = %d, seharusnya 201", status)
	}
	if got := enrolledCount(t, course.ID); got != 1 {
		t.Fatalf("enrolled_count setelah mendaftar = %d, seharusnya 1", got)
	}

	if status := a.drop(t, student, course.ID); status != http.StatusNoContent {
		t.Fatalf("keluar kelas: status = %d, seharusnya 204", status)
	}

	if isEnrolled(t, course.ID, student.ID) {
		t.Errorf("baris enrollment masih ada; keluar kelas harus menghapus pendaftaran")
	}
	if got := enrolledCount(t, course.ID); got != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0 (berkurang satu)", got)
	}
	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped)
	if events != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat 1", events)
	}
}
