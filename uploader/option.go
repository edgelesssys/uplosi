/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/BurntSushi/toml"
)

type Option[T any] struct {
	Val   T
	Valid bool
}

func Some[T any](val T) Option[T] {
	return Option[T]{Val: val, Valid: true}
}

func None[T any]() Option[T] {
	return Option[T]{}
}

func (o Option[T]) IsSome() bool {
	return o.Valid
}

func (o *Option[T]) IsNone() bool {
	return !o.IsSome()
}

func (o *Option[T]) Unwrap() T {
	if !o.Valid {
		panic("invalid option")
	}
	return o.Val
}

func (o *Option[T]) UnwrapOr(def T) T {
	if !o.Valid {
		return def
	}
	return o.Val
}

func (o *Option[T]) UnwrapOrElse(f func() T) T {
	if !o.Valid {
		return f()
	}
	return o.Val
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.Valid = false
		return nil
	}

	if err := json.Unmarshal(data, &o.Val); err != nil {
		return err
	}
	o.Valid = true
	return nil
}

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.Val)
}

func (o *Option[T]) UnmarshalTOML(v any) error {
	if vAsT, ok := v.(T); ok {
		o.Val = vAsT
		o.Valid = true
		return nil
	}

	oVal := any(&o.Val)

	switch oVal.(type) {
	case *int:
		o.Val = any(int(v.(int64))).(T)
		o.Valid = true
		return nil
	}

	if valUnmarshaler, ok := oVal.(toml.Unmarshaler); ok {
		o.Valid = true
		return valUnmarshaler.UnmarshalTOML(v)
	}

	o.Valid = false
	return errors.New("invalid type")
}

func (o Option[T]) MarshalTOML() ([]byte, error) {
	if !o.Valid {
		return []byte{}, nil
	}
	buf := new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(o.Val)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type OptionTransformer struct{}

func (t OptionTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	if _, ok := typ.MethodByName("IsSome"); !ok {
		return nil
	}
	return func(dst, src reflect.Value) error {
		srcIsSome := src.MethodByName("IsSome")
		srcResult := srcIsSome.Call([]reflect.Value{})
		dstIsSome := dst.MethodByName("IsSome")
		dstResult := dstIsSome.Call([]reflect.Value{})
		if srcResult[0].Bool() && !dstResult[0].Bool() && dst.CanSet() {
			dst.Set(src)
		}

		return nil
	}
}

var (
	_ toml.Unmarshaler = (*Option[int])(nil)
	_ toml.Marshaler   = Option[int]{}
)
