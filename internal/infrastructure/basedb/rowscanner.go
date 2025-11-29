package basedb

type RowScanner interface {
	Scan(dest ...interface{}) error
}
