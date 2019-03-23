package reditool

import (
	"database/sql"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/redis.v5"
)

func TTLLock(r *redis.Client, lKey string, d time.Duration) error {
	return r.Set(lKey, "LOCK", d).Err()
}

func JoinKey(r *redis.Client, fields ...string) string {
	return strings.Join(fields, ":")
}

// client := redis.NewClient(&redis.Options{
// 		Addr:     address,
// 		Password: "", // no password set
// 		DB:       0,  // use default DB
// 	})
func SwitchSet(r *redis.Client, key, dval string) (bool, error) {
	switch r.SIsMember(key, dval).Val() {
	case true:
		return false, r.SRem(key, dval).Err()

	case false:
		return true, r.SAdd(key, dval).Err()
	}
	return false, errors.New("switch")
}

func HGetOrSet(r *redis.Client, key, hk, val string) (string, bool) {
	v, err := r.HGet(key, hk).Result()
	switch {
	case err == redis.Nil:
		if val != "" {
			err := r.HSet(key, hk, val).Err()
			if err != nil {
				log.Fatal(err)
				return "", false
			}
			return val, true
		}
		return val, false
	case err != nil:
		log.Fatal(err)
		return val, false
	default:
		return v, false
	}
}

func GetOrSet(r *redis.Client, key, dval string) (string, bool) {
	v, err := r.Get(key).Result()
	switch {
	case err == redis.Nil:
		if dval != "" {
			err := r.Set(key, dval, 0).Err()
			if err != nil {
				log.Fatal(err)
				return "", false
			}
			return dval, true
		}
		return dval, false
	case err != nil:
		log.Fatal(err)
		return dval, false
	default:
		return v, false
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
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
	return false
}

func UpsertStruct(r *redis.Client, index string, s interface{}) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		rn := field.Tag.Get("json")
		if rn != "" {
			fv := v.FieldByIndex(field.Index)

			IsOmitEmpty := false
			if strings.Contains(rn, "omitempty") {
				IsOmitEmpty = true
				rn = strings.Split(rn, ",")[0]
			}
			if !fv.IsValid() || IsOmitEmpty && isEmptyValue(fv) {
				continue
			}
			//fmt.Println(r.JoinKey(index, rn), v.FieldByName(field.Name).Type())
			vv := v.FieldByName(field.Name)
			var sv interface{}
			switch vv.Kind() {
			case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
				sv = vv.String()
			case reflect.Bool:
				sv = vv.Bool()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sv = vv.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				sv = vv.Uint()
			case reflect.Float32, reflect.Float64:
				sv = v.Float()
				//case reflect.Interface, reflect.Ptr:
				//sv = v.IsNil()
			}
			r.Set(JoinKey(r, index, rn), sv, 0).Result()
		}
	}
	return
}

func NullInt64toInt64(i sql.NullInt64) int64 {
	if v, _ := i.Value(); v != nil {
		return v.(int64)
	}
	return 0
}

func NullStrtoStr(i sql.NullString) string {
	if v, _ := i.Value(); v != nil {
		return v.(string)
	}
	return ""
}

func wrap(s string, w string) string {
	return w + s + w
}

func kvq(k string, v interface{}) string {
	switch v.(type) {
	case string:
		if v != "" {
			return k + " = " + wrap(v.(string), "'") + ","
		}
	case int64:
		if v != 0 {
			return k + " = " + strconv.FormatInt(v.(int64), 10) + ","
		}
	}
	return ""
}
