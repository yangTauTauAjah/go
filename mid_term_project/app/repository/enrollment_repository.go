package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/app/model"
)

type EnrollmentRepository interface {
	// EnrollAtomic: cek duplikasi, ambil & lock row course, validasi kuota,
	// validasi batas SKS, lalu menyisipkan enrollment. Semua dalam satu
	// transaksi agar business rule tidak bisa dilanggar oleh race condition.
	EnrollAtomic(ctx context.Context, studentID int, courseID int, tahunAkademik string) (model.Enrollment, error)
	// FindByStudent melihat seluruh enrollment milik seorang mahasiswa
	// pada tahun akademik tertentu (kosong = semua tahun).
	FindByStudent(ctx context.Context, studentID int, tahunAkademik string) ([]model.Enrollment, error)
	// FindByID melihat satu enrollment (untuk delete).
	FindByID(ctx context.Context, id int) (model.Enrollment, error)
	// DeleteByStudentAndID menghapus enrollment dengan verifikasi pemilik.
	DeleteByStudentAndID(ctx context.Context, enrollmentID, studentID int) (model.Enrollment, error)
	// SumSKSByStudentAndTahun menghitung total SKS aktif seorang mahasiswa
	// pada tahun akademik tertentu.
	SumSKSByStudentAndTahun(ctx context.Context, studentID int, tahunAkademik string) (int, error)
}

type enrollmentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) EnrollmentRepository {
	return &enrollmentPostgresRepository{pool: pool}
}

func (r *enrollmentPostgresRepository) EnrollAtomic(
	ctx context.Context, studentID int, courseID int, tahunAkademik string,
) (model.Enrollment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Enrollment{}, fmt.Errorf("memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Cek duplikasi
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
 SELECT 1 FROM enrollments
 WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3
 )`, studentID, courseID, tahunAkademik,
	).Scan(&exists)
	if err != nil {
		return model.Enrollment{}, fmt.Errorf("memeriksa duplikasi enrollment: %w", err)
	}
	if exists {
		return model.Enrollment{}, ErrDuplicate
	}

	// 2. Lock course row untuk mencegah race condition pada kuota.
	var course model.Course
	err = tx.QueryRow(ctx,
		`SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
 COALESCE((SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id), 0) AS terisi
 FROM courses c WHERE c.id = $1 FOR UPDATE`, courseID,
	).Scan(&course.ID, &course.KodeMK, &course.NamaMK, &course.SKS, &course.Semester, &course.Kuota, &course.Terisi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrNotFound
		}
		return model.Enrollment{}, fmt.Errorf("mengunci course: %w", err)
	}

	// 3. Validasi kuota
	if course.Terisi >= course.Kuota {
		return model.Enrollment{}, ErrKuotaPenuh
	}

	// 4. Batas SKS dihitung di service (butuh IPK mahasiswa). Repository
	//    menyediakan cara mengambil total SKS existing pada tahun akademik ini.
	var inserted model.Enrollment
	err = tx.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, course_id, tahun_akademik)
 VALUES ($1, $2, $3)
 RETURNING id, student_id, course_id, tahun_akademik, created_at`,
		studentID, courseID, tahunAkademik,
	).Scan(&inserted.ID, &inserted.StudentID, &inserted.CourseID, &inserted.TahunAkademik, &inserted.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Enrollment{}, ErrDuplicate
		}
		return model.Enrollment{}, fmt.Errorf("menyimpan enrollment: %w", err)
	}
	inserted.KodeMK = course.KodeMK
	inserted.NamaMK = course.NamaMK
	inserted.SKS = course.SKS

	if err := tx.Commit(ctx); err != nil {
		return model.Enrollment{}, fmt.Errorf("menyimpan transaksi: %w", err)
	}
	return inserted, nil
}

func (r *enrollmentPostgresRepository) FindByStudent(
	ctx context.Context, studentID int, tahunAkademik string,
) ([]model.Enrollment, error) {
	q := `SELECT e.id, e.student_id, e.course_id, e.tahun_akademik, e.created_at,
 c.kode_mk, c.nama_mk, c.sks
 FROM enrollments e
 JOIN courses c ON c.id = e.course_id
 WHERE e.student_id = $1`
	args := []any{studentID}
	if tahunAkademik != "" {
		args = append(args, tahunAkademik)
		q += fmt.Sprintf(" AND e.tahun_akademik = $%d", len(args))
	}
	q += " ORDER BY e.created_at DESC"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar enrollment: %w", err)
	}
	defer rows.Close()
	out := []model.Enrollment{}
	for rows.Next() {
		var e model.Enrollment
		if err := rows.Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik,
			&e.CreatedAt, &e.KodeMK, &e.NamaMK, &e.SKS); err != nil {
			return nil, fmt.Errorf("membaca baris enrollment: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *enrollmentPostgresRepository) FindByID(ctx context.Context, id int) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.pool.QueryRow(ctx,
		`SELECT id, student_id, course_id, tahun_akademik, created_at
 FROM enrollments WHERE id = $1`, id,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrNotFound
		}
		return model.Enrollment{}, fmt.Errorf("mengambil enrollment: %w", err)
	}
	return e, nil
}

func (r *enrollmentPostgresRepository) DeleteByStudentAndID(
	ctx context.Context, enrollmentID, studentID int,
) (model.Enrollment, error) {
	var e model.Enrollment
	err := r.pool.QueryRow(ctx,
		`DELETE FROM enrollments WHERE id = $1 AND student_id = $2
 RETURNING id, student_id, course_id, tahun_akademik, created_at`,
		enrollmentID, studentID,
	).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrNotFound
		}
		return model.Enrollment{}, fmt.Errorf("menghapus enrollment: %w", err)
	}
	return e, nil
}

func (r *enrollmentPostgresRepository) SumSKSByStudentAndTahun(
	ctx context.Context, studentID int, tahunAkademik string,
) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.sks), 0)::int
 FROM enrollments e
 JOIN courses c ON c.id = e.course_id
 WHERE e.student_id = $1 AND e.tahun_akademik = $2`,
		studentID, tahunAkademik,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("menghitung total SKS: %w", err)
	}
	return total, nil
}
