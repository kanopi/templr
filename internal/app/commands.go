// Package app implements the core templr CLI commands and application logic.
package app

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/araddon/dateparse"
	"github.com/beevik/etree"
	"github.com/dustin/go-humanize"
	"github.com/kanopi/templr/pkg/templr"
	"github.com/montanaflynn/stats"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
)

// SharedOptions contains flags common to all commands
type SharedOptions struct {
	Data           string
	Files          []string
	Sets           []string
	Strict         bool
	DryRun         bool
	Guard          string
	InjectGuard    bool
	DefaultMissing string
	NoColor        bool
	Debug          bool
	Ldelim         string
	Rdelim         string
	ExtraExts      []string
}

// WalkOptions contains options specific to walk mode
type WalkOptions struct {
	Shared SharedOptions
	Src    string
	Dst    string
}

// DirOptions contains options specific to directory mode
type DirOptions struct {
	Shared SharedOptions
	Dir    string
	In     string
	Out    string
}

// RenderOptions contains options specific to single-file render mode
type RenderOptions struct {
	Shared  SharedOptions
	In      string
	Out     string
	Helpers string
}

// SchemaOptions contains options for schema commands
type SchemaOptions struct {
	Shared          SharedOptions
	SchemaPath      string
	Mode            string
	Output          string
	Required        string
	AdditionalProps bool
	Format          string
}

