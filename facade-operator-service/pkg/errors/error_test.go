package customerrors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExpectedError(t *testing.T) {
	testMessage := "test"
	expectedError := ExpectedError{testMessage}
	assert.Equal(t, testMessage, expectedError.Error())
}
