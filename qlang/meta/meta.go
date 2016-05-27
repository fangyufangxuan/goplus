package meta

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"qlang.io/exec.v2"
	"qlang.io/qlang.spec.v1"
)

// Exports is the export table of this module.
//
var Exports = map[string]interface{}{
	"_name":   "qlang.io/qlang/meta",
	"fnlist":  FnList,
	"fntable": FnTable,
	"pkgs":    GoPkgList,
	"dir":     Dir,
	"doc":     Doc,
}

// FnList returns qlang all function list
//
func FnList() (list []string) {
	for k, _ := range qlang.Fntable {
		if !strings.HasPrefix(k, "$") {
			list = append(list, k)
		}
	}
	return
}

// FnTable returns qlang all function table
//
func FnTable() map[string]interface{} {
	table := make(map[string]interface{})
	for k, v := range qlang.Fntable {
		if !strings.HasPrefix(k, "$") {
			table[k] = v
		}
	}
	return table
}

// GoPkgList returns qlang Go implemented module list
//
func GoPkgList() (list []string) {
	return qlang.GoModuleList()
}

func IsExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func ExporStructField(t reflect.Type) ([]string, error) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.New("type is not struct")
	}
	var list []string
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		if IsExported(name) {
			list = append(list, name)
		}
	}
	return list, nil
}

// Dir returns list object of strings.
// for a module: the module's attributes.
// for a go struct object: field and func list.
// for a qlang class: function list.
// for a qlang class object: function and vars list.
//
func Dir(i interface{}) (list []string) {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Map {
		for _, k := range v.MapKeys() {
			list = append(list, k.String())
		}
	} else {
		switch e := i.(type) {
		case *exec.Class:
			for k := range e.Fns {
				list = append(list, k)
			}
		case *exec.Object:
			for k := range e.Cls.Fns {
				list = append(list, k)
			}
			for k := range e.Vars() {
				list = append(list, k)
			}
		default:
			t := v.Type()
			// list struct field
			if field, err := ExporStructField(t); err == nil {
				list = append(list, field...)
			}
			// list type method
			for i := 0; i < t.NumMethod(); i++ {
				name := t.Method(i).Name
				if IsExported(name) {
					list = append(list, name)
				}
			}
		}
	}
	return
}

func findPackageName(i interface{}) (string, bool) {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Map {
		for _, k := range v.MapKeys() {
			if k.Kind() == reflect.String && k.String() == "_name" {
				ev := v.MapIndex(k)
				if ev.Kind() == reflect.Interface {
					rv := ev.Elem()
					if rv.Kind() == reflect.String {
						return rv.String(), true
					}
				}
			}
		}
	}
	return "", false
}

// Doc returns doc info of object
//
func Doc(i interface{}) string {
	var buf bytes.Buffer
	outf := func(format string, a ...interface{}) (err error) {
		_, err = buf.WriteString(fmt.Sprintf(format, a...))
		return
	}
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Map {
		pkgName, isPkg := findPackageName(i)
		if isPkg {
			outf("package %v", pkgName)
			for _, k := range v.MapKeys() {
				if strings.HasPrefix(k.String(), "_") {
					continue
				}
				ev := v.MapIndex(k)
				if ev.Kind() == reflect.Interface {
					rv := ev.Elem()
					outf("\n%v\t%v ", k, rv.Type())
				}
			}
		} else {
			for _, k := range v.MapKeys() {
				ev := v.MapIndex(k)
				outf("\n%v\t%v", k, ev)
			}
		}
	} else {
		switch e := i.(type) {
		case *exec.Class:
			outf("*exec.Class")
			for k, v := range e.Fns {
				outf("\n%v\t%T", k, v)
			}
		case *exec.Object:
			outf("*exec.Object")
			for k, v := range e.Cls.Fns {
				outf("\n%v\t%T", k, v)
			}
			for k, kv := range e.Vars() {
				outf("\n%v\t%T", k, kv)
			}
		default:
			t := v.Type()
			outf("%v", t)
			{
				t := v.Type()
				for t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				if t.Kind() == reflect.Struct {
					for i := 0; i < t.NumField(); i++ {
						field := t.Field(i)
						if IsExported(field.Name) {
							outf("\n%v\t%v", field.Name, field.Type)
						}
					}
				}
			}
			for i := 0; i < t.NumMethod(); i++ {
				m := t.Method(i)
				if IsExported(m.Name) {
					outf("\n%v\t%v", m.Name, m.Type)
				}
			}
		}
	}
	return buf.String()
}