// buildFuncMap creates the template function map with Sprig and custom functions.
// The returned function map includes a closure reference to tpl for the include function.
//
//nolint:gocyclo // Function map builders naturally have high complexity
func buildFuncMap(tpl **template.Template) template.FuncMap {
	funcs := sprig.TxtFuncMap()

	// Ensure YAML helpers exist (some environments/vendors strip these from Sprig)
	if _, ok := funcs["toYaml"]; !ok {
		funcs["toYaml"] = func(v any) (string, error) {
			b, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	if _, ok := funcs["fromYaml"]; !ok {
		funcs["fromYaml"] = func(s string) (map[string]any, error) {
			var m map[string]any
			if err := yaml.Unmarshal([]byte(s), &m); err != nil {
				return nil, err
			}
			return m, nil
		}
	}
	if _, ok := funcs["mustToYaml"]; !ok {
		funcs["mustToYaml"] = func(v any) string {
			b, err := yaml.Marshal(v)
			if err != nil {
				panic(err)
			}
			return string(b)
		}
	}
	if _, ok := funcs["mustFromYaml"]; !ok {
		funcs["mustFromYaml"] = func(s string) map[string]any {
			var m map[string]any
			if err := yaml.Unmarshal([]byte(s), &m); err != nil {
				panic(err)
			}
			return m
		}
	}

	// Helm-like helpers
	funcs["include"] = func(name string, data any) (string, error) {
		var b bytes.Buffer
		if *tpl == nil {
			return "", fmt.Errorf("template not initialized")
		}
		if err := (*tpl).ExecuteTemplate(&b, name, data); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	funcs["required"] = func(msg string, v any) (any, error) {
		switch x := v.(type) {
		case nil:
			return nil, errors.New(msg)
		case string:
			if strings.TrimSpace(x) == "" {
				return nil, errors.New(msg)
			}
		case []any:
			if len(x) == 0 {
				return nil, errors.New(msg)
			}
		case map[string]any:
			if len(x) == 0 {
				return nil, errors.New(msg)
			}
		}
		return v, nil
	}
	funcs["fail"] = func(msg string) (string, error) { return "", errors.New(msg) }

	// set: mutate a map with key=value and return it (useful for introducing new vars)
	funcs["set"] = func(m map[string]any, key string, val any) (map[string]any, error) {
		if m == nil {
			return nil, fmt.Errorf("set: target map is nil")
		}
		m[key] = val
		return m, nil
	}
	// setd: dotted-key set (e.g., "a.b.c")
	funcs["setd"] = func(m map[string]any, dotted string, val any) (map[string]any, error) {
		if m == nil {
			return nil, fmt.Errorf("setd: target map is nil")
		}
		setByDottedKey(m, dotted, val)
		return m, nil
	}
	// mergeDeep: deep-merge two maps (right wins); returns new merged map
	funcs["mergeDeep"] = func(a, b map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range a {
			out[k] = v
		}
		return deepMerge(out, b)
	}
	// safe: render value or fallback when missing/empty
	funcs["safe"] = func(v any, def string) string {
		if v == nil {
			return def
		}
		switch vv := v.(type) {
		case string:
			if strings.TrimSpace(vv) == "" {
				return def
			}
			return vv
		default:
			return fmt.Sprint(v)
		}
	}

	// Humanize functions
	funcs["humanizeBytes"] = func(size any) string {
		var bytes uint64
		switch v := size.(type) {
		case int:
			bytes = uint64(v)
		case int64:
			bytes = uint64(v)
		case uint64:
			bytes = v
		case float64:
			bytes = uint64(v)
		default:
			return fmt.Sprint(size)
		}
		return humanize.Bytes(bytes)
	}

	funcs["humanizeTime"] = func(t any) string {
		var timeVal time.Time
		switch v := t.(type) {
		case time.Time:
			timeVal = v
		case string:
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return v
			}
			timeVal = parsed
		default:
			return fmt.Sprint(t)
		}
		return humanize.Time(timeVal)
	}

	funcs["humanizeNumber"] = func(num any) string {
		switch v := num.(type) {
		case int:
			return humanize.Comma(int64(v))
		case int64:
			return humanize.Comma(v)
		case float64:
			return humanize.Commaf(v)
		default:
			return fmt.Sprint(num)
		}
	}

	funcs["ordinal"] = func(num any) string {
		var n int
		switch v := num.(type) {
		case int:
			n = v
		case int64:
			n = int(v)
		case float64:
			n = int(v)
		default:
			return fmt.Sprint(num)
		}
		return humanize.Ordinal(n)
	}

	// TOML functions
	funcs["toToml"] = func(v any) (string, error) {
		b, err := toml.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	funcs["fromToml"] = func(s string) (map[string]any, error) {
		var m map[string]any
		if err := toml.Unmarshal([]byte(s), &m); err != nil {
			return nil, err
		}
		return m, nil
	}

	// Path functions
	funcs["pathExt"] = func(path string) string {
		return filepath.Ext(path)
	}

	funcs["pathStem"] = func(path string) string {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}

	funcs["pathNormalize"] = func(path string) string {
		// Clean the path and convert to forward slashes for cross-platform consistency
		cleaned := filepath.Clean(path)
		return filepath.ToSlash(cleaned)
	}

	funcs["mimeType"] = func(path string) string {
		return detectMimeType(path)
	}

	// Validation functions
	funcs["isEmail"] = func(email string) bool {
		_, err := mail.ParseAddress(email)
		return err == nil
	}

	funcs["isURL"] = func(rawURL string) bool {
		u, err := url.Parse(rawURL)
		return err == nil && u.Scheme != "" && u.Host != ""
	}

	funcs["isIPv4"] = func(ip string) bool {
		parsed := net.ParseIP(ip)
		return parsed != nil && parsed.To4() != nil
	}

	funcs["isIPv6"] = func(ip string) bool {
		parsed := net.ParseIP(ip)
		return parsed != nil && parsed.To4() == nil
	}

	funcs["isUUID"] = func(uuid string) bool {
		uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
		return uuidRegex.MatchString(uuid)
	}

	// Advanced Base64 & Encoding functions
	funcs["base64url"] = func(data string) string {
		return base64.URLEncoding.EncodeToString([]byte(data))
	}

	funcs["base64urlDecode"] = func(encoded string) (string, error) {
		decoded, err := base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	funcs["base32"] = func(data string) string {
		return base32.StdEncoding.EncodeToString([]byte(data))
	}

	funcs["base32Decode"] = func(encoded string) (string, error) {
		decoded, err := base32.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	// CSV functions
	funcs["toCsv"] = func(data any) (string, error) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)

		switch v := data.(type) {
		case []map[string]any:
			if len(v) == 0 {
				return "", nil
			}
			// Get headers from first row
			var headers []string
			for k := range v[0] {
				headers = append(headers, k)
			}
			if err := w.Write(headers); err != nil {
				return "", err
			}
			// Write rows
			for _, row := range v {
				var record []string
				for _, h := range headers {
					record = append(record, fmt.Sprint(row[h]))
				}
				if err := w.Write(record); err != nil {
					return "", err
				}
			}
		case [][]string:
			for _, row := range v {
				if err := w.Write(row); err != nil {
					return "", err
				}
			}
		default:
			return "", fmt.Errorf("toCsv: unsupported type %T", data)
		}

		w.Flush()
		if err := w.Error(); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	funcs["fromCsv"] = func(csvData string) ([]map[string]string, error) {
		r := csv.NewReader(strings.NewReader(csvData))
		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return []map[string]string{}, nil
		}

		headers := records[0]
		var result []map[string]string

		for i := 1; i < len(records); i++ {
			row := make(map[string]string)
			for j, value := range records[i] {
				if j < len(headers) {
					row[headers[j]] = value
				}
			}
			result = append(result, row)
		}

		return result, nil
	}

	funcs["csvColumn"] = func(csvData, columnName string) ([]string, error) {
		r := csv.NewReader(strings.NewReader(csvData))
		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return []string{}, nil
		}

		headers := records[0]
		columnIndex := -1
		for i, h := range headers {
			if h == columnName {
				columnIndex = i
				break
			}
		}
		if columnIndex == -1 {
			return nil, fmt.Errorf("column %q not found", columnName)
		}

		var result []string
		for i := 1; i < len(records); i++ {
			if columnIndex < len(records[i]) {
				result = append(result, records[i][columnIndex])
			}
		}

		return result, nil
	}

	// Network utility functions
	funcs["cidrContains"] = func(ip, cidr string) (bool, error) {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return false, fmt.Errorf("invalid CIDR: %w", err)
		}
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return false, fmt.Errorf("invalid IP address: %s", ip)
		}
		return ipNet.Contains(parsedIP), nil
	}

	funcs["cidrHosts"] = func(cidr string) ([]string, error) {
		ip, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR: %w", err)
		}

		// Safety check: limit to reasonable sizes
		ones, bits := ipNet.Mask.Size()
		hostBits := bits - ones
		if hostBits > 10 { // Max 1024 hosts
			return nil, fmt.Errorf("CIDR range too large (max /22 for IPv4, /118 for IPv6)")
		}

		var ips []string
		for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
			ips = append(ips, ip.String())
		}

		// Remove network and broadcast addresses for IPv4
		if len(ips) > 2 && bits == 32 {
			return ips[1 : len(ips)-1], nil
		}
		return ips, nil
	}

	funcs["ipAdd"] = func(ip string, offset any) (string, error) {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return "", fmt.Errorf("invalid IP address: %s", ip)
		}

		var offsetInt int64
		switch v := offset.(type) {
		case int:
			offsetInt = int64(v)
		case int64:
			offsetInt = v
		case float64:
			offsetInt = int64(v)
		default:
			return "", fmt.Errorf("offset must be numeric, got %T", offset)
		}

		ipBigInt := new(big.Int).SetBytes(parsedIP)
		ipBigInt.Add(ipBigInt, big.NewInt(offsetInt))

		var newIP net.IP
		if parsedIP.To4() != nil {
			newIP = net.IP(ipBigInt.Bytes())
			// Ensure it's 4 bytes
			if len(newIP) > 4 {
				newIP = newIP[len(newIP)-4:]
			}
		} else {
			bytes := ipBigInt.Bytes()
			newIP = make(net.IP, 16)
			copy(newIP[16-len(bytes):], bytes)
		}

		return newIP.String(), nil
	}

	funcs["ipVersion"] = func(ip string) int {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return 0
		}
		if parsed.To4() != nil {
			return 4
		}
		return 6
	}

	funcs["ipPrivate"] = func(ip string) bool {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return false
		}
		return parsed.IsPrivate()
	}

	// Math and Statistics functions
	funcs["sum"] = func(numbers any) (float64, error) {
		floats, err := toFloat64Slice(numbers)
		if err != nil {
			return 0, err
		}
		return stats.Sum(floats)
	}

	funcs["avg"] = func(numbers any) (float64, error) {
		floats, err := toFloat64Slice(numbers)
		if err != nil {
			return 0, err
		}
		return stats.Mean(floats)
	}

	funcs["median"] = func(numbers any) (float64, error) {
		floats, err := toFloat64Slice(numbers)
		if err != nil {
			return 0, err
		}
		return stats.Median(floats)
	}

	funcs["stddev"] = func(numbers any) (float64, error) {
		floats, err := toFloat64Slice(numbers)
		if err != nil {
			return 0, err
		}
		return stats.StandardDeviation(floats)
	}

	funcs["percentile"] = func(numbers, p any) (float64, error) {
		floats, err := toFloat64Slice(numbers)
		if err != nil {
			return 0, err
		}

		var percentile float64
		switch v := p.(type) {
		case int:
			percentile = float64(v)
		case int64:
			percentile = float64(v)
		case float64:
			percentile = v
		default:
			return 0, fmt.Errorf("percentile must be numeric, got %T", p)
		}

		return stats.Percentile(floats, percentile)
	}

	funcs["clamp"] = func(value, minValue, maxValue any) (float64, error) {
		v, err := toFloat64(value)
		if err != nil {
			return 0, err
		}
		minVal, err := toFloat64(minValue)
		if err != nil {
			return 0, err
		}
		maxVal, err := toFloat64(maxValue)
		if err != nil {
			return 0, err
		}

		return math.Max(minVal, math.Min(maxVal, v)), nil
	}

	funcs["roundTo"] = func(value, decimals any) (float64, error) {
		v, err := toFloat64(value)
		if err != nil {
			return 0, err
		}

		var dec int
		switch d := decimals.(type) {
		case int:
			dec = d
		case int64:
			dec = int(d)
		case float64:
			dec = int(d)
		default:
			return 0, fmt.Errorf("decimals must be numeric, got %T", decimals)
		}

		multiplier := math.Pow(10, float64(dec))
		return math.Round(v*multiplier) / multiplier, nil
	}

	// Enhanced JSON Querying functions (Tier 3)
	funcs["jsonPath"] = func(jsonData, path string) (any, error) {
		result := gjson.Get(jsonData, path)
		if !result.Exists() {
			return nil, nil
		}
		return result.Value(), nil
	}

	funcs["jsonQuery"] = func(jsonData, path string) ([]any, error) {
		result := gjson.Get(jsonData, path)
		if !result.Exists() {
			return []any{}, nil
		}
		if result.IsArray() {
			var values []any
			for _, item := range result.Array() {
				values = append(values, item.Value())
			}
			return values, nil
		}
		return []any{result.Value()}, nil
	}

	funcs["jsonSet"] = func(jsonData, path string, value any) (string, error) {
		result, err := sjson.Set(jsonData, path, value)
		if err != nil {
			return "", err
		}
		return result, nil
	}

	// Advanced Date Parsing functions (Tier 3)
	funcs["dateParse"] = func(dateStr string) (time.Time, error) {
		return dateparse.ParseAny(dateStr)
	}

	funcs["dateAdd"] = func(dateStr, duration string) (time.Time, error) {
		t, err := dateparse.ParseAny(dateStr)
		if err != nil {
			return time.Time{}, err
		}

		// Parse duration string (supports "7 days", "2 weeks 3 days", etc.)
		d, err := parseDurationString(duration)
		if err != nil {
			return time.Time{}, err
		}

		return t.Add(d), nil
	}

	funcs["dateRange"] = func(startStr, endStr string) ([]time.Time, error) {
		start, err := dateparse.ParseAny(startStr)
		if err != nil {
			return nil, err
		}
		end, err := dateparse.ParseAny(endStr)
		if err != nil {
			return nil, err
		}

		var dates []time.Time
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
		return dates, nil
	}

	funcs["workdays"] = func(startStr, endStr string) (int, error) {
		start, err := dateparse.ParseAny(startStr)
		if err != nil {
			return 0, err
		}
		end, err := dateparse.ParseAny(endStr)
		if err != nil {
			return 0, err
		}

		count := 0
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			weekday := d.Weekday()
			if weekday != time.Saturday && weekday != time.Sunday {
				count++
			}
		}
		return count, nil
	}

	// XML Support functions (Tier 3)
	funcs["toXml"] = func(data any) (string, error) {
		doc := etree.NewDocument()
		doc.Indent(2)

		if err := buildXMLElement(doc.CreateElement("root"), data); err != nil {
			return "", err
		}

		var buf bytes.Buffer
		if _, err := doc.WriteTo(&buf); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	funcs["fromXml"] = func(xmlData string) (map[string]any, error) {
		doc := etree.NewDocument()
		if err := doc.ReadFromString(xmlData); err != nil {
			return nil, err
		}

		root := doc.Root()
		if root == nil {
			return map[string]any{}, nil
		}

		result := parseXMLElement(root)
		return map[string]any{root.Tag: result}, nil
	}

	return funcs
}

