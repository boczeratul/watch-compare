package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

const alertColumns = `id, subscriber_id, name, query, checked_at, last_notified_at, created_at`

func scanAlerts(rows pgx.Rows) ([]model.Alert, error) {
	defer rows.Close()
	out := []model.Alert{} // never nil: a JSON null would break array consumers
	for rows.Next() {
		var a model.Alert
		if err := rows.Scan(&a.ID, &a.SubscriberID, &a.Name, &a.Query, &a.CheckedAt, &a.LastNotifiedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AlertsFor lists one subscriber's alerts, newest first.
func (r *Repo) AlertsFor(ctx context.Context, subscriberID string) ([]model.Alert, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+alertColumns+` FROM alerts WHERE subscriber_id = $1 ORDER BY created_at DESC, id DESC`, subscriberID)
	if err != nil {
		return nil, err
	}
	return scanAlerts(rows)
}

// AllAlerts lists every alert in id order (the notifier's work list).
func (r *Repo) AllAlerts(ctx context.Context) ([]model.Alert, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+alertColumns+` FROM alerts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return scanAlerts(rows)
}

// CreateAlert stores an alert unless the subscriber already has `limit` of them; ok is false when
// the limit was hit. An identical query from the same subscriber returns the existing alert.
// New alerts start checking from now, so only listings found after they were saved notify.
func (r *Repo) CreateAlert(ctx context.Context, subscriberID, name, query string, limit int) (a model.Alert, ok bool, err error) {
	rows, err := r.pool.Query(ctx, `SELECT `+alertColumns+` FROM alerts WHERE subscriber_id = $1 AND query = $2`, subscriberID, query)
	if err != nil {
		return a, false, err
	}
	existing, err := scanAlerts(rows)
	if err != nil {
		return a, false, err
	}
	if len(existing) > 0 {
		return existing[0], true, nil
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO alerts (subscriber_id, name, query)
		SELECT $1, $2, $3 WHERE (SELECT count(*) FROM alerts WHERE subscriber_id = $1) < $4
		RETURNING `+alertColumns, subscriberID, name, query, limit).
		Scan(&a.ID, &a.SubscriberID, &a.Name, &a.Query, &a.CheckedAt, &a.LastNotifiedAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, false, nil
	}
	return a, err == nil, err
}

// DeleteAlert removes one of the subscriber's alerts; it returns ErrNotFound when there is none.
func (r *Repo) DeleteAlert(ctx context.Context, subscriberID string, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM alerts WHERE id = $1 AND subscriber_id = $2`, id, subscriberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAlertChecked advances the alert's watermark; notified also stamps last_notified_at.
func (r *Repo) MarkAlertChecked(ctx context.Context, id int64, at time.Time, notified bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE alerts SET checked_at = $2, last_notified_at = CASE WHEN $3 THEN $2 ELSE last_notified_at END WHERE id = $1`, id, at, notified)
	return err
}

// NewMatches counts active listings matching q (whose FirstSeenAfter is the alert's watermark)
// and returns up to limit of them, cheapest first.
func (r *Repo) NewMatches(ctx context.Context, q model.ListingQuery, limit int) (int, []model.Listing, error) {
	b := listingFilter(q)
	whereSQL := b.whereSQL()
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+listingFrom+whereSQL, b.args...).Scan(&total); err != nil {
		return 0, nil, err
	}
	if total == 0 {
		return 0, []model.Listing{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+listingColumns+listingFrom+whereSQL+` ORDER BY l.price_usd ASC NULLS LAST, l.id LIMIT `+fmt.Sprint(limit), b.args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	items, err := collect(rows)
	return total, items, err
}
