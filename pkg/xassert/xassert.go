package xassert

import "github.com/stretchr/testify/assert"

func ErrorContains(expected string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.ErrorContains(t, err, expected, i...)
	}
}