// Helper function to increment IP address
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Helper function to convert various types to float64
func toFloat64(val any) (float64, error) {
	switch v := val.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// Helper function to convert slice/array to []float64
func toFloat64Slice(val any) ([]float64, error) {
	switch v := val.(type) {
	case []float64:
		return v, nil
	case []int:
		result := make([]float64, len(v))
		for i, n := range v {
			result[i] = float64(n)
		}
		return result, nil
	case []int64:
		result := make([]float64, len(v))
		for i, n := range v {
			result[i] = float64(n)
		}
		return result, nil
	case []any:
		result := make([]float64, len(v))
		for i, item := range v {
			f, err := toFloat64(item)
			if err != nil {
				return nil, err
			}
			result[i] = f
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []float64", val)
	}
}

// Helper function to parse duration strings like "7 days", "2 weeks 3 days"
func parseDurationString(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Try standard Go duration first (e.g., "24h", "1h30m")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Parse human-friendly durations (e.g., "7 days", "2 weeks 3 days")
	var total time.Duration
	parts := strings.Fields(s)

	for i := 0; i < len(parts); i += 2 {
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("invalid duration format: %s", s)
		}

		value, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration: %s", parts[i])
		}

		unit := strings.ToLower(parts[i+1])
		unit = strings.TrimSuffix(unit, "s") // Handle both "day" and "days"

		switch unit {
		case "year":
			total += time.Duration(value * 365 * 24 * float64(time.Hour))
		case "month":
			total += time.Duration(value * 30 * 24 * float64(time.Hour))
		case "week":
			total += time.Duration(value * 7 * 24 * float64(time.Hour))
		case "day":
			total += time.Duration(value * 24 * float64(time.Hour))
		case "hour", "h":
			total += time.Duration(value * float64(time.Hour))
		case "minute", "min", "m":
			total += time.Duration(value * float64(time.Minute))
		case "second", "sec", "s":
			total += time.Duration(value * float64(time.Second))
		default:
			return 0, fmt.Errorf("unknown duration unit: %s", unit)
		}
	}

	return total, nil
}

