package parser

import "sort"

// Distinct executes a DISTINCT() query over log results.
// SELECT DISTINCT(line.cmd) FROM testapp WHERE date > '2016-10-05'
func (tp *TidalwaveParser) Distinct() *[]string {
	keys := []string{}
	for key := range *tp.CountDistinct() {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return &keys
}
