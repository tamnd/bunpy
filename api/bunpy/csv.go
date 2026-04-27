package bunpy

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildCSV(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.csv", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("parse", &goipyObject.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("csv.parse() requires a string argument")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("csv.parse(): argument must be str")
			}
			withHeader := true
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("header"); ok2 {
					if b, ok3 := hv.(*goipyObject.Bool); ok3 {
						withHeader = b.V
					}
				}
			}
			return csvParse(s.V, withHeader)
		},
	})

	mod.Dict.SetStr("parse_file", &goipyObject.BuiltinFunc{
		Name: "parse_file",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("csv.parse_file() requires a path argument")
			}
			p, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("csv.parse_file(): path must be str")
			}
			data, err := os.ReadFile(p.V)
			if err != nil {
				return nil, fmt.Errorf("csv.parse_file(): %w", err)
			}
			withHeader := true
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("header"); ok2 {
					if b, ok3 := hv.(*goipyObject.Bool); ok3 {
						withHeader = b.V
					}
				}
			}
			return csvParse(string(data), withHeader)
		},
	})

	mod.Dict.SetStr("write", &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("csv.write() requires a rows argument")
			}
			rows, ok := args[0].(*goipyObject.List)
			if !ok {
				return nil, fmt.Errorf("csv.write(): rows must be a list")
			}
			var headerRow []string
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("header"); ok2 {
					if hl, ok3 := hv.(*goipyObject.List); ok3 {
						for _, h := range hl.V {
							if hs, ok4 := h.(*goipyObject.Str); ok4 {
								headerRow = append(headerRow, hs.V)
							}
						}
					}
				}
			}
			out, err := csvWrite(rows, headerRow)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: out}, nil
		},
	})

	mod.Dict.SetStr("write_file", &goipyObject.BuiltinFunc{
		Name: "write_file",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("csv.write_file() requires path and rows arguments")
			}
			p, ok1 := args[0].(*goipyObject.Str)
			rows, ok2 := args[1].(*goipyObject.List)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("csv.write_file(): path must be str, rows must be list")
			}
			out, err := csvWrite(rows, nil)
			if err != nil {
				return nil, err
			}
			if err2 := os.WriteFile(p.V, []byte(out), 0o644); err2 != nil {
				return nil, fmt.Errorf("csv.write_file(): %w", err2)
			}
			return goipyObject.None, nil
		},
	})

	return mod
}

func csvParse(src string, withHeader bool) (goipyObject.Object, error) {
	r := csv.NewReader(strings.NewReader(src))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv.parse(): %w", err)
	}
	if len(records) == 0 {
		return &goipyObject.List{V: nil}, nil
	}

	if !withHeader {
		items := make([]goipyObject.Object, 0, len(records))
		for _, row := range records {
			rowList := make([]goipyObject.Object, len(row))
			for j, cell := range row {
				rowList[j] = &goipyObject.Str{V: cell}
			}
			items = append(items, &goipyObject.List{V: rowList})
		}
		return &goipyObject.List{V: items}, nil
	}

	headers := records[0]
	items := make([]goipyObject.Object, 0, len(records)-1)
	for _, row := range records[1:] {
		d := goipyObject.NewDict()
		for j, cell := range row {
			if j < len(headers) {
				d.SetStr(headers[j], &goipyObject.Str{V: cell})
			}
		}
		items = append(items, d)
	}
	return &goipyObject.List{V: items}, nil
}

func csvWrite(rows *goipyObject.List, headerRow []string) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if len(rows.V) == 0 {
		w.Flush()
		return buf.String(), nil
	}

	// auto-detect: list of dicts or list of lists
	switch rows.V[0].(type) {
	case *goipyObject.Dict:
		// collect header from first row if not provided
		if headerRow == nil {
			first := rows.V[0].(*goipyObject.Dict)
			keys, _ := first.Items()
			for _, k := range keys {
				if ks, ok := k.(*goipyObject.Str); ok {
					headerRow = append(headerRow, ks.V)
				}
			}
		}
		w.Write(headerRow)
		for _, rowObj := range rows.V {
			d, ok := rowObj.(*goipyObject.Dict)
			if !ok {
				continue
			}
			record := make([]string, len(headerRow))
			for j, h := range headerRow {
				if v, ok2 := d.GetStr(h); ok2 {
					record[j] = fmt.Sprintf("%v", pyObjToGoValue(v))
				}
			}
			w.Write(record)
		}
	default:
		if headerRow != nil {
			w.Write(headerRow)
		}
		for _, rowObj := range rows.V {
			lst, ok := rowObj.(*goipyObject.List)
			if !ok {
				continue
			}
			record := make([]string, len(lst.V))
			for j, cell := range lst.V {
				record[j] = fmt.Sprintf("%v", pyObjToGoValue(cell))
			}
			w.Write(record)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("csv.write(): %w", err)
	}
	return buf.String(), nil
}