// Helper function to build XML element from Go data
func buildXMLElement(elem *etree.Element, data any) error {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			child := elem.CreateElement(key)
			if err := buildXMLElement(child, val); err != nil {
				return err
			}
		}
	case []any:
		for i, item := range v {
			child := elem.CreateElement(fmt.Sprintf("item%d", i))
			if err := buildXMLElement(child, item); err != nil {
				return err
			}
		}
	case string:
		elem.SetText(v)
	case int, int64, float64, bool:
		elem.SetText(fmt.Sprintf("%v", v))
	case nil:
		// Empty element
	default:
		elem.SetText(fmt.Sprintf("%v", v))
	}
	return nil
}

// Helper function to parse XML element to Go data
func parseXMLElement(elem *etree.Element) any {
	// If element has no children, return text content
	if len(elem.ChildElements()) == 0 {
		text := elem.Text()
		if text == "" {
			return nil
		}
		return text
	}

	// If all children have the same tag, treat as array
	children := elem.ChildElements()
	if len(children) > 0 {
		firstTag := children[0].Tag
		allSame := true
		for _, child := range children {
			if child.Tag != firstTag {
				allSame = false
				break
			}
		}

		if allSame {
			var arr []any
			for _, child := range children {
				arr = append(arr, parseXMLElement(child))
			}
			return arr
		}
	}

	// Otherwise, treat as map
	result := make(map[string]any)
	for _, child := range children {
		result[child.Tag] = parseXMLElement(child)
	}
	return result
}

