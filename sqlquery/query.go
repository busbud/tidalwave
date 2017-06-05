package sqlquery

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dustinblackman/tidalwave/logger"
	"github.com/tidwall/gjson"
	dry "github.com/ungerik/go-dry"
	sqlparser "github.com/youtube/vitess/go/vt/sqlparser"
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

// List of operators that use the Regex field in QueryParam
var regexOperators = []string{"like", "ilike", "regexp"}

// A list of strings replaced in a query string before being passed to the parser to avoid parsing errors.
var stringReplacements = [][]string{
	{"-", "__DASH__"},
	{".", "__DOT__"},
}

// QueryParam holds a single piece of a queries WHERE and SELECT statements to be processed on log lines
type QueryParam struct {
	IsInt          bool
	KeyPath        string
	Regex          *regexp.Regexp
	Operator       string
	ValInt         int
	ValIntArray    []int
	ValString      string
	ValStringArray []string
}

// QueryParams holds all the information for a given query such SELECT, FROM, and WHERE statements to be easily processed later.
type QueryParams struct {
	From []string // TODO: Rename to Froms

	AggrPath string
	Dates    []DateParam
	Queries  []QueryParam // TODO Rename to Where
	Selects  []QueryParam
	Type     string
}

// Youtube's SQL parser doesn't like some characters in parts of the query.
// We replace them in New, and them restore them here after parsing the sql parsers response.
func repairString(key string) string {
	for _, entry := range stringReplacements {
		key = strings.Replace(key, entry[1], entry[0], -1)
	}
	return key
}

func assignTypeFieldsToParam(param QueryParam, value string) QueryParam {
	if i, err := strconv.Atoi(value); err == nil {
		param.IsInt = true
		param.ValInt = i
	} else {
		param.ValString = repairString(stripQuotes(value))

		// Handles building the Regex field on param when a string is selected
		if dry.StringListContains(regexOperators, param.Operator) {
			regexString := ""
			if param.ValString[:1] == "%" && param.ValString[len(param.ValString)-1:] != "%" {
				regexString = "^" + param.ValString[1:]
			} else if param.ValString[:1] != "%" && param.ValString[len(param.ValString)-1:] == "%" {
				regexString = param.ValString[:len(param.ValString)-1] + "$"
			} else if param.ValString[:1] == "%" && param.ValString[len(param.ValString)-1:] == "%" {
				regexString = param.ValString[1 : len(param.ValString)-1]
			} else {
				regexString = param.ValString
			}

			if param.Operator == "regexp" {
				param.Operator = "ilike" // TODO: Temp workaround
				regexString = "(?i)" + regexString
			}
			param.Regex = regexp.MustCompile(regexString)
		}
	}
	return param
}

func handleCompareExpr(expr *sqlparser.ComparisonExpr) QueryParam {
	param := QueryParam{
		KeyPath:  repairString(stripQuotes(sqlparser.String(expr.Left))),
		Operator: expr.Operator,
	}

	right := sqlparser.String(expr.Right)
	if expr.Operator == "in" {
		arrayValues := strings.Split(right[1:len(right)-1], ", ")
		for _, val := range arrayValues {
			if i, err := strconv.Atoi(val); err == nil {
				param.IsInt = true
				param.ValIntArray = append(param.ValIntArray, i)
			} else {
				param.ValStringArray = append(param.ValStringArray, repairString(stripQuotes(val)))
			}
		}

		// We can't have mixed types wheh comparing arrays. We default to strings if not all values were convertable to
		// numbers
		if len(param.ValIntArray) > 0 && len(param.ValStringArray) != 0 {
			param.IsInt = false
			for _, val := range param.ValIntArray {
				param.ValStringArray = append(param.ValStringArray, string(val))
			}
			param.ValIntArray = []int{}
		}
	} else {
		param = assignTypeFieldsToParam(param, right)
	}

	return param
}

func handleAndExpr(expr *sqlparser.AndExpr) []QueryParam {
	params := []QueryParam{}

	for _, side := range []interface{}{expr.Left, expr.Right} {
		params = append(params, handleExpr(side)...)
	}
	return params
}

func handleRandExpr(expr *sqlparser.RangeCond) []QueryParam {
	keypath := repairString(stripQuotes(sqlparser.String(expr.Left)))
	fromQuery := assignTypeFieldsToParam(QueryParam{
		KeyPath:  keypath,
		Operator: ">=",
	}, sqlparser.String(expr.From))
	toQuery := assignTypeFieldsToParam(QueryParam{
		KeyPath:  keypath,
		Operator: "<=",
	}, sqlparser.String(expr.To))

	return []QueryParam{fromQuery, toQuery}
}

