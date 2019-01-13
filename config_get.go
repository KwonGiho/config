package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	errInvalidKey = errors.New("invalid config key string")
	// errNotFound   = errors.New("this key does not exist in the configuration data")
)

// Get config value by key string, support get sub-value by key path(eg. 'map.key'),
// ok is true, find value from config
// ok is false, not found or error
func (c *Config) Get(key string, findByPath ...bool) (value interface{}, ok bool) {
	key = formatKey(key)
	if key == "" {
		c.addError(errInvalidKey)
		return
	}

	// if not is readonly
	if !c.opts.Readonly {
		c.lock.RLock()
		defer c.lock.RUnlock()
	}

	// is top key
	if value, ok = c.data[key]; ok {
		return
	}

	// disable find by path.
	if len(findByPath) > 0 && !findByPath[0] {
		// c.addError(errNotFound)
		return
	}

	// has sub key? eg. "lang.dir"
	if !strings.Contains(key, ".") {
		// c.addError(errNotFound)
		return
	}

	keys := strings.Split(key, ".")
	topK := keys[0]

	// find top item data based on top key
	var item interface{}
	if item, ok = c.data[topK]; !ok {
		// c.addError(errNotFound)
		return
	}

	// find child
	// NOTICE: don't merge case, will result in an error.
	// e.g. case []int, []string
	// OR
	// case []int:
	// case []string:
	for _, k := range keys[1:] {
		switch typeData := item.(type) {
		case map[string]string: // is map(from Set)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[string]interface{}: // is map(decode from toml/json)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[interface{}]interface{}: // is map(decode from yaml)
			if item, ok = typeData[k]; !ok {
				return
			}
		case []int: // is array(is from Set)
			i, err := strconv.Atoi(k)

			// check slice index
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		case []string: // is array(is from Set)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		case []interface{}: // is array(load from file)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		default: // error
			ok = false
			c.addErrorf("cannot get value of the key '%s'", key)
			return
		}
	}

	return item, true
}

/*************************************************************
 * config get(basic data type)
 *************************************************************/

// String get a string by key
func (c *Config) String(key string) (value string, ok bool) {
	// find from cache
	if c.opts.EnableCache && len(c.strCache) > 0 {
		value, ok = c.strCache[key]
		if ok {
			return
		}
	}

	val, ok := c.Get(key)
	if !ok {
		return
	}

	switch val.(type) {
	// from json int always is float64
	case bool, int, uint, int8, uint8, int16, uint16, int32, uint64, int64, float32, float64:
		value = fmt.Sprintf("%v", val)
	case string:
		value = fmt.Sprintf("%v", val)

		// if opts.ParseEnv is true
		if c.opts.ParseEnv {
			value = c.parseEnvValue(value)
		}
	default:
		c.addErrorf("value cannot be convert to string of the key '%s'", key)
		ok = false
	}

	// add cache
	if ok && c.opts.EnableCache {
		if c.strCache == nil {
			c.strCache = make(map[string]string)
		}

		c.strCache[key] = value
	}
	return
}

// DefString get a string value, if not found return default value
func (c *Config) DefString(key string, defVal ...string) string {
	if value, ok := c.String(key); ok {
		return value
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return ""
}

// MustString get a string value, if not found return empty string
func (c *Config) MustString(key string) string {
	return c.DefString(key, "")
}

// Int get a int by key
func (c *Config) Int(key string) (value int, ok bool) {
	rawVal, ok := c.String(key)
	if !ok {
		return
	}

	if value, err := strconv.Atoi(rawVal); err == nil {
		c.addError(err)
		return value, true
	}

	ok = false
	return
}

// DefInt get a int value, if not found return default value
func (c *Config) DefInt(key string, defVal ...int) int {
	if value, ok := c.Int(key); ok {
		return value
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return 0
}

// MustInt get a int value, if not found return 0
func (c *Config) MustInt(key string) int {
	return c.DefInt(key, 0)
}

// Int64 get a int64 by key
func (c *Config) Int64(key string) (value int64, ok bool) {
	if intVal, ok := c.Int(key); ok {
		return int64(intVal), true
	}
	return
}

// DefInt64 get a int64 with a default value
func (c *Config) DefInt64(key string, defVal ...int64) int64 {
	if intVal, ok := c.Int(key); ok {
		return int64(intVal)
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return 0
}

// MustInt64 get a int value, if not found return 0
func (c *Config) MustInt64(key string) int64 {
	return c.DefInt64(key, 0)
}

// Bool looks up a value for a key in this section and attempts to parse that value as a boolean,
// along with a boolean result similar to a map lookup.
// of following(case insensitive):
//  - true
//  - yes
//  - false
//  - no
//  - 1
//  - 0
// The `ok` boolean will be false in the event that the value could not be parsed as a bool
func (c *Config) Bool(key string) (value bool, ok bool) {
	rawVal, ok := c.String(key)
	if !ok {
		return
	}

	lowerCase := strings.ToLower(rawVal)
	switch lowerCase {
	case "", "0", "false", "no":
		value = false
	case "1", "true", "yes":
		value = true
	default:
		c.addErrorf("the value '%s' cannot be convert to bool", lowerCase)
		ok = false
	}

	return
}

// DefBool get a bool value, if not found return default value
func (c *Config) DefBool(key string, defVal ...bool) bool {
	if value, ok := c.Bool(key); ok {
		return value
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return false
}

// MustBool get a string value, if not found return false
func (c *Config) MustBool(key string) bool {
	return c.DefBool(key, false)
}

// Float get a float64 by key
func (c *Config) Float(key string) (value float64, ok bool) {
	str, ok := c.String(key)
	if !ok {
		return
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		c.addError(err)
		ok = false
	}
	return
}

// DefFloat get a float value, if not found return default value
func (c *Config) DefFloat(key string, defVal ...float64) float64 {
	if value, ok := c.Float(key); ok {
		return value
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	return 0
}

/*************************************************************
 * config get(complex data type)
 *************************************************************/

// Ints get config data as a int slice/array
func (c *Config) Ints(key string) (arr []int, ok bool) {
	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case []int:
		arr = typeData
	case []interface{}:
		for _, v := range typeData {
			// iv, err := strconv.Atoi(v.(string))
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				c.addError(err)
				return
			}

			arr = append(arr, iv)
		}
	default:
		c.addErrorf("value cannot be convert to []int, key is '%s'", key)
		ok = false
	}
	return
}

// IntMap get config data as a map[string]int
func (c *Config) IntMap(key string) (mp map[string]int, ok bool) {
	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case map[string]int: // from Set
		mp = typeData
	case map[string]interface{}: // decode from json,toml
		mp = make(map[string]int)
		for k, v := range typeData {
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				c.addError(err)
				return
			}
			mp[k] = iv
		}
	case map[interface{}]interface{}: // if decode from yaml
		mp = make(map[string]int)
		for k, v := range typeData {
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				c.addError(err)
				return
			}

			sk := fmt.Sprintf("%v", k)
			mp[sk] = iv
		}
	default:
		ok = false
		c.addErrorf("value cannot be convert to map[string]int, key is '%s'", key)
	}
	return
}

