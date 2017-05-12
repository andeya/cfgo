// Cfgo from the YAML document, bi-directional synchronous multi-module configuration.
//
// The structure of the generated document will reflect the structure of the value itself.
// Maps and pointers (to struct, string, int, etc) are accepted as the in value.
//
// Struct fields are only unmarshalled if they are exported (have an upper case
// first letter), and are unmarshalled using the field name lowercased as the
// default key. Custom keys may be defined via the "yaml" name in the field
// tag: the content preceding the first comma is used as the key, and the
// following comma-separated options are used to tweak the marshalling process.
// Conflicting names result in a runtime error.
//
// The field tag format accepted is:
//
//     `(...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The following flags are currently supported:
//
//     omitempty    Only include the field if it's not set to the zero
//                  value for the type or to empty slices or maps.
//                  Does not apply to zero valued structs.
//
//     flow         Marshal using a flow style (useful for structs,
//                  sequences and maps).
//
//     inline       Inline the field, which must be a struct or a map,
//                  causing all of its fields or keys to be processed as if
//                  they were part of the outer struct. For maps, keys must
//                  not conflict with the yaml keys of other struct fields.
//
// In addition, if the key is `-`, the field is ignored.
//
package cfgo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/yaml.v2"
)

var (
	Default   = MustGet("config/config.yaml")
	logchan   = make(chan *Msg, 100)
	getOnce   sync.Once
	hasLogger bool
)

func GetLogchan() (c <-chan *Msg, ok bool) {
	getOnce.Do(func() {
		c = logchan
		ok = true
		hasLogger = true
	})
	return
}

type Msg struct {
	Ok  bool
	Txt string
}

func init() {
	go func() {
		chSignal := make(chan os.Signal)
		defer signal.Stop(chSignal)
		signal.Notify(chSignal, syscall.SIGUSR1)
		for {
			<-chSignal
			err := ReloadAll()
			var msg = new(Msg)
			if err != nil {
				msg.Ok = false
				msg.Txt = "reload config: " + err.Error()
			} else {
				msg.Ok = true
				msg.Txt = "reload config ok"
			}
			if hasLogger {
				logchan <- msg
			} else {
				log.Println(msg.Txt)
			}
		}
	}()
}

// MustReg is similar to Reg(), but panic if having error.
func MustReg(section string, strucePtr Setting) {
	Default.MustReg(section, strucePtr)
}

// Reg registers config section to default config file 'config/config.yaml'.
func Reg(section string, strucePtr Setting) error {
	return Default.Reg(section, strucePtr)
}

// Content returns default yaml config bytes.
func Content() []byte {
	return Default.Content()
}

// GetConfig returns default config section.
func GetConfig(section string) (interface{}, bool) {
	return Default.GetConfig(section)
}

// BindConfig returns default config section copy.
func BindConfig(section string, v interface{}) error {
	return Default.BindConfig(section, v)
}

// ReloadAll reloads all configs.
func ReloadAll() error {
	lock.Lock()
	defer lock.Unlock()
	var errs []string
	for _, c := range cfgos {
		if err := c.Reload(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, string(lineend)))
	}
	return nil
}

// Reload reloads default config.
func Reload() error {
	return Default.Reload()
}

type (
	// Cfgo a whole config
	Cfgo struct {
		filename        string
		originalContent []byte
		content         []byte
		regConfig       map[string]Setting
		allConfig       map[string]interface{}
		regSections     Sections
		otherSections   Sections
		lc              sync.RWMutex
	}
	// Setting must be struct pointer
	Setting interface {
		// load or reload config to app
		Reload(bind BindFunc) error
	}
	// BindFunc bind config to setting
	BindFunc func() error
)

var (
	cfgos   = make(map[string]*Cfgo, 1)
	lock    sync.Mutex
	lineend = func() []byte {
		if runtime.GOOS == "windows" {
			return []byte("\r\n")
		}
		return []byte("\n")
	}()
	indent       = append(lineend, []byte("  ")...)
	dividingLine = append([]byte("# ------------------------- non-automated configuration -------------------------"), lineend...)
)

// MustGet creates a new Cfgo
func MustGet(filename string) *Cfgo {
	c, err := Get(filename)
	if err != nil {
		panic(err)
	}
	return c
}

// Get creates or gets a Cfgo.
func Get(filename string) (*Cfgo, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("[cfgo] %s", err.Error())
	}
	lock.Lock()
	defer lock.Unlock()
	c := cfgos[abs]
	if c != nil {
		return c, nil
	}
	c = &Cfgo{
		filename:      abs,
		regConfig:     make(map[string]Setting),
		allConfig:     make(map[string]interface{}),
		regSections:   make([]*Section, 0, 1),
		otherSections: make([]*Section, 0),
	}
	cfgos[abs] = c
	return c, nil
}

// MustReg is similar to Reg(), but panic if having error.
func (c *Cfgo) MustReg(section string, strucePtr Setting) {
	err := c.Reg(section, strucePtr)
	if err != nil {
		panic(err)
	}
}

// Reg registers config section to config file.
func (c *Cfgo) Reg(section string, strucePtr Setting) error {
	t := reflect.TypeOf(strucePtr)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("[cfgo] Setting type must be struct pointer:\nsection: %s\nSetting: %s", section, t.String())
	}
	if _, ok := c.regConfig[section]; ok {
		return fmt.Errorf("[cfgo] multiple register:\nsection: %s\nSetting: %s", section, t.String())
	}

	c.lc.Lock()
	defer c.lc.Unlock()

	c.regConfig[section] = strucePtr

	// sync config
	return c.sync(func(s string, setting Setting, b []byte) error {
		if s == section {
			return setting.Reload(func() error {
				return yaml.Unmarshal(b, setting)
			})
		}
		return nil
	})
}

