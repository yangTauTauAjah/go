package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad/app/model"
)

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
	// ErrKuotaPenuh dikembalikan ketika slot course sudah penuh.
	ErrKuotaPenuh = errors.New("kuota course sudah penuh")
	// ErrSksMelebihiBatas dikembalikan ketika total SKS akan melebihi
	// batas IPK mahasiswa.
	ErrSksMelebihiBatas = errors.New("total SKS melebihi batas IPK")
)

type StudentRepository interface {
	// FindAll admin list dengan pagination/filter/search/sort
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	// FindByID melihat data student
	FindByID(ctx context.Context, id int) (model.Student, error)
	// FindByIDWithDeleted sama seperti FindByID tetapi tidak menutup
	// baris yang sudah soft delete. Dipakai untuk membedakan 404
	// dengan 410 (sudah dihapus).
	FindByIDWithDeleted(ctx context.Context, id int) (model.Student, error)
	// FindByUserID mengambil data student berdasarkan user_id.
	// Dipakai oleh /auth/me dan endpoint enrollment.
	FindByUserID(ctx context.Context, userID int) (model.Student, error)
	// FindByEmail untuk login.
	FindByEmail(ctx context.Context, email string) (model.User, error)
	// CreateWithUser membuat user + student dalam satu transaction.
	CreateWithUser(ctx context.Context, user model.User, student model.Student) (model.Student, error)
	// UpdateAdmin mengubah data student oleh admin.
	UpdateAdmin(ctx context.Context, s model.Student) (model.Student, error)
	// SoftDelete mengisi deleted_at.
	SoftDelete(ctx context.Context, id int) error
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// allowedSortColumns memetakan alias sort (whitelisted) ke nama
// kolom SQL yang aman. Hindari SQL injection dengan memetakan
// ke nilai konstan, bukan menggabungkan string dari input user.
var allowedSortColumns = map[string]string{
	"id":           "s.id",
	"nim":          "s.nim",
	"nama":         "s.nama",
	"ipk_terakhir": "s.ipk_terakhir",
	"created_at":   "s.created_at",
}

func (r *studentPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.Student, int, error) {
	// Bangun WHERE clause dan args secara paralel: setiap placeholder
	// $N harus memiliki tepat satu argumen di posisi yang sama.
	parts := []string{"s.deleted_at IS NULL"}
	args := []any{}
	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		idx := len(args)
		parts = append(parts, fmt.Sprintf("(s.nim ILIKE $%d OR s.nama ILIKE $%d)", idx, idx))
	}
	if q.Prodi != "" {
		args = append(args, q.Prodi)
		parts = append(parts, fmt.Sprintf("s.prodi = $%d", len(args)))
	}
	if q.Angkatan != nil {
		args = append(args, *q.Angkatan)
		parts = append(parts, fmt.Sprintf("s.angkatan = $%d", len(args)))
	}
	where := " WHERE " + strings.Join(parts, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students s"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("menghitung student: %w", err)
	}

	kolom, ok := allowedSortColumns[q.Sort]
	if !ok {
		kolom = "s.id"
	}
	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	args = append(args, q.Limit, (q.Page-1)*q.Limit)
	limitIdx := len(args) - 1
	offsetIdx := len(args)
	sqlText := fmt.Sprintf(
		`SELECT s.id, s.user_id, s.nim, s.nama, s.prodi, s.angkatan, s.ipk_terakhir, s.created_at
 FROM students s
 %s
 ORDER BY %s %s, s.id ASC
 LIMIT $%d OFFSET $%d`, where, kolom, arah, limitIdx, offsetIdx)

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close()
	out := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPK, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris student: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}
	return out, total, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	return r.findByIDInternal(ctx, id, true)
}
func (r *studentPostgresRepository) FindByIDWithDeleted(ctx context.Context, id int) (model.Student, error) {
	return r.findByIDInternal(ctx, id, false)
}

func (r *studentPostgresRepository) findByIDInternal(ctx context.Context, id int, skipDeleted bool) (model.Student, error) {
	q := `SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at
 FROM students WHERE id = $1`
	if skipDeleted {
		q += " AND deleted_at IS NULL"
	}
	var s model.Student
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPK, &s.DeletedAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) FindByUserID(ctx context.Context, userID int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at
 FROM students WHERE user_id = $1 AND deleted_at IS NULL`, userID,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPK, &s.DeletedAt, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student by user: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, role, is_active, created_at
 FROM users WHERE LOWER(email) = LOWER($1)`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

func (r *studentPostgresRepository) CreateWithUser(
	ctx context.Context, user model.User, student model.Student,
) (model.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Student{}, fmt.Errorf("memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password, role, is_active)
 VALUES ($1, $2, 'mahasiswa', TRUE)
 RETURNING id, created_at`,
		user.Email, user.Password,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan user: %w", err)
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir)
 VALUES ($1, $2, $3, $4, $5, $6)
 RETURNING id, created_at`,
		user.ID, student.NIM, student.Nama, student.Prodi, student.Angkatan, student.IPK,
	).Scan(&student.ID, &student.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}

	student.UserID = user.ID
	student.Email = user.Email
	student.Role = user.Role
	if err := tx.Commit(ctx); err != nil {
		return model.Student{}, fmt.Errorf("menyimpan transaksi: %w", err)
	}
	return student, nil
}

func (r *studentPostgresRepository) UpdateAdmin(ctx context.Context, s model.Student) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE students
 SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4
 WHERE id = $5 AND deleted_at IS NULL
 RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at`,
		s.Nama, s.Prodi, s.Angkatan, s.IPK, s.ID,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IPK, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("memperbarui student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) SoftDelete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE students SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
