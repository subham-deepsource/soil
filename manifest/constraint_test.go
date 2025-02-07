//go:build ide || test_unit
// +build ide test_unit

package manifest_test

import (
	"github.com/da-moon/soil/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConstraint_Merge(t *testing.T) {
	t.Run(`nil`, func(t *testing.T) {
		res := manifest.Constraint{"a": "b"}.Merge(nil)
		assert.Equal(t, manifest.Constraint{"a": "b"}, res)
	})
}

func TestConstraint_Check(t *testing.T) {

	t.Run("equal", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "one,two",
		}))
	})
	t.Run("equal strict", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "= ${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "one,two",
		}))
	})
	t.Run("equal strict empty", func(t *testing.T) {
		constraint := manifest.Constraint{
			"": "= ${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "",
		}))
	})
	t.Run("not equal", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "!= two",
		}))
	})
	t.Run("not equal empty", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "!= ",
		}))
	})
	t.Run("in ok", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "~ ${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "one,two,three",
		}))
	})
	t.Run("in fail", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "~ ${meta.field}",
		}
		assert.Error(t, constraint.Check(map[string]string{
			"meta.field": "one,three",
		}))
	})
	t.Run("not in ok", func(t *testing.T) {
		constraint := manifest.Constraint{
			"none": "!~ ${meta.field}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.field": "one,two,three",
		}))
	})
	t.Run("not in fail", func(t *testing.T) {
		constraint := manifest.Constraint{
			"one,two": "!~ ${meta.field}",
		}
		assert.Error(t, constraint.Check(map[string]string{
			"meta.field": "one,two,three",
		}))
	})
	t.Run("less ok", func(t *testing.T) {
		constraint := manifest.Constraint{
			"2": "< ${meta.num}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.num": "11",
		}))
	})
	t.Run("less fail", func(t *testing.T) {
		constraint := manifest.Constraint{
			"2": "< ${meta.num}",
		}
		assert.Error(t, constraint.Check(map[string]string{
			"meta.num": "1",
		}))
	})
	t.Run("greater ok", func(t *testing.T) {
		constraint := manifest.Constraint{
			"2": "> ${meta.num}",
		}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.num": "1",
		}))
	})
	t.Run("greater fail", func(t *testing.T) {
		constraint := manifest.Constraint{
			"2": "> ${meta.num}",
		}
		assert.Error(t, constraint.Check(map[string]string{
			"meta.num": "3",
		}))
	})
	t.Run("empty", func(t *testing.T) {
		constraint := manifest.Constraint{}
		assert.NoError(t, constraint.Check(map[string]string{
			"meta.num": "3",
		}))
	})
}

func TestConstraint_FilterOut(t *testing.T) {
	constraint := manifest.Constraint{
		"${meta.a}":                       "true",
		"${resource.counter.a.allocated}": "true",
		"${resource.port.8080.allocated}": "true",
	}
	t.Run("non-existent", func(t *testing.T) {
		res := constraint.FilterOut("none")
		assert.Equal(t, res, manifest.Constraint{
			"${meta.a}":                       "true",
			"${resource.counter.a.allocated}": "true",
			"${resource.port.8080.allocated}": "true",
		})
	})
	t.Run("resource", func(t *testing.T) {
		res := constraint.FilterOut("resource.")
		assert.Equal(t, res, manifest.Constraint{
			"${meta.a}": "true",
		})
	})
}
