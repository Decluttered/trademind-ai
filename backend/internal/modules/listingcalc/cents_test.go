package listingcalc

import (
	"github.com/stretchr/testify/require"
	"testing"
	"testing/quick"
)

func TestProfitCentsProperty(t *testing.T) {
	property := func(a, b, c, d, e uint16) bool {
		got, err := ProfitCents(int64(a), int64(b), int64(c), int64(d), int64(e))
		return err == nil && got == int64(a)-int64(b)-int64(c)-int64(d)-int64(e)
	}
	require.NoError(t, quick.Check(property, nil))
}

func TestBasisPointsBoundaries(t *testing.T) {
	got, err := BasisPoints(1, 3)
	require.NoError(t, err)
	require.Equal(t, int64(3333), got)
	_, err = BasisPoints(1, 0)
	require.ErrorIs(t, err, ErrInvalidMoney)
}
