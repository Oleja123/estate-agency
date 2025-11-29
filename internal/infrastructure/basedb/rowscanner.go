package basedb

// RowScanner abstracts types that implement Scan(...interface{}) error such as
// *pgx.Row and *pgx.Rows. Exported so infra repositories can reuse it.
type RowScanner interface {
	Scan(dest ...interface{}) error
}
