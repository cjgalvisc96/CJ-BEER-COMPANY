package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

func startBatch(t *testing.T) *domain.Batch {
	t.Helper()
	beerID, err := domain.ParseBeerRef(shared.NewEntityID().String())
	require.NoError(t, err)
	batch, err := domain.StartBatch(beerID, 500)
	require.NoError(t, err)
	return batch
}

func TestStartBatchRecordsEvent(t *testing.T) {
	batch := startBatch(t)

	assert.Equal(t, domain.BatchStatusBrewing, batch.Status())
	events := batch.PullEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "brewing.batch_started", events[0].EventName())
}

func TestStartBatchRejectsNonPositiveUnits(t *testing.T) {
	beerID, err := domain.ParseBeerRef(shared.NewEntityID().String())
	require.NoError(t, err)

	_, err = domain.StartBatch(beerID, 0)

	assert.Error(t, err)
}

func TestCompleteBatch(t *testing.T) {
	batch := startBatch(t)
	batch.PullEvents()

	require.NoError(t, batch.Complete(480))

	assert.Equal(t, domain.BatchStatusCompleted, batch.Status())
	assert.Equal(t, 480, batch.Units(), "actual yield replaces the plan")
	require.NotNil(t, batch.CompletedAt())

	events := batch.PullEvents()
	require.Len(t, events, 1)
	completed := events[0].(domain.BatchCompleted)
	assert.Equal(t, "brewing.batch_completed", completed.EventName())
	assert.Equal(t, 480, completed.Units)
}

func TestCompleteBatchTwiceFails(t *testing.T) {
	batch := startBatch(t)
	require.NoError(t, batch.Complete(100))

	assert.ErrorIs(t, batch.Complete(100), domain.ErrBatchAlreadyCompleted)
}
