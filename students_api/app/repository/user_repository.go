package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"students_api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	FindByUsername(ctx context.Context, username string) (model.Student, error)
	Create(ctx context.Context, u model.Student) (model.Student, error)
	Update(ctx context.Context, u model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
	UpdateRole(ctx context.Context, id int, role string) (model.Student, error)
}

var kolomUrut = map[string]string{
	"id":         "id",
	"username":   "username",
	"email":      "email",
	"created_at": "created_at",
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func buildFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}
	if q.Search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)",
			len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}
	return where, args
}

func (r *studentPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.Student, int, error) {
	where, args := buildFilter(q)
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung student: %w", err)
	}

	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	// Gunakan LEFT JOIN + GROUP BY + json_agg agar student tanpa achievement tetap muncul
	// dan pagination (LIMIT/OFFSET) tetap konsisten per-student.
	sqlText := fmt.Sprintf(
		`SELECT 
			u.id, 
			u.username, 
			u.role,
			u.email, 
			u.password, 
			u.is_active, 
			u.created_at,
			COALESCE(
				json_agg(
					json_build_object(
						'id', a.id,
						'name', a.achievement_name,
						'score', a.achievement_score,
						'created_at', a.created_at
					)
				) FILTER (WHERE a.id IS NOT NULL), 
				'[]'
			) AS achievements
		FROM users u
		LEFT JOIN achievements a ON u.id = a.student_id
		%s
		GROUP BY u.id
		ORDER BY u.%s %s
		LIMIT $%d OFFSET $%d`,
		where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
	)

	args = append(args, q.Limit, q.Offset())
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close()
	hasil := []model.Student{}
	for rows.Next() {
		var u model.Student
		var achievementsJSON []byte

		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Role,
			&u.Email,
			&u.Password,
			&u.IsActive,
			&u.CreatedAt,
			&achievementsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("membaca baris student: %w", err)
		}

		if err := json.Unmarshal(achievementsJSON, &u.Achievements); err != nil {
			return nil, 0, fmt.Errorf("unmarshal achievements: %w", err)
		}

		hasil = append(hasil, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}
	return hasil, total, nil
}

func (r *studentPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Student, error) {
	var u model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, role, email, password, is_active, created_at
 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Email, &u.Password, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}
	return u, nil
}
func (r *studentPostgresRepository) Create(
	ctx context.Context, u model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, role, email, password, is_active)
 VALUES ($1, $2, $3, $4, $5)
 RETURNING id, created_at`,
		u.Username, u.Role, u.Email, u.Password, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}
	return u, nil
}
func (r *studentPostgresRepository) Update(
	ctx context.Context, u model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET username = $1, role = $2, email = $3, is_active = $4
 WHERE id = $5
 RETURNING id, username, role, email, password, is_active, created_at`,
		u.Username, u.Role, u.Email, u.IsActive, u.ID,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Email, &u.Password, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("memperbarui user: %w", err)
	}
	return u, nil
}
func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// huruf besar dan kecil, sama seperti unique index-nya.
func (r *studentPostgresRepository) FindByUsername(
	ctx context.Context, username string,
) (model.Student, error) {
	var u model.Student

	err := r.pool.QueryRow(ctx,
		`SELECT id, username, role, email, password, is_active, created_at 
         FROM users WHERE LOWER(username) = LOWER($1)`, username,
	).Scan(&u.ID, &u.Username, &u.Role, &u.Email, &u.Password, &u.IsActive, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil user: %w", err)
	}

	return u, nil
}

func (r *studentPostgresRepository) UpdateRole(
	ctx context.Context, id int, role string,
) (model.Student, error) {
	studentColumns := "id, username, email, password, role, is_active, created_at"
	var updated model.Student
	err := r.pool.QueryRow(ctx,
		"UPDATE users SET role = $1 WHERE id = $2 RETURNING "+studentColumns,
		role, id).Scan(&updated.ID, &updated.Username, &updated.Email, &updated.Password, &updated.Role, &updated.IsActive, &updated.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengubah role user: %w", err)
	}
	return updated, nil
}
