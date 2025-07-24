// Package valid is a robust and extensible validation library.
//
// This package handle data and input validation in Go applications.
package valid

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/SeyramWood/valid/locale"
)

const (
	kilobyte = 1024
	megabyte = kilobyte * kilobyte
	gigabyte = megabyte * kilobyte

	// LocaleFR constant variable for fr locale
	LocaleFR = "fr"
	// DriverPostgres postgres driver for database connection
	DriverPostgres = "postgres"
	// DriverMysql mysql driver for database connection
	DriverMysql = "mysql"
)

type (
	message struct {
		K string
		V any
	}
	// Config is configuration struct for Validate.
	Config struct {
		Locale string
		DB     *Database
	}
)

type Validator interface {
	// ValidateStruct performs validation on struct.
	// It takes struct pointer as parameter.
	ValidateStruct(elem any) map[string]any
	// RequestStruct takes struct pointer as parameter.
	RequestStruct(elem any) Validator
	// ValidateRequest performs validation on in coming request.
	// It is a middleware that takes http.Handler as parameter and return  http.Handler.
	ValidateRequest(next http.Handler) http.Handler
	// ValidateMap performs validation on map.
	// It takes map pointer as parameter.
	ValidateMap(elem map[string]any, rule map[string]string, message ...map[string]string) map[string]any
}

type validation struct {
	jsonRes struct {
		Status bool `json:"status"`
		Errors any  `json:"errors"`
	}
	elem      any
	elemType  reflect.Type
	elemValue reflect.Value
	locale    string
	dbConfig  *Database
}

// New takes optional @Config object.
// Valid use this configuration to connect to your database to check for exist field in validation.
func New(config ...*Config) Validator {
	instance := new(validation)
	if config != nil {
		instance.locale = config[0].Locale
		instance.dbConfig = config[0].DB
	}
	return instance
}

// ValidateStruct performs validation on struct.
// It takes struct pointer as parameter.
func (v *validation) ValidateStruct(elem any) map[string]any {
	elemType := reflect.TypeOf(elem)
	elemValue := reflect.ValueOf(elem)
	if elemType.Kind() != reflect.Pointer || elemValue.Kind() != reflect.Pointer {
		panic("validate: a pointer is expected as an argument")
	}
	v.elem = elem
	v.elemType = elemType.Elem()
	v.elemValue = elemValue.Elem()
	switch v.elemType.Kind() {
	case reflect.Struct:
		return v.structValidator()
	}
	panic("validate: a struct pointer is expected as an argument")
}

// ValidateMap performs validation on map.
// It takes map pointer as parameter.
func (v *validation) ValidateMap(elem map[string]any, rule map[string]string, message ...map[string]string) map[string]any {
	panic("implement me")
}

// RequestStruct takes struct pointer as parameter.
func (v *validation) RequestStruct(elem any) Validator {
	elemType := reflect.TypeOf(elem)
	elemValue := reflect.ValueOf(elem)
	if elemType.Kind() != reflect.Pointer || elemValue.Kind() != reflect.Pointer {
		panic("validate: a pointer is expected as an argument")
	}
	v.elem = elem
	v.elemType = elemType.Elem()
	v.elemValue = elemValue.Elem()
	return v
}

