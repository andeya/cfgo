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
// In addition, if the key is "-", the field is ignored.
//
package cfgo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// MustReg is similar to Reg(), but panic if having error.
func MustReg(section string, strucePtr Setting) {
	defaultCfgo.MustReg(section, strucePtr)
}

// Reg registers config section to default config file 'config/config.yaml'.
func Reg(section string, strucePtr Setting) error {
	return defaultCfgo.Reg(section, strucePtr)
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
	return defaultCfgo.Reload()
}

type (
	// Cfgo a whole config
	Cfgo struct {
		filename string
		config   map[string]Setting
		content  []byte
		sections sections
		lc       sync.Mutex
	}
	// Setting must be struct pointer
	Setting interface {
		// load or reload config to app
		Reload() error
	}
)

var (
	cfgos       = make(map[string]*Cfgo, 1)
	lock        sync.Mutex
	defaultCfgo = MustNew("config/config.yaml")
)

// MustNew creates a new Cfgo
func MustNew(filename string) *Cfgo {
	c, err := New(filename)
	if err != nil {
		panic(err)
	}
	return c
}

// New creates a new Cfgo
func New(filename string) (*Cfgo, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("[cfgo] %s", err.Error())
	}
	lock.Lock()
	defer lock.Unlock()
	if _, ok := cfgos[abs]; ok {
		return nil, fmt.Errorf("[cfgo] multiple new: %s", abs)
	}
	c := &Cfgo{
		filename: abs,
		config:   make(map[string]Setting),
		sections: make([]*section, 0, 1),
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
	if _, ok := c.config[section]; ok {
		return fmt.Errorf("[cfgo] multiple register:\nsection: %s\nSetting: %s", section, t.String())
	}

	c.lc.Lock()
	c.config[section] = strucePtr
	c.lc.Unlock()

	// sync config
	return c.sync(strucePtr.Reload)
}

// Reload reloads config.
func (c *Cfgo) Reload() error {
	return c.sync(c.reload)
}

func (c *Cfgo) reload() error {
	// load config
	var errs []string
	for _, v := range c.config {
		if err := v.Reload(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, string(lineend)))
	}
	return nil
}

func (c *Cfgo) sync(load func() error) (err error) {
	c.lc.Lock()
	defer c.lc.Unlock()
	c.sections = make([]*section, 0, len(c.config))
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
	err = c.read()
	if err != nil {
		return
	}

	// load config
	err = load()
	if err != nil {
		return
	}

	// marshal
	err = c.write()
	if err != nil {
		return
	}

	return nil
}

func (c *Cfgo) read() (err error) {
	file, err := os.OpenFile(c.filename, os.O_RDONLY|os.O_SYNC|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	var (
		r    = bufio.NewReader(file)
		line = make([]byte, 0, 10)
	)
	for ; err == nil; c.readSection(line) {
		line, _, err = r.ReadLine()
	}
	if err != nil && err != io.EOF {
		return
	}

	for k, v := range c.config {
		for _, section := range c.sections {
			if k != section.title {
				continue
			}
			err = yaml.Unmarshal(section.single, v)
			if err != nil {
				return
			}
		}
	}
	return nil
}

func (c *Cfgo) write() error {
	file, err := os.OpenFile(c.filename, os.O_WRONLY|os.O_SYNC|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
LOOP:
	for k, v := range c.config {
		single, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		united := bytes.Replace(single, []byte("\r\n"), []byte("\n"), -1)
		united = bytes.Replace(united, []byte("\n"), indent, -1)
		united = append([]byte("  "), united[:len(united)-2]...)
		for _, section := range c.sections {
			if section.title == k {
				section.single = single
				section.united = united
				continue LOOP
			}
		}
		c.sections = append(c.sections, &section{
			title:  k,
			single: single,
			united: united,
		})
	}
	sort.Sort(c.sections)
	for i, section := range c.sections {
		if i != 0 {
			_, err = file.Write(lineend)
			if err != nil {
				return err
			}
		}
		_, err = file.Write(append([]byte(section.title+":"), lineend...))
		if err != nil {
			return err
		}
		_, err = file.Write(section.united)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cfgo) readSection(line []byte) {
	if len(line) > 0 && !bytes.HasPrefix(line, []byte(" ")) && line[len(line)-1] == ':' {
		line = bytes.TrimSpace(line)
		line = bytes.TrimRight(line, ":")
		c.sections = append(c.sections, &section{title: string(line)})
		return
	}
	last := len(c.sections) - 1
	if last == -1 {
		return
	}
	line = append(line, lineend...)
	c.sections[last].united = append(c.sections[last].united, line...)
	line = bytes.TrimPrefix(line, []byte("  "))
	c.sections[last].single = append(c.sections[last].single, line...)
}

var (
	lineend = func() []byte {
		if runtime.GOOS == "windows" {
			return []byte("\r\n")
		}
		return []byte("\n")
	}()
	indent = append(lineend, []byte("  ")...)
)

type (
	sections []*section
	section  struct {
		title  string
		united []byte
		single []byte
	}
)

// Len is the number of elements in the collection.
func (s sections) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s sections) Less(i, j int) bool {
	return s[i].title < s[j].title
}

// Swap swaps the elements with indexes i and j.
func (s sections) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
