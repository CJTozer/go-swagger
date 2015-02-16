package validate

import (
	"fmt"
	"reflect"

	"github.com/casualjim/go-swagger/errors"
	"github.com/casualjim/go-swagger/spec"
)

type schemaSliceValidator struct {
	Path            string
	In              string
	MaxItems        *int64
	MinItems        *int64
	UniqueItems     bool
	AdditionalItems *spec.SchemaOrBool
	Items           *spec.SchemaOrArray
	Root            interface{}
	KnownFormats    map[string]FormatValidator
}

func (s *schemaSliceValidator) SetPath(path string) {
	s.Path = path
}

func (s *schemaSliceValidator) Applies(source interface{}, kind reflect.Kind) bool {
	_, ok := source.(*spec.Schema)
	r := ok && kind == reflect.Slice
	// fmt.Printf("slice validator for %q applies %t for %T (kind: %v)\n", s.Path, r, source, kind)
	return r
}

func (s *schemaSliceValidator) Validate(data interface{}) *Result {
	result := new(Result)
	if data == nil {
		return result
	}
	val := reflect.ValueOf(data)
	size := val.Len()

	if s.Items != nil && s.Items.Schema != nil {
		validator := newSchemaValidator(s.Items.Schema, s.Root, s.Path, s.KnownFormats)
		for i := 0; i < size; i++ {
			validator.SetPath(fmt.Sprintf("%s.%d", s.Path, i))
			value := val.Index(i)
			result.Merge(validator.Validate(value.Interface()))
		}
	}

	itemsSize := int64(0)
	if s.Items != nil && len(s.Items.Schemas) > 0 {
		itemsSize = int64(len(s.Items.Schemas))
		for i := int64(0); i < itemsSize; i++ {
			validator := newSchemaValidator(&s.Items.Schemas[i], s.Root, fmt.Sprintf("%s.%d", s.Path, i), s.KnownFormats)
			result.Merge(validator.Validate(val.Index(int(i)).Interface()))
		}

	}
	if s.AdditionalItems != nil && itemsSize < int64(size) {
		if s.Items != nil && len(s.Items.Schemas) > 0 && !s.AdditionalItems.Allows {
			result.AddErrors(errors.New(422, "array doesn't allow for additional items"))
		}
		if s.AdditionalItems.Schema != nil {
			for i := itemsSize; i < (int64(size)-itemsSize)+1; i++ {
				validator := newSchemaValidator(s.AdditionalItems.Schema, s.Root, fmt.Sprintf("%s.%d", s.Path, i), s.KnownFormats)
				result.Merge(validator.Validate(val.Index(int(i)).Interface()))
			}
		}
	}

	if s.MinItems != nil {
		if err := MinItems(s.Path, s.In, itemsSize, *s.MinItems); err != nil {
			result.AddErrors(err)
		}
	}
	if s.MaxItems != nil {
		if err := MaxItems(s.Path, s.In, itemsSize, *s.MaxItems); err != nil {
			result.AddErrors(err)
		}
	}
	if s.UniqueItems {
		if err := UniqueItems(s.Path, s.In, val.Interface()); err != nil {
			result.AddErrors(err)
		}
	}
	result.Inc()
	return result
}

func (s *schemaSliceValidator) hasDuplicates(value reflect.Value, size int) bool {
	var unique []interface{}
	for i := 0; i < value.Len(); i++ {
		v := value.Index(i).Interface()
		for _, u := range unique {
			if reflect.DeepEqual(v, u) {
				return true
			}
		}
		unique = append(unique, v)
	}
	return false
}
