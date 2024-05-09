package main

import (
	"bufio"
	"bytes"
	stdjson "encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// use jsoniner for handling JSON stuff
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// command-line flags
var (
	// colorize non-json input or not?
	markNonJson bool
	// separate json and non-json with extra newline
	sepOK bool
	// skip these keys in output (comma-separated)
	skipKeys    string
	skipKeyList []string
	// show only these keys in output (comma-separated)
	showOnlyKeys    string
	showOnlyKeyList []string
	// when only is specified show entries that have all fields present
	showOnlyGroup bool
	// key order (comma-separated), overrides keyOrder
	orderKeys string
	// skip post-processing
	skipPostProc bool
	// colorize all keys
	colorizeAllKeys bool
	// additionally colorize these keys
	colorizeKeys string
	// enable re-scan (will exit after first scanner.Scan exits)
	rescan bool
)

var (
	jsonStart        = []byte("{")
	jsonStartReplace = []byte(`{"`)
	reFileName       = regexp.MustCompile("^[^{]+{\"")
)

var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

var (
	// initial sort order done by these key names:
	keyOrder = []string{
		"level",
		"time",
		"ts",
		"file",
		"func",
		"method",
		"path",
		"status",
	}
)

var (
	// special key names whose values will get colorized (except level which
	// gets its own color already):
	colorizeKeyVals = map[string]color.Attribute{
		"msg":       color.FgHiYellow,
		"status":    color.FgHiYellow,
		"path":      color.FgHiYellow,
		"sql":       color.FgHiYellow,
		"sql_query": color.FgHiYellow,
		"params":    color.FgGreen,
		"error":     color.FgRed,
		"err":       color.FgRed,
	}
)

func orderSortKeys(keys []string, order []string) []string {
	// sort alphabetically
	sort.Strings(keys)
	// swap keys with order priority
	for io := len(order) - 1; io >= 0; io-- {
		for ik := range keys {
			// move key to front
			if io < len(keys) && keys[ik] == order[io] {
				// delete
				keys = append(keys[:ik], keys[ik+1:]...)
				// push to front
				keys = append([]string{order[io]}, keys...)
			}
		}
	}
	return keys
}

func keyColor(m KVMap) func(a ...interface{}) string {
	col := cyan
	if !colorizeAllKeys {
		return col
	}
	for key, val := range m {
		if key == "level" {
			val, ok := val.(string)
			if !ok {
				break
			}
			col = levelColor(val)
			break
		}
	}
	return col
}

func levelColor(lvl string) func(a ...interface{}) string {
	switch lvl {
	case "info":
		return cyan
	case "error":
		return red
	case "warning":
		return yellow
	default:
		return green
	}
}

func skip(key string) bool {
	for _, k := range skipKeyList {
		if k == key {
			return true
		}
	}
	return false
}

func showKey(key string) bool {
	for _, k := range showOnlyKeyList {
		if k == key {
			return true
		}
	}
	return false
}

// KVMap represents key=value map that the input JSON text will be
// serialized into
type KVMap map[string]interface{}

func (kvMap KVMap) String() string {
	var keys []string
	for key := range kvMap {
		keys = append(keys, key)
	}
	keys = orderSortKeys(keys, keyOrder)
	var output []string

	keyCol := keyColor(kvMap)
	for _, key := range keys {
		val := kvMap[key]
		if val == "" {
			continue
		}
		if val == nil {
			continue
		}
		if skip(key) {
			continue
		}
		if len(showOnlyKeyList) > 0 && !showKey(key) {
			continue
		}
		if key == "level" {
			switch val {
			case "info":
				output = append(output, fmt.Sprintf("%s=%s\n", cyan(key), green(val)))
			case "error":
				output = append(output, fmt.Sprintf("%s=%s\n", cyan(key), red(val)))
			default:
				output = append(output, fmt.Sprintf("%s=%s\n", cyan(key), yellow(val)))
			}
			continue
		}

		// override value color?
		if col, ok := colorizeKeyVals[key]; ok {
			valColor := color.New(col).SprintFunc()
			output = append(output, fmt.Sprintf("%s=%v\n", keyCol(key), valColor(kvMap[key])))
		} else {
			output = append(output, fmt.Sprintf("%s=%v\n", keyCol(key), kvMap[key]))
		}
	}
	return strings.Join(output, "")
}

func (kvMap KVMap) HasKeys(keys []string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		_, ok := kvMap[key]
		if !ok {
			return false
		}
	}
	return true
}

