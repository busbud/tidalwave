package sqlquery

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dustinblackman/gjson"
	"github.com/dustinblackman/tidalwave/logger"
	pgQuery "github.com/lfittl/pg_query_go"
	pgNodes "github.com/lfittl/pg_query_go/nodes"
	dry "github.com/ungerik/go-dry"
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
var regexOperators = []string{"regexp", "~~", "~~*"}

// List of supported postgres functions
var supportedFunctions = []string{"count", "distinct"}

// A list of strings replaced in a query string before being passed to the parser to avoid parsing errors.
var stringReplacements = [][]string{
	{"-", "__dash__"},
	{".*.", ".__map__."},
}

// QueryParam holds a single piece of a queries WHERE and SELECT statements to be processed on log lines
type QueryParam struct {
	IsInt          bool
	KeyName        string
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
	SQLString      string
	SQLStringLower string

	From []string // TODO: Rename to Froms

	AggrPath  string
	Dates     []DateParam
	Queries   []QueryParam // TODO Rename to Where
	QueryKeys []string
	Selects   []string
	Type      string
}

func convertAConst(expr pgNodes.A_Const) string {
	switch val := expr.Val.(type) {
	case pgNodes.String:
		return val.Str
	case pgNodes.Integer:
		return strconv.Itoa(int(val.Ival))
	}

	return "" // TODO
}

// Postgres' SQL parser doesn't like some characters in parts of the query.
// We replace them in New, and them restore them here after parsing the sql parsers response.
func (qp *QueryParams) repairString(key string) string {
	for _, entry := range stringReplacements {
		key = strings.Replace(key, entry[1], entry[0], -1)
	}

	// Postgres' parser makes the entire string lower case before parsing it. This restores the casing.
	idx := strings.Index(qp.SQLStringLower, strings.ToLower(key))
	return qp.SQLString[idx : idx+len(key)]
}

func (qp *QueryParams) getSelectNodeString(selectNodeVal pgNodes.ColumnRef) string {
	selectStrings := []string{}
	for _, item := range selectNodeVal.Fields.Items {
		switch item := item.(type) {
		case pgNodes.String:
			selectStrings = append(selectStrings, item.Str)
		}
	}

	return qp.repairString(strings.Join(selectStrings, "."))
}

func (qp *QueryParams) assignTypeFieldsToParam(param QueryParam, value string) QueryParam {
	if i, err := strconv.Atoi(value); err == nil {
		param.IsInt = true
		param.ValInt = i
	} else {
		param.ValString = qp.repairString(stripQuotes(value))

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

			if param.Operator == "~~*" {
				regexString = "(?i)" + regexString
			}
			param.Regex = regexp.MustCompile(regexString)
		}
	}
	return param
}

