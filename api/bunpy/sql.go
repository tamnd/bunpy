package bunpy

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildSQL returns the bunpy.sql built-in function.
func BuildSQL(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "sql",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			rawURL := ""
			var urlArg goipyObject.Object
			if len(args) >= 1 {
				urlArg = args[0]
			} else if kwargs != nil {
				urlArg, _ = kwargs.GetStr("url")
			}
			if urlArg != nil {
				if s, ok := urlArg.(*goipyObject.Str); ok {
					rawURL = s.V
				}
			}

			db, err := openDB(rawURL)
			if err != nil {
				return nil, fmt.Errorf("bunpy.sql(): %w", err)
			}
			if err2 := db.Ping(); err2 != nil {
				db.Close()
				return nil, fmt.Errorf("bunpy.sql(): cannot connect: %w", err2)
			}
			return buildDBInstance(i, db), nil
		},
	}
}

// openDB selects the driver and DSN based on the URL scheme.
func openDB(rawURL string) (*sql.DB, error) {
	switch {
	case rawURL == "" || rawURL == ":memory:":
		return sql.Open("sqlite", "file::memory:?cache=shared")
	case strings.HasPrefix(rawURL, "sqlite:"):
		return sql.Open("sqlite", parseSQLiteDSN(rawURL))
	case strings.HasPrefix(rawURL, "postgres://"), strings.HasPrefix(rawURL, "postgresql://"):
		return sql.Open("pgx", rawURL)
	case strings.HasPrefix(rawURL, "mysql://"):
		dsn, err := mysqlDSN(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid mysql URL: %w", err)
		}
		return sql.Open("mysql", dsn)
	default:
		return nil, fmt.Errorf("unsupported database URL scheme: %q", rawURL)
	}
}

// mysqlDSN converts "mysql://user:pass@host:port/db" to the go-sql-driver/mysql DSN format.
func mysqlDSN(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	user := u.User.Username()
	pass, _ := u.User.Password()
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":3306"
	}
	db := strings.TrimPrefix(u.Path, "/")
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, db), nil
}

func parseSQLiteDSN(raw string) string {
	if strings.HasPrefix(raw, "sqlite:") {
		path := strings.TrimPrefix(raw, "sqlite:")
		if path == ":memory:" || path == "::memory:" {
			return "file::memory:?cache=shared"
		}
		return "file:" + path + "?cache=shared"
	}
	return raw
}

func buildDBInstance(i *goipyVM.Interp, db *sql.DB) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Database", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	attachDBMethods(i, inst, db)
	return inst
}