// ValidateRequest performs validation on in coming request.
// It is a middleware that takes http.Handler as parameter and return  http.Handler.
func (v *validation) ValidateRequest(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") || strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			decodeMultipart(r, v.elem)
		} else if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, v.elem); err != nil {
				resByte, _ := json.Marshal(map[string]any{
					"status":  false,
					"message": fmt.Sprintf("failed to unmarshal: %s", err.Error()),
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write(resByte)
				return
			}
		} else if strings.HasPrefix(r.Header.Get("Content-Type"), "text/xml") || strings.HasPrefix(r.Header.Get("Content-Type"), "application/xml") {
			body, _ := io.ReadAll(r.Body)
			if err := xml.Unmarshal(body, v.elem); err != nil {
				resByte, _ := json.Marshal(map[string]any{
					"status":  false,
					"message": fmt.Sprintf("failed to unmarshal: %s", err.Error()),
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write(resByte)
				return
			}
		} else {
			v.jsonRes.Status = false
			resByte, _ := json.Marshal(map[string]any{
				"status":  false,
				"message": fmt.Sprintf("content-type: %s, not supported.", r.Header.Get("Content-Type")),
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(resByte)
			return
		}

		switch v.elemType.Kind() {
		case reflect.Struct:
			message := v.structValidator()
			if len(message) > 0 {
				v.jsonRes.Errors = message
			} else {
				v.jsonRes.Errors = nil
			}
		}

		if v.jsonRes.Errors != nil {
			v.jsonRes.Status = false
			resByte, _ := json.Marshal(v.jsonRes)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(resByte)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (v *validation) structValidator() map[string]any {
	mChan := make(chan message, v.elemType.NumField())
	wg := &sync.WaitGroup{}
	for i := 0; i < v.elemType.NumField(); i++ {
		if _, ok := v.elemType.Field(i).Tag.Lookup("json"); ok {
			if _, ok := v.elemType.Field(i).Tag.Lookup("validate"); ok {
				wg.Add(1)
				go v.validateStruct(i, mChan, wg)
				continue
			}
		}
		panic("json or validate tag missing")
	}
	wg.Wait()
	errMsg := make(map[string]any)
	noErrMsg := make(map[string]any)
	for i := 0; i < v.elemType.NumField(); i++ {
		if msg, ok := <-mChan; ok {
			if msg.V == nil || msg.V == "" {
				noErrMsg[msg.K] = msg.V
			}
			errMsg[msg.K] = msg.V
		}
	}
	close(mChan)
	if len(errMsg) != len(noErrMsg) {
		return errMsg
	}
	return nil
}

// func (v *validation) mapValidator() map[string]any {
// 	panic("implement me")
// }

func (v *validation) validateStruct(index int, msgChan chan message, wg *sync.WaitGroup) {
	defer wg.Done()
	ruleOrMsgs := strings.Split(v.elemType.Field(index).Tag.Get("validate"), "|")
	value := v.elemValue.Field(index)
	formattedField := formatFieldName(v.elemType.Field(index).Tag.Get("json"))
	jsonTag := v.elemType.Field(index).Tag.Get("json")

	for _, ruleOrMsg := range ruleOrMsgs {
		rule, customMsg := getRuleAndMsg(ruleOrMsg)
		if rule == "required" && isEmpty(value) {
			if value.Kind() == reflect.Bool {
				v.setMessage("bool", customMsg, jsonTag, formattedField, msgChan)
				return
			}
			v.setMessage("required", customMsg, jsonTag, formattedField, msgChan)
			return
		}
		if !isEmpty(value) {
			switch value.Kind() {
			case reflect.String:
				switch rule {
				case "string":
					if isNotString(value) {
						v.setMessage("string", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "ascii":
					if isNotASCII(value) {
						v.setMessage("string", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "alpha":
					if isNotAlpha(value) {
						v.setMessage("alpha", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "numeric":
					if isNotNumeric(value) {
						v.setMessage("numeric", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "alpha_numeric":
					if isNotAlphanumeric(value) {
						v.setMessage("alpha_numeric", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "email":
					if isNotEmail(value) {
						v.setMessage("email", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "rfc3339":
					if isNotDatetime(value, "rfc3339") {
						v.setMessage("date.rfc3339", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "datetime":
					if isNotDatetime(value, "datetime") {
						v.setMessage("date.datetime", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "dateonly":
					if isNotDatetime(value, "dateonly") {
						v.setMessage("date.dateonly", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "phone":
					if isNotPhone(value) {
						v.setMessage("phone", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "phone_with_code":
					if isNotPhoneWithCode(value) {
						v.setMessage("phone_with_code", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "username":
					if isNotUsername(value) {
						v.setMessage("username", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "gh_card":
					if isNotGHCard(value) {
						v.setMessage("gh_card", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "gh_gps":
					if isNotGHGPS(value) {
						v.setMessage("gh_gps", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				default:
					if strings.Contains(rule, ":") {
						rSlice := strings.SplitN(rule, ":", 2)
						switch rSlice[0] {
						case "min":
							if isNotMin(value, rSlice[1]) {
								v.setMessage("min.string", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "max":
							if isNotMax(value, rSlice[1]) {
								v.setMessage("max.string", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "equal":
							if isNotEqual(value, rSlice[1]) {
								v.setMessage("equal.string", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "size":
							if isNotSize(value, rSlice[1]) {
								v.setMessage("size.string", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "from":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotFrom(value, minMax[0], minMax[1]) {
								v.setMessage("from.string", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "between":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotBetween(value, minMax[0], minMax[1]) {
								v.setMessage("between.string", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "enum":
							eunms := strings.Split(rSlice[1], ",")
							if isNotEnum(value, eunms) {
								v.setMessage("enum", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "same":
							tag, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("same", customMsg, jsonTag, formattedField, msgChan, tag)
								return
							}
						case "match":
							_, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("match", customMsg, jsonTag, formattedField, msgChan)
								return
							}
						case "unique":
							if tc := strings.SplitN(rSlice[1], ".", 2); len(tc) == 2 {
								if isNotUnique(v.dbConfig, value.String(), tc[1], tc[0]) {
									v.setMessage("unique", customMsg, jsonTag, formattedField, msgChan)
									return
								}
							}
						}
					}
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				switch rule {
				case "int":
					if isNotInt(value) {
						v.setMessage("int", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				case "uint":
					if isNotUint(value) {
						v.setMessage("uint", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				default:
					if strings.Contains(rule, ":") {
						rSlice := strings.SplitN(rule, ":", 2)
						switch rSlice[0] {
						case "min":
							if isNotMin(value, rSlice[1]) {
								v.setMessage("min.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "max":
							if isNotMax(value, rSlice[1]) {
								v.setMessage("max.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "equal":
							if isNotEqual(value, rSlice[1]) {
								v.setMessage("equal.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "size":
							if isNotSize(value, rSlice[1]) {
								v.setMessage("size.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "from":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotFrom(value, minMax[0], minMax[1]) {
								v.setMessage("from.numeric", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "between":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotBetween(value, minMax[0], minMax[1]) {
								v.setMessage("between.numeric", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "enum":
							eunms := strings.Split(rSlice[1], ",")
							if isNotEnum(value, eunms) {
								v.setMessage("enum", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "same":
							tag, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("same", customMsg, jsonTag, formattedField, msgChan, tag)
								return
							}
						case "match":
							_, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("match", customMsg, jsonTag, formattedField, msgChan)
								return
							}
						}

					}
				}
			case reflect.Float32, reflect.Float64:
				switch rule {
				case "float":
					if isNotFloat(value) {
						v.setMessage("float", customMsg, jsonTag, formattedField, msgChan)
						return
					}
				default:
					if strings.Contains(rule, ":") {
						rSlice := strings.SplitN(rule, ":", 2)
						switch rSlice[0] {
						case "min":
							if isNotMin(value, rSlice[1]) {
								v.setMessage("min.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "max":
							if isNotMax(value, rSlice[1]) {
								v.setMessage("max.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "equal":
							if isNotEqual(value, rSlice[1]) {
								v.setMessage("equal.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "size":
							if isNotSize(value, rSlice[1]) {
								v.setMessage("size.numeric", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "from":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotFrom(value, minMax[0], minMax[1]) {
								v.setMessage("from.numeric", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "between":
							minMax := strings.SplitN(rSlice[1], ",", 2)
							if isNotBetween(value, minMax[0], minMax[1]) {
								v.setMessage("between.numeric", customMsg, jsonTag, formattedField, msgChan, minMax[0], minMax[1])
								return
							}
						case "enum":
							eunms := strings.Split(rSlice[1], ",")
							if isNotEnum(value, eunms) {
								v.setMessage("enum", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
								return
							}
						case "same":
							tag, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("same", customMsg, jsonTag, formattedField, msgChan, tag)
								return
							}
						case "match":
							_, val := v.getTagAndValue(rSlice[1])
							if isNotSame(value, val) {
								v.setMessage("match", customMsg, jsonTag, formattedField, msgChan)
								return
							}
						}

					}
				}
			case reflect.Slice, reflect.Array:
				if strings.HasPrefix(rule, "slice") && strings.Contains(rule, ":") {
					rSlice := strings.SplitN(rule, ":", 3)[1:]
					switch rSlice[0] {
					case "min":
						if isNotMin(value, rSlice[1]) {
							v.setMessage("min.slice", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
							return
						}
					case "max":
						if isNotMax(value, rSlice[1]) {
							v.setMessage("max.slice", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
							return
						}
					}
				}
				switch value.Type().Elem().Kind() {
				case reflect.String:
					errMsgs := make([]any, 0, value.Len())
					for i := 1; i <= value.Len(); i++ {
						value := value.Index(i - 1)
						switch rule {
						case "string":
							if isNotString(value) {
								errMsgs = append(errMsgs, v.generateMessage("string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "ascii":
							if isNotASCII(value) {
								errMsgs = append(errMsgs, v.generateMessage("string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "alpha":
							if isNotAlpha(value) {
								errMsgs = append(errMsgs, v.generateMessage("alpha", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "numeric":
							if isNotNumeric(value) {
								errMsgs = append(errMsgs, v.generateMessage("numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "alpha_numeric":
							if isNotAlphanumeric(value) {
								errMsgs = append(errMsgs, v.generateMessage("alpha_numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "email":
							if isNotEmail(value) {
								errMsgs = append(errMsgs, v.generateMessage("email", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "phone":
							if isNotPhone(value) {
								errMsgs = append(errMsgs, v.generateMessage("phone", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "rfc3339":
							if isNotDatetime(value, "rfc3339") {
								errMsgs = append(errMsgs, v.generateMessage("date.rfc3339", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "datetime":
							if isNotDatetime(value, "datetime") {
								errMsgs = append(errMsgs, v.generateMessage("date.datetime", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "dateonly":
							if isNotDatetime(value, "dateonly") {
								errMsgs = append(errMsgs, v.generateMessage("date.dateonly", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "phone_with_code":
							if isNotPhoneWithCode(value) {
								errMsgs = append(errMsgs, v.generateMessage("phone_with_code", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "username":
							if isNotUsername(value) {
								errMsgs = append(errMsgs, v.generateMessage("username", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "gh_card":
							if isNotGHCard(value) {
								errMsgs = append(errMsgs, v.generateMessage("gh_card", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "gh_gps":
							if isNotGHGPS(value) {
								errMsgs = append(errMsgs, v.generateMessage("gh_gps", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						default:
							if strings.Contains(rule, ":") {
								rSlice := strings.SplitN(rule, ":", 2)
								switch rSlice[0] {
								case "min":
									if isNotMin(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("min.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "max":
									if isNotMax(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("max.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "equal":
									if isNotEqual(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("equal.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "size":
									if isNotSize(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("size.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "from":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotFrom(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("from.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "between":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotBetween(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("between.string", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "same":
									tag, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("same", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), tag))
										continue
									}
								case "match":
									_, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("match", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
										continue
									}
								case "unique":
									if tc := strings.SplitN(rSlice[1], ".", 2); len(tc) == 2 {
										if isNotUnique(v.dbConfig, value.String(), tc[1], tc[0]) {
											errMsgs = append(errMsgs, v.generateMessage("unique", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
											continue
										}
									}
								}
							}
						}
					}
					if len(errMsgs) > 0 {
						v.setMessage("", errMsgs, jsonTag, formattedField, msgChan)
						return
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					errMsgs := make([]any, 0, value.Len())
					for i := 1; i <= value.Len(); i++ {
						value := value.Index(i - 1)
						switch rule {
						case "int":
							if isNotInt(value) {
								errMsgs = append(errMsgs, v.generateMessage("int", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						case "uint":
							if isNotUint(value) {
								errMsgs = append(errMsgs, v.generateMessage("uint", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						default:
							if strings.Contains(rule, ":") {
								rSlice := strings.SplitN(rule, ":", 2)
								switch rSlice[0] {
								case "min":
									if isNotMin(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("min.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "max":
									if isNotMax(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("max.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "equal":
									if isNotEqual(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("equal.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "from":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotFrom(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("from.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "between":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotBetween(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("between.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "same":
									tag, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("same", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), tag))
										continue
									}
								case "match":
									_, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("match", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
										continue
									}
								}
							}
						}
					}
					if len(errMsgs) > 0 {
						v.setMessage("", errMsgs, jsonTag, formattedField, msgChan)
						return
					}
				case reflect.Float32, reflect.Float64:
					errMsgs := make([]any, 0, value.Len())
					for i := 1; i <= value.Len(); i++ {
						value := value.Index(i - 1)
						switch rule {
						case "float":
							if isNotFloat(value) {
								errMsgs = append(errMsgs, v.generateMessage("float", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
								continue
							}
						default:
							if strings.Contains(rule, ":") {
								rSlice := strings.SplitN(rule, ":", 2)
								switch rSlice[0] {
								case "min":
									if isNotMin(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("min.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "max":
									if isNotMax(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("max.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "equal":
									if isNotEqual(value, rSlice[1]) {
										errMsgs = append(errMsgs, v.generateMessage("equal.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
										continue
									}
								case "from":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotFrom(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("from.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "between":
									minMax := strings.SplitN(rSlice[1], ",", 2)
									if isNotBetween(value, minMax[0], minMax[1]) {
										errMsgs = append(errMsgs, v.generateMessage("between.numeric", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), minMax[0], minMax[1]))
										continue
									}
								case "same":
									tag, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("same", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), tag))
										continue
									}
								case "match":
									_, val := v.getTagAndValue(rSlice[1])
									if isNotSame(value, val) {
										errMsgs = append(errMsgs, v.generateMessage("match", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
										continue
									}
								}
							}
						}
					}
					if len(errMsgs) > 0 {
						v.setMessage("", errMsgs, jsonTag, formattedField, msgChan)
						return
					}
				case reflect.Pointer, reflect.Interface:
					if _, ok := value.Interface().([]*multipart.FileHeader); ok {
						errMsgs := make([]any, 0, value.Len())
						for i := 1; i <= value.Len(); i++ {
							value := value.Index(i - 1)
							switch rule {
							case "image":
								if isNotMimes(value, "jpg,jpeg,png,webp") {
									errMsgs = append(errMsgs, v.generateMessage("image", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
									continue
								}
							case "file":
								if isNotFile(value) {
									errMsgs = append(errMsgs, v.generateMessage("file", customMsg, fmt.Sprintf("%s (%d)", formattedField, i)))
									continue
								}
							default:
								if strings.Contains(rule, ":") {
									rSlice := strings.SplitN(rule, ":", 2)
									switch rSlice[0] {
									case "image":
										if isNotMimes(value, rSlice[1]) {
											errMsgs = append(errMsgs, v.generateMessage("image_type", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
											continue
										}
									case "file":
										if isNotMimes(value, rSlice[1]) {
											errMsgs = append(errMsgs, v.generateMessage("file_type", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
											continue
										}
									case "mimes":
										if isNotMimes(value, rSlice[1]) {
											errMsgs = append(errMsgs, v.generateMessage("mimes", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), rSlice[1]))
											continue
										}
									case "size":
										rgx := regexp.MustCompile(`^([1-9]|[1-9][0-9]+)(kb|KB|mb|MB|gb|GB|tb|TB)$`)
										matches := rgx.FindAllStringSubmatch(rSlice[1], -1)
										size, symbol := matches[0][1], matches[0][2]
										size64, _ := strconv.ParseInt(size, 10, 64)
										if fh, ok := value.Interface().(*multipart.FileHeader); ok {
											switch strings.ToLower(symbol) {
											case "kb":
												if fh.Size > int64(kilobyte*size64) {
													errMsgs = append(errMsgs, v.generateMessage("size.file_kb", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), size))
													continue
												}
											case "mb":
												if fh.Size > int64(megabyte*size64) {
													errMsgs = append(errMsgs, v.generateMessage("size.file_mb", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), size))
													continue
												}
											case "gb":
												if fh.Size > int64(gigabyte*size64) {
													errMsgs = append(errMsgs, v.generateMessage("size.file_gb", customMsg, fmt.Sprintf("%s (%d)", formattedField, i), size))
													continue
												}
											}
										}

									}
								}
							}
						}
						if len(errMsgs) > 0 {
							v.setMessage("", errMsgs, jsonTag, formattedField, msgChan)
							return
						}
					} else {
						err := make(map[string]any)
						for i := 0; i < value.Len(); i++ {
							if msg := New(&Config{Locale: v.locale, DB: v.dbConfig}).ValidateStruct(value.Index(i).Interface()); msg != nil {
								err[fmt.Sprintf("%s.%d", jsonTag, i)] = msg
							}
						}
						if len(err) > 0 {
							v.setMessage("", err, jsonTag, formattedField, msgChan)
							return
						}
					}
				}
			case reflect.Pointer, reflect.Interface:
				if _, ok := value.Interface().(*multipart.FileHeader); ok {
					switch rule {
					case "image":
						if isNotMimes(value, "jpg,jpeg,png,webp") {
							v.setMessage("image", customMsg, jsonTag, formattedField, msgChan)
							return
						}
					case "file":
						if isNotFile(value) {
							v.setMessage("file", customMsg, jsonTag, formattedField, msgChan)
							return
						}
					default:
						if strings.Contains(rule, ":") {
							rSlice := strings.SplitN(rule, ":", 2)
							switch rSlice[0] {
							case "image":
								if isNotMimes(value, rSlice[1]) {
									v.setMessage("image_type", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
									return
								}
							case "file":
								if isNotMimes(value, rSlice[1]) {
									v.setMessage("file_type", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
									return
								}
							case "mimes":
								if isNotMimes(value, rSlice[1]) {
									v.setMessage("mimes", customMsg, jsonTag, formattedField, msgChan, rSlice[1])
									return
								}
							case "size":
								rgx := regexp.MustCompile(`^([1-9]|[1-9][0-9]+)(kb|KB|mb|MB|gb|GB|tb|TB)$`)
								matches := rgx.FindAllStringSubmatch(rSlice[1], -1)
								size, symbol := matches[0][1], matches[0][2]
								size64, _ := strconv.ParseInt(size, 10, 64)
								fh := value.Interface().(*multipart.FileHeader)
								switch strings.ToLower(symbol) {
								case "kb":
									if fh.Size > int64(kilobyte*size64) {
										v.setMessage("size.file_kb", customMsg, jsonTag, formattedField, msgChan, size)
										return
									}
								case "mb":
									if fh.Size > int64(megabyte*size64) {
										v.setMessage("size.file_mb", customMsg, jsonTag, formattedField, msgChan, size)
										return
									}
								case "gb":
									if fh.Size > int64(gigabyte*size64) {
										v.setMessage("size.file_gb", customMsg, jsonTag, formattedField, msgChan, size)
										return
									}
								}

							}
						}
					}
				} else {
					if msg := New(&Config{Locale: v.locale, DB: v.dbConfig}).ValidateStruct(value.Interface()); msg != nil {
						v.setMessage("", msg, jsonTag, formattedField, msgChan)
						return
					}
				}
			}

		}
	}
	v.setMessage("empty", "", jsonTag, formattedField, msgChan)
}

func decodeMultipart(r *http.Request, v any) {
	elType := reflect.TypeOf(v)
	elValue := reflect.ValueOf(v)
	if elType.Kind() != reflect.Pointer || elValue.Kind() != reflect.Pointer {
		panic("validate: a pointer is expected as an argument")
	}
	elemType := elType.Elem()
	elemValue := elValue.Elem()
	r.ParseMultipartForm(0)
	wg := &sync.WaitGroup{}
	for i := 0; i < elemType.NumField(); i++ {
		wg.Add(1)
		go func(i int, r *http.Request, v any) {
			defer wg.Done()
			val, _ := elemType.Field(i).Tag.Lookup("json")
			switch elemValue.Field(i).Kind() {
			case reflect.String:
				elemValue.Field(i).Set(reflect.ValueOf(r.FormValue(val)))

			case reflect.Int:
				formVal, _ := strconv.Atoi(r.FormValue(val))
				elemValue.Field(i).Set(reflect.ValueOf(formVal))
			case reflect.Int8:
				formVal, _ := strconv.ParseInt(r.FormValue(val), 10, 8)
				elemValue.Field(i).Set(reflect.ValueOf(int8(formVal)))
			case reflect.Int16:
				formVal, _ := strconv.ParseInt(r.FormValue(val), 10, 16)
				elemValue.Field(i).Set(reflect.ValueOf(int16(formVal)))
			case reflect.Int32:
				formVal, _ := strconv.ParseInt(r.FormValue(val), 10, 32)
				elemValue.Field(i).Set(reflect.ValueOf(int32(formVal)))
			case reflect.Int64:
				formVal, _ := strconv.ParseInt(r.FormValue(val), 10, 64)
				elemValue.Field(i).Set(reflect.ValueOf(int64(formVal)))

			case reflect.Uint:
				formVal, _ := strconv.ParseUint(r.FormValue(val), 10, 64)
				elemValue.Field(i).Set(reflect.ValueOf(uint64(formVal)))
			case reflect.Uint8:
				formVal, _ := strconv.ParseUint(r.FormValue(val), 10, 8)
				elemValue.Field(i).Set(reflect.ValueOf(uint8(formVal)))
			case reflect.Uint16:
				formVal, _ := strconv.ParseUint(r.FormValue(val), 10, 16)
				elemValue.Field(i).Set(reflect.ValueOf(uint16(formVal)))
			case reflect.Uint32:
				formVal, _ := strconv.ParseUint(r.FormValue(val), 10, 32)
				elemValue.Field(i).Set(reflect.ValueOf(uint32(formVal)))
			case reflect.Uint64:
				formVal, _ := strconv.ParseUint(r.FormValue(val), 10, 64)
				elemValue.Field(i).Set(reflect.ValueOf(uint64(formVal)))

			case reflect.Float32:
				formVal, _ := strconv.ParseFloat(r.FormValue(val), 32)
				elemValue.Field(i).Set(reflect.ValueOf(float32(formVal)))
			case reflect.Float64:
				formVal, _ := strconv.ParseFloat(r.FormValue(val), 64)
				elemValue.Field(i).Set(reflect.ValueOf(float64(formVal)))

			case reflect.Bool:
				formVal, _ := strconv.ParseBool(r.FormValue(val))
				elemValue.Field(i).Set(reflect.ValueOf(formVal))

			case reflect.Slice, reflect.Array:

				if _, ok := elemValue.Field(i).Interface().([]*multipart.FileHeader); ok {
					elemValue.Field(i).Set(reflect.ValueOf(r.MultipartForm.File[val]))
				} else {

					switch elemValue.Field(i).Type().Elem().Kind() {
					case reflect.Int:
						sliceVal := make([]int, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.Atoi(sv)
							sliceVal = append(sliceVal, formVal)
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Int8:
						sliceVal := make([]int8, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseInt(r.FormValue(sv), 10, 8)
							sliceVal = append(sliceVal, int8(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Int16:
						sliceVal := make([]int16, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseInt(r.FormValue(sv), 10, 16)
							sliceVal = append(sliceVal, int16(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Int32:
						sliceVal := make([]int32, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseInt(r.FormValue(sv), 10, 32)
							sliceVal = append(sliceVal, int32(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Int64:
						sliceVal := make([]int64, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseInt(r.FormValue(sv), 10, 64)
							sliceVal = append(sliceVal, int64(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))

					case reflect.Uint:
						sliceVal := make([]uint, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseUint(r.FormValue(sv), 10, 64)
							sliceVal = append(sliceVal, uint(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Uint8:
						sliceVal := make([]uint8, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseUint(r.FormValue(sv), 10, 8)
							sliceVal = append(sliceVal, uint8(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Uint16:
						sliceVal := make([]uint16, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseUint(r.FormValue(sv), 10, 16)
							sliceVal = append(sliceVal, uint16(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Uint32:
						sliceVal := make([]uint32, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseUint(r.FormValue(sv), 10, 32)
							sliceVal = append(sliceVal, uint32(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Uint64:
						sliceVal := make([]uint64, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseUint(r.FormValue(sv), 10, 64)
							sliceVal = append(sliceVal, uint64(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))

					case reflect.Float32:
						sliceVal := make([]float32, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseFloat(r.FormValue(sv), 32)
							sliceVal = append(sliceVal, float32(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					case reflect.Float64:
						sliceVal := make([]float64, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseFloat(r.FormValue(sv), 64)
							sliceVal = append(sliceVal, float64(formVal))
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))

					case reflect.Bool:
						sliceVal := make([]bool, 0, len(r.PostForm[val]))
						for _, sv := range r.PostForm[val] {
							formVal, _ := strconv.ParseBool(r.FormValue(sv))
							sliceVal = append(sliceVal, formVal)
						}
						elemValue.Field(i).Set(reflect.ValueOf(sliceVal))
					default:
						elemValue.Field(i).Set(reflect.ValueOf(r.PostForm[val]))

					}
				}
			case reflect.Interface, reflect.Ptr:
				if _, ok := elemValue.Field(i).Interface().(*multipart.FileHeader); ok {
					_, fh, _ := r.FormFile(val)
					elemValue.Field(i).Set(reflect.ValueOf(fh))
				}
			}

		}(i, r, v)
	}
	wg.Wait()
}

func (v *validation) getTagAndValue(lookupTag string) (tag string, value reflect.Value) {
	for i := 0; i < v.elemType.NumField(); i++ {
		if t, ok := v.elemType.Field(i).Tag.Lookup("json"); ok {
			if t == lookupTag {
				tag = t
				value = v.elemValue.Field(i)
				return
			}
		}
	}
	return
}

func (v *validation) setMessage(ruleKey string, customMsg any, msgKey, field string, msgChan chan message, values ...string) {
	if ruleKey == "empty" {
		msgChan <- message{
			K: msgKey,
			V: customMsg,
		}
	} else if customMsg != "" {
		msgChan <- message{
			K: msgKey,
			V: customMsg,
		}
	} else {
		if values != nil {
			if len(values) > 1 {
				msgChan <- message{
					K: msgKey,
					V: fmt.Sprintf(v.getMessage(ruleKey), field, values[0], values[1]),
				}
			} else {
				msgChan <- message{
					K: msgKey,
					V: fmt.Sprintf(v.getMessage(ruleKey), field, values[0]),
				}
			}
		} else {
			msgChan <- message{
				K: msgKey,
				V: fmt.Sprintf(v.getMessage(ruleKey), field),
			}
		}
	}
}

func (v *validation) generateMessage(ruleKey string, customMsg any, field string, values ...string) any {
	if ruleKey == "empty" {
		return ""
	} else if customMsg != "" {
		return customMsg
	} else {
		if values != nil {
			if len(values) > 1 {
				return fmt.Sprintf(v.getMessage(ruleKey), field, values[0], values[1])
			}
			return fmt.Sprintf(v.getMessage(ruleKey), field, values[0])
		}
		return fmt.Sprintf(v.getMessage(ruleKey), field)
	}
}

func (v *validation) getMessage(rule string) string {
	var message map[string]any
	switch strings.ToLower(v.locale) {
	case "fr":
		message = locale.FR
	default:
		message = locale.EN
	}
	if strings.Contains(rule, ".") {
		keys := strings.SplitN(rule, ".", 2)
		msg := message[keys[0]].(map[string]string)
		return msg[keys[1]]
	}
	return message[rule].(string)
}