// Content returns yaml config bytes.
func (c *Cfgo) Content() []byte {
	c.lc.RLock()
	defer c.lc.RUnlock()
	return c.content
}

// GetConfig returns yaml config section.
func (c *Cfgo) GetConfig(section string) (interface{}, bool) {
	c.lc.RLock()
	defer c.lc.RUnlock()
	if v, ok := c.regConfig[section]; ok {
		return v, ok
	}
	v, ok := c.allConfig[section]
	return v, ok
}

// BindConfig returns yaml config section copy.
func (c *Cfgo) BindConfig(section string, v interface{}) error {
	c.lc.RLock()
	defer c.lc.RUnlock()
	for _, s := range c.regSections {
		if section == s.title {
			return yaml.Unmarshal(s.single, v)
		}
	}
	for _, s := range c.otherSections {
		if section == s.title {
			return yaml.Unmarshal(s.single, v)
		}
	}
	return fmt.Errorf("config section not exist: %s", section)
}

// Reload reloads config.
func (c *Cfgo) Reload() error {
	c.lc.Lock()
	defer c.lc.Unlock()
	return c.sync(func(_ string, setting Setting, b []byte) error {
		return setting.Reload(func() error {
			return yaml.Unmarshal(b, setting)
		})
	})
}

func (c *Cfgo) sync(load func(section string, setting Setting, b []byte) error) (err error) {
	c.originalContent = c.originalContent[:0]
	defer func() {
		if err != nil {
			err = fmt.Errorf("[cfgo] %s", err.Error())
		}
	}()
	d, _ := filepath.Split(c.filename)
	err = os.MkdirAll(d, 0777)
	if err != nil {
		return
	}

	// unmarshal
	err = c.read(load)
	if err != nil {
		return
	}

	// Restore the original configuration
	defer func() {
		if err != nil {
			file, err := os.OpenFile(c.filename, os.O_WRONLY|os.O_SYNC|os.O_TRUNC|os.O_CREATE, 0666)
			if err != nil {
				return
			}
			defer file.Close()
			file.Write(c.originalContent)
		}
	}()

	// marshal
	err = c.write()
	if err != nil {
		return
	}

	return nil
}

func (c *Cfgo) read(load func(section string, setting Setting, b []byte) error) (err error) {
	file, err := os.OpenFile(c.filename, os.O_RDONLY|os.O_SYNC|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	c.originalContent, err = ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		return
	}

	c.allConfig = make(map[string]interface{})
	err = yaml.Unmarshal(c.originalContent, &c.allConfig)
	if err != nil {
		return
	}

	// load config
	var errs []string
	var single []byte
	var has = make(map[string]bool, len(c.regConfig))
	for k, v := range c.allConfig {
		for kk, vv := range c.regConfig {
			if k == kk {
				has[k] = true
				if single, err = yaml.Marshal(v); err != nil {
					return
				}
				// load
				if err = load(kk, vv, single); err != nil {
					errs = append(errs, err.Error())
				}
				break
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, string(lineend)))
	}

	var section *Section
	c.regSections = make([]*Section, 0, len(c.regConfig))
	for k, v := range c.regConfig {
		if section, err = createSection(k, v); err != nil {
			delete(c.allConfig, k)
			return
		}
		var vv interface{}
		if err = yaml.Unmarshal(section.single, &vv); err != nil {
			delete(c.allConfig, k)
			return
		}
		c.allConfig[k] = vv
		c.regSections = append(c.regSections, section)
	}
	sort.Sort(c.regSections)

	c.otherSections = make([]*Section, 0, len(c.allConfig)-len(has))
	for k, v := range c.allConfig {
		if has[k] {
			continue
		}
		if section, err = createSection(k, v); err != nil {
			return
		}
		c.otherSections = append(c.otherSections, section)
	}
	sort.Sort(c.otherSections)
	return nil
}

func createSection(k string, v interface{}) (section *Section, err error) {
	section = &Section{
		title: k,
	}
	var single []byte
	if single, err = yaml.Marshal(v); err != nil {
		return
	}
	section.single = single
	var united = bytes.Replace(single, []byte("\r\n"), []byte("\n"), -1)
	united = bytes.Replace(united, []byte("\n"), indent, -1)
	united = append([]byte(k+":"+string(lineend)+"  "), united[:len(united)-2]...)
	section.united = united
	return
}

func (c *Cfgo) write() error {
	file, err := os.OpenFile(c.filename, os.O_WRONLY|os.O_SYNC|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	content := bytes.NewBuffer(c.content[:0])
	w := io.MultiWriter(file, content)
	for i, section := range c.regSections {
		if i != 0 {
			_, err = w.Write(lineend)
			if err != nil {
				return err
			}
		}
		_, err = w.Write(section.united)
		if err != nil {
			return err
		}
	}

	for i, section := range c.otherSections {
		if i == 0 {
			_, err = w.Write(lineend)
			if err != nil {
				return err
			}
			_, err = w.Write(dividingLine)
			if err != nil {
				return err
			}
		}
		_, err = w.Write(lineend)
		if err != nil {
			return err
		}
		_, err = w.Write(section.united)
		if err != nil {
			return err
		}
	}

	c.content = content.Bytes()
	return nil
}

type (
	Sections []*Section
	Section  struct {
		title  string
		single []byte
		united []byte
	}
)

// Len is the number of elements in the collection.
func (s Sections) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s Sections) Less(i, j int) bool {
	return s[i].title < s[j].title
}

// Swap swaps the elements with indexes i and j.
func (s Sections) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
