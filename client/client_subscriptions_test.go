package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/xerrors"
)

func Test_parseSubscriptionResponse(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		r := &fakeRes{}
		err := parseSubscriptionResponse([]byte(qqlSingleErr), r)

		expectedType := &ErrorResponse{}
		require.IsType(t, expectedType, err)

		gqlExpectedType := &gqlerror.List{}
		require.IsType(t, gqlExpectedType, err.(*ErrorResponse).GqlErrors)

		require.Nil(t, err.(*ErrorResponse).NetworkError)
	})

	t.Run("bad error format", func(t *testing.T) {
		r := &fakeRes{}
		err := parseSubscriptionResponse([]byte(withBadErrorsFormat), r)

		expectedType := xerrors.Errorf("%w", errors.New("some"))
		require.IsType(t, expectedType, err)
	})

	t.Run("no error", func(t *testing.T) {
		r := &fakeRes{}
		err := parseSubscriptionResponse([]byte(validData), r)

		require.Nil(t, err)
	})
}