// buildValues constructs the values map from defaults, data files, and --set overrides
func buildValues(baseDir string, shared SharedOptions) (map[string]any, error) {
	debugSection(shared.Debug, "Value Loading Sequence")
	values := map[string]any{}

	// Load default values.yaml from baseDir if it exists
	debugf(shared.Debug, "Loading default values from %s", baseDir)
	def, err := loadDefaultValues(baseDir)
	if err != nil {
		return nil, fmt.Errorf("load default values: %w", err)
	}
	if len(def) > 0 {
		debugf(shared.Debug, "  → Loaded %d key(s) from default values.yaml", len(def))
		if shared.Debug {
			for k := range def {
				debugf(shared.Debug, "     - %s", k)
			}
		}
	} else {
		debugf(shared.Debug, "  → No default values.yaml found")
	}
	values = deepMerge(values, def)

	// Load --data file if specified
	if shared.Data != "" {
		debugf(shared.Debug, "Loading data from --data=%s", shared.Data)
		add, err := loadData(shared.Data)
		if err != nil {
			return nil, fmt.Errorf("load data: %w", err)
		}
		debugf(shared.Debug, "  → Loaded %d key(s)", len(add))
		if shared.Debug {
			for k := range add {
				debugf(shared.Debug, "     - %s", k)
			}
		}
		values = deepMerge(values, add)
	}

	// Load -f files
	for _, f := range shared.Files {
		debugf(shared.Debug, "Loading data from -f %s", f)
		add, err := loadData(f)
		if err != nil {
			return nil, fmt.Errorf("load -f %s: %w", f, err)
		}
		debugf(shared.Debug, "  → Loaded %d key(s)", len(add))
		if shared.Debug {
			for k := range add {
				debugf(shared.Debug, "     - %s", k)
			}
		}
		values = deepMerge(values, add)
	}

	// Apply --set overrides
	if len(shared.Sets) > 0 {
		debugf(shared.Debug, "Applying %d --set override(s)", len(shared.Sets))
	}
	for _, kv := range shared.Sets {
		idx := strings.Index(kv, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("--set expects key=value, got: %s", kv)
		}
		key := kv[:idx]
		val := parseScalar(kv[idx+1:])
		debugf(shared.Debug, "  → Setting %s = %v", key, val)
		setByDottedKey(values, key, val)
	}

	debugValues(shared.Debug, values, "Final Merged Values")

	return values, nil
}

// RunWalkMode executes walk mode: recursively render all templates in src to dst
func RunWalkMode(opts WalkOptions) error {
	if opts.Src == "" || opts.Dst == "" {
		return fmt.Errorf("-walk requires -src and -dst")
	}

	absSrc, _ := filepath.Abs(opts.Src)
	absDst, _ := filepath.Abs(opts.Dst)

	// Build values
	values, err := buildValues(absSrc, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: absSrc}

	// Create template with functions
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Parse ALL templates (so includes/partials are available)
	allowExts := buildAllowedExts(opts.Shared.ExtraExts)
	var names []string
	var sources map[string][]byte
	tpl, names, sources, err = readAllTplsIntoSet(tpl, absSrc, allowExts)
	if err != nil {
		return fmt.Errorf("parse tree: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}

	// Render each non-partial template; skip empty; enforce guard on overwrite
	for _, name := range names {
		if !shouldRender(name) {
			continue
		}
		relOut := trimAnyExt(name, allowExts)
		dstPath := filepath.Join(absDst, filepath.FromSlash(relOut))

		// render to buffer first
		outBytes, rerr := renderToBuffer(tpl, name, values)
		if rerr != nil {
			if opts.Shared.Strict {
				strictErrf(rerr, sources, opts.Shared.NoColor)
			}
			return fmt.Errorf("render error %s: %w", name, rerr)
		}
		// apply global default-missing replacement
		outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

		if isEmpty(outBytes) {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip empty %s (no file created)\n", dstPath)
			}
			continue
		}

		// Guard check BEFORE any mkdir/write
		ok, gerr := canOverwrite(dstPath, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", dstPath, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", dstPath)
			} else {
				warnf("guard", "skip (guard missing) %s", dstPath)
			}
			continue
		}

		if opts.Shared.DryRun {
			simulated := outBytes
			if opts.Shared.InjectGuard {
				simulated = injectGuardForExt(dstPath, simulated, opts.Shared.Guard)
				if !bytes.Equal(simulated, outBytes) {
					fmt.Printf("[dry-run] would inject guard into %s\n", dstPath)
				}
			}
			// Check if file would change
			same, _ := fastEqual(dstPath, simulated)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", dstPath)
			} else {
				fmt.Printf("[dry-run] would render %s -> %s (changed)\n", name, dstPath)
			}
			continue
		}

		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(dstPath, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(dstPath, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write %s: %w", dstPath, err)
		}
		if changed {
			fmt.Printf("rendered %s -> %s\n", name, dstPath)
		}
	}

	// Cleanup: remove empty directories under dst
	if err := templr.PruneEmptyDirs(absDst); err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	return nil
}

