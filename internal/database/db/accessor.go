package db

// RawDB exposes the underlying sqlc DBTX handle for modules that still use
// direct SQL while they are being integrated into the generated query layer.
func (q *Queries) RawDB() DBTX {
	return q.db
}
