// Package inject provides utilities for mapping and injecting dependencies in various ways.
package zinject

import (
	"fmt"
	"reflect"
)

// Injector represents an interface for mapping and injecting dependencies into structs
// and function arguments.
type Injector interface {
	// Maps dependencies in the Type map to each field in the struct
	// that is tagged with 'inject'. Returns an error if the injection
	// fails.
	Inject(interface{}) error

	// Maps the interface{} value based on its immediate type from reflect.TypeOf.
	Register(interface{}, string) Injector

	// Maps the interface{} value based on the pointer of an Interface provided.
	// This is really only useful for mapping a value as an interface, as interfaces
	// cannot at this time be referenced directly without a pointer.
	RegisterAs(interface{}, string, interface{}) Injector

	// Provides a possibility to directly insert a mapping based on type and value.
	// This makes it possible to directly map type arguments not possible to instantiate
	// with reflect like unidirectional channels.
	Set(reflect.Type, string, reflect.Value) Injector

	// Returns the Value that is mapped to the current type. Returns a zeroed Value if
	// the Type has not been mapped.
	Get(reflect.Type, string) reflect.Value

	// SetParent sets the parent of the injector. If the injector cannot find a
	// dependency in its Type map it will check its parent before returning an
	// error.
	SetParent(Injector)
}

type injector struct {
	values map[reflect.Type]map[string]reflect.Value
	parent Injector
}

// InterfaceOf dereferences a pointer to an Interface type.
// It panics if value is not an pointer to an interface.
func InterfaceOf(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Interface {
		panic("Called inject.InterfaceOf with a value that is not a pointer to an interface. (*MyInterface)(nil)")
	}

	return t
}

// New returns a new Injector.
func New() Injector {
	return &injector{
		values: make(map[reflect.Type]map[string]reflect.Value),
	}
}

// Maps dependencies in the Type map to each field in the struct
// that is tagged with 'inject'.
// Returns an error if the injection fails.
func (inj *injector) Inject(val interface{}) error {
	v := reflect.ValueOf(val)

	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil // Should not panic here ?
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		sf := t.Field(i)
		if !f.CanSet() {
			continue
		}
		if k, found := sf.Tag.Lookup("inject"); found {
			ft := f.Type()
			v := inj.Get(ft, k)
			if !v.IsValid() {
				return fmt.Errorf("Value not found for type %v", ft)
			}
			f.Set(v)
		}
	}

	return nil
}

func (inj *injector) mapOf(typ reflect.Type) map[string]reflect.Value {
	m := inj.values[typ]
	if m == nil {
		m = map[string]reflect.Value{}
		inj.values[typ] = m
	}
	return m
}

// Maps the concrete value of val to its dynamic type using reflect.TypeOf,
// It returns the TypeMapper registered in.
func (inj *injector) Register(val interface{}, key string) Injector {
	inj.mapOf(reflect.TypeOf(val))[key] = reflect.ValueOf(val)
	return inj
}

func (inj *injector) RegisterAs(val interface{}, key string, ifacePtr interface{}) Injector {
	inj.mapOf(InterfaceOf(ifacePtr))[key] = reflect.ValueOf(val)
	return inj
}

// Maps the given reflect.Type to the given reflect.Value and returns
// the Typemapper the mapping has been registered in.
func (inj *injector) Set(typ reflect.Type, key string, val reflect.Value) Injector {
	inj.mapOf(typ)[key] = val
	return inj
}

func (inj *injector) Get(t reflect.Type, key string) reflect.Value {
	val := inj.mapOf(t)[key]

	if val.IsValid() {
		return val
	}

	// no concrete types found, try to find implementors
	// if t is an interface
	if t.Kind() == reflect.Interface {
		for k, v := range inj.values {
			if k.Implements(t) {
				val = v[key]
				if val.IsValid() {
					break
				}
			}
		}
	}

	// Still no type found, try to look it up on the parent
	if !val.IsValid() && inj.parent != nil {
		val = inj.parent.Get(t, key)
	}

	return val

}

func (inj *injector) SetParent(parent Injector) {
	inj.parent = parent
}
