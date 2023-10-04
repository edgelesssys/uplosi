/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestSomeNone(t *testing.T) {
	assert := assert.New(t)
	var optInt Option[int]
	someXorNone(t, optInt)
	assert.Equal(optInt, None[int]())
	assert.Equal(9001, optInt.UnwrapOr(9001))
	assert.Equal(9001, optInt.UnwrapOrElse(func() int { return 9001 }))
	optInt = Some[int](42)
	someXorNone(t, optInt)
	assert.NotEqual(optInt, None[int]())
	assert.Equal(optInt, Some[int](42))
	assert.Equal(optInt.Unwrap(), 42)
	assert.True(optInt.IsSome())
	assert.False(optInt.IsNone())
}

func TestUnwrap(t *testing.T) {
	assert := assert.New(t)
	optString := None[string]()
	assert.Panics(func() { optString.Unwrap() })
	assert.NotPanics(func() { optString.UnwrapOr("foo") })
	optString = Some[string]("bar")
	assert.NotPanics(func() { optString.Unwrap() })
	assert.Equal("bar", optString.Unwrap())
	assert.Equal(optString.Unwrap(), optString.UnwrapOr("foo"))
	assert.Equal(optString.Unwrap(), optString.UnwrapOrElse(func() string { return "foo" }))
}

func TestJSON(t *testing.T) {
	assert := assert.New(t)
	var optInt Option[int]
	assert.NoError(json.Unmarshal([]byte("null"), &optInt))
	assert.Equal(optInt, None[int]())
	assert.NoError(json.Unmarshal([]byte("42"), &optInt))
	assert.Equal(optInt, Some[int](42))
	b, err := json.Marshal(optInt)
	assert.NoError(err)
	assert.Equal("42", string(b))
	optInt = None[int]()
	b, err = json.Marshal(optInt)
	assert.NoError(err)
	assert.Equal("null", string(b))

	type complexT struct {
		I Option[int]
		S Option[string]
		P Option[*int]
		M map[string]Option[map[string]Option[string]]
	}

	var complexV Option[complexT]
	assert.False(complexV.IsSome())
	b, err = json.Marshal(complexV)
	assert.NoError(err)
	assert.Equal("null", string(b))

	assert.NoError(json.Unmarshal([]byte(`{"I": 42, "S": "foo", "P": null}`), &complexV))
	assert.True(complexV.IsSome())
	assert.Equal(complexV, Some[complexT](complexT{I: Some[int](42), S: Some[string]("foo"), P: None[*int]()}))

	assert.NoError(json.Unmarshal([]byte(`{"I": 42, "S": "foo", "P": null, "M": {"foo": {"bar": "baz"}}}`), &complexV))
	assert.Equal(complexV, Some[complexT](complexT{I: Some[int](42), S: Some[string]("foo"), P: None[*int](), M: map[string]Option[map[string]Option[string]]{"foo": Some[map[string]Option[string]](map[string]Option[string]{"bar": Some[string]("baz")})}}))
}

func TestTOML(t *testing.T) {
	assert := assert.New(t)
	type someConf struct {
		B      Option[bool]   `toml:"B,omitempty"`
		I      Option[int]    `toml:"I,omitempty"`
		S      Option[string] `toml:"S,omitempty"`
		Normal string         `toml:"Normal,omitempty"`
	}
	buf := new(bytes.Buffer)
	assert.NoError(toml.NewEncoder(buf).Encode(someConf{}))
	assert.Empty(buf.String())
	buf.Reset()

	assert.NoError(toml.NewEncoder(buf).Encode(someConf{I: Some[int](42), S: Some[string]("foo")}))
	assert.Equal("I = 42\nS = \"foo\"\n", buf.String())

	var conf someConf
	buf = new(bytes.Buffer)
	_, err := toml.NewDecoder(buf).Decode(&conf)
	assert.NoError(err)
	assert.Equal(someConf{}, conf)

	conf = someConf{}
	_, err = toml.Decode("B = true\nI = 42\nS = \"foo\"\n", &conf)
	assert.NoError(err)
	assert.Equal(someConf{
		B: Some[bool](true),
		I: Some[int](42),
		S: Some[string]("foo"),
	}, conf)

	conf = someConf{}
	_, err = toml.Decode("B = false\nS = \"bar\"\nNormal = \"abc\"", &conf)
	assert.NoError(err)
	assert.Equal(someConf{
		B:      Some[bool](false),
		S:      Some[string]("bar"),
		Normal: "abc",
	}, conf)
}

func TestTransformer(t *testing.T) {
	assert := assert.New(t)

	dst := None[int]()
	src := Some[int](42)

	tran := OptionTransformer{}
	cb := tran.Transformer(reflect.TypeOf(dst))
	assert.NotNil(cb)

	assert.NoError(cb(reflect.ValueOf(&dst).Elem(), reflect.ValueOf(&src).Elem()))
	assert.Equal(dst, src)

	var nonMatchingType int
	assert.Nil(tran.Transformer(reflect.TypeOf(nonMatchingType)))
	assert.Nil(tran.Transformer(reflect.TypeOf(&nonMatchingType)))
}

func someXorNone[T any](t *testing.T, o Option[T]) {
	if o.IsSome() == o.IsNone() {
		t.Errorf("invalid option: %v", o)
	}
}
