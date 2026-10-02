// Package main menjalankan seeder untuk database SIAKAD Mini.
//
// Cara pakai:
//
//	DB_HOST=... DB_NAME=... DB_USER=... DB_PASSWORD=... go run ./cmd/seed
//
// Apa yang dilakukan:
//  1. Mengeksekusi file migrations/*.sql (secara idempotent).
//  2. Menambahkan 1 admin (email: admin@siakad.test, password: admin123).
//  3. Menambahkan 20 mahasiswa (password awal = NIM, di-hash bcrypt).
//  4. Menambahkan 10 mata kuliah dengan kuota realistis.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"siakad/config"
)

type seedCourse struct {
	KodeMK   string
	NamaMK   string
	SKS      int
	Semester int
	Kuota    int
}

var defaultCourses = []seedCourse{
	{"IF101", "Algoritma dan Pemrograman", 3, 1, 40},
	{"IF102", "Struktur Data", 3, 2, 35},
	{"IF201", "Basis Data", 3, 3, 35},
	{"IF202", "Pemrograman Berorientasi Objek", 3, 3, 30},
	{"IF301", "Sistem Operasi", 3, 4, 30},
	{"IF302", "Jaringan Komputer", 3, 4, 30},
	{"IF401", "Rekayasa Perangkat Lunak", 3, 5, 25},
	{"IF402", "Kecerdasan Buatan", 3, 6, 25},
	{"IF501", "Keamanan Informasi", 3, 7, 20},
	{"IF502", "Cloud Computing", 3, 7, 20},
}

func main() {
	config.LoadEnv()
	ctx := context.Background()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.GetEnv("DB_USER", "postgres"),
		config.GetEnv("DB_PASSWORD", ""),
		config.GetEnv("DB_HOST", "localhost"),
		config.GetEnv("DB_PORT", "5432"),
		config.GetEnv("DB_NAME", "siakad"),
		config.GetEnv("DB_SSLMODE", "disable"),
	)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer conn.Close(ctx)

	// 1. Eksekusi migrations (idempotent)
	if err := applyMigrations(ctx, conn); err != nil {
		log.Fatalf("gagal menerapkan migrations: %v", err)
	}

	// 2. Admin
	if err := seedAdmin(ctx, conn); err != nil {
		log.Fatalf("gagal men-seed admin: %v", err)
	}

	// 3. Mahasiswa
	if err := seedMahasiswa(ctx, conn); err != nil {
		log.Fatalf("gagal men-seed mahasiswa: %v", err)
	}

	// 4. Mata kuliah
	if err := seedCourses(ctx, conn); err != nil {
		log.Fatalf("gagal men-seed mata kuliah: %v", err)
	}
	log.Println("Seeder selesai")
}

func applyMigrations(ctx context.Context, conn *pgx.Conn) error {
	pattern := filepath.Join("migrations", "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("tidak ada file .sql di %s", pattern)
	}
	sort.Strings(files)
	for _, f := range files {
		log.Printf("menerapkan %s", f)
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		// Tulis placeholder hash admin di 005_seed.sql tidak dipakai —
		// hash di-generate ulang di seedAdmin.
		cleaned := stripSeedAdmin(string(data))
		if _, err := conn.Exec(ctx, cleaned); err != nil {
			return fmt.Errorf("eksekusi %s: %w", f, err)
		}
	}
	return nil
}

// stripSeedAdmin menghapus baris INSERT users pada 005_seed.sql agar
// seedAdmin dapat menyisipkan admin dengan hash yang benar-benar valid.
func stripSeedAdmin(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "INSERT INTO users") {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func seedAdmin(ctx context.Context, conn *pgx.Conn) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx,
		`INSERT INTO users (email, password, role, is_active)
 VALUES ($1, $2, 'admin', TRUE)
 ON CONFLICT (LOWER(email)) DO NOTHING`,
		"admin@siakad.test", string(hash))
	return err
}

func seedMahasiswa(ctx context.Context, conn *pgx.Conn) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	prodies := []string{"Sistem Informasi", "Teknik Informatika", "Ilmu Komputer"}

	for i := 1; i <= 20; i++ {
		nim := fmt.Sprintf("187221%06d", 1+i) // 12 digit
		email := fmt.Sprintf("mhs%02d@siakad.test", i)
		nama := randomNama(rng, i)
		prodi := prodies[rng.Intn(len(prodies))]
		angkatan := 2021 + rng.Intn(4)
		ipk := 2.0 + rng.Float64()*2.0 // 2.0 - 4.0
		ipk = float64(int(ipk*100)) / 100.0

		hash, err := bcrypt.GenerateFromPassword([]byte(nim), 12)
		if err != nil {
			return err
		}
		var userID int
		err = conn.QueryRow(ctx,
			`INSERT INTO users (email, password, role, is_active)
 VALUES ($1, $2, 'mahasiswa', TRUE)
 ON CONFLICT (LOWER(email)) DO UPDATE SET email = EXCLUDED.email
 RETURNING id`,
			email, string(hash),
		).Scan(&userID)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", email, err)
		}
		_, err = conn.Exec(ctx,
			`INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
 VALUES ($1, $2, $3, $4, $5, $6)
 ON CONFLICT (nim) DO UPDATE SET nama = EXCLUDED.nama`,
			userID, nim, nama, prodi, angkatan, ipk,
		)
		if err != nil {
			return fmt.Errorf("insert student %s: %w", nim, err)
		}
	}
	return nil
}

func randomNama(rng *rand.Rand, idx int) string {
	depan := []string{"Rina", "Budi", "Siti", "Agus", "Dewi", "Andi", "Putri",
		"Rizky", "Anisa", "Fajar", "Maya", "Dimas", "Tari", "Yoga",
		"Indah", "Bayu", "Nadia", "Reza", "Sari", "Hadi"}
	belakang := []string{"Putri", "Wijaya", "Lestari", "Pratama", "Anggraini",
		"Saputra", "Maulana", "Permata", "Nugroho", "Sari"}
	return fmt.Sprintf("%s %s %d", depan[(idx-1)%len(depan)], belakang[rng.Intn(len(belakang))], idx)
}

func seedCourses(ctx context.Context, conn *pgx.Conn) error {
	for _, c := range defaultCourses {
		_, err := conn.Exec(ctx,
			`INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota)
 VALUES ($1, $2, $3, $4, $5)
 ON CONFLICT (kode_mk) DO UPDATE SET nama_mk = EXCLUDED.nama_mk`,
			c.KodeMK, c.NamaMK, c.SKS, c.Semester, c.Kuota,
		)
		if err != nil {
			return fmt.Errorf("insert course %s: %w", c.KodeMK, err)
		}
	}
	return nil
}
