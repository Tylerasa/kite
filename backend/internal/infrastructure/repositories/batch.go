package repositories

import "github.com/jackc/pgx/v5"

// pgxBatch is a small helper to build a pgx.Batch with queued queries.
type pgxBatch struct {
	b pgx.Batch
}

func (pb *pgxBatch) Queue(sql string, args ...interface{}) {
	pb.b.Queue(sql, args...)
}

func (pb *pgxBatch) Batch() *pgx.Batch {
	return &pb.b
}
