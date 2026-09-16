package repository

import (
	"context"
	"errors"
	"fmt"
	"students_api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AchievementRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Achievement, int, error)
	FindByID(ctx context.Context, id int) (model.Achievement, error)
}

type achievementPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewAchievementRepository(pool *pgxpool.Pool) AchievementRepository {
	return &achievementPostgresRepository{pool: pool}
}

// func buildAchievementFilter(q model.ListQuery) (string, []any) {
// 	where := " WHERE 1 = 1"
// 	args := []any{}
// 	if q.Search != "" {
// 		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)",
// 			len(args)+1, len(args)+1)
// 		args = append(args, "%"+q.Search+"%")
// 	}
// 	if q.IsActive != nil {
// 		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
// 		args = append(args, *q.IsActive)
// 	}
// 	return where, args
// }

func (r *achievementPostgresRepository) FindAll(
	ctx context.Context, q model.ListQuery,
) ([]model.Achievement, int, error) {
	// where, args := buildFilter(q)
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM achievements").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung achievement: %w", err)
	}
	rows, err := r.pool.Query(ctx, "SELECT id, student_id, achievement_name, achievement_score, created_at FROM achievements" /* , args... */)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar achievement: %w", err)
	}
	defer rows.Close()
	hasil := []model.Achievement{}
	for rows.Next() {
		var a model.Achievement
		if err := rows.Scan(&a.ID, &a.StudentID, &a.Name, &a.Score, &a.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris achievement: %w", err)
		}
		hasil = append(hasil, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}
	return hasil, total, nil
}

func (r *achievementPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Achievement, error) {
	var a model.Achievement
	err := r.pool.QueryRow(ctx,
		`SELECT id, student_id, achievement_name, achievement_score, created_at
 FROM achievements WHERE id = $1`, id,
	).Scan(&a.ID, &a.StudentID, &a.Name, &a.Score, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Achievement{}, ErrNotFound
		}
		return model.Achievement{}, fmt.Errorf("mengambil achievement: %w", err)
	}
	return a, nil
}
