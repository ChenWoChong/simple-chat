package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseContent(t *testing.T) {
	name, content, err := ParseContent(`@chenwochong test1`)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, `chenwochong`, name)
	assert.Equal(t, `test1`, content)

	name, content, err = ParseContent(`@chenwochong test1 test2`)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, `chenwochong`, name)
	assert.Equal(t, `test1 test2`, content)

	name, content, err = ParseContent(`@ test1 test2`)
	if err != nil {
	}
	t.Log(err)

	name, content, err = ParseContent(`@`)
	if err != nil {
	}
	t.Log(err)

	name, content, err = ParseContent(`@adfadsadsf`)
	if err != nil {
	}
	t.Log(err)



	name, content, err = ParseContent(`chenwochong test1 test2`)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, ``, name)
	assert.Equal(t, `chenwochong test1 test2`, content)
}
