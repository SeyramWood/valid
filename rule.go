package valid

import (
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}
func isNotInt(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^(?:[-]?(?:0|[1-9][0-9]*))$`)
	return !rgx.MatchString(fmt.Sprintf("%d", v.Interface()))
}
func isNotUint(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[1-9]\d+$`)
	return !rgx.MatchString(fmt.Sprintf("%d", v.Interface()))
}
func isNotFloat(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?$`)
	return !rgx.MatchString(fmt.Sprintf("%.2f", v.Interface()))
}
func isNotAlpha(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[a-zA-Z]+$`)
	return !rgx.MatchString(v.String())
}
func isNotAlphanumeric(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[a-zA-Z0-9]+$`)
	return !rgx.MatchString(v.String())
}
func isNotNumeric(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[0-9]+$`)
	return !rgx.MatchString(v.String())
}
func isNotString(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^[0-9a-zA-Z-+ .]+$`)
	return !rgx.MatchString(v.String())
}
func isNotSame(v1, v2 reflect.Value) bool {
	return !(strings.TrimSpace(v1.String()) == strings.TrimSpace(v2.String()))
}
func isNotASCII(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`[\x00-\x7F]+`)
	return !rgx.MatchString(v.String())
}
func isNotEmail(v reflect.Value) bool {
	if len(v.String()) < 6 || len(v.String()) > 254 {
		return true
	}
	at := strings.LastIndex(v.String(), "@")
	if at <= 0 || at > len(v.String())-3 {
		return true
	}
	switch v.String()[at+1:] {
	case "localhost", "localhost.com", "example.com":
		return true
	}
	if len(v.String()[:at]) > 64 {
		return true
	}
	if _, err := mail.ParseAddress(v.String()); err != nil {
		return true
	}
	return false
}
func isNotPhone(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^0\d{9}$`)
	return !rgx.MatchString(v.String())
}
func isNotPhoneWithCode(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^\+\d{12}$`)
	return !rgx.MatchString(v.String())
}
func isNotUsername(v reflect.Value) bool {
	if strings.Contains(v.String(), "@") {
		return isNotEmail(v)
	}
	if strings.HasPrefix(v.String(), "+") {
		return isNotPhoneWithCode(v)
	}
	return isNotPhone(v)
}
func isNotGHCard(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`^GHA-\d{9}-\d{1}$`)
	return !rgx.MatchString(v.String())
}
func isNotGHGPS(v reflect.Value) bool {
	rgx, _ := regexp.Compile(`[A-Z]{2}-\d{1,4}-\d{4}$`)
	return !rgx.MatchString(v.String())
}
func isNotMin(v reflect.Value, comparable string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		val, _ := strconv.Atoi(comparable)
		return !(v.Len() >= val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, _ := strconv.ParseInt(comparable, 10, 64)
		return !(v.Int() >= val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val, _ := strconv.ParseUint(comparable, 10, 64)
		return !(v.Uint() >= val)
	case reflect.Float32, reflect.Float64:
		val, _ := strconv.ParseFloat(comparable, 64)
		return !(v.Float() >= val)
	}
	return false
}
func isNotMax(v reflect.Value, comparable string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		val, _ := strconv.Atoi(comparable)
		return !(v.Len() <= val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, _ := strconv.ParseInt(comparable, 10, 64)
		return !(v.Int() <= val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val, _ := strconv.ParseUint(comparable, 10, 64)
		return !(v.Uint() <= val)
	case reflect.Float32, reflect.Float64:
		val, _ := strconv.ParseFloat(comparable, 64)
		return !(v.Float() <= val)
	}
	return false
}
func isNotEqual(v reflect.Value, comparable string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		val, _ := strconv.Atoi(comparable)
		return v.Len() != val
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, _ := strconv.ParseInt(comparable, 10, 64)
		return v.Int() != val
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val, _ := strconv.ParseUint(comparable, 10, 64)
		return v.Uint() != val
	case reflect.Float32, reflect.Float64:
		val, _ := strconv.ParseFloat(comparable, 64)
		return v.Float() != val
	}
	return false
}
func isNotBetween(v reflect.Value, min, max string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		minVal, _ := strconv.Atoi(min)
		maxVal, _ := strconv.Atoi(max)
		return !(v.Len() > minVal && v.Len() < maxVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, _ := strconv.ParseInt(min, 10, 64)
		maxVal, _ := strconv.ParseInt(max, 10, 64)
		return !(v.Int() > minVal && v.Int() < maxVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		minVal, _ := strconv.ParseUint(min, 10, 64)
		maxVal, _ := strconv.ParseUint(max, 10, 64)
		return !(v.Uint() > minVal && v.Uint() < maxVal)
	case reflect.Float32, reflect.Float64:
		minVal, _ := strconv.ParseFloat(min, 64)
		maxVal, _ := strconv.ParseFloat(max, 64)
		return !(v.Float() > minVal && v.Float() < maxVal)
	}
	return false
}
func isNotFrom(v reflect.Value, min, max string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		minVal, _ := strconv.Atoi(min)
		maxVal, _ := strconv.Atoi(max)
		return !(v.Len() >= minVal && v.Len() <= maxVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, _ := strconv.ParseInt(min, 10, 64)
		maxVal, _ := strconv.ParseInt(max, 10, 64)
		return !(v.Int() >= minVal && v.Int() <= maxVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		minVal, _ := strconv.ParseUint(min, 10, 64)
		maxVal, _ := strconv.ParseUint(max, 10, 64)
		return !(v.Uint() >= minVal && v.Uint() <= maxVal)
	case reflect.Float32, reflect.Float64:
		minVal, _ := strconv.ParseFloat(min, 64)
		maxVal, _ := strconv.ParseFloat(max, 64)
		return !(v.Float() >= minVal && v.Float() <= maxVal)
	}
	return false
}
func isNotSize(v reflect.Value, comparable string) bool {
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		val, _ := strconv.Atoi(comparable)
		return !(v.Len() == val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, _ := strconv.ParseInt(comparable, 10, 64)
		return !(v.Int() == val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val, _ := strconv.ParseUint(comparable, 10, 64)
		return !(v.Uint() == val)
	case reflect.Float32, reflect.Float64:
		val, _ := strconv.ParseFloat(comparable, 64)
		return !(v.Float() == val)
	}
	return false
}
func isNotFile(v reflect.Value) bool {
	if fh, ok := v.Interface().(*multipart.FileHeader); ok {
		if _, err := readFile(fh); err != nil {
			return true
		}
		return false
	}
	for _, fh := range v.Interface().([]*multipart.FileHeader) {
		if _, err := readFile(fh); err != nil {
			return true
		}
	}
	return false
}
func isNotMimes(v reflect.Value, mimes string) bool {
	if fh, ok := v.Interface().(*multipart.FileHeader); ok {
		buffer, err := readFile(fh)
		if err != nil {
			return true
		}
		if !mimetype.EqualsAny(mimetype.Detect(buffer).Extension(), prepareMimes(mimes)...) {
			return true
		}
		return false
	}
	for _, fh := range v.Interface().([]*multipart.FileHeader) {
		buffer, err := readFile(fh)
		if err != nil {
			return true
		}
		if !mimetype.EqualsAny(mimetype.Detect(buffer).Extension(), prepareMimes(mimes)...) {
			return true
		}
	}
	return false
}
func isNotUnique(dbConfig *Database, value, field, table string) bool {
	db := connectDB(dbConfig)
	defer db.Close()

	dbField := snakeCase(field)
	queryStr := fmt.Sprintf("SELECT %s FROM %s WHERE %s=?", dbField, table, dbField)
	if err := db.QueryRow(queryStr, value).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		panic(err)
	}
	return true
}
func isNotDatetime(v reflect.Value, kind string) bool {
	switch kind {
	case "rfc3339":
		if _, err := time.Parse(time.RFC3339, v.String()); err != nil {
			return true
		}
		return false
	case "datetime":
		if _, err := time.Parse(time.DateTime, v.String()); err != nil {
			return true
		}
		return false
	case "dateonly":
		if _, err := time.Parse(time.DateOnly, v.String()); err != nil {
			return true
		}
		return false
	case "timeonly":
		if _, err := time.Parse(time.TimeOnly, v.String()); err != nil {
			return true
		}
		return false
	default:
		panic(fmt.Sprintf("date format not supported: %s", kind))
	}
}