// RunDirMode executes directory mode: parse all templates in dir, execute one entry
//
//nolint:gocyclo,cyclop // orchestration function with inherent complexity
func RunDirMode(opts DirOptions) error {
	if opts.Dir == "" {
		return fmt.Errorf("--dir is required")
	}

	absDir, _ := filepath.Abs(opts.Dir)

	// Build values
	values, err := buildValues(absDir, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: absDir}

	// Create template with functions
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Parse all *.tpl in dir using path-based names
	allowExts := buildAllowedExts(opts.Shared.ExtraExts)
	var names []string
	var sources map[string][]byte
	tpl, names, sources, err = readAllTplsIntoSet(tpl, absDir, allowExts)
	if err != nil {
		return fmt.Errorf("parse dir templates: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}

	// Determine entry template name
	entryName := ""
	if opts.In != "" {
		// If -in is a file path, convert to rel name; otherwise assume it's already a template name.
		if info, err := os.Stat(opts.In); err == nil && !info.IsDir() {
			if rel, er := filepath.Rel(absDir, opts.In); er == nil {
				entryName = filepath.ToSlash(rel)
			} else {
				entryName = filepath.Base(opts.In)
			}
		} else {
			entryName = opts.In
		}
	} else if tpl.Lookup("root") != nil {
		entryName = "root"
	} else if len(names) > 0 {
		entryName = names[0]
	} else {
		return fmt.Errorf("no templates found in --dir")
	}

	// render to buffer
	outBytes, rerr := renderToBuffer(tpl, entryName, values)
	if rerr != nil {
		if opts.Shared.Strict {
			strictErrf(rerr, sources, opts.Shared.NoColor)
		}
		return rerr
	}
	// apply global default-missing replacement
	outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

	if isEmpty(outBytes) {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Shared.DryRun {
			fmt.Printf("[dry-run] skip empty render for entry %s -> %s\n", entryName, target)
			return nil
		}
		fmt.Fprintf(os.Stderr, "skipping empty render for entry %s -> %s\n", entryName, target)
		return nil
	}

	// If writing to a file, guard-verify when target exists
	if opts.Out != "" {
		ok, gerr := canOverwrite(opts.Out, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", opts.Out, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", opts.Out)
			} else {
				warnf("guard", "skip (guard missing) %s", opts.Out)
			}
			return nil
		}
	}

	if opts.Shared.DryRun {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Out != "" && opts.Shared.InjectGuard {
			simulated := injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			if !bytes.Equal(simulated, outBytes) {
				fmt.Printf("[dry-run] would inject guard into %s\n", opts.Out)
			}
		}
		// Check if file would change
		if opts.Out != "" {
			simToCheck := outBytes
			if opts.Shared.InjectGuard {
				simToCheck = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			}
			same, _ := fastEqual(opts.Out, simToCheck)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", opts.Out)
			} else {
				fmt.Printf("[dry-run] would render entry %s -> %s (changed)\n", entryName, target)
			}
		} else {
			fmt.Printf("[dry-run] would render entry %s -> %s\n", entryName, target)
		}
		return nil
	}

	// write (stdout or file)
	if opts.Out != "" {
		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(opts.Out, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write out: %w", err)
		}
		if changed {
			fmt.Printf("rendered entry %s -> %s\n", entryName, opts.Out)
		}
		return nil
	}

	if _, err := os.Stdout.Write(outBytes); err != nil {
		return err
	}
	return nil
}

