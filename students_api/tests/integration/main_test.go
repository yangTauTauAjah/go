//go:build integration

// Package integration_test berisi integration test: test yang berbicara
// dengan PostgreSQL sungguhan. Baris go:build di atas membuat file ini
// HANYA ikut dikompilasi bila perintahnya memakai -tags=integration,
// sehingga "go test ./..." biasa tetap cepat dan tidak butuh database.
package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"students_api/app/model"
)

// pool dipakai bersama oleh seluruh test di package ini.
var pool *pgxpool.Pool

// TestMain berjalan SEKALI sebelum seluruh test di package ini. Ia
// menyiapkan koneksi dan skema, lalu memanggil m.Run() untuk menjalankan
// test satu per satu.
func TestMain(m *testing.M) {
	// Lokasi relatif terhadap folder package ini: tests/integration.
	_ = godotenv.Load(filepath.Join("..", "..", ".env.test"))

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL tidak diisi: integration test dilewati")
		os.Exit(0)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Println("TEST_DATABASE_URL tidak valid:", err)
		os.Exit(1)
	}

	// PENGAMAN. Integration test menulis dan menghapus data. Test hanya
	// boleh berjalan pada database yang namanya berakhiran _test, agar
	// salah ketik pada .env.test tidak pernah menyentuh database development.
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		fmt.Printf("menolak berjalan: database %q tidak berakhiran _test\n", cfg.ConnConfig.Database)
		os.Exit(1)
	}

	// Pool harus punya cukup koneksi agar test konkurensi benar-benar
	// berjalan SERENTAK. Dengan MaxConns = 1, seluruh request antre di
	// satu koneksi: request saling mendahului satu per satu, tidak pernah
	// bertabrakan, dan bug konkurensi (mis. tanpa FOR UPDATE) tidak
	// pernah bisa terlihat. Production memakai DB_MAX_CONNS (bawaan 10),
	// dan test memakai 10 request serentak, jadi pool test diberi 20.
	cfg.MaxConns = 20
	cfg.MinConns = 2

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		fmt.Println("gagal terhubung ke database test:", err)
		os.Exit(1)
	}

	if err := applyMigrations(ctx); err != nil {
		fmt.Println("gagal menjalankan migration:", err)
		os.Exit(1)
	}

	// Pemanasan pool: seluruh koneksi dibuka SEBELUM test apa pun jalan.
	//
	// Tanpa ini, repetisi pertama memakai pool yang baru dingin: sebagian
	// besar request harus menunggu koneksi dibuat, sehingga request tidak
	// benar-benar berangkat serentak dan bug konkurensi lolos pada
	// repetisi pertama saja. Setelah dipanaskan, semua repetisi punya
	// kondisi yang sama.
	warmUpPool(ctx, int(cfg.MaxConns))

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// warmUpPool membuka sebanyak mungkin koneksi sekaligus lalu mengembalikannya
// ke pool, sehingga seluruh koneksi sudah ada dan siap dipinjam ketika test
// konkurensi mulai.
func warmUpPool(ctx context.Context, want int) {
	acquired := make(chan *pgxpool.Conn, want)
	var wg sync.WaitGroup
	for i := 0; i < want; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := pool.Acquire(ctx)
			if err != nil {
				return
			}
			acquired <- c
		}()
	}
	wg.Wait()
	close(acquired)
	for c := range acquired {
		c.Release()
	}
}

// applyMigrations menjalankan seluruh file migrations/*.sql berurutan.
// Semua migration kita ditulis dengan IF NOT EXISTS dan ON CONFLICT,
// sehingga aman dijalankan berulang kali.
func applyMigrations(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		return err
	}
	slices.Sort(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// uniqueSuffix menghasilkan akhiran acak agar data milik setiap test tidak
// pernah bertabrakan dengan data test lain, termasuk saat dijalankan ulang.
func uniqueSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("membuat akhiran acak: %v", err)
	}
	return hex.EncodeToString(b)
}

// seedStudent membuat satu mahasiswa khusus untuk test yang memanggilnya.
//
// t.Cleanup mendaftarkan pembersihan yang dijalankan SETELAH test selesai
// — berhasil maupun gagal. Seluruh data turunannya (enrollments,
// notifications) ikut terhapus lewat ON DELETE CASCADE.
func seedStudent(t *testing.T) model.Student {
	t.Helper()
	ctx := context.Background()
	s := uniqueSuffix(t)

	u := model.Student{Username: "it_" + s, Email: "it_" + s + "@student.unair.ac.id", Role: "user"}
	err := pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role) VALUES ($1, $2, '!', 'user') RETURNING 
id`,
		u.Username, u.Email).Scan(&u.ID)
	if err != nil {
		t.Fatalf("seed mahasiswa: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM outbox WHERE (payload->>'user_id')::INT = $1`, u.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return u
}

// seedCourse membuat satu kelas khusus untuk test yang memanggilnya.
//
// Pembersihan memakai t.Cleanup, BUKAN defer. defer di dalam fungsi ini
// akan berjalan begitu seedCourse selesai mengembalikan nilai — yaitu
// kelas dihapus SEBELUM test memakainya. Akibatnya setiap query yang
// menyentuh kelas itu gagal dengan "data tidak ditemukan".
// t.Cleanup menunda penghapusan sampai test benar-benar selesai.
func seedCourse(t *testing.T, credits, quota int) model.Course {
	t.Helper()
	ctx := context.Background()

	c := model.Course{Code: "IT" + uniqueSuffix(t), Name: "Kelas Uji", Credits: credits, Quota: quota}
	err := pool.QueryRow(ctx,
		`INSERT INTO courses (code, name, credits, quota) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.Code, c.Name, c.Credits, c.Quota).Scan(&c.ID)
	if err != nil {
		t.Fatalf("seed kelas: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, c.ID)
	})
	return c
}
