package listingstudio

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIEvalFixtureRubricRequiresFactBackedSpecifics(t *testing.T) {
	for i := 1; i <= 20; i++ {
		brand := fmt.Sprintf("Marke %02d", i)
		material := fmt.Sprintf("Material %02d", i)
		facts := SourceFacts{Brand: brand, Attributes: map[string]any{"Material": material}}
		content := ListingContentVersion{
			Title:       fmt.Sprintf("Sachlicher Gartentitel %02d", i),
			Description: "Beschreibung auf Basis der erfassten Fakten.",
			Specifics:   j(map[string]string{"Marke": brand, "Material": material}),
		}
		require.Empty(t, ValidateContent(content), "fixture %d violates content rubric", i)
		require.Empty(t, ValidateFactfulness(content, facts), "fixture %d violates factfulness rubric", i)

		content.Specifics = j(map[string]string{"Material": "erfunden"})
		require.NotEmpty(t, ValidateFactfulness(content, facts), "fixture %d must reject unsupported facts", i)
	}
}