// RunRenderMode executes single-file render mode
//
//nolint:gocyclo,cyclop // orchestration function with inherent complexity
func RunRenderMode(opts RenderOptions) error {
	debugSection(opts.Shared.Debug, "Template Rendering Flow")

	// Determine Files.Root (dir of -in if present)
	filesRoot := "."
	if opts.In != "" {
		if info, err := os.Stat(opts.In); err == nil && !info.IsDir() {
			if abs, e := filepath.Abs(opts.In); e == nil {
				filesRoot = filepath.Dir(abs)
			}
		}
	}
	debugf(opts.Shared.Debug, "Files.Root directory: %s", filesRoot)

	// Build values
	values, err := buildValues(filesRoot, opts.Shared)
	if err != nil {
		return err
	}

	// Add .Files API
	values["Files"] = FilesAPI{Root: filesRoot}
	debugf(opts.Shared.Debug, "Added .Files API with root: %s", filesRoot)

	// Create template with functions
	debugf(opts.Shared.Debug, "Creating template with delimiters: %s ... %s", opts.Shared.Ldelim, opts.Shared.Rdelim)
	if opts.Shared.Strict {
		debugf(opts.Shared.Debug, "Strict mode enabled (missingkey=error)")
	}
	var tpl *template.Template
	funcs := buildFuncMap(&tpl)
	tpl = template.New("root").Funcs(funcs).Option("missingkey=default")
	if opts.Shared.Strict {
		tpl = tpl.Option("missingkey=error")
	}
	tpl = tpl.Delims(opts.Shared.Ldelim, opts.Shared.Rdelim)

	// Read template source
	var srcBytes []byte
	sources := make(map[string][]byte)
	tplName := "stdin"
	if opts.In == "" {
		debugf(opts.Shared.Debug, "Reading template from stdin")
		srcBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		debugf(opts.Shared.Debug, "Reading template from file: %s", opts.In)
		srcBytes, err = os.ReadFile(opts.In)
		if err != nil {
			return fmt.Errorf("read template: %w", err)
		}
		tplName = filepath.Base(opts.In)
	}
	debugf(opts.Shared.Debug, "Main template: %s (%d bytes)", tplName, len(srcBytes))
	sources[tplName] = srcBytes
	sources["root"] = srcBytes // Also map to "root" since that's what template.Parse uses

	// Load sidecar helpers in the same directory based on -helpers glob (default: _helpers.tpl)
	if filesRoot != "" && filesRoot != "." && opts.Helpers != "" {
		pattern := filepath.Join(filesRoot, opts.Helpers)
		debugf(opts.Shared.Debug, "Looking for helper templates: %s", pattern)
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			debugf(opts.Shared.Debug, "Found %d helper template(s)", len(matches))
			for _, hp := range matches {
				if b, e := os.ReadFile(hp); e == nil {
					helperName := filepath.ToSlash(filepath.Base(hp))
					debugf(opts.Shared.Debug, "  → Loading helper: %s (%d bytes)", helperName, len(b))
					sources[helperName] = b
					if _, e2 := tpl.New(helperName).Parse(string(b)); e2 != nil {
						return fmt.Errorf("parse helper %s: %w", hp, e2)
					}
				}
			}
		} else {
			debugf(opts.Shared.Debug, "  → No helper templates found")
		}
	}

	debugf(opts.Shared.Debug, "Parsing main template")
	tpl, err = tpl.Parse(string(srcBytes))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Compute helper-driven variables (templr.vars)
	debugf(opts.Shared.Debug, "Checking for templr.vars template")
	if err := computeHelperVars(tpl, values); err != nil {
		return fmt.Errorf("helpers: %w", err)
	}
	if tpl.Lookup("templr.vars") != nil {
		debugf(opts.Shared.Debug, "  → templr.vars executed, values updated")
		if opts.Shared.Debug {
			debugValues(opts.Shared.Debug, values, "Values After templr.vars")
		}
	} else {
		debugf(opts.Shared.Debug, "  → No templr.vars template found")
	}

	// render to buffer
	debugf(opts.Shared.Debug, "Rendering template")
	outBytes, rerr := renderToBuffer(tpl, "", values)
	if rerr != nil {
		if opts.Shared.Strict {
			strictErrf(rerr, sources, opts.Shared.NoColor)
		}
		return rerr
	}
	debugf(opts.Shared.Debug, "Render complete (%d bytes)", len(outBytes))

	// apply global default-missing replacement
	outBytes = applyDefaultMissing(outBytes, opts.Shared.DefaultMissing)

	if isEmpty(outBytes) {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		if opts.Shared.DryRun {
			srcLabel := "stdin"
			if opts.In != "" {
				srcLabel = opts.In
			}
			fmt.Printf("[dry-run] skip empty render %s -> %s\n", srcLabel, target)
			return nil
		}
		fmt.Fprintf(os.Stderr, "skipping empty render -> %s\n", target)
		return nil
	}

	// If writing to a file, guard-verify when target exists
	if opts.Out != "" {
		ok, gerr := canOverwrite(opts.Out, opts.Shared.Guard)
		if gerr != nil && !os.IsNotExist(gerr) {
			return fmt.Errorf("guard check %s: %w", opts.Out, gerr)
		}
		if !ok {
			if opts.Shared.DryRun {
				fmt.Printf("[dry-run] skip (guard missing) %s\n", opts.Out)
				return nil
			}
			warnf("guard", "skip (guard missing) %s", opts.Out)
			return nil
		}
	}

	if opts.Shared.DryRun {
		target := "stdout"
		if opts.Out != "" {
			target = opts.Out
		}
		srcLabel := "stdin"
		if opts.In != "" {
			srcLabel = opts.In
		}
		if opts.Out != "" && opts.Shared.InjectGuard {
			simulated := injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			if !bytes.Equal(simulated, outBytes) {
				fmt.Printf("[dry-run] would inject guard into %s\n", opts.Out)
			}
		}
		// Check if file would change
		if opts.Out != "" {
			simToCheck := outBytes
			if opts.Shared.InjectGuard {
				simToCheck = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
			}
			same, _ := fastEqual(opts.Out, simToCheck)
			if same {
				fmt.Printf("[dry-run] would skip unchanged %s\n", opts.Out)
			} else {
				fmt.Printf("[dry-run] would render %s -> %s (changed)\n", srcLabel, target)
			}
		} else {
			fmt.Printf("[dry-run] would render %s -> %s\n", srcLabel, target)
		}
		return nil
	}

	// write (stdout or file)
	if opts.Out != "" {
		// Optionally inject guard comment
		if opts.Shared.InjectGuard {
			outBytes = injectGuardForExt(opts.Out, outBytes, opts.Shared.Guard)
		}
		// Write only if content changed
		changed, err := writeIfChanged(opts.Out, outBytes, 0o644)
		if err != nil {
			return fmt.Errorf("write out: %w", err)
		}
		if changed {
			srcLabel := "stdin"
			if opts.In != "" {
				srcLabel = opts.In
			}
			fmt.Printf("rendered %s -> %s\n", srcLabel, opts.Out)
		}
		return nil
	}

	if _, err := os.Stdout.Write(outBytes); err != nil {
		return err
	}
	return nil
}

