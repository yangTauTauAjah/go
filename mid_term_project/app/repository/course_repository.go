package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/app/model"
)

type CourseRepository interface {
	FindAll(ctx context.Context, semester int, search string, availableOnly bool) ([]model.Course, error)
	FindByID(ctx context.Context, id int) (model.Course, error)
}

type coursePostgresRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &coursePostgresRepository{pool: pool}
}

func (r *coursePostgresRepository) FindAll(
	ctx context.Context, semester int, search string, availableOnly bool,
) ([]model.Course, error) {
	where := " WHERE 1 = 1"
	args := []any{}
	if semester > 0 {
		args = append(args, semester)
		where += " AND c.semester = $" + strconv.Itoa(len(args))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", len(args), len(args))
	}
	if availableOnly {
		where += " AND c.kuota > (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)"
	}

	q := fmt.Sprintf(
		`SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
 COALESCE((SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id), 0) AS terisi
 FROM courses c %s
 ORDER BY c.semester ASC, c.kode_mk ASC`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar course: %w", err)
	}
	defer rows.Close()
	out := []model.Course{}
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi); err != nil {
			return nil, fmt.Errorf("membaca baris course: %w", err)
		}
		c.SisaKuota = c.Kuota - c.Terisi
		if c.SisaKuota < 0 {
			c.SisaKuota = 0
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query: %w", err)
	}
	return out, nil
}

func (r *coursePostgresRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.pool.QueryRow(ctx,
		`SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
 COALESCE((SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id), 0) AS terisi
 FROM courses c WHERE c.id = $1`, id,
	).Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("mengambil course: %w", err)
	}
	c.SisaKuota = c.Kuota - c.Terisi
	if c.SisaKuota < 0 {
		c.SisaKuota = 0
	}
	return c, nil
}