func json2kvmap(input []byte, into *KVMap) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&into); err != nil {
		return errors.Wrapf(err, "cannot Unmarshall input: %s", string(input))
	}
	return nil
}

func init() {
	flag.BoolVar(&markNonJson, "mark", false, "mark non-json input?")
	flag.BoolVar(&sepOK, "sep", false, "separate JSON and non-JSON")
	flag.StringVar(&skipKeys, "skip", "", "comma-separated list of keys to be skipped from output")
	flag.StringVar(&showOnlyKeys, "only", "", "comma-separated list of keys to be shown only and the rest skipped from output")
	flag.BoolVar(&showOnlyGroup, "group", false, "group entries that have all fields present when using -only")
	flag.StringVar(&orderKeys, "order", "", "comma-separated list of keys order")
	flag.BoolVar(&skipPostProc, "no-pp", false, "skip post-processing")
	flag.BoolVar(&colorizeAllKeys, "colorize", false, "colorize all keys")
	flag.StringVar(&colorizeKeys, "colorize-keys", "", "comma-separated list of additional keys to colorize")
	flag.BoolVar(&rescan, "rescan", false, "enable Scanner restart")
	flag.Parse()

	if skipKeys != "" {
		skipKeyList = strings.Split(skipKeys, ",")
	}

	if showOnlyKeys != "" {
		showOnlyKeyList = strings.Split(showOnlyKeys, ",")
	}

	if orderKeys != "" {
		keyOrder = strings.Split(orderKeys, ",")
	}

	if colorizeKeys != "" {
		for _, key := range strings.Split(colorizeKeys, ",") {
			colorizeKeyVals[key] = color.FgHiYellow
		}
	}
}

func main() {
	for {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			// does line start with file name (tailing multiple files)
			if reFileName.Match(line) {
				line = reFileName.ReplaceAll(line, jsonStartReplace)
			}
			// if not, does it start with jsonStart
			if !bytes.HasPrefix(line, jsonStart) {
				sep := "\n"
				if sepOK {
					sep = "\n\n"
				}
				if markNonJson {
					fmt.Printf("%s\n%v%s", yellow("[not json]"), string(line), sep)
				} else {
					fmt.Printf("%s%s", string(line), sep)
				}
				continue
			}
			kvMap := make(KVMap)
			if err := json2kvmap(line, &kvMap); err != nil {
				continue
			}
			if !skipPostProc {
				postprocess(kvMap)
			}

			if showOnlyGroup && !kvMap.HasKeys(showOnlyKeyList) {
				continue
			}

			// everything OK here - print keyval
			if len(kvMap) > 0 {
				fmt.Printf("%s\n", kvMap)
			}
		}

		// continue scanning after pause (file truncation)
		if rescan {
			time.Sleep(100 * time.Millisecond)
		} else {
			break
		}
	}
}

func postprocess(m KVMap) {
	mergeQueryAndParams(m)
	addInsertParamMap(m)
	mergeFileAndFunc(m)
	convertUnixTimestamp(m)
}

// convertUnixTimestamp converts unix timestamp into readable date-time format
// Example: 1715264548.7267861 -> 2024-05-09 16:22:28.007267861 +0200 CEST
func convertUnixTimestamp(m KVMap) {
	ts, ok := m["ts"]
	if !ok {
		return
	}
	if ts, ok := ts.(stdjson.Number); ok {
		ts := string(ts)
		dot := strings.Index(ts, ".")
		if dot > 0 && dot < len(ts) {
			sec, e1 := strconv.ParseInt(ts[:dot], 10, 64)
			nsec, e2 := strconv.ParseInt(ts[dot+1:], 10, 64)
			if e1 == nil && e2 == nil {
				timestamp := time.Unix(sec, nsec)
				m["ts"] = timestamp
			}
		}
	}
}