// RunSchemaValidate validates data against a schema
func RunSchemaValidate(opts SchemaOptions, config *Config) error {
	// Load and merge data
	vals, err := buildValues(".", opts.Shared)
	if err != nil {
		return err
	}

	// Determine schema path
	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		// Try auto-discovery
		schemaPath = FindSchemaFile(config.Schema.Path)
		if schemaPath == "" {
			return fmt.Errorf("no schema file found (checked: %s, .templr.schema.yml, .templr/schema.yml)", config.Schema.Path)
		}
	}

	// Determine mode
	mode := opts.Mode
	if mode == "" {
		mode = config.Schema.Mode
	}
	if mode == "" {
		mode = "warn"
	}

	// Validate
	result, err := ValidateWithSchema(vals, schemaPath, mode)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Format and print errors
	if !result.Passed {
		output := FormatSchemaErrors(result, mode)
		fmt.Fprint(os.Stderr, output)

		if mode != "warn" {
			return fmt.Errorf("validation failed")
		}

		fmt.Printf("✓ Validation complete (%d warning%s)\n", len(result.Errors), pluralize(len(result.Errors)))
		return nil
	}

	fmt.Println("✓ Validation passed")
	return nil
}

// RunSchemaGenerate generates a schema from data
func RunSchemaGenerate(opts SchemaOptions, config *Config) error {
	// Load and merge data
	vals, err := buildValues(".", opts.Shared)
	if err != nil {
		return err
	}

	// Build generation config
	genConfig := config.Schema.Generate
	if opts.Required != "" {
		genConfig.Required = opts.Required
	}
	genConfig.AdditionalProps = opts.AdditionalProps

	// Generate schema
	schema, err := GenerateSchema(vals, genConfig)
	if err != nil {
		return fmt.Errorf("generate schema: %w", err)
	}

	// Marshal to YAML
	schemaBytes, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	// Write output
	if opts.Output != "" {
		if err := os.WriteFile(opts.Output, schemaBytes, 0o644); err != nil {
			return fmt.Errorf("write schema file: %w", err)
		}
		fmt.Printf("Generated schema -> %s\n", opts.Output)
	} else {
		// Print to stdout
		fmt.Print(string(schemaBytes))
	}

	return nil
}

// pluralize returns "s" if count is not 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// Debug logging helpers
func debugf(debug bool, format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func debugSection(debug bool, title string) {
	if debug {
		fmt.Fprint(os.Stderr, "\n"+strings.Repeat("=", 60)+"\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", title)
		fmt.Fprint(os.Stderr, strings.Repeat("=", 60)+"\n")
	}
}

func debugValues(debug bool, values map[string]any, title string) {
	if !debug {
		return
	}

	debugSection(debug, title)

	// Convert to YAML for pretty printing
	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Error marshaling values: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "%s\n", string(yamlBytes))
}
