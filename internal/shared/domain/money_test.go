package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

func TestNewMoneyValidations(t *testing.T) {
	_, err := shared.NewMoney(-1, "USD")
	assert.Error(t, err)

	_, err = shared.NewMoney(100, "usd")
	assert.Error(t, err)

	money, err := shared.NewMoney(100, "COP")
	require.NoError(t, err)
	assert.Equal(t, int64(100), money.Cents())
	assert.Equal(t, "COP", money.Currency())
}

func TestMoneyArithmetic(t *testing.T) {
	a, err := shared.NewMoney(250, "USD")
	require.NoError(t, err)
	b, err := shared.NewMoney(150, "USD")
	require.NoError(t, err)

	sum, err := a.Add(b)
	require.NoError(t, err)
	assert.Equal(t, int64(400), sum.Cents())

	assert.Equal(t, int64(750), a.Mul(3).Cents())

	other, err := shared.NewMoney(100, "COP")
	require.NoError(t, err)
	_, err = a.Add(other)
	assert.Error(t, err, "currency mismatch must fail")
}