// mergeQueryAndParams extracts `sql_query` and `params` key values and merges
// them into a single key `sql`, effectively creating a single SQL INSERT query
// with all values in place.
func mergeQueryAndParams(m KVMap) {
	sqlQuery, okq := m["sql_query"]
	sqlParams, okp := m["params"]

	if !(okq && okp) {
		return
	}

	if sqlParams == nil {
		return
	}

	sql := sqlQuery.(string)
	params := sqlParams.([]interface{})

	if len(params) == 0 {
		return
	}

	var replaced int
	for idx, param := range params {
		place := fmt.Sprintf("$%d", idx+1)
		val := fmt.Sprintf("'%v'", param)
		sql = strings.Replace(sql, place, val, 1)
		replaced++
	}

	if replaced == len(params) {
		delete(m, "sql_query")
		delete(m, "params")
		m["sql"] = sql
	}
}

// mergeFileAndFunc merges `file` and `func` entries into one `func` entry and
// deletes `file` entry.
func mergeFileAndFunc(m KVMap) {
	filePath, ok1 := m["file"]
	funcName, ok2 := m["func"]

	if !(ok1 && ok2) {
		return
	}

	m["func"] = fmt.Sprintf("%s (%s)", filePath, funcName)
	delete(m, "file")
}

// addInsertParamMap adds `sql_insert_map` to log which consists of columns and
// values found in SQL INSERT query key `sql`.
// Run this *after* mergeQueryAndParams() function.
func addInsertParamMap(m KVMap) {
	sqlVal, ok := m["sql"]
	if !ok {
		return
	}

	sqlQuery, ok := sqlVal.(string)
	if !ok {
		return
	}

	parsed, err := parseInsert(sqlQuery)
	if err != nil {
		return
	}

	if len(parsed.Values) != len(parsed.Columns) {
		return
	}

	var params []string
	for i := 0; i < len(parsed.Values); i++ {
		key := parsed.Columns[i]
		val := parsed.Values[i]
		params = append(params, fmt.Sprintf("%s=%s", green(key), val))
	}

	m["sql_insert_map"] = strings.Join(params, " ")
}

// SQL insert query consists of:
// INSERT INTO "table" ("c1", "c2", ..., "cn")
// VALUES ('v1', v2, ..., NULL, ..., true, ...)
// RETURNING "table"."col"
// which will be matched into:
//
//	1: table name
//	2: columns
//	3: values
//	4: return table
//	5: return column
var reSqlInsert = regexp.MustCompile(
	`INSERT INTO "([^\"]+)" \(([^\)]+)\) VALUES \(([^\)]+)\) RETURNING "([^\"]+)"."([^\"]+)"`)

var reComma = regexp.MustCompile(`,`)

// total number of significant matches
var expectMatches = 6

var ErrInvalidMatchCount = errors.New("Invalid match count.")

// Insert represents information about parsed SQL query.
type Insert struct {
	// Table name to get inserted data.
	Table string
	// Values to be inserted.
	Values []string
	// Column names.
	Columns []string
	// Returning Table name.
	RetTable string
	// RetTable column name.
	RetColumn string
}

// parseInsert parses Insert SQL query and returns parse information.
func parseInsert(sql string) (Insert, error) {
	matches := reSqlInsert.FindStringSubmatch(sql)
	if len(matches) != expectMatches {
		return Insert{}, ErrInvalidMatchCount
	}
	insert := Insert{
		Table:     matches[1],
		Columns:   parseValues(matches[2]),
		Values:    parseQuotedValues(matches[3]),
		RetTable:  matches[4],
		RetColumn: matches[5],
	}

	return insert, nil
}

// parseQuotedValues splits quoted value string into list of strings.
// Value string contains all column values in the form:
// `'val1', 'val2', ..., 'val_n'` where a single val can be empty.
// Example: ”,'02a56888-ea30-11eb-b3e9-1f5878e115ac','73553','false','Dolní Lutyně, 73553, Stará cesta 1014','<nil>','<nil>','<nil>'
func parseQuotedValues(str string) []string {
	var start = []rune("'")[0]
	var values []string
	var begin bool
	var token string
	for _, char := range str {
		if char == start {
			if begin {
				values = append(values, token)
				begin = false
				token = ""
				continue
			}
			begin = true
			continue
		}
		if begin {
			token += string(char)
		}
	}
	return values
}

func parseValues(str string) []string {
	vals := reComma.Split(str, -1)
	res := []string{}
	for _, val := range vals {
		val = strings.TrimPrefix(val, `"`)
		val = strings.TrimSuffix(val, `"`)
		val = strings.TrimPrefix(val, `'`)
		val = strings.TrimSuffix(val, `'`)
		res = append(res, val)
	}
	return res
}
