package listingstudio

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPublicImageURLRejectsSSRF(t *testing.T) {
	_, err := PublicImageURL(context.Background(), "https://localhost/image.jpg")
	require.Error(t, err)
	_, err = PublicImageURL(context.Background(), "http://example.com/image.jpg")
	require.Error(t, err)
}