// Strings get config data as a string slice/array
func (c *Config) Strings(key string) (arr []string, ok bool) {
	// find from cache
	if c.opts.EnableCache && len(c.sArrCache) > 0 {
		arr, ok = c.sArrCache[key]
		if ok {
			return
		}
	}

	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case []string:
		arr = typeData
	case []interface{}:
		for _, v := range typeData {
			arr = append(arr, fmt.Sprintf("%v", v))
		}
	default:
		ok = false
		c.addErrorf("value cannot be convert to []string, key is '%s'", key)
	}

	// add cache
	if ok && c.opts.EnableCache {
		if c.sArrCache == nil {
			c.sArrCache = make(map[string]strArr)
		}

		c.sArrCache[key] = arr
	}
	return
}

// StringMap get config data as a map[string]string
func (c *Config) StringMap(key string) (mp map[string]string, ok bool) {
	// find from cache
	if c.opts.EnableCache && len(c.sMapCache) > 0 {
		mp, ok = c.sMapCache[key]
		if ok {
			return
		}
	}

	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case map[string]string: // from Set
		mp = typeData
	case map[string]interface{}: // decode from json,toml
		mp = make(map[string]string)
		for k, v := range typeData {
			mp[k] = fmt.Sprintf("%v", v)
		}
	case map[interface{}]interface{}: // if decode from yaml
		mp = make(map[string]string)
		for k, v := range typeData {
			sk := fmt.Sprintf("%v", k)
			mp[sk] = fmt.Sprintf("%v", v)
		}
	default:
		ok = false
		c.addErrorf("value cannot be convert to map[string]string, key is '%s'", key)
	}

	// add cache
	if ok && c.opts.EnableCache {
		if c.sMapCache == nil {
			c.sMapCache = make(map[string]strMap)
		}

		c.sMapCache[key] = mp
	}
	return
}

// MapStruct alias method of the 'Structure'
func (c *Config) MapStruct(key string, v interface{}) (err error) {
	return c.Structure(key, v)
}

// MapStructure alias method of the 'Structure'
func (c *Config) MapStructure(key string, v interface{}) (err error) {
	return c.Structure(key, v)
}

// Structure get config data and map to a structure.
// usage:
// 	dbInfo := Db{}
// 	config.Structure("db", &dbInfo)
func (c *Config) Structure(key string, v interface{}) (err error) {
	var ok bool
	var data interface{}

	// map all data
	if key == "" {
		ok = true
		data = c.data
	} else {
		data, ok = c.Get(key)
	}

	if ok {
		blob, err := JSONEncoder(data)
		if err != nil {
			return err
		}

		err = JSONDecoder(blob, v)
	}
	return
}

// IsEmpty of the config
func (c *Config) IsEmpty() bool {
	return len(c.data) == 0
}

// parse env value, eg: "${SHELL}" ${NotExist|defValue}
var envRegex = regexp.MustCompile(`\${([\w-| ]+)}`)

// parse Env Value
func (c *Config) parseEnvValue(val string) string {
	if strings.Index(val, "${") == -1 {
		return val
	}

	// nodes like: ${VAR} -> [${VAR}]
	// val = "${GOPATH}/${APP_ENV | prod}/dir" -> [${GOPATH} ${APP_ENV | prod}]
	vars := envRegex.FindAllString(val, -1)
	if len(vars) == 0 {
		return val
	}

	var oldNew []string
	var name, def string
	for _, fVar := range vars {
		ss := strings.SplitN(fVar[2:len(fVar)-1], "|", 2)

		// has default ${NotExist|defValue}
		if len(ss) == 2 {
			name, def = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
		} else {
			def = fVar
			name = ss[0]
		}

		envVal := os.Getenv(name)
		if envVal == "" {
			envVal = def
		}

		oldNew = append(oldNew, fVar, envVal)
	}

	return strings.NewReplacer(oldNew...).Replace(val)
}

// format key
func formatKey(key string) string {
	return strings.Trim(strings.TrimSpace(key), ".")
}
