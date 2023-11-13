package valid

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	// import github.com/go-sql-driver/mysql
	_ "github.com/go-sql-driver/mysql"
	// import github.com/lib/pq
	_ "github.com/lib/pq"
)

func getRuleAndMsg(r string) (rule, customMsg string) {
	if strings.Contains(r, ">") {
		rcm := strings.SplitN(r, ">", 2)
		rule = rcm[0]
		customMsg = rcm[1]
	} else {
		rule = r
	}
	return
}
func formatFieldName(field string) string {
	var text string
	for i := 0; i < len(field); i++ {
		c := string([]byte{field[i]})
		if c == strings.ToUpper(c) {
			if len(text) != 0 {
				text += " "
			}
			text += strings.ToLower(c)
		} else {
			text += strings.ToLower(c)
		}
	}
	return text
}
func readFile(fh *multipart.FileHeader) ([]byte, error) {
	file, err := fh.Open()
	if err != nil {
		return nil, err
	}
	buffer, _ := io.ReadAll(file)
	return buffer, nil
}
func prepareMimes(mimes string) []string {
	buffer := make([]string, 0, len(mimes))
	for _, m := range strings.Split(mimes, ",") {
		buffer = append(buffer, "."+m)
	}
	return buffer
}

func snakeCase(camel string) (snake string) {
	var b strings.Builder
	diff := 'a' - 'A'
	l := len(camel)
	for i, v := range camel {
		// A is 65, a is 97
		if v >= 'a' {
			b.WriteRune(v)
			continue
		}
		if (i != 0 || i == l-1) && ((i > 0 && rune(camel[i-1]) >= 'a') || (i < l-1 && rune(camel[i+1]) >= 'a')) {
			b.WriteRune('_')
		}
		b.WriteRune(v + diff)
	}
	return b.String()
}

// Database configuration
type Database struct {
	Host     string
	Port     int
	Name     string
	Username string
	Password string
	Driver   string
	SSLMode  string
}

func connectDB(config *Database) *sql.DB {
	switch config.Driver {
	case DriverPostgres:
		var SSLMode string
		if config.SSLMode != "" {
			SSLMode = config.SSLMode
		} else {
			SSLMode = "disable"
		}
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.Username,
			config.Name,
			config.Password,
			SSLMode,
		)
		db, err := sql.Open(config.Driver, dsn)
		if err != nil {
			panic(fmt.Sprintf("[db connection error] %v", err))
		}
		return db
	case DriverMysql:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s",
			config.Username,
			config.Password,
			config.Host,
			config.Port,
			config.Name,
		)
		db, err := sql.Open(config.Driver, dsn)
		if err != nil {
			panic(fmt.Sprintf("[db connection error] %v", err))
		}
		return db
	default:
		panic(fmt.Sprintf("[db connection error]: provide db driver. validata currently support %s and %s drivers.", DriverMysql, DriverPostgres))
	}
}
