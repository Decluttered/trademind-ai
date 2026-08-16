package listingstudio

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestListingStateTransitions(t *testing.T) {
	require.True(t, CanTransition(StateBlocked, StateDrafting))
	require.True(t, CanTransition(StateNeedsReview, StateReady))
	require.False(t, CanTransition(StateReady, StateDrafting))
	require.True(t, CanTransition(StatePublished, StatePaused))
	require.True(t, CanTransition(StatePaused, StatePublished))
	require.True(t, CanTransition(StatePublished, StateEnded))
	require.False(t, CanTransition(StateEnded, StatePublished))
}
func TestContentValidation(t *testing.T) {
	require.Empty(t, ValidateContent(ListingContentVersion{Title: "Solider Produkttitel", Description: "Sachliche Beschreibung"}))
	errs := ValidateContent(ListingContentVersion{Title: "Garantiert garantiert Heilung"})
	require.NotEmpty(t, errs)
}
