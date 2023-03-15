package evtx

import (
	"fmt"
	"reflect"
	"strings"
)

type GoEvtxElement interface{}

type GoEvtxMap map[string]interface{}

type GoEvtxPath []string

func (p GoEvtxPath) String() string {
	return strings.Join(p, "/")
}

type ErrEvtxEltNotFound struct {
	path GoEvtxPath
}

func (e *ErrEvtxEltNotFound) Error() string {
	return fmt.Sprintf("Element at path %v not found", e.path)
}

func Path(s string) GoEvtxPath {
	return strings.Split(strings.Trim(s, PathSeparator), PathSeparator)
}

func (pg *GoEvtxMap) HasKeys(keys ...string) bool {
	for _, k := range keys {
		if _, ok := (*pg)[k]; !ok {
			return false
		}
	}
	return true
}

func (pg *GoEvtxMap) Add(other GoEvtxMap) {
	for k, v := range other {
		if _, ok := (*pg)[k]; ok {
			panic("Duplicated key")
		}
		(*pg)[k] = v
	}
}

func (pg *GoEvtxMap) GetMap(path *GoEvtxPath) (*GoEvtxMap, error) {
	if len(*path) > 0 {
		if ge, ok := (*pg)[(*path)[0]]; ok {
			if len(*path) == 1 {
				return pg, nil
			}
			switch ge.(type) {
			case GoEvtxMap:
				p := ge.(GoEvtxMap)
				np := (*path)[1:]
				return p.GetMap(&np)
			}
		}
	}
	return nil, &ErrEvtxEltNotFound{*path}
}

func (pg *GoEvtxMap) Get(path *GoEvtxPath) (*GoEvtxElement, error) {
	if len(*path) > 0 {
		if i, ok := (*pg)[(*path)[0]]; ok {
			if len(*path) == 1 {
				cge := GoEvtxElement(i)
				return &cge, nil
			}
			switch i.(type) {
			case GoEvtxMap:
				p := i.(GoEvtxMap)
				np := (*path)[1:]
				return p.Get(&np)
			case map[string]interface{}:
				p := GoEvtxMap(i.(map[string]interface{}))
				np := (*path)[1:]
				return p.Get(&np)
			}
		}
	}
	return nil, &ErrEvtxEltNotFound{*path}
}

func (pg *GoEvtxMap) AnyEqual(path *GoEvtxPath, is []interface{}) bool {
	t, err := pg.Get(path)
	if err != nil {
		return false
	}
	for _, i := range is {
		if reflect.DeepEqual(i, *t) {
			return true
		}
	}
	return false
}

func (pg *GoEvtxMap) IsEventID(eids ...interface{}) bool {
	return pg.AnyEqual(&EventIDPath, eids)
}

func (pg *GoEvtxMap) Set(path *GoEvtxPath, new GoEvtxElement) error {
	if len(*path) > 0 {
		i := (*pg)[(*path)[0]]
		if len(*path) == 1 {
			(*pg)[(*path)[0]] = new
			return nil
		}
		switch i.(type) {
		case GoEvtxMap:
			p := i.(GoEvtxMap)
			np := (*path)[1:]
			return p.Set(&np, new)
		case map[string]interface{}:
			p := GoEvtxMap(i.(map[string]interface{}))
			np := (*path)[1:]
			return p.Set(&np, new)
		}

	}
	return &ErrEvtxEltNotFound{*path}
}

func (pg *GoEvtxMap) Del(path *GoEvtxPath) {
	if len(*path) > 0 {
		if ge, ok := (*pg)[(*path)[0]]; ok {
			if len(*path) == 1 {
				delete(*pg, (*path)[0])
			}
			switch ge.(type) {
			case GoEvtxMap:
				p := ge.(GoEvtxMap)
				np := (*path)[1:]
				p.Del(&np)

			case map[string]interface{}:
				p := GoEvtxMap(ge.(map[string]interface{}))
				np := (*path)[1:]
				p.Del(&np)
			}
		}
	}
}

func (pg *GoEvtxMap) DelXmlns() {
	pg.Del(&XmlnsPath)
}
