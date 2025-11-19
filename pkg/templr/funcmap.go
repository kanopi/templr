// Package templr provides template function map builders.
package templr

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/mail"
	"net/url"
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
	"github.com/montanaflynn/stats"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
)

// BuildFuncMap creates the template function map with Sprig and custom functions.
// The returned function map includes a closure reference to tpl for the include function.
// The tpl parameter is a pointer-to-pointer so that the include function can access
// the template even when it's initialized after the func map is created.
//
//nolint:gocyclo // Function map builders naturally have high complexity
func BuildFuncMap(tpl **template.Template) template.FuncMap {
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
		if tpl == nil || *tpl == nil {
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

	// Enhanced JSON Querying functions
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

	// Advanced Date Parsing functions
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

	// XML Support functions
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

// Helper functions

// incIP increments an IP address
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// toFloat64 converts various types to float64
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

// toFloat64Slice converts slice/array to []float64
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

// parseDurationString parses duration strings like "7 days", "2 weeks 3 days"
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

// buildXMLElement builds XML element from Go data
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

// parseXMLElement parses XML element to Go data
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

// deepMerge performs deep merge of two maps (right wins)
func deepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		if dm, ok := dst[k].(map[string]any); ok {
			if sm, ok := v.(map[string]any); ok {
				dst[k] = deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

// setByDottedKey assigns val into m using a dotted path (e.g., "a.b.c")
func setByDottedKey(m map[string]any, dotted string, val any) {
	parts := strings.Split(dotted, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = val
			return
		}
		next, ok := cur[p]
		if !ok {
			nm := map[string]any{}
			cur[p] = nm
			cur = nm
			continue
		}
		nmm, ok := next.(map[string]any)
		if !ok {
			nmm = map[string]any{}
			cur[p] = nmm
		}
		cur = nmm
	}
}

// detectMimeType detects MIME type from file extension
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}
