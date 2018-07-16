package config

import (
	"strings"
	"strconv"
	"encoding/json"
	"fmt"
	"os"
)

// Get config value by key string, support get sub-value by key path(eg. 'map.key'),
// ok is true, find value from config
// ok is false, not found or error
func (c *Config) Get(key string, findByPath ...bool) (value interface{}, ok bool) {
	key = strings.Trim(strings.TrimSpace(key), ".")
	if key == "" {
		return
	}

	// if not is readonly
	if !c.opts.Readonly {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	// is top key
	if value, ok = c.data[key]; ok {
		return
	}

	// disable find by path.
	if len(findByPath) > 0 && !findByPath[0] {
		return
	}

	// has sub key? eg. "lang.dir"
	if !strings.Contains(key, ".") {
		return
	}

	keys := strings.Split(key, ".")
	topK := keys[0]

	// find top item data based on top key
	var item interface{}
	if item, ok = c.data[topK]; !ok {
		return
	}

	// find child
	for _, k := range keys[1:] {
		switch item.(type) {
		case map[string]interface{}: // is map(decode from toml/json)
			item, ok = item.(map[string]interface{})[k]
			if !ok {
				return
			}
		case map[interface{}]interface{}: // is map(decode from yaml)
			item, ok = item.(map[interface{}]interface{})[k]
			if !ok {
				return
			}
		case []interface{}: // is array
			i, err := strconv.Atoi(k)
			if err != nil {
				return
			}

			// 检查slice index是否存在
			arrItem := item.([]interface{})
			if len(arrItem) < i {
				return
			}

			item = arrItem[i]
		default: // error
			ok = false
			return
		}
	}

	return item, true
}

/*************************************************************
 * config get(basic data type)
 *************************************************************/

// GetString
func (c *Config) GetString(key string) (value string, ok bool) {
	val, ok := c.Get(key)
	if !ok {
		return
	}

	switch val.(type) {
	case bool, int, int64:
		value = fmt.Sprintf("%v", val)
	case string:
		value = fmt.Sprintf("%v", val)

		// if opts.ParseEnv is true
		if c.opts.ParseEnv && strings.Index(value, "${") == 0 {
			var name, def string
			str := strings.Trim(strings.TrimSpace(value), "${}")
			ss := strings.SplitN(str, "|", 2)

			// ${NotExist|defValue}
			if len(ss) == 2 {
				name, def = strings.TrimSpace(ss[0]), strings.TrimSpace(ss[1])
				// ${SHELL}
			} else {
				name = ss[0]
			}

			value = os.Getenv(name)
			if value == "" {
				value = def
			}
		}
	default:
		ok = false
	}

	return
}

// DefString get a string value, if not found return default value
func (c *Config) DefString(key string, def string) string {
	if value, ok := c.GetString(key); ok {
		return value
	}

	return def
}

// GetInt
func (c *Config) GetInt(key string) (value int, ok bool) {
	rawVal, ok := c.GetString(key)
	if !ok {
		return
	}

	if value, err := strconv.Atoi(rawVal); err == nil {
		return value, true
	}

	return
}

// DefInt get a int value, if not found return default value
func (c *Config) DefInt(key string, def int) int {
	if value, ok := c.GetInt(key); ok {
		return value
	}

	return def
}

// GetInt64
func (c *Config) GetInt64(key string) (value int64, ok bool) {
	if intVal, ok := c.GetInt(key); ok {
		value = int64(intVal)
	}

	return
}

// DefInt64
func (c *Config) DefInt64(key string, def int64) int64 {
	if intVal, ok := c.GetInt(key); ok {
		return int64(intVal)
	}

	return def
}

// GetBool Looks up a value for a key in this section and attempts to parse that value as a boolean,
// along with a boolean result similar to a map lookup.
// of following(case insensitive):
//  - true
//  - yes
//  - false
//  - no
//  - 1
//  - 0
// The `ok` boolean will be false in the event that the value could not be parsed as a bool
func (c *Config) GetBool(key string) (value bool, ok bool) {
	rawVal, ok := c.GetString(key)
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
		ok = false
	}

	return
}

// DefBool get a bool value, if not found return default value
func (c *Config) DefBool(key string, def bool) bool {
	if value, ok := c.GetBool(key); ok {
		return value
	}

	return def
}

/*************************************************************
 * config get(complex data type)
 *************************************************************/

// GetIntArr get config data as a int slice/array
func (c *Config) GetIntArr(key string) (arr []int, ok bool) {
	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch rawVal.(type) {
	case []interface{}:
		for _, v := range rawVal.([]interface{}) {
			// iv, err := strconv.Atoi(v.(string))
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				return
			}

			arr = append(arr, iv)
		}
	default:
		ok = false
	}

	return
}

// GetIntMap get config data as a map[string]int
func (c *Config) GetIntMap(key string) (mp map[string]int, ok bool) {
	rawVal, ok := c.Get(key)
	if !ok {
		return
	}

	switch rawVal.(type) {
	case map[string]interface{}: // decode from json,toml
		mp = make(map[string]int)
		for k, v := range rawVal.(map[string]interface{}) {
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				return
			}
			mp[k] = iv
		}
	case map[interface{}]interface{}: // if decode from yaml
		mp = make(map[string]int)
		for k, v := range rawVal.(map[interface{}]interface{}) {
			iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				ok = false
				return
			}

			sk := fmt.Sprintf("%v", k)
			mp[sk] = iv
		}
	default:
		ok = false
	}

	return
}

// GetStringArr  get config data as a string slice/array
func (c *Config) GetStringArr(key string) (arr []string, ok bool) {
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

	switch rawVal.(type) {
	case []interface{}:
		for _, v := range rawVal.([]interface{}) {
			arr = append(arr, fmt.Sprintf("%v", v))
		}
	default:
		ok = false
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

// GetStringMap get config data as a map[string]string
func (c *Config) GetStringMap(key string) (mp map[string]string, ok bool) {
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

	switch rawVal.(type) {
	case map[string]interface{}: // decode from json,toml
		mp = make(map[string]string)
		for k, v := range rawVal.(map[string]interface{}) {
			mp[k] = fmt.Sprintf("%v", v)
		}
	case map[interface{}]interface{}: // if decode from yaml
		mp = make(map[string]string)
		for k, v := range rawVal.(map[interface{}]interface{}) {
			sk := fmt.Sprintf("%v", k)
			mp[sk] = fmt.Sprintf("%v", v)
		}
	default:
		ok = false
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

// MapStructure alias method of the 'GetStructure'
func (c *Config) MapStructure(key string, v interface{}) (err error) {
	return c.GetStructure(key, v)
}

// GetStructure get config data and map to a structure.
// usage:
// 	dbInfo := Db{}
// 	config.GetStructure("db", &dbInfo)
func (c *Config) GetStructure(key string, v interface{}) (err error) {
	if rawVal, ok := c.Get(key); ok {
		blob, err := json.Marshal(rawVal)
		if err != nil {
			return err
		}

		err = json.Unmarshal(blob, v)
	}

	return
}