func handleExpr(entry interface{}) []QueryParam {
	params := []QueryParam{}
	switch expr := entry.(type) {
	case *sqlparser.AndExpr:
		params = append(params, handleAndExpr(expr)...)
	case *sqlparser.ComparisonExpr:
		params = append(params, handleCompareExpr(expr))
	case *sqlparser.RangeCond:
		params = append(params, handleRandExpr(expr)...)
	}

	return params
}

// ProcessLine interates through all Queries created during the query parsing returning a bool stating whether all matched.
func (qp *QueryParams) ProcessLine(line *[]byte) bool {
	matchMap := []bool{}
	for _, q := range qp.Queries {
		value := gjson.GetBytes(*line, q.KeyPath)
		if value.Type == 0 { // gjson way of saying key not found
			break
		}

		if value.Type == gjson.JSON && q.Operator == "exists" {
			matchMap = append(matchMap, true)
		} else if q.IsInt && value.Type == gjson.Number {
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
	logger.Logger.Debug("Query: " + queryString)
	queryParams := QueryParams{Type: TypeSearch} // Default is search. TODO Move to if statement

	// Fixes "date" breaking the parser by wrapping it in quotes
	queryString = strings.Replace(queryString, " date", " 'date'", -1)
	// Adds support for ~~.
	queryString = strings.Replace(queryString, " ~~ ", " like ", -1)
	// TODO This is a temporary workaround to have "ilike" support.
	// The sqlparser doesn't accept ilike, so we use rlike instead until a better solution comes around.
	queryString = strings.Replace(queryString, " ilike ", " rlike ", -1)
	// Replace characters that the SQL parser won't accept that will be reverted back after parsing
	for _, entry := range stringReplacements {
		queryString = strings.Replace(queryString, entry[0], entry[1], -1)
	}

	tree, err := sqlparser.Parse(queryString)
	if err != nil {
		logger.Logger.Error(err.Error())
	}
	queryTree := tree.(*sqlparser.Select)

	// Selects
	// Makes sure the selected keys we want exists in the line.
	logger.Logger.Debugf("Query Tree: %s", spew.Sdump(queryTree))
	for _, entry := range queryTree.SelectExprs {
		// TODO: Support star expression
		switch entry := entry.(type) {
		case *sqlparser.NonStarExpr:
			switch exp := entry.Expr.(type) {
			// Simple selects
			case *sqlparser.ColName:
				queryParams.Selects = append(queryParams.Selects, QueryParam{
					KeyPath:  repairString(sqlparser.String(exp)),
					Operator: "exists",
				})
			// DISTINCT()
			case *sqlparser.ParenExpr:
				keyPath := repairString(sqlparser.String(exp.Expr.(*sqlparser.ColName)))
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
					queryParams.AggrPath = repairString(sqlparser.String(aggrPath[0].(*sqlparser.ColName)))
				case *sqlparser.ColName:
					queryParams.AggrPath = repairString(sqlparser.String(aggrPath))
				case *sqlparser.ParenExpr:
					// Fixes COUNT(DISTINCT())
					switch aggrPath := aggrPath.Expr.(type) {
					case sqlparser.ValTuple:
						queryParams.AggrPath = repairString(sqlparser.String(aggrPath[0].(*sqlparser.ColName)))
					case *sqlparser.ColName:
						queryParams.AggrPath = repairString(sqlparser.String(aggrPath))
					}
				}

				queryParams.Selects = append(queryParams.Selects, QueryParam{
					KeyPath:  queryParams.AggrPath,
					Operator: "exists",
				})

				// TODO This will need to get a bit more advance when we start adding other functions.
				name := sqlparser.String(exp.Name)
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
		queryParams.From = append(queryParams.From, repairString(sqlparser.String(entry)))
	}

	if queryTree.Where != nil {
		for _, entry := range handleExpr(queryTree.Where.Expr) {
			if entry.KeyPath == "date" {
				queryParams.Dates = append(queryParams.Dates, createDateParam(entry.ValString, entry.Operator))
			} else {
				queryParams.Queries = append(queryParams.Queries, entry)
			}
		}
	}

	queryParams.Queries = append(queryParams.Selects, queryParams.Queries...)
	logger.Logger.Debugf("Query Params: %s", spew.Sdump(queryParams))
	return &queryParams
}