func attachDBMethods(i *goipyVM.Interp, inst *goipyObject.Instance, db *sql.DB) {
	inst.Dict.SetStr("query", &goipyObject.BuiltinFunc{
		Name: "query",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q, qArgs, err := parseQueryArgs(args)
			if err != nil {
				return nil, err
			}
			rows, err2 := db.Query(q, qArgs...)
			if err2 != nil {
				return nil, fmt.Errorf("db.query(): %w", err2)
			}
			defer rows.Close()
			return rowsToDicts(rows)
		},
	})
	inst.Dict.SetStr("query_one", &goipyObject.BuiltinFunc{
		Name: "query_one",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q, qArgs, err := parseQueryArgs(args)
			if err != nil {
				return nil, err
			}
			rows, err2 := db.Query(q, qArgs...)
			if err2 != nil {
				return nil, fmt.Errorf("db.query_one(): %w", err2)
			}
			defer rows.Close()
			lst, err3 := rowsToDicts(rows)
			if err3 != nil {
				return nil, err3
			}
			items := lst.(*goipyObject.List).V
			if len(items) == 0 {
				return goipyObject.None, nil
			}
			return items[0], nil
		},
	})
	inst.Dict.SetStr("run", &goipyObject.BuiltinFunc{
		Name: "run",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q, qArgs, err := parseQueryArgs(args)
			if err != nil {
				return nil, err
			}
			if _, err2 := db.Exec(q, qArgs...); err2 != nil {
				return nil, fmt.Errorf("db.run(): %w", err2)
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("run_many", &goipyObject.BuiltinFunc{
		Name: "run_many",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("db.run_many() requires query and rows arguments")
			}
			qStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("db.run_many(): query must be str")
			}
			rowsList, ok2 := args[1].(*goipyObject.List)
			if !ok2 {
				return nil, fmt.Errorf("db.run_many(): rows must be a list")
			}
			tx, err := db.Begin()
			if err != nil {
				return nil, fmt.Errorf("db.run_many(): %w", err)
			}
			stmt, err2 := tx.Prepare(qStr.V)
			if err2 != nil {
				tx.Rollback()
				return nil, fmt.Errorf("db.run_many(): %w", err2)
			}
			defer stmt.Close()
			for _, rowObj := range rowsList.V {
				row, ok3 := rowObj.(*goipyObject.List)
				if !ok3 {
					tx.Rollback()
					return nil, fmt.Errorf("db.run_many(): each row must be a list")
				}
				goRow := pyListToGoArgs(row)
				if _, err3 := stmt.Exec(goRow...); err3 != nil {
					tx.Rollback()
					return nil, fmt.Errorf("db.run_many(): %w", err3)
				}
			}
			if err3 := tx.Commit(); err3 != nil {
				return nil, fmt.Errorf("db.run_many(): %w", err3)
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("transaction", &goipyObject.BuiltinFunc{
		Name: "transaction",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("db.transaction() requires a callable argument")
			}
			tx, err := db.Begin()
			if err != nil {
				return nil, fmt.Errorf("db.transaction(): %w", err)
			}
			txInst := buildTxInstance(tx)
			_, callErr := i.Call(args[0], []goipyObject.Object{txInst}, nil)
			if callErr != nil {
				tx.Rollback()
				return nil, callErr
			}
			if err2 := tx.Commit(); err2 != nil {
				return nil, fmt.Errorf("db.transaction(): %w", err2)
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("close", &goipyObject.BuiltinFunc{
		Name: "close",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			db.Close()
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("__enter__", &goipyObject.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return inst, nil
		},
	})
	inst.Dict.SetStr("__exit__", &goipyObject.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			db.Close()
			return goipyObject.BoolOf(false), nil
		},
	})
}

func buildTxInstance(tx *sql.Tx) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Transaction", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	inst.Dict.SetStr("run", &goipyObject.BuiltinFunc{
		Name: "run",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q, qArgs, err := parseQueryArgs(args)
			if err != nil {
				return nil, err
			}
			if _, err2 := tx.Exec(q, qArgs...); err2 != nil {
				return nil, fmt.Errorf("tx.run(): %w", err2)
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("query", &goipyObject.BuiltinFunc{
		Name: "query",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q, qArgs, err := parseQueryArgs(args)
			if err != nil {
				return nil, err
			}
			rows, err2 := tx.Query(q, qArgs...)
			if err2 != nil {
				return nil, fmt.Errorf("tx.query(): %w", err2)
			}
			defer rows.Close()
			return rowsToDicts(rows)
		},
	})
	return inst
}

func rowsToDicts(rows *sql.Rows) (goipyObject.Object, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("rowsToDicts: %w", err)
	}
	var result []goipyObject.Object
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for j := range vals {
			ptrs[j] = &vals[j]
		}
		if err2 := rows.Scan(ptrs...); err2 != nil {
			return nil, fmt.Errorf("rowsToDicts scan: %w", err2)
		}
		d := goipyObject.NewDict()
		for j, col := range cols {
			d.SetStr(col, goValueToPyObj(vals[j]))
		}
		result = append(result, &goipyObject.Instance{
			Class: &goipyObject.Class{Name: "Row", Dict: goipyObject.NewDict()},
			Dict:  d,
		})
	}
	if err3 := rows.Err(); err3 != nil {
		return nil, fmt.Errorf("rowsToDicts: %w", err3)
	}
	return &goipyObject.List{V: result}, nil
}

func goValueToPyObj(v any) goipyObject.Object {
	switch val := v.(type) {
	case nil:
		return goipyObject.None
	case int64:
		return goipyObject.NewInt(val)
	case float64:
		return &goipyObject.Float{V: val}
	case bool:
		return goipyObject.BoolOf(val)
	case []byte:
		return &goipyObject.Bytes{V: val}
	case string:
		return &goipyObject.Str{V: val}
	case map[string]any:
		d := goipyObject.NewDict()
		for k, vv := range val {
			d.SetStr(k, goValueToPyObj(vv))
		}
		return d
	case []any:
		items := make([]goipyObject.Object, len(val))
		for i, vv := range val {
			items[i] = goValueToPyObj(vv)
		}
		return &goipyObject.List{V: items}
	default:
		return &goipyObject.Str{V: fmt.Sprintf("%v", val)}
	}
}

func parseQueryArgs(args []goipyObject.Object) (string, []any, error) {
	if len(args) < 1 {
		return "", nil, fmt.Errorf("query requires a SQL string argument")
	}
	qStr, ok := args[0].(*goipyObject.Str)
	if !ok {
		return "", nil, fmt.Errorf("query: first argument must be str")
	}
	var goArgs []any
	if len(args) >= 2 {
		lst, ok2 := args[1].(*goipyObject.List)
		if !ok2 {
			return "", nil, fmt.Errorf("query: second argument must be a list")
		}
		goArgs = pyListToGoArgs(lst)
	}
	return qStr.V, goArgs, nil
}

func pyListToGoArgs(lst *goipyObject.List) []any {
	result := make([]any, len(lst.V))
	for i, item := range lst.V {
		result[i] = pyObjToGoValue(item)
	}
	return result
}

func pyObjToGoValue(obj goipyObject.Object) any {
	switch v := obj.(type) {
	case *goipyObject.Int:
		return v.Int64()
	case *goipyObject.Float:
		return v.V
	case *goipyObject.Str:
		return v.V
	case *goipyObject.Bytes:
		return v.V
	case *goipyObject.Bool:
		return v.V
	case *goipyObject.NoneType:
		return nil
	default:
		return fmt.Sprintf("%v", obj)
	}
}
