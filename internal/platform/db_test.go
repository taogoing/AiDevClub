package platform

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestIsDuplicateEntry(t *testing.T) {
	if IsDuplicateEntry(gorm.ErrRecordNotFound) {
		t.Fatal("RecordNotFound should not be duplicate")
	}
	if !IsDuplicateEntry(&mysql.MySQLError{Number: 1062}) {
		t.Fatal("1062 should be duplicate")
	}
	if IsDuplicateEntry(&mysql.MySQLError{Number: 1452}) {
		t.Fatal("1452 should not be duplicate")
	}
}
