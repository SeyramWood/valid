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
	rgx, _ := regexp.Compile(`^\+(999|998|997|996|995|994|993|992|991|990|979|978|977|976|975|974|973|972|971|970|969|968|967|966|965|964|963|962|961|960|899|898|897|896|895|894|893|892|891|890|889|888|887|886|885|884|883|882|881|880|879|878|877|876|875|874|873|872|871|870|859|858|857|856|855|854|853|852|851|850|839|838|837|836|835|834|833|832|831|830|809|808|807|806|805|804|803|802|801|800|699|698|697|696|695|694|693|692|691|690|689|688|687|686|685|684|683|682|681|680|679|678|677|676|675|674|673|672|671|670|599|598|597|596|595|594|593|592|591|590|509|508|507|506|505|504|503|502|501|500|429|428|427|426|425|424|423|422|421|420|389|388|387|386|385|384|383|382|381|380|379|378|377|376|375|374|373|372|371|370|359|358|357|356|355|354|353|352|351|350|299|298|297|296|295|294|293|292|291|290|289|288|287|286|285|284|283|282|281|280|269|268|267|266|265|264|263|262|261|260|259|258|257|256|255|254|253|252|251|250|249|248|247|246|245|244|243|242|241|240|239|238|237|236|235|234|233|232|231|230|229|228|227|226|225|224|223|222|221|220|219|218|217|216|215|214|213|212|211|210|98|95|94|93|92|91|90|86|84|82|81|66|65|64|63|62|61|60|58|57|56|55|54|53|52|51|49|48|47|46|45|44|43|41|40|39|36|34|33|32|31|30|27|20|7|1)[0-9]{1,14}$`)
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