func (qp *QueryParams) handleCompareExpr(expr pgNodes.A_Expr) []QueryParam {
	// Param root used for everything except BETWEEN.
	param := QueryParam{
		KeyPath:  qp.getSelectNodeString(expr.Lexpr.(pgNodes.ColumnRef)),
		Operator: expr.Name.Items[0].(pgNodes.String).Str,
	}

	switch right := expr.Rexpr.(type) {
	case pgNodes.A_Const:
		param = qp.assignTypeFieldsToParam(param, convertAConst(right))

	case pgNodes.List:
		if param.Operator == "BETWEEN" {
			fromQuery := qp.assignTypeFieldsToParam(QueryParam{
				KeyPath:  param.KeyPath,
				Operator: ">=",
			}, convertAConst(right.Items[0].(pgNodes.A_Const)))
			toQuery := qp.assignTypeFieldsToParam(QueryParam{
				KeyPath:  param.KeyPath,
				Operator: "<=",
			}, convertAConst(right.Items[1].(pgNodes.A_Const)))

			return []QueryParam{fromQuery, toQuery}
		}

		for _, val := range right.Items {
			val := convertAConst(val.(pgNodes.A_Const))
			if i, err := strconv.Atoi(val); err == nil {
				param.IsInt = true
				param.ValIntArray = append(param.ValIntArray, i)
			} else {
				param.ValStringArray = append(param.ValStringArray, qp.repairString(stripQuotes(val)))
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
	}

	return []QueryParam{param}
}

func (qp *QueryParams) handleAndExpr(expr pgNodes.BoolExpr) []QueryParam {
	params := []QueryParam{}
	for _, whereExpr := range expr.Args.Items {
		params = append(params, qp.handleExpr(whereExpr)...)
	}

	return params
}

func (qp *QueryParams) handleExpr(entry interface{}) []QueryParam {
	params := []QueryParam{}
	switch expr := entry.(type) {
	case pgNodes.A_Expr:
		params = append(params, qp.handleCompareExpr(expr)...)
	case pgNodes.BoolExpr:
		params = append(params, qp.handleAndExpr(expr)...)
	}

	return params
}

// ProcessLine interates through all Queries created during the query parsing returning a bool stating whether all matched.
func (qp *QueryParams) ProcessLine(line *[]byte) bool {
	for idx, path := range qp.QueryKeys {
		value := gjson.GetBytes(*line, path)
		if value.Type == 0 { // gjson way of saying key not found
			return false
		}

		q := &qp.Queries[idx]
		if q.IsInt && value.Type == gjson.Number {
			if !ProcessInt(q, int(value.Num)) {
				return false
			}
		} else {
			if !ProcessString(q, value.String()) {
				return false
			}
		}
	}

	return true
}

// New parses a query string and returns a newly created QueryParams struc holding all parsed data.
func New(queryString string) *QueryParams {
	logger.Log.Debug("Query: " + queryString)
	qp := QueryParams{
		SQLString:      queryString,
		SQLStringLower: strings.ToLower(queryString),
		Type:           TypeSearch, // Default is search. TODO Move to if statement
	}

	// Replace characters that the SQL parser won't accept that will be reverted back after parsing
	for _, entry := range stringReplacements {
		queryString = strings.Replace(queryString, entry[0], entry[1], -1)
	}

	tree, err := pgQuery.Parse(queryString)
	if err != nil {
		logger.Log.Error(err.Error())
	}

	logger.Log.Debugf("Query Tree: %s", spew.Sdump(tree))
	statement := tree.Statements[0].(pgNodes.RawStmt).Stmt.(pgNodes.SelectStmt)
	isDistrinct := len(statement.DistinctClause.Items) > 0

	// Where clauses
	if statement.WhereClause != nil {
		for _, entry := range qp.handleExpr(statement.WhereClause) {
			if entry.KeyPath == "date" {
				qp.Dates = append(qp.Dates, createDateParam(entry.ValString, entry.Operator)...)
			} else {
				qp.Queries = append(qp.Queries, entry)
			}
		}
	}

	// Select statements
	for _, selectNode := range statement.TargetList.Items {
		selectNode := selectNode.(pgNodes.ResTarget)
		keyName := ""
		keyPath := ""

		if selectNode.Name != nil {
			keyName = *selectNode.Name
		}

		switch selectNodeVal := selectNode.Val.(type) {
		case pgNodes.ColumnRef: // Regular select statement
			keyPath = qp.getSelectNodeString(selectNodeVal)
			if len(keyPath) > 0 {
				if keyName == "" {
					keySplit := strings.Split(keyPath, ".")
					keyName = keySplit[len(keySplit)-1]
				}

				// TODO Kill the need for SELECTS
				qp.Selects = append(qp.Selects, keyPath)
				qp.Queries = append(qp.Queries, QueryParam{
					KeyName:  keyName,
					KeyPath:  keyPath,
					Operator: "exists",
				})
			}

		case pgNodes.FuncCall: // COUNT
			if len(selectNodeVal.Args.Items) > 0 {
				qp.AggrPath = qp.getSelectNodeString(selectNodeVal.Args.Items[0].(pgNodes.ColumnRef))
				qp.Selects = append(qp.Selects, qp.AggrPath)
				qp.Queries = append(qp.Queries, QueryParam{
					KeyPath:  qp.AggrPath,
					Operator: "exists",
				})
			}

			funcType := selectNodeVal.Funcname.Items[0].(pgNodes.String).Str
			if !dry.StringListContains(supportedFunctions, funcType) {
				logger.Log.Panicf("%s is not a supported function", funcType)
			}

			// Default to just support count and distinct for now. Redo this later.
			if selectNodeVal.AggDistinct {
				qp.Type = TypeCountDistinct
			} else {
				qp.Type = TypeCount
			}
		}

		if isDistrinct && qp.Type != TypeCountDistinct {
			qp.AggrPath = keyPath
			qp.Type = TypeDistinct
		}
	}

	// From clauses
	for _, fromNode := range statement.FromClause.Items {
		qp.From = append(qp.From, qp.repairString(*fromNode.(pgNodes.RangeVar).Relname))
	}

	// Create QueryKeys to be used by ProcessLine
	for _, query := range qp.Queries {
		qp.QueryKeys = append(qp.QueryKeys, query.KeyPath)
	}

	logger.Log.Debugf("Query Params: %s", spew.Sdump(qp))
	return &qp
}
