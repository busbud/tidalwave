package sqlquery

import (
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/tidwall/gjson"
	"github.com/xwb1989/sqlparser"
)

const (
	// TypeCount specifies result is a count result
	TypeCount = "count"
	// TypeDistinct specifies result is a distinct result
	TypeDistinct = "distinct"
	// TypeCountDistinct specifies result is a count distinct result
	TypeCountDistinct = "count-distinct"
	// TypeSearch specifies specifies result is a search result
	TypeSearch = "search"
)

// QueryParam holds a single piece of a queries WHERE and SELECT statements to be processed on log lines
type QueryParam struct {
	KeyPath   string
	Operator  string
	ValString string
	ValInt    int
}

// QueryParams holds all the information for a given query such SELECT, FROM, and WHERE statements to be easily processed later.
type QueryParams struct {
	From []string // TODO: Rename to Froms

	LogPaths []string
	AggrPath string
	Dates    []DateParam
	Queries  []QueryParam // TODO Rename to Where
	Selects  []QueryParam
	Type     string
}

func handleCompareExpr(expr *sqlparser.ComparisonExpr) QueryParam {
	param := QueryParam{
		KeyPath:  stripQuotes(sqlparser.String(expr.Left)),
		Operator: expr.Operator,
	}

	right := sqlparser.String(expr.Right)
	if i, err := strconv.Atoi(right); err == nil {
		param.ValInt = i
	} else {
		param.ValString = stripQuotes(right)
	}

	return param
}

func handleAndExpr(expr *sqlparser.AndExpr) []QueryParam {
	params := []QueryParam{}

	for _, side := range []interface{}{expr.Left, expr.Right} {
		switch expr := side.(type) {
		case *sqlparser.AndExpr:
			params = append(params, handleAndExpr(expr)...)
		case *sqlparser.ComparisonExpr:
			params = append(params, handleCompareExpr(expr))
		}
	}
	return params
}

// ProcessLine interates through all Queries created during the query parsing returning a bool stating whether all matched.
func (qp *QueryParams) ProcessLine(line string) bool {
	matchMap := []bool{}
	for _, q := range qp.Queries {
		value := gjson.Get(line, q.KeyPath) // TODO: Test switching to GetBytes and use Scanner.Bytes() for better performance.
		if value.Type == 0 {                // gjson way of saying key not found
			break
		}

		if value.Type == gjson.Number { // TODO: Check if ValInt exists first.
			if ProcessInt(&q, int(value.Num)) {
				matchMap = append(matchMap, true)
			} else {
				break
			}
		} else {
			if ProcessString(&q, value.Str) {
				matchMap = append(matchMap, true)
			} else {
				break
			}
		}
	}

	return len(matchMap) == len(qp.Queries)
}

// New parses a query string and returns a newly created QueryParams struc holding all parsed data.
func New(queryString string) *QueryParams {
	logrus := logrus.WithFields(logrus.Fields{"module": "sqlquery"})
	logrus.Debug("Query: " + queryString)
	queryParams := QueryParams{Type: TypeSearch} // Default is search. TODO Move to if statement

	// Fixes "date" breaking the parser by wrapping it in quotes
	queryString = strings.Replace(queryString, " date", " 'date'", -1)
	tree, err := sqlparser.Parse(queryString)
	if err != nil {
		logrus.Error(err.Error())
	}
	queryTree := tree.(*sqlparser.Select)

	// Selects
	// Makes sure the selected keys we want exists in the line.
	for _, entry := range queryTree.SelectExprs {
		switch entry := entry.(type) {
		case *sqlparser.NonStarExpr:
			switch exp := entry.Expr.(type) {
			// Simply selects
			case *sqlparser.ColName:
				queryParams.Selects = append(queryParams.Selects, QueryParam{
					KeyPath:  sqlparser.String(exp),
					Operator: "exists",
				})
			// DISTINCT()
			case sqlparser.ValTuple:
				keyPath := sqlparser.String(exp[0].(*sqlparser.ColName))
				queryParams.Selects = append(queryParams.Selects, QueryParam{
					KeyPath:  keyPath,
					Operator: "exists",
				})

				// Where distinct is set
				if len(queryTree.Distinct) >= 8 {
					queryParams.Type = TypeDistinct
					queryParams.AggrPath = keyPath
				}
			// All other function expressions. COUNT(), COUNT(DISTINCT())
			case *sqlparser.FuncExpr:
				switch aggrPath := exp.Exprs[0].(*sqlparser.NonStarExpr).Expr.(type) {
				case sqlparser.ValTuple:
					queryParams.AggrPath = sqlparser.String(aggrPath[0].(*sqlparser.ColName))
				case *sqlparser.ColName:
					queryParams.AggrPath = sqlparser.String(aggrPath)
				}

				queryParams.Selects = append(queryParams.Selects, QueryParam{
					KeyPath:  queryParams.AggrPath,
					Operator: "exists",
				})

				// This will need to get a bit more advance when we start adding other functions.
				name := string(exp.Name)
				if name == "count" && exp.Distinct {
					queryParams.Type = TypeCountDistinct
				} else {
					queryParams.Type = TypeCount
				}
			}
		}
	}

	// Froms
	for _, entry := range queryTree.From {
		queryParams.From = append(queryParams.From, sqlparser.String(entry))
	}

	if queryTree.Where != nil {
		wheres := []QueryParam{}
		switch expr := queryTree.Where.Expr.(type) {
		case *sqlparser.AndExpr:
			wheres = append(wheres, handleAndExpr(expr)...)
		case *sqlparser.ComparisonExpr:
			wheres = append(wheres, handleCompareExpr(expr))
		}

		for _, entry := range wheres {
			if entry.KeyPath == "date" {
				queryParams.Dates = append(queryParams.Dates, createDateParam(entry.ValString, entry.Operator))
			} else {
				queryParams.Queries = append(queryParams.Queries, entry)
			}
		}
	}

	queryParams.Queries = append(queryParams.Selects, queryParams.Queries...)
	logrus.Debug(spew.Sdump(queryParams))
	return &queryParams
}